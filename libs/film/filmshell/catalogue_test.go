package filmshell

import "testing"

// NOTE F12 : GetTiming (et la table WeaponTiming*) sont RELOCALISABLES vers
// la brique film au chantier K (F12). Ce test suit la fonction —
// déplacer avec elle le cas échéant.

func TestGetTiming_KnownReturnsTableEntry(t *testing.T) {
	// Cherche une arme dont le timing DIFFÈRE du défaut : garantit que la table
	// est réellement consultée (pas un fallback silencieux sur defaultTiming).
	var id uint64
	var want WeaponTiming
	found := false
	for wid, tm := range weaponTimingByID {
		if tm != defaultTiming {
			id, want, found = wid, tm, true
			break
		}
	}
	if !found {
		t.Fatal("weaponTimingByID ne contient aucune arme au timing != défaut (table vide ?)")
	}
	if got := GetTiming(id); got != want {
		t.Errorf("GetTiming(%d) = %+v, want %+v (table)", id, got, want)
	}
}

func TestGetTiming_UnknownReturnsDefault(t *testing.T) {
	const improbableID uint64 = 0xFFFFFFFFFFFFFFFF
	if got := GetTiming(improbableID); got != defaultTiming {
		t.Errorf("GetTiming(inconnu) = %+v, want defaultTiming %+v", got, defaultTiming)
	}
}
