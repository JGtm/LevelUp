package port

import (
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/games/canonical"
)

func TestHighlightEventFilters_Validate_OK(t *testing.T) {
	t.Parallel()
	xuid := "xuid-1"
	since := time.Now().AddDate(0, -1, 0)
	cases := []struct {
		name    string
		filters HighlightEventFilters
	}{
		{"matchIDs only", HighlightEventFilters{MatchIDs: []string{"m1"}}},
		{"player + since", HighlightEventFilters{PlayerXUID: &xuid, Since: &since}},
		{"matchIDs + types + limit", HighlightEventFilters{
			MatchIDs:   []string{"m1", "m2"},
			EventTypes: []canonical.HighlightEventType{canonical.EventKill, canonical.EventDeath},
			Limit:      100,
		}},
		{"both matchIDs and player+since", HighlightEventFilters{
			MatchIDs:   []string{"m1"},
			PlayerXUID: &xuid,
			Since:      &since,
		}},
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

func TestHighlightEventFilters_Validate_TooBroad_NoFilters(t *testing.T) {
	t.Parallel()
	f := HighlightEventFilters{}
	err := f.Validate()
	if err == nil {
		t.Fatal("expected error for completely empty filters")
	}
	if !errors.Is(err, ErrHighlightEventFiltersTooBroad) {
		t.Errorf("expected ErrHighlightEventFiltersTooBroad, got %v", err)
	}
}

func TestHighlightEventFilters_Validate_TooBroad_PlayerOnly(t *testing.T) {
	t.Parallel()
	xuid := "xuid-1"
	f := HighlightEventFilters{PlayerXUID: &xuid}
	err := f.Validate()
	if !errors.Is(err, ErrHighlightEventFiltersTooBroad) {
		t.Errorf("expected ErrHighlightEventFiltersTooBroad (player without since), got %v", err)
	}
}

func TestHighlightEventFilters_Validate_TooBroad_SinceOnly(t *testing.T) {
	t.Parallel()
	since := time.Now().AddDate(0, -1, 0)
	f := HighlightEventFilters{Since: &since}
	err := f.Validate()
	if !errors.Is(err, ErrHighlightEventFiltersTooBroad) {
		t.Errorf("expected ErrHighlightEventFiltersTooBroad (since without player), got %v", err)
	}
}

func TestHighlightEventFilters_Validate_NegativeLimit(t *testing.T) {
	t.Parallel()
	f := HighlightEventFilters{
		MatchIDs: []string{"m1"},
		Limit:    -1,
	}
	err := f.Validate()
	if err == nil {
		t.Fatal("expected error for negative Limit")
	}
	if !errors.Is(err, ErrHighlightEventFiltersInvalid) {
		t.Errorf("error chain should contain ErrHighlightEventFiltersInvalid, got %v", err)
	}
}

func TestHighlightEventFilters_Validate_UnknownEventType(t *testing.T) {
	t.Parallel()
	f := HighlightEventFilters{
		MatchIDs:   []string{"m1"},
		EventTypes: []canonical.HighlightEventType{canonical.EventKill, canonical.HighlightEventType("hacked")},
	}
	err := f.Validate()
	if err == nil {
		t.Fatal("expected error for unknown EventType")
	}
	if !errors.Is(err, ErrHighlightEventFiltersInvalid) {
		t.Errorf("error chain mismatch, got %v", err)
	}
}

func TestHighlightEventFilters_Validate_EmptyMatchIDsListNotPointer(t *testing.T) {
	t.Parallel()
	// Garantit que `MatchIDs: []string{}` (slice vide) est traite comme "absent"
	// par la regle too-broad, pas comme "filtre present mais sans match".
	xuid := "xuid-1"
	since := time.Now()
	f := HighlightEventFilters{MatchIDs: []string{}, PlayerXUID: &xuid, Since: &since}
	if err := f.Validate(); err != nil {
		t.Errorf("empty MatchIDs with player+since should pass, got %v", err)
	}
}
