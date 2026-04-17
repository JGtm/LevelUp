// Package analysis — kill_attribution_test.go : tests unitaires KillAttribution.
package analysis

import "testing"

func TestKillAttribution_EffectiveWeaponID_ReconciledAs(t *testing.T) {
	reconciled := uint64(999)
	weapon := uint64(1)
	ka := &KillAttribution{
		WeaponID:     &weapon,
		ReconciledAs: &reconciled,
	}
	eff := ka.EffectiveWeaponID()
	if eff == nil || *eff != 999 {
		t.Errorf("expected 999 (reconciled), got %v", eff)
	}
}

func TestKillAttribution_EffectiveWeaponID_FallbackWeapon(t *testing.T) {
	weapon := uint64(42)
	ka := &KillAttribution{
		WeaponID:     &weapon,
		ReconciledAs: nil,
	}
	eff := ka.EffectiveWeaponID()
	if eff == nil || *eff != 42 {
		t.Errorf("expected 42 (weapon), got %v", eff)
	}
}

func TestKillAttribution_EffectiveWeaponID_BothNil(t *testing.T) {
	ka := &KillAttribution{
		WeaponID:     nil,
		ReconciledAs: nil,
	}
	eff := ka.EffectiveWeaponID()
	if eff != nil {
		t.Errorf("expected nil, got %v", eff)
	}
}

func TestKillAttribution_Fields(t *testing.T) {
	conf := "high"
	path := "fire_event"
	timeMS := 1500
	ka := &KillAttribution{
		MatchID:         "match-1",
		XUID:            "xuid-123",
		TimeMS:          timeMS,
		Confidence:      conf,
		AttributionPath: path,
		SwapDetected:    true,
	}

	if ka.MatchID != "match-1" {
		t.Errorf("MatchID = %q", ka.MatchID)
	}
	if ka.Confidence != "high" {
		t.Errorf("Confidence = %q", ka.Confidence)
	}
	if !ka.SwapDetected {
		t.Error("expected SwapDetected=true")
	}
}
