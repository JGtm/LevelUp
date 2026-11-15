package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/coalesce"
	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

// deadlineRepo relève l'échéance du contexte avec lequel l'écriture de l'instantané est demandée.
type deadlineRepo struct {
	mockCareerLiveRepo
	deadline chan time.Time
}

func (r *deadlineRepo) InsertCareerProgressionPartial(ctx context.Context, xuid string, p *duckdb.CareerProgressionPartial) (bool, error) {
	d, _ := ctx.Deadline()
	r.deadline <- d
	return r.mockCareerLiveRepo.InsertCareerProgressionPartial(ctx, xuid, p)
}

// L'écriture de l'instantané de carrière attend l'écrivain de la base joueur plus longtemps que le
// budget des appels à l'API (careerLiveBgTimeout) : un recalcul des notes qui tient l'écrivain
// plusieurs minutes ne la fait plus échouer.
func TestCareerLive_PersistAttendLEcrivainAuDelaDuBudgetDeFetch(t *testing.T) {
	repo := &deadlineRepo{deadline: make(chan time.Time, 1)}
	fetcher := &mockCareerFetcher{
		progress: &syncpkg.CareerRankData{CurrentRank: 40, CurrentXP: 100},
		custom:   &syncpkg.SpartanCustomizationData{SpartanID: "SR-1"},
	}
	svc := NewCareerLiveService(repo, &mockIdentityBuilder{},
		func(context.Context) CareerFetcher { return fetcher }, NewCareerLiveCache(CareerLiveCacheConfig{}))

	debut := time.Now()
	svc.kickoffBackgroundRefresh("1234567890123456", "", &domain.HaloTokens{SpartanToken: "tok"}, "halo_infinite")

	select {
	case d := <-repo.deadline:
		if reste := d.Sub(debut); reste <= careerLiveBgTimeout+time.Minute {
			t.Errorf("échéance de l'écriture = %v après le départ, attendu au-delà du budget de fetch (%v) : %v",
				reste, careerLiveBgTimeout, careerLivePersistWait)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("l'instantané n'a pas été écrit")
	}
}

// writerBusyRepo joue un écrivain de base joueur occupé : l'écriture n'aboutit que si son contexte
// n'est pas annulé, sinon elle rend l'erreur du contexte, comme le fait l'attente de l'écrivain.
type writerBusyRepo struct {
	mockCareerLiveRepo
	waiting chan struct{}
	release chan struct{}
	written atomic.Int32
	calls   atomic.Int32
}

func (r *writerBusyRepo) InsertCareerProgressionPartial(ctx context.Context, _ string, _ *duckdb.CareerProgressionPartial) (bool, error) {
	r.calls.Add(1)
	select {
	case r.waiting <- struct{}{}:
	default:
	}
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-r.release:
		r.written.Add(1)
		return true, nil
	}
}

func newBusyWriterService() (*CareerLiveService, *writerBusyRepo) {
	repo := &writerBusyRepo{waiting: make(chan struct{}, 1), release: make(chan struct{})}
	fetcher := &mockCareerFetcher{
		progress: &syncpkg.CareerRankData{CurrentRank: 40, CurrentXP: 100},
		custom:   &syncpkg.SpartanCustomizationData{SpartanID: "SR-1"},
	}
	svc := NewCareerLiveService(repo, &mockIdentityBuilder{},
		func(context.Context) CareerFetcher { return fetcher }, NewCareerLiveCache(CareerLiveCacheConfig{}))
	return svc, repo
}

func waitRefreshIdle(t *testing.T, svc *CareerLiveService) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		svc.bgInflightMu.Lock()
		idle := len(svc.bgInflight) == 0
		svc.bgInflightMu.Unlock()
		if idle {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("le refresh de fond ne s'est pas terminé")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// L'arrêt du process pendant l'attente de l'écrivain annule l'écriture de l'instantané et l'attend :
// à son retour (avant la fermeture des bases), plus aucune écriture n'est en vol ni ne partira.
func TestCareerLive_ArretPendantLAttenteDeLEcrivainAnnuleLEcriture(t *testing.T) {
	t.Cleanup(coalesce.IsolateTracking())
	svc, repo := newBusyWriterService()

	svc.kickoffBackgroundRefresh("1234567890123456", "", &domain.HaloTokens{SpartanToken: "tok"}, "halo_infinite")
	select {
	case <-repo.waiting:
	case <-time.After(5 * time.Second):
		t.Fatal("l'écriture n'a pas commencé à attendre l'écrivain")
	}

	ctx, cancel := context.WithTimeout(context.Background(), coalesce.DrainWait)
	defer cancel()
	if err := coalesce.Drain(ctx); err != nil {
		t.Fatalf("l'arrêt n'a pas attendu la fin de l'écriture annulée : %v", err)
	}
	waitRefreshIdle(t, svc)

	// L'écrivain se libère après l'arrêt : l'écriture, annulée, ne doit pas partir.
	close(repo.release)
	time.Sleep(50 * time.Millisecond)
	if n := repo.written.Load(); n != 0 {
		t.Errorf("%d instantané(s) écrit(s) après l'arrêt, attendu 0", n)
	}
}

// Un refresh dont le fetch finit après le début de l'arrêt n'écrit rien : l'écriture n'est pas lancée.
func TestCareerLive_ArretAvantLEcritureNeLanceRien(t *testing.T) {
	t.Cleanup(coalesce.IsolateTracking())
	svc, repo := newBusyWriterService()
	close(repo.release)

	ctx, cancel := context.WithTimeout(context.Background(), coalesce.DrainWait)
	defer cancel()
	if err := coalesce.Drain(ctx); err != nil {
		t.Fatalf("Drain : %v", err)
	}

	svc.kickoffBackgroundRefresh("1234567890123456", "", &domain.HaloTokens{SpartanToken: "tok"}, "halo_infinite")
	waitRefreshIdle(t, svc)
	if n := repo.calls.Load(); n != 0 {
		t.Errorf("%d écriture(s) lancée(s) après le début de l'arrêt, attendu 0", n)
	}
}
