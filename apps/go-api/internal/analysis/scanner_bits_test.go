package analysis

import "testing"

// ---------------------------------------------------------------------------
// readBitsUint64
// ---------------------------------------------------------------------------

func TestReadBitsUint64_SingleByte(t *testing.T) {
	data := []byte{0b10110000}
	got := readBitsUint64(data, 0, 4)
	if got != 11 {
		t.Errorf("expected 11, got %d", got)
	}
}

func TestReadBitsUint64_CrossByte(t *testing.T) {
	data := []byte{0xFF, 0x00}
	got := readBitsUint64(data, 4, 8)
	if got != 0xF0 {
		t.Errorf("expected 0xF0, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// readBitsUint8
// ---------------------------------------------------------------------------

func TestReadBitsUint8_FullByte(t *testing.T) {
	data := []byte{0xAB}
	got := readBitsUint8(data, 0, 8)
	if got != 0xAB {
		t.Errorf("expected 0xAB, got 0x%X", got)
	}
}

// ---------------------------------------------------------------------------
// hasSuffix
// ---------------------------------------------------------------------------

func TestHasSuffix_Match(t *testing.T) {
	wb := [8]byte{0, 0, 0, 0, 0xDE, 0xAD, 0xBE, 0xEF}
	suffix := [4]byte{0xDE, 0xAD, 0xBE, 0xEF}
	if !hasSuffix(wb, suffix) {
		t.Error("expected match")
	}
}

func TestHasSuffix_NoMatch(t *testing.T) {
	wb := [8]byte{0, 0, 0, 0, 1, 2, 3, 4}
	suffix := [4]byte{5, 6, 7, 8}
	if hasSuffix(wb, suffix) {
		t.Error("expected no match")
	}
}

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

// ---------------------------------------------------------------------------
// matchMarkerAt
// ---------------------------------------------------------------------------

func TestMatchMarkerAt_TooShort(t *testing.T) {
	data := []byte{0xFF}
	if matchMarkerAt(data, 0) {
		t.Error("expected false for short data")
	}
}
