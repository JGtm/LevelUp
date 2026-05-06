package duckdb

import (
	"testing"
)

func TestUBigint_Scan_Uint64(t *testing.T) {
	// Pour les valeurs > INT64_MAX, on calcule le want depuis le src via le
	// reinterpret bit-à-bit (les littéraux >INT64_MAX overflowent int64 au
	// compile-time, mais uint64 -> int64 en runtime est OK).
	bit63Set := uint64(0xf41fcfa642c9679f)
	cases := []struct {
		name string
		src  uint64
		want int64
	}{
		{"zero", 0, 0},
		{"small", 42, 42},
		{"max int64", 1<<63 - 1, 1<<63 - 1},
		{"bit63 set (filmshell hash typique)", bit63Set, int64(bit63Set)}, //nolint:gosec // bit-preserving
		{"max uint64", ^uint64(0), -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var u UBigint
			if err := u.Scan(c.src); err != nil {
				t.Fatalf("Scan(uint64=%d): unexpected err %v", c.src, err)
			}
			if u.Int64() != c.want {
				t.Errorf("Int64() = %d, want %d", u.Int64(), c.want)
			}
		})
	}
}

func TestUBigint_Scan_Int64(t *testing.T) {
	var u UBigint
	if err := u.Scan(int64(-42)); err != nil {
		t.Fatalf("Scan(int64): %v", err)
	}
	if u.Int64() != -42 {
		t.Errorf("Int64() = %d, want -42", u.Int64())
	}
}

func TestUBigint_Scan_Nil(t *testing.T) {
	u := UBigint(99) // valeur initiale non-zéro pour vérifier le reset
	if err := u.Scan(nil); err != nil {
		t.Fatalf("Scan(nil): %v", err)
	}
	if u.Int64() != 0 {
		t.Errorf("nil should reset to 0, got %d", u.Int64())
	}
}

func TestUBigint_Scan_UnsupportedType(t *testing.T) {
	var u UBigint
	if err := u.Scan("forty-two"); err == nil {
		t.Error("string source should error")
	}
	if err := u.Scan(42.0); err == nil {
		t.Error("float source should error")
	}
}

func TestUBigint_RoundTrip_BitPreserving(t *testing.T) {
	// Vérifie que la conversion uint64 -> int64 -> uint64 préserve les bits
	// pour les valeurs les plus problématiques (bit63 = 1).
	originals := []uint64{
		0xd791556542c9679f, // Mutilateur
		0xf408190f42c9679f, // MK50 Sidekick
		0x91eb16de42c9679f, // skin inconnu (sample diag)
		^uint64(0),         // 2^64 - 1
	}
	for _, orig := range originals {
		var u UBigint
		if err := u.Scan(orig); err != nil {
			t.Fatalf("Scan(%#x): %v", orig, err)
		}
		// reinterpret int64 -> uint64 doit retomber sur orig
		if got := uint64(u.Int64()); got != orig { //nolint:gosec
			t.Errorf("round-trip lost bits: orig=%#x got=%#x", orig, got)
		}
	}
}

func TestNullableUBigint_Scan(t *testing.T) {
	t.Run("nil source", func(t *testing.T) {
		var n NullableUBigint
		if err := n.Scan(nil); err != nil {
			t.Fatalf("Scan(nil): %v", err)
		}
		if n.Valid {
			t.Error("Valid should be false on nil")
		}
	})
	t.Run("uint64 source", func(t *testing.T) {
		var n NullableUBigint
		if err := n.Scan(uint64(0xd791556542c9679f)); err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if !n.Valid {
			t.Error("Valid should be true on non-nil")
		}
		if uint64(n.Value.Int64()) != 0xd791556542c9679f { //nolint:gosec
			t.Errorf("value mismatch")
		}
	})
}
