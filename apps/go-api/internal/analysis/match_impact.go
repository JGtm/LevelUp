// Package analysis — match_impact.go : détection des badges d'impact sur un match.
//
// Port de src/analysis/_impact_event_badges.py + src/analysis/friends_impact.py.
// 8 badges calculés à partir des événements highlight_events et des stats scoreboard.
package analysis

import "math"

// ImpactEvent représente un événement horodaté d'un match.
// Pour un event de type "kill", ActorXUID est le tueur.
// Pour un event de type "death", ActorXUID est la victime.
type ImpactEvent struct {
	TimeMS    int64
	EventType string // "kill" | "death"
	ActorXUID string
}

// ParticipantSnap est le snapshot minimal par joueur pour les badges stat-based.
type ParticipantSnap struct {
	XUID    string
	Outcome int // 2=WIN, 3=LOSS, 1=TIE, 4=DNF
	Kills   int
	Deaths  int
	Assists int
}

// ImpactBadge représente un badge attribué à un joueur sur ce match.
//
// TimeMS est l'instant (ms depuis le début du match) auquel le badge a été
// déclenché pour les badges event-based (first_blood, first_group_death,
// clutch_finisher, last_casualty, last_group_kill, top_gun). Vaut 0 pour les
// badges stat-based qui n'ont pas de timestamp (top_killer, silent_hero,
// false_brother).
type ImpactBadge struct {
	BadgeKey   string // identifiant technique
	BadgeFR    string // libellé français
	PlayerXUID string
	TimeMS     int64
}

// MatchImpactInput regroupe toutes les données nécessaires au calcul des badges.
type MatchImpactInput struct {
	// Events : événements highlight_events triés ou non (kill/death + horodatage + acteur).
	Events []ImpactEvent
	// Participants : tous les joueurs du match avec leurs stats brutes.
	Participants []ParticipantSnap
}

// topGunKillThreshold est le nombre de kills pour décrocher le badge top_gun.
const topGunKillThreshold = 10

// ComputeMatchImpactFull calcule les 9 badges d'impact pour l'ensemble des joueurs du match.
//
// Badges produits (au plus 1 par badge-type) :
//   - first_blood           : premier tueur du match
//   - first_group_death     : premier joueur à mourir
//   - clutch_finisher       : dernier tueur parmi les gagnants
//   - last_casualty         : dernier joueur à mourir parmi les perdants (Boulet)
//   - last_group_kill       : gagnant le plus lent à obtenir son premier kill (Touriste)
//   - top_killer            : joueur avec le plus de kills (Bourreau)
//   - silent_hero           : max assists + min deaths en victoire hors top-killer (Héros silencieux)
//   - false_brother         : max deaths + min assists en défaite hors top-killer (Faux-frère)
//   - top_gun               : premier joueur à atteindre topGunKillThreshold kills en ordre chrono
func ComputeMatchImpactFull(input MatchImpactInput) []ImpactBadge {
	var badges []ImpactBadge

	// --- Pré-calcul : séparer kills et deaths, indexer par outcome ---
	var kills, deaths []ImpactEvent
	for _, ev := range input.Events {
		switch ev.EventType {
		case "kill":
			kills = append(kills, ev)
		case "death":
			deaths = append(deaths, ev)
		}
	}

	winXUIDs := make(map[string]bool)
	lossXUIDs := make(map[string]bool)
	for _, p := range input.Participants {
		if p.Outcome == 2 {
			winXUIDs[p.XUID] = true
		} else if p.Outcome == 3 {
			lossXUIDs[p.XUID] = true
		}
	}

	// --- 1. first_blood ---
	if fb := firstByTime(kills); fb != nil {
		badges = append(badges, ImpactBadge{
			BadgeKey:   "first_blood",
			BadgeFR:    "Premier sang",
			PlayerXUID: fb.ActorXUID,
			TimeMS:     fb.TimeMS,
		})
	}

	// --- 2. first_group_death ---
	if fd := firstByTime(deaths); fd != nil {
		badges = append(badges, ImpactBadge{
			BadgeKey:   "first_group_death",
			BadgeFR:    "Première victime",
			PlayerXUID: fd.ActorXUID,
			TimeMS:     fd.TimeMS,
		})
	}

	// --- 3. clutch_finisher : dernier kill d'un gagnant ---
	if cf := lastByTimeFiltered(kills, winXUIDs); cf != nil {
		badges = append(badges, ImpactBadge{
			BadgeKey:   "clutch_finisher",
			BadgeFR:    "Finisseur",
			PlayerXUID: cf.ActorXUID,
			TimeMS:     cf.TimeMS,
		})
	}

	// --- 4. last_casualty (Boulet) : dernière mort d'un perdant ---
	if lc := lastByTimeFiltered(deaths, lossXUIDs); lc != nil {
		badges = append(badges, ImpactBadge{
			BadgeKey:   "last_casualty",
			BadgeFR:    "Boulet",
			PlayerXUID: lc.ActorXUID,
			TimeMS:     lc.TimeMS,
		})
	}

	// --- 5. last_group_kill (Touriste) : plus lent à obtenir son premier kill ---
	if xuid, t := slowestFirstKillerWithTime(kills); xuid != "" {
		badges = append(badges, ImpactBadge{
			BadgeKey:   "last_group_kill",
			BadgeFR:    "Touriste",
			PlayerXUID: xuid,
			TimeMS:     t,
		})
	}

	// --- 6. top_killer (Bourreau) ---
	if xuid := topKiller(input.Participants); xuid != "" {
		badges = append(badges, ImpactBadge{
			BadgeKey:   "top_killer",
			BadgeFR:    "Bourreau",
			PlayerXUID: xuid,
		})
	}

	// --- 7. silent_hero (Héros silencieux) : victoire, max assists + min deaths hors top-killer ---
	if xuid := silentHero(input.Participants); xuid != "" {
		badges = append(badges, ImpactBadge{
			BadgeKey:   "silent_hero",
			BadgeFR:    "Héros silencieux",
			PlayerXUID: xuid,
		})
	}

	// --- 8. false_brother (Faux-frère) : défaite, max deaths + min assists hors top-killer ---
	if xuid := falseBrother(input.Participants); xuid != "" {
		badges = append(badges, ImpactBadge{
			BadgeKey:   "false_brother",
			BadgeFR:    "Faux-frère",
			PlayerXUID: xuid,
		})
	}

	// --- 9. top_gun : premier joueur à atteindre topGunKillThreshold kills ---
	if xuid, t := topGunWithTime(kills); xuid != "" {
		badges = append(badges, ImpactBadge{
			BadgeKey:   "top_gun",
			BadgeFR:    "Top Gun",
			PlayerXUID: xuid,
			TimeMS:     t,
		})
	}

	return badges
}

// ---------------------------------------------------------------------------
// Helpers internes
// ---------------------------------------------------------------------------

func firstByTime(events []ImpactEvent) *ImpactEvent {
	if len(events) == 0 {
		return nil
	}
	best := &events[0]
	for i := 1; i < len(events); i++ {
		if events[i].TimeMS < best.TimeMS {
			best = &events[i]
		}
	}
	return best
}

// lastByTimeFiltered retourne l'événement avec le TimeMS maximum parmi les événements
// dont ActorXUID est dans le set allowedXUIDs.
func lastByTimeFiltered(events []ImpactEvent, allowedXUIDs map[string]bool) *ImpactEvent {
	var best *ImpactEvent
	for i := range events {
		ev := &events[i]
		if !allowedXUIDs[ev.ActorXUID] {
			continue
		}
		if best == nil || ev.TimeMS > best.TimeMS {
			best = ev
		}
	}
	return best
}

// slowestFirstKillerWithTime retourne le XUID + timestamp du premier kill
// le plus tardif parmi tous les joueurs ayant fait au moins 1 kill.
// Retourne "" si moins de 2 tueurs distincts (badge n'a pas de sens).
func slowestFirstKillerWithTime(kills []ImpactEvent) (string, int64) {
	if len(kills) == 0 {
		return "", 0
	}
	// firstKillTime[xuid] = min TimeMS de ce tueur
	firstKillTime := make(map[string]int64)
	for _, ev := range kills {
		t, seen := firstKillTime[ev.ActorXUID]
		if !seen || ev.TimeMS < t {
			firstKillTime[ev.ActorXUID] = ev.TimeMS
		}
	}
	if len(firstKillTime) < 2 {
		return "", 0
	}
	var slowestXUID string
	var slowestTime int64 = math.MinInt64
	for xuid, t := range firstKillTime {
		if t > slowestTime {
			slowestTime = t
			slowestXUID = xuid
		}
	}
	return slowestXUID, slowestTime
}

// topKiller retourne le XUID du joueur avec le plus de kills.
// Nécessite ≥2 joueurs et ≥1 kill pour que le badge soit attribué.
func topKiller(participants []ParticipantSnap) string {
	if len(participants) < 2 {
		return ""
	}
	maxKills := 0
	var best string
	for _, p := range participants {
		if p.Kills > maxKills {
			maxKills = p.Kills
			best = p.XUID
		}
	}
	if maxKills == 0 {
		return ""
	}
	return best
}

// silentHero : en victoire, joueur avec max assists ET min deaths hors top-killer.
// Nécessite ≥2 joueurs éligibles et ≥1 assist.
func silentHero(participants []ParticipantSnap) string {
	wins := filterOutcome(participants, 2)
	if len(wins) < 2 {
		return ""
	}
	tkXUID := topKillerXUID(wins)
	eligible := excludeXUID(wins, tkXUID)
	if len(eligible) < 2 {
		return ""
	}
	maxAssists := 0
	for _, p := range eligible {
		if p.Assists > maxAssists {
			maxAssists = p.Assists
		}
	}
	if maxAssists == 0 {
		return ""
	}
	withMaxAssists := filterAssists(eligible, maxAssists)
	minDeaths := withMaxAssists[0].Deaths
	for _, p := range withMaxAssists {
		if p.Deaths < minDeaths {
			minDeaths = p.Deaths
		}
	}
	for _, p := range withMaxAssists {
		if p.Deaths == minDeaths {
			return p.XUID
		}
	}
	return ""
}

// falseBrother : en défaite, joueur avec max deaths ET min assists hors top-killer.
// Nécessite ≥2 joueurs éligibles et ≥1 mort.
func falseBrother(participants []ParticipantSnap) string {
	losses := filterOutcome(participants, 3)
	if len(losses) < 2 {
		return ""
	}
	tkXUID := topKillerXUID(losses)
	eligible := excludeXUID(losses, tkXUID)
	if len(eligible) < 2 {
		return ""
	}
	maxDeaths := 0
	for _, p := range eligible {
		if p.Deaths > maxDeaths {
			maxDeaths = p.Deaths
		}
	}
	if maxDeaths == 0 {
		return ""
	}
	withMaxDeaths := filterDeaths(eligible, maxDeaths)
	minAssists := withMaxDeaths[0].Assists
	for _, p := range withMaxDeaths {
		if p.Assists < minAssists {
			minAssists = p.Assists
		}
	}
	for _, p := range withMaxDeaths {
		if p.Assists == minAssists {
			return p.XUID
		}
	}
	return ""
}

func filterOutcome(ps []ParticipantSnap, outcome int) []ParticipantSnap {
	out := make([]ParticipantSnap, 0, len(ps))
	for _, p := range ps {
		if p.Outcome == outcome {
			out = append(out, p)
		}
	}
	return out
}

func topKillerXUID(ps []ParticipantSnap) string {
	if len(ps) == 0 {
		return ""
	}
	best := ps[0]
	for _, p := range ps[1:] {
		if p.Kills > best.Kills {
			best = p
		}
	}
	if best.Kills == 0 {
		return ""
	}
	return best.XUID
}

func excludeXUID(ps []ParticipantSnap, xuid string) []ParticipantSnap {
	if xuid == "" {
		return ps
	}
	out := make([]ParticipantSnap, 0, len(ps))
	for _, p := range ps {
		if p.XUID != xuid {
			out = append(out, p)
		}
	}
	return out
}

func filterAssists(ps []ParticipantSnap, minVal int) []ParticipantSnap {
	out := make([]ParticipantSnap, 0)
	for _, p := range ps {
		if p.Assists == minVal {
			out = append(out, p)
		}
	}
	return out
}

func filterDeaths(ps []ParticipantSnap, minVal int) []ParticipantSnap {
	out := make([]ParticipantSnap, 0)
	for _, p := range ps {
		if p.Deaths == minVal {
			out = append(out, p)
		}
	}
	return out
}

// topGunWithTime retourne le XUID + le TimeMS auquel le premier joueur atteint
// topGunKillThreshold kills, en parcourant les kills dans l'ordre chronologique.
// Retourne ("", 0) si aucun joueur n'atteint le seuil.
func topGunWithTime(kills []ImpactEvent) (string, int64) {
	sorted := sortedByTime(kills)
	killCount := make(map[string]int)
	for _, ev := range sorted {
		killCount[ev.ActorXUID]++
		if killCount[ev.ActorXUID] >= topGunKillThreshold {
			return ev.ActorXUID, ev.TimeMS
		}
	}
	return "", 0
}

// sortedByTime retourne une copie des événements triée par TimeMS ASC.
func sortedByTime(events []ImpactEvent) []ImpactEvent {
	out := make([]ImpactEvent, len(events))
	copy(out, events)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].TimeMS < out[j-1].TimeMS; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
