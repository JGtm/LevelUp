// Package handlers — sync_handler_align_test.go : tests internes de l'alignement
// du sync manuel delta sur l'auto-sync (cooldown + EngineBuilder partagé).
//
// Tests INTERNES (package handlers, pas handlers_test) : ils accèdent aux méthodes
// non-exportées tryManualSyncCooldown / newPooledEngine sans driver un vrai
// RunDelta (pas de réseau, pas de DuckDB — NewSyncEngine est une construction pure).
package handlers

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	go_sync "levelup/go-api/internal/sync"
)

// TestTryManualSyncCooldown : 1er déclenchement passe, 2e dans la fenêtre bloqué
// (retry>0), clé différente indépendante.
func TestTryManualSyncCooldown(t *testing.T) {
	h := NewSyncHandler(&config.AppConfig{}, nil, nil, nil).WithManualSyncCooldown(time.Hour)

	if retry, ok := h.tryManualSyncCooldown("k"); !ok || retry != 0 {
		t.Fatalf("1er appel doit passer sans retry, got retry=%v ok=%v", retry, ok)
	}
	retry, ok := h.tryManualSyncCooldown("k")
	if ok || retry <= 0 {
		t.Fatalf("2e appel doit être bloqué avec retry>0, got retry=%v ok=%v", retry, ok)
	}
	if retry > time.Hour {
		t.Fatalf("retry borné par le cooldown (1h), got %v", retry)
	}
	if _, ok := h.tryManualSyncCooldown("autre"); !ok {
		t.Fatal("une clé différente doit passer indépendamment")
	}
}

// TestTryManualSyncCooldown_Disabled : cooldown=0 → toujours autorisé.
func TestTryManualSyncCooldown_Disabled(t *testing.T) {
	h := NewSyncHandler(&config.AppConfig{}, nil, nil, nil).WithManualSyncCooldown(0)
	for i := 0; i < 3; i++ {
		if retry, ok := h.tryManualSyncCooldown("k"); !ok || retry != 0 {
			t.Fatalf("cooldown désactivé doit toujours passer (appel %d), got retry=%v ok=%v", i, retry, ok)
		}
	}
}

// TestNewPooledEngine_UsesInjectedBuilder : avec un EngineBuilder injecté,
// newPooledEngine délègue au builder (même mécanique que l'auto-sync) et lui
// transmet (gamertag, xuid) — il ne reconstruit PAS un moteur legacy.
func TestNewPooledEngine_UsesInjectedBuilder(t *testing.T) {
	sentinel := &go_sync.SyncEngine{} // jamais exécuté : on teste la sélection, pas RunDelta
	var gotGT, gotXUID string
	h := NewSyncHandler(&config.AppConfig{RepoRoot: t.TempDir()}, nil, nil, nil).
		WithEngineBuilder(func(_ context.Context, gt, xuid string) *go_sync.SyncEngine {
			gotGT, gotXUID = gt, xuid
			return sentinel
		})

	got := h.newPooledEngine(context.Background(), "DemoPlayer", "xuid-1")
	if got != sentinel {
		t.Fatal("newPooledEngine doit retourner le moteur construit par le builder injecté")
	}
	if gotGT != "DemoPlayer" || gotXUID != "xuid-1" {
		t.Fatalf("(gamertag, xuid) non transmis au builder: gt=%q xuid=%q", gotGT, gotXUID)
	}
}

// TestNewPooledEngine_FallbackWhenNoBuilder : sans builder (tests / bootstrap sans
// pool) → fallback legacy newEngineFor, moteur non-nil, sans panique (settingsStore
// nil toléré : loader friends + media hook sont lazy / gardés).
func TestNewPooledEngine_FallbackWhenNoBuilder(t *testing.T) {
	h := NewSyncHandler(&config.AppConfig{RepoRoot: t.TempDir()}, nil, nil, nil)
	if got := h.newPooledEngine(context.Background(), "DemoPlayer", "xuid-1"); got == nil {
		t.Fatal("fallback legacy doit retourner un moteur non-nil")
	}
}

// TestNewEngineFor_WiresPrestigeHook : garde anti-régression VF-1. Quand le
// SyncHandler a reçu un hook Prestige (WithPrestigeHook), le SyncEngine construit
// par newEngineFor (chemin StartInitialSync) doit le porter. Ce test échoue si
// quelqu'un remet un stub qui jette le hook (le bug d'origine).
func TestNewEngineFor_WiresPrestigeHook(t *testing.T) {
	called := false
	h := NewSyncHandler(&config.AppConfig{RepoRoot: t.TempDir()}, nil, nil, nil).
		WithPrestigeHook(func(_ context.Context, _, _ string) { called = true })

	engine := h.newEngineFor("halo_infinite", "DemoPlayer", "xuid-1", nil)
	if engine == nil {
		t.Fatal("newEngineFor doit retourner un moteur non-nil")
	}
	if !engine.HasPrestigeHook() {
		t.Fatal("REGRESSION VF-1: le SyncEngine construit par newEngineFor ne porte pas le hook Prestige alors que WithPrestigeHook a été appelé (prestige.RunPostSyncHook ne tournera jamais sur le sync HTTP initial)")
	}
	_ = called // la closure n'est pas exécutée ici (pas de RunDelta) : on teste le câblage
}

// TestNewEngineFor_NoPrestigeHookWhenNotWired : sans WithPrestigeHook, le moteur
// ne porte pas de hook (pas de fuite).
func TestNewEngineFor_NoPrestigeHookWhenNotWired(t *testing.T) {
	h := NewSyncHandler(&config.AppConfig{RepoRoot: t.TempDir()}, nil, nil, nil)
	engine := h.newEngineFor("halo_infinite", "DemoPlayer", "xuid-1", nil)
	if engine == nil {
		t.Fatal("newEngineFor doit retourner un moteur non-nil")
	}
	if engine.HasPrestigeHook() {
		t.Error("PrestigeHook câblé alors que WithPrestigeHook pas appelé — fuite")
	}
}
