package filmdec

import "testing"

// Vectors are the hand-computed acceptance vectors from the M1 RE workflow
// (adversarially verified against HaloInfinite.exe FUN_1406d6c7c / FUN_140c18a1c /
// FUN_140c18794). Buffers are padded to 8 bytes so the full-word path is exercised.

func TestReadBits_V1(t *testing.T) {
	br := NewBitReader([]byte{0xB5, 0x6A, 0xFF, 0x00, 0x12, 0x34, 0x56, 0x78})
	ns := []uint{4, 4, 8, 3, 13, 16}
	want := []uint64{0xB, 0x5, 0x6A, 0x7, 0x1F00, 0x1234}
	for i, n := range ns {
		if got := br.ReadBits(n); got != want[i] {
			t.Fatalf("ReadBits #%d (n=%d) = %#x, want %#x", i, n, got, want[i])
		}
	}
}

func TestReadBit_V2(t *testing.T) {
	br := NewBitReader([]byte{0xB5, 0x6A, 0xFF, 0, 0, 0, 0, 0})
	want := []bool{true, false, true, true, false, true, false, true, false}
	for i, w := range want {
		if got := br.ReadBit(); got != w {
			t.Fatalf("ReadBit #%d = %v, want %v", i, got, w)
		}
	}
}

func TestReadSignedVarWidth(t *testing.T) {
	cases := []struct {
		name string
		buf  []byte
		want int32
	}{
		{"V3 w8 neg", []byte{0x20, 0, 0, 0, 0, 0, 0, 0}, -128},
		{"V4 w16 pos", []byte{0x44, 0x8D, 0, 0, 0, 0, 0, 0}, 4660},
		{"V5 w16 neg", []byte{0x7F, 0xFF, 0x80, 0, 0, 0, 0, 0}, -2},
		// field 0xFFFFFF00 packed MSB-first after the 2-bit selector: byte3 = field[9..2] = 0b11000000 = 0xC0
		// (the M1 workflow vector wrote 0xFC here, an arithmetic slip; 0xFC decodes to -16, the faithful result).
		{"V6 w32 neg", []byte{0xBF, 0xFF, 0xFF, 0xC0, 0, 0, 0, 0}, -256},
		{"V6b w32 raw FC", []byte{0xBF, 0xFF, 0xFF, 0xFC, 0, 0, 0, 0}, -16},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NewBitReader(c.buf).ReadSignedVarWidth(); got != c.want {
				t.Fatalf("ReadSignedVarWidth = %d, want %d", got, c.want)
			}
		})
	}
}

func TestParseOptionalValue_V7(t *testing.T) {
	if got := ParseOptionalValue(NewBitReader([]byte{0x80, 0, 0, 0, 0, 0, 0, 0})); got != 0 {
		t.Fatalf("present flag set => %d, want 0", got)
	}
	if got := ParseOptionalValue(NewBitReader([]byte{0x05, 0x40, 0, 0, 0, 0, 0, 0})); got != 42 {
		t.Fatalf("present flag clear => %d, want 42", got)
	}
}
