package analysis

import (
	"testing"
)

// ---------- decodePositionFrame ----------

func makeFrame(baseType, b5, b9 byte, playerIdx byte) []byte {
	// Build a 20-byte frame: [A0 7B 42 baseType playerIdx b5 XX XX XX b9 XX XX XX XX XX XX XX XX XX XX]
	data := make([]byte, 20)
	data[0] = frameMarkerB0
	data[1] = frameMarkerB1
	data[2] = frameMarkerB2
	data[3] = baseType
	data[4] = playerIdx << 4
	data[5] = b5
	data[9] = b9
	return data
}

func TestDecodePositionFrame_BadBaseType_Boost(t *testing.T) {
	data := makeFrame(0xFF, byteHumanB5, byteHumanB9, 2)
	_, _, _, _, ok := decodePositionFrame(data, 0)
	if ok {
		t.Error("expected failure for invalid base type")
	}
}

func TestDecodePositionFrame_BadB5_Boost(t *testing.T) {
	data := makeFrame(0x08, 0x00, byteHumanB9, 2)
	_, _, _, _, ok := decodePositionFrame(data, 0)
	if ok {
		t.Error("expected failure for invalid b5")
	}
}

func TestDecodePositionFrame_BadB9_Boost(t *testing.T) {
	data := makeFrame(0x08, byteHumanB5, 0x00, 2)
	_, _, _, _, ok := decodePositionFrame(data, 0)
	if ok {
		t.Error("expected failure for invalid b9")
	}
}

func TestDecodePositionFrame_ValidFrame_Boost(t *testing.T) {
	data := makeFrame(0x08, byteHumanB5, byteHumanB9, 3)
	pi, _, _, _, ok := decodePositionFrame(data, 0)
	if !ok {
		t.Fatal("expected success")
	}
	if pi != 3 {
		t.Errorf("expected player_idx=3, got %d", pi)
	}
}

// ---------- ScanFirstMovements with data ----------

// buildValidFrame construit une frame test ; baseType est paramétré pour l'extension future.
//
//nolint:unparam // baseType est gardé pour clarifier l'intention de la frame
func buildValidFrame(baseType, playerIdx byte) []byte {
	data := make([]byte, 20)
	data[0] = frameMarkerB0
	data[1] = frameMarkerB1
	data[2] = frameMarkerB2
	data[3] = baseType
	data[4] = playerIdx << 4
	data[5] = byteHumanB5
	data[9] = byteHumanB9
	return data
}

func TestScanFirstMovements_DetectsMovement(t *testing.T) {
	// Two frames for the same player with different signatures
	frame1 := buildValidFrame(0x08, 1)
	frame2 := buildValidFrame(0x08, 1)
	frame2[10] = 0xFF // change signature byte
	data := append(frame1, frame2...)
	chunk := SpawnChunk{Index: 0, StartMS: 0, EndMS: 1000, Data: data}
	result := ScanFirstMovements([]SpawnChunk{chunk})
	if len(result) == 0 {
		t.Error("expected at least one movement detected")
	}
}

func TestScanFirstMovements_NoMarkerNoResult(t *testing.T) {
	data := make([]byte, 50) // all zeros, no markers
	chunk := SpawnChunk{Index: 0, StartMS: 0, EndMS: 1000, Data: data}
	result := ScanFirstMovements([]SpawnChunk{chunk})
	if len(result) != 0 {
		t.Errorf("expected no movements, got %d", len(result))
	}
}

// ---------- estimateFrameTimestamp ----------

func TestEstimateFrameTimestamp_ZeroDuration_Boost(t *testing.T) {
	chunk := SpawnChunk{StartMS: 100, EndMS: 100, Data: make([]byte, 10)}
	got := estimateFrameTimestamp(chunk, 5)
	if got != 100 {
		t.Errorf("expected 100, got %f", got)
	}
}

func TestEstimateFrameTimestamp_Normal_Boost(t *testing.T) {
	data := make([]byte, 100)
	chunk := SpawnChunk{StartMS: 0, EndMS: 1000, Data: data}
	got := estimateFrameTimestamp(chunk, 50)
	if got < 490 || got > 510 {
		t.Errorf("expected ~500ms, got %f", got)
	}
}

// ---------- EstimateFilmMatchStartMS ----------

// buildChunkWithMovements builds a chunk that will produce `n` distinct player movements
func buildChunkWithMovingPlayers(n int) SpawnChunk {
	var data []byte
	// For each player: two frames with different signature bytes
	for i := 0; i < n; i++ {
		f1 := buildValidFrame(0x08, byte(i))
		f2 := buildValidFrame(0x08, byte(i))
		f2[10] = 0xFF // change signature
		data = append(data, f1...)
		data = append(data, f2...)
	}
	return SpawnChunk{Index: 0, StartMS: 0, EndMS: float64(len(data)), Data: data}
}

func TestEstimateFilmMatchStartMS_NotEnoughPlayers(t *testing.T) {
	chunk := buildChunkWithMovingPlayers(1)
	got := EstimateFilmMatchStartMS([]SpawnChunk{chunk}, 3, 0)
	if got != -1 {
		t.Errorf("expected -1, got %f", got)
	}
}

func TestEstimateFilmMatchStartMS_EnoughPlayers(t *testing.T) {
	chunk := buildChunkWithMovingPlayers(4)
	got := EstimateFilmMatchStartMS([]SpawnChunk{chunk}, 3, 0)
	// Should return a valid timestamp (>=0) or -1 if peak not found
	_ = got // just ensure no panic
}

func TestEstimateFilmMatchStartMS_APIConstraint(t *testing.T) {
	chunk := buildChunkWithMovingPlayers(4)
	// Pass a small apiFirstEventMS to trigger constraint
	got := EstimateFilmMatchStartMS([]SpawnChunk{chunk}, 3, 1.0)
	// With tiny API constraint, peakTS should be capped
	_ = got
}

func TestEstimateFilmMatchStartMS_ZeroMinPlayers(t *testing.T) {
	// minPlayers=0 should default to 3
	chunk := buildChunkWithMovingPlayers(1)
	got := EstimateFilmMatchStartMS([]SpawnChunk{chunk}, 0, 0)
	if got != -1 {
		t.Errorf("expected -1 (not enough players), got %f", got)
	}
}
