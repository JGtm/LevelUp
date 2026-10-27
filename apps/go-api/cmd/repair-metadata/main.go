// cmd/repair-metadata — répare les index ART corrompus dans metadata.duckdb.
//
// Symptôme : "Failed to delete all rows from index. Only deleted 0 out of N rows."
// Cause    : DELETE WHERE non-PK a corrompu les index secondaires DuckDB ART.
// Fix      : copie les données dans une table temporaire, recrée la table et ses index depuis leur
//
//	définition STOCKÉE par DuckDB (duckdb_tables / duckdb_indexes), réinsère par liste de
//	colonnes depuis la table temporaire.
//
// Aucune définition n'est écrite en dur : la table recréée a exactement les colonnes de celle qui existe,
// que la base soit celle d'avant ou d'après une migration de colonnes (textes par langue, tagging...).
// Une définition en dur périmée faisait échouer la réinsertion (nombre de colonnes différent) APRÈS le
// DROP : table vide, données seulement dans la table temporaire. Chaque étape destructive est précédée
// d'un contrôle de nombre de lignes ; un échec après le DROP rend la table à ses données.
//
// Usage :
//
//	go run ./cmd/repair-metadata/ [--db path/to/metadata.duckdb]
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// repairedTables : tables à réparer, dans l'ordre (preset_arc_step dépend de preset_arc).
var repairedTables = []string{"challenge_template", "preset_arc_step", "preset_arc"}

func main() {
	dbPath := flag.String("db", "data/titles/halo_infinite/warehouse/metadata.duckdb", "chemin vers metadata.duckdb")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	if err := run(*dbPath); err != nil {
		slog.Error("repair failed", "err", err)
		os.Exit(1)
	}
	slog.Info("repair completed successfully")
}

func run(dbPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	return repairAll(ctx, db)
}

// repairAll répare les tables de repairedTables, dans l'ordre.
func repairAll(ctx context.Context, db *sql.DB) error {
	for _, table := range repairedTables {
		slog.InfoContext(ctx, "repairing table", "table", table)
		if err := repairTable(ctx, db, table); err != nil {
			return fmt.Errorf("%s: %w", table, err)
		}
	}
	return nil
}

// tableDefinition est la forme actuelle d'une table : sa définition, ses index et ses colonnes.
type tableDefinition struct {
	createSQL string
	indexSQL  []string
	columns   []string
}

func countRows(ctx context.Context, db *sql.DB, table string) (int, error) {
	var n int
	if err := db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM %s`, table)).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// readDefinition lit, dans le catalogue de DuckDB, la définition courante de `table`.
func readDefinition(ctx context.Context, db *sql.DB, table string) (tableDefinition, error) {
	var def tableDefinition
	err := db.QueryRowContext(ctx,
		`SELECT sql FROM duckdb_tables() WHERE database_name = current_database() AND table_name = ?`, table,
	).Scan(&def.createSQL)
	if errors.Is(err, sql.ErrNoRows) {
		return def, errors.New("table absente du catalogue")
	}
	if err != nil {
		return def, fmt.Errorf("lecture de la définition: %w", err)
	}

	cols, err := db.QueryContext(ctx,
		`SELECT column_name FROM duckdb_columns()
		  WHERE database_name = current_database() AND table_name = ? ORDER BY column_index`, table)
	if err != nil {
		return def, fmt.Errorf("lecture des colonnes: %w", err)
	}
	defer cols.Close()
	for cols.Next() {
		var c string
		if err := cols.Scan(&c); err != nil {
			return def, fmt.Errorf("lecture d'une colonne: %w", err)
		}
		def.columns = append(def.columns, `"`+strings.ReplaceAll(c, `"`, `""`)+`"`)
	}
	if err := cols.Err(); err != nil {
		return def, fmt.Errorf("lecture des colonnes: %w", err)
	}

	idx, err := db.QueryContext(ctx,
		`SELECT sql FROM duckdb_indexes()
		  WHERE database_name = current_database() AND table_name = ? ORDER BY index_name`, table)
	if err != nil {
		return def, fmt.Errorf("lecture des index: %w", err)
	}
	defer idx.Close()
	for idx.Next() {
		var s sql.NullString
		if err := idx.Scan(&s); err != nil {
			return def, fmt.Errorf("lecture d'un index: %w", err)
		}
		if s.Valid && s.String != "" {
			def.indexSQL = append(def.indexSQL, s.String)
		}
	}
	return def, idx.Err()
}

// repairTable reconstruit `table` sans réutiliser ses index. Aucune étape destructive ne s'exécute sans que
// les données soient en sûreté : la copie est comptée avant le DROP, la réinsertion est comptée avant la
// suppression de la copie, et un échec après le DROP remet les données sous le nom de la table.
func repairTable(ctx context.Context, db *sql.DB, table string) error {
	tmp := table + "_repair_tmp"
	exec := func(q string) error {
		_, err := db.ExecContext(ctx, q)
		return err
	}

	def, err := readDefinition(ctx, db, table)
	if err != nil {
		return err
	}
	cols := strings.Join(def.columns, ", ")

	before, err := countRows(ctx, db, table)
	if err != nil {
		return fmt.Errorf("count original: %w", err)
	}

	// 1. Copier les données dans une table temporaire (scan séquentiel, pas d'index).
	if err := exec(fmt.Sprintf(`CREATE TABLE %s AS SELECT %s FROM %s`, tmp, cols, table)); err != nil {
		return fmt.Errorf("create temp table (une copie d'un essai précédent est à examiner avant tout): %w", err)
	}
	saved, err := countRows(ctx, db, tmp)
	if err != nil {
		return fmt.Errorf("count temp: %w", err)
	}
	slog.InfoContext(ctx, "data backed up", "table", table, "rows", saved, "columns", len(def.columns))
	if saved != before {
		_ = exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s`, tmp))
		return fmt.Errorf("copie incomplète (%d lignes sur %d) : la table d'origine est intacte", saved, before)
	}

	// 2. Supprimer la table corrompue (avec ses index) : les données sont en sûreté dans tmp.
	if err := exec(fmt.Sprintf(`DROP TABLE %s`, table)); err != nil {
		_ = exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s`, tmp))
		return fmt.Errorf("drop original: %w", err)
	}

	// 3. Recréer la table depuis sa définition d'origine, 4. réinsérer par liste de colonnes.
	if err := reinsert(ctx, db, table, tmp, def, saved); err != nil {
		return restoreFromTemp(ctx, db, table, tmp, saved, err)
	}
	slog.InfoContext(ctx, "data restored", "table", table, "rows", saved)

	// 5. Recréer les index, 6. supprimer la table temporaire.
	for _, idx := range def.indexSQL {
		if err := exec(idx); err != nil {
			return fmt.Errorf("create index (données restaurées, copie conservée dans %s): %w", tmp, err)
		}
	}
	if err := exec(fmt.Sprintf(`DROP TABLE %s`, tmp)); err != nil {
		return fmt.Errorf("drop temp: %w", err)
	}
	return nil
}

// reinsert recrée `table` depuis `def` et y recopie `tmp`, puis vérifie le nombre de lignes.
func reinsert(ctx context.Context, db *sql.DB, table, tmp string, def tableDefinition, want int) error {
	if _, err := db.ExecContext(ctx, def.createSQL); err != nil {
		return fmt.Errorf("recreate table: %w", err)
	}
	cols := strings.Join(def.columns, ", ")
	if _, err := db.ExecContext(ctx,
		fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM %s`, table, cols, cols, tmp)); err != nil {
		return fmt.Errorf("reinsert data: %w", err)
	}
	got, err := countRows(ctx, db, table)
	if err != nil {
		return fmt.Errorf("count restored: %w", err)
	}
	if got != want {
		return fmt.Errorf("réinsertion incomplète (%d lignes sur %d)", got, want)
	}
	return nil
}

// restoreFromTemp remet les données sous le nom de la table après une reconstruction échouée : elles ne doivent
// jamais rester seulement dans la table temporaire. La table est alors sans clé ni index (à reconstruire).
func restoreFromTemp(ctx context.Context, db *sql.DB, table, tmp string, rows int, cause error) error {
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS %s`, table)); err != nil {
		slog.ErrorContext(ctx, "restauration impossible : la table partielle ne se supprime pas",
			"table", table, "temp_table", tmp, "err", err)
		return errors.Join(cause, fmt.Errorf("drop partielle: %w", err))
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE %s RENAME TO %s`, tmp, table)); err != nil {
		slog.ErrorContext(ctx, "restauration impossible : les données sont dans la table temporaire",
			"table", table, "temp_table", tmp, "err", err)
		return errors.Join(cause, fmt.Errorf("restauration depuis %s: %w", tmp, err))
	}
	slog.ErrorContext(ctx, "reconstruction échouée : données restaurées sans clé ni index",
		"table", table, "rows", rows, "err", cause)
	return fmt.Errorf("%w (données restaurées sous le nom d'origine, sans clé ni index)", cause)
}
