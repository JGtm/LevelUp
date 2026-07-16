// Package service — encounter_badges.go : calcul PARTAGÉ des badges de rencontre.
//
// Source UNIQUE réutilisée par la Match View (« Historique des rencontres », via
// convertEncounters) ET la page Carrière (« Top rencontres », via
// computeCareerEncounterBadges). Les deux surfaces réutilisent le même tableau
// MatchEncountersTable côté front et DOIVENT afficher le même jeu de badges que le
// hub Communauté > Relations : les 4 badges de rencontre (ordinal / allié+ / dur à
// cuire / coriace) ET les 5 badges « solid » (duo_gagnant / caméléon / ancien /
// recrue / proie_favorite), via relations.ComputeBadges. Le badge cross-jeu reste
// propre au hub Relations (dépendance cross-titre non câblée sur ces surfaces).
package service

import (
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/relations"
	"levelup/go-api/internal/domain"
)

// relationStatsFromEncounterStats projette l'identité d'une rencontre (+ stats riches
// EncounterStatsRaw optionnelles) en relations.RelationStats, pour réutiliser
// relations.ComputeBadges. TotalMatches reste calé sur countTogether (ordinal
// identique à l'ancien calcul narrative). Sans stats riches (hasStats == false), seul
// TotalMatches est renseigné → seul l'ordinal est attribuable.
func relationStatsFromEncounterStats(
	xuid, gamertag string,
	countTogether int,
	s domain.EncounterStatsRaw,
	hasStats bool,
) relations.RelationStats {
	st := relations.RelationStats{
		XUID:         xuid,
		Gamertag:     gamertag,
		TotalMatches: countTogether,
	}
	if !hasStats {
		return st
	}
	st.TeammateMatches = s.AllyCount
	st.EnemyMatches = s.EnemyCount
	st.TeammateWins = s.WinsAsAlly
	st.EnemyWins = s.WinsVsEnemy
	st.KillsDealt = s.KillsDealt
	st.DeathsSuffered = s.DeathsSuffered
	st.TeammateWinRate = encounterWinrate(s.WinsAsAlly, s.LossesAsAlly)
	st.EnemyWinRate = encounterWinrate(s.WinsVsEnemy, s.LossesVsEnemy)
	st.DuelRatio = duelRatio(s.KillsDealt, s.DeathsSuffered)
	if !s.FirstSeen.IsZero() {
		t := s.FirstSeen
		st.FirstSeen = &t
	}
	if !s.LastSeenAt.IsZero() {
		t := s.LastSeenAt
		st.LastSeen = &t
	}
	return st
}

// encounterWinrate : nil si W+L == 0 (pas assez de matchs pour calculer), sinon ratio
// 0..1. relations.ComputeBadges traite nil comme « pas d'attribution ».
func encounterWinrate(wins, losses int) *float64 {
	total := wins + losses
	if total == 0 {
		return nil
	}
	rate := analysis.WinRate(wins, total)
	return &rate
}

// convertRelationBadges : []relations.Badge -> []domain.MatchEncounterBadge. Le Kind
// est dérivé du dernier segment de la clé i18n (ex. narrative.encounter.duo_gagnant
// -> "duo_gagnant") ; le front ne s'en sert pas pour le rendu mais le DTO l'expose
// (parité avec l'ancien mapping narrative).
func convertRelationBadges(raw []relations.Badge) []domain.MatchEncounterBadge {
	if len(raw) == 0 {
		return nil
	}
	out := make([]domain.MatchEncounterBadge, 0, len(raw))
	for _, b := range raw {
		out = append(out, domain.MatchEncounterBadge{
			Kind:       badgeKindFromLabelKey(b.LabelKey),
			LabelKey:   b.LabelKey,
			ColorToken: b.ColorToken,
			Detail:     b.Detail,
		})
	}
	return out
}

// badgeKindFromLabelKey extrait le kind du dernier segment d'une clé i18n
// "namespace.<kind>" (ex. "narrative.encounter.ordinal" -> "ordinal"). Retourne la
// clé entière si aucun point.
func badgeKindFromLabelKey(labelKey string) string {
	if i := strings.LastIndex(labelKey, "."); i >= 0 && i+1 < len(labelKey) {
		return labelKey[i+1:]
	}
	return labelKey
}
