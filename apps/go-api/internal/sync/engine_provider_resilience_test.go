//go:build integration

// Package sync — engine_provider_resilience_test.go : tests E2E de résilience
// sync engine ↔ Provider sous conditions adverses.
//
// Couvre deux scénarios critiques de friction sync↔readers :
//   - Cancel du contexte sync mid-run : les readers HTTP en cours / nouveaux
//     ne doivent PAS être impactés, le Provider revient proprement à StateRO.
//   - Erreur métier du sync (mock client) : le sync échoue à l'API call, le
//     Provider doit recover gracieusement (state cohérent, readers OK ensuite).
//
// Le 3e scénario "Get timeout pendant swap" est déjà couvert par
// provider_timeout_integration_test.go dans le package sharedprovider — il
// utilise des helpers test-only non accessibles cross-package.
//
// Le commit 10a a corrigé le bug deadlock double-dblease prérequis pour
// que ces tests aient un sens E2E.
package sync

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// TestE2E_SyncEngine_ContextCancel_ReadersUnaffected_integration : un sync
// annulé en cours (ctx cancel) ne doit pas perturber les readers. Le Provider
// doit faire son rollback proprement et retomber sur StateRO.
func TestE2E_SyncEngine_ContextCancel_ReadersUnaffected_integration(t *testing.T) {
	env := newE2EEnv(t)
	env.seedMockMatches("cancel-test", 10)

	// Goroutines readers en boucle continue pendant tout le test.
	readerCtx, readerCancel := context.WithCancel(context.Background())
	defer readerCancel()
	var (
		readerOK         atomic.Int64
		readerErr        atomic.Int64
		readerCatalogErr atomic.Int64
		readerDiffCfgErr atomic.Int64
		readerCtxErr     atomic.Int64
		readersWG        sync.WaitGroup
	)
	const nReaders = 8
	for i := 0; i < nReaders; i++ {
		readersWG.Add(1)
		go func() {
			defer readersWG.Done()
			for readerCtx.Err() == nil {
				db, release, err := env.pool.SharedReadDB().Get(readerCtx)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						readerCtxErr.Add(1)
						return
					}
					readerErr.Add(1)
					classifyErr(err, &readerCatalogErr, &readerDiffCfgErr)
					continue
				}
				var n int
				qErr := db.QueryRowContext(readerCtx, "SELECT COUNT(*) FROM match_registry").Scan(&n)
				release()
				if qErr != nil {
					if errors.Is(qErr, context.Canceled) {
						readerCtxErr.Add(1)
						return
					}
					readerErr.Add(1)
					classifyErr(qErr, &readerCatalogErr, &readerDiffCfgErr)
					continue
				}
				readerOK.Add(1)
				time.Sleep(time.Millisecond)
			}
		}()
	}

	// Sync : on lance RunDelta avec un ctx qu'on cancel après 50ms.
	syncCtx, syncCancel := context.WithCancel(context.Background())
	opts := domain.SyncOptions{
		MatchType:         "matchmaking",
		MaxMatches:        10,
		WithParticipants:  true,
		WithMedals:        true,
		RequestsPerSecond: 100,
	}
	syncDone := make(chan error, 1)
	go func() {
		_, err := env.engine.RunDelta(syncCtx, opts)
		syncDone <- err
	}()

	// Cancel après 50ms — le sync est probablement en plein milieu d'un batch.
	time.Sleep(50 * time.Millisecond)
	syncCancel()

	// Attendre que le sync termine (avec erreur ctx ou pas, selon timing).
	select {
	case <-syncDone:
	case <-time.After(10 * time.Second):
		t.Fatal("sync ne termine pas après cancel (deadlock potentiel)")
	}

	// Laisse les readers tourner 200ms supplémentaires pour exercer le post-cancel.
	time.Sleep(200 * time.Millisecond)
	readerCancel()
	readersWG.Wait()

	t.Logf("Cancel test : readers OK=%d err=%d (Catalog=%d, DiffCfg=%d, CtxCancel=%d)",
		readerOK.Load(), readerErr.Load(),
		readerCatalogErr.Load(), readerDiffCfgErr.Load(), readerCtxErr.Load())

	// ─── Assertions ─────────────────────────────────────────────────────────────
	if readerCatalogErr.Load() > 0 {
		t.Errorf("%d Catalog Error reader (attendu 0)", readerCatalogErr.Load())
	}
	if readerDiffCfgErr.Load() > 0 {
		t.Errorf("%d \"different configuration\" reader (attendu 0)", readerDiffCfgErr.Load())
	}
	if readerOK.Load() < 50 {
		t.Errorf("readerOK=%d trop faible — le cancel a peut-être trop pénalisé les readers", readerOK.Load())
	}
	// Provider doit avoir rollback vers RO après cancel.
	if state := env.provider.State(); state != sharedprovider.StateRO {
		t.Errorf("Provider.State final = %v (attendu StateRO après cancel)", state)
	}
}

// TestE2E_SyncEngine_MockClientError_ProviderRecovers_integration : le mock
// client retourne une erreur de GetMatchHistory. Le sync renvoie status=success
// avec inserted=0 et warnings≥1 (comportement engine : best-effort, continue
// si l'API ne répond pas). Le Provider doit terminer proprement (release
// writer, retour à StateRO). Le run suivant avec un mock fonctionnel doit
// processer les matchs normalement — preuve de la résilience.
func TestE2E_SyncEngine_MockClientError_ProviderRecovers_integration(t *testing.T) {
	env := newE2EEnv(t)
	env.mock.history = nil
	env.mock.getHistoryErr = errors.New("mock: simulated API failure")

	ctx := context.Background()

	// 1er run : GetMatchHistory échoue → warnings ≥ 1, inserted = 0.
	opts := domain.SyncOptions{
		MatchType:         "matchmaking",
		MaxMatches:        5,
		WithParticipants:  true,
		WithMedals:        true,
		RequestsPerSecond: 100,
	}
	result, err := env.engine.RunDelta(ctx, opts)
	if err != nil {
		t.Logf("RunDelta retourne err=%v (best-effort tolère l'API failure)", err)
	}
	if result.MatchesInserted != 0 {
		t.Errorf("MatchesInserted = %d (attendu 0)", result.MatchesInserted)
	}
	if len(result.Warnings) < 1 {
		t.Errorf("Warnings count = %d (attendu ≥ 1 pour signaler l'échec API)",
			len(result.Warnings))
	}

	// Provider doit être revenu à StateRO malgré l'échec API.
	if state := env.provider.State(); state != sharedprovider.StateRO {
		t.Errorf("Provider.State après échec sync = %v (attendu StateRO)", state)
	}

	// Readers doivent fonctionner normalement après l'échec.
	db, release, err := env.provider.Get(ctx)
	if err != nil {
		t.Fatalf("provider.Get après échec sync : %v", err)
	}
	var n int
	queryErr := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM match_registry").Scan(&n)
	release()
	if queryErr != nil {
		t.Errorf("query post-échec : %v", queryErr)
	}
	if n != 0 {
		t.Errorf("match count après échec = %d (attendu 0)", n)
	}

	// 2e run : sans erreur mock cette fois → doit réussir, prouvant la recovery.
	env.mock.getHistoryErr = nil
	env.seedMockMatches("recovery", 3)
	result, err = env.engine.RunDelta(ctx, opts)
	if err != nil {
		t.Fatalf("RunDelta après recovery : %v", err)
	}
	// Sync engine pagine GetMatchHistory tant qu'il reçoit des matchs ; le
	// mockHaloClient renvoie statiquement la même liste à chaque appel, donc
	// le nombre exact dépend de la logique de stop pagination — on vérifie
	// uniquement le contrat post-recovery : ≥ 1 match inséré.
	if result.MatchesInserted < 1 {
		t.Errorf("MatchesInserted post-recovery = %d (attendu ≥ 1)", result.MatchesInserted)
	}
	if state := env.provider.State(); state != sharedprovider.StateRO {
		t.Errorf("Provider.State post-recovery = %v (attendu StateRO)", state)
	}
}

// classifyErr et helper compteur sont déjà définis dans engine_provider_e2e_test.go
// (même package, même build tag integration → réutilisation directe).
