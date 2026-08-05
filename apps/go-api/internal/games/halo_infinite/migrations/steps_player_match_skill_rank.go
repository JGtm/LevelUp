package migrations

// steps_player_match_skill_rank.go — chaîne match_skill_rank (player DB) CONSOMMATRICE,
// déplacée depuis internal/migration/steps_player_*.go (Phase 1.5 b20, voie B).
//
// match_skill_rank est CRÉÉE par add_skill_rating_table (god-file player, RACINE restée
// globale jusqu'à b-player-root). Ces 6 steps l'ALTERent / la migrent en append-only /
// recréent la vue match_skill_rank_latest → consommateurs (créateur global + consommateur
// titre = safe ; RunForDB combine global+title trié par canonicalOrder). DML ART-prone :
// le rebuild append-only fait un CTAS swap (jamais d'UPDATE sur index ART).
//
// NB : RebuildMatchSkillRankART (steps_player_rebuild_match_skill_rank.go) n'est PAS ici —
// ce n'est pas un step enregistré mais un util runtime exporté, appelé par
// cmd/force_rebuild_art ; il reste dans le package migration.
//
// Helpers : migration.LoadTableColumns + migration.FirstWords (formes privées conservées
// globalement, b13). consts col* inlinées.

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/migration"
)

// playerMatchSkillRankSteps retourne la chaîne match_skill_rank title-owned (b20).
func playerMatchSkillRankSteps() []migration.Migration {
	return []migration.Migration{
		// NB (squash v1, N4) : player_add_expected_win_prob était le 1er step de cette
		// chaîne mais appartient au bloc squashé (borne M3a) → il est désormais dans
		// create_baseline_player_v1 (steps_player_baseline.go). expected_win_prob est donc
		// posée par la baseline sur DB vierge ; les steps ci-dessous (post-borne) restent.
		{
			Name:     "lusr_chain_rework_v1",
			TargetDB: migration.TargetPlayer,
			Description: "Wipe des ratings LUSR pour recompute complet avec les nouvelles chaînes" +
				" (arena_slayer / arena_objectif / btb / chaos remplacent ranked/arena/btb/fun).",
			ApplySchema: lusrChainRework,
		},
		{
			Name:        "player_append_only_match_skill_rank_v1",
			TargetDB:    migration.TargetPlayer,
			Description: "Rebuild match_skill_rank en append-only (id PK + written_at + vue latest) — élimine bug ART par construction",
			ApplySchema: applyAppendOnlyMatchSkillRank,
		},
		{
			Name:        "msr_written_at_default_now_repair_v1",
			TargetDB:    migration.TargetPlayer,
			Description: "Répare written_at de match_skill_rank (DEFAULT UTC + backfill des NULL) — migration append-only partielle sur certaines bases",
			ApplySchema: repairMatchSkillRankWrittenAt,
		},
		{
			Name:        "player_msr_view_lusr_over_v2_v1",
			TargetDB:    migration.TargetPlayer,
			Description: "match_skill_rank_latest : priorité CSR > LUSR > LUSR_V2 (fin du masquage LUSR par la row audit LUSR_V2)",
			ApplySchema: applyMSRViewLUSROverV2,
		},
		{
			Name:        "player_msr_view_priority_csr_v1",
			TargetDB:    migration.TargetPlayer,
			Description: "Recrée match_skill_rank_latest avec priorité CSR > LUSR par match_id (sémantique préservée sans garde SQL)",
			ApplySchema: applyMSRViewPriorityCSR,
		},
	}
}

// lusrChainRework purge les lignes LUSR de match_skill_rank pour forcer un recompute
// avec les nouvelles chaînes de playlists.
//
// Append-only #23046 : PAS de `DELETE FROM match_skill_rank WHERE rating_type='LUSR'`
// — un DELETE per-row sur une table append-only INDEXÉE (PK id + idx_msr_*) est un
// vecteur ART (« Failed to delete all rows from index »), même au boot. On purge via
// rebuild CTAS (table sans index pendant la copie, index/PK reposés après), modèle
// applyAppendOnlyMatchSkillRank. Garde no-op si table absente ou aucune ligne LUSR
// (DB neuve). canonicalOrder ordonne ce step APRÈS player_append_only_match_skill_rank_v1,
// donc match_skill_rank est déjà append-only (id PK, msr_seq, written_at, indexes) ici.
func lusrChainRework(db *sql.DB) error {
	ctx := migration.BootCtx()
	has, err := migration.TableExists(db, "match_skill_rank")
	if err != nil {
		return fmt.Errorf("lusr_chain_rework: check table: %w", err)
	}
	if !has {
		return nil
	}
	var n int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_skill_rank WHERE rating_type = 'LUSR'`).Scan(&n); err != nil {
		return fmt.Errorf("lusr_chain_rework: count LUSR: %w", err)
	}
	if n == 0 {
		return nil
	}
	// Rebuild CTAS conservant tout SAUF LUSR ; restaure PK(id) + DEFAULTs + 3 index +
	// vue match_skill_rank_latest (priorité CSR>LUSR, à l'identique de sync/schema.go).
	_, err = db.ExecContext(ctx, `
		DROP VIEW IF EXISTS match_skill_rank_latest;
		CREATE TABLE match_skill_rank__lusrwipe AS
			SELECT * FROM match_skill_rank WHERE rating_type <> 'LUSR';
		DROP TABLE match_skill_rank;
		ALTER TABLE match_skill_rank__lusrwipe RENAME TO match_skill_rank;
		ALTER TABLE match_skill_rank ADD PRIMARY KEY (id);
		ALTER TABLE match_skill_rank ALTER COLUMN id SET DEFAULT nextval('msr_seq');
		ALTER TABLE match_skill_rank ALTER COLUMN written_at SET DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP);
		CREATE INDEX IF NOT EXISTS idx_msr_match_lookup ON match_skill_rank(match_id, rating_type, written_at);
		CREATE INDEX IF NOT EXISTS idx_msr_rating_type ON match_skill_rank(rating_type);
		CREATE INDEX IF NOT EXISTS idx_msr_playlist    ON match_skill_rank(playlist_group);
		CREATE OR REPLACE VIEW match_skill_rank_latest AS
			SELECT * FROM match_skill_rank
			QUALIFY ROW_NUMBER() OVER (
				PARTITION BY match_id
				ORDER BY
					CASE rating_type WHEN 'CSR' THEN 0 WHEN 'LUSR' THEN 1 ELSE 2 END,
					start_time DESC NULLS LAST,
					written_at DESC,
					id DESC
			) = 1;
	`)
	if err != nil {
		return fmt.Errorf("lusr_chain_rework: rebuild CTAS: %w", err)
	}
	return nil
}

func applyAppendOnlyMatchSkillRank(db *sql.DB) error {
	ctx := migration.BootCtx()

	hasTable, err := migration.TableExists(db, "match_skill_rank")
	if err != nil {
		return fmt.Errorf("append-only msr: check table: %w", err)
	}
	if !hasTable {
		return nil
	}

	hasIDCol, err := migration.ColumnExists(db, "match_skill_rank", "id")
	if err != nil {
		return fmt.Errorf("append-only msr: check id column: %w", err)
	}
	if hasIDCol {
		return nil
	}

	cols, err := migration.LoadTableColumns(ctx, db, "match_skill_rank")
	if err != nil {
		return fmt.Errorf("append-only msr: enumerate columns: %w", err)
	}
	if len(cols) == 0 {
		return nil
	}
	colList := strings.Join(cols, ", ")

	var before int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_skill_rank`).Scan(&before); err != nil {
		return fmt.Errorf("append-only msr: count before: %w", err)
	}

	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS msr_seq START 1`,
		`DROP TABLE IF EXISTS match_skill_rank__appendonly`,
		fmt.Sprintf(`
			CREATE TABLE match_skill_rank__appendonly AS
			SELECT
				nextval('msr_seq') AS id,
				%s,
				CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP) AS written_at
			FROM match_skill_rank
		`, colList),
		`DROP TABLE match_skill_rank`,
		`ALTER TABLE match_skill_rank__appendonly RENAME TO match_skill_rank`,
		`ALTER TABLE match_skill_rank ADD PRIMARY KEY (id)`,
		`ALTER TABLE match_skill_rank ALTER COLUMN id SET DEFAULT nextval('msr_seq')`,
		`ALTER TABLE match_skill_rank ALTER COLUMN written_at SET DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)`,
		`CREATE INDEX IF NOT EXISTS idx_msr_match_lookup ON match_skill_rank(match_id, rating_type, written_at)`,
		`CREATE INDEX IF NOT EXISTS idx_msr_rating_type ON match_skill_rank(rating_type)`,
		`CREATE INDEX IF NOT EXISTS idx_msr_playlist ON match_skill_rank(playlist_group)`,
		`CREATE OR REPLACE VIEW match_skill_rank_latest AS
			SELECT * FROM match_skill_rank
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, rating_type ORDER BY written_at DESC, id DESC) = 1`,
	}
	for _, sqlStmt := range stmts {
		if _, err := db.ExecContext(ctx, sqlStmt); err != nil {
			return fmt.Errorf("append-only msr: step (%s): %w",
				migration.FirstWords(sqlStmt, 3), err)
		}
	}

	var after int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_skill_rank`).Scan(&after); err != nil {
		return fmt.Errorf("append-only msr: count after: %w", err)
	}

	slog.InfoContext(ctx, "append-only match_skill_rank: migration appliquée (ART eradication phase 2.B)",
		"rows_before", before,
		"rows_after", after,
		"columns_preserved", len(cols),
	)
	return nil
}

func repairMatchSkillRankWrittenAt(db *sql.DB) error {
	ctx := migration.BootCtx()

	hasTable, err := migration.TableExists(db, "match_skill_rank")
	if err != nil {
		return fmt.Errorf("repair written_at: check table: %w", err)
	}
	if !hasTable {
		return nil
	}
	hasCol, err := migration.ColumnExists(db, "match_skill_rank", "written_at")
	if err != nil {
		return fmt.Errorf("repair written_at: check column: %w", err)
	}
	if !hasCol {
		return nil
	}

	if _, err := db.ExecContext(ctx, `ALTER TABLE match_skill_rank ALTER COLUMN written_at SET DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)`); err != nil {
		return fmt.Errorf("repair written_at: ALTER SET DEFAULT: %w", err)
	}
	slog.InfoContext(ctx, "match_skill_rank: written_at DEFAULT UTC (ré)appliqué (réparation migration append-only partielle)")
	return nil
}

func applyMSRViewLUSROverV2(db *sql.DB) error {
	ctx := migration.BootCtx()

	hasIDCol, err := migration.ColumnExists(db, "match_skill_rank", "id")
	if err != nil {
		return fmt.Errorf("msr_view_lusr_over_v2: check id column: %w", err)
	}
	if !hasIDCol {
		return nil
	}

	const stmt = `
		CREATE OR REPLACE VIEW match_skill_rank_latest AS
			SELECT * FROM match_skill_rank
			QUALIFY ROW_NUMBER() OVER (
				PARTITION BY match_id
				ORDER BY
					CASE rating_type WHEN 'CSR' THEN 0 WHEN 'LUSR' THEN 1 ELSE 2 END,
					written_at DESC,
					id DESC
			) = 1
	`
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("msr_view_lusr_over_v2: recreate view: %w", err)
	}

	slog.InfoContext(ctx, "match_skill_rank_latest: vue recréée avec priorité CSR > LUSR > LUSR_V2")
	return nil
}

func applyMSRViewPriorityCSR(db *sql.DB) error {
	ctx := migration.BootCtx()

	hasIDCol, err := migration.ColumnExists(db, "match_skill_rank", "id")
	if err != nil {
		return fmt.Errorf("msr_view_priority_csr: check id column: %w", err)
	}
	if !hasIDCol {
		return nil
	}

	const stmt = `
		CREATE OR REPLACE VIEW match_skill_rank_latest AS
			SELECT * FROM match_skill_rank
			QUALIFY ROW_NUMBER() OVER (
				PARTITION BY match_id
				ORDER BY
					CASE rating_type WHEN 'CSR' THEN 0 WHEN 'LUSR' THEN 1 ELSE 2 END,
					start_time DESC NULLS LAST, written_at DESC,
					id DESC
			) = 1
	`
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("msr_view_priority_csr: recreate view: %w", err)
	}

	slog.InfoContext(ctx, "match_skill_rank_latest: vue recréée avec priorité CSR > LUSR (Phase 2.E)")
	return nil
}
