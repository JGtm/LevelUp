package sessionusage

// objective_row.go — LA LIGNE D'OBJECTIF PAR RÔLE (prendre / défendre / tenir) d'un match et d'un
// joueur, lue dans `match_objective_stats_latest` (les deux camps).
//
// La classification colonne -> rôle est narrative.ObjectiveRoleColumns (objective_roles.go), source
// unique dont la couche repo génère ses sommes par rôle. Lue par l'échantillon d'objectif de la page
// Tendances (`trends.ObjectiveSamples`).

import "levelup/go-api/internal/analysis/narrative"

// ObjectiveRow — une ligne (match, joueur) déjà projetée par rôle (repo :
// ObjectiveStatsRepo.LoadObjectiveRoleRows, sommes générées depuis
// narrative.ObjectiveRoleColumns).
type ObjectiveRow struct {
	MatchID     string
	XUID        string
	Family      narrative.ObjectiveFamily
	Take        float64
	Defend      float64
	HoldSeconds float64
}
