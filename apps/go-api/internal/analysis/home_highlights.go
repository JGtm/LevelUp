// Package analysis â€” home_highlights.go : sÃ©lection de la fenÃªtre de matchs
// pertinente et moteur principal des faits marquants (BuildHighlights).
//
// Les tuiles Ã  dÃ©filement (MaÃ®trise, Stats par min., SÃ©rie) sont dans
// home_highlights_tiles.go pour garder ce fichier sous 500 lignes.
package analysis

import (
	"levelup/go-api/internal/legacymatch"
)

// maxSessionlessHighlights borne le fallback « aucun match avec session_label » aux N
// premiers matchs, pour ne pas renvoyer un historique entier non sessionnisé. Partagé
// par BuildHighlights (legacy) et la variante canonique (home_canonical_highlights.go).
const maxSessionlessHighlights = 50

// ---------------------------------------------------------------------------
// selectHighlightWindow â€” fenÃªtre de sessions similaires
// ---------------------------------------------------------------------------

// selectHighlightWindow sÃ©lectionne les matchs de la derniÃ¨re session et des
// 4 sessions les plus rÃ©centes ayant la mÃªme composition (IsWithFriends) et
// la mÃªme playlist dominante (SkillPlaylistGroup).
// Fallback : si moins de 5 sessions similaires existent, toutes les sessions
// disponibles sont retournÃ©es. Les matchs sans SessionLabel ne font pas partie
// de la fenÃªtre calculÃ©e.
func selectHighlightWindow(matches []legacymatch.HomeMatchRow) []legacymatch.HomeMatchRow {
	if len(matches) == 0 {
		return nil
	}

	type sessionEntry struct {
		label         string
		isWithFriends bool
		playlistGroup string
		indices       []int
	}

	sessionOrder := []string{}
	sessionMap := map[string]*sessionEntry{}

	for i, m := range matches {
		if m.SessionLabel == nil {
			continue
		}
		lbl := *m.SessionLabel
		if _, exists := sessionMap[lbl]; !exists {
			sessionMap[lbl] = &sessionEntry{
				label:         lbl,
				isWithFriends: m.IsWithFriends,
			}
			sessionOrder = append(sessionOrder, lbl)
		}
		sessionMap[lbl].indices = append(sessionMap[lbl].indices, i)
	}

	if len(sessionOrder) == 0 {
		// Aucun match avec session_label : fallback sur les 50 premiers.
		if len(matches) > maxSessionlessHighlights {
			return matches[:maxSessionlessHighlights]
		}
		return matches
	}

	// Calculer la playlist dominante de chaque session.
	for _, lbl := range sessionOrder {
		entry := sessionMap[lbl]
		freq := map[string]int{}
		for _, idx := range entry.indices {
			if matches[idx].SkillPlaylistGroup != nil && *matches[idx].SkillPlaylistGroup != "" {
				freq[*matches[idx].SkillPlaylistGroup]++
			}
		}
		best, bestCount := "", 0
		for pg, cnt := range freq {
			if cnt > bestCount {
				bestCount = cnt
				best = pg
			}
		}
		entry.playlistGroup = best
	}

	// Session de rÃ©fÃ©rence = la plus rÃ©cente (sessionOrder[0]).
	ref := sessionMap[sessionOrder[0]]

	// Collecter jusqu'Ã  5 sessions similaires (mÃªme composition + mÃªme playlist).
	collected := []string{}
	for _, lbl := range sessionOrder {
		e := sessionMap[lbl]
		if e.isWithFriends == ref.isWithFriends && e.playlistGroup == ref.playlistGroup {
			collected = append(collected, lbl)
			if len(collected) >= 5 {
				break
			}
		}
	}

	labelSet := map[string]bool{}
	for _, lbl := range collected {
		labelSet[lbl] = true
	}

	var window []legacymatch.HomeMatchRow
	for _, m := range matches {
		if m.SessionLabel != nil && labelSet[*m.SessionLabel] {
			window = append(window, m)
		}
	}
	return window
}

// ---------------------------------------------------------------------------
// Helpers couleur et sÃ©lection
// ---------------------------------------------------------------------------

// highlightPerfColor retourne le niveau de couleur d'un score de performance.
// Les seuils sont identiques Ã  ceux de perf-color.ts cÃ´tÃ© frontend.
func highlightPerfColor(perf float64) string {
	switch {
	case perf >= 80:
		return "perf-excellent"
	case perf >= 65:
		return "perf-good"
	case perf >= 50:
		return "perf-ok"
	case perf >= 35:
		return "perf-low"
	default:
		return "perf-bad"
	}
}

// highlightKDAColor retourne la couleur sÃ©mantique d'un FDA/KDA.
func highlightKDAColor(kda float64) string {
	switch {
	case kda > 1:
		return homeColorPositive
	case kda >= 0:
		return homeColorNeutral
	default:
		return homeColorNegative
	}
}
