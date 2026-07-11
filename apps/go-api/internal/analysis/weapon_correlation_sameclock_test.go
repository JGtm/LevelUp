package analysis

import "testing"

// dataset hétérogène : 4 joueurs, kills firearm/melee/grenade, dégâts de plusieurs joueurs,
// dégâts avant/après le kill, joueur hors roster.
func TestCorrelateKillsSameClock(t *testing.T) {
	const (
		br75    = uint64(0x1111)
		ma40    = uint64(0x2222)
		fuelRod = uint64(0x3333)
	)
	xuidToPI := map[string]int{"A": 0, "B": 1, "C": 2} // "D" absent volontairement

	damages := []DamageEvent{
		{PlayerIndex: 0, WeaponID: ma40, TimeMS: 1000},    // A, vieux
		{PlayerIndex: 0, WeaponID: br75, TimeMS: 4900},    // A, fatal du kill@5000
		{PlayerIndex: 1, WeaponID: fuelRod, TimeMS: 4800}, // B
		{PlayerIndex: 0, WeaponID: ma40, TimeMS: 5200},    // A, APRÈS le kill@5000
		{PlayerIndex: 2, WeaponID: br75, TimeMS: 7000},    // C
	}
	kills := []Kill{
		{MatchID: "m", XUID: "A", TimeMS: 5000},                  // → br75 (dernier dégât A avant 5000)
		{MatchID: "m", XUID: "B", TimeMS: 4900},                  // → fuelRod (dégât B@4800)
		{MatchID: "m", XUID: "C", TimeMS: 6000},                  // → aucun (dégât C@7000 après)
		{MatchID: "m", XUID: "A", TimeMS: 900},                   // → aucun (aucun dégât A avant 900)
		{MatchID: "m", XUID: "A", TimeMS: 5300, IsMelee: true},   // melee → hors périmètre
		{MatchID: "m", XUID: "B", TimeMS: 5300, IsGrenade: true}, // grenade → hors périmètre
		{MatchID: "m", XUID: "D", TimeMS: 5000},                  // hors roster → aucun
	}

	got := CorrelateKillsSameClock(kills, damages, xuidToPI)
	if len(got) != len(kills) {
		t.Fatalf("attendu %d attributions, obtenu %d", len(kills), len(got))
	}

	type exp struct {
		weapon *uint64
		path   string
		delta  int
	}
	brP, frP := br75, fuelRod
	want := []exp{
		{&brP, AttributionPathDamageSource, 100}, // kill A@5000 - dégât@4900
		{&frP, AttributionPathDamageSource, 100}, // kill B@4900 - dégât@4800
		{nil, AttributionPathNone, 0},            // C : dégât après
		{nil, AttributionPathNone, 0},            // A@900 : rien avant
		{nil, AttributionPathNone, 0},            // melee
		{nil, AttributionPathNone, 0},            // grenade
		{nil, AttributionPathNone, 0},            // hors roster
	}

	for i, w := range want {
		g := got[i]
		if g.AttributionPath != w.path {
			t.Errorf("kill[%d] path = %q, attendu %q", i, g.AttributionPath, w.path)
		}
		if (g.WeaponID == nil) != (w.weapon == nil) {
			t.Errorf("kill[%d] WeaponID nil? = %v, attendu %v", i, g.WeaponID == nil, w.weapon == nil)
			continue
		}
		if g.WeaponID != nil && *g.WeaponID != *w.weapon {
			t.Errorf("kill[%d] WeaponID = %#x, attendu %#x", i, *g.WeaponID, *w.weapon)
		}
		if w.weapon != nil {
			if g.DeltaMS == nil || *g.DeltaMS != w.delta {
				t.Errorf("kill[%d] DeltaMS = %v, attendu %d", i, g.DeltaMS, w.delta)
			}
			if g.Confidence != confidenceHigh {
				t.Errorf("kill[%d] Confidence = %q, attendu %q", i, g.Confidence, confidenceHigh)
			}
		}
	}
}

// vérifie que le dégât le PLUS PROCHE avant le kill gagne (pas simplement le dernier inséré).
func TestCorrelateKillsSameClock_ClosestWins(t *testing.T) {
	xuidToPI := map[string]int{"A": 0}
	damages := []DamageEvent{
		{PlayerIndex: 0, WeaponID: 0xAAAA, TimeMS: 100},
		{PlayerIndex: 0, WeaponID: 0xBBBB, TimeMS: 4990}, // le plus proche de 5000
		{PlayerIndex: 0, WeaponID: 0xCCCC, TimeMS: 2000},
	}
	got := CorrelateKillsSameClock([]Kill{{XUID: "A", TimeMS: 5000}}, damages, xuidToPI)
	if got[0].WeaponID == nil || *got[0].WeaponID != 0xBBBB {
		t.Fatalf("attendu 0xBBBB (le plus proche), obtenu %v", got[0].WeaponID)
	}
}
