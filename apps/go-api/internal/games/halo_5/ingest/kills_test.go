package ingest

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

func kill(killerGT, victimGT, weaponID string, timeMs int) canonical.MatchEvent {
	ev := canonical.MatchEvent{Type: canonical.MatchEventKill, TimeMs: timeMs}
	if killerGT != "" {
		ev.Killer = &canonical.PlayerIdentity{Gamertag: killerGT}
	}
	if victimGT != "" {
		ev.Victim = &canonical.PlayerIdentity{Gamertag: victimGT}
	}
	if weaponID != "" {
		ev.Weapon = &canonical.AssetReference{Kind: "weapon", ID: weaponID}
	}
	return ev
}

func TestMapKillEvents_PairsAndWeapons(t *testing.T) {
	events := []canonical.MatchEvent{
		kill("Madina97294", "JGtm", "12345", 5000),
		kill("JGtm", "Madina97294", "", 8000),   // pas d'arme → pas de weapon_kill
		kill("", "Madina97294", "999", 10000),   // sans attaquant (env) → ignoré
		kill("Madina97294", "", "12345", 11000), // sans victime → ignoré
		medal("Madina97294", "100", 1000),       // non-kill → ignoré
	}
	resolve := fakeResolver(map[string]string{"Madina97294": "xA", "JGtm": "xB"})

	pairs, weapons := MapKillEvents("m1", events, resolve)

	if len(pairs) != 2 {
		t.Fatalf("paires: %d, attendu 2 — %+v", len(pairs), pairs)
	}
	if pairs[0].KillerXUID != "xA" || pairs[0].VictimXUID != "xB" || pairs[0].KillerGamertag != "Madina97294" {
		t.Errorf("1re paire mal mappée: %+v", pairs[0])
	}
	if pairs[0].Count != 1 || pairs[0].TimeMS != 5000 {
		t.Errorf("forme par-kill attendue (count=1, time_ms): %+v", pairs[0])
	}
	if len(weapons) != 1 {
		t.Fatalf("weapon_kills: %d, attendu 1 — %+v", len(weapons), weapons)
	}
	if weapons[0].WeaponID == nil || *weapons[0].WeaponID != 12345 || weapons[0].XUID != "xA" || weapons[0].TimeMS != 5000 {
		t.Errorf("weapon_kill mal mappé: %+v", weapons[0])
	}
	if weapons[0].Confidence != "native" || weapons[0].AttributionPath != "h5_native" {
		t.Errorf("attribution native attendue: %+v", weapons[0])
	}
}

// killKind construit un event kill avec arme ET mecanique (Kind) posee.
func killKind(killerGT, victimGT, weaponID string, timeMs int, kind canonical.KillKind) canonical.MatchEvent {
	ev := kill(killerGT, victimGT, weaponID, timeMs)
	ev.Kind = kind
	return ev
}

// La mecanique de kill (canonical.KillKind, deja derivee par h5KillKind) doit etre
// recopiee dans WeaponKillInsert.KillKind (capture Phase 1 — on cesse de la jeter).
func TestMapKillEvents_KillKindCaptured(t *testing.T) {
	events := []canonical.MatchEvent{
		killKind("A", "B", "111", 1000, canonical.KillKindWeapon),
		killKind("A", "B", "222", 2000, canonical.KillKindMelee),
		killKind("A", "B", "333", 3000, canonical.KillKindGroundPound),
		killKind("A", "B", "444", 4000, canonical.KillKindShoulderBash),
		killKind("A", "B", "555", 5000, canonical.KillKindAssassination),
	}
	_, weapons := MapKillEvents("m", events, fakeResolver(map[string]string{"A": "xA", "B": "xB"}))

	want := []string{"weapon", "melee", "groundpound", "shoulderbash", "assassination"}
	if len(weapons) != len(want) {
		t.Fatalf("weapon_kills: %d, attendu %d — %+v", len(weapons), len(want), weapons)
	}
	for i, w := range want {
		if weapons[i].KillKind != w {
			t.Errorf("kill %d: KillKind=%q, attendu %q", i, weapons[i].KillKind, w)
		}
	}
}

// Un kill event SANS mecanique (Kind vide, ex. chemin non-h5) laisse KillKind vide
// → le persist le mappera sur NULL (non-regression Infinite).
func TestMapKillEvents_NoKindLeavesEmpty(t *testing.T) {
	events := []canonical.MatchEvent{kill("A", "B", "111", 1000)} // Kind non pose
	_, weapons := MapKillEvents("m", events, fakeResolver(map[string]string{"A": "xA", "B": "xB"}))
	if len(weapons) != 1 {
		t.Fatalf("weapon_kills: %d, attendu 1", len(weapons))
	}
	if weapons[0].KillKind != "" {
		t.Errorf("KillKind attendu vide (=> NULL en base), got %q", weapons[0].KillKind)
	}
}

func TestMapKillEvents_NonNumericWeaponSkipped(t *testing.T) {
	events := []canonical.MatchEvent{kill("A", "B", "not-num", 1000)}
	pairs, weapons := MapKillEvents("m", events, fakeResolver(map[string]string{"A": "xA", "B": "xB"}))
	if len(pairs) != 1 {
		t.Fatalf("la paire doit exister même sans arme valide: %+v", pairs)
	}
	if len(weapons) != 0 {
		t.Fatalf("arme non numérique ignorée: %+v", weapons)
	}
}
