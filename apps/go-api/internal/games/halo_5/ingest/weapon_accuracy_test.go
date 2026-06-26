package ingest

import (
	"strconv"
	"testing"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/persist"
)

func weaponDrop(playerGT, weaponID string, sf, sl, timeMs int) canonical.MatchEvent {
	ev := canonical.MatchEvent{Type: canonical.MatchEventWeaponDrop, TimeMs: timeMs}
	if playerGT != "" {
		ev.Player = &canonical.PlayerIdentity{Gamertag: playerGT}
	}
	if weaponID != "" {
		ev.Weapon = &canonical.AssetReference{Kind: "weapon", ID: weaponID}
	}
	f, l := sf, sl
	ev.ShotsFired = &f
	ev.ShotsLanded = &l
	return ev
}

func TestMapWeaponAccuracy_AggregatesPerPlayerWeapon(t *testing.T) {
	pickupWithShots := weaponDrop("JGtm", "100", 99, 99, 9999)
	pickupWithShots.Type = canonical.MatchEventWeaponPickup // pickup ≠ drop → jamais compté

	events := []canonical.MatchEvent{
		weaponDrop("JGtm", "100", 10, 4, 1000),       // arme 100
		weaponDrop("JGtm", "100", 5, 3, 5000),        // même (joueur, arme) → cumul
		weaponDrop("JGtm", "200", 8, 2, 6000),        // autre arme
		weaponDrop("Madina97294", "100", 7, 7, 7000), // autre joueur
		weaponDrop("JGtm", "100", 0, 0, 8000),        // 0 tir → ignoré (somme inchangée)
		weaponDrop("JGtm", "0", 3, 1, 9000),          // arme 0 → ignorée
		weaponDrop("", "100", 3, 1, 9500),            // sans joueur → ignoré
		pickupWithShots,
	}
	resolve := fakeResolver(map[string]string{"JGtm": "xJ", "Madina97294": "xM"})

	rows := MapWeaponAccuracy("m1", events, resolve)

	if len(rows) != 3 {
		t.Fatalf("rows: %d, attendu 3 — %+v", len(rows), rows)
	}
	byKey := map[string]persist.WeaponAccuracyInsert{}
	for _, r := range rows {
		if r.MatchID != "m1" {
			t.Errorf("match_id non propagé: %+v", r)
		}
		byKey[r.XUID+"/"+strconv.FormatUint(r.WeaponID, 10)] = r
	}
	if r := byKey["xJ/100"]; r.ShotsFired != 15 || r.ShotsLanded != 7 || r.Drops != 2 {
		t.Errorf("(xJ,100) = %+v, attendu 15/7 drops=2", r)
	}
	if r := byKey["xJ/200"]; r.ShotsFired != 8 || r.ShotsLanded != 2 || r.Drops != 1 {
		t.Errorf("(xJ,200) = %+v, attendu 8/2 drops=1", r)
	}
	if r := byKey["xM/100"]; r.ShotsFired != 7 || r.ShotsLanded != 7 || r.Drops != 1 {
		t.Errorf("(xM,100) = %+v, attendu 7/7 drops=1", r)
	}
}

func TestMapWeaponAccuracy_EmptyWhenNoFiredDrops(t *testing.T) {
	events := []canonical.MatchEvent{
		weaponDrop("JGtm", "100", 0, 0, 1000), // jamais tiré
		{Type: canonical.MatchEventKill, TimeMs: 2000},
	}
	if rows := MapWeaponAccuracy("m1", events, fakeResolver(map[string]string{"JGtm": "xJ"})); rows != nil {
		t.Fatalf("attendu nil (aucune arme tirée), obtenu %+v", rows)
	}
}
