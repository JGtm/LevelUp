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
