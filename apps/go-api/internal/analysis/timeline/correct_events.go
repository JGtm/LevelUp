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

// CorrectEventRaws est l'équivalent de CorrectEvents pour les domain.EventRaw
// (events bruts Q21 consommés par Match View : kill-feed/evtList + badges
// d'impact). Retourne une copie avec TimeMS ramené au référentiel gameplay (T0
// retranché). Signature UNITAIRE (tl du match unique) et non map match_id→tl :
// Match View est strictement mono-match, contrairement à CorrectEvents /
// CorrectImpactEvents qui agrègent plusieurs matchs (Timeseries / Escouade).
//
// Pièges respectés :
//   - EventRaw.TimeMS est un *int64 : copy() duplique le POINTEUR, pas l'int64
//     pointé. On réalloue (v := ...; out[i].TimeMS = &v) pour NE PAS muter
//     l'input partagé (un double appel re-soustrairait T0 sinon).
//   - TimeMS nil est préservé tel quel.
//   - Un TimeMS corrigé peut être négatif si l'event précède T0 (countdown) —
//     au caller de filtrer (cf. evtList / buildImpactInput skip < 0).
func CorrectEventRaws(events []domain.EventRaw, tl domain.MatchTimeline) []domain.EventRaw {
	if events == nil {
		return nil
	}
	out := make([]domain.EventRaw, len(events))
	copy(out, events)
	for i := range out {
		if out[i].TimeMS == nil {
			continue
		}
		v := tl.CorrectEventTime(*out[i].TimeMS)
		out[i].TimeMS = &v
	}
	return out
}

// CorrectImpactEvents est l'équivalent de CorrectEvents pour les
// domain.ImpactEventRow (page Escouade / TeammatesService, type Q32 distinct de
// canonical.HighlightEvent). Retourne une copie avec TimeMS ramené au
// référentiel gameplay (T0 du match retranché). Mêmes propriétés que
// CorrectEvents : match absent / timelines nil → identité (T0=0) ; l'identité
// des gagnants de badges est invariante (soustraction d'une constante par
// match) ; l'input n'est pas muté. Un TimeMS corrigé peut être négatif si
// l'event précède T0 (countdown) — au caller de filtrer.
func CorrectImpactEvents(
	events []domain.ImpactEventRow,
	timelines map[string]domain.MatchTimeline,
) []domain.ImpactEventRow {
	if events == nil {
		return nil
	}
	out := make([]domain.ImpactEventRow, len(events))
	copy(out, events)
	for i := range out {
		tl := timelines[out[i].MatchID] // zéro-value {0,0} si absent → identité
		out[i].TimeMS = tl.CorrectEventTime(out[i].TimeMS)
	}
	return out
}
