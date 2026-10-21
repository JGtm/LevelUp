package duckdbbackup

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"levelup/go-api/pkg/duckdbkeys"
)

// tableSchema rejoue, pour une clé, le schéma exporté avec la sauvegarde (nil : sauvegarde sans
// schéma lisible) et verse ses tables.
type tableSchema struct {
	key    string
	cycle  string // horodatage de l'export dont le schéma est rejoué
	ddl    *schemaDDL
	result *ImportResult
}

func (ts *tableSchema) sequences() []string {
	if ts.ddl == nil {
		return nil
	}
	return ts.ddl.Sequences
}

func (ts *tableSchema) indexes() []string {
	if ts.ddl == nil {
		return nil
	}
	return ts.ddl.Indexes
}

// replay exécute chaque ordre DDL ; un refus est journalisé en WARN et compté, sans arrêter la
// restauration (les lignes, elles, sont déjà ou seront versées).
func (ts *tableSchema) replay(ctx context.Context, db *sql.DB, stmts []string) {
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			ts.schemaFailure(ctx, stmt, err)
		}
	}
}

// replayViews recrée les vues ; une vue qui en lit une autre peut précéder celle-ci dans l'ordre
// alphabétique : passes successives tant qu'une passe en crée au moins une.
func (ts *tableSchema) replayViews(ctx context.Context, db *sql.DB) {
	if ts.ddl == nil {
		return
	}
	pending := ts.ddl.Views
	for len(pending) > 0 {
		var retry []string
		lastErr := map[string]error{}
		for _, stmt := range pending {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				retry = append(retry, stmt)
				lastErr[stmt] = err
			}
		}
		if len(retry) == len(pending) {
			for _, stmt := range retry {
				ts.schemaFailure(ctx, stmt, lastErr[stmt])
			}
			return
		}
		pending = retry
	}
}

func (ts *tableSchema) schemaFailure(ctx context.Context, stmt string, err error) {
	short := strings.Join(strings.Fields(stmt), " ")
	if len(short) > 80 {
		short = short[:80]
	}
	ts.result.SchemaFailures = append(ts.result.SchemaFailures, ts.key+":"+short)
	slog.WarnContext(ctx, "restore: objet du schéma de la sauvegarde non recréé",
		"key", ts.key, "ddl", short, "err", err)
}

// importTable crée la table depuis le schéma de la sauvegarde puis y verse le Parquet, colonne
// par colonne nommée. Aucune ligne n'est jamais écartée : si le schéma est refusé, ou s'il
// refuse une ligne (base sauvegardée elle-même hors de ses contraintes), la table est importée
// telle quelle par le repli sans clé.
func (ts *tableSchema) importTable(ctx context.Context, db *sql.DB, f parquetFile) error {
	table := f.table
	src := "read_parquet(" + duckdbkeys.Literal(filepath.ToSlash(f.path)) + ")"
	stmt := ts.tableDDL(ctx, f)
	if stmt == "" {
		return ts.fallback(ctx, db, table, src)
	}
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		slog.WarnContext(ctx, "restore: schéma de la table refusé, repli sans clé",
			"key", ts.key, "table", table, "err", err)
		return ts.fallback(ctx, db, table, src)
	}
	cols, err := parquetColumns(ctx, db, src)
	if err != nil {
		return err
	}
	list := identList(cols)
	if _, err := db.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM %s",
		duckdbkeys.Ident(table), list, list, src)); err != nil {
		slog.WarnContext(ctx, "restore: lignes refusées par le schéma de leur époque, repli sans clé (aucune ligne écartée)",
			"key", ts.key, "table", table, "err", err)
		if _, dropErr := db.ExecContext(ctx, "DROP TABLE "+duckdbkeys.Ident(table)); dropErr != nil {
			return fmt.Errorf("abandon du schéma refusé: %w", dropErr)
		}
		return ts.fallback(ctx, db, table, src)
	}
	return nil
}

// fallback importe la table par CREATE TABLE AS (sans clé ni défaut), journalisé et compté.
func (ts *tableSchema) fallback(ctx context.Context, db *sql.DB, table, src string) error {
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE %s AS SELECT * FROM %s`,
		duckdbkeys.Ident(table), src)); err != nil {
		return fmt.Errorf("CREATE TABLE AS: %w", err)
	}
	ts.result.FallbackTables = append(ts.result.FallbackTables, ts.key+":"+table)
	slog.WarnContext(ctx, "restore: table importée par CREATE TABLE AS : lignes seules, sans clé, défaut ni séquence",
		"key", ts.key, "table", table)
	return nil
}

// parquetColumns rend les noms des colonnes de la relation src.
func parquetColumns(ctx context.Context, db *sql.DB, src string) ([]string, error) {
	rows, err := db.QueryContext(ctx, "SELECT column_name FROM (DESCRIBE SELECT * FROM "+src+")")
	if err != nil {
		return nil, fmt.Errorf("colonnes du Parquet: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, fmt.Errorf("colonnes du Parquet: lecture: %w", err)
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

// identList cite et joint des noms de colonnes.
func identList(cols []string) string {
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = duckdbkeys.Ident(c)
	}
	return strings.Join(quoted, ", ")
}

// tableDDL rend le CREATE TABLE de la table, "" (repli) si la sauvegarde n'a pas de schéma, si
// la table en est absente, ou si son Parquet vient d'un autre export que le schéma (dernier
// export de la table en échec : ses lignes sont d'une autre époque que le schéma).
func (ts *tableSchema) tableDDL(ctx context.Context, f parquetFile) string {
	if ts.ddl == nil {
		return ""
	}
	if f.ts != ts.cycle {
		slog.WarnContext(ctx, "restore: Parquet d'un autre export que le schéma, schéma ignoré pour cette table",
			"key", ts.key, "table", f.table, "parquet_export", f.ts, "schema_export", ts.cycle)
		return ""
	}
	stmt, ok := ts.ddl.Tables[f.table]
	if !ok {
		slog.WarnContext(ctx, "restore: table absente du schéma de la sauvegarde", "key", ts.key, "table", f.table)
	}
	return stmt
}
