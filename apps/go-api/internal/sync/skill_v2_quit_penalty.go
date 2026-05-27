package sync

// skill_v2_quit_penalty.go — détection quitter + ordering primary/secondary
// + construction des CountInputs avec quit flags.
//
// Extrait de skill_v2_shadow.go (2026-05-27) pour respecter le seuil 500L.
// Concern unique : transformer un roster en signal v2 "quit penalty" avec
// l'ordering produit "1er quitter = delta plein, suivants = 50%".

import (
	skillv2 "levelup/go-api/internal/analysis/skill_v2"
)

// QuitSecondaryFactor : multiplicateur appliqué aux quitters non-primaires.
// 0.5 = 50% du delta plein. Décision produit 2026-05-27 : le 1er quitter
// (= plus petit last_leave_time, fallback time_played_seconds) est responsable,
// les suivants subissent la cascade et reçoivent un malus réduit.
//
// Si modifié, garder le ratio plein/réduit ≥ 2× pour préserver l'incitation
// à NE PAS être le premier quitter.
const QuitSecondaryFactor = 0.5

// isQuitter détecte un joueur ayant quitté en cours de match, par ordre de
// signal le plus fiable au moins fiable :
//   - last_leave_time IS NOT NULL → signal direct API (le joueur a explicitement
//     quitté avant la fin du match)
//   - left_in_progress = TRUE
//   - present_at_beginning=TRUE && present_at_completion=FALSE
//   - outcome=4 (DNF) en dernier recours pour les très vieux matchs
//
// joined_in_progress (late-join) n'est PAS un quit — un late-joiner subit déjà
// la pénalité de moins de temps de jeu via ses counts plus bas. On ne le
// pénalise pas en plus.
func isQuitter(m rosterMember) bool {
	if m.lastLeaveTime.Valid {
		return true
	}
	if m.leftInProgress.Valid && m.leftInProgress.Bool {
		return true
	}
	if m.presentAtStart.Valid && m.presentAtEnd.Valid {
		return m.presentAtStart.Bool && !m.presentAtEnd.Bool
	}
	// Pas d'info de participation : on se replie sur outcome=4 (DNF). C'est
	// une approximation — un DNF peut aussi être une déco involontaire.
	return m.outcome == 4
}

// identifyPrimaryQuitter retourne le xuid du quitter PARTI EN PREMIER selon,
// par ordre de précision décroissant :
//  1. last_leave_time MIN (timestamp absolu API — signal idéal)
//  2. time_played_seconds MIN (proxy fallback : moins de temps = parti plus tôt)
//
// Les 2 niveaux ne sont JAMAIS mélangés dans un même match — si un seul
// quitter a last_leave_time renseigné, on utilise ce signal pour LUI et on
// considère les autres (sans timestamp) comme indéterminés (= traités
// secondaires). Cette stratégie évite qu'un quitter pré-backfill (sans
// timestamp) batte un quitter post-backfill juste parce que son
// time_played_seconds est plus petit.
//
// Retourne "" s'il n'y a aucun quitter OU si aucun n'a NI timestamp NI
// time_played (matchs ancien backfill participation booleans seul) — dans ce
// cas le caller traite tous les quitters comme primaires (cf. scaledQuitDelta).
func identifyPrimaryQuitter(teamA, teamB []rosterMember) string {
	type candidate struct {
		xuid    string
		leaveT  rosterMember
		timeSec float64
		hasTime bool
	}
	var quitters []candidate
	collect := func(team []rosterMember) {
		for _, m := range team {
			if !isQuitter(m) {
				continue
			}
			c := candidate{xuid: m.xuid, leaveT: m}
			if m.timePlayedSecs.Valid {
				c.timeSec = m.timePlayedSecs.Float64
				c.hasTime = true
			}
			quitters = append(quitters, c)
		}
	}
	collect(teamA)
	collect(teamB)
	if len(quitters) == 0 {
		return ""
	}

	// Préférence 1 : un ou plusieurs quitters ont last_leave_time → on les classe
	// entre eux par ce timestamp. Les autres quitters (sans timestamp) sont
	// hors du classement (= forcément secondaires).
	hasTimestamp := false
	for _, c := range quitters {
		if c.leaveT.lastLeaveTime.Valid {
			hasTimestamp = true
			break
		}
	}
	if hasTimestamp {
		var primary candidate
		first := true
		for _, c := range quitters {
			if !c.leaveT.lastLeaveTime.Valid {
				continue
			}
			if first || c.leaveT.lastLeaveTime.Time.Before(primary.leaveT.lastLeaveTime.Time) {
				primary = c
				first = false
			}
		}
		return primary.xuid
	}

	// Préférence 2 (fallback) : time_played_seconds MIN parmi ceux qui l'ont.
	var primary candidate
	first := true
	for _, c := range quitters {
		if !c.hasTime {
			continue
		}
		if first || c.timeSec < primary.timeSec {
			primary = c
			first = false
		}
	}
	if first {
		// Aucun n'a NI timestamp NI time_played : pas de primary identifiable.
		return ""
	}
	return primary.xuid
}

// scaledQuitDelta retourne le delta plein si xuid est le primary quitter
// (ou si primary est inconnu = "" → on traite tout comme primaire pour ne
// pas réduire silencieusement les anciens matchs sans time_played).
func scaledQuitDelta(xuid, primaryXUID string, baseDelta float64) float64 {
	if primaryXUID == "" || xuid == primaryXUID {
		return baseDelta
	}
	return baseDelta * QuitSecondaryFactor
}

// quitDeltaForTeam : pénalité quit selon outcome final de l'équipe.
//
//	TeamLoss → "related" : équipe perdait probablement déjà → δ modéré
//	TeamWin / TeamDraw → "unrelated" : équipe non perdante → δ fort
func quitDeltaForTeam(o skillv2.TeamResult) float64 {
	if o == skillv2.TeamLoss {
		return skillv2.DefaultQuitDeltaRelated
	}
	return skillv2.DefaultQuitDeltaUnrelated
}

// invertOutcome retourne l'outcome de l'équipe adverse.
func invertOutcome(o skillv2.TeamResult) skillv2.TeamResult {
	switch o {
	case skillv2.TeamWin:
		return skillv2.TeamLoss
	case skillv2.TeamLoss:
		return skillv2.TeamWin
	default:
		return skillv2.TeamDraw
	}
}

// buildCountInputs construit la structure CountInputs depuis les rosters.
// Inclut counts (Phase 3c) ET quit flags (Phase 3-quit TS2 §9).
//
// Si AUCUN joueur n'a kills/deaths NI quit, retourne nil (le code en aval
// traitera comme TS classique sans observations).
//
// outcomeA est l'outcome de la team A (depuis le owner) — utilisé pour
// distinguer related quit (team perdait → δ petit) d'unrelated quit (team
// gagnait/égalisait → δ grand). Sans timeline du score AU moment du quit,
// on approxime par le final outcome.
func buildCountInputs(teamA, teamB []rosterMember, outcomeA skillv2.TeamResult) *skillv2.CountInputs {
	hasAny := false
	for _, m := range teamA {
		if m.kills != nil || m.deaths != nil || isQuitter(m) {
			hasAny = true
			break
		}
	}
	if !hasAny {
		for _, m := range teamB {
			if m.kills != nil || m.deaths != nil || isQuitter(m) {
				hasAny = true
				break
			}
		}
	}
	if !hasAny {
		return nil
	}
	deltaA := quitDeltaForTeam(outcomeA)
	deltaB := quitDeltaForTeam(invertOutcome(outcomeA))
	primaryXUID := identifyPrimaryQuitter(teamA, teamB)
	pa := make([]skillv2.PlayerCounts, len(teamA))
	for i, m := range teamA {
		pa[i] = skillv2.PlayerCounts{Kills: m.kills, Deaths: m.deaths}
		if isQuitter(m) {
			pa[i].Quit = true
			pa[i].QuitPenaltyDelta = scaledQuitDelta(m.xuid, primaryXUID, deltaA)
		}
	}
	pb := make([]skillv2.PlayerCounts, len(teamB))
	for i, m := range teamB {
		pb[i] = skillv2.PlayerCounts{Kills: m.kills, Deaths: m.deaths}
		if isQuitter(m) {
			pb[i].Quit = true
			pb[i].QuitPenaltyDelta = scaledQuitDelta(m.xuid, primaryXUID, deltaB)
		}
	}
	return &skillv2.CountInputs{TeamA: pa, TeamB: pb}
}
