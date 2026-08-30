package teammates

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// classKills projette les classes d'un joueur en (class -> kills).
func classKills(entries []domain.FragClassEntry) map[string]int {
	out := map[string]int{}
	for _, c := range entries {
		out[c.Class] = c.Kills
	}
	return out
}

// TestSquadFragClassesByPlayer_PerPlayerSplit : l'adapter Escouade ventile les rows
// weapon_kills PAR gamertag (via xuid), agrège les compteurs kill-type de la série de
// performance de chaque joueur et produit une FragDistribution au niveau classe par
// joueur. Cap OFF (Infinite) : pas de spartan, Mêlée feuille.
func TestSquadFragClassesByPlayer_PerPlayerSplit(t *testing.T) {
	players := []string{"Alice", "Bob"}
	xuidByPlayer := map[string]string{"Alice": "xa", "Bob": "xb"}
	rows := []port.WeaponKillRow{
		{XUID: "xa", WeaponID: 1, Kills: 6, Class: "shoulder", Role: "precision"},
		{XUID: "xb", WeaponID: 2, Kills: 4, Class: "heavy", Role: "sniper"},
		{XUID: "zz", WeaponID: 3, Kills: 9, Class: "shoulder", Role: "automatic"}, // xuid inconnu → ignoré
	}
	perf := map[string][]domain.SquadPerformanceSeriesPoint{
		"Alice": {{Kills: 10, MeleeKills: intPtr(2), GrenadeKills: intPtr(1)}},
		"Bob":   {{Kills: 8, MeleeKills: intPtr(1)}},
	}
	out := squadFragClassesByPlayer(squadFragInputs{
		rows: rows, playersOrdered: players, xuidByPlayer: xuidByPlayer, perf: perf,
	})
	if out == nil {
		t.Fatal("want per-player classes, got nil")
	}
	alice := classKills(out["Alice"])
	// Alice : shoulder=6 (gun), melee=2, grenade=1 ; total=10 → unattr=1.
	if alice["shoulder"] != 6 || alice["melee"] != 2 || alice["grenade"] != 1 || alice["unattributed"] != 1 {
		t.Errorf("Alice = %+v, want shoulder6/melee2/grenade1/unattr1", alice)
	}
	if _, ok := alice["spartan_ability"]; ok {
		t.Error("cap off : aucune classe spartan_ability pour Alice")
	}
	bob := classKills(out["Bob"])
	// Bob : heavy=4 (gun), melee=1 ; total=8 → unattr=3.
	if bob["heavy"] != 4 || bob["melee"] != 1 || bob["unattributed"] != 3 {
		t.Errorf("Bob = %+v, want heavy4/melee1/unattr3", bob)
	}
}

// TestSquadFragClassesByPlayer_MechMergeCapOn : cap ON (H5) — mechByGT fusionne les
// mécaniques natives par gamertag (assassinats + capacités spartanes), produisant la
// classe Capacités spartanes et le split Mêlée.
func TestSquadFragClassesByPlayer_MechMergeCapOn(t *testing.T) {
	players := []string{"Alice"}
	xuidByPlayer := map[string]string{"Alice": "xa"}
	rows := []port.WeaponKillRow{
		{XUID: "xa", WeaponID: 1, Kills: 10, Class: "shoulder", Role: "automatic"},
	}
	perf := map[string][]domain.SquadPerformanceSeriesPoint{
		"Alice": {{Kills: 20, MeleeKills: intPtr(3), GrenadeKills: intPtr(2)}},
	}
	mechByGT := map[string]port.KillMechanicsRow{
		"Alice": {Assassinations: 2, GroundPound: 1, ShoulderBash: 1},
	}
	out := squadFragClassesByPlayer(squadFragInputs{
		rows: rows, playersOrdered: players, xuidByPlayer: xuidByPlayer, perf: perf,
		mechByGT: mechByGT, hasMechanics: true,
	})
	alice := classKills(out["Alice"])
	// melee classe = melee(3)+assass(2)=5 ; spartan = ground(1)+shoulder(1)=2.
	if alice["melee"] != 5 {
		t.Errorf("melee = %d, want 5 (3 mêlée + 2 assassinats disjoints)", alice["melee"])
	}
	if alice["spartan_ability"] != 2 {
		t.Errorf("spartan_ability = %d, want 2", alice["spartan_ability"])
	}
}

// TestAggregateFragCounts : somme les compteurs kill-type sur la série de performance
// (Total = Σ Kills, Melee/Grenade depuis les pointeurs nil-safe).
func TestAggregateFragCounts(t *testing.T) {
	pts := []domain.SquadPerformanceSeriesPoint{
		{Kills: 5, MeleeKills: intPtr(1), GrenadeKills: intPtr(2)},
		{Kills: 3, MeleeKills: intPtr(1)}, // GrenadeKills nil → ignoré
	}
	c := aggregateFragCounts(pts)
	if c.Total != 8 || c.Melee != 2 || c.Grenade != 2 {
		t.Errorf("counts = %+v, want Total8/Melee2/Grenade2", c)
	}
}
