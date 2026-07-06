package skill

// skill_v2_quit_penalty.go — détection quitter + ordering primary/secondary
// + construction des CountInputs avec quit flags.
//
// Extrait de skill_v2_shadow.go (2026-05-27) pour respecter le seuil 500L.
// Concern unique : transformer un roster en signal v2 "quit penalty" avec
// l'ordering produit "1er quitter = delta plein, suivants = 50%".

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
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
	return m.outcome == domain.OutcomeDNF
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

// quitDeltaForTeam : pénalité quit selon outcome final de l'équipe (fallback
// Sprint 2.A quand la timeline des frags est indisponible).
//
//	TeamLoss → "related" : équipe perdait probablement déjà → δ modéré
//	TeamWin / TeamDraw → "unrelated" : équipe non perdante → δ fort
func quitDeltaForTeam(o skillv2.TeamResult) float64 {
	if o == skillv2.TeamLoss {
		return skillv2.DefaultQuitDeltaRelated
	}
	return skillv2.DefaultQuitDeltaUnrelated
}

// quitDeltaForContext : pénalité quit selon la situation AU MOMENT du quit
// (Sprint 2.A). Seul "perdait" est related (modéré) ; menait/égalité = abandon
// d'une situation non-perdante → fort.
func quitDeltaForContext(ctx skillv2.QuitContext) float64 {
	if ctx == skillv2.QuitWhileTrailing {
		return skillv2.DefaultQuitDeltaRelated
	}
	return skillv2.DefaultQuitDeltaUnrelated
}

// quitTimeline porte la timeline des frags d'un match (attribués par side :
// teamA=0, teamB=1) et le repère temporel servant à situer le moment du quit.
// available=false → buildCountInputs retombe sur l'outcome final.
type quitTimeline struct {
	frags        []skillv2.TeamFrag
	filmStartUTC time.Time
	available    bool
}

// quitOffsetMs convertit le timestamp ABSOLU de départ d'un joueur en
// millisecondes dans le repère des frags (killer_victim_pairs.time_ms).
//
// ⚠️ HOOK ADAPTER MULTI-TITRE — T0 du match ⚠️
// Le vrai T0 = match_registry.real_start_time (début réel après countdown,
// disponible dans ~99% des cas) et doit être obtenu via un ADAPTER
// titre-spécifique. CE N'EST PAS FAIT ICI : on utilise filmStartUTC (début du
// film = start_time_utc) comme repère, ce qui est correct pour Halo Infinite car
// killer_victim_pairs.time_ms est relatif au début du film. POUR BRANCHER LE T0
// RÉEL (ou pour un autre titre), c'est ICI que l'adapter doit remplacer
// filmStartUTC par le real_start_time résolu. Ne pas exploiter durée/début de
// match ailleurs sans passer par ce point. Cf. roadmap Sprint 2.A.
func quitOffsetMs(leaveTime, filmStartUTC time.Time) int64 {
	return leaveTime.Sub(filmStartUTC).Milliseconds()
}

// hasAnyQuitter retourne true si au moins un joueur a quitté (évite de charger
// la timeline des frags pour rien sur les matchs sans quitter).
func hasAnyQuitter(teamA, teamB []rosterMember) bool {
	for _, m := range teamA {
		if isQuitter(m) {
			return true
		}
	}
	for _, m := range teamB {
		if isQuitter(m) {
			return true
		}
	}
	return false
}

// loadQuitTimeline charge la timeline des frags (killer_victim_pairs) du match et
// attribue chaque frag à un side (teamA=0, teamB=1). available=false si erreur
// ou aucun frag exploitable → fallback outcome final.
func loadQuitTimeline(ctx context.Context, sharedDB *sql.DB, matchID string,
	filmStartUTC time.Time, teamA, teamB []rosterMember) quitTimeline {
	side := make(map[string]int, len(teamA)+len(teamB))
	for _, m := range teamA {
		side[m.xuid] = 0
	}
	for _, m := range teamB {
		side[m.xuid] = 1
	}
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT killer_xuid, time_ms FROM killer_victim_pairs
		WHERE match_id = ? AND time_ms IS NOT NULL AND killer_xuid IS NOT NULL
		ORDER BY time_ms ASC`, matchID)
	if err != nil {
		slog.WarnContext(ctx, "LUSR v2 quit-context: chargement frags échoué — fallback outcome",
			"match_id", matchID, "err", err)
		return quitTimeline{available: false}
	}
	defer rows.Close() //nolint:errcheck

	var frags []skillv2.TeamFrag
	for rows.Next() {
		var killer string
		var t int64
		if err := rows.Scan(&killer, &t); err != nil {
			continue
		}
		s, ok := side[killer]
		if !ok {
			continue // frag par un xuid hors des 2 équipes (rare)
		}
		frags = append(frags, skillv2.TeamFrag{TimeMs: t, TeamID: s})
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "LUSR v2 quit-context: itération frags échouée — fallback outcome",
			"match_id", matchID, "err", err)
		return quitTimeline{available: false}
	}
	if len(frags) == 0 {
		slog.DebugContext(ctx, "LUSR v2 quit-context: aucun frag — fallback outcome",
			"match_id", matchID)
		return quitTimeline{available: false}
	}
	slog.DebugContext(ctx, "LUSR v2 quit-context: timeline frags chargée",
		"match_id", matchID, "frags", len(frags))
	return quitTimeline{frags: frags, filmStartUTC: filmStartUTC, available: true}
}

// quitBaseDelta retourne le delta de base d'un quitter : depuis le CONTEXTE
// (score au moment du quit, Sprint 2.A) si la timeline est dispo et que le joueur
// a un timestamp de départ ; sinon depuis l'outcome FINAL de son équipe.
func quitBaseDelta(ctx context.Context, m rosterMember, teamOutcome skillv2.TeamResult, side int, qt quitTimeline) float64 {
	if qt.available && m.lastLeaveTime.Valid {
		quitMs := quitOffsetMs(m.lastLeaveTime.Time, qt.filmStartUTC)
		qc := skillv2.InferQuitContext(qt.frags, quitMs, side)
		delta := quitDeltaForContext(qc)
		slog.DebugContext(ctx, "LUSR v2 quit-context appliqué",
			"xuid", m.xuid, "context", qc.String(), "quit_ms", quitMs, "delta", delta)
		return delta
	}
	return quitDeltaForTeam(teamOutcome)
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
// outcomeA est l'outcome de la team A (depuis le owner). Sprint 2.A : si `qt`
// est disponible, la magnitude du quit penalty dépend de la situation AU MOMENT
// du quit (équipe perdait → modéré ; menait/égalité → fort) ; sinon fallback sur
// l'outcome final via quitDeltaForTeam. gameplayDurMs alimente le poids TS2
// wᵢ = time_played / match_length (sum-factor team_perf) via playerTeamWeight.
func buildCountInputs(ctx context.Context, teamA, teamB []rosterMember, outcomeA skillv2.TeamResult, qt quitTimeline, gameplayDurMs int64) *skillv2.CountInputs {
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
	outcomeB := invertOutcome(outcomeA)
	primaryXUID := identifyPrimaryQuitter(teamA, teamB)
	pa := make([]skillv2.PlayerCounts, len(teamA))
	for i, m := range teamA {
		pa[i] = skillv2.PlayerCounts{Kills: m.kills, Deaths: m.deaths, Weight: playerTeamWeight(m, gameplayDurMs)}
		if isQuitter(m) {
			pa[i].Quit = true
			base := quitBaseDelta(ctx, m, outcomeA, 0, qt) // teamA = side 0
			pa[i].QuitPenaltyDelta = scaledQuitDelta(m.xuid, primaryXUID, base)
		}
	}
	pb := make([]skillv2.PlayerCounts, len(teamB))
	for i, m := range teamB {
		pb[i] = skillv2.PlayerCounts{Kills: m.kills, Deaths: m.deaths, Weight: playerTeamWeight(m, gameplayDurMs)}
		if isQuitter(m) {
			pb[i].Quit = true
			base := quitBaseDelta(ctx, m, outcomeB, 1, qt) // teamB = side 1
			pb[i].QuitPenaltyDelta = scaledQuitDelta(m.xuid, primaryXUID, base)
		}
	}
	return &skillv2.CountInputs{TeamA: pa, TeamB: pb}
}

// playerTeamWeight calcule le poids TS2 wᵢ = time_played_i / match_length pour
// le sum-factor team_perf (cf. ep/sum_factor.go). Retourne 0 (→ wᵢ=1 côté EP,
// participation pleine) si la durée gameplay ou le time_played est inconnu. Le
// clamp [0,1] + plancher est appliqué côté EP (resolveTeamWeight).
func playerTeamWeight(m rosterMember, gameplayDurMs int64) float64 {
	if gameplayDurMs <= 0 || !m.timePlayedSecs.Valid || m.timePlayedSecs.Float64 <= 0 {
		return 0
	}
	w := (m.timePlayedSecs.Float64 * 1000) / float64(gameplayDurMs)
	if w > 1 {
		w = 1
	}
	return w
}
