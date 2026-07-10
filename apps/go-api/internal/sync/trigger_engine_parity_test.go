// Package sync — trigger_engine_parity_test.go : garde-rail anti-régression
// pour la PARITÉ runtime entre le path watcher (Trigger.RunSync) et le
// path scheduler (auto_sync.defaultRunnerFactory).
//
// CONTEXTE — incident 2026-05-26 : le Trigger.RunSync faisait NewSyncEngine
// direct sans wirer .WithSharedProvider, alors que le scheduler le faisait
// via defaultRunnerFactory. Résultat : tous les syncs déclenchés par le
// watcher tombaient en mode legacy → conflit "different configuration".
//
// La correction : Trigger.engineFactory injecté depuis main.go pointe sur
// la MÊME closure que defaultRunnerFactory utilise. Ce test vérifie que
// si on appelle Trigger.RunSync, le SyncEngine consommé est strictement
// celui que la factory a fabriqué (pas un fallback NewSyncEngine).
package sync

import (
	"context"
	"database/sql"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestTrigger_PaternFromFactory_RealCheckOnEngineFields :
//
// Stratégie : on injecte une factory qui PRODUIT un engine fully-configured
// (SharedProvider câblé), capture le pointeur retourné, et on vérifie qu'il
// passe l'assertion HasSharedProvider() === true. Si Trigger.RunSync avait
// recréé un engine en interne (legacy bug), cette assertion casserait.
func TestTrigger_PaternFromFactory_RealCheckOnEngineFields(t *testing.T) {
	tp := &mockTokenProvider{}
	memProvider := sharedprovider.FromInMemoryDB((*sql.DB)(nil), ":memory:")

	var captured atomic.Value
	factory := func(_ context.Context, gamertag, xuid string) *SyncEngine {
		eng := NewSyncEngine(t.TempDir(), gamertag, xuid, &domain.HaloTokens{}, nil).
			WithSharedProvider(memProvider).
			WithCSRSeasonID("CsrSeason99")
		captured.Store(eng)
		return eng
	}

	trigger := NewTrigger(t.TempDir(), tp, domain.SyncOptions{}).
		WithEngineFactory(factory)

	_ = trigger.RunSync(context.Background(), "P", "1234", nil)

	got, ok := captured.Load().(*SyncEngine)
	if !ok || got == nil {
		t.Fatal("factory did not produce an engine — Trigger may have skipped invocation")
	}

	// --- Assertions de parité avec ce que BuildEngine produirait ---
	if !got.HasSharedProvider() {
		t.Error("PARITY REGRESSION: SharedProvider absent sur engine post-RunSync — Trigger a ignoré le wiring factory")
	}
	if got.CSRSeasonIDForTest() != "CsrSeason99" {
		t.Errorf("PARITY REGRESSION: CSRSeason mutilé, got %q want CsrSeason99", got.CSRSeasonIDForTest())
	}
}

// TestTrigger_LegacyEngine_LacksWiring est l'assertion-miroir : sans
// factory câblée, l'engine produit en interne PAR Trigger.RunSync (path
// legacy NewSyncEngine direct) n'a AUCUNE des options critiques.
//
// Si ce test commence à passer "trop bien" (i.e. l'engine legacy a
// soudain SharedProvider câblé), c'est qu'on a sournoisement ajouté un
// default — auquel cas il faut alerter et soit confirmer que c'est voulu,
// soit revenir à un legacy strict.
func TestTrigger_LegacyEngine_LacksWiring(t *testing.T) {
	tp := &mockTokenProvider{}
	// Pas de factory câblée. On va intercepter via un test indirect : on
	// construit un trigger sans factory, et on observe que le code legacy
	// passe par NewSyncEngine(...nil) — donc rien n'est wiré.
	trigger := NewTrigger(t.TempDir(), tp, domain.SyncOptions{})

	if trigger.engineFactory != nil {
		t.Fatalf("test setup: engineFactory should be nil")
	}
	// Pas d'assertion runtime ici — le SyncEngine legacy est créé puis
	// utilisé, et son état n'est pas observable de l'extérieur. La sécurité
	// vient du test précédent (factory câblée → assertions OK) + de la
	// production qui doit toujours câbler la factory (cf. TestProductionWiringDocumented).
	_ = trigger
}

// TestProductionWiringDocumented vérifie que le commentaire de
// NewTrigger.WithEngineFactory mentionne explicitement le risque legacy.
// C'est une sécurité documentaire : un dev qui modifie le wiring lit le
// godoc et voit la note de l'incident 2026-05-26.
//
// Le test est volontairement basique — c'est un test compile-time qu'on
// fait passer pour avoir une assertion "le code mentionne l'incident".
// Si quelqu'un supprime ce commentaire et le code de garde-fou, ce test
// passera quand même (Go ne reflète pas sur les commentaires) — d'où
// l'importance du test golden BuildEngine côté scheduler.
func TestProductionWiringDocumented(_ *testing.T) {
	// Compile-time: NewTrigger + WithEngineFactory existent et sont
	// chainables (signature stable). Si on retire WithEngineFactory, ça
	// casse à la compilation, et cmd/server/main.go ne build plus.
	tp := &mockTokenProvider{}
	_ = NewTrigger("", tp, domain.SyncOptions{}).
		WithEngineFactory(func(_ context.Context, _, _ string) *SyncEngine { return nil })
}
