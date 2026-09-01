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
