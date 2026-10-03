package weaponv3

import "testing"

// TestCanonWeaponID_KnownSidekick — le Mk51 Sidekick 0xf408190f42c9679f doit
// se canoniser sur son high-32 et être reconnu.
func TestCanonWeaponID_KnownSidekick(t *testing.T) {
	const sidekick = uint64(0xf408190f42c9679f)
	high, known := CanonWeaponID(sidekick)
	if high != 0xf408190f {
		t.Fatalf("high-32 attendu 0xf408190f, obtenu 0x%08x", high)
	}
	if !known {
		t.Fatalf("Mk51 Sidekick devrait être connu")
	}
	if name := WeaponName(high); name != "Mk51 Sidekick" {
		t.Fatalf("nom attendu 'Mk51 Sidekick', obtenu %q", name)
	}
}

// TestCanonWeaponID_Noise — un id bruit (high-32 absent du set) est inconnu.
func TestCanonWeaponID_Noise(t *testing.T) {
	const noise = uint64(0x18e1fea000000000)
	high, known := CanonWeaponID(noise)
	if known {
		t.Fatalf("l'id bruit 0x%016x ne devrait pas être connu (high 0x%08x)", noise, high)
	}
	if name := WeaponName(high); name != "" {
		t.Fatalf("nom attendu vide pour un high-32 inconnu, obtenu %q", name)
	}
}

// TestCanonWeaponID_Fold — deux ids partageant le même high-32 mais avec des
// suffixes bas différents (variantes) se canonisent sur la MÊME identité.
func TestCanonWeaponID_Fold(t *testing.T) {
	// Gravity Hammer 0x841ac5e5 : suffixe standard vs variante Rushdown Hammer.
	const gravityHammer = uint64(0x841ac5e542c9679f)
	const rushdownHammer = uint64(0x841ac5e5d8d07ca1)

	h1, known1 := CanonWeaponID(gravityHammer)
	h2, known2 := CanonWeaponID(rushdownHammer)

	if h1 != h2 {
		t.Fatalf("fold attendu : high identiques, obtenu 0x%08x vs 0x%08x", h1, h2)
	}
	if !known1 || !known2 {
		t.Fatalf("les deux variantes devraient être connues (known1=%v known2=%v)", known1, known2)
	}
	if h1 != 0x841ac5e5 {
		t.Fatalf("high attendu 0x841ac5e5, obtenu 0x%08x", h1)
	}
}
