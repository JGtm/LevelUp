package analysis

import "testing"

// NOTE F12 : GetTiming (et la table WeaponTiming*) sont RELOCALISABLES vers
// games/halo_infinite/film au chantier K (F12). Ce test suit la fonction —
// déplacer avec elle le cas échéant.

func TestGetTiming_KnownReturnsTableEntry(t *testing.T) {
	// Cherche une arme dont le timing DIFFÈRE du défaut : garantit que la table
	// est réellement consultée (pas un fallback silencieux sur DefaultTiming).
	var id uint64
	var want WeaponTiming
	found := false
	for wid, tm := range WeaponTimingByID {
		if tm != DefaultTiming {
			id, want, found = wid, tm, true
			break
		}
	}
	if !found {
		t.Fatal("WeaponTimingByID ne contient aucune arme au timing != défaut (table vide ?)")
	}
	if got := GetTiming(id); got != want {
		t.Errorf("GetTiming(%d) = %+v, want %+v (table)", id, got, want)
	}
}

func TestGetTiming_UnknownReturnsDefault(t *testing.T) {
	const improbableID uint64 = 0xFFFFFFFFFFFFFFFF
	if got := GetTiming(improbableID); got != DefaultTiming {
		t.Errorf("GetTiming(inconnu) = %+v, want DefaultTiming %+v", got, DefaultTiming)
	}
}
