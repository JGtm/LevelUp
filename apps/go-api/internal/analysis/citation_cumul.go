// Package analysis — citation_cumul.go : cumul d'une citation arrêté à un match.
// Fonction pure, sans accès DB.
package analysis

import (
	"sort"
	"time"
)

// CitationHistoryEntry est une ligne de l'historique d'un joueur (`match_citations_latest`)
// datée par le début de son match. Une StartTime zéro signifie « début inconnu ».
type CitationHistoryEntry struct {
	MatchID   string
	Norm      string
	Value     int
	StartTime time.Time
}

// CitationMatchKey désigne la citation `Norm` gagnée au match `MatchID`.
type CitationMatchKey struct {
	MatchID string
	Norm    string
}

// CitationCumulAfterMatch rend, pour chaque (match, citation) de l'historique, le total de la
// citation APRÈS ce match : la somme des valeurs des matchs joués jusqu'à lui inclus. C'est le
// `cumulative` qu'attend ComputeTierProgression (`cumulative - delta` = le total d'avant).
//
// Ordre d'une citation : début du match croissant ; à début égal, match_id croissant (départage
// déterministe) ; un match de début inconnu passe après tous les matchs datés, jamais avant
// (son cumul vaut alors le total de la citation, ce qui ne ressuscite aucune citation maîtrisée
// plus tôt). Un même (match, citation) répété voit ses valeurs sommées.
func CitationCumulAfterMatch(history []CitationHistoryEntry) map[CitationMatchKey]int {
	ordered := make([]CitationHistoryEntry, len(history))
	copy(ordered, history)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a.Norm != b.Norm {
			return a.Norm < b.Norm
		}
		if a.StartTime.IsZero() != b.StartTime.IsZero() {
			return !a.StartTime.IsZero()
		}
		if !a.StartTime.Equal(b.StartTime) {
			return a.StartTime.Before(b.StartTime)
		}
		return a.MatchID < b.MatchID
	})

	out := make(map[CitationMatchKey]int, len(ordered))
	running := 0
	for i, e := range ordered {
		if i == 0 || ordered[i-1].Norm != e.Norm {
			running = 0
		}
		running += e.Value
		out[CitationMatchKey{MatchID: e.MatchID, Norm: e.Norm}] = running
	}
	return out
}
