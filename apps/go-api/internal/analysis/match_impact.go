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

// kamikazeWindowMS est la fenêtre (ms) entre un kill et la mort de son auteur
// pour qu'une séquence compte comme "kamikaze". 1500 ms.
const kamikazeWindowMS int64 = 1500

// ComputeMatchImpactFull calcule les 10 badges d'impact pour un match.
//
// Convention de périmètre (parité Python compute_single_match_impact) : le
// caller doit passer dans input.Participants UNIQUEMENT les joueurs de
// l'équipe alliée du joueur principal. Les filtres internes (squadXUIDs,
// winXUIDs, lossXUIDs) en découlent — tous les badges hors first_blood sont
// donc team-wide alliée. first_blood reste GLOBAL (toutes équipes confondues)
// car il opère sur input.Events non-filtré.
//
// Badges produits (au plus 1 par badge-type) :
//   - first_blood           : premier tueur du MATCH (global, toutes équipes)
//   - first_group_death     : première mort dans l'équipe alliée
//   - clutch_finisher       : dernier kill d'un allié gagnant (∅ si défaite)
//   - last_casualty         : dernière mort d'un allié perdant (Boulet, ∅ si victoire)
//   - last_group_kill       : allié le plus lent à obtenir son premier kill (Touriste)
//   - top_killer            : allié avec le plus de kills (Bourreau)
//   - silent_hero           : max assists + min deaths en victoire hors top-killer
//   - false_brother         : max deaths + min assists en défaite hors top-killer
//   - top_gun               : premier allié à atteindre topGunKillThreshold kills
//   - kamikaze              : allié tué dans kamikazeWindowMS après un de ses frags
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
	squadXUIDs := make(map[string]bool, len(input.Participants))
	for _, p := range input.Participants {
		if p.XUID != "" {
			squadXUIDs[p.XUID] = true
		}
		if p.Outcome == 2 {
			winXUIDs[p.XUID] = true
		} else if p.Outcome == 3 {
			lossXUIDs[p.XUID] = true
		}
	}

	// --- 1. first_blood : premier kill global, retenu seulement si squad (parité Python identify_first_blood) ---
	if fb := firstByTime(kills); fb != nil {
		badges = append(badges, ImpactBadge{
			BadgeKey:   "first_blood",
			BadgeFR:    "Premier sang",
			PlayerXUID: fb.ActorXUID,
			TimeMS:     fb.TimeMS,
		})
	}

	// --- 2. first_group_death : première mort PARMI le squad (parité Python identify_first_group_death) ---
	deathsForSquad := deaths
	if len(squadXUIDs) > 0 {
		deathsForSquad = filterEventsByActor(deaths, squadXUIDs)
	}
	if fd := firstByTime(deathsForSquad); fd != nil {
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

	// --- 5. last_group_kill (Touriste) : plus lent à obtenir son premier kill PARMI le squad
	//        (parité Python identify_last_group_kill — filtre amis avant de chercher le slowest) ---
	killsForSquad := kills
	if len(squadXUIDs) > 0 {
		killsForSquad = filterEventsByActor(kills, squadXUIDs)
	}
	if xuid, t := slowestFirstKillerWithTime(killsForSquad); xuid != "" {
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

	// --- 9. top_gun : premier joueur de l'ÉQUIPE ALLIÉE à atteindre
	//        topGunKillThreshold kills (parité Python _find_top_gun_event qui
	//        reçoit kills déjà filtrés par team_xuids). Réutilise killsForSquad
	//        construit pour last_group_kill.
	if xuid, t := topGunWithTime(killsForSquad); xuid != "" {
		badges = append(badges, ImpactBadge{
			BadgeKey:   "top_gun",
			BadgeFR:    "Top Gun",
			PlayerXUID: xuid,
			TimeMS:     t,
		})
	}

	// --- 10. kamikaze : allié dont une death survient dans kamikazeWindowMS
	//         après un de ses propres frags. 1 badge max par match : joueur avec
	//         le plus de séquences (tie-break : T_death tardif, puis XUID asc).
	if xuid, t := kamikaze(kills, deaths, squadXUIDs); xuid != "" {
		badges = append(badges, ImpactBadge{
			BadgeKey:   "kamikaze",
			BadgeFR:    "Kamikaze",
			PlayerXUID: xuid,
			TimeMS:     t,
		})
	}

	return badges
}

// kamikaze attribue le badge à l'allié qui meurt le plus souvent dans la
// fenêtre kamikazeWindowMS après un de ses propres frags. Le caller doit avoir
// déjà séparé kills et deaths depuis input.Events.
//
// Critère de séquence : kill par X à T_k, suivi d'une death de X à T_d, avec
// T_k < T_d ≤ T_k + kamikazeWindowMS. Chaque kill peut être apparié au plus à
// une death (le death le plus proche). Inversement, une death ne peut compter
// que pour une seule séquence (la dernière qui la précède dans la fenêtre).
//
// Retourne "" si aucun candidat. Tie-break sur (count desc, T_d desc, XUID asc).
func kamikaze(kills, deaths []ImpactEvent, squadXUIDs map[string]bool) (string, int64) {
	if len(kills) == 0 || len(deaths) == 0 {
		return "", 0
	}
	// Grouper kills et deaths par acteur, triés par TimeMS asc.
	killsByXUID := make(map[string][]int64)
	for _, ev := range kills {
		if len(squadXUIDs) > 0 && !squadXUIDs[ev.ActorXUID] {
			continue
		}
		killsByXUID[ev.ActorXUID] = append(killsByXUID[ev.ActorXUID], ev.TimeMS)
	}
	deathsByXUID := make(map[string][]int64)
	for _, ev := range deaths {
		if len(squadXUIDs) > 0 && !squadXUIDs[ev.ActorXUID] {
			continue
		}
		deathsByXUID[ev.ActorXUID] = append(deathsByXUID[ev.ActorXUID], ev.TimeMS)
	}
	for _, ts := range killsByXUID {
		sortInt64Asc(ts)
	}
	for _, ts := range deathsByXUID {
		sortInt64Asc(ts)
	}

	type acc struct {
		count   int
		lastDie int64
	}
	per := make(map[string]*acc)
	for xuid, ks := range killsByXUID {
		ds := deathsByXUID[xuid]
		if len(ds) == 0 {
			continue
		}
		// Two-pointer : pour chaque kill, on cherche le 1er death > kill et ≤ kill+window.
		// Chaque death est consommé au plus une fois pour éviter qu'une mort unique
		// soit comptée par 2 kills successifs trop rapprochés.
		dIdx := 0
		for _, kT := range ks {
			for dIdx < len(ds) && ds[dIdx] <= kT {
				dIdx++
			}
			if dIdx >= len(ds) {
				break
			}
			dT := ds[dIdx]
			if dT-kT > kamikazeWindowMS {
				continue
			}
			a, ok := per[xuid]
			if !ok {
				a = &acc{}
				per[xuid] = a
			}
			a.count++
			if dT > a.lastDie {
				a.lastDie = dT
			}
			dIdx++
		}
	}
	if len(per) == 0 {
		return "", 0
	}
	var bestXUID string
	var bestCount int
	var bestT int64
	for xuid, a := range per {
		better := a.count > bestCount ||
			(a.count == bestCount && a.lastDie > bestT) ||
			(a.count == bestCount && a.lastDie == bestT && (bestXUID == "" || xuid < bestXUID))
		if better {
			bestXUID = xuid
			bestCount = a.count
			bestT = a.lastDie
		}
	}
	return bestXUID, bestT
}

// sortInt64Asc trie un slice d'int64 ascendant (insertion sort, suffisant pour
// les tailles attendues — quelques dizaines d'events par joueur).
func sortInt64Asc(a []int64) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
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

// filterEventsByActor retourne les événements dont ActorXUID est dans allowedXUIDs.
// Utilisé pour les badges qui doivent être calculés PARMI le squad uniquement
// (first_group_death, last_group_kill — parité Python identify_*_group_*).
func filterEventsByActor(events []ImpactEvent, allowedXUIDs map[string]bool) []ImpactEvent {
	out := make([]ImpactEvent, 0, len(events))
	for _, ev := range events {
		if allowedXUIDs[ev.ActorXUID] {
			out = append(out, ev)
		}
	}
	return out
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
// Nécessite ≥2 joueurs éligibles et ≥1 assist. Parité Python
// (identify_silent_hero_multi) : exclut TOUS les joueurs avec kills == max_kills,
// puis n'attribue le badge QUE SI le même joueur cumule à la fois max_assists ET
// min_deaths parmi tous les éligibles. Si le porteur du max_assists n'a pas
// aussi le min_deaths absolu, aucun badge n'est attribué.
func silentHero(participants []ParticipantSnap) string {
	wins := filterOutcome(participants, 2)
	if len(wins) < 2 {
		return ""
	}
	eligible := excludeAllTopKillers(wins)
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
	minDeaths := eligible[0].Deaths
	for _, p := range eligible {
		if p.Deaths < minDeaths {
			minDeaths = p.Deaths
		}
	}
	for _, p := range eligible {
		if p.Assists == maxAssists && p.Deaths == minDeaths {
			return p.XUID
		}
	}
	return ""
}

// falseBrother : en défaite, joueur avec max deaths ET min assists hors top-killer.
// Nécessite ≥2 joueurs éligibles et ≥1 mort. Parité Python
// (identify_false_brother_multi) : exclut TOUS les joueurs avec kills ==
// max_kills, puis n'attribue le badge QUE SI le même joueur cumule à la fois
// max_deaths ET min_assists parmi tous les éligibles.
func falseBrother(participants []ParticipantSnap) string {
	losses := filterOutcome(participants, 3)
	if len(losses) < 2 {
		return ""
	}
	eligible := excludeAllTopKillers(losses)
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
	minAssists := eligible[0].Assists
	for _, p := range eligible {
		if p.Assists < minAssists {
			minAssists = p.Assists
		}
	}
	for _, p := range eligible {
		if p.Deaths == maxDeaths && p.Assists == minAssists {
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

// excludeAllTopKillers retire TOUS les participants ayant le max de kills > 0
// (parité Python : `eligible = [r for r in rows if r.kills < max_kills]`).
// Si max_kills == 0, tous sont éligibles.
func excludeAllTopKillers(ps []ParticipantSnap) []ParticipantSnap {
	maxK := 0
	for _, p := range ps {
		if p.Kills > maxK {
			maxK = p.Kills
		}
	}
	if maxK == 0 {
		// Aucun kill — pas de top killer à exclure
		out := make([]ParticipantSnap, len(ps))
		copy(out, ps)
		return out
	}
	out := make([]ParticipantSnap, 0, len(ps))
	for _, p := range ps {
		if p.Kills < maxK {
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
