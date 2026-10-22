package analysis

import (
	"testing"
	"time"
)

func cumulAt(t *testing.T, got map[CitationMatchKey]int, match, norm string, want int) {
	t.Helper()
	if v, ok := got[CitationMatchKey{MatchID: match, Norm: norm}]; !ok || v != want {
		t.Errorf("cumul(%s,%s) = %d (présent=%v), want %d", match, norm, v, ok, want)
	}
}

func TestCitationCumulAfterMatch_ArreteAuMatch(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 1, d, 12, 0, 0, 0, time.UTC) }
	history := []CitationHistoryEntry{
		// Volontairement hors ordre : le cumul suit l'heure de début, pas l'ordre de lecture.
		{MatchID: "m3", Norm: "presque", Value: 80, StartTime: day(3)},
		{MatchID: "m1", Norm: "presque", Value: 300, StartTime: day(1)},
		{MatchID: "m2", Norm: "presque", Value: 150, StartTime: day(2)},
		{MatchID: "m2", Norm: "autre", Value: 7, StartTime: day(2)},
		{MatchID: "m1", Norm: "autre", Value: 5, StartTime: day(1)},
	}
	got := CitationCumulAfterMatch(history)
	cumulAt(t, got, "m1", "presque", 300)
	cumulAt(t, got, "m2", "presque", 450)
	cumulAt(t, got, "m3", "presque", 530)
	// Une autre citation repart de zéro.
	cumulAt(t, got, "m1", "autre", 5)
	cumulAt(t, got, "m2", "autre", 12)
}

func TestCitationCumulAfterMatch_DepartageDeterministeEtHeureInconnue(t *testing.T) {
	same := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	history := []CitationHistoryEntry{
		{MatchID: "mb", Norm: "c", Value: 10, StartTime: same},
		{MatchID: "ma", Norm: "c", Value: 20, StartTime: same},
		// Début inconnu : après tous les matchs datés, jamais avant.
		{MatchID: "m0", Norm: "c", Value: 1},
	}
	got := CitationCumulAfterMatch(history)
	cumulAt(t, got, "ma", "c", 20) // à heure égale, match_id croissant
	cumulAt(t, got, "mb", "c", 30)
	cumulAt(t, got, "m0", "c", 31)

	// L'entrée n'est pas modifiée (ordre de l'appelant conservé).
	if history[0].MatchID != "mb" {
		t.Errorf("l'historique de l'appelant a été réordonné")
	}
}

func TestCitationCumulAfterMatch_Vide(t *testing.T) {
	if got := CitationCumulAfterMatch(nil); len(got) != 0 {
		t.Errorf("historique vide : %d entrées, want 0", len(got))
	}
}

// Reproduit le défaut : avec le total d'aujourd'hui comme cumul, des matchs anciens passent pour
// « maîtrisés pendant ce match ». Avec le cumul arrêté au match, seul le match qui franchit le
// dernier palier l'est ; les matchs d'avant montrent leur progression d'alors et ceux d'après
// sont filtrés.
func TestCitationCumulAfterMatch_NouvellementMaitriseUneSeuleFois(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 1, d, 12, 0, 0, 0, time.UTC) }
	tiers := ParseTierTargets("100,250,500")
	history := []CitationHistoryEntry{
		{MatchID: "m1", Norm: "c", Value: 300, StartTime: day(1)},
		{MatchID: "m2", Norm: "c", Value: 150, StartTime: day(2)}, // 450
		{MatchID: "m3", Norm: "c", Value: 100, StartTime: day(3)}, // 550 : franchit 500
		{MatchID: "m4", Norm: "c", Value: 80, StartTime: day(4)},  // 630 : déjà maîtrisée
	}
	cumul := CitationCumulAfterMatch(history)
	total := 630

	newly := func(useTotal bool) (out []string) {
		for _, h := range history {
			c := cumul[CitationMatchKey{MatchID: h.MatchID, Norm: h.Norm}]
			if useTotal {
				c = total
			}
			if ComputeTierProgression(c, h.Value, tiers).IsNewlyMastered {
				out = append(out, h.MatchID)
			}
		}
		return out
	}
	if got := newly(true); len(got) < 2 {
		t.Fatalf("le total d'aujourd'hui doit reproduire le défaut (plusieurs « nouvellement maîtrisée »), obtenu %v", got)
	}
	if got := newly(false); len(got) != 1 || got[0] != "m3" {
		t.Errorf("cumul arrêté au match : nouvellement maîtrisée = %v, want [m3]", got)
	}
	if !ComputeTierProgression(cumul[CitationMatchKey{MatchID: "m4", Norm: "c"}], 80, tiers).AlreadyMastered {
		t.Errorf("m4 doit être filtré (déjà maîtrisée avant)")
	}
	if tp := ComputeTierProgression(cumul[CitationMatchKey{MatchID: "m1", Norm: "c"}], 300, tiers); tp.TierIndex != 2 || tp.IsNewlyMastered {
		t.Errorf("m1 : palier %d maîtrisée=%v, want palier 2 non maîtrisée", tp.TierIndex, tp.IsNewlyMastered)
	}
}
