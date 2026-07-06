package wire

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func TestAllZeroXPTotal_EmptyHistory(t *testing.T) {
	if !allZeroXPTotal(nil) {
		t.Error("expected true for nil history")
	}
	if !allZeroXPTotal([]domain.XPHistoryPoint{}) {
		t.Error("expected true for empty history")
	}
}

func TestAllZeroXPTotal_AllZero(t *testing.T) {
	now := time.Now()
	history := []domain.XPHistoryPoint{
		{RecordedAt: now.Add(-2 * time.Hour), XPTotal: 0},
		{RecordedAt: now.Add(-time.Hour), XPTotal: 0},
		{RecordedAt: now, XPTotal: 0},
	}
	if !allZeroXPTotal(history) {
		t.Error("expected true for all-zero history")
	}
}

func TestAllZeroXPTotal_AnyPositive(t *testing.T) {
	now := time.Now()
	history := []domain.XPHistoryPoint{
		{RecordedAt: now.Add(-2 * time.Hour), XPTotal: 0},
		{RecordedAt: now.Add(-time.Hour), XPTotal: 100_000},
		{RecordedAt: now, XPTotal: 0},
	}
	if allZeroXPTotal(history) {
		t.Error("expected false when at least one XPTotal > 0")
	}
}

func TestAllZeroXPTotal_NegativeIgnored(t *testing.T) {
	// Defensive : XPTotal negatif (jamais legitime) traite comme zero.
	now := time.Now()
	history := []domain.XPHistoryPoint{
		{RecordedAt: now, XPTotal: -100},
	}
	if !allZeroXPTotal(history) {
		t.Error("expected true when only negative values (treated as zero)")
	}
}
