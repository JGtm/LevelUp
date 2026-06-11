package sharedprovider

import (
	"expvar"
	"testing"
)

// TestSnapshot_ReturnsValidShape : Snapshot() est lecture seule et retourne
// toujours un état canonique + des compteurs non négatifs, quel que soit
// l'historique du process de test (d'autres tests peuvent avoir swappé avant).
func TestSnapshot_ReturnsValidShape(t *testing.T) {
	s := Snapshot()
	if s.SwapsToRW < 0 || s.SwapsToRO < 0 || s.ReadersInUse < 0 {
		t.Fatalf("compteurs négatifs: %+v", s)
	}
	switch s.State {
	case "unknown", "ro", "draining", "rw", "reopening", "error", "closed":
	default:
		t.Fatalf("état inattendu %q", s.State)
	}
}

// TestMapInt vérifie la lecture d'une clé int d'un expvar.Map sans toucher aux
// compteurs globaux du package (déterministe).
func TestMapInt(t *testing.T) {
	m := new(expvar.Map).Init()
	m.Add("k", 7)
	if got := mapInt(m, "k"); got != 7 {
		t.Fatalf("mapInt(present) = %d, want 7", got)
	}
	if got := mapInt(m, "absent"); got != 0 {
		t.Fatalf("mapInt(absent) = %d, want 0", got)
	}
	if got := mapInt(nil, "k"); got != 0 {
		t.Fatalf("mapInt(nil) = %d, want 0", got)
	}
}
