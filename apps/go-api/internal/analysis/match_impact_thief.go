// Package analysis — match_impact_thief.go : badge d'impact « Voleur ».
//
// Critère (demande utilisateur 2026-09-16) : sur un match, le joueur de l'escouade qui a
// le plus d'éliminations « volées » à un ami. Une élimination est volée quand :
//
//   - le tueur crédité ET l'assistant sont tous deux membres de l'escouade ;
//   - la part de dégâts du tueur sur ce kill est MESURÉE et vaut au plus
//     thiefMaxKillerDamagePct (10 %) ;
//   - l'assistant est vivant AVANT et APRÈS le kill.
//
// « VIVANT AVANT ET APRÈS », à la demi-seconde (précision utilisateur 2026-09-16) :
// l'assistant ne doit avoir AUCUNE mort dans [kill - thiefAssistAliveMarginMS,
// kill + thiefAssistAliveMarginMS], bornes incluses. Un échange (assistant tué juste
// avant, au même instant ou juste après le kill) n'est donc pas un vol.
//
// Les parts de dégâts ne sont pas bornées à 100 (cf. migration/steps_shared_kill_events.go)
// : le seuil porte sur une borne haute côté tueur, une valeur > 100 n'est jamais volée.
package analysis

// thiefMaxKillerDamagePct : part de dégâts maximale (incluse) du tueur crédité.
const thiefMaxKillerDamagePct = 10

// thiefAssistAliveMarginMS : marge (ms) de part et d'autre du kill pendant laquelle
// l'assistant ne doit pas mourir (cf. en-tête).
const thiefAssistAliveMarginMS int64 = 500

// BadgeKeyThief : clé technique du badge « Voleur ».
const BadgeKeyThief = "thief"

// KillLogEntry : une mort du journal des morts d'un match (match_kill_events_latest),
// réduite à ce que le badge « Voleur » consulte. Les champs vides / nil valent
// « non mesuré » : jamais un zéro fabriqué.
type KillLogEntry struct {
	TimeMS     int64
	KillerXUID string
	VictimXUID string
	AssistXUID string
	// KillerDamagePct : part de dégâts du tueur crédité. nil = non mesurée → jamais volée.
	KillerDamagePct *int
}

// ComputeThiefBadge rend le badge « Voleur » d'un match, ou nil.
//
// friends : xuids de l'escouade (tueur ET assistant doivent y appartenir).
// Départage : nombre de vols décroissant, puis dernier vol le plus tardif, puis xuid
// croissant (même règle que kamikaze). TimeMS du badge = instant du dernier vol.
func ComputeThiefBadge(entries []KillLogEntry, friends map[string]bool) *ImpactBadge {
	if len(entries) == 0 || len(friends) == 0 {
		return nil
	}
	deathsByXUID := make(map[string][]int64)
	for _, e := range entries {
		if e.VictimXUID != "" {
			deathsByXUID[e.VictimXUID] = append(deathsByXUID[e.VictimXUID], e.TimeMS)
		}
	}
	type acc struct {
		count int
		last  int64
	}
	per := make(map[string]*acc)
	for _, e := range entries {
		if !isStolenKill(e, friends, deathsByXUID[e.AssistXUID]) {
			continue
		}
		a, ok := per[e.KillerXUID]
		if !ok {
			a = &acc{}
			per[e.KillerXUID] = a
		}
		a.count++
		if e.TimeMS > a.last {
			a.last = e.TimeMS
		}
	}
	var bestXUID string
	var best acc
	for xuid, a := range per {
		better := a.count > best.count ||
			(a.count == best.count && a.last > best.last) ||
			(a.count == best.count && a.last == best.last && (bestXUID == "" || xuid < bestXUID))
		if better {
			bestXUID, best = xuid, *a
		}
	}
	if bestXUID == "" {
		return nil
	}
	return &ImpactBadge{BadgeKey: BadgeKeyThief, BadgeFR: "Voleur", PlayerXUID: bestXUID, TimeMS: best.last}
}

// isStolenKill applique le critère « élimination volée » à une mort.
func isStolenKill(e KillLogEntry, friends map[string]bool, assistDeaths []int64) bool {
	if e.KillerXUID == "" || e.AssistXUID == "" || e.KillerXUID == e.AssistXUID {
		return false
	}
	if !friends[e.KillerXUID] || !friends[e.AssistXUID] {
		return false
	}
	if e.KillerDamagePct == nil || *e.KillerDamagePct > thiefMaxKillerDamagePct {
		return false
	}
	for _, d := range assistDeaths {
		if d >= e.TimeMS-thiefAssistAliveMarginMS && d <= e.TimeMS+thiefAssistAliveMarginMS {
			return false
		}
	}
	return true
}
