package filmdec

import "testing"

func bp(slot uint32, ms int, x, y float32) BipedPosition {
	return BipedPosition{Slot: slot, TimestampUS: uint64(ms) * 1000, X: x, Y: y}
}

// TestDropIsolated reproduit la signature réelle des faux positifs de 000d5950 : un point
// unique séparé de plus de 60 s du reste de son slot, en tête comme en queue de séquence.
func TestDropIsolated(t *testing.T) {
	in := []BipedPosition{
		bp(513, 0, 1, 1),
		bp(513, 16, 1, 1),
		bp(513, 32, 1, 1),
		bp(513, 200_000, -37, -55), // isolé de 200 s en fin de slot -> écarté
		bp(577, 0, -12, -54),       // isolé en tête de slot (170 s avant la vie) -> écarté
		bp(577, 170_000, 5, 12),
		bp(577, 170_016, 5, 12),
	}
	got := DropIsolated(in, DefaultIsolationGapMS)
	if len(got) != 5 {
		t.Fatalf("attendu 5 positions conservées, obtenu %d (%+v)", len(got), got)
	}
	for _, p := range got {
		if p.X < -10 {
			t.Errorf("aberration conservée: %+v", p)
		}
	}
	// slot réduit à un seul échantillon : aucun voisin -> écarté
	if got := DropIsolated([]BipedPosition{bp(600, 5000, 1, 1)}, DefaultIsolationGapMS); len(got) != 0 {
		t.Errorf("un slot à un seul échantillon devrait disparaître, obtenu %+v", got)
	}
	// filtre désactivé : rien n'est touché
	if got := DropIsolated(in, 0); len(got) != len(in) {
		t.Errorf("gapMS=0 devrait désactiver le filtre, obtenu %d/%d", len(got), len(in))
	}
}

// TestDropTeleports : un saut instantané est écarté, la trajectoire continue conservée, et
// une série longue de rejets réancre le filtre (pas de slot condamné par une mauvaise ancre).
func TestDropTeleports(t *testing.T) {
	in := []BipedPosition{
		bp(513, 0, 0, 0),
		bp(513, 16, 0.1, 0), // 6,25 m/s : normal
		bp(513, 32, 60, 0),  // ~3700 m/s : téléportation
		bp(513, 48, 0.2, 0), // retour sur la trajectoire
	}
	got := DropTeleports(in, DefaultMaxSpeedMPS)
	if len(got) != 3 {
		t.Fatalf("attendu 3 positions, obtenu %d (%+v)", len(got), got)
	}
	for _, p := range got {
		if p.X > 1 {
			t.Errorf("téléportation conservée: %+v", p)
		}
	}

	// 4 rejets d'affilée : le 4e est accepté (réancrage) — sinon une ancre fausse
	// supprimerait tout le reste du slot.
	seq := []BipedPosition{bp(600, 0, 0, 0)}
	for i := 1; i <= 4; i++ {
		seq = append(seq, bp(600, i*16, float32(100*i), 0))
	}
	if got := DropTeleports(seq, DefaultMaxSpeedMPS); len(got) != 2 {
		t.Errorf("attendu 2 positions (ancre + réancrage après %d rejets), obtenu %d", maxRejectStreak, len(got))
	}

	if got := DropTeleports(in, 0); len(got) != len(in) {
		t.Errorf("maxSpeed=0 devrait désactiver le filtre, obtenu %d/%d", len(got), len(in))
	}
}
