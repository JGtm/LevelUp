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

// CorrectKillSourceRaws est l'équivalent de CorrectEventRaws pour les
// domain.KillSourceRaw (Q21b — l'arme du kill feed). Retourne une copie avec
// TimeMS ramené au référentiel gameplay (T0 retranché). Même signature UNITAIRE
// que CorrectEventRaws : Match View est strictement mono-match.
//
// RAISON D'ÊTRE : decorateKillFeed apparie sources et events sur la clé EXACTE
// (xuid, time_ms). Les events Q21 passent par CorrectEventRaws — si les sources
// restaient sur l'horloge film, AUCUNE clé ne coïnciderait sur un match à T0
// non nul et tout le feed perdrait arme et assistance (bug du 2026-08-12 :
// décalage constant de T0 ms entre les deux tranches, zéro appariement). La
// correction doit être LA MÊME fonction des deux côtés de la jointure.
//
// TimeMS est un int64 plein : copy() isole l'input (pas de pointeur partagé).
func CorrectKillSourceRaws(rows []domain.KillSourceRaw, tl domain.MatchTimeline) []domain.KillSourceRaw {
	if rows == nil {
		return nil
	}
	out := make([]domain.KillSourceRaw, len(rows))
	copy(out, rows)
	for i := range out {
		out[i].TimeMS = tl.CorrectEventTime(out[i].TimeMS)
	}
	return out
}

// CorrectKVPairRaws — même rôle que CorrectKillSourceRaws pour les
// domain.KVPairRaw (Q20 — la victime du kill feed), même raison d'être : la clé
// d'appariement (tueur, time_ms) doit vivre dans le même référentiel que les
// events corrigés. La correction s'applique à une COPIE réservée au feed : les
// consommateurs historiques des paires (tug-of-war, KD timeline) restent sur
// l'horloge brute, leur changer d'horloge changerait leurs bins.
func CorrectKVPairRaws(rows []domain.KVPairRaw, tl domain.MatchTimeline) []domain.KVPairRaw {
	if rows == nil {
		return nil
	}
	out := make([]domain.KVPairRaw, len(rows))
	copy(out, rows)
	for i := range out {
		out[i].TimeMS = tl.CorrectEventTime(out[i].TimeMS)
	}
	return out
}

// CorrectKillAssistRaws — même rôle que CorrectKillSourceRaws pour les
// domain.KillAssistRaw (Q21c — l'assistant du kill feed), même raison d'être :
// la clé d'appariement (xuid, time_ms) doit vivre dans le même référentiel que
// les events corrigés. Les champs pointeurs (gamertag, parts de dégâts) sont
// partagés avec l'input mais jamais modifiés ici.
func CorrectKillAssistRaws(rows []domain.KillAssistRaw, tl domain.MatchTimeline) []domain.KillAssistRaw {
	if rows == nil {
		return nil
	}
	out := make([]domain.KillAssistRaw, len(rows))
	copy(out, rows)
	for i := range out {
		out[i].TimeMS = tl.CorrectEventTime(out[i].TimeMS)
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
