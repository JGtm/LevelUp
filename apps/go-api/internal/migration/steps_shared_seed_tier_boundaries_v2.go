package migration

// steps_shared_seed_tier_boundaries_v2.go — Phase 3e v2 du chantier LUSR v2.
//
// Seed les seuils tier (Bronze..Onyx) dans lusr_hyperparams_v2 pour chaque
// playlist_group LUSR. Permet au sysadmin / Phase 5 batch de ré-écrire ces
// seuils SQL sans toucher au code Go.
//
// Idempotent : INSERT ... WHERE NOT EXISTS sur la clé (playlist_group, name)
// — re-run = 0 row inséré. La table est append-only avec vue `_latest`,
// donc même si on ré-insérait, la vue retournerait la version la plus
// récente. Mais éviter les inserts inutiles garde la table propre.
//
// Source : "phase_3e_v2_default" — signifie "valeurs initiales Phase 3e v2,
// pas encore ré-estimées par batch". Une future ré-estimation (Phase 5)
// ajoutera des rows avec source="batch_YYYY_MM" qui domineront via le
// MAX(written_at) de la vue _latest.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_seed_tier_boundaries_v2",
		TargetDB:    TargetShared,
		Description: "LUSR v2 Phase 3e v2 — seed des tier_boundary_* (Bronze..Onyx) dans lusr_hyperparams_v2",
		ApplySchema: func(db *sql.DB) error {
			return nil // pas de DDL — la table existe déjà (migration shared_create_skill_v2_tables)
		},
		ApplyBackfill: func(db *sql.DB) error {
			// Seuils Phase 3e v2 (cf. internal/analysis/skill_v2/tier.go DefaultTierBoundaries).
			// Largeurs non uniformes : Bronze large (queue), Or = bulk pop,
			// Platine étroit, Diamant large, Onyx ouvert top ~10%.
			boundaries := []struct {
				name  string
				value float64
			}{
				{"tier_boundary_bronze", 0.0},
				{"tier_boundary_silver", 21.0},
				{"tier_boundary_gold", 22.0},
				{"tier_boundary_platinum", 25.0},
				{"tier_boundary_diamond", 25.8},
				{"tier_boundary_onyx", 27.0},
			}
			// Seed pour les 4 playlist_groups LUSR. Phase 5 batch pourra
			// ré-estimer par groupe si la distribution diverge.
			groups := []string{"arena_slayer", "arena_objectif", "btb", "chaos"}
			for _, g := range groups {
				for _, b := range boundaries {
					_, err := db.ExecContext(bootCtx(), `
						INSERT INTO lusr_hyperparams_v2 (playlist_group, name, value, source)
						SELECT ?, ?, ?, 'phase_3e_v2_default'
						WHERE NOT EXISTS (
							SELECT 1 FROM lusr_hyperparams_v2
							WHERE playlist_group = ? AND name = ?
						)`,
						g, b.name, b.value, g, b.name)
					if err != nil {
						return err
					}
				}
			}
			return nil
		},
	})
}
