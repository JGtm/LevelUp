package analysis

// weapon_correlation_sameclock.go — Corrélation SAME-CLOCK (doctrine « source de dégât »).
//
// Remplace la corrélation fire-events (weapon_correlation.go) : l'arme d'un kill = la SOURCE
// DE DÉGÂT fatale, pas l'arme tirée la plus proche ni l'arme tenue.
//
// Pour chaque kill (tueur=player_index, instant=TimeMS), arme = le dernier record de dégât
// (attaquant == player_index du tueur) AVANT l'instant du kill, tous paquets confondus (cross-
// paquet). Validé RE : couverture-ARME 70%→~82%, accuracy per-paire 94% vs capture live
// (9b191a7f). Détail : .ai/RE_LOG_KILLWEAPON.md §7ter.12 (branche feat/filmdec-continuation) ;
// preuve de référence = tool RE `cmd/tmp_kwval` mode `augcovsc`.
//
// Prérequis prouvé (RE_LOG §7ter.11) : player_index == slot == index-local du film par IDENTITÉ ;
// l'attaquant décodé du record de dégât et le player_index du tueur vivent dans le même espace.

import "sort"

// DamageEvent : un record de dégât arme-à-feu décodé du film (marqueurs 0xd2 & frères), réduit
// aux champs utiles à la corrélation. TimeMS est dans la MÊME unité que Kill.TimeMS (l'alignement
// horloge film↔kill est de la responsabilité de l'appelant, cf. pipeline de sync).
type DamageEvent struct {
	PlayerIndex int    // slot de l'attaquant (== player_index du tueur si c'est son dégât)
	WeaponID    uint64 // identifiant d'arme source (family/asset-id)
	TimeMS      int    // instant du record de dégât (unité commune avec Kill.TimeMS)
}

// CorrelateKillsSameClock attribue à chaque kill l'arme de son dernier dégât fatal.
//
// Reproduit fidèlement `sameClockW` du tool RE : parmi les DamageEvent de l'attaquant == tueur
// dont TimeMS <= kill.TimeMS, retient celui au TimeMS le plus proche du kill. Les kills melee /
// grenade sont HORS périmètre (attribution laissée à leur logique dédiée) → path "none".
func CorrelateKillsSameClock(kills []Kill, damages []DamageEvent, xuidToPI map[string]int) []KillAttribution {
	byPI := damagesByPlayerIndex(damages)
	out := make([]KillAttribution, 0, len(kills))
	for _, k := range kills {
		ka := KillAttribution{
			MatchID:         k.MatchID,
			XUID:            k.XUID,
			TimeMS:          k.TimeMS,
			Confidence:      confidenceNone,
			AttributionPath: AttributionPathNone,
		}
		pi, hasPI := xuidToPI[k.XUID]
		if !k.IsMelee && !k.IsGrenade && hasPI {
			if wid, delta, found := lastDamageBefore(byPI[pi], k.TimeMS); found {
				piCopy, dCopy := pi, delta
				ka.WeaponID = &wid
				ka.PlayerIndex = &piCopy
				ka.DeltaMS = &dCopy
				ka.Confidence = confidenceHigh
				ka.AttributionPath = AttributionPathDamageSource
			}
		}
		out = append(out, ka)
	}
	return out
}

// damagesByPlayerIndex regroupe les DamageEvent par attaquant, chaque groupe trié par TimeMS asc.
func damagesByPlayerIndex(damages []DamageEvent) map[int][]DamageEvent {
	byPI := make(map[int][]DamageEvent)
	for _, d := range damages {
		byPI[d.PlayerIndex] = append(byPI[d.PlayerIndex], d)
	}
	for pi := range byPI {
		g := byPI[pi]
		sort.SliceStable(g, func(i, j int) bool { return g[i].TimeMS < g[j].TimeMS })
	}
	return byPI
}

// lastDamageBefore retourne (arme, delta=killTime-damageTime, found) du dégât au TimeMS le plus
// proche de killTime parmi ceux <= killTime. `sorted` est trié par TimeMS ascendant.
func lastDamageBefore(sorted []DamageEvent, killTimeMS int) (uint64, int, bool) {
	// premier index dont TimeMS > killTimeMS ; le candidat est celui juste avant.
	idx := sort.Search(len(sorted), func(i int) bool { return sorted[i].TimeMS > killTimeMS })
	if idx == 0 {
		return 0, 0, false
	}
	d := sorted[idx-1]
	return d.WeaponID, killTimeMS - d.TimeMS, true
}
