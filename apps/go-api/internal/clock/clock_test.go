package clock

import (
	"testing"
	"time"
)

func TestSystem_NowReturnsRecent(t *testing.T) {
	c := System{}
	before := time.Now()
	got := c.Now()
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Errorf("System.Now() = %v, hors borne [%v, %v]", got, before, after)
	}
}

func TestFake_NowReturnsConfiguredTime(t *testing.T) {
	want := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	c := Fake{T: want}
	got := c.Now()
	if !got.Equal(want) {
		t.Errorf("Fake.Now() = %v, want %v", got, want)
	}
}

func TestFake_DeterministicAcrossCalls(t *testing.T) {
	want := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	c := Fake{T: want}
	for i := 0; i < 5; i++ {
		if got := c.Now(); !got.Equal(want) {
			t.Errorf("appel %d : got %v, want %v", i, got, want)
		}
	}
}

// Vérifie que l'interface Clock est implémentée par System et Fake.
var (
	_ Clock = System{}
	_ Clock = Fake{}
)
