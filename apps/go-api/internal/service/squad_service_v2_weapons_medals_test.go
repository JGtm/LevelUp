package service

import (
	"testing"

	"levelup/go-api/internal/port"
)

func TestBuildWeaponsTable_AggregatesByWeaponAndPlayer(t *testing.T) {
	t.Parallel()
	rows := []port.WeaponKillRow{
		{XUID: "x_p1", WeaponID: 100, Kills: 10, Label: "BR75"},
		{XUID: "x_p1", WeaponID: 100, Kills: 5, Label: "BR75"}, // meme arme : sommer
		{XUID: "x_p2", WeaponID: 100, Kills: 8, Label: "BR75"},
		{XUID: "x_p1", WeaponID: 200, Kills: 3, Label: "Sidekick"},
	}
	out := BuildWeaponsTable(rows, nil, 0)
	if len(out) != 2 {
		t.Fatalf("want 2 weapons aggregated, got %d", len(out))
	}
	// Tri par total desc : BR75 (23) > Sidekick (3)
	if out[0].WeaponID != 100 || out[0].Total != 23 {
		t.Errorf("first weapon want BR75/23, got id=%d total=%d", out[0].WeaponID, out[0].Total)
	}
	if out[0].KillsByXUID["x_p1"] != 15 {
		t.Errorf("BR75 x_p1 want 15, got %d", out[0].KillsByXUID["x_p1"])
	}
	if out[0].KillsByXUID["x_p2"] != 8 {
		t.Errorf("BR75 x_p2 want 8, got %d", out[0].KillsByXUID["x_p2"])
	}
	if out[1].WeaponID != 200 || out[1].Total != 3 {
		t.Errorf("second weapon want Sidekick/3, got id=%d total=%d", out[1].WeaponID, out[1].Total)
	}
}

func TestBuildWeaponsTable_RemapsXUIDToGamertag(t *testing.T) {
	t.Parallel()
	rows := []port.WeaponKillRow{
		{XUID: "x_p1", WeaponID: 100, Kills: 10},
		{XUID: "x_p2", WeaponID: 100, Kills: 8},
	}
	xuidToGT := map[string]string{"x_p1": "main", "x_p2": "f1"}
	out := BuildWeaponsTable(rows, xuidToGT, 0)
	if len(out) != 1 {
		t.Fatalf("want 1 weapon, got %d", len(out))
	}
	if out[0].KillsByXUID["main"] != 10 {
		t.Errorf("main kills want 10, got %d", out[0].KillsByXUID["main"])
	}
	if out[0].KillsByXUID["f1"] != 8 {
		t.Errorf("f1 kills want 8, got %d", out[0].KillsByXUID["f1"])
	}
}

func TestBuildWeaponsTable_AppliesMinKillsFilter(t *testing.T) {
	t.Parallel()
	rows := []port.WeaponKillRow{
		{XUID: "x_p1", WeaponID: 100, Kills: 10},
		{XUID: "x_p1", WeaponID: 200, Kills: 2}, // total 2 < min 5
	}
	out := BuildWeaponsTable(rows, nil, 5)
	if len(out) != 1 {
		t.Fatalf("want 1 weapon (min 5), got %d", len(out))
	}
	if out[0].WeaponID != 100 {
		t.Errorf("want weapon 100, got %d", out[0].WeaponID)
	}
}

func TestBuildWeaponsTable_GrenadeMeleeSeparated(t *testing.T) {
	t.Parallel()
	rows := []port.WeaponKillRow{
		{XUID: "x_p1", WeaponID: 100, Kills: 10},
		{XUID: "x_p1", WeaponID: 100, Kills: 3, IsGrenadeMelee: true}, // separe
	}
	out := BuildWeaponsTable(rows, nil, 0)
	if len(out) != 2 {
		t.Fatalf("want 2 entries (grenade/melee separate), got %d", len(out))
	}
	// Tri par total desc : 10 puis 3
	if out[0].Total != 10 || out[0].IsGrenadeMelee {
		t.Errorf("[0] want primary 10, got %v", out[0])
	}
	if out[1].Total != 3 || !out[1].IsGrenadeMelee {
		t.Errorf("[1] want grenade 3, got %v", out[1])
	}
}

func TestBuildWeaponsTable_EmptyInput(t *testing.T) {
	t.Parallel()
	if got := BuildWeaponsTable(nil, nil, 0); got != nil {
		t.Errorf("nil rows: want nil, got %v", got)
	}
}

func TestBuildMedalsGallery_GroupsByMatchAndPlayer(t *testing.T) {
	t.Parallel()
	rows := []port.MedalRow{
		{XUID: "x_p1", MatchID: "m1", MedalID: 100, Count: 1, Label: "Killing Spree"},
		{XUID: "x_p1", MatchID: "m1", MedalID: 200, Count: 3, Label: "Headshot"},
		{XUID: "x_p2", MatchID: "m1", MedalID: 100, Count: 2, Label: "Killing Spree"},
		{XUID: "x_p1", MatchID: "m2", MedalID: 100, Count: 1, Label: "Killing Spree"},
	}
	out := BuildMedalsGallery(rows, nil, []string{"m1", "m2"})
	if len(out) != 2 {
		t.Fatalf("want 2 match entries, got %d", len(out))
	}
	if out[0].MatchID != "m1" {
		t.Errorf("[0] want m1, got %s", out[0].MatchID)
	}
	// m1 a 2 joueurs (x_p1 et x_p2)
	if len(out[0].MedalsByXUID) != 2 {
		t.Errorf("m1 want 2 players, got %d", len(out[0].MedalsByXUID))
	}
	// x_p1 sur m1 : 2 medailles, triees par count desc (Headshot 3 > KillingSpree 1)
	x_p1Medals := out[0].MedalsByXUID["x_p1"]
	if len(x_p1Medals) != 2 {
		t.Fatalf("x_p1 m1 want 2 medals, got %d", len(x_p1Medals))
	}
	if x_p1Medals[0].MedalID != 200 || x_p1Medals[0].Count != 3 {
		t.Errorf("x_p1 [0] want id=200 count=3 (Headshot first), got %+v", x_p1Medals[0])
	}
}

func TestBuildMedalsGallery_RemapsXUIDToGamertag(t *testing.T) {
	t.Parallel()
	rows := []port.MedalRow{
		{XUID: "x_p1", MatchID: "m1", MedalID: 100, Count: 1},
	}
	xuidToGT := map[string]string{"x_p1": "main"}
	out := BuildMedalsGallery(rows, xuidToGT, nil)
	if len(out) != 1 {
		t.Fatalf("want 1 entry, got %d", len(out))
	}
	if _, ok := out[0].MedalsByXUID["main"]; !ok {
		t.Errorf("expected key 'main' (remapped), got keys %v", out[0].MedalsByXUID)
	}
}

func TestBuildMedalsGallery_MatchOrderRespected(t *testing.T) {
	t.Parallel()
	rows := []port.MedalRow{
		{XUID: "x_p1", MatchID: "m_b", MedalID: 100, Count: 1},
		{XUID: "x_p1", MatchID: "m_a", MedalID: 100, Count: 1},
	}
	// matchOrder fourni : m_b first, m_a second
	out := BuildMedalsGallery(rows, nil, []string{"m_b", "m_a"})
	if out[0].MatchID != "m_b" || out[1].MatchID != "m_a" {
		t.Errorf("matchOrder respected want [m_b, m_a], got [%s, %s]",
			out[0].MatchID, out[1].MatchID)
	}
}

func TestBuildMedalsGallery_EmptyInput(t *testing.T) {
	t.Parallel()
	if got := BuildMedalsGallery(nil, nil, nil); got != nil {
		t.Errorf("nil rows: want nil, got %v", got)
	}
}
