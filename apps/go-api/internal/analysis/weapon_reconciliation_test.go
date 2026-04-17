// Package analysis — weapon_reconciliation_test.go : tests de réconciliation d'armes.
package analysis

import "testing"

func TestCountConfidentAttributions(t *testing.T) {
	wid1 := uint64(100)
	wid2 := uint64(200)
	attrs := []KillAttribution{
		{WeaponID: &wid1, Confidence: "high"},
		{WeaponID: &wid1, Confidence: "medium"},
		{WeaponID: &wid2, Confidence: "high"},
		{Confidence: "none"}, // pas de weapon_id
	}
	counts := countConfidentAttributions(attrs)
	if counts[100] != 2 {
		t.Errorf("expected 2 for weapon 100, got %d", counts[100])
	}
	if counts[200] != 1 {
		t.Errorf("expected 1 for weapon 200, got %d", counts[200])
	}
}

func TestComputeSurplus_NoSurplus(t *testing.T) {
	api := map[uint64]int{100: 5, 200: 3}
	film := map[uint64]int{100: 5, 200: 3}
	surplus := computeSurplus(api, film)
	for k, v := range surplus {
		if v > 0 {
			t.Errorf("unexpected surplus for weapon %d: %d", k, v)
		}
	}
}

func TestComputeSurplus_WithSurplus(t *testing.T) {
	api := map[uint64]int{100: 10, 200: 3}
	film := map[uint64]int{100: 5, 200: 3}
	surplus := computeSurplus(api, film)
	if surplus[100] != 5 {
		t.Errorf("expected surplus of 5 for weapon 100, got %d", surplus[100])
	}
}

func TestFindBestSurplus_Empty(t *testing.T) {
	result := findBestSurplus(nil)
	if result != nil {
		t.Error("expected nil for empty surplus")
	}
}

func TestFindBestSurplus_OneEntry(t *testing.T) {
	surplus := map[uint64]int{100: 5}
	result := findBestSurplus(surplus)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if *result != 100 {
		t.Errorf("expected weapon 100, got %d", *result)
	}
}

func TestAssignSentinels_Empty(t *testing.T) {
	result := AssignSentinels(nil, nil)
	if len(result) != 0 {
		t.Errorf("expected 0, got %d", len(result))
	}
}

func TestReconcileAPIAggregates_Empty(t *testing.T) {
	result := ReconcileAPIAggregates(nil, nil)
	if len(result) != 0 {
		t.Errorf("expected 0, got %d", len(result))
	}
}

func TestItoa_Zero(t *testing.T) {
	if itoa(0) != "0" {
		t.Errorf("expected 0, got %s", itoa(0))
	}
}

func TestItoa_Positive(t *testing.T) {
	if itoa(42) != "42" {
		t.Errorf("expected 42, got %s", itoa(42))
	}
}

func TestItoa_Negative(t *testing.T) {
	if itoa(-7) != "-7" {
		t.Errorf("expected -7, got %s", itoa(-7))
	}
}

func TestItoa_Large(t *testing.T) {
	if itoa(123456) != "123456" {
		t.Errorf("expected 123456, got %s", itoa(123456))
	}
}
