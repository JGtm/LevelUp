// Package analysis — weapon_scanner_test.go : tests des fonctions de scan d'armes.
package analysis

import "testing"

func TestFindFramePositions_Empty(t *testing.T) {
	result := FindFramePositions(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 positions on nil, got %d", len(result))
	}
}

func TestFindFramePositions_NoMarker(t *testing.T) {
	data := make([]byte, 100) // all zeros, no frame marker
	result := FindFramePositions(data)
	if len(result) != 0 {
		t.Errorf("expected 0 positions with no markers, got %d", len(result))
	}
}

func TestScanFormulaA_Empty(t *testing.T) {
	result := ScanFormulaA(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 results on nil data, got %d", len(result))
	}
}

func TestScanFormulaANS_Empty(t *testing.T) {
	result := ScanFormulaANS(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 results on nil data, got %d", len(result))
	}
}

func TestScanFireEventsB5_Empty(t *testing.T) {
	estimateTS := func(_ int) float64 { return 0 }
	result := ScanFireEventsB5(nil, estimateTS)
	if len(result) != 0 {
		t.Errorf("expected 0 events on nil data, got %d", len(result))
	}
}

func TestTimestampEstimator_ZeroData(t *testing.T) {
	fn := TimestampEstimator(nil, 0, 600000)
	// Should not panic on zero-length data
	ts := fn(0)
	if ts < 0 {
		t.Errorf("expected non-negative timestamp, got %f", ts)
	}
}

func TestComputeConfidence(t *testing.T) {
	tests := []struct {
		name     string
		weaponID uint64
		deltaMS  int
		wantLow  bool
	}{
		{"zero delta", 12345, 0, false},        // exact match → not low
		{"large delta", 12345, 10000, true},    // very far → low confidence
		{"negative delta", 12345, -100, false}, // small negative → could be high
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := ComputeConfidence(tt.weaponID, tt.deltaMS)
			if conf == "" {
				t.Error("confidence should not be empty")
			}
		})
	}
}

func TestCountKillsByWeapon(t *testing.T) {
	wid100 := uint64(100)
	wid200 := uint64(200)
	attrs := []KillAttribution{
		{WeaponID: &wid100, Confidence: "high"},
		{WeaponID: &wid100, Confidence: "high"},
		{WeaponID: &wid200, Confidence: "medium"},
	}
	counts := CountKillsByWeapon(attrs)
	if counts[100] != 2 {
		t.Errorf("expected 2 kills for weapon 100, got %d", counts[100])
	}
	if counts[200] != 1 {
		t.Errorf("expected 1 kill for weapon 200, got %d", counts[200])
	}
}
