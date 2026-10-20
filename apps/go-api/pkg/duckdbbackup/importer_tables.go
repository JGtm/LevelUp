package duckdbbackup

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"

	"levelup/go-api/pkg/duckdbkeys"
)

// importTable verse le Parquet parqPath dans la table `table` de db.
//
// Table présente dans le schéma canonique : ses lignes posées par le provisionnement (seeds,
// registre des migrations) sont remplacées par celles de la sauvegarde, versées colonne par
// colonne NOMMÉE dans la table existante — ses clés, défauts et index restent ceux du schéma.
// Les doublons d'une clé sont écartés selon la règle de duckdbkeys.DedupSelect (la ligne la plus
// récente est gardée), comptés et journalisés. Une colonne de la sauvegarde que le schéma ne
// connaît plus est ajoutée à la table (WARN) plutôt que perdue.
//
// Table absente du schéma canonique : recréée depuis le DDL que la sauvegarde porte pour elle
// (clés et défauts de la base sauvegardée), puis remplie comme ci-dessus. Sans DDL (sauvegarde
// antérieure à son introduction, ou DDL refusé) : repli CREATE TABLE AS, sans clé ni défaut,
// journalisé en WARN et compté dans ImportResult.FallbackTables.
func (im *importer) importTable(ctx context.Context, db *sql.DB, key, table, parqPath string) error {
	src := "read_parquet(" + duckdbkeys.Literal(filepath.ToSlash(parqPath)) + ")"
	exists, err := baseTableExists(ctx, db, table)
	if err != nil {
		return err
	}
	if !exists && !im.createFromSourceDDL(ctx, db, key, table) {
		if _, err := db.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE %s AS SELECT * FROM %s`,
			duckdbkeys.Ident(table), src)); err != nil {
			return fmt.Errorf("CREATE TABLE AS (hors schéma canonique): %w", err)
		}
		im.result.FallbackTables = append(im.result.FallbackTables, key+":"+table)
		slog.WarnContext(ctx, "restore: table hors schéma canonique, importée sans clé ni défaut",
			"key", key, "table", table)
		return nil
	}

	parqCols, err := parquetColumns(ctx, db, src)
	if err != nil {
		return err
	}
	if err := addMissingColumns(ctx, db, key, table, parqCols); err != nil {
		return err
	}
	keys, err := duckdbkeys.TableKeys(ctx, db, table)
	if err != nil {
		return err
	}
	if err := relaxNotNull(ctx, db, tableSource{key: key, table: table, src: src, cols: parqCols}, keys); err != nil {
		return err
	}
	names := duckdbkeys.ColumnNames(parqCols)
	// Vidage en autocommit, AVANT la transaction d'écriture : supprimer puis réinsérer les
	// mêmes clés dans une seule transaction est un motif à proscrire sur un index ART.
	if _, err := db.ExecContext(ctx, "DELETE FROM "+duckdbkeys.Ident(table)); err != nil {
		return fmt.Errorf("vidage des lignes du provisionnement: %w", err)
	}
	st, err := copyInTx(ctx, db, table, src, names, duckdbkeys.CoveredKeys(keys, names),
		duckdbkeys.RecencyColumn(parqCols))
	if err != nil {
		return err
	}
	dropped := st.NullKeyDropped + st.DuplicatesDropped
	im.result.DroppedRows += dropped
	if dropped > 0 {
		slog.WarnContext(ctx, "restore: lignes écartées à l'import (clé du schéma canonique)",
			"key", key, "table", table, "rows_source", st.Source, "rows_kept", st.Copied,
			"null_key_dropped", st.NullKeyDropped, "duplicates_dropped", st.DuplicatesDropped,
			"divergent_keys", st.DivergentKeys)
	}
	return nil
}

// copyInTx recopie la source dans la table sous une transaction.
func copyInTx(ctx context.Context, db *sql.DB, table, src string, cols []string,
	keys []duckdbkeys.Key, recency string) (duckdbkeys.CopyStats, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return duckdbkeys.CopyStats{}, fmt.Errorf("transaction: %w", err)
	}
	st, err := duckdbkeys.CopyInto(ctx, tx, table, src, cols, keys, recency)
	if err != nil {
		_ = tx.Rollback()
		return st, err
	}
	if err := tx.Commit(); err != nil {
		return st, fmt.Errorf("commit: %w", err)
	}
	return st, nil
}

// addMissingColumns ajoute à la table les colonnes de la sauvegarde que le schéma canonique ne
// porte pas (colonne retirée depuis la sauvegarde, ou ajoutée hors migrations) : leurs valeurs
// sont conservées plutôt que perdues.
func addMissingColumns(ctx context.Context, db *sql.DB, key, table string, parqCols []duckdbkeys.Column) error {
	cols, err := duckdbkeys.TableColumns(ctx, db, table)
	if err != nil {
		return err
	}
	have := make(map[string]bool, len(cols))
	for _, c := range cols {
		have[c.Name] = true
	}
	for _, c := range parqCols {
		if have[c.Name] {
			continue
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
			duckdbkeys.Ident(table), duckdbkeys.Ident(c.Name), c.Type)); err != nil {
			return fmt.Errorf("ajout de la colonne %s absente du schéma: %w", c.Name, err)
		}
		slog.WarnContext(ctx, "restore: colonne de la sauvegarde absente du schéma canonique, ajoutée",
			"key", key, "table", table, "column", c.Name, "type", c.Type)
	}
	return nil
}

// relaxNotNull retire le NOT NULL du schéma canonique sur une colonne (hors clé primaire) dont
// la sauvegarde porte des NULL : la base sauvegardée les acceptait (colonne née sans la
// contrainte sur une base ancienne). Les valeurs sont restaurées telles quelles — jamais
// remplacées par le défaut, qui changerait leur sens (un `written_at` NULL rempli par `now()`
// passerait pour la version la plus récente dans une vue `_latest`).
func relaxNotNull(ctx context.Context, db *sql.DB, ts tableSource, keys []duckdbkeys.Key) error {
	cols, err := duckdbkeys.TableColumns(ctx, db, ts.table)
	if err != nil {
		return err
	}
	inPK := map[string]bool{}
	for _, k := range keys {
		if k.Primary {
			for _, c := range k.Columns {
				inPK[c] = true
			}
		}
	}
	inSrc := map[string]bool{}
	for _, c := range ts.cols {
		inSrc[c.Name] = true
	}
	for _, c := range cols {
		if !c.NotNull || inPK[c.Name] || !inSrc[c.Name] {
			continue
		}
		var nulls int64
		if err := db.QueryRowContext(ctx, fmt.Sprintf("SELECT count(*) FROM %s WHERE %s IS NULL",
			ts.src, duckdbkeys.Ident(c.Name))).Scan(&nulls); err != nil {
			return fmt.Errorf("NULL de la colonne %s: %w", c.Name, err)
		}
		if nulls == 0 {
			continue
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL",
			duckdbkeys.Ident(ts.table), duckdbkeys.Ident(c.Name))); err != nil {
			return fmt.Errorf("retrait du NOT NULL de %s: %w", c.Name, err)
		}
		slog.WarnContext(ctx, "restore: NOT NULL du schéma canonique retiré, la sauvegarde porte des NULL",
			"key", ts.key, "table", ts.table, "column", c.Name, "null_rows", nulls)
	}
	return nil
}

// parquetColumns rend les colonnes (nom, type) de la relation src.
func parquetColumns(ctx context.Context, db *sql.DB, src string) ([]duckdbkeys.Column, error) {
	rows, err := db.QueryContext(ctx, "SELECT column_name, column_type FROM (DESCRIBE SELECT * FROM "+src+")")
	if err != nil {
		return nil, fmt.Errorf("colonnes du Parquet: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var cols []duckdbkeys.Column
	for rows.Next() {
		var c duckdbkeys.Column
		if err := rows.Scan(&c.Name, &c.Type); err != nil {
			return nil, fmt.Errorf("colonnes du Parquet: lecture: %w", err)
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

// baseTableExists indique si la base courante porte une table (pas une vue) `table` dans main.
func baseTableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	var n int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM duckdb_tables()
		WHERE database_name = current_database() AND schema_name = 'main' AND table_name = ?`,
		table).Scan(&n); err != nil {
		return false, fmt.Errorf("existence de la table %s: %w", table, err)
	}
	return n > 0, nil
}

// tableSource : une table à remplir et sa source Parquet.
type tableSource struct {
	key, table, src string
	cols            []duckdbkeys.Column // colonnes de la source
}

// createFromSourceDDL crée la table depuis le DDL de la sauvegarde. Rend false (repli) quand la
// sauvegarde n'en porte pas pour elle ou que DuckDB le refuse (journalisé en WARN).
func (im *importer) createFromSourceDDL(ctx context.Context, db *sql.DB, key, table string) bool {
	stmt, ok := im.ddl[table]
	if !ok {
		return false
	}
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		slog.WarnContext(ctx, "restore: DDL de la sauvegarde refusé, repli sans clé",
			"key", key, "table", table, "err", err)
		return false
	}
	im.result.SourceDDLTables = append(im.result.SourceDDLTables, key+":"+table)
	slog.InfoContext(ctx, "restore: table hors schéma canonique recréée depuis le DDL de la sauvegarde",
		"key", key, "table", table)
	return true
}
