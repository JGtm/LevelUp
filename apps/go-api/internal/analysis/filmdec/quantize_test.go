package filmdec

import (
	"math"
	"testing"
)

// step for bits=8, range +/-100 is 200/256 = 0.78125; with mid-bucket +0.5:
//
//	q=0   -> -100 + 0.78125*0.5      = -99.609375
//	q=128 -> -100 + 128*0.78125 + .. =   0.390625
//	q=255 -> -100 + 255*0.78125 + .. =  99.609375
func TestReadQuantizedVec3_World100(t *testing.T) {
	br := NewBitReader([]byte{0x00, 0x80, 0xFF, 0, 0, 0, 0, 0}) // q = 0, 128, 255 (8 bits each)
	v := br.ReadQuantizedVec3(8, QuantRangeWorld100)
	want := [3]float32{-99.609375, 0.390625, 99.609375}
	for i := range want {
		if math.Abs(float64(v[i]-want[i])) > 1e-3 {
			t.Fatalf("axis %d = %v, want %v", i, v[i], want[i])
		}
	}
}

// Boundaries: q=0 sits just above Min, q=max-1 just below Max (mid-bucket).
func TestReadQuantizedVec3_Bounds(t *testing.T) {
	br := NewBitReader([]byte{0x00, 0x00, 0x00, 0, 0, 0, 0, 0})
	v := br.ReadQuantizedVec3(8, QuantRangeUnit3)
	for i := range v {
		if v[i] <= -3 || v[i] >= 3 {
			t.Fatalf("axis %d = %v out of (-3,3)", i, v[i])
		}
	}
	if math.Abs(float64(v[0]-(-3+(6.0/256)*0.5))) > 1e-4 {
		t.Fatalf("q=0 axis = %v, want ~%v", v[0], -3+(6.0/256)*0.5)
	}
}
