package service

// bootstrap_privacy_test.go — /bootstrap et l'appel live de privacy (plan perf
// 2026-09-23, lot L5b, D5b.7) : un échec (budget dépassé, erreur) est mémorisé 5 min —
// aucun appel ni attente pendant ce temps —, puis l'appel est retenté ; l'annulation
// de la requête n'est pas un échec.

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
)

// scriptedPrivacyProvider : `block` = attendre la fin du contexte (Waypoint muet),
// sinon rendre info/err. Compte ses appels.
type scriptedPrivacyProvider struct {
	calls atomic.Int32
	mu    sync.Mutex
	block bool
	info  *domain.MatchPrivacyInfo
	err   error
}

func (p *scriptedPrivacyProvider) GetMatchPrivacy(ctx context.Context, _ string) (*domain.MatchPrivacyInfo, error) {
	p.calls.Add(1)
	p.mu.Lock()
	block, info, err := p.block, p.info, p.err
	p.mu.Unlock()
	if block {
		<-ctx.Done()
		return &domain.MatchPrivacyInfo{IsPartial: true, Hint: "fetch_error"}, nil
	}
	return info, err
}

func (p *scriptedPrivacyProvider) respond(info *domain.MatchPrivacyInfo, err error) {
	p.mu.Lock()
	p.block, p.info, p.err = false, info, err
	p.mu.Unlock()
}

// newPrivacyTestBootstrap : budget court (les tests n'attendent pas 2 s), horloge pilotée.
func newPrivacyTestBootstrap(p *scriptedPrivacyProvider, now *time.Time) *BootstrapService {
	s := NewBootstrapService(nil, nil).WithPrivacyProvider(p)
	s.privacyLive.budget = 40 * time.Millisecond
	s.privacyLive.now = func() time.Time { return *now }
	return s
}

// TestFetchPrivacy_BudgetExceededMemorized : Waypoint muet → nil au bout du budget ;
// pendant les 5 min suivantes, nil IMMÉDIAT sans appel ; ensuite l'appel est retenté.
func TestFetchPrivacy_BudgetExceededMemorized(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	p := &scriptedPrivacyProvider{block: true}
	s := newPrivacyTestBootstrap(p, &now)

	if got := s.fetchPrivacyNonBlocking(context.Background(), "x1"); got != nil {
		t.Fatalf("budget dépassé : %+v, want nil (repli state persisté)", got)
	}
	p.respond(&domain.MatchPrivacyInfo{IsPrivate: true}, nil) // Waypoint rétabli : l'attente prime
	now = now.Add(privacyFailureBackoff - time.Minute)
	start := time.Now()
	ctx, tm := timing.WithTimings(context.Background())
	if got := s.fetchPrivacyNonBlocking(ctx, "x1"); got != nil {
		t.Fatalf("pendant l'attente : %+v, want nil", got)
	}
	if waited := time.Since(start); waited > 20*time.Millisecond {
		t.Errorf("pendant l'attente, /bootstrap a patienté %v", waited)
	}
	if p.calls.Load() != 1 {
		t.Fatalf("appels pendant l'attente = %d, want 1", p.calls.Load())
	}
	markers := map[string]int{}
	for _, sec := range tm.Snapshot() {
		markers[sec.Name] = sec.Calls
	}
	if markers["privacy_live_backoff"] != 1 {
		t.Errorf("marqueurs = %v, want privacy_live_backoff=1", markers)
	}

	now = now.Add(2 * time.Minute) // attente écoulée
	if got := s.fetchPrivacyNonBlocking(context.Background(), "x1"); got == nil || !got.IsPrivate {
		t.Errorf("après l'attente : %+v, want la réponse Waypoint", got)
	}
	if p.calls.Load() != 2 {
		t.Errorf("appels = %d, want 2", p.calls.Load())
	}
}

// TestFetchPrivacy_ProviderErrorMemorized : une erreur du provider ouvre aussi
// l'attente, pour ce seul xuid.
func TestFetchPrivacy_ProviderErrorMemorized(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	p := &scriptedPrivacyProvider{err: errors.New("waypoint: 500")}
	s := newPrivacyTestBootstrap(p, &now)
	_ = s.fetchPrivacyNonBlocking(context.Background(), "x1")
	_ = s.fetchPrivacyNonBlocking(context.Background(), "x1")
	if p.calls.Load() != 1 {
		t.Fatalf("appels pour x1 = %d, want 1", p.calls.Load())
	}
	p.respond(&domain.MatchPrivacyInfo{}, nil)
	if got := s.fetchPrivacyNonBlocking(context.Background(), "x2"); got == nil {
		t.Errorf("autre xuid : nil, want la réponse (l'attente est par xuid)")
	}
}

// TestFetchPrivacy_RequestCanceledNotMemorized : l'annulation de /bootstrap n'est pas
// un échec de Waypoint — la requête suivante appelle.
func TestFetchPrivacy_RequestCanceledNotMemorized(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	p := &scriptedPrivacyProvider{block: true}
	s := newPrivacyTestBootstrap(p, &now)
	s.privacyLive.budget = time.Minute // l'annulation arrive avant le budget

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(10*time.Millisecond, cancel)
	if got := s.fetchPrivacyNonBlocking(ctx, "x1"); got != nil {
		t.Fatalf("requête annulée : %+v, want nil", got)
	}
	p.respond(&domain.MatchPrivacyInfo{IsPartial: true, Hint: "partial_private"}, nil)
	if got := s.fetchPrivacyNonBlocking(context.Background(), "x1"); got == nil {
		t.Errorf("après une annulation : nil, want la réponse Waypoint (pas d'attente ouverte)")
	}
}

// TestFetchPrivacy_SuccessNotBlocking : une réussite passe telle quelle, sans attente.
func TestFetchPrivacy_SuccessNotBlocking(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	p := &scriptedPrivacyProvider{info: &domain.MatchPrivacyInfo{IsPrivate: true, Hint: "full_private"}}
	s := newPrivacyTestBootstrap(p, &now)
	for i := 0; i < 2; i++ {
		if got := s.fetchPrivacyNonBlocking(context.Background(), "x1"); got == nil || !got.IsPrivate {
			t.Fatalf("appel %d : %+v", i, got)
		}
	}
	if p.calls.Load() != 2 {
		t.Errorf("appels = %d, want 2 (le succès est caché par le provider, pas ici)", p.calls.Load())
	}
}
