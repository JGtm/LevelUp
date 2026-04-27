package narrative

import (
	"sort"

	"levelup/go-api/internal/games/canonical"
)

// ImpactRole identifie un des 8 roles d'impact attribuables a un joueur sur
// un match donne (heatmap Squad + impact timeline MatchView).
type ImpactRole string

// Constantes des 8 roles. La cle stable est utilisee pour les LabelKey
// (`narrative.role.<key>`) et ColorToken (`narrative.role.<key>`).
const (
	RoleFirstBlood      ImpactRole = "first_blood"       // premier kill du match
	RoleClutchFinisher  ImpactRole = "clutch_finisher"   // dernier kill, dans la fenetre clutch, equipe gagnante
	RoleLastCasualty    ImpactRole = "last_casualty"     // dernier mort du match
	RoleLastGroupKill   ImpactRole = "last_group_kill"   // dernier kill par un membre du squad
	RoleFirstGroupDeath ImpactRole = "first_group_death" // premier mort du squad
	RoleSilentHero      ImpactRole = "silent_hero"       // squad gagnant, kills < moyenne squad
	RoleFalseBrother    ImpactRole = "false_brother"     // squad perdant, K/D le pire (deaths > kills)
	RoleTopKiller       ImpactRole = "top_killer"        // squad, plus de kills sur le match
)

// ClutchWindowMS est la fenetre temporelle (en ms) avant la fin du match
// dans laquelle un kill compte comme "clutch". 30 secondes.
const ClutchWindowMS int64 = 30_000

// RoleAssignment est l'attribution d'un role a un joueur sur un match donne.
//
//	Inverted = true pour les roles "negatifs" (LastCasualty, FirstGroupDeath,
//	FalseBrother, et LastGroupKill si l'equipe a perdu) : le rendu doit
//	utiliser une nuance de couleur opposee pour le distinguer.
type RoleAssignment struct {
	XUID       string
	Role       ImpactRole
	MatchID    string
	LabelKey   string
	ColorToken string
	Inverted   bool
}

// IdentifyImpactRoles attribue les 8 roles d'impact pour les matchs presents
// dans events. teamOutcomes mappe xuid -> outcome (win / loss / tie / dnf).
// squad est la liste des xuid candidates aux roles de squad (LastGroupKill,
// FirstGroupDeath, SilentHero, FalseBrother, TopKiller). FirstBlood,
// LastCasualty et ClutchFinisher s'attribuent independamment du squad.
//
// Si events couvre plusieurs matchs (champ MatchID different), ils sont
// traites independamment et les RoleAssignment retournes incluent le
// MatchID. Le slice est trie par (MatchID asc, Role asc).
//
// Heuristiques implementees pour cette version (a affiner en Phase 1
// pilote sur Squad / MatchView) :
//
//	SilentHero : equipe gagnante, kills > 0 et kills < moyenne squad gagnant.
//	FalseBrother : equipe perdante, ratio deaths/kills le pire et > 1.0.
//
// Ces deux roles requierent au moins 2 membres du squad sur la meme equipe
// (sinon "moins que la moyenne" / "le pire" n'a pas de sens). Si moins,
// ils ne sont pas attribues.
func IdentifyImpactRoles(
	events []canonical.HighlightEvent,
	teamOutcomes map[string]canonical.Outcome,
	squad []string,
) []RoleAssignment {
	if len(events) == 0 {
		return nil
	}
	squadSet := buildSquadSet(squad)
	byMatch := groupEventsByMatch(events)

	var all []RoleAssignment
	for matchID, matchEvents := range byMatch {
		all = append(all, identifyForMatch(matchID, matchEvents, teamOutcomes, squadSet)...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].MatchID != all[j].MatchID {
			return all[i].MatchID < all[j].MatchID
		}
		return all[i].Role < all[j].Role
	})
	return all
}

func buildSquadSet(squad []string) map[string]bool {
	out := make(map[string]bool, len(squad))
	for _, x := range squad {
		out[x] = true
	}
	return out
}

func groupEventsByMatch(events []canonical.HighlightEvent) map[string][]canonical.HighlightEvent {
	byMatch := make(map[string][]canonical.HighlightEvent)
	for _, ev := range events {
		if ev.MatchID == "" {
			continue
		}
		byMatch[ev.MatchID] = append(byMatch[ev.MatchID], ev)
	}
	return byMatch
}

// playerStat agrege les stats d'un joueur sur un match (kills, deaths,
// premier / dernier kill).
type playerStat struct {
	kills, deaths int
	firstKillMS   *int64
	lastKillMS    *int64
}

// matchSnapshot porte tous les agregats utiles pour identifier les roles d'un
// match.
type matchSnapshot struct {
	stats             map[string]*playerStat
	firstKill         *killEvent // premier kill du match
	lastKill          *killEvent // dernier kill du match
	firstDeathInSquad *deathEvent
	lastKillBySquad   *killEvent
}

type killEvent struct {
	killerXUID string
	victimXUID string
	timeMS     int64
}

type deathEvent struct {
	victimXUID string
	timeMS     int64
}

// collectMatchStats parcourt les events d'un match et construit un snapshot
// agrege pour les attributions de roles.
func collectMatchStats(events []canonical.HighlightEvent, squadSet map[string]bool) *matchSnapshot {
	snap := &matchSnapshot{stats: map[string]*playerStat{}}
	for _, ev := range events {
		switch ev.EventType {
		case string(canonical.EventKill), string(canonical.EventFirstKill):
			handleKillEvent(snap, ev, squadSet)
		case string(canonical.EventDeath), string(canonical.EventFirstDeath):
			handleStandaloneDeath(snap, ev, squadSet)
		}
	}
	return snap
}

func handleKillEvent(snap *matchSnapshot, ev canonical.HighlightEvent, squadSet map[string]bool) {
	if ev.KillerXUID == nil {
		return
	}
	k := *ev.KillerXUID
	ks := ensureStat(snap.stats, k)
	ks.kills++
	t := ev.TimeMS
	if ks.firstKillMS == nil || t < *ks.firstKillMS {
		ks.firstKillMS = &t
	}
	if ks.lastKillMS == nil || t > *ks.lastKillMS {
		ks.lastKillMS = &t
	}
	victim := ""
	if ev.VictimXUID != nil {
		victim = *ev.VictimXUID
		vs := ensureStat(snap.stats, victim)
		vs.deaths++
		registerDeath(snap, victim, t, squadSet)
	}
	registerKill(snap, k, victim, t, squadSet)
}

func handleStandaloneDeath(snap *matchSnapshot, ev canonical.HighlightEvent, squadSet map[string]bool) {
	if ev.VictimXUID == nil {
		return
	}
	v := *ev.VictimXUID
	vs := ensureStat(snap.stats, v)
	vs.deaths++
	registerDeath(snap, v, ev.TimeMS, squadSet)
}

func ensureStat(m map[string]*playerStat, xuid string) *playerStat {
	if s := m[xuid]; s != nil {
		return s
	}
	s := &playerStat{}
	m[xuid] = s
	return s
}

func registerKill(snap *matchSnapshot, killer, victim string, t int64, squadSet map[string]bool) {
	if snap.firstKill == nil || t < snap.firstKill.timeMS {
		snap.firstKill = &killEvent{killerXUID: killer, victimXUID: victim, timeMS: t}
	}
	if snap.lastKill == nil || t > snap.lastKill.timeMS {
		snap.lastKill = &killEvent{killerXUID: killer, victimXUID: victim, timeMS: t}
	}
	if squadSet[killer] {
		if snap.lastKillBySquad == nil || t > snap.lastKillBySquad.timeMS {
			snap.lastKillBySquad = &killEvent{killerXUID: killer, victimXUID: victim, timeMS: t}
		}
	}
}

func registerDeath(snap *matchSnapshot, victim string, t int64, squadSet map[string]bool) {
	if !squadSet[victim] {
		return
	}
	if snap.firstDeathInSquad == nil || t < snap.firstDeathInSquad.timeMS {
		snap.firstDeathInSquad = &deathEvent{victimXUID: victim, timeMS: t}
	}
}

// identifyForMatch oriente l'identification des 8 roles pour un match donne.
func identifyForMatch(
	matchID string,
	events []canonical.HighlightEvent,
	teamOutcomes map[string]canonical.Outcome,
	squadSet map[string]bool,
) []RoleAssignment {
	snap := collectMatchStats(events, squadSet)

	var out []RoleAssignment
	add := func(xuid string, role ImpactRole, inverted bool) {
		if xuid == "" {
			return
		}
		out = append(out, RoleAssignment{
			XUID:       xuid,
			Role:       role,
			MatchID:    matchID,
			LabelKey:   "narrative.role." + string(role),
			ColorToken: "narrative.role." + string(role),
			Inverted:   inverted,
		})
	}

	if snap.firstKill != nil {
		add(snap.firstKill.killerXUID, RoleFirstBlood, false)
	}
	if snap.lastKill != nil && snap.lastKill.victimXUID != "" {
		add(snap.lastKill.victimXUID, RoleLastCasualty, true)
	}
	if snap.firstDeathInSquad != nil {
		add(snap.firstDeathInSquad.victimXUID, RoleFirstGroupDeath, true)
	}
	if snap.lastKillBySquad != nil {
		killer := snap.lastKillBySquad.killerXUID
		inverted := teamOutcomes[killer] != canonical.OutcomeWin
		add(killer, RoleLastGroupKill, inverted)
	}
	if snap.lastKill != nil {
		if x := findClutchFinisher(snap, teamOutcomes); x != "" {
			add(x, RoleClutchFinisher, false)
		}
	}
	if x := findTopKiller(snap, squadSet); x != "" {
		add(x, RoleTopKiller, false)
	}
	if x := findSilentHero(snap, teamOutcomes, squadSet); x != "" {
		add(x, RoleSilentHero, false)
	}
	if x := findFalseBrother(snap, teamOutcomes, squadSet); x != "" {
		add(x, RoleFalseBrother, true)
	}
	return out
}

func findClutchFinisher(snap *matchSnapshot, teamOutcomes map[string]canonical.Outcome) string {
	if snap.lastKill == nil {
		return ""
	}
	clutchStart := snap.lastKill.timeMS - ClutchWindowMS
	for xuid, s := range snap.stats {
		if s.lastKillMS == nil {
			continue
		}
		if *s.lastKillMS < clutchStart || *s.lastKillMS > snap.lastKill.timeMS {
			continue
		}
		if teamOutcomes[xuid] == canonical.OutcomeWin {
			return xuid
		}
	}
	return ""
}

func findTopKiller(snap *matchSnapshot, squadSet map[string]bool) string {
	var topXUID string
	var topKills int
	for xuid, s := range snap.stats {
		if !squadSet[xuid] {
			continue
		}
		if s.kills > topKills {
			topKills = s.kills
			topXUID = xuid
		}
	}
	if topKills < 1 {
		return ""
	}
	return topXUID
}

func findSilentHero(snap *matchSnapshot, teamOutcomes map[string]canonical.Outcome, squadSet map[string]bool) string {
	var winSquadXUIDs []string
	var winSquadKills []int
	for xuid, s := range snap.stats {
		if !squadSet[xuid] {
			continue
		}
		if teamOutcomes[xuid] != canonical.OutcomeWin {
			continue
		}
		winSquadXUIDs = append(winSquadXUIDs, xuid)
		winSquadKills = append(winSquadKills, s.kills)
	}
	if len(winSquadKills) < 2 {
		return ""
	}
	var sum int
	for _, k := range winSquadKills {
		sum += k
	}
	avg := float64(sum) / float64(len(winSquadKills))
	var candidate string
	min := avg
	for i, xuid := range winSquadXUIDs {
		k := float64(winSquadKills[i])
		if k > 0 && k < min {
			min = k
			candidate = xuid
		}
	}
	return candidate
}

func findFalseBrother(snap *matchSnapshot, teamOutcomes map[string]canonical.Outcome, squadSet map[string]bool) string {
	var lossXUIDs []string
	var lossKills, lossDeaths []int
	for xuid, s := range snap.stats {
		if !squadSet[xuid] {
			continue
		}
		if teamOutcomes[xuid] != canonical.OutcomeLoss {
			continue
		}
		lossXUIDs = append(lossXUIDs, xuid)
		lossKills = append(lossKills, s.kills)
		lossDeaths = append(lossDeaths, s.deaths)
	}
	if len(lossXUIDs) < 2 {
		return ""
	}
	var worstRatio float64
	var worstXUID string
	for i, xuid := range lossXUIDs {
		if lossDeaths[i] == 0 {
			continue
		}
		var ratio float64
		if lossKills[i] == 0 {
			ratio = float64(lossDeaths[i]) // sentinelle "infini" : gagne sur tout le monde
		} else {
			ratio = float64(lossDeaths[i]) / float64(lossKills[i])
		}
		if ratio > worstRatio && ratio > 1.0 {
			worstRatio = ratio
			worstXUID = xuid
		}
	}
	return worstXUID
}
