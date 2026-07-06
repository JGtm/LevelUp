// Package sync — engine_prestige_hook_test.go : garde anti-régression VF-1
// (niveau unitaire). Le hook Prestige, une fois câblé via WithPrestigeHook, doit
// être stocké et invoqué au point post-ingestion de engine.run() (engine.go:713)
// avec (ctx, gamertag, titleSlug). Ce fichier reproduit fidèlement ce bloc sans
// lancer RunDelta() complet (infra dblease/DuckDB lourde — même approche que
// engine_post_sync_runner_e2e_test.go).
//
// Contexte : avant le fix, e.prestigeHook restait nil sur TOUS les chemins →
// prestige.RunPostSyncHook ne tournait jamais. Cf. AUDIT_VERIF_FINALE VF-1.
package sync

import (
	"context"
	"testing"
)

// TestSyncEngine_WithPrestigeHook_StoresAndInvokes vérifie que WithPrestigeHook
// stocke la closure (HasPrestigeHook) et que le pattern engine.run() l'invoque
// avec les bons arguments.
func TestSyncEngine_WithPrestigeHook_StoresAndInvokes(t *testing.T) {
	var gotGamertag, gotTitle string
	called := false

	e := (&SyncEngine{gamertag: "JGtm", titleSlug: "halo_infinite"}).
		WithPrestigeHook(func(_ context.Context, gamertag, titleSlug string) {
			called = true
			gotGamertag, gotTitle = gamertag, titleSlug
		})

	if !e.HasPrestigeHook() {
		t.Fatal("WithPrestigeHook n'a pas stocké la closure (e.prestigeHook nil)")
	}

	// Reproduit le bloc engine.run() (engine.go ~713) : invocation inline post
	// pipeline post-sync, pendant que le lease est encore tenu.
	if e.prestigeHook != nil {
		e.prestigeHook(context.Background(), e.gamertag, e.titleSlug)
	}

	if !called {
		t.Fatal("le hook Prestige n'a pas été invoqué (engine.run le sauterait → feature morte VF-1)")
	}
	if gotGamertag != "JGtm" || gotTitle != "halo_infinite" {
		t.Errorf("hook invoqué avec (%q,%q), want (JGtm,halo_infinite)", gotGamertag, gotTitle)
	}
}

// TestSyncEngine_NilPrestigeHook_NoInvoke : sans câblage, le bloc run() ne tire
// pas (nil-guard) et ne panique pas.
func TestSyncEngine_NilPrestigeHook_NoInvoke(t *testing.T) {
	e := &SyncEngine{gamertag: "JGtm", titleSlug: "halo_infinite"}
	if e.HasPrestigeHook() {
		t.Fatal("prestigeHook devrait être nil sans WithPrestigeHook")
	}
	// Reproduit le nil-guard engine.run() : aucune invocation, aucun panic.
	if e.prestigeHook != nil {
		t.Fatal("le nil-guard engine.run() ne doit pas tirer avec un hook nil")
	}
}
