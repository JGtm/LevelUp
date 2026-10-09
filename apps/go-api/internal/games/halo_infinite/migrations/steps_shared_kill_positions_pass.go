package migrations

// steps_shared_kill_positions_pass.go — `kill_positions` passe d'un arbitrage PAR CLÉ à un
// arbitrage PAR PASSE DE DÉCODAGE (lot 1.7, 2026-09-09).
//
// ─── LE DÉFAUT QUE CE STEP FERME ──────────────────────────────────────────────────────────
//
// Depuis G.2 (steps_appendonly_misc.go) la table est append-only et sa vue
// `kill_positions_latest` retenait la DERNIÈRE LIGNE PAR CLÉ `(match_id, killer_xuid,
// time_ms)`. Cet arbitrage-là ne sait pas RÉTRACTER : une position qu'un re-décodage ne
// retrouve plus n'est pas réécrite — elle n'est simplement PAS dans la nouvelle passe — et la
// vue continuait donc de servir A JAMAIS la ligne de la passe précédente, mélangée aux
// nouvelles. C'est exactement le constat qui a fait naître `decode_pass` sur
// `match_kill_events`, puis sur `kill_openings` (cf. steps_shared_kill_openings.go, section
// « l'unité de génération est la passe »). `kill_positions` était restée en arrière.
//
// Le cas n'est pas théorique : `replay.BuildKillPositions` n'écrit aucune ligne pour une mort
// dont ni le tueur ni la victime n'ont pu être localisés (bornes de trajectoire, joueur non
// résolu, film re-téléchargé plus court). Un décodeur amélioré en écarte d'autres. Sans
// rétractation, une position fausse survit à sa propre correction.
//
// ─── POURQUOI UN REBUILD, ET PAS UN `ALTER TABLE ADD COLUMN` ──────────────────────────────
//
// La colonne est `NOT NULL` : une ligne sans passe serait injointable au mécanisme (la vue ne
// pourrait ni la retenir ni l'écarter). Un `ADD COLUMN ... NOT NULL` sur une table peuplée
// n'a pas de valeur à donner aux lignes existantes, et un DEFAULT constant les fondrait TOUTES
// dans une seule passe inter-matchs — la première passe neuve d'UN match retirerait alors de
// la vue les lignes de TOUS les autres. Le rebuild CTAS de `migration.ApplyAppendOnlyRebuild`
// permet une expression PAR LIGNE : `'legacy-' || match_id`. Il apporte en prime la garde
// anti-perte (`rebuilt == before`, rollback intégral) et la reprise d'un crash mid-swap.
//
// ─── LA PASSE SYNTHÉTIQUE EST PAR MATCH, ET C'EST LE POINT DÉLICAT ────────────────────────
//
// Les lignes déjà en base n'ont JAMAIS été écrites en passes : elles viennent du chemin
// builder Halo 5 (`ingest.MapKillPositions`, à l'insertion du match) et des passes film
// Infinite antérieures, toutes indistinctes. Leur donner `legacy-<match_id>` les range dans UNE
// passe par match, donc la vue les sert TOUTES — l'état exact d'avant la migration — jusqu'à
// ce qu'une passe neuve arrive sur CE match et la remplace entière. Aucun match n'est
// vidé par la migration, aucun n'est affecté par le re-décodage d'un autre.
//
// ─── LA VUE ARBITRE PAR PASSE, PUIS DÉDOUBLONNE PAR CLÉ DANS LA PASSE RETENUE ─────────────
//
// C'est la SEULE différence avec `kill_openings_latest`, et elle est imposée par l'histoire de
// cette table-ci : sa sœur est née append-only, la nôtre a un passé. Regrouper tout le passé
// d'un match dans une seule passe synthétique y fait cohabiter des doublons de clé que
// l'ancienne vue par clé masquait (deux écritures successives sur le même match : re-décodage
// avant G.2, re-insertion builder). Servir la passe `legacy-<match_id>` telle quelle les
// RESSUSCITERAIT dans la vue — une régression introduite par la migration elle-même, qui
// double-compterait des morts chez tous les lecteurs (carte tactique, distances, isolement).
// Le second étage `ROW_NUMBER() = 1` par `(match_id, decode_pass, killer_xuid, time_ms)` rend
// donc à la vue sa cardinalité d'avant. Sur une passe BIEN FORMÉE — toute passe écrite par
// `persist.KillPositionPersister`, qui n'écrit qu'une ligne par clé — il est un no-op ; il ne
// borne QUE l'intérieur d'une passe et ne peut donc pas ressusciter une ligne d'une passe
// précédente, ce qui serait le défaut qu'on ferme ici.
//
// ─── STEP AU NOM NEUF, ET NON UNE MODIFICATION DE G.2 ─────────────────────────────────────
//
// Les migrations sont name-keyed : `shared_append_only_kill_positions_v1` a DÉJÀ été appliquée
// (locale et prod), la modifier ne rejouerait rien et ferait diverger deux schémas. Le nouveau
// nom rejoue partout, une seule fois. Il est placé APRÈS elle dans `migration.canonicalOrder`.

import (
	"database/sql"

	"levelup/go-api/internal/migration"
)

// sharedKillPositionsPassSteps — bascule de `kill_positions` sur un arbitrage par passe.
func sharedKillPositionsPassSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:     "shared_kill_positions_decode_pass_v1",
			TargetDB: migration.TargetShared,
			Description: "kill_positions : ajout de decode_pass (NOT NULL, passe synthetique legacy-<match_id> " +
				"pour les lignes existantes) et vue kill_positions_latest arbitrant par DERNIERE PASSE " +
				"ENTIERE par match — une position qu un re-decodage ne retrouve plus est enfin retractee",
			ApplySchema: applyKillPositionsDecodePass,
		},
	}
}

// applyKillPositionsDecodePass délègue au helper commun de rebuild append-only.
//
// `MarkerColumn: "decode_pass"` est ce qui rend le step idempotent ET rejouable APRÈS G.2 :
// la table porte déjà `id` (marqueur par défaut du helper), donc le marqueur par défaut
// dirait « déjà migrée » et le step ne ferait rien. `IDConditional: true` pour la même
// raison inverse — l'`id` existant est PRÉSERVÉ tel quel par le CTAS, jamais renuméroté (les
// lignes de la vue seraient sinon départagées par des id neufs sans rapport avec leur ordre
// d'écriture).
//
// `PostSwap` restaure ce que le CTAS ne transporte pas : le DEFAULT de `written_at`, la
// contrainte NOT NULL de la colonne neuve, et l'index de jointure. Seul
// `idx_kill_positions_lookup` est recréé : c'est le seul que la table portait en entrant
// (G.2 n'avait pas recréé `idx_kill_positions_match`), et une migration de forme n'ajoute
// pas d'index.
func applyKillPositionsDecodePass(db *sql.DB) error {
	return migration.ApplyAppendOnlyRebuild(db, migration.AppendOnlyRebuild{
		Table:         "kill_positions",
		IDSeq:         "kill_positions_seq",
		IDConditional: true,
		MarkerColumn:  "decode_pass",
		SyntheticCols: `'legacy-' || match_id AS decode_pass`,
		PostSwap: []string{
			`ALTER TABLE kill_positions ALTER COLUMN written_at SET DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)`,
			`ALTER TABLE kill_positions ALTER COLUMN decode_pass SET NOT NULL`,
			`CREATE INDEX IF NOT EXISTS idx_kill_positions_lookup ON kill_positions(match_id, killer_xuid, time_ms, written_at)`,
		},
		ViewSQL: `CREATE OR REPLACE VIEW kill_positions_latest AS
			SELECT * FROM kill_positions AS p
			QUALIFY p.decode_pass = FIRST_VALUE(p.decode_pass) OVER (
				PARTITION BY p.match_id ORDER BY p.written_at DESC, p.id DESC
			)
			AND ROW_NUMBER() OVER (
				PARTITION BY p.match_id, p.decode_pass, p.killer_xuid, p.time_ms
				ORDER BY p.written_at DESC, p.id DESC
			) = 1`,
	})
}
