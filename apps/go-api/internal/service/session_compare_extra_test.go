package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func TestExtractSessionLabels_Empty(t *testing.T) {
	got := extractSessionLabels(nil)
	if len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestExtractSessionLabels_Dedup(t *testing.T) {
	lbl := "2025-01-15"
	matches := []domain.StatsMatchRow{
		{SessionLabel: &lbl},
		{SessionLabel: &lbl},
	}
	got := extractSessionLabels(matches)
	if len(got) != 1 {
		t.Errorf("expected 1 unique, got %d", len(got))
	}
}

func TestExtractSessionLabels_Sorted(t *testing.T) {
	l1, l2 := "2025-02-01", "2025-01-01"
	matches := []domain.StatsMatchRow{
		{SessionLabel: &l1},
		{SessionLabel: &l2},
	}
	got := extractSessionLabels(matches)
	if got[0] != "2025-01-01" {
		t.Errorf("expected sorted, got %v", got)
	}
}

func TestExtractSessionLabels_NilSkipped(t *testing.T) {
	lbl := "S1"
	matches := []domain.StatsMatchRow{
		{SessionLabel: nil},
		{SessionLabel: &lbl},
	}
	got := extractSessionLabels(matches)
	if len(got) != 1 || got[0] != "S1" {
		t.Errorf("expected [S1], got %v", got)
	}
}
