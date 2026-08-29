package migrations

// steps_shared_team_rounds.go — LES MANCHES D'UN MATCH, au registre.
//
// POURQUOI CES COLONNES EXISTENT. Sur les modes qui se decident aux MANCHES, le score
// d'equipe rendu par l'API (`CoreStats.Score`, deja au registre sous team_{0,1}_score) est un
// CUMUL DE POINTS sur toutes les manches — il ne dit pas qui a gagne. Mesure du 2026-08-29 sur
// les 1 942 matchs a score du corpus (`.ai/V7.5/RAPPORT_MANCHES_2026-08-29.md`) : sur 4 matchs
// Oddball, l'equipe VICTORIEUSE affiche MOINS de points que la perdante. Le compte de manches,
// lui, tranche — et l'API le publie depuis toujours (`CoreStats.RoundsWon/RoundsLost/RoundsTied`,
// jamais lu jusqu'ici).
//
// CE QUE PORTE CHAQUE COLONNE :
//   - team_0_rounds_won / team_1_rounds_won : manches GAGNEES par chaque camp.
//   - rounds_total : nombre de manches JOUEES, pris comme le MAX des totaux des deux camps.
//     Le max, et non le total d'un camp : 4 matchs abandonnes du corpus creditent 1 manche a
//     un camp et 0 a l'autre (rapport §4.1). Une manche NULLE se deduit
//     (rounds_total - rw0 - rw1) : pas de colonne dediee pour une valeur derivable.
//
// AUCUN BACKFILL SQL ICI, ET C'EST VOULU. La valeur ne se derive d'aucune table locale : elle
// n'existe que dans le payload de l'API. Les lignes anterieures restent donc a NULL jusqu'a
// `cmd/backfill-team-rounds` (INSERT nu de persistMatchRegistry : un re-sync ne reecrit jamais
// une ligne existante). NULL = « on ne sait pas », et le lecteur retombe sur les points —
// jamais un zero substitue, qui se lirait comme « zero manche gagnee ».

import (
	"database/sql"

	"levelup/go-api/internal/migration"
)

// sharedTeamRoundsSteps retourne la migration additive des manches sur match_registry.
func sharedTeamRoundsSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:        "add_team_rounds_to_match_registry",
			TargetDB:    migration.TargetShared,
			Description: "Colonnes team_0/1_rounds_won + rounds_total sur match_registry (manches gagnees par camp, source CoreStats.RoundsWon/Lost/Tied) — additif, aucun backfill SQL possible (donnee API uniquement)",
			ApplySchema: func(db *sql.DB) error {
				if err := migration.AddColumnIfMissing(db, "match_registry", "team_0_rounds_won", "SMALLINT"); err != nil {
					return err
				}
				if err := migration.AddColumnIfMissing(db, "match_registry", "team_1_rounds_won", "SMALLINT"); err != nil {
					return err
				}
				return migration.AddColumnIfMissing(db, "match_registry", "rounds_total", "SMALLINT")
			},
		},
	}
}
