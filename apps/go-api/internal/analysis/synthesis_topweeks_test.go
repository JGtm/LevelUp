package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

func TestComputeSynthesisTopWeeks_Empty(t *testing.T) {
	// Retourne une slice vide initialisée et NON nil — un slice nil sérialise
	// en JSON `null` et crashe le front non-nullable (cf. testutil.RequireNoNilSlicesWithoutOmitempty).
	result := ComputeSynthesisTopWeeks(nil)
	if result == nil {
		t.Fatalf("expected non-nil empty slice, got nil (sérialiserait en JSON null)")
	}
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %d entries", len(result))
	}
}

func TestComputeSynthesisTopWeeks_Single(t *testing.T) {
	kda := 2.5
	base := time.Date(2024, 6, 10, 14, 0, 0, 0, time.UTC) // Monday
	var rows []legacymatch.SynthesisMatchRow
	for i := 0; i < 3; i++ { // minimum 3 matches per week
		rows = append(rows, legacymatch.SynthesisMatchRow{
			MatchID:   "m" + string(rune('1'+i)),
			StartTime: base.Add(time.Duration(i) * time.Hour),
			Outcome:   domain.OutcomeWin,
			Kills:     20,
			Deaths:    5,
			KDA:       &kda,
		})
	}
	result := ComputeSynthesisTopWeeks(rows)
	if len(result) != 1 {
		t.Fatalf("expected 1 week, got %d", len(result))
	}
	if result[0].MatchCount != 3 {
		t.Fatalf("expected 3 matches, got %d", result[0].MatchCount)
	}
}

func TestComputeSynthesisTopWeeks_MultipleWeeks(t *testing.T) {
	kda := 3.0
	rows := make([]legacymatch.SynthesisMatchRow, 0, 20)
	// Week 1: 10 matches with wins
	base := time.Date(2024, 6, 10, 14, 0, 0, 0, time.UTC) // Monday
	for i := 0; i < 10; i++ {
		rows = append(rows, legacymatch.SynthesisMatchRow{
			MatchID:   "w1-" + string(rune('a'+i)),
			StartTime: base.Add(time.Duration(i) * time.Hour),
			Outcome:   domain.OutcomeWin,
			Kills:     20,
			Deaths:    5,
			KDA:       &kda,
		})
	}
	// Week 2: 3 matches
	base2 := base.AddDate(0, 0, 7)
	for i := 0; i < 3; i++ {
		rows = append(rows, legacymatch.SynthesisMatchRow{
			MatchID:   "w2-" + string(rune('a'+i)),
			StartTime: base2.Add(time.Duration(i) * time.Hour),
			Outcome:   domain.OutcomeLoss,
			Kills:     5,
			Deaths:    10,
			KDA:       &kda,
		})
	}
	result := ComputeSynthesisTopWeeks(rows)
	if len(result) > 5 {
		t.Fatalf("expected max 5, got %d", len(result))
	}
}

func TestComputeSynthesisTopWeeks_NoKDA(t *testing.T) {
	base := time.Date(2024, 6, 10, 14, 0, 0, 0, time.UTC)
	var rows []legacymatch.SynthesisMatchRow
	for i := 0; i < 3; i++ {
		rows = append(rows, legacymatch.SynthesisMatchRow{
			MatchID:   "m" + string(rune('a'+i)),
			StartTime: base.Add(time.Duration(i) * time.Hour),
			Outcome:   domain.OutcomeWin,
			Kills:     10,
			Deaths:    5,
			KDA:       nil, // no KDA
		})
	}
	result := ComputeSynthesisTopWeeks(rows)
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
}
