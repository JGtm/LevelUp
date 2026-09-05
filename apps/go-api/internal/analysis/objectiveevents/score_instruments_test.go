package objectiveevents

import (
	"sort"

	"levelup/go-api/internal/analysis/filmsource"
)

// score_instruments_test.go — LES COURBES QUI NE SERVENT QU AUX INSTRUMENTS DE MESURE.
//
// POURQUOI CES FONCTIONS SONT ICI ET PLUS DANS `score.go` (revue R1, 2026-08-18). Elles etaient
// EXPORTEES depuis le paquet de production et leur seul client etait un test : c est l anti-patron
// « dead code museum » que le depot interdit — du code conserve « au cas ou », avec des tests verts
// qui entretiennent l illusion qu il sert.
//
// Ce qu elles servent vraiment : les instruments de mesure du lot A (phases 0, 0-bis, 0-ter), qui
// comparent la derniere valeur d une courbe a l oracle du registre. La PRODUCTION, elle, passe par
// `SeriesByRound` / `SeriesTotal` (score.go) et par l assembleur `replay/score_timeline.go`, qui
// publient les deux formes — par manche et en total — au lieu d une seule courbe cumulee.
//
// Elles ne sont donc ni supprimees (les mesures se rejouent) ni exportees en production.
// ScoreCurve decode la progression du score de MODE : une entree par changement, par
// entite, horodatee sur l'horloge du film.
//
// En Strongholds le composant n'est emis que par les 2 entites d'equipe ; en CTF les 8
// entites de joueur l'emettent aussi, ou il vaut leur compte de captures. Ce que porte le
// composant depend donc du mode — c'est une mesure, pas une supposition.
func ScoreCurve(film *filmsource.Film) []ScorePoint {
	return ScoreCurveFrom(StatRecords(film))
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
	var all []ScorePoint
	for _, teams := range []bool{true, false} {
		for _, pts := range SeriesTotal(recs, ModeScoreComponent, teams) {
			all = append(all, pts...)
		}
	}
	return keepMonotoneBySlot(all)
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
func PersonalScoreCurve(film *filmsource.Film) []ScorePoint {
	return collectComponent(StatRecords(film), personalScoreComp, true)
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
