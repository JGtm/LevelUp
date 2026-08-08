package migration

// steps_shared_objective_score_drop.go — retrait de match_objective_score_timeline
// (v7.5 lot 3, 2026-08-08).
//
// POURQUOI LA TABLE PART. Elle n'avait qu'un producteur, le décodeur
// internal/analysis/objectivescore, supprimé dans le même commit : le lot 2 l'a
// confronté à des octets de films réels et sa courbe Strongholds s'est révélée ne
// PAS être un score per-équipe (valeur brute plafonnée quel que soit le final réel,
// qui ne « retombait dessus » que par calibration). Son successeur
// analysis/objectiveevents.ScoreCurve fait mieux sur tous les axes — per-joueur, à
// la milliseconde, sans calibration.
//
// ÉTABLI SUR PIÈCES AVANT DE COUPER (2026-08-08) :
//   - lecteurs applicatifs de la table : ZÉRO. ObjectiveScoreRepo.LoadMatch n'avait
//     aucun appelant hors son propre test — ni port, ni service, ni handler, ni wire.
//   - écrivains : le seul était `cmd/diag_weapons_v3 -score -write`, un CLI manuel
//     dont les chemins par défaut pointent un poste de développement.
//   - contenu : 0 ligne sur la base locale. Rien à préserver.
//
// POURQUOI ON NE GARDE PAS LE SCHÉMA « POUR LE SUCCESSEUR ». Sa forme ne convient
// pas : elle est per-équipe (team_0_score/team_1_score) quand le successeur produit
// du per-joueur, et elle s'écrit en DELETE-then-INSERT sur une PK (match_id, time_ms)
// — exactement le motif ART qu'ADR 0026 éradique. La persistance prescrite pour la
// courbe de score est append-only + vue `_latest` (handoff MODE_SCORE_CHAINE §6.4).
// Réutiliser cette table reviendrait à recréer la dette ; le jour venu, le successeur
// crée la sienne, append-only, avec ses propres tests de vérité terrain.
//
// NOM NEUF, PAS D'EXTENSION DU STEP EXISTANT. `shared_objective_score_v1` est inscrit
// dans schema_migrations sur les bases déployées ; runSteps saute tout step déjà
// inscrit, donc y ajouter un DROP ne serait JAMAIS rejoué là où il compte. Le retrait
// passe par ce step au nom neuf (même leçon ledger que la campagne UTC S6c).
//
// Idempotent (DROP TABLE IF EXISTS) : no-op sur une base fraîche, où la table n'est
// plus jamais créée puisque le step de création est parti avec le décodeur.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_objective_score_v1_drop",
		TargetDB:    TargetShared,
		Description: "Supprime match_objective_score_timeline (décodeur objectivescore retiré, v7.5 lot 3)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `DROP TABLE IF EXISTS match_objective_score_timeline;`)
		},
	})
}
