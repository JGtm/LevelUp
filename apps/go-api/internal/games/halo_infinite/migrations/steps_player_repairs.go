package migrations

// steps_player_repairs.go — migrations correctives/rebuild player CONSOMMATRICES,
// déplacées depuis internal/migration/steps_player_repair_pk.go +
// steps_player_rebuild_career_progression.go (Phase 1.5 b21, voie B).
//
// 3 steps : repair PK manquante sur player_match_enrichment + match_citations
// (player DB legacy), rebuild career_progression (défait corruption ART). Tous
// consommateurs de tables créées par le god-file player (RACINE globale). repair_pme
// réutilise migration.RebuildPlayerMatchEnrichmentART (util runtime resté dans le
// package migration, appelé aussi par cmd/force_rebuild_art). Helpers
// migration.LoadTableColumns + migration.FirstWords (b13). col* inlinés.

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/migration"
)

const careerProgressionRebuildMetaKey = "career_progression_rebuilt_v1"

// playerRepairSteps retourne les migrations repair/rebuild player title-owned (b21).
func playerRepairSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:        "repair_player_match_enrichment_primary_key",
			TargetDB:    migration.TargetPlayer,
			Description: "Reconstruit player_match_enrichment avec PRIMARY KEY (match_id) quand elle manque (player DB legacy CREATE TABLE IF NOT EXISTS)",
			ApplySchema: repairPlayerMatchEnrichmentPK,
		},
		{
			Name:        "repair_match_citations_primary_key",
			TargetDB:    migration.TargetPlayer,
			Description: "Reconstruit match_citations avec PRIMARY KEY (match_id, citation_name_norm) quand elle manque (player DB legacy)",
			ApplySchema: repairMatchCitationsPK,
		},
		{
			Name:        "rebuild_career_progression_defeat_art_corruption",
			TargetDB:    migration.TargetPlayer,
			Description: "Rebuild career_progression via swap pour défaire la corruption ART (WHERE xuid = ? retournait sous-ensemble strict — diag 2026-05-21).",
			ApplySchema: rebuildCareerProgression,
		},
	}
}

// repairPlayerMatchEnrichmentPK pose la PK (match_id) si absente, via le rebuild CTAS
// dynamique exporté du package migration (préserve les colonnes additives).
func repairPlayerMatchEnrichmentPK(db *sql.DB) error {
	exists, err := migration.TableExists(db, "player_match_enrichment")
	if err != nil {
		return fmt.Errorf("repair pme PK: check table: %w", err)
	}
	if !exists {
		return nil
	}
	hasPK, err := migration.HasPrimaryKey(db, "player_match_enrichment")
	if err != nil {
		return fmt.Errorf("repair pme PK: check PK: %w", err)
	}
	if hasPK {
		return nil
	}
	return migration.RebuildPlayerMatchEnrichmentART(migration.BootCtx(), db)
}

// repairMatchCitationsPK pose la PK composite (match_id, citation_name_norm) si absente.
func repairMatchCitationsPK(db *sql.DB) error {
	exists, err := migration.TableExists(db, "match_citations")
	if err != nil {
		return fmt.Errorf("repair citations PK: check table: %w", err)
	}
	if !exists {
		return nil
	}
	hasPK, err := migration.HasPrimaryKey(db, "match_citations")
	if err != nil {
		return fmt.Errorf("repair citations PK: check PK: %w", err)
	}
	if hasPK {
		return nil
	}
	if err := migration.AddColumnIfMissing(db, "match_citations", "citation_name_norm", "VARCHAR"); err != nil {
		return err
	}
	if err := migration.AddColumnIfMissing(db, "match_citations", "value", "INTEGER DEFAULT 1"); err != nil {
		return err
	}
	cols, err := migration.LoadTableColumns(migration.BootCtx(), db, "match_citations")
	if err != nil {
		return fmt.Errorf("repair citations PK: columns: %w", err)
	}
	if len(cols) == 0 {
		return nil
	}
	orderBy := "created_at DESC NULLS LAST"
	if ok, _ := migration.ColumnExists(db, "match_citations", "created_at"); !ok {
		orderBy = "match_id"
	}
	script := fmt.Sprintf(`
		CREATE TABLE match_citations__pkfix AS
			SELECT %s FROM (
				SELECT *, ROW_NUMBER() OVER (
					PARTITION BY match_id, citation_name_norm ORDER BY %s
				) AS __rn
				FROM match_citations
				WHERE match_id IS NOT NULL AND citation_name_norm IS NOT NULL
			) WHERE __rn = 1;
		DROP TABLE match_citations;
		ALTER TABLE match_citations__pkfix RENAME TO match_citations;
		ALTER TABLE match_citations ADD PRIMARY KEY (match_id, citation_name_norm);
	`, strings.Join(cols, ", "), orderBy)
	return migration.ExecScript(db, script)
}

func rebuildCareerProgression(db *sql.DB) error {
	hasMeta, err := migration.TableExists(db, "sync_meta")
	if err != nil {
		return fmt.Errorf("rebuild_career: check sync_meta: %w", err)
	}
	if hasMeta {
		var marker sql.NullString
		if err := db.QueryRowContext(migration.BootCtx(), `SELECT value FROM sync_meta WHERE key = ?`,
			careerProgressionRebuildMetaKey).Scan(&marker); err == nil && marker.Valid {
			return nil
		}
	}

	hasTable, err := migration.TableExists(db, "career_progression")
	if err != nil {
		return fmt.Errorf("rebuild_career: check table: %w", err)
	}
	if !hasTable {
		return markCareerRebuildDone(db)
	}

	var before int
	if err := db.QueryRowContext(migration.BootCtx(), `SELECT COUNT(*) FROM career_progression`).Scan(&before); err != nil {
		return fmt.Errorf("rebuild_career: count before: %w", err)
	}

	stmts := []string{
		`CREATE TABLE career_progression__rebuilt (
			xuid VARCHAR,
			rank INTEGER,
			rank_name VARCHAR,
			rank_tier VARCHAR,
			current_xp INTEGER DEFAULT 0,
			xp_for_next_rank INTEGER DEFAULT 0,
			xp_total INTEGER DEFAULT 0,
			is_max_rank BOOLEAN DEFAULT FALSE,
			adornment_path VARCHAR DEFAULT '',
			spartan_id VARCHAR DEFAULT '',
			banner_image_url VARCHAR DEFAULT '',
			emblem_image_url VARCHAR DEFAULT '',
			backdrop_image_url VARCHAR DEFAULT '',
			recorded_at TIMESTAMP
		)`,
		`INSERT INTO career_progression__rebuilt
		 SELECT xuid, rank, rank_name, rank_tier, current_xp,
		        xp_for_next_rank, xp_total, is_max_rank,
		        adornment_path, spartan_id, banner_image_url,
		        emblem_image_url, backdrop_image_url, recorded_at
		 FROM career_progression`,
		`DROP TABLE career_progression`,
		`ALTER TABLE career_progression__rebuilt RENAME TO career_progression`,
	}
	for _, sqlStmt := range stmts {
		if _, err := db.ExecContext(migration.BootCtx(), sqlStmt); err != nil {
			return fmt.Errorf("rebuild_career: swap step (%s): %w",
				migration.FirstWords(sqlStmt, 3), err)
		}
	}

	var after int
	if err := db.QueryRowContext(migration.BootCtx(), `SELECT COUNT(*) FROM career_progression`).Scan(&after); err != nil {
		return fmt.Errorf("rebuild_career: count after: %w", err)
	}
	slog.Info("migration rebuild_career_progression: table rebuilt (ART corruption defeated)",
		"rows_before_scan", before,
		"rows_after_rebuild", after,
	)
	return markCareerRebuildDone(db)
}

func markCareerRebuildDone(db *sql.DB) error {
	hasMeta, err := migration.TableExists(db, "sync_meta")
	if err != nil || !hasMeta {
		return nil
	}
	if err := migration.AddColumnIfMissing(db, "sync_meta", "updated_at", "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"); err != nil {
		return fmt.Errorf("rebuild_career: ensure updated_at: %w", err)
	}
	_, err = db.ExecContext(migration.BootCtx(), `
		INSERT INTO sync_meta (key, value, updated_at)
		VALUES (?, 'true', NOW())
		ON CONFLICT (key) DO UPDATE SET
			value = 'true',
			updated_at = NOW()
	`, careerProgressionRebuildMetaKey)
	if err != nil {
		return fmt.Errorf("rebuild_career: mark done: %w", err)
	}
	return nil
}
