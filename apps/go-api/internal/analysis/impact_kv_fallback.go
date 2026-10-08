// Package analysis — impact_kv_fallback.go : la règle du repli des frags reconstitués d'une
// lecture des événements d'impact (Q32).
package analysis

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// ImpactMatchesNeedingKVFallback rend les matchs dont les frags et morts doivent être
// reconstitués depuis le journal des frags : ceux de chaque groupe dont AUCUNE ligne n'est un
// frag ou une mort (titre dont highlight_events ne porte que des médailles, comme Halo 5).
//
// Un groupe est l'ensemble de matchs qu'une lecture dédiée aurait lu seul : la décision se
// prend par groupe, jamais sur l'union, pour qu'une lecture de plusieurs groupes rende à chacun
// exactement ce que sa lecture dédiée lui aurait rendu. Les groupes sont disjoints.
func ImpactMatchesNeedingKVFallback(rows []domain.ImpactEventRow, groups [][]string) []string {
	withKill := make(map[string]bool)
	for _, r := range rows {
		switch canonical.HighlightEventType(r.EventType) {
		case canonical.EventKill, canonical.EventDeath:
			withKill[r.MatchID] = true
		}
	}
	var out []string
	for _, g := range groups {
		native := false
		for _, id := range g {
			native = native || withKill[id]
		}
		if !native {
			out = append(out, g...)
		}
	}
	return out
}
