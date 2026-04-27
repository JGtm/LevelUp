package port

import (
	"errors"
	"testing"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/games/canonical"
)

func TestPlayerMatchFilters_Validate_OK(t *testing.T) {
	t.Parallel()
	period := temporal.Period1Y
	had := false
	min := 180
	cases := []struct {
		name    string
		filters PlayerMatchFilters
	}{
		{"empty filters", PlayerMatchFilters{}},
		{"period only", PlayerMatchFilters{Period: &period}},
		{"outcome win/loss only", PlayerMatchFilters{
			OutcomeIn: []canonical.Outcome{canonical.OutcomeWin, canonical.OutcomeLoss},
		}},
		{"all canonical filters", PlayerMatchFilters{
			Period:               &period,
			OutcomeIn:            []canonical.Outcome{canonical.OutcomeWin},
			HadBotTeammate:       &had,
			MinTimePlayedSeconds: &min,
			BTBExcluded:          true,
			Limit:                10,
		}},
		{"limit zero allowed", PlayerMatchFilters{Limit: 0}},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if err := c.filters.Validate(); err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestPlayerMatchFilters_Validate_NegativeLimit(t *testing.T) {
	t.Parallel()
	f := PlayerMatchFilters{Limit: -1}
	err := f.Validate()
	if err == nil {
		t.Fatal("expected error for negative Limit")
	}
	if !errors.Is(err, ErrPlayerMatchFiltersInvalid) {
		t.Errorf("error chain should contain ErrPlayerMatchFiltersInvalid, got %v", err)
	}
}

func TestPlayerMatchFilters_Validate_NegativeMinTime(t *testing.T) {
	t.Parallel()
	min := -10
	f := PlayerMatchFilters{MinTimePlayedSeconds: &min}
	err := f.Validate()
	if err == nil {
		t.Fatal("expected error for negative MinTimePlayedSeconds")
	}
	if !errors.Is(err, ErrPlayerMatchFiltersInvalid) {
		t.Errorf("error chain mismatch, got %v", err)
	}
}

func TestPlayerMatchFilters_Validate_UnknownPeriod(t *testing.T) {
	t.Parallel()
	bad := temporal.Period("3y")
	f := PlayerMatchFilters{Period: &bad}
	err := f.Validate()
	if err == nil {
		t.Fatal("expected error for unknown Period")
	}
	if !errors.Is(err, ErrPlayerMatchFiltersInvalid) {
		t.Errorf("error chain mismatch, got %v", err)
	}
}

func TestPlayerMatchFilters_Validate_UnknownOutcome(t *testing.T) {
	t.Parallel()
	f := PlayerMatchFilters{
		OutcomeIn: []canonical.Outcome{canonical.OutcomeWin, canonical.Outcome("invalid")},
	}
	err := f.Validate()
	if err == nil {
		t.Fatal("expected error for unknown Outcome")
	}
	if !errors.Is(err, ErrPlayerMatchFiltersInvalid) {
		t.Errorf("error chain mismatch, got %v", err)
	}
}

func TestPlayerMatchFilters_Validate_AllowsZeroPeriodPointer(t *testing.T) {
	t.Parallel()
	// Period nil != Period(""). Period nil = pas de filtre.
	f := PlayerMatchFilters{Period: nil}
	if err := f.Validate(); err != nil {
		t.Errorf("nil Period should be valid (no filter), got %v", err)
	}
}
