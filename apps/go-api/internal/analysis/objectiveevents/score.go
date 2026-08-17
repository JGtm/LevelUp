package objectiveevents

import "sort"

// score.go — les COURBES tirees des enregistrements d'entite : score de mode et score
// personnel, a la milliseconde.
//
// Complement direct des events d'objectif : ceux-ci disent QUI prend une base ou capture un
// drapeau, celles-ci disent OU EN EST le score a cet instant. Meme horloge, superposables
// sans recalage. Le decodage des enregistrements lui-meme vit dans statborg.go.
//
// # Ou vit quoi, et c'est mesure et non suppose
//
//	composant 0, valeur A : score de MODE      (200-94 en Strongholds, 3-0 en CTF)
//	composant 1, valeur B : score PERSONNEL    (valeur A constamment nulle)
//	composant 2           : frags et morts
//
// # Rendement, confronte position de bit par position de bit a une capture Cheat Engine
//
//	Strongholds 696a9d7c, composant 0 : 283/284 (99,6 %), 2 faux positifs ;
//	CTF         530820e5, composant 0 : 6/6 (100 %), 0 faux positif ;
//	Strongholds, composant 1 : 374/381 · composant 2 : 385/397 ;
//	zero valeur fausse a bonne position dans tous les cas.

// modeScoreComp / personalScoreComp : les index de composant nommes par la mesure.
const (
	modeScoreComp     = 0
	personalScoreComp = 1
)

// ScorePoint est une emission de score par une entite du match.
type ScorePoint struct {
	// TimeMS est l'instant de l'emission sur l'horloge du film.
	TimeMS int
	// Slot identifie l'entite : 6 et 8 sont les deux equipes, 10..24 les huit joueurs.
	Slot int
	// Value est le score a cet instant.
	Value int64
}

// ScoreCurve decode la progression du score de MODE : une entree par changement, par
// entite, horodatee sur l'horloge du film.
//
// En Strongholds le composant n'est emis que par les 2 entites d'equipe ; en CTF les 8
// entites de joueur l'emettent aussi, ou il vaut leur compte de captures. Ce que porte le
// composant depend donc du mode — c'est une mesure, pas une supposition.
func ScoreCurve(src FilmSource) []ScorePoint {
	return ScoreCurveFrom(StatRecords(src))
}

// ScoreCurveFrom est le coeur pur : il travaille sur des enregistrements deja decodes, ce qui
// evite de re-decoder le film a chaque courbe demandee.
//
// # Les manches (2026-08-18)
//
// Le score de mode repart de zero a chaque manche. La courbe rendue est le TOTAL DU MATCH :
// chaque manche est filtree separement puis decalee du total des precedentes. Mesure sur les
// 4 films Oddball du corpus : le score final passe de la seule manche 1 (100/78 sur `24dbb67d`)
// au total (200/121), soit l'oracle, 4 fois sur 4.
//
// Sur un match a une seule manche, le resultat est identique a l'ancien (verifie par les tests
// de verite terrain Strongholds et CTF).
func ScoreCurveFrom(recs []StatRecord) []ScorePoint {
	real := RealRounds(recs)
	teams := cumulateRounds(rawSeriesByRound(recs, statSlotKey{modeScoreComp, sideA}, true), real)
	players := cumulateRounds(rawSeriesByRound(recs, statSlotKey{modeScoreComp, sideA}, false), real)
	var all []ScorePoint
	for _, bySlot := range []map[int][]ScorePoint{teams, players} {
		for _, pts := range bySlot {
			all = append(all, pts...)
		}
	}
	return keepMonotoneBySlot(all)
}

// ScoreRoundsFrom rend la courbe du score de mode PAR MANCHE : par slot, par manche, les
// emissions retenues (valeurs propres a la manche, non cumulees). C'est la forme que publie
// l'artefact de rejeu, ou chaque manche est une courbe distincte.
func ScoreRoundsFrom(recs []StatRecord) map[int]map[int][]ScorePoint {
	out := map[int]map[int][]ScorePoint{}
	real := RealRounds(recs)
	for _, teams := range []bool{true, false} {
		for slot, byRound := range rawSeriesByRound(recs, statSlotKey{modeScoreComp, sideA}, teams) {
			for round, pts := range byRound {
				if !real[round] {
					continue
				}
				sort.SliceStable(pts, func(i, j int) bool { return pts[i].TimeMS < pts[j].TimeMS })
				kept := longestRun(pts, true)
				if len(kept) == 0 {
					continue
				}
				if out[slot] == nil {
					out[slot] = map[int][]ScorePoint{}
				}
				out[slot][round] = kept
			}
		}
	}
	return out
}

// PersonalScoreCurve decode la progression du score PERSONNEL de chaque entite.
//
// Son interet n'est pas le chiffre mais ses INCREMENTS : le jeu attribue un nombre de
// points propre a chaque action, et le depot en a deja le catalogue nomme — la table
// `personal_score_awards` (colonnes `award_name`, `award_category`, `award_score`) donne
// killed_player = 100, kill_assist = 50, flag_captured = 300, flag_returned = 25,
// sensor_assist = 10... Les increments lus ici se rapprochent donc de noms existants.
//
// ATTENTION — le score personnel N'EST PAS MONOTONE : `self_destruction` et
// `betrayed_player` valent **-100**. Aucun filtre de monotonie n'est applique ici (il
// supprimerait ces evenements), contrairement a [ScoreCurve] ou le score de mode, lui, ne
// recule jamais. Le prix a payer est mesure et faible : 3 ancrages parasites pour 377
// lectures reelles (0,8 %) sur le film de reference.
//
// Autre limite mesuree : les increments ne sont pas atomiques. Plusieurs actions tombant
// dans le meme paquet se somment (125 = 100 + 25 observe en CTF). Un increment ne se lit
// donc pas comme UNE action.
func PersonalScoreCurve(src FilmSource) []ScorePoint {
	return collectComponent(StatRecords(src), personalScoreComp, true)
}

// collectComponent extrait un composant des enregistrements ; useB choisit la valeur B
// plutot que la valeur A.
func collectComponent(recs []StatRecord, comp int, useB bool) []ScorePoint {
	var out []ScorePoint
	for _, r := range recs {
		v, ok := r.Comps[comp]
		if !ok {
			continue
		}
		val := v.A
		if useB {
			val = v.B
		}
		out = append(out, ScorePoint{TimeMS: r.TimeMS, Slot: r.Slot, Value: val})
	}
	return out
}

// keepMonotoneBySlot ne garde, par entite, que la plus longue suite d'emissions
// strictement croissante — un score ne recule pas.
//
// Le critere est la LONGUEUR, et c'est ce qui le rend sans parametre : les emissions
// reelles ecrasent en nombre les rares ancrages parasites (283 contre 2 sur le film de
// reference). Un simple filtre glouton « garder ce qui depasse le dernier retenu » a le
// defaut inverse : un parasite a valeur enorme arrivant en PREMIER masque toute la vraie
// courbe derriere lui. Mesure sur les 951 films du cache, a contraintes egales : 10 films
// a valeur aberrante en glouton, **5** avec ce critere.
func keepMonotoneBySlot(pts []ScorePoint) []ScorePoint {
	bySlot := map[int][]ScorePoint{}
	var order []int
	for _, p := range pts {
		if _, seen := bySlot[p.Slot]; !seen {
			order = append(order, p.Slot)
		}
		bySlot[p.Slot] = append(bySlot[p.Slot], p)
	}
	out := pts[:0:0]
	for _, slot := range order {
		out = append(out, longestRun(bySlot[slot], true)...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TimeMS != out[j].TimeMS {
			return out[i].TimeMS < out[j].TimeMS
		}
		return out[i].Slot < out[j].Slot
	})
	return out
}

// longestRun rend la plus longue sous-suite croissante en valeur, l'ordre d'entree etant
// deja chronologique (patience sorting, O(n log n)).
//
// strict choisit entre les deux usages, et la difference n'est pas cosmetique :
//   - le score de MODE est strictement croissant (une egalite serait un doublon a jeter) ;
//   - un compteur de recompense est NON DECROISSANT — un composant porte deux valeurs et
//     il est reemis des que l'UNE des deux bouge, donc la meme valeur revient
//     legitimement. Exiger la stricte croissance y jetterait des emissions reelles.
func longestRun(pts []ScorePoint, strict bool) []ScorePoint {
	if len(pts) == 0 {
		return nil
	}
	var tailIdx []int             // tailIdx[k] = index de la fin de la suite de longueur k+1
	prev := make([]int, len(pts)) // chainage arriere pour la reconstruction
	for i, p := range pts {
		lo, hi := 0, len(tailIdx)
		for lo < hi {
			mid := (lo + hi) / 2
			fits := pts[tailIdx[mid]].Value <= p.Value
			if strict {
				fits = pts[tailIdx[mid]].Value < p.Value
			}
			if fits {
				lo = mid + 1
			} else {
				hi = mid
			}
		}
		prev[i] = -1
		if lo > 0 {
			prev[i] = tailIdx[lo-1]
		}
		if lo == len(tailIdx) {
			tailIdx = append(tailIdx, i)
		} else {
			tailIdx[lo] = i
		}
	}
	out := make([]ScorePoint, len(tailIdx))
	for i, k := len(tailIdx)-1, tailIdx[len(tailIdx)-1]; i >= 0; i, k = i-1, prev[k] {
		out[i] = pts[k]
	}
	return out
}
