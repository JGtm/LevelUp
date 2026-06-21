package migration

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:     "lusr_chain_rework_v1",
		TargetDB: TargetPlayer,
		Description: "Wipe des ratings LUSR pour recompute complet avec les nouvelles chaînes" +
			" (arena_slayer / arena_objectif / btb / chaos remplacent ranked/arena/btb/fun).",
		ApplySchema: lusrChainRework,
	})
}

// lusrChainRework purge les lignes LUSR de match_skill_rank pour forcer un recompute.
//
// Append-only #23046 : PAS de `DELETE FROM match_skill_rank WHERE rating_type='LUSR'`
// — un DELETE per-row sur une table append-only INDEXÉE (PK id + idx_msr_*) est un
// vecteur ART (« Failed to delete all rows from index »), même au boot. On purge via
// rebuild CTAS (table sans index pendant la copie, index/PK reposés après), modèle
// dedup_record_history_v1 / applyAppendOnlyMatchSkillRank. Garde no-op si table absente
// ou aucune ligne LUSR (DB neuve).
func lusrChainRework(db *sql.DB) error {
	ctx := bootCtx()
	has, err := tableExists(db, "match_skill_rank")
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
		ALTER TABLE match_skill_rank ALTER COLUMN written_at SET DEFAULT now();
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
