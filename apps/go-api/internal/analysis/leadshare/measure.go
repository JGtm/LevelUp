package leadshare

import "sort"

// Reason est la raison STABLE d'un refus de mesure. Une valeur de donnée (écrite en base,
// comptée, imprimée), jamais un libellé d'affichage.
type Reason string

const (
	// ReasonNone : la mesure est faite.
	ReasonNone Reason = ""
	// ReasonModeEvents : le mode marque peu de points (drapeau, colline, bombe) ; la part du
	// temps en tête y dirait surtout la date de la première prise.
	ReasonModeEvents Reason = "mode_events"
	// ReasonScoreMissing : le document ne porte aucun calque de score.
	ReasonScoreMissing Reason = "score_missing"
	// ReasonScoreNotCovered : le film ne réplique pas le score du mode, ou aucun palier n'existe.
	ReasonScoreNotCovered Reason = "score_not_covered"
	// ReasonScoreTruncated : la lecture du score s'est arrêtée avant la fin du match.
	ReasonScoreTruncated Reason = "score_truncated"
	// ReasonClockOriginMissing : l'origine du film n'est pas établie ; une série datée sans elle
	// est décalée d'un écart inconnu.
	ReasonClockOriginMissing Reason = "clock_origin_missing"
	// ReasonClockStepMissing : le document ne dit pas la durée d'une image.
	ReasonClockStepMissing Reason = "clock_step_missing"
	// ReasonKickoffUnmeasured : le coup d'envoi n'est pas mesuré dans le film.
	ReasonKickoffUnmeasured Reason = "kickoff_unmeasured"
	// ReasonDurationUnknown : le registre ne donne pas la durée du match.
	ReasonDurationUnknown Reason = "duration_unknown"
	// ReasonWindowIncoherent : la fenêtre jouée est vide ou inversée.
	ReasonWindowIncoherent Reason = "window_incoherent"
	// ReasonTeamUnresolved : une série du score n'a pas de camp, ou deux séries portent le même.
	ReasonTeamUnresolved Reason = "team_unresolved"
	// ReasonSingleCamp : moins de deux camps concourent, rien ne se compare.
	ReasonSingleCamp Reason = "single_camp"
)

// Window est la partie JOUÉE du match, en millisecondes sur l'axe du film (depuis l'image zéro).
type Window struct {
	StartMS int64
	EndMS   int64
}

// Input est tout ce que la mesure lit.
type Input struct {
	// Teams : les séries cumulées du score, une par camp publié.
	Teams []Team
	// CampCount : camps du roster du film (0 = inconnu), pour compter les camps muets.
	CampCount int
	// SilentCampIDs : identifiants des camps du roster sans série publiée, quand l'appelant sait les
	// nommer. Ils ne sont retenus (Result.SilentIDs) que s'ils sont EXACTEMENT les camps muets : autant
	// que SilentCamps, aucun doublon, aucun qui porte une série. Sinon la mesure se fait quand même,
	// mais aucun camp muet n'est nommé.
	SilentCampIDs []int
	// FrameIntervalMS : durée d'une image du document ; une frame vaut Frame x FrameIntervalMS.
	FrameIntervalMS int
	// Window : la partie jouée, sur le même axe que les frames.
	Window Window
}

// CampShare est le temps passé SEUL en tête par un camp.
type CampShare struct {
	Camp   int
	LeadMS int64
}

// Result est la mesure d'un match. Invariant : la somme des LeadMS et de TieMS vaut MeasuredMS.
type Result struct {
	// Camps : un camp par série publiée, triés par identifiant, zéro compris.
	Camps []CampShare
	// TieMS : temps où personne n'est seul en tête (égalité, y compris avant le premier point).
	TieMS int64
	// MeasuredMS : durée de la fenêtre jouée.
	MeasuredMS int64
	// LeadChanges : retournements de la fenêtre jouée (ChangesWithin) : ceux d'avant le coup d'envoi
	// ou d'après la fin ne sont pas joués. `leadChanges` côté web (frise, courbe) reste la lecture du
	// film entier, et ne sert pas ce nombre.
	LeadChanges int
	// SilentCamps : camps du roster sans série publiée. Ils concourent à zéro et ne mènent jamais.
	SilentCamps int
	// SilentIDs : leurs identifiants, triés, quand l'appelant les a dits et qu'ils coïncident avec les
	// camps muets (cf. Input.SilentCampIDs) ; nil = non nommés. Un camp nommé ici est un camp du match
	// à part nulle MESURÉE ; un camp qui n'y est pas et n'a pas de série n'est pas lisible.
	SilentIDs []int
}

// Measure rend le temps en tête de chaque camp sur la fenêtre jouée, ou la raison du refus.
// Un refus n'écrit rien : jamais un zéro qui se lirait comme une mesure.
func Measure(in Input) (Result, Reason) {
	if in.FrameIntervalMS <= 0 {
		return Result{}, ReasonClockStepMissing
	}
	if in.Window.EndMS <= in.Window.StartMS {
		return Result{}, ReasonWindowIncoherent
	}
	camps, reason := campsOf(in.Teams)
	if reason != ReasonNone {
		return Result{}, reason
	}
	if !anyTick(in.Teams) {
		return Result{}, ReasonScoreNotCovered
	}
	states := LeaderStates(in.Teams, in.CampCount)
	if states == nil {
		return Result{}, ReasonSingleCamp
	}
	silent := max(0, in.CampCount-len(in.Teams))
	res := Result{
		Camps:       camps,
		MeasuredMS:  in.Window.EndMS - in.Window.StartMS,
		LeadChanges: len(ChangesWithin(states, in.Window, int64(in.FrameIntervalMS))),
		SilentCamps: silent,
		SilentIDs:   namedSilent(in.SilentCampIDs, camps, silent),
	}
	integrate(&res, states, in)
	return res, ReasonNone
}

// campsOf rend les camps publiés, triés, ou un refus quand une série n'a pas de camp ou que
// deux séries portent le même.
func campsOf(teams []Team) ([]CampShare, Reason) {
	camps := make([]CampShare, 0, len(teams))
	seen := map[int]struct{}{}
	for _, t := range teams {
		if t.ID == nil {
			return nil, ReasonTeamUnresolved
		}
		if _, dup := seen[*t.ID]; dup {
			return nil, ReasonTeamUnresolved
		}
		seen[*t.ID] = struct{}{}
		camps = append(camps, CampShare{Camp: *t.ID})
	}
	sort.Slice(camps, func(i, j int) bool { return camps[i].Camp < camps[j].Camp })
	return camps, ReasonNone
}

// namedSilent rend les identifiants des camps muets, triés, ou nil quand la liste de l'appelant ne
// coïncide pas avec eux : autant que de camps muets, sans doublon, sans camp qui porte une série.
func namedSilent(ids []int, published []CampShare, silent int) []int {
	if silent == 0 || len(ids) != silent {
		return nil
	}
	taken := make(map[int]struct{}, len(published)+len(ids))
	for _, c := range published {
		taken[c.Camp] = struct{}{}
	}
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if _, dup := taken[id]; dup {
			return nil
		}
		taken[id] = struct{}{}
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}

func anyTick(teams []Team) bool {
	for _, t := range teams {
		if len(t.Totals) > 0 {
			return true
		}
	}
	return false
}

// integrate additionne, segment par segment de la fenêtre, le temps de chaque état. Avant le
// premier état le score est nul des deux côtés : égalité.
func integrate(res *Result, states []State, in Input) {
	step := int64(in.FrameIntervalMS)
	var held *int // meneur tenu au début du segment courant
	cursor := in.Window.StartMS
	credit := func(until int64) {
		if until <= cursor {
			return
		}
		d := until - cursor
		cursor = until
		if held == nil {
			res.TieMS += d
			return
		}
		for i := range res.Camps {
			if res.Camps[i].Camp == *held {
				res.Camps[i].LeadMS += d
			}
		}
	}
	for _, s := range states {
		at := int64(s.Frame) * step
		if at >= in.Window.EndMS {
			break
		}
		credit(at) // no-op tant que l'état précède le début de la fenêtre
		held = s.Leader
	}
	credit(in.Window.EndMS)
}
