package service

import (
	"testing"

	"levelup/go-api/internal/port"
)

// TestBuildKillsByRole verrouille l'agrégation des frags par rôle : somme par
// rôle, tri kills desc, exclusion des rows sans rôle et des sentinels grenade/melee.
func TestBuildKillsByRole(t *testing.T) {
	rows := []port.WeaponKillRow{
		{WeaponID: 100, Kills: 10, Role: "precision"},
		{WeaponID: 101, Kills: 5, Role: "precision"},
		{WeaponID: 102, Kills: 20, Role: "automatic"},
		{WeaponID: 103, Kills: 3, Role: ""},                             // arme non mappée → ignorée
		{WeaponID: 0, Kills: 99, Role: "grenade", IsGrenadeMelee: true}, // sentinel grenade → ignoré
	}
	out := buildKillsByRole(rows)
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2 (%+v)", len(out), out)
	}
	if out[0].Role != "automatic" || out[0].Kills != 20 {
		t.Errorf("out[0] = %+v, want automatic/20", out[0])
	}
	if out[1].Role != "precision" || out[1].Kills != 15 {
		t.Errorf("out[1] = %+v, want precision/15", out[1])
	}
	if buildKillsByRole(nil) != nil {
		t.Error("nil rows → nil")
	}
	if buildKillsByRole([]port.WeaponKillRow{{Role: "", Kills: 5}}) != nil {
		t.Error("rows sans rôle → nil attendu")
	}
}
