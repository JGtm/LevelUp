package analysis

import "testing"

// spawn_detection_bits_test.go — les deux aides de `spawn_detection.go`.
//
// ELLES VIVAIENT DANS `scanner_bits_test.go`, un fichier qui couvrait DEUX paquets a la fois :
// les lecteurs de bits prives du scanner d armes et ces deux-ci. Le scanner est descendu dans la
// couche `grammar` au lot 2.5.e (`film/grammar/weaponscan`) ; la detection de spawn, elle, ne
// bouge pas — d ou la coupe.

// ---------------------------------------------------------------------------
// estimateFrameTimestamp
// ---------------------------------------------------------------------------

func TestEstimateFrameTimestamp_ZeroDuration(t *testing.T) {
	chunk := SpawnChunk{StartMS: 1000, EndMS: 1000, Data: []byte{0, 0, 0}}
	got := estimateFrameTimestamp(chunk, 1)
	if got != 1000 {
		t.Errorf("expected 1000, got %f", got)
	}
}

func TestEstimateFrameTimestamp_Midpoint(t *testing.T) {
	chunk := SpawnChunk{StartMS: 0, EndMS: 100, Data: make([]byte, 100)}
	got := estimateFrameTimestamp(chunk, 50)
	if got != 50 {
		t.Errorf("expected 50, got %f", got)
	}
}

func TestEstimateFrameTimestamp_EmptyData(t *testing.T) {
	chunk := SpawnChunk{StartMS: 500, EndMS: 1000, Data: nil}
	got := estimateFrameTimestamp(chunk, 0)
	if got != 500 {
		t.Errorf("expected 500, got %f", got)
	}
}

// ---------------------------------------------------------------------------
// decodePositionFrame
// ---------------------------------------------------------------------------

func TestDecodePositionFrame_TooShort(t *testing.T) {
	data := make([]byte, 10)
	_, _, _, _, ok := decodePositionFrame(data, 0)
	if ok {
		t.Error("expected not ok for short data")
	}
}
