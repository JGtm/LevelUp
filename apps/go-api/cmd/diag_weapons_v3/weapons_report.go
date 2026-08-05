package main

// weapons_report.go — calcul + impression du rapport de comparaison §4 (v3 vs v2).
//
// Rapport par match (et agrégat panel via panelAgg) :
//  1. Distribution confidence v3 (high/medium/low/none) count + %.
//  2. % NULL/no-weapon v3 (WeaponID==nil ET HighWeaponID==nil).
//  3. # armes distinctes effectives v3 (par high-32).
//  4. Breakdown SourceSignal v3 (melee/grenade/fire/formula_a/none).
//  5. vs v2 : distribution confidence v2 + AGREEMENT (sur v2.confidence=='high',
//     %% où v3 donne le MÊME high-32 ; régression si < 98%).
//  6. Récupération : # kills v2=none/null → v3 résolu (melee/grenade/fire).
//  7. vs agrégats : melee/grenade v3 vs match_participants.{melee,grenade}_kills (±1).

import (
	"sort"

	"levelup/go-api/internal/analysis/weaponv3"
)

// confLevels — ordre stable d'affichage des niveaux de confiance.
var confLevels = []string{"high", "medium", "low", "none"}

// signalLevels — ordre stable d'affichage des SourceSignal v3.
var signalLevels = []string{
	weaponv3.SignalMelee, weaponv3.SignalGrenade, weaponv3.SignalFire,
	weaponv3.SignalFormulaA, weaponv3.SignalNone,
}

// aggToleranceKills — tolérance |v3 - API| sur les compteurs melee/grenade
// per-joueur (§0 : melee/grenade attribués ≈ agrégats ±1).
const aggToleranceKills = 1

// weaponReport — métriques §4 d'UN match (et agrégeable sur le panel).
type weaponReport struct {
	total      int            // kills v3 (= kills highlight_events)
	confV3     map[string]int // distribution confiance v3
	nullV3     int            // kills v3 sans arme (WeaponID nil ET HighWeaponID nil)
	signalV3   map[string]int // distribution SourceSignal v3
	distinctV3 map[uint32]bool

	confV2  map[string]int // distribution confiance v2 (kills appariés)
	v2HighN int            // # kills v2=high appariés à un v3
	agreeN  int            // # où v3 donne le même high-32 que v2=high
	pairedN int            // # kills appariés v2<->v3

	recoveredMelee   int // v2 none/null → v3 melee
	recoveredGrenade int // v2 none/null → v3 grenade
	recoveredFire    int // v2 none/null → v3 fire

	aggHasCols bool
	aggLines   []aggCompareLine // §4.7 par joueur (vide si !aggHasCols)
}

// aggCompareLine — comparaison melee/grenade v3 vs API pour un joueur.
type aggCompareLine struct {
	xuid                       string
	v3Melee, apiMelee          int
	v3Grenade, apiGrenade      int
	meleeWithinTol, grenWithin bool
}

// buildWeaponReport calcule les métriques §4 d'un match depuis les attributions
// v3, la baseline v2 appariée et les agrégats per-joueur.
func buildWeaponReport(
	attrs []weaponv3.AttributionV3,
	baseline map[killKey]v2Kill,
	agg participantAgg,
) weaponReport {
	r := weaponReport{
		confV3:     map[string]int{},
		signalV3:   map[string]int{},
		distinctV3: map[uint32]bool{},
		confV2:     map[string]int{},
		aggHasCols: agg.hasCols,
	}
	v3Melee := map[string]int{}
	v3Grenade := map[string]int{}

	for _, a := range attrs {
		accumulateV3(&r, a)
		accumulateSignalCounts(a, v3Melee, v3Grenade)
		accumulateVsV2(&r, a, baseline)
	}
	for _, k := range baseline {
		r.confV2[normConf(k.confidence)]++
	}
	if agg.hasCols {
		r.aggLines = buildAggLines(v3Melee, v3Grenade, agg)
	}
	return r
}

// accumulateV3 met à jour les compteurs purement-v3 (confiance, no-weapon,
// distinct high-32, signal).
func accumulateV3(r *weaponReport, a weaponv3.AttributionV3) {
	r.total++
	r.confV3[normConf(a.Confidence)]++
	r.signalV3[normSignal(a.SourceSignal)]++
	if a.WeaponID == nil && a.HighWeaponID == nil {
		r.nullV3++
	}
	if a.HighWeaponID != nil {
		r.distinctV3[*a.HighWeaponID] = true
	}
}

// accumulateSignalCounts incrémente les compteurs melee/grenade per-joueur v3
// (pour la comparaison agrégats §4.7).
func accumulateSignalCounts(a weaponv3.AttributionV3, melee, grenade map[string]int) {
	switch a.SourceSignal {
	case weaponv3.SignalMelee:
		melee[a.XUID]++
	case weaponv3.SignalGrenade:
		grenade[a.XUID]++
	}
}

// accumulateVsV2 apparie un kill v3 à la baseline v2 (xuid,time_ms) et met à jour
// l'agreement (v2=high → même high-32) + la récupération (v2 none/null → v3 résolu).
func accumulateVsV2(r *weaponReport, a weaponv3.AttributionV3, baseline map[killKey]v2Kill) {
	v2, ok := baseline[killKey{xuid: a.XUID, timeMS: a.TimeMS}]
	if !ok {
		return
	}
	r.pairedN++
	if normConf(v2.confidence) == "high" {
		r.v2HighN++
		if sameHigh32(a, v2) {
			r.agreeN++
		}
	}
	if v2NoneOrNull(v2) {
		accumulateRecovery(r, a)
	}
}

// accumulateRecovery compte un kill v2 none/null que la v3 a résolu, par signal.
func accumulateRecovery(r *weaponReport, a weaponv3.AttributionV3) {
	if a.WeaponID == nil && a.HighWeaponID == nil {
		return // toujours non résolu côté v3
	}
	switch a.SourceSignal {
	case weaponv3.SignalMelee:
		r.recoveredMelee++
	case weaponv3.SignalGrenade:
		r.recoveredGrenade++
	case weaponv3.SignalFire:
		r.recoveredFire++
	}
}

// buildAggLines construit les lignes §4.7 (melee/grenade v3 vs API ±1) triées par
// xuid, sur l'union des joueurs apparaissant dans v3 ou dans les agrégats.
func buildAggLines(v3Melee, v3Grenade map[string]int, agg participantAgg) []aggCompareLine {
	xuids := map[string]bool{}
	for x := range v3Melee {
		xuids[x] = true
	}
	for x := range v3Grenade {
		xuids[x] = true
	}
	for x := range agg.byXUID {
		xuids[x] = true
	}
	out := make([]aggCompareLine, 0, len(xuids))
	for x := range xuids {
		api := agg.byXUID[x]
		line := aggCompareLine{
			xuid: x, v3Melee: v3Melee[x], apiMelee: api.melee,
			v3Grenade: v3Grenade[x], apiGrenade: api.grenade,
		}
		line.meleeWithinTol = absDiff(line.v3Melee, line.apiMelee) <= aggToleranceKills
		line.grenWithin = absDiff(line.v3Grenade, line.apiGrenade) <= aggToleranceKills
		// On n'affiche que les joueurs ayant au moins un melee/grenade (v3 ou API).
		if line.v3Melee+line.apiMelee+line.v3Grenade+line.apiGrenade > 0 {
			out = append(out, line)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].xuid < out[j].xuid })
	return out
}

// sameHigh32 indique si l'attribution v3 et le kill v2 désignent le même high-32.
func sameHigh32(a weaponv3.AttributionV3, v2 v2Kill) bool {
	v3High, v3OK := highOf(a)
	if !v3OK || v2.weaponID == nil {
		return false
	}
	return v3High == uint32(*v2.weaponID>>32)
}

// highOf renvoie le high-32 de l'attribution v3 (HighWeaponID, ou dérivé de
// WeaponID), ok=false si aucune arme.
func highOf(a weaponv3.AttributionV3) (uint32, bool) {
	if a.HighWeaponID != nil {
		return *a.HighWeaponID, true
	}
	if a.WeaponID != nil {
		return uint32(*a.WeaponID >> 32), true
	}
	return 0, false
}

// v2NoneOrNull indique un kill v2 non résolu (confidence none OU sans arme).
func v2NoneOrNull(v2 v2Kill) bool {
	return normConf(v2.confidence) == "none" || v2.weaponID == nil
}

// normConf normalise une string confidence (vide → "none").
func normConf(c string) string {
	if c == "" {
		return "none"
	}
	return c
}

// normSignal normalise un SourceSignal (vide → none).
func normSignal(s string) string {
	if s == "" {
		return weaponv3.SignalNone
	}
	return s
}

// absDiff renvoie |a-b|.
func absDiff(a, b int) int {
	if a < b {
		return b - a
	}
	return a - b
}
