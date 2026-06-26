// Package analysis — kv_synthetic_events.go : synthèse d'events kill/death
// canoniques depuis les paires killer→victim (killer_victim_pairs).
//
// Contexte (title-agnostic) : certains titres ne portent PAS les kills dans
// `shared.highlight_events`. En Halo 5 par exemple, highlight_events ne contient
// que des médailles, tandis que les kills horodatés vivent dans
// `killer_victim_pairs` (avec un time_ms relatif au match). Les consommateurs
// qui dérivent cadence / rôles / impact badges / courbe d'engagement depuis les
// highlight_events kill/death se retrouvent alors vides.
//
// Ce helper reconstruit, pour chaque paire, deux events canoniques :
//   - {EventType: "kill",  TimeMS, XUID/KillerXUID = killer}
//   - {EventType: "death", TimeMS, XUID/VictimXUID = victim}
//
// Sémantique du champ XUID alignée sur la convention historique de
// shared.highlight_events (XUID = tueur pour un kill, victime pour un death) et
// sur convertEventsRawToCanonical (service). Les pointeurs KillerXUID/VictimXUID
// sont aussi posés pour les consommateurs canonical-aware.
//
// Sur Halo Infinite (qui porte ses kills dans highlight_events) ce chemin n'est
// JAMAIS emprunté : les callers ne synthétisent que lorsque les highlight_events
// ne contiennent aucun kill/death. NO-OP par construction sur Infinite.
package analysis

import (
	"sort"

	"levelup/go-api/internal/games/canonical"
)

// KVSyntheticInput est la forme minimale d'une paire killer→victim consommée
// par SynthesizeKillEventsFromKVPairs. Découplée de domain.KVPairRaw pour que
// le package analysis reste sans dépendance vers domain (et réutilisable par
// les loaders qui scannent directement la DB).
type KVSyntheticInput struct {
	KillerXUID string
	VictimXUID string
	TimeMS     int64
	// KillCount : nombre de kills agrégés sur cette paire à ce timestamp.
	// 0 ou négatif est traité comme 1 (une paire = au moins un kill).
	KillCount int
}

// HasCanonicalKillOrDeath indique si la liste d'events canoniques contient au
// moins un event kill ou death. Sert aux loaders à décider s'il faut activer le
// fallback synthétique (kvPairs → events) : si des kills sont déjà présents
// (cas Infinite), on ne synthétise rien.
func HasCanonicalKillOrDeath(events []canonical.HighlightEvent) bool {
	for _, e := range events {
		switch canonical.HighlightEventType(e.EventType) {
		case canonical.EventKill, canonical.EventDeath:
			return true
		}
	}
	return false
}

// SynthesizeKillEventsFromKVPairs construit des events canoniques kill/death à
// partir des paires killer→victim. Pour chaque paire (et chaque kill agrégé via
// KillCount), émet un event kill (acteur = tueur) et un event death (acteur =
// victime), tous deux au même TimeMS. matchID est rattaché à chaque event.
//
// Les events sont triés par TimeMS croissant (cohérent avec le ORDER BY time_ms
// des loaders highlight_events).
//
// Retourne nil si aucune paire exploitable (killer/victim vides ignorés).
func SynthesizeKillEventsFromKVPairs(pairs []KVSyntheticInput, matchID string) []canonical.HighlightEvent {
	if len(pairs) == 0 {
		return nil
	}
	out := make([]canonical.HighlightEvent, 0, len(pairs)*2)
	for _, kv := range pairs {
		if kv.KillerXUID == "" || kv.VictimXUID == "" {
			continue
		}
		n := kv.KillCount
		if n <= 0 {
			n = 1
		}
		killer := kv.KillerXUID
		victim := kv.VictimXUID
		for i := 0; i < n; i++ {
			out = append(out, canonical.HighlightEvent{
				MatchID:    matchID,
				EventType:  string(canonical.EventKill),
				TimeMS:     kv.TimeMS,
				XUID:       killer,
				KillerXUID: &killer,
			})
			out = append(out, canonical.HighlightEvent{
				MatchID:    matchID,
				EventType:  string(canonical.EventDeath),
				TimeMS:     kv.TimeMS,
				XUID:       victim,
				VictimXUID: &victim,
			})
		}
	}
	if len(out) == 0 {
		return nil
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TimeMS < out[j].TimeMS })
	return out
}

// MergeAndSortCanonicalEvents fusionne deux listes d'events canoniques (ex.
// médailles existantes + kill/death synthétiques) et les trie par TimeMS
// croissant. Préserve l'ordre relatif des events de même timestamp (stable).
func MergeAndSortCanonicalEvents(a, b []canonical.HighlightEvent) []canonical.HighlightEvent {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	out := make([]canonical.HighlightEvent, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].TimeMS < out[j].TimeMS })
	return out
}
