package objectiveevents

import (
	"sort"
	"strings"
)

// awards.go — ETIQUETER les increments de score personnel : donner un NOM a chaque
// evenement date du film.
//
// # Pourquoi ce n'est pas une inference
//
// Le depot possede deja le catalogue nomme des recompenses de score : la table
// `personal_score_awards` des bases joueur porte, par match et par joueur, le nom de la
// recompense (`killed_player`, `flag_returned`, `zone_secured`...), sa categorie et son
// score. Le bareme unitaire s'en deduit (`award_score / award_count`) :
//
//	300 : flag_captured
//	100 : killed_player · flag_capture_assist · hill_scored · destroyed_scorpion/wraith
//	 75 : zone_captured (variable, vaut aussi 50)
//	 50 : kill_assist · ball_control · emp_assist · driver_assist · vehicules legers
//	 25 : flag_stolen · flag_returned · zone_secured · hill_control · runner_stopped · hijacked_*
//	 10 : flag_taken · ball_taken · sensor_assist · mark_assist
//	-100 : self_destruction · betrayed_player
//
// Le film, lui, apporte ce que l'API n'a pas : **l'INSTANT**. L'etiquetage consiste donc a
// rapprocher les deux — la valeur de l'increment et le QUOTA exact du joueur sur ce match.
//
// # Ce qui est resolu, et ce qui ne l'est pas
//
// La resolution se fait a la VALEUR, ce qui est exact et sans arbitrage : un increment de
// 300 ne peut etre qu'une capture de drapeau. Quand plusieurs recompenses du quota
// partagent la meme valeur (six noms possibles a 25 points), l'evenement porte la LISTE des
// candidates plutot qu'un nom choisi au hasard — et les quotas disent combien de chaque.
// Aucune heuristique ne tranche a la place de la mesure.
//
// # Increments composes
//
// Plusieurs actions tombant dans le meme paquet se somment (125 = 100 + 25, mesure en CTF).
// La decomposition est recherchee dans les seules valeurs du quota du joueur, et n'est
// retenue que si elle est UNIQUE a nombre de parts minimal — sinon l'increment reste brut,
// signale comme non decompose.
//
// # Reconciliation, sur un match reel
//
// JGtm, film `1bc77d2e` (CTF), slot 18 : le film porte 22 x (+100), 6 x (+25), 2 x (+50),
// 2 x (+125), 1 x (+300), 1 x (+10). L'API compte 24 `killed_player`, 1 `flag_captured`,
// 2 `kill_assist`, 4 `flag_stolen` + 2 `flag_returned` + 2 `runner_stopped`, 1 `flag_taken`.
// Une fois les +125 decomposes, les deux comptes coincident A L'UNITE, et les sommes
// valent 3 010 des deux cotes.

// maxCompoundParts borne la recherche de decomposition d'un increment compose. Trois parts
// couvrent tout ce qui a ete observe (le maximum mesure est deux) et bornent le cout.
const maxCompoundParts = 3

// Award decrit une recompense de score telle que `personal_score_awards` la donne pour un
// joueur sur un match.
type Award struct {
	// Name est le nom canonique de la recompense (`award_name`).
	Name string
	// Category est sa famille (`award_category`) : kill, assist, objective, vehicle, penalty.
	Category string
	// Unit est le score d'UNE occurrence (`award_score / award_count`).
	Unit int64
	// Count est le nombre d'occurrences sur le match.
	Count int
}

// LabelledEvent est un evenement de score du film, date et nomme.
type LabelledEvent struct {
	// TimeMS est l'instant sur l'horloge du film, meme base que les ObjectiveEvent.
	TimeMS int
	// Slot identifie l'entite (cf. IsTeamSlot).
	Slot int
	// Value est le score de cette action.
	Value int64
	// Award est le nom canonique quand une seule recompense du quota porte cette valeur ;
	// vide sinon.
	Award string
	// Category est la categorie de la recompense nommee, ou la categorie commune aux
	// candidates quand elles la partagent.
	Category string
	// Candidates liste les recompenses du quota compatibles quand Award est vide.
	Candidates []string
	// Compound dit que cet evenement provient de la decomposition d'un increment
	// transportant plusieurs actions dans le meme paquet.
	Compound bool
}

// Resolved dit si l'evenement porte un nom unique.
func (e LabelledEvent) Resolved() bool { return e.Award != "" }

// LabelPersonalScore etiquette les increments de score personnel d'un film.
//
// points vient de la courbe de score personnel (composant 1, valeur B) ; quotas donne, par slot d'entite, les recompenses
// que l'API attribue au joueur de ce slot sur ce match (la correspondance slot -> joueur
// est du ressort de l'appelant). Un slot sans quota est ignore : sans quota il n'y a pas de
// nommage possible, et inventer un nom serait pire que se taire.
func LabelPersonalScore(points []ScorePoint, quotas map[int][]Award) []LabelledEvent {
	bySlot := map[int][]ScorePoint{}
	for _, p := range points {
		if _, ok := quotas[p.Slot]; !ok {
			continue
		}
		bySlot[p.Slot] = append(bySlot[p.Slot], p)
	}
	// Un point de score produit au plus quelques evenements (un increment compose se
	// decompose en 2 ou 3 parts), et souvent aucun (valeur inchangee). Le nombre de points
	// retenus est donc l'ordre de grandeur juste pour la reservation : ni sous-dimensionne
	// au point de recopier a chaque tour, ni gonfle par les slots sans quota.
	retenus := 0
	for _, ps := range bySlot {
		retenus += len(ps)
	}
	out := make([]LabelledEvent, 0, retenus)
	for slot, ps := range bySlot {
		sort.SliceStable(ps, func(i, j int) bool { return ps[i].TimeMS < ps[j].TimeMS })
		out = append(out, labelSlot(slot, ps, quotas[slot])...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TimeMS != out[j].TimeMS {
			return out[i].TimeMS < out[j].TimeMS
		}
		return out[i].Slot < out[j].Slot
	})
	return out
}

// labelSlot etiquette la suite d'un slot : increments successifs du score personnel.
//
// LA PREMIERE LECTURE EST UN INCREMENT DEPUIS ZERO, et c'est exact : un score personnel
// part de 0 au coup d'envoi, la premiere valeur observee est donc bien ce que le joueur
// vient de gagner. La reconciliation JGtm le confirme a l'unite sur un match reel.
//
// (Un garde `if first` vivait ici jusqu'au 2026-08-05, constat D-P2 de la revue du
// 2026-08-02. Son commentaire annoncait de traiter la premiere lecture a part — le
// CONTRAIRE de ce que la fonction fait —, et sa condition ne pouvait pas s'en charger :
// `d == p.Value` est TOUJOURS vrai a la premiere iteration puisque `prev` vaut 0, si bien
// que le garde se reduisait au `d == 0` teste deux lignes plus bas. Il ne faisait rien.
// Retire avec sa variable ; le comportement, lui, est desormais epingle par
// TestLabelPersonalScorePremiereLectureEstUnIncrementDepuisZero.)
func labelSlot(slot int, ps []ScorePoint, awards []Award) []LabelledEvent {
	units := unitIndex(awards)
	var out []LabelledEvent
	prev := int64(0)
	for _, p := range ps {
		d := p.Value - prev
		prev = p.Value
		if d == 0 {
			continue
		}
		parts, compound := decompose(d, units)
		for _, u := range parts {
			out = append(out, buildEvent(p.TimeMS, slot, u, units[u], compound))
		}
	}
	return out
}

// unitIndex regroupe les recompenses du quota par valeur unitaire.
func unitIndex(awards []Award) map[int64][]Award {
	out := map[int64][]Award{}
	for _, a := range awards {
		if a.Count <= 0 {
			continue
		}
		out[a.Unit] = append(out[a.Unit], a)
	}
	for u := range out {
		sort.Slice(out[u], func(i, j int) bool { return out[u][i].Name < out[u][j].Name })
	}
	return out
}

// buildEvent construit l'evenement nomme pour une valeur unitaire donnee.
func buildEvent(timeMS, slot int, unit int64, cands []Award, compound bool) LabelledEvent {
	e := LabelledEvent{TimeMS: timeMS, Slot: slot, Value: unit, Compound: compound}
	switch len(cands) {
	case 0:
		return e
	case 1:
		e.Award, e.Category = cands[0].Name, cands[0].Category
		return e
	}
	cat := cands[0].Category
	for _, c := range cands {
		e.Candidates = append(e.Candidates, c.Name)
		if c.Category != cat {
			cat = ""
		}
	}
	e.Category = cat
	return e
}

// decompose rend les valeurs unitaires dont la somme vaut d. Une decomposition n'est
// retenue que si elle est UNIQUE a nombre de parts minimal ; sinon l'increment est rendu
// tel quel (une seule part valant d), ce que l'appelant verra comme non nomme.
func decompose(d int64, units map[int64][]Award) ([]int64, bool) {
	if _, ok := units[d]; ok {
		return []int64{d}, false
	}
	var us []int64
	for u := range units {
		us = append(us, u)
	}
	sort.Slice(us, func(i, j int) bool { return us[i] > us[j] })
	for n := 2; n <= maxCompoundParts; n++ {
		sols := combinations(d, us, n)
		if len(sols) == 1 {
			return sols[0], true
		}
		if len(sols) > 1 {
			break
		}
	}
	return []int64{d}, false
}

// combinations enumere les multisets de n valeurs de us dont la somme vaut d.
func combinations(d int64, us []int64, n int) [][]int64 {
	var out [][]int64
	var walk func(start int, left int, rest int64, acc []int64)
	walk = func(start, left int, rest int64, acc []int64) {
		if left == 0 {
			if rest == 0 {
				out = append(out, append([]int64(nil), acc...))
			}
			return
		}
		for i := start; i < len(us); i++ {
			walk(i, left-1, rest-us[i], append(acc, us[i]))
		}
	}
	walk(0, n, d, nil)
	return out
}

// LabelSummary resume un etiquetage : de quoi juger sur pieces plutot que sur impression.
type LabelSummary struct {
	// Total est le nombre d'evenements produits.
	Total int
	// Resolved est le nombre d'evenements portant un nom unique.
	Resolved int
	// Compound est le nombre d'evenements issus d'un increment compose.
	Compound int
	// Unnamed est le nombre d'evenements sans nom ni candidate (valeur hors quota).
	Unnamed int
	// ByAward compte les evenements par nom resolu.
	ByAward map[string]int
}

// SummarizeLabels calcule le resume d'un etiquetage.
func SummarizeLabels(events []LabelledEvent) LabelSummary {
	s := LabelSummary{ByAward: map[string]int{}}
	for _, e := range events {
		s.Total++
		if e.Compound {
			s.Compound++
		}
		switch {
		case e.Resolved():
			s.Resolved++
			s.ByAward[e.Award]++
		case len(e.Candidates) == 0:
			s.Unnamed++
		}
	}
	return s
}

// Describe rend une description lisible d'un evenement : son nom, ou la liste des
// candidates quand la valeur ne suffit pas a trancher.
func (e LabelledEvent) Describe() string {
	if e.Resolved() {
		return e.Award
	}
	if len(e.Candidates) == 0 {
		return "inconnu"
	}
	return "l'un de : " + strings.Join(e.Candidates, ", ")
}
