// Package ops — lying_bits_reset_cgo_test.go : reset des bits menteurs sur
// DuckDB :memory: (driver CGO requis). Vérifie détection (dry-run sans
// mutation), clear effectif, et idempotence.
package ops

import (
	"context"
	"testing"
)

func TestResetLyingBits(t *testing.T) {
	ctx := context.Background()
	shared := openDQTestShared(t)

	// m1 : bits events+weapons posés, tables vides → menteur events ET weapons.
	seedDQMatch(t, shared, "m1", "pl", "Quick Play", "p1", "Arena:Slayer", dqBitEvents|dqBitWeaponKills)
	// m2 : bit events posé MAIS highlight_events présent → PAS menteur.
	seedDQMatch(t, shared, "m2", "pl", "Quick Play", "p1", "Arena:Slayer", dqBitEvents)
	if _, err := shared.Exec(`INSERT INTO highlight_events (match_id) VALUES ('m2')`); err != nil {
		t.Fatalf("seed highlight m2: %v", err)
	}
	// m3 : events_loaded=TRUE sans highlight_events → menteur events_loaded.
	seedDQMatch(t, shared, "m3", "pl", "Quick Play", "p1", "Arena:Slayer", 0)
	if _, err := shared.Exec(`UPDATE match_registry SET events_loaded = TRUE WHERE match_id = 'm3'`); err != nil {
		t.Fatalf("seed events_loaded m3: %v", err)
	}

	// Dry-run : détecte 1/1/1 et ne mute RIEN.
	dry, err := ResetLyingBits(ctx, shared, true)
	if err != nil {
		t.Fatalf("ResetLyingBits dry: %v", err)
	}
	if dry.EventsBitsCleared != 1 || dry.WeaponsBitsCleared != 1 || dry.EventsLoadedCleared != 1 {
		t.Fatalf("dry counts = %d/%d/%d (attendu 1/1/1)",
			dry.EventsBitsCleared, dry.WeaponsBitsCleared, dry.EventsLoadedCleared)
	}
	if dry.Total() != 3 {
		t.Errorf("dry Total = %d (attendu 3)", dry.Total())
	}
	// Le bit events de m1 doit être TOUJOURS posé (dry-run = lecture seule).
	var bits int
	if err := shared.QueryRowContext(ctx,
		`SELECT backfill_completed FROM match_registry WHERE match_id = 'm1'`).Scan(&bits); err != nil {
		t.Fatalf("read m1 bits: %v", err)
	}
	if bits&dqBitEvents == 0 {
		t.Error("dry-run a muté le bit events de m1 (ne devrait pas)")
	}

	// Exécution : clear effectif, mêmes counts.
	run, err := ResetLyingBits(ctx, shared, false)
	if err != nil {
		t.Fatalf("ResetLyingBits run: %v", err)
	}
	if run.EventsBitsCleared != 1 || run.WeaponsBitsCleared != 1 || run.EventsLoadedCleared != 1 {
		t.Fatalf("run counts = %d/%d/%d (attendu 1/1/1)",
			run.EventsBitsCleared, run.WeaponsBitsCleared, run.EventsLoadedCleared)
	}

	// m1 : les deux bits menteurs sont clearés.
	if err := shared.QueryRowContext(ctx,
		`SELECT backfill_completed FROM match_registry WHERE match_id = 'm1'`).Scan(&bits); err != nil {
		t.Fatalf("read m1 bits post-run: %v", err)
	}
	if bits&dqBitEvents != 0 || bits&dqBitWeaponKills != 0 {
		t.Errorf("m1 bits = %d, events/weapons devraient être clearés", bits)
	}
	// m2 : bit events conservé (highlight_events présent, pas menteur).
	if err := shared.QueryRowContext(ctx,
		`SELECT backfill_completed FROM match_registry WHERE match_id = 'm2'`).Scan(&bits); err != nil {
		t.Fatalf("read m2 bits: %v", err)
	}
	if bits&dqBitEvents == 0 {
		t.Error("m2 bit events clearé à tort (highlight_events présent)")
	}
	// m3 : events_loaded remis à FALSE.
	var loaded bool
	if err := shared.QueryRowContext(ctx,
		`SELECT events_loaded FROM match_registry WHERE match_id = 'm3'`).Scan(&loaded); err != nil {
		t.Fatalf("read m3 events_loaded: %v", err)
	}
	if loaded {
		t.Error("m3 events_loaded toujours TRUE après reset")
	}

	// Idempotence : un second run ne trouve plus rien.
	again, err := ResetLyingBits(ctx, shared, false)
	if err != nil {
		t.Fatalf("ResetLyingBits run2: %v", err)
	}
	if again.Total() != 0 {
		t.Errorf("run2 Total = %d (attendu 0, idempotent)", again.Total())
	}
}
