package migration

// steps_shared_player_positions_appendonly.go — `match_player_positions` devient APPEND-ONLY
// (decision utilisateur 1 du plan v2, 2026-09-06).
//
// # CE QUI CHANGE, ET POURQUOI
//
// La table etait ecrite en DELETE-then-INSERT par match, par un OUTIL DE DIAGNOSTIC
// (`cmd/diag_weapons_v3 -positions -write`), sur le handle de LECTURE du pool. Deux defauts :
//
//	le VECTEUR ART      un `DELETE FROM ... WHERE match_id = ?` sur une table indexee est le
//	                    declencheur du bug DuckDB #23046 (ADR 0019/0026). L'outil tournait
//	                    « serveur arrete », donc hors pression concurrente — mais la table
//	                    devient desormais une PROJECTION DE L'ARTEFACT ecrite dans le cycle de
//	                    sync, c'est-a-dire exactement le regime que la doctrine interdit.
//	le PRODUCTEUR       les positions venaient d'un decodeur keyframe local a l'outil. Elles
//	                    viennent maintenant de l'ARTEFACT DE REJEU, comme toutes les autres
//	                    projections post-rangement (usage, statistiques d'Assaut, coup d'envoi).
//
// # L'UNITE DE GENERATION EST LA PASSE, PAS LA LIGNE
//
// Une projection porte TOUTES les positions d'un match : la remplacer ligne par ligne n'aurait
// aucun sens (une position n'a pas de cle naturelle — deux joueurs peuvent occuper le meme
// point au meme instant, et le schema d'origine le dit deja : « pas de PK contraignante »).
// Toutes les lignes d'une projection partagent donc `positions_pass` et `written_at`, et la vue
// `match_player_positions_latest` retient LA DERNIERE PASSE PAR MATCH — meme mecanique que
// `match_weapon_hit_distance_latest` et `match_usage_films_latest`.
//
// # LES LIGNES DEJA EN BASE SURVIVENT, EN UNE PASSE
//
// Le CTAS leur donne `positions_pass = 'legacy-diag'` : elles forment UNE generation coherente,
// servie par la vue tant qu'aucune projection ne la supersede. Sans cette colonne synthetique,
// des lignes ecrites a des millisecondes differentes par l'ancien DELETE+INSERT auraient ete
// eclatees en autant de « passes » d'une ligne, et la vue n'en aurait servi qu'une.

import "database/sql"

// legacyPositionsPass : l'identifiant de passe donne aux lignes ANTERIEURES a la conversion.
// Une valeur en toutes lettres, pas un UUID : un operateur qui lit la table doit voir d'ou
// viennent ces lignes.
const legacyPositionsPass = "legacy-diag"

func init() {
	Register(Migration{
		Name:     "shared_match_player_positions_appendonly_v1",
		TargetDB: TargetShared,
		Description: "Rebuild shared.match_player_positions en append-only (id PK + positions_pass + " +
			"vue match_player_positions_latest par PASSE) — decision 1 du plan v2 : la table devient " +
			"une projection de l'artefact de rejeu, ecrite dans le cycle de sync",
		ApplySchema: applyAppendOnlyPlayerPositions,
	})
}

func applyAppendOnlyPlayerPositions(db *sql.DB) error {
	return ApplyAppendOnlyRebuild(db, AppendOnlyRebuild{
		Table: "match_player_positions",
		IDSeq: "match_player_positions_seq",
		// `written_at` EXISTE DEJA au schema d'origine : seule la passe est synthetisee. Le
		// marqueur d'idempotence reste `id` (aucun ancien schema ne le porte).
		SyntheticCols: `'` + legacyPositionsPass + `' AS positions_pass`,
		PostSwap: []string{
			`ALTER TABLE match_player_positions ALTER COLUMN written_at SET DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)`,
			// UN SEUL INDEX, meme argument que match_weapon_shots et match_weapon_hit_distance :
			// DuckDB est colonnaire, le seul acces ponctuel reel est « les positions de CE
			// match », et chaque index en plus coute a l'INSERT et elargit la surface ART.
			`CREATE INDEX IF NOT EXISTS idx_match_player_positions_match ON match_player_positions(match_id, written_at)`,
		},
		ViewSQL: `CREATE OR REPLACE VIEW match_player_positions_latest AS
			SELECT p.* FROM match_player_positions AS p
			QUALIFY p.positions_pass = FIRST_VALUE(p.positions_pass) OVER (
				PARTITION BY p.match_id ORDER BY p.written_at DESC, p.id DESC
			)`,
	})
}
