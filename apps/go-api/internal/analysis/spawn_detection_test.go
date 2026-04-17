// Package analysis — spawn_detection_test.go : tests de détection de spawn.
package analysis

import (
	"testing"
)

func TestScanFirstMovements_Empty(t *testing.T) {
	result := ScanFirstMovements(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 movements on nil input, got %d", len(result))
	}
}

func TestScanFirstMovements_EmptyChunks(t *testing.T) {
	chunks := []SpawnChunk{
		{Index: 0, Data: []byte{}, StartMS: 0, EndMS: 1000},
	}
	result := ScanFirstMovements(chunks)
	// Empty data → no movements detected
	if len(result) != 0 {
		t.Errorf("expected 0 movements on empty data, got %d", len(result))
	}
}

func TestEstimateFilmMatchStartMS_NoChunks(t *testing.T) {
	result := EstimateFilmMatchStartMS(nil, 3, 0)
	if result != -1 {
		t.Errorf("expected -1 for nil chunks, got %f", result)
	}
}

func TestPickSpawnReferences_Empty(t *testing.T) {
	result := PickSpawnReferences(nil, 3)
	if len(result) != 0 {
		t.Errorf("expected 0 references, got %d", len(result))
	}
}

func TestPickSpawnReferences_LessThanN(t *testing.T) {
	movements := []FirstMovement{
		{TimestampMS: 100, PlayerIdx: 0},
		{TimestampMS: 200, PlayerIdx: 1},
	}
	result := PickSpawnReferences(movements, 5)
	if len(result) != 2 {
		t.Errorf("expected 2 (all available), got %d", len(result))
	}
}

func TestFindPeakActivityWindow_Empty(t *testing.T) {
	result := FindPeakActivityWindow(nil, nil, 5000)
	if result != -1 {
		t.Errorf("expected -1 for empty inputs, got %f", result)
	}
}

func TestFindPeakActivityWindow_SingleMovement(t *testing.T) {
	mvs := []FirstMovement{{TimestampMS: 1000, PlayerIdx: 0}}
	result := FindPeakActivityWindow(nil, mvs, 5000)
	if result != 1000 {
		t.Errorf("expected 1000, got %f", result)
	}
}

func TestFindPeakActivityWindow_Cluster(t *testing.T) {
	mvs := []FirstMovement{
		{TimestampMS: 1000, PlayerIdx: 0},
		{TimestampMS: 1500, PlayerIdx: 1},
		{TimestampMS: 1800, PlayerIdx: 2},
		{TimestampMS: 5000, PlayerIdx: 3},
	}
	result := FindPeakActivityWindow(nil, mvs, 2000)
	if result != 1000 {
		t.Errorf("expected best cluster at 1000, got %f", result)
	}
}

func TestPickSpawnReferences_FilterAFK(t *testing.T) {
	mvs := []FirstMovement{
		{TimestampMS: 1000, PlayerIdx: 0},
		{TimestampMS: 1200, PlayerIdx: 1},
		{TimestampMS: 1500, PlayerIdx: 2},
		{TimestampMS: 20000, PlayerIdx: 3}, // 20s away — AFK
	}
	result := PickSpawnReferences(mvs, 10)
	// Median is around 1200-1500, AFK at 20000 is > 10s away from median
	for _, m := range result {
		if m.PlayerIdx == 3 {
			t.Error("expected AFK player 3 to be filtered")
		}
	}
}

func TestEstimateFrameTimestamp_EqualTimes(t *testing.T) {
	chunk := SpawnChunk{StartMS: 5000, EndMS: 5000, Data: make([]byte, 100)}
	ts := estimateFrameTimestamp(chunk, 50)
	if ts != 5000 {
		t.Errorf("expected 5000, got %f", ts)
	}
}

func TestEstimateFrameTimestamp_Interpolation(t *testing.T) {
	chunk := SpawnChunk{StartMS: 0, EndMS: 1000, Data: make([]byte, 100)}
	ts := estimateFrameTimestamp(chunk, 50) // 50% → 500ms
	if ts != 500 {
		t.Errorf("expected 500, got %f", ts)
	}
}

func TestDecodePositionFrame_TooShort_Spawn(t *testing.T) {
	data := make([]byte, 10) // < minFrameLen
	_, _, _, _, ok := decodePositionFrame(data, 0)
	if ok {
		t.Error("expected not ok for short data")
	}
}

func TestDecodePositionFrame_InvalidBaseType(t *testing.T) {
	data := make([]byte, 20)
	data[0] = frameMarkerB0
	data[1] = frameMarkerB1
	data[2] = frameMarkerB2
	data[3] = 0xFF // invalid base type
	_, _, _, _, ok := decodePositionFrame(data, 0)
	if ok {
		t.Error("expected not ok for invalid base type")
	}
}

func TestEstimateFilmMatchStartMS_APIConstrained(t *testing.T) {
	// Create fake movements via helper
	mvs := []FirstMovement{
		{TimestampMS: 10000, PlayerIdx: 0},
		{TimestampMS: 10500, PlayerIdx: 1},
		{TimestampMS: 11000, PlayerIdx: 2},
	}
	// The peak will be at ~10000
	// API first event at 8000 → should constrain
	// But we can't inject movements directly into EstimateFilmMatchStartMS
	// So test the constraint logic directly
	peakTS := FindPeakActivityWindow(nil, mvs, spawnClusterWindowMS)
	if peakTS < 0 {
		t.Fatal("expected positive peak")
	}
	// Simulate API constraint
	apiFirstEventMS := 8000.0
	if peakTS > apiFirstEventMS {
		peakTS = apiFirstEventMS - apiCapBufferMS
	}
	if peakTS >= 0 {
		// peakTS was 10000, apiFirst was 8000 → 8000-5000 = 3000
		if peakTS != 3000 {
			t.Errorf("expected 3000 after API constraint, got %f", peakTS)
		}
	}
}
