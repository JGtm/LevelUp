// cmd/repair-metadata — répare les index ART corrompus dans metadata.duckdb.
//
// Symptôme : "Failed to delete all rows from index. Only deleted 0 out of N rows."
// Cause    : DELETE WHERE non-PK a corrompu les index secondaires DuckDB ART.
// Fix      : copie les données dans des tables temporaires, recrée les tables et leurs index,
//
//	réinsère les données depuis les tables temporaires.
//
// Usage :
//
//	go run ./cmd/repair-metadata/ [--db path/to/metadata.duckdb]
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

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

	// Repair challenge_template
	slog.Info("repairing challenge_template...")
	if err := repairTable(ctx, db, repairSpec{
		table:     "challenge_template",
		tempTable: "challenge_template_repair_tmp",
		createSQL: `CREATE TABLE challenge_template (
			id                  VARCHAR PRIMARY KEY,
			title_slug          VARCHAR NOT NULL,
			metric              VARCHAR NOT NULL,
			window_type         VARCHAR NOT NULL,
			window_value        VARCHAR,
			cadence             VARCHAR NOT NULL,
			eval_type           VARCHAR NOT NULL,
			mode_filter         VARCHAR NOT NULL DEFAULT 'universal',
			label_en            VARCHAR NOT NULL,
			label_fr            VARCHAR NOT NULL,
			description_en      VARCHAR,
			description_fr      VARCHAR,
			normal_target       DOUBLE NOT NULL,
			heroic_target       DOUBLE NOT NULL,
			legendary_target    DOUBLE NOT NULL,
			mythic_target       DOUBLE NOT NULL,
			schema_version      INTEGER NOT NULL DEFAULT 1,
			updated_at          TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)`,
		indexSQL: []string{
			`CREATE INDEX idx_ctmpl_title_cadence ON challenge_template(title_slug, cadence)`,
			`CREATE INDEX idx_ctmpl_metric ON challenge_template(metric)`,
		},
	}); err != nil {
		return fmt.Errorf("challenge_template: %w", err)
	}

	// Repair preset_arc_step first (FK dependency on preset_arc)
	slog.Info("repairing preset_arc_step...")
	if err := repairTable(ctx, db, repairSpec{
		table:     "preset_arc_step",
		tempTable: "preset_arc_step_repair_tmp",
		createSQL: `CREATE TABLE preset_arc_step (
			preset_arc_id   VARCHAR NOT NULL,
			position        INTEGER NOT NULL,
			template_id     VARCHAR NOT NULL,
			target_tier     VARCHAR NOT NULL,
			PRIMARY KEY (preset_arc_id, position)
		)`,
		indexSQL: nil,
	}); err != nil {
		return fmt.Errorf("preset_arc_step: %w", err)
	}

	// Repair preset_arc
	slog.Info("repairing preset_arc...")
	if err := repairTable(ctx, db, repairSpec{
		table:     "preset_arc",
		tempTable: "preset_arc_repair_tmp",
		createSQL: `CREATE TABLE preset_arc (
			id              VARCHAR PRIMARY KEY,
			title_slug      VARCHAR NOT NULL,
			title_en        VARCHAR NOT NULL,
			title_fr        VARCHAR NOT NULL,
			description_en  VARCHAR,
			description_fr  VARCHAR,
			schema_version  INTEGER NOT NULL DEFAULT 1,
			updated_at      TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)`,
		indexSQL: []string{
			`CREATE INDEX idx_parc_title ON preset_arc(title_slug)`,
		},
	}); err != nil {
		return fmt.Errorf("preset_arc: %w", err)
	}

	return nil
}

type repairSpec struct {
	table     string
	tempTable string
	createSQL string
	indexSQL  []string
}

func repairTable(ctx context.Context, db *sql.DB, spec repairSpec) error {
	exec := func(q string, args ...any) error {
		_, err := db.ExecContext(ctx, q, args...)
		return err
	}

	// 1. Copier les données dans une table temporaire (scan séquentiel, pas d'index)
	if err := exec(fmt.Sprintf(`CREATE TABLE %s AS SELECT * FROM %s`, spec.tempTable, spec.table)); err != nil {
		return fmt.Errorf("create temp table: %w", err)
	}

	// Vérifier combien de lignes sauvegardées
	var count int
	if err := db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM %s`, spec.tempTable)).Scan(&count); err != nil {
		return fmt.Errorf("count temp: %w", err)
	}
	slog.Info("data backed up", "table", spec.table, "rows", count)

	// 2. Supprimer la table corrompue (avec ses index)
	if err := exec(fmt.Sprintf(`DROP TABLE %s`, spec.table)); err != nil {
		// Cleanup
		_ = exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s`, spec.tempTable))
		return fmt.Errorf("drop original: %w", err)
	}

	// 3. Recréer la table avec schéma propre
	if err := exec(spec.createSQL); err != nil {
		// Tenter de restaurer
		_ = exec(fmt.Sprintf(`ALTER TABLE %s RENAME TO %s`, spec.tempTable, spec.table))
		return fmt.Errorf("recreate table: %w", err)
	}

	// 4. Réinsérer les données
	if err := exec(fmt.Sprintf(`INSERT INTO %s SELECT * FROM %s`, spec.table, spec.tempTable)); err != nil {
		return fmt.Errorf("reinsert data: %w", err)
	}
	slog.Info("data restored", "table", spec.table, "rows", count)

	// 5. Recréer les index
	for _, idx := range spec.indexSQL {
		if err := exec(idx); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}

	// 6. Supprimer la table temporaire
	if err := exec(fmt.Sprintf(`DROP TABLE %s`, spec.tempTable)); err != nil {
		return fmt.Errorf("drop temp: %w", err)
	}

	return nil
}
