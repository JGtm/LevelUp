package analysis

import (
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
)

// TestImpactMatchesNeedingKVFallback_ParGroupe : un groupe sans frag ni mort natif est
// reconstitué même quand un autre groupe de la même lecture en porte ; un groupe dont un seul
// match porte un frag natif ne l'est pas (décision de sa lecture dédiée).
func TestImpactMatchesNeedingKVFallback_ParGroupe(t *testing.T) {
	rows := []domain.ImpactEventRow{
		{MatchID: "p1", EventType: "medal"},
		{MatchID: "e1", EventType: "kill"},
		{MatchID: "f1", EventType: "death"},
	}
	groups := [][]string{{"p1", "p2"}, {"e1"}, {"f1", "f2"}}
	got := fmt.Sprint(ImpactMatchesNeedingKVFallback(rows, groups))
	if got != "[p1 p2]" {
		t.Errorf("matchs à reconstituer %s, attendu [p1 p2]", got)
	}
	if got := ImpactMatchesNeedingKVFallback(rows, [][]string{{"p1", "e1"}}); got != nil {
		t.Errorf("groupe unique avec un frag natif : rien à reconstituer, obtenu %v", got)
	}
}
