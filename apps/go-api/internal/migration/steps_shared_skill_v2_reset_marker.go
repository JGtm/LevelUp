package migration

// steps_shared_skill_v2_reset_marker.go — éradication ART de
// player_skill_state_v2 (shared DB) — Phase 2 campagne #23046.
//
// **Pourquoi** : RecomputeLUSRCanonicalForPlayer (sync/lusr_full_recompute.go)
// faisait `DELETE FROM player_skill_state_v2 WHERE xuid=?` pour réinitialiser le
// watermark avant un replay LUSR v2 complet (post-import OpenSpartan). Ce DELETE
// retire les lignes du joueur de la PK(id) ET de l'index idx_pssv2 → vecteur
// DuckDB #23046, même sur un chemin rare/sérialisé.
//
// **Stratégie append-only (sentinelle)** : la table est déjà append-only
// (written_at, vue _latest = MAX par (xuid, playlist_group)). On ajoute une
// colonne `is_reset` : le reset INSÈRE une row sentinelle is_reset=TRUE par
// groupe (written_at courant) au lieu de DELETE. La vue _latest exclut la
// dernière génération si c'est un reset (WHERE NOT COALESCE(is_reset,FALSE)) →
// LoadState renvoie nil → le replay re-seed depuis les priors, puis INSÈRE des
// états frais (written_at postérieur) qui réapparaissent dans la vue.
//
// Idempotence : ADD COLUMN IF NOT EXISTS + CREATE OR REPLACE VIEW.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "player_skill_state_v2_reset_marker_v1",
		TargetDB:    TargetShared,
		Description: "player_skill_state_v2 : colonne is_reset (sentinelle reset append-only) + vue _latest filtrée — élimine DELETE WHERE xuid (#23046)",
		ApplySchema: func(db *sql.DB) error {
			// Garde d'existence : la table player_skill_state_v2 est créée par
			// shared_create_skill_v2_tables, désormais TITLE-OWNED (relocalisée voie B
			// dans games/halo_infinite/migrations). Ce reset_marker reste dans le
			// registre GLOBAL (ALTER idempotent nécessaire aux DB existantes). En prod,
			// le provider title-owned est câblé → la table existe quand ce step tourne.
			// Mais dans le binaire de test de internal/migration, le provider title-owned
			// est nil (cycle d'import) → la table n'est pas créée → on skippe proprement
			// (rien à reset). Sans cette garde, l'ALTER échoue (table absente).
			var n int
			if err := db.QueryRow(
				`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'player_skill_state_v2'`,
			).Scan(&n); err != nil {
				return err
			}
			if n == 0 {
				return nil
			}
			return execScript(db, `
				ALTER TABLE player_skill_state_v2 ADD COLUMN IF NOT EXISTS is_reset BOOLEAN DEFAULT FALSE;
				CREATE OR REPLACE VIEW player_skill_state_v2_latest AS
				SELECT s.id, s.xuid, s.playlist_group, s.mu, s.sigma, s.experience,
				       s.last_match_id, s.last_match_at, s.written_at
				FROM player_skill_state_v2 s
				JOIN (
					SELECT xuid, playlist_group, MAX(written_at) AS max_written_at
					FROM player_skill_state_v2
					GROUP BY xuid, playlist_group
				) m
					ON s.xuid = m.xuid
					AND s.playlist_group = m.playlist_group
					AND s.written_at = m.max_written_at
				WHERE NOT COALESCE(s.is_reset, FALSE);
			`)
		},
	})
}
