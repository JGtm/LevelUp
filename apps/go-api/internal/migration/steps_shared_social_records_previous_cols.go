package migration

// steps_shared_social_records_previous_cols.go — étend player_records_history
// (append-only) avec previous_value / previous_achieved_at, puis recrée la vue
// player_records_latest pour les exposer.
//
// Contexte (fix 2026-05-30) : le chemin progression V2 (PersonalRecordsRepo)
// est migré de l'UPSERT sur la table legacy player_records vers l'append-only
// player_records_history (via SocialPersister.AppendPlayerRecord), pour éliminer
// le dernier site ON CONFLICT à risque ART. Or l'API /records expose
// previous_value / previous_achieved_at (le PB précédent), colonnes présentes
// dans la table legacy mais ABSENTES de player_records_history. Sans cette
// migration, basculer perdrait ces champs.
//
// Le tie-break `id DESC` dans la vue lève l'indéterminisme quand deux écritures
// partagent le même written_at (CURRENT_TIMESTAMP identique dans une même TX).
//
// GAP one-shot ACCEPTÉ : le backfill initial (create_player_records_history_append_only)
// a copié player_records → _history AVANT que ces colonnes existent ; les rows
// déjà migrées ont donc previous_value/previous_achieved_at = NULL. Conséquence :
// les PB pré-existants exposent un "previous best" vide via /records JUSQU'AU
// prochain PB battu (qui repeuple previous_*). On accepte cette dégradation
// cosmétique one-shot plutôt qu'un UPDATE de réconciliation (mutation risquée
// dans une migration pour un champ d'affichage qui s'auto-corrige).
//
// Idempotente : addColumnIfMissing no-op si la colonne existe déjà ; la vue est
// recréée via CREATE OR REPLACE. S'appuie sur l'ordre alphabétique des fichiers
// (records_append_only < records_previous_cols) → player_records_history existe
// déjà quand cette migration tourne.

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:        "player_records_history_previous_cols_v1",
		TargetDB:    TargetSharedSocial,
		Description: "Ajoute previous_value/previous_achieved_at à player_records_history + recrée la vue player_records_latest (cols previous_* + tie-break id DESC).",
		ApplySchema: func(db *sql.DB) error {
			has, err := tableExists(db, "player_records_history")
			if err != nil {
				return fmt.Errorf("records_previous_cols: check table: %w", err)
			}
			if !has {
				// Garde défensive : en pratique inatteignable car l'ordre
				// alphabétique des fichiers fait tourner
				// create_player_records_history_append_only dans le MÊME cycle
				// juste avant. Si elle se déclenchait, l'erreur ferait avorter le
				// boot (RunForDB propage, pas de retry intra-cycle) — la migration
				// ne serait pas marquée done et serait re-tentée au prochain boot.
				return fmt.Errorf("records_previous_cols: player_records_history absente (ordre de migration brisé ?)")
			}

			if err := addColumnIfMissing(db, "player_records_history", "previous_value", "DOUBLE"); err != nil {
				return fmt.Errorf("records_previous_cols: add previous_value: %w", err)
			}
			if err := addColumnIfMissing(db, "player_records_history", "previous_achieved_at", "TIMESTAMP"); err != nil {
				return fmt.Errorf("records_previous_cols: add previous_achieved_at: %w", err)
			}

			if _, err := db.ExecContext(bootCtx(), `
				CREATE OR REPLACE VIEW player_records_latest AS
				SELECT DISTINCT ON (xuid, metric, period)
					id,
					xuid,
					metric,
					period,
					value,
					achieved_at,
					achieved_match_id,
					previous_value,
					previous_achieved_at,
					written_at
				FROM player_records_history
				ORDER BY xuid, metric, period, written_at DESC, id DESC;
			`); err != nil {
				return fmt.Errorf("records_previous_cols: recreate view: %w", err)
			}
			return nil
		},
	})
}
