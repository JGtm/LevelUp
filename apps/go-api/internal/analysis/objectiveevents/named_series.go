package objectiveevents

// named_series.go — DES ENREGISTREMENTS AUX SERIES CUMULEES.
//
// Extrait de `named.go` le 2026-09-03 (lot 4) : le regroupement en une passe y avait porte le
// fichier au-dela du seuil de 500 lignes. La coupe suit la RESPONSABILITE, pas la ligne — ici
// vit tout ce qui va d'une liste d'enregistrements d'entite a une suite de valeurs
// exploitable (groupement par emplacement / slot / manche, rejet des ancrages parasites,
// cumul des manches) ; `named.go` garde la table des emplacements, le type d'evenement et la
// publication, et `named_bounds.go` la derivation unique des increments filtres (correctif 6.R,
// 2026-09-11 — le seuil de 500 lignes etait de nouveau atteint).

import (
	"log/slog"
	"sort"
)

// rawSeriesByKey est [rawSeriesByRound] pour TOUS les emplacements NON REDONDANTS d'une table
// a la fois, en une seule marche de `recs` — memes filtres, meme ordre d'insertion par
// (emplacement, slot, manche), donc memes series.
//
// Les emplacements redondants sont ecartes ICI : ils n'emettent aucun evenement, et les
// grouper serait du travail jete.
func rawSeriesByKey(recs []StatRecord, table map[statSlotKey]statSlot) map[statSlotKey]map[int]map[int][]ScorePoint {
	bornes := ResolveRoundBounds(recs)
	out := make(map[statSlotKey]map[int]map[int][]ScorePoint, len(table))
	for key, slot := range table {
		if !slot.Redundant {
			out[key] = map[int]map[int][]ScorePoint{}
		}
	}
	for _, r := range recs {
		// Seuls les slots de JOUEUR nomment des evenements ; et la manche declaree doit
		// s'accorder au temps, comme dans [rawSeriesByRound] (les deux marches rendent les
		// memes series, `named_onepass_test.go` en est le garde-rail).
		if IsTeamSlot(r.Slot) || bornes.Excludes(r) {
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
//
// LA MANCHE DECLAREE EST CONFRONTEE AU TEMPS (2026-09-06, cf. round_bounds.go) : les manches
// se jouent dans l'ordre, donc un enregistrement date hors de l'intervalle de la manche qu'il
// declare a une manche mal lue et n'alimente aucune serie. C'est le filtre qui manquait pour
// que [longestRun] ne soit pas trompe — une valeur mal lue mais PLUS GRANDE prolonge la suite
// non decroissante au lieu de la rompre.
func rawSeriesByRound(recs []StatRecord, key statSlotKey, teams bool) map[int]map[int][]ScorePoint {
	bornes := ResolveRoundBounds(recs)
	raw := map[int]map[int][]ScorePoint{}
	for _, r := range recs {
		if IsTeamSlot(r.Slot) != teams || bornes.Excludes(r) {
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
//
// La suite assemblee passe par [ChronologicalTotal] : le cumul suppose que l'ordre des MANCHES
// est l'ordre du TEMPS, et cette supposition doit etre verifiee, pas presumee.
func cumulateRounds(raw map[int]map[int][]ScorePoint, real map[int]bool) map[int][]ScorePoint {
	out := make(map[int][]ScorePoint, len(raw))
	for slot, byRound := range raw {
		var offset int64
		var serie []ScorePoint
		for _, round := range sortedIntKeys(byRound) {
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
				serie = append(serie, ScorePoint{
					TimeMS: p.TimeMS, Slot: slot, Value: p.Value + offset})
			}
			offset += kept[len(kept)-1].Value
		}
		if serie = ChronologicalTotal(serie); len(serie) > 0 {
			out[slot] = serie
		}
	}
	return out
}

// ChronologicalTotal ecarte d'une suite CUMULEE tout point date avant le dernier point retenu,
// et rend le nombre d'ecartes.
//
// # POURQUOI CE CONTROLE EXISTE
//
// Un total de match est construit en concatenant les manches DANS L'ORDRE DES MANCHES, chacune
// decalee du total des precedentes. Cela ne donne une suite chronologique que si l'ordre des
// manches EST celui du temps — c'est-a-dire si la decoupe par manche est juste. Elle ne l'etait
// pas : un enregistrement de la manche 1 range en manche 0 faisait rendre `{3167, 60}` puis
// `{3057, 61}` sur `51ebbc0f`, une courbe qui RECULE dans le temps (mesure du 2026-09-06). La
// cause est corrigee a la source (cf. round_bounds.go) ; ce controle est le filet qui interdit
// a une courbe non chronologique d'etre publiee en silence si une autre cause apparait.
//
// Le point ecarte est le point TARDIF-DANS-LA-LISTE mais PRECOCE-DANS-LE-TEMPS : il vient
// forcement d'une manche rangee apres celle dont il porte l'instant, donc c'est lui qui est mal
// range, pas ceux qui le precedent.
//
// L'ECART EST JOURNALISE ICI, une fois, avec le slot et le premier recul : c'est un defaut, pas
// un cas nominal, et il ne doit jamais etre avale. Le journal vit dans la fonction plutot que
// chez ses appelants pour que les DEUX cumuls (par slot ici, par joueur dans `analysis/replay`)
// le rendent de la meme facon, sans dupliquer ni le message ni la decision.
func ChronologicalTotal(pts []ScorePoint) []ScorePoint {
	out := make([]ScorePoint, 0, len(pts))
	last, dropped, recul := 0, 0, 0
	for i, p := range pts {
		if i > 0 && p.TimeMS < last {
			if dropped == 0 {
				recul = p.TimeMS
			}
			dropped++
			continue
		}
		out = append(out, p)
		last = p.TimeMS
	}
	if dropped > 0 {
		slog.Warn("objectiveevents: serie cumulee NON CHRONOLOGIQUE — points ecartes",
			"slot", pts[0].Slot, "ecartes", dropped, "retenus", len(out), "premierRecul", recul)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// sortedIntKeys rend les cles entieres d'un groupe de series, dans l'ordre. Elle sert aux
// MANCHES d'un slot (l'ordre du cumul en depend) et aux SLOTS d'entite d'un emplacement
// (depuis le lot 4b, l'ordre de consommation du budget d'evenements en depend : deux slots
// qui se disputent la fin du solde doivent le faire toujours dans le meme ordre, sinon la
// sortie d'un film tronque changerait a chaque execution).
//
// Anciennement `sortedRounds` : le nom disait la premiere des deux, ce qui aurait fait
// ecrire une seconde copie identique pour la seconde.
func sortedIntKeys(bySlot map[int][]ScorePoint) []int {
	out := make([]int, 0, len(bySlot))
	for r := range bySlot {
		out = append(out, r)
	}
	sort.Ints(out)
	return out
}

// sortedSlotKeys rend les emplacements d'une table, dans un ordre TOTAL (composant puis
// cote). Meme raison que ci-dessus : le budget d'evenements rend l'ordre de parcours
// OBSERVABLE sur un film tronque, et un parcours de map ne se rejoue pas a l'identique.
func sortedSlotKeys(table map[statSlotKey]statSlot) []statSlotKey {
	out := make([]statSlotKey, 0, len(table))
	for k := range table {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Comp != out[j].Comp {
			return out[i].Comp < out[j].Comp
		}
		return out[i].Side < out[j].Side
	})
	return out
}
