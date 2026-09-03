package objectiveevents

// named_series.go — DES ENREGISTREMENTS AUX SERIES CUMULEES.
//
// Extrait de `named.go` le 2026-09-03 (lot 4) : le regroupement en une passe y avait porte le
// fichier au-dela du seuil de 500 lignes. La coupe suit la RESPONSABILITE, pas la ligne — ici
// vit tout ce qui va d'une liste d'enregistrements d'entite a une suite de valeurs
// exploitable (groupement par emplacement / slot / manche, rejet des ancrages parasites,
// cumul des manches, conversion d'un compteur en instants) ; `named.go` garde la table des
// emplacements, le type d'evenement et la publication.

import "sort"

// rawSeriesByKey est [rawSeriesByRound] pour TOUS les emplacements NON REDONDANTS d'une table
// a la fois, en une seule marche de `recs` — memes filtres, meme ordre d'insertion par
// (emplacement, slot, manche), donc memes series.
//
// Les emplacements redondants sont ecartes ICI : ils n'emettent aucun evenement, et les
// grouper serait du travail jete.
func rawSeriesByKey(recs []StatRecord, table map[statSlotKey]statSlot) map[statSlotKey]map[int]map[int][]ScorePoint {
	out := make(map[statSlotKey]map[int]map[int][]ScorePoint, len(table))
	for key, slot := range table {
		if !slot.Redundant {
			out[key] = map[int]map[int][]ScorePoint{}
		}
	}
	for _, r := range recs {
		if IsTeamSlot(r.Slot) { // seuls les slots de JOUEUR nomment des evenements
			continue
		}
		for key, raw := range out {
			v, ok := r.Comps[key.Comp]
			if !ok {
				continue
			}
			val := v.A
			if key.Side == sideB {
				val = v.B
			}
			// Memes deux rejets que [rawSeriesByRound], et pour les memes raisons : une
			// emission negative est un ancrage parasite, et un score de mode hors domaine
			// denonce une emission mal alignee sur ses DEUX canaux.
			if val < 0 || (key.Comp == modeScoreComp && !modeScoreInDomain(v)) {
				continue
			}
			if raw[r.Slot] == nil {
				raw[r.Slot] = map[int][]ScorePoint{}
			}
			raw[r.Slot][r.Round] = append(raw[r.Slot][r.Round],
				ScorePoint{TimeMS: r.TimeMS, Slot: r.Slot, Value: val})
		}
	}
	return out
}

// seriesBySlot rend, par slot de JOUEUR, la suite chronologique des valeurs d'un
// emplacement, debarrassee des ancrages parasites.
//
// Le filtre est le meme que celui du score de mode et pour la meme raison : un compteur de
// recompense ne recule jamais, donc la plus longue sous-suite NON DECROISSANTE est la vraie
// suite. Non decroissante et non strictement croissante : un composant porte deux valeurs
// et il est reemis des que l'UNE des deux bouge, donc la meme valeur revient legitimement.
// # Les MANCHES, et pourquoi la suite est cumulee (2026-08-18)
//
// Un compteur repart de zero a chaque manche (`StatRecord.Round`). Concatener les manches sans
// rien faire donnerait une suite qui RECULE, et le filtre de plus longue sous-suite n'en
// garderait qu'une — c'est exactement ce que faisait la version d'avant, qui ne voyait de toute
// facon que la manche 1. Chaque manche est donc filtree separement, puis DECALEE du total des
// manches precedentes : la suite rendue est croissante sur tout le match et son dernier point
// est le total du match. Mesure : les frags d'un Oddball passent de 48 a 87 sur 88 attendus.
func seriesBySlot(recs []StatRecord, key statSlotKey) map[int][]ScorePoint {
	return cumulateRounds(rawSeriesByRound(recs, key, false), RealRounds(recs))
}

// rawSeriesByRound groupe les emissions par slot puis par manche, en jetant les ancrages
// parasites. teams choisit les slots d'equipe plutot que ceux de joueur.
func rawSeriesByRound(recs []StatRecord, key statSlotKey, teams bool) map[int]map[int][]ScorePoint {
	raw := map[int]map[int][]ScorePoint{}
	for _, r := range recs {
		if IsTeamSlot(r.Slot) != teams {
			continue
		}
		v, ok := r.Comps[key.Comp]
		if !ok {
			continue
		}
		val := v.A
		if key.Side == sideB {
			val = v.B
		}
		// Une emission NEGATIVE est un ancrage parasite : un compteur de recompense est
		// positif. Elle est jetee ICI, avant le choix de la sous-suite, et pas apres —
		// sinon elle fausse ce choix. Mesure : sur la suite (1, -115, 1), la plus longue
		// sous-suite non decroissante retenue devenait (-115, 1), ce qui datait
		// l'evenement de la DERNIERE emission au lieu de la premiere.
		if val < 0 {
			continue
		}
		// LE SCORE DE MODE EST BORNE SUR SES DEUX CANAUX, pas seulement sur celui qu'on lit.
		// Les deux valeurs d'un composant sortent de la MEME emission : un canal aberrant
		// prouve que l'emission etait mal alignee, et la valeur de l'autre ne vaut rien non
		// plus. Mesure du 2026-08-31 sur 65 films (3 986 enregistrements joueur porteurs du
		// composant 0) : le canal B vaut ZERO dans 98,3 % des cas, et l'enregistrement
		// `ce083875` slot 16 a 219075 ms porte A=66 avec B=16635 — un saut de 66 unites que
		// [incrementTimes] transformait en 66 explosions publiees au meme instant. Sa seule
		// marque distinctive est ce B hors domaine ; son A passait la borne.
		if key.Comp == modeScoreComp && !modeScoreInDomain(v) {
			continue
		}
		if raw[r.Slot] == nil {
			raw[r.Slot] = map[int][]ScorePoint{}
		}
		raw[r.Slot][r.Round] = append(raw[r.Slot][r.Round],
			ScorePoint{TimeMS: r.TimeMS, Slot: r.Slot, Value: val})
	}
	return raw
}

// cumulateRounds filtre chaque manche par la plus longue sous-suite non decroissante, puis
// decale les manches successives du total des precedentes. Les manches absentes de `real` sont
// IGNOREES : ce sont des ancrages fortuits, et les cumuler ferait exploser les compteurs.
func cumulateRounds(raw map[int]map[int][]ScorePoint, real map[int]bool) map[int][]ScorePoint {
	out := make(map[int][]ScorePoint, len(raw))
	for slot, byRound := range raw {
		var offset int64
		for _, round := range sortedRounds(byRound) {
			if !real[round] {
				continue
			}
			pts := byRound[round]
			sort.SliceStable(pts, func(i, j int) bool { return pts[i].TimeMS < pts[j].TimeMS })
			kept := longestRun(pts, false)
			if len(kept) == 0 {
				continue
			}
			for _, p := range kept {
				out[slot] = append(out[slot], ScorePoint{
					TimeMS: p.TimeMS, Slot: slot, Value: p.Value + offset})
			}
			offset += kept[len(kept)-1].Value
		}
	}
	return out
}

// sortedRounds rend les manches d'un slot, dans l'ordre — l'ordre du cumul en depend.
func sortedRounds(byRound map[int][]ScorePoint) []int {
	out := make([]int, 0, len(byRound))
	for r := range byRound {
		out = append(out, r)
	}
	sort.Ints(out)
	return out
}

// incrementTimes rend un instant par UNITE gagnee par le compteur : c'est la conversion
// d'un compteur en evenements.
//
// La premiere valeur observee est comptee depuis zero — un compteur de recompense part de
// zero au coup d'envoi. Si le film ne montre le slot qu'apres coup, les unites deja
// acquises sont datees de cette premiere emission, ce qui MAJORE leur instant.
//
// # Le garde-fou, et il a ete paye
//
// `prev` ne redescend JAMAIS, sinon la meme unite se compte deux fois apres un creux. Sans
// cela, une seule emission aberrante a -115 faisait remonter le compteur de 0 a 1 en **116**
// evenements (mesure sur `1bc77d2e`, slot 24, comp 0 A). Les emissions negatives elles-memes
// sont ecartees plus tot, par [seriesBySlot].
func incrementTimes(pts []ScorePoint) []int {
	var out []int
	prev := int64(0)
	for _, p := range pts {
		for ; prev < p.Value; prev++ {
			out = append(out, p.TimeMS)
		}
	}
	return out
}
