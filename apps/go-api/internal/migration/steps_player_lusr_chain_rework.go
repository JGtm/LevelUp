package migration

import "database/sql"

func init() {
	Register(Migration{
		Name:     "lusr_chain_rework_v1",
		TargetDB: TargetPlayer,
		Description: "Wipe des ratings LUSR pour recompute complet avec les nouvelles chaînes" +
			" (arena_slayer / arena_objectif / btb / chaos remplacent ranked/arena/btb/fun).",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.ExecContext(bootCtx(), `DELETE FROM match_skill_rank WHERE rating_type = 'LUSR'`)
			return err
		},
	})
}
