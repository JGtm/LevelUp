package timeline

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// CorrectEvents retourne une copie des events avec TimeMS ramené au référentiel
// gameplay (soustraction du T0 du match correspondant).
//
// C'est le POINT UNIQUE d'application de la correction T0 sur les events. Les
// fonctions analysis (first_events, intensity, cadence, match_impact) restent
// agnostiques : elles reçoivent des events déjà corrigés.
//
// timelines : T0 par match_id. Un match_id absent (ou MatchTimeline zéro-value)
// donne T0=0, donc TimeMS inchangé — c'est le comportement de fallback Phase 1
// et celui des matchs sans T0 connu.
//
// L'identité des "gagnants" de badges (first_blood, top_gun, etc.) est invariante
// par cette correction : soustraire une constante par match préserve l'ordre
// relatif et les écarts inter-events. Seuls les TimeMS affichés se décalent.
//
// Les events ne sont pas mutés en place : une nouvelle slice est retournée.
func CorrectEvents(
	events []canonical.HighlightEvent,
	timelines map[string]domain.MatchTimeline,
) []canonical.HighlightEvent {
	if events == nil {
		return nil
	}
	out := make([]canonical.HighlightEvent, len(events))
	copy(out, events)
	for i := range out {
		tl := timelines[out[i].MatchID] // zéro-value {0,0} si absent → identité
		out[i].TimeMS = tl.CorrectEventTime(out[i].TimeMS)
	}
	return out
}
