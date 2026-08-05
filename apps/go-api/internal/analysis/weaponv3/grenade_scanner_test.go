package weaponv3

import (
	"fmt"
	"testing"
)

// scanAllGrenadeThrows agrège ScanGrenadeThrows sur tous les chunks d'un match
// (estimateur µs ancré sur le start_ms de chaque chunk). Renvoie nil si cache/
// manifeste absent (skip-friendly), sinon le slice (possiblement vide).
func scanAllGrenadeThrows(t *testing.T, short8 string) ([]GrenadeThrow, bool) {
	t.Helper()
	manifest := loadFilmManifest(t, short8)
	if manifest == nil {
		return nil, false
	}
	var all []GrenadeThrow
	for _, c := range manifest.Chunks {
		chunk := loadCachedChunk(t, short8, fmt.Sprintf("chunk_%02d.bin", c.Index))
		if chunk == nil {
			continue
		}
		est := USEstimator(chunk, c.StartMS)
		all = append(all, ScanGrenadeThrows(chunk, est)...)
	}
	return all, true
}

// TestScanGrenadeThrows_AllowlistInvariant — sur 000d5950 (skip si absent), aucun
// panic et TOUT GrenadeThrow renvoyé porte un WeaponID de l'allowlist. Le match peut
// décoder 0 grenade : on n'asserte que l'invariant + non-panic.
func TestScanGrenadeThrows_AllowlistInvariant(t *testing.T) {
	throws, ok := scanAllGrenadeThrows(t, "000d5950")
	if !ok {
		t.Skip("cache/manifeste film 000d5950 absent — skip grenade robustesse")
	}
	for _, g := range throws {
		if !isGrenadeID(g.WeaponID) {
			t.Fatalf("GrenadeThrow hors allowlist : 0x%08x", g.WeaponID)
		}
	}
	t.Logf("grenade 000d5950 : %d lancers décodés (allowlist respectée)", len(throws))
}

// TestScanGrenadeThrows_Allowlist — vérifie le filtre allowlist sur un buffer
// synthétique : un marqueur suivi d'un id Frag (BE) est retenu, un id hors allowlist
// est rejeté, et la grenade décode bien vers la valeur Frag attendue.
func TestScanGrenadeThrows_Allowlist(t *testing.T) {
	// [marqueur 4c 0c 00][Frag B0 17 10 62 en BE] = 1 lancer valide.
	valid := []byte{0x4c, 0x0c, 0x00, 0xb0, 0x17, 0x10, 0x62, 0x00}
	got := ScanGrenadeThrows(valid, func(int) float64 { return 1234 })
	if len(got) != 1 {
		t.Fatalf("attendu 1 lancer Frag, obtenu %d", len(got))
	}
	if got[0].WeaponID != GrenadeFrag {
		t.Fatalf("WeaponID attendu Frag 0x%08x, obtenu 0x%08x", GrenadeFrag, got[0].WeaponID)
	}
	if got[0].TimeMS != 1234 {
		t.Fatalf("TimeMS attendu 1234 (de l'estimateur), obtenu %d", got[0].TimeMS)
	}

	// Marqueur suivi d'un object-id hors allowlist (0x18e1fea0, §C) = rejeté.
	junk := []byte{0x4c, 0x0c, 0x00, 0x18, 0xe1, 0xfe, 0xa0, 0x00}
	if got := ScanGrenadeThrows(junk, func(int) float64 { return 0 }); len(got) != 0 {
		t.Fatalf("object-id hors allowlist attendu 0 lancer, obtenu %d", len(got))
	}
}

// TestScanGrenadeThrows_Bonus — bonus : scanne quelques matchs cachés et rapporte
// les Frag/Plasma trouvés (réf §C : 53ce4390 CTF contient des Frags). N'échoue
// jamais sur l'absence de cache ; vérifie seulement l'invariant allowlist.
func TestScanGrenadeThrows_Bonus(t *testing.T) {
	for _, short8 := range []string{"53ce4390", "7344d24f"} {
		throws, ok := scanAllGrenadeThrows(t, short8)
		if !ok {
			t.Logf("grenade bonus %s : cache absent", short8)
			continue
		}
		byType := map[uint32]int{}
		for _, g := range throws {
			if !isGrenadeID(g.WeaponID) {
				t.Fatalf("GrenadeThrow hors allowlist (%s) : 0x%08x", short8, g.WeaponID)
			}
			byType[g.WeaponID]++
		}
		t.Logf("grenade bonus %s : Frag=%d Plasma=%d Shock=%d Spike=%d",
			short8, byType[GrenadeFrag], byType[GrenadePlasma], byType[GrenadeShock], byType[GrenadeSpike])
	}
}
