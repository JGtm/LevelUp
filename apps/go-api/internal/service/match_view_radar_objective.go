// Package service — match_view_radar_objective.go : l'axe « Objectif » du radar de la vue Match,
// la note d'objectif DU MATCH SEUL (narrative.ObjectiveScoreOf, 0-100), sans aucune comparaison à
// l'historique du joueur. La feuille de match porte déjà le bloc objectif de CHAQUE participant (les
// deux camps) : la part du joueur dans l'objectif de son équipe se calcule sur elle, sans requête.
package service

import (
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
)

// matchObjectiveScore rend la note d'objectif de `xuid` pour ce match depuis la feuille de match :
// prise et soutien compris, mêmes poids que la note de performance (frags de soutien à
// narrative.ObjectiveSupportFragWeight). ok == false — l'axe est alors retiré, jamais 0 — quand la
// feuille n'a aucune ligne d'objectif (Slayer, Halo 5) ou que le joueur n'y a ni ligne ni camp.
func matchObjectiveScore(scoreboard []domain.ScoreboardRaw, xuid string) (float64, bool) {
	rows := make([]narrative.ObjectiveTeamRow, 0, len(scoreboard))
	for i := range scoreboard {
		if row, ok := objectiveTeamRowFromScoreboard(scoreboard[i]); ok {
			rows = append(rows, row)
		}
	}
	return narrative.ObjectiveMatchScore(rows, xuid)
}

// objectiveTeamRowFromScoreboard ramène un participant de la feuille de match au vocabulaire du noyau
// (narrative.ObjectiveTeamRow). ok == false : pas de bloc objectif reconnu (mode sans objectif) ou pas
// d'identité.
func objectiveTeamRowFromScoreboard(row domain.ScoreboardRaw) (narrative.ObjectiveTeamRow, bool) {
	if row.XUID == "" || !row.Obj.HasObjective() {
		return narrative.ObjectiveTeamRow{}, false
	}
	fam, ok := narrative.ObjectiveFamilyOfRaw(row.Obj)
	if !ok {
		return narrative.ObjectiveTeamRow{}, false
	}
	out := narrative.ObjectiveTeamRow{
		XUID: row.XUID, TeamID: narrative.NoObjectiveTeam, Family: fam,
		Values: narrative.ObjectiveColumnValuesOf(row.Obj),
	}
	if row.TeamID != nil {
		out.TeamID = *row.TeamID
	}
	if row.TimePlayed != nil {
		out.TimePlayedSeconds = *row.TimePlayed
	}
	return out, true
}
