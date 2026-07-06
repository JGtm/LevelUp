package teammates

// Tests purs sur filterSynthesisByPeriodInput et filterSynthesisByPickedSessions —
// les deux helpers ajoutés pour faire fonctionner la navigation rail
// (PeriodSessionRail) sur l'écran Escouade. Avant ce fix, teammates_service
// chargeait toutes les SynthesisMatchRow et n'appliquait que session-labels +
// cascade — ni `req.Filters.Period` ni `req.Filters.Sessions.PickedSessions`
// n'étaient consommés, donc cliquer "Précédente" sur le rail ne changeait
// rien aux charts/tableaux.

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

func makeSynthAt(matchID string, when time.Time, label string) legacymatch.SynthesisMatchRow {
	lbl := label
	return legacymatch.SynthesisMatchRow{
		MatchID:       matchID,
		StartTime:     when,
		IsWithFriends: true,
		SessionLabel:  &lbl,
	}
}

// ---------- filterSynthesisByPeriodInput ----------

func TestFilterSynthesisByPeriodInput_NoFilter(t *testing.T) {
	rows := []legacymatch.SynthesisMatchRow{
		makeSynthAt("m1", time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC), "s-1"),
		makeSynthAt("m2", time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC), "s-2"),
	}
	got := filterSynthesisByPeriodInput(rows, domain.PeriodInput{})
	if len(got) != 2 {
		t.Errorf("expected 2 rows (no filter), got %d", len(got))
	}
}

func TestFilterSynthesisByPeriodInput_StartDate(t *testing.T) {
	start := time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC)
	rows := []legacymatch.SynthesisMatchRow{
		makeSynthAt("m1", time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC), "s-1"),
		makeSynthAt("m2", time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC), "s-2"),
	}
	got := filterSynthesisByPeriodInput(rows, domain.PeriodInput{StartDate: &start})
	if len(got) != 1 {
		t.Fatalf("expected 1 row >= start, got %d", len(got))
	}
	if got[0].MatchID != "m2" {
		t.Errorf("expected m2, got %s", got[0].MatchID)
	}
}

func TestFilterSynthesisByPeriodInput_EndDateInclusive(t *testing.T) {
	// EndDate inclusive : un match à 23:59 le jour d'end_date doit passer.
	end := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	rows := []legacymatch.SynthesisMatchRow{
		makeSynthAt("m1", time.Date(2026, 4, 1, 23, 59, 0, 0, time.UTC), "s-1"),
		makeSynthAt("m2", time.Date(2026, 4, 2, 0, 0, 1, 0, time.UTC), "s-2"),
	}
	got := filterSynthesisByPeriodInput(rows, domain.PeriodInput{EndDate: &end})
	if len(got) != 1 {
		t.Fatalf("expected 1 row <= end (inclusive 23:59:59), got %d", len(got))
	}
	if got[0].MatchID != "m1" {
		t.Errorf("expected m1, got %s", got[0].MatchID)
	}
}

func TestFilterSynthesisByPeriodInput_BothBounds(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)
	rows := []legacymatch.SynthesisMatchRow{
		makeSynthAt("m0", time.Date(2026, 3, 31, 23, 0, 0, 0, time.UTC), "s-0"),
		makeSynthAt("m1", time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC), "s-1"),
		makeSynthAt("m2", time.Date(2026, 4, 5, 23, 0, 0, 0, time.UTC), "s-2"),
		makeSynthAt("m3", time.Date(2026, 4, 6, 0, 0, 1, 0, time.UTC), "s-3"),
	}
	got := filterSynthesisByPeriodInput(rows, domain.PeriodInput{StartDate: &start, EndDate: &end})
	if len(got) != 2 {
		t.Fatalf("expected 2 rows in [4/1, 4/5], got %d", len(got))
	}
}

// ---------- filterSynthesisByPickedSessions ----------

func TestFilterSynthesisByPickedSessions_NoFilter(t *testing.T) {
	rows := []legacymatch.SynthesisMatchRow{
		makeSynthAt("m1", time.Now(), "30/04/2026 18h"),
		makeSynthAt("m2", time.Now(), "01/05/2026 14h"),
	}
	got := filterSynthesisByPickedSessions(rows, nil)
	if len(got) != 2 {
		t.Errorf("expected 2 (empty picked), got %d", len(got))
	}
}

func TestFilterSynthesisByPickedSessions_SingleLabel(t *testing.T) {
	// Régression : le rail nav écrit le label dans picked_sessions ;
	// teammates_service doit le matcher contre SessionLabel.
	rows := []legacymatch.SynthesisMatchRow{
		makeSynthAt("m1", time.Now(), "30/04/2026 18h"),
		makeSynthAt("m2", time.Now(), "01/05/2026 14h"),
	}
	got := filterSynthesisByPickedSessions(rows, []string{"01/05/2026 14h"})
	if len(got) != 1 {
		t.Fatalf("expected 1 row matching label, got %d", len(got))
	}
	if got[0].MatchID != "m2" {
		t.Errorf("expected m2, got %s", got[0].MatchID)
	}
}

func TestFilterSynthesisByPickedSessions_MultiLabels(t *testing.T) {
	rows := []legacymatch.SynthesisMatchRow{
		makeSynthAt("m1", time.Now(), "A"),
		makeSynthAt("m2", time.Now(), "B"),
		makeSynthAt("m3", time.Now(), "C"),
	}
	got := filterSynthesisByPickedSessions(rows, []string{"A", "C"})
	if len(got) != 2 {
		t.Fatalf("expected 2 rows (A + C), got %d", len(got))
	}
}

func TestFilterSynthesisByPickedSessions_NilLabel(t *testing.T) {
	row := legacymatch.SynthesisMatchRow{MatchID: "m1", StartTime: time.Now(), SessionLabel: nil}
	got := filterSynthesisByPickedSessions([]legacymatch.SynthesisMatchRow{row}, []string{"anything"})
	if len(got) != 0 {
		t.Errorf("expected 0 (nil label can't match), got %d", len(got))
	}
}
