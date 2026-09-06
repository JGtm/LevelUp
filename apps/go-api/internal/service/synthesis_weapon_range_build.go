// Package service — synthesis_weapon_range_build.go : L'ASSEMBLAGE DU BLOC « PORTÉE PAR ARME ».
//
// Fonctions PURES : elles prennent la sortie de `internal/analysis` et la mettent à la forme du
// contrat (`domain.SynthesisWeaponRange`). Aucune I/O, aucun log, aucune horloge — d'où leur
// testabilité sans mock, et d'où la séparation d'avec synthesis_weapon_range.go, qui porte le
// repo et le régime d'échec.
//
// # CE QUI SE DÉCIDE ICI
//
// Une seule chose, et elle mérite son fichier : le passage d'UNE LIGNE PAR COUPLE (arme, côté)
// — la forme de l'agrégat, qui est la bonne pour un calcul — à UNE LIGNE PAR ARME PORTANT SES
// DEUX CÔTÉS, la forme du rendu (l'utilisateur a demandé la fusion des deux graphes jumeaux).
// Le reste est de la conversion d'unités : les classes de dénivelé deviennent des pourcentages.
package service

import (
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// buildWeaponRangeBlock assemble le bloc publié.
//
// LES MÉDIANES GLOBALES NE SORTENT PAS DES LIGNES PUBLIÉES : elles sont calculées sur TOUS les
// frags mesurés (`analysis.WeaponRangeSideTotals`), seuil non appliqué. Sinon les deux nombres
// de tête bougeraient au gré du seuil, et « N frags mesurés sur M » ne décrirait plus la
// couverture réelle mais la sélection d'armes publiables.
func buildWeaponRangeBlock(
	kills, openings []analysis.MeasuredKill, scope weaponRangeScopeInfo,
) *domain.SynthesisWeaponRange {
	rows, summary := analysis.WeaponRangeAggregate(kills, analysis.WeaponRangeMinMeasured)

	medKills, measuredKills := analysis.WeaponRangeSideTotals(kills, analysis.SideKiller)
	medDeaths, measuredDeaths := analysis.WeaponRangeSideTotals(kills, analysis.SideVictim)

	block := &domain.SynthesisWeaponRange{
		Weapons:              mergeWeaponSides(rows),
		MedianKillsM:         medKills,
		MedianDeathsM:        medDeaths,
		MeasuredKills:        measuredKills,
		TotalKills:           scope.totalKills,
		MeasuredDeaths:       measuredDeaths,
		TotalDeaths:          scope.totalDeaths,
		BelowThresholdKills:  belowThresholdOfSide(summary, analysis.SideKiller),
		BelowThresholdDeaths: belowThresholdOfSide(summary, analysis.SideVictim),
		Opening:              buildOpening(kills, openings),
	}
	return block
}

// mergeWeaponSides replie les couples (arme, côté) en une ligne par arme.
//
// Tri de sortie : médiane des FRAGS croissante — le graphe se lit du contact à la longue
// portée (D6). Une arme sans frag publié (je meurs sous cette arme, je n'en tue pas) se range
// à la médiane de ses MORTS : elle a sa place dans le même continuum, et l'exclure du tri la
// renverrait arbitrairement en tête ou en queue.
func mergeWeaponSides(rows []analysis.WeaponRange) []domain.WeaponRangeRow {
	index := make(map[string]int, len(rows))
	out := make([]domain.WeaponRangeRow, 0, len(rows))
	cles := make([]float64, 0, len(rows))

	for _, r := range rows {
		i, connue := index[r.WeaponKey]
		if !connue {
			i = len(out)
			index[r.WeaponKey] = i
			out = append(out, domain.WeaponRangeRow{WeaponKey: r.WeaponKey})
			cles = append(cles, r.Median)
		}
		side := weaponRangeSideOf(r)
		if r.Side == analysis.SideKiller {
			out[i].Kills = side
			cles[i] = r.Median // la médiane des frags PRIME sur celle des morts
			continue
		}
		out[i].Deaths = side
		if out[i].Kills == nil {
			cles[i] = r.Median
		}
	}

	ordre := make([]int, len(out))
	for i := range ordre {
		ordre[i] = i
	}
	sort.SliceStable(ordre, func(a, b int) bool {
		ia, ib := ordre[a], ordre[b]
		if cles[ia] != cles[ib] {
			return cles[ia] < cles[ib]
		}
		return out[ia].WeaponKey < out[ib].WeaponKey
	})
	trie := make([]domain.WeaponRangeRow, len(out))
	for rang, i := range ordre {
		trie[rang] = out[i]
	}
	return trie
}

// weaponRangeSideOf convertit UN côté agrégé : les percentiles passent tels quels, les trois
// classes de dénivelé deviennent des pourcentages (convention `*Pct` du dépôt, 0..100).
//
// Le dénominateur est `Measured`, qui ne peut pas être nul : un couple (arme, côté) naît d'au
// moins un frag, et un couple sous le seuil n'arrive jamais jusqu'ici.
func weaponRangeSideOf(r analysis.WeaponRange) *domain.WeaponRangeSide {
	part := func(n int) float64 { return 100 * float64(n) / float64(r.Measured) }
	return &domain.WeaponRangeSide{
		Measured: r.Measured,
		P10:      r.P10,
		Median:   r.Median,
		P90:      r.P90,
		AbovePct: part(r.Above),
		LevelPct: part(r.Level),
		BelowPct: part(r.Below),
	}
}

// belowThresholdOfSide projette les couples écartés d'UN côté. L'ordre vient de l'agrégat
// (effectif décroissant, puis clé) et n'est pas rejoué ici : deux tris du même fait
// divergeraient au premier changement de doctrine.
func belowThresholdOfSide(s analysis.WeaponRangeSummary, side analysis.Side) []domain.WeaponBelowThreshold {
	var out []domain.WeaponBelowThreshold
	for _, b := range s.BelowThresholdRows {
		if b.Side != side {
			continue
		}
		out = append(out, domain.WeaponBelowThreshold{WeaponKey: b.WeaponKey, Measured: b.Measured})
	}
	return out
}

// buildOpening rend le bloc d'entame, ou NIL si aucune entame n'est mesurée.
//
// NIL ET JAMAIS UN BLOC À ZÉRO (D5) : tant que le backfill de `kill_openings` n'a pas tourné —
// et il ne tournera que sur décision utilisateur — la couverture est nulle ou partielle. Un
// bloc à zéro se lirait « ce joueur engage au contact » ; l'absence se lit « on ne sait pas ».
//
// LA MÊME RÈGLE S'APPLIQUE UN CRAN PLUS BAS, AU DELTA (2026-09-06, constat F9). Les deux
// tables s'écrivent sous deux leases indépendants : un scope peut porter des entames sans
// aucune position de coup fatal, et `Paired` vaut alors zéro alors que `MeasuredOpenings` ne
// l'est pas. Le sous-bloc est OMIS dans ce cas — un `closing_share_pct: 0` publié dirait « ce
// joueur ne ferme jamais la distance », ce qu'aucune mesure ne soutient.
func buildOpening(kills, openings []analysis.MeasuredKill) *domain.SynthesisOpening {
	st := analysis.WeaponOpeningDelta(kills, openings, analysis.SideKiller)
	if st.MeasuredOpenings == 0 {
		return nil
	}
	block := &domain.SynthesisOpening{
		MedianM:       st.MedianOpeningM,
		MeasuredKills: st.MeasuredOpenings,
	}
	if st.Paired > 0 {
		block.Delta = &domain.SynthesisOpeningDelta{
			MedianM:         st.MedianDeltaM,
			ClosingSharePct: 100 * st.ClosingShare,
			N:               st.Paired,
		}
	}
	return block
}
