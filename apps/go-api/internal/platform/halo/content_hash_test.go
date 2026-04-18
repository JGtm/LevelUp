package halo

import (
	"testing"
)

func TestContentHash_Deterministic(t *testing.T) {
	data := []byte("hello world")
	h1 := ContentHash(data)
	h2 := ContentHash(data)
	if h1 != h2 {
		t.Fatal("expected same hash for same data")
	}
	if len(h1) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(h1))
	}
}

func TestContentHash_Empty(t *testing.T) {
	h := ContentHash([]byte{})
	if h == "" {
		t.Fatal("expected non-empty hash for empty input")
	}
}

func TestContentHash_Different(t *testing.T) {
	h1 := ContentHash([]byte("a"))
	h2 := ContentHash([]byte("b"))
	if h1 == h2 {
		t.Fatal("expected different hashes")
	}
}
