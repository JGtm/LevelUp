package migration

// steps_player_fix_career_xp_total_default.go — supprime le DEFAULT 0 sur
// xp_total et corrige les rows corrompues.
//
// Contexte du bug : career_progression définissait `xp_total INTEGER DEFAULT 0`.
// Lors des syncs qui ne mettent à jour que la personnalisation (bannière, emblem,
// spartan_id), InsertCareerProgressionPartial omettait xp_total de l'INSERT et
// DuckDB écrivait 0 par défaut. Q7 utilisait COALESCE(xp_total, rank*1000) qui
// ne traite pas le 0 → l'UI affichait une chute à 0 XP dans l'Historique XP.
//
// Deux corrections :
//  1. UPDATE toutes les rows où xp_total=0 → NULL. PartialFromLive n'écrit
//     xp_total QUE si > 0 (guard explicite), donc 0 est toujours un artefact
//     du DEFAULT — jamais une valeur légitime (même en début de carrière).
//  2. ALTER TABLE pour retirer DEFAULT 0 sur xp_total → les futurs inserts partiels
//     sans xp_total écriront NULL (correctement ignoré par COALESCE côté lecture).
//
// Q7 a également été durcie (NULLIF + filtre WHERE) comme filet de sécurité.

import (
	"database/sql"
	"fmt"
	"log/slog"
)

const careerXPTotalFixMetaKey = "career_xp_total_default_fixed_v1"

func init() {
	Register(Migration{
		Name:        "fix_career_xp_total_default_zero",
		TargetDB:    TargetPlayer,
		Description: "Supprime DEFAULT 0 sur xp_total et corrige toutes les rows corrompues (xp_total=0 jamais légitime).",
		ApplySchema: func(db *sql.DB) error {
			// Idempotence : skip si déjà appliqué.
			hasMeta, err := tableExists(db, "sync_meta")
			if err != nil {
				return fmt.Errorf("fix_xp_default: check sync_meta: %w", err)
			}
			if hasMeta {
				var marker sql.NullString
				if scanErr := db.QueryRowContext(bootCtx(),
					`SELECT value FROM sync_meta WHERE key = ?`,
					careerXPTotalFixMetaKey,
				).Scan(&marker); scanErr == nil && marker.Valid {
					return nil
				}
			}

			hasTable, err := tableExists(db, "career_progression")
			if err != nil {
				return fmt.Errorf("fix_xp_default: check table: %w", err)
			}
			if !hasTable {
				return markXPTotalFixDone(db)
			}

			// Compter les rows corrompues avant nettoyage.
			var corrupt int
			if err := db.QueryRowContext(bootCtx(),
				`SELECT COUNT(*) FROM career_progression WHERE xp_total = 0`,
			).Scan(&corrupt); err != nil {
				return fmt.Errorf("fix_xp_default: count corrupt: %w", err)
			}

			// Étape 1 : nettoyer TOUTES les rows où xp_total=0.
			// PartialFromLive n'écrit xp_total QUE si > 0, donc 0 en base
			// est toujours un artefact du DEFAULT — jamais une vraie valeur.
			res, err := db.ExecContext(bootCtx(),
				`UPDATE career_progression SET xp_total = NULL WHERE xp_total = 0`)
			if err != nil {
				return fmt.Errorf("fix_xp_default: update null: %w", err)
			}
			fixed, _ := res.RowsAffected()

			// Étape 2 : retirer DEFAULT 0 sur xp_total (DuckDB supporte DROP DEFAULT).
			if _, err := db.ExecContext(bootCtx(),
				`ALTER TABLE career_progression ALTER COLUMN xp_total DROP DEFAULT`); err != nil {
				return fmt.Errorf("fix_xp_default: drop default xp_total: %w", err)
			}

			slog.Info("migration fix_career_xp_total_default: done",
				"corrupt_rows_found", corrupt,
				"rows_set_to_null", fixed,
			)
			return markXPTotalFixDone(db)
		},
	})
}

func markXPTotalFixDone(db *sql.DB) error {
	hasMeta, err := tableExists(db, "sync_meta")
	if err != nil || !hasMeta {
		return nil
	}
	if err := addColumnIfMissing(db, "sync_meta", "updated_at", "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"); err != nil {
		return fmt.Errorf("fix_xp_default: ensure updated_at: %w", err)
	}
	// SELECT-then-INSERT-or-UPDATE : résilient aux player DBs sans PK sur sync_meta
	// (DBs seed-demo ou legacy créées avant la migration add_sync_meta_pk).
	var exists bool
	_ = db.QueryRowContext(bootCtx(), `SELECT EXISTS(SELECT 1 FROM sync_meta WHERE key = ?)`, careerXPTotalFixMetaKey).Scan(&exists)
	if exists {
		_, err = db.ExecContext(bootCtx(), `UPDATE sync_meta SET value = 'true', updated_at = NOW() WHERE key = ?`, careerXPTotalFixMetaKey)
	} else {
		_, err = db.ExecContext(bootCtx(), `INSERT INTO sync_meta (key, value, updated_at) VALUES (?, 'true', NOW())`, careerXPTotalFixMetaKey)
	}
	if err != nil {
		return fmt.Errorf("fix_xp_default: mark done: %w", err)
	}
	return nil
}
