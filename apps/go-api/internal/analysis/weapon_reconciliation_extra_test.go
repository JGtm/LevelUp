package analysis

import (
	"testing"
)

func TestReconcileAPIAggregates_NoSurplus(t *testing.T) {
	wid := uint64(100)
	attrs := []KillAttribution{
		{Confidence: "high", WeaponID: &wid},
	}
	api := map[uint64]int{100: 1}
	result := ReconcileAPIAggregates(attrs, api)
	if result[0].ReconciledAs != nil {
		t.Fatal("expected no reconciliation when no surplus")
	}
}

func TestReconcileAPIAggregates_WithSurplus(t *testing.T) {
	wid := uint64(100)
	attrs := []KillAttribution{
		{Confidence: "high", WeaponID: &wid},
		{Confidence: "low", WeaponID: nil, AttributionPath: "formula_a"},
	}
	api := map[uint64]int{100: 2}
	result := ReconcileAPIAggregates(attrs, api)
	if result[1].ReconciledAs == nil {
		t.Fatal("expected reconciliation for low confidence")
	}
	if *result[1].ReconciledAs != 100 {
		t.Fatalf("expected reconciled to 100, got %d", *result[1].ReconciledAs)
	}
}

func TestReconcileAPIAggregates_SkipNonePath(t *testing.T) {
	attrs := []KillAttribution{
		{Confidence: "none", AttributionPath: "none"},
	}
	api := map[uint64]int{100: 5}
	result := ReconcileAPIAggregates(attrs, api)
	if result[0].ReconciledAs != nil {
		t.Fatal("expected skip for path=none")
	}
}

func TestReconcileAPIAggregates_SkipSentinel(t *testing.T) {
	sentinel := uint64(0) // GRENADE_WEAPON_ID in SentinelIDs
	attrs := []KillAttribution{
		{Confidence: "low", WeaponID: &sentinel, AttributionPath: "fire_event"},
	}
	api := map[uint64]int{100: 5}
	result := ReconcileAPIAggregates(attrs, api)
	if result[0].ReconciledAs != nil {
		t.Fatal("expected skip for sentinel weapon_id")
	}
}

func TestAssignSentinels_Extra(t *testing.T) {
	attrs := []KillAttribution{
		{XUID: "123", TimeMS: 5000, Confidence: "none"},
		{XUID: "456", TimeMS: 6000, Confidence: "none"},
	}
	sentinelMap := map[string]uint64{
		"123_5000": 1,
	}
	result := AssignSentinels(attrs, sentinelMap)
	if result[0].ReconciledAs == nil || *result[0].ReconciledAs != 1 {
		t.Fatal("expected sentinel assignment for matching key")
	}
	if result[1].ReconciledAs != nil {
		t.Fatal("expected no assignment for non-matching key")
	}
}

func TestItoa_Extra(t *testing.T) {
	cases := []struct {
		in  int
		out string
	}{
		{0, "0"},
		{42, "42"},
		{-7, "-7"},
		{1000, "1000"},
	}
	for _, c := range cases {
		got := itoa(c.in)
		if got != c.out {
			t.Errorf("itoa(%d) = %q, want %q", c.in, got, c.out)
		}
	}
}
