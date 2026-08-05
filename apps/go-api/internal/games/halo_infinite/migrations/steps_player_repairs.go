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
			Description: "Garantit player_match_enrichment append-only (id PK + vue) sur player DB legacy (no-op si déjà id ; jamais de PK match_id — append-only #23046)",
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

// repairPlayerMatchEnrichmentPK garantit que player_match_enrichment est
// append-only. Append-only #23046 : on ne pose JAMAIS de PK(match_id). Si la
// table porte déjà la PK technique `id`, elle est append-only → no-op. Sinon
// (legacy sans schéma append-only), on délègue à la conversion idempotente
// (RebuildPlayerMatchEnrichmentART → applyAppendOnlyMatchEnrichment).
func repairPlayerMatchEnrichmentPK(db *sql.DB) error {
	exists, err := migration.TableExists(db, "player_match_enrichment")
	if err != nil {
		return fmt.Errorf("repair pme PK: check table: %w", err)
	}
	if !exists {
		return nil
	}
	hasID, err := migration.ColumnExists(db, "player_match_enrichment", "id")
	if err != nil {
		return fmt.Errorf("repair pme PK: check id: %w", err)
	}
	if hasID {
		return nil // déjà append-only — rien à réparer (surtout pas de PK match_id)
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
	// Append-only #23046 (Phase 2) : si match_citations est en append-only
	// (generation_id présent), la PK composite (match_id, citation_name_norm) n'a
	// plus de sens (multiples générations par clé) — no-op. La PK technique id est
	// déjà posée par la conversion append-only.
	if hasGen, _ := migration.ColumnExists(db, "match_citations", "generation_id"); hasGen {
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

// rebuildCareerProgression reconstruit career_progression par swap (DROP + rename) pour
// défaire la corruption ART. Sentinelle sync_meta → une seule exécution par DB.
//
// INVARIANT (dérive corrigée le 2026-08-05) : ce DDL est un CLICHÉ FIGÉ du schéma de la
// table ; il tourne à la position 109 de canonicalOrder, donc APRÈS create_baseline_player_v1
// (66) et fix_career_xp_total_default_zero (83). Toute colonne/tout DEFAULT qu'il n'énumère
// pas est SILENCIEUSEMENT PERDU sur une DB fraîche (les DB de prod portent la sentinelle,
// donc n'ont jamais rejoué le swap : la dérive n'était visible que sur une DB reconstruite).
// Constaté en prod : last_fetch_status absent (career_live cassé, INSERT partiel en Binder
// Error) et xp_total repassé en DEFAULT 0 alors que fix_career_xp_total_default_zero venait
// de le retirer. TOUTE évolution de career_progression antérieure à la position 109 DOIT
// être répercutée ici — garde-rail : TestFreshMigratedPlayerDB_CareerProgressionSchema
// (internal/platform/duckdb).
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

	// Le SELECT de recopie ci-dessous ÉNUMÈRE last_fetch_status : sur une DB dont la
	// table pré-existe au schéma partiel (créée hors migrations), le bind échouerait.
	// AddColumnIfMissing est idempotent et no-ope sur le chemin nominal (la baseline
	// v1 émet déjà la colonne).
	if err := migration.AddColumnIfMissing(db, "career_progression", "last_fetch_status", "VARCHAR"); err != nil {
		return fmt.Errorf("rebuild_career: ensure last_fetch_status: %w", err)
	}

	stmts := []string{
		`CREATE TABLE career_progression__rebuilt (
			xuid VARCHAR,
			rank INTEGER,
			rank_name VARCHAR,
			rank_tier VARCHAR,
			current_xp INTEGER DEFAULT 0,
			xp_for_next_rank INTEGER DEFAULT 0,
			xp_total INTEGER,
			is_max_rank BOOLEAN DEFAULT FALSE,
			adornment_path VARCHAR DEFAULT '',
			spartan_id VARCHAR DEFAULT '',
			banner_image_url VARCHAR DEFAULT '',
			emblem_image_url VARCHAR DEFAULT '',
			backdrop_image_url VARCHAR DEFAULT '',
			recorded_at TIMESTAMP,
			last_fetch_status VARCHAR
		)`,
		`INSERT INTO career_progression__rebuilt
		 SELECT xuid, rank, rank_name, rank_tier, current_xp,
		        xp_for_next_rank, xp_total, is_max_rank,
		        adornment_path, spartan_id, banner_image_url,
		        emblem_image_url, backdrop_image_url, recorded_at,
		        last_fetch_status
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
