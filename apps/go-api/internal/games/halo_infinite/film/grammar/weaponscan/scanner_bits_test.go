package weaponscan

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
// matchMarkerAt
// ---------------------------------------------------------------------------

func TestMatchMarkerAt_TooShort(t *testing.T) {
	data := []byte{0xFF}
	if matchMarkerAt(data, 0) {
		t.Error("expected false for short data")
	}
}
