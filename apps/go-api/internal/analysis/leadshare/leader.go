// Package leadshare calcule la MAINMISE d'un match : la part de la durée jouée pendant laquelle
// un camp est EN TÊTE du score, avec le nombre de retournements.
//
// Calcul pur : aucune base, aucun HTTP, aucun import du décodeur de film. L'appelant
// (sync/replayartifacts) tire le score du document de rejeu et le passe ici sous la forme de
// paliers par camp.
//
// UNE SEULE DÉFINITION DU MENEUR, partagée avec le web. Le meneur est l'équipe SEULE en tête ;
// une égalité n'a pas de meneur ; la première prise de tête n'est pas un retournement ; un
// retournement est un changement de meneur comparé au dernier meneur CONNU (une égalité
// suspend le meneur, elle ne le ferme pas). Cette définition est celle de `leaderStates` et
// `leadChanges` (apps/web/src/lib/replay/scoreTimeline.ts) ; le jeu de cas
// testdata/meneur_cas.json est lu par les deux tests, et c'est lui qui les empêche de diverger.
package leadshare

import "sort"

// Tick est un palier de score : à partir de la frame Frame, la valeur est Value. Les paliers
// d'un camp sont croissants en Frame, et la valeur est TENUE jusqu'au palier suivant.
type Tick struct {
	Frame int
	Value int
}

// Team est la série cumulée du score d'UN camp, telle que le film la publie.
//
// ID nil = identité du camp non résolue : la série ne se rattache à aucun camp du registre et
// n'est jamais comparée ni nommée.
type Team struct {
	ID     *int
	Totals []Tick
}

// State est un état de la course au score : à partir de Frame, Leader mène, ou personne
// (Leader nil = égalité, mesurée et non inconnue).
type State struct {
	Frame  int
	Leader *int
}

// Change est un retournement : la frame où le nouveau meneur passe devant.
type Change struct {
	Frame  int
	Leader int
}

// ScoreAt rend la valeur du palier tenu à `frame` : le dernier palier au plus tard à cette
// frame, et 0 avant le premier (le compteur ne commence qu'au premier changement).
func ScoreAt(totals []Tick, frame int) int {
	i := sort.Search(len(totals), func(i int) bool { return totals[i].Frame > frame })
	if i == 0 {
		return 0
	}
	return totals[i-1].Value
}

// LeaderStates rend la suite des états de la course au score : un état par CHANGEMENT de
// meneur, égalités comprises. Le premier état est celui du premier palier publié ; avant lui le
// calque ne dit rien (0-0 est l'affaire de l'appelant, qui connaît la fenêtre jouée).
//
// campCount est le nombre de camps du roster du film (0 = inconnu). Quand TOUTES les séries
// publiées ont un camp, chaque camp absent concourt à zéro (un camp qui n'a jamais marqué
// n'émet rien) : il est compté, jamais nommé. Une seule série sans camp suffit à tout taire.
// Rend nil quand moins de deux camps concourent ou quand aucun palier n'existe.
func LeaderStates(teams []Team, campCount int) []State {
	named := namedTeams(teams)
	silent := 0
	if len(named) == len(teams) {
		silent = max(0, campCount-len(named))
	}
	if len(named) == 0 || len(named)+silent < 2 {
		return nil
	}
	frames := distinctFrames(named)
	out := make([]State, 0, len(frames))
	for _, frame := range frames {
		leader := leaderAt(named, frame, silent > 0)
		if len(out) > 0 && sameLeader(leader, out[len(out)-1].Leader) {
			continue
		}
		out = append(out, State{Frame: frame, Leader: leader})
	}
	return out
}

// ChangesWithin rend les retournements JOUÉS : ceux dont l'image tombe dans [StartMS, EndMS) de la
// fenêtre, au pas du document (même bornes que l'intégration des temps : début inclus, fin exclue).
// Le meneur d'avant la fenêtre sert de repère : un retournement qui défait un meneur tenu avant le
// coup d'envoi est joué s'il tombe dans la fenêtre. `Changes` reste la lecture du film entier.
func ChangesWithin(states []State, w Window, stepMS int64) []Change {
	var out []Change
	for _, c := range Changes(states) {
		at := int64(c.Frame) * stepMS
		if at >= w.StartMS && at < w.EndMS {
			out = append(out, c)
		}
	}
	return out
}

// Changes rend les retournements d'une suite d'états : un état à égalité ne compte pas et ne
// ferme pas le meneur précédent, la première prise de tête n'en est pas un.
func Changes(states []State) []Change {
	var out []Change
	var previous *int
	for _, s := range states {
		if s.Leader == nil {
			continue
		}
		if previous != nil && *s.Leader != *previous {
			out = append(out, Change{Frame: s.Frame, Leader: *s.Leader})
		}
		previous = s.Leader
	}
	return out
}

// namedTeams garde les séries dont le camp est résolu : une série sans camp ne peut être ni
// meneuse ni menée.
func namedTeams(teams []Team) []Team {
	out := make([]Team, 0, len(teams))
	for _, t := range teams {
		if t.ID != nil {
			out = append(out, t)
		}
	}
	return out
}

// distinctFrames rend les frames de palier de toutes les séries, triées, sans doublon.
func distinctFrames(teams []Team) []int {
	seen := map[int]struct{}{}
	var frames []int
	for _, t := range teams {
		for _, p := range t.Totals {
			if _, ok := seen[p.Frame]; ok {
				continue
			}
			seen[p.Frame] = struct{}{}
			frames = append(frames, p.Frame)
		}
	}
	sort.Ints(frames)
	return frames
}

// leaderAt rend le camp SEUL en tête à `frame`, ou nil (égalité). silent : au moins un camp sans
// série concourt à zéro, un camp publié à 0 est alors à égalité avec lui et seul un score
// strictement positif mène.
func leaderAt(teams []Team, frame int, silent bool) *int {
	best := -1
	tied := false
	if silent {
		best, tied = 0, true
	}
	var bestTeam *int
	for _, t := range teams {
		v := ScoreAt(t.Totals, frame)
		switch {
		case v > best:
			best, bestTeam, tied = v, t.ID, false
		case v == best:
			tied = true
		}
	}
	if tied {
		return nil
	}
	return bestTeam
}

func sameLeader(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
