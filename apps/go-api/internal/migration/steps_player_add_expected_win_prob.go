// Package migration — steps_player_add_expected_win_prob.go : LUSR v2 Sprint 1.A.
//
// Ajoute la colonne `expected_win_prob FLOAT` à match_skill_rank. Elle porte la
// probabilité de victoire pré-match de l'équipe du joueur, calculée par le LUSR
// v2 au moment de l'écriture canonique (Stratégie C) à partir des ratings
// AVANT-match des participants. NULL pour les rows héritées v1 et pour les
// chemins CSR (qui n'écrivent pas cette colonne).
//
// Additive et idempotente (addColumnIfMissing) — aucune réécriture de table,
// donc aucun risque ART. Ordre d'exécution sans importance : la colonne est
// préservée par le CTAS de la migration append-only (colList) si appliquée
// avant, ou simplement ajoutée à la table reconstruite si appliquée après.

package migration

import "database/sql"

func init() {
	Register(Migration{
		Name:        "player_add_expected_win_prob",
		TargetDB:    TargetPlayer,
		Description: "Colonne expected_win_prob sur match_skill_rank — proba de victoire pré-match (LUSR v2 Sprint 1.A)",
		ApplySchema: func(db *sql.DB) error {
			return addColumnIfMissing(db, "match_skill_rank", "expected_win_prob", colFloat)
		},
	})
}
