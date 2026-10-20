package service

// tactical_service_cellule_horloge.go — l'instant que l'horloge AFFICHEE du rejeu lira à
// l'événement d'une contribution (`TacticalContribution.InstantAfficheMs`).
//
// L'horloge affichée du rejeu part du coup d'envoi et ne descend pas sous 0:00 : sur l'axe du
// match, `affiché = max(0, instant − T0)`. L'origine du film s'annule dans la soustraction, d'où
// l'absence de lecture d'artefact. Le T0 vient du registre (`TacticalMatch.T0Ms`, déjà lu avec
// l'univers) : le coup d'envoi mesuré dans le film quand il a été reporté, l'estimation de l'API
// sinon — la même priorité que la fenêtre du rejeu.

import "levelup/go-api/internal/domain"

// instantAffiche rend l'instant affiché d'une contribution à horloge « match », ou nil quand le T0
// du match est inconnu ou que l'horloge est celle du film.
func instantAffiche(clock string, instantMs int64, t0 *int64) *int64 {
	if clock != domain.TacticalClockMatch || t0 == nil {
		return nil
	}
	v := max(instantMs-*t0, 0)
	return &v
}

// poserInstantAffiche renseigne l'instant affiché de chaque contribution retenue, d'après le T0 de
// son match dans l'univers de la lecture.
func poserInstantAffiche(retenues []contributionLue, univers domain.TacticalUnivers) {
	t0 := make(map[string]*int64, len(univers.Matchs))
	for _, m := range univers.Matchs {
		t0[m.MatchID] = m.T0Ms
	}
	for i := range retenues {
		retenues[i].InstantAfficheMs = instantAffiche(retenues[i].Clock, retenues[i].InstantMs, t0[retenues[i].MatchID])
	}
}
