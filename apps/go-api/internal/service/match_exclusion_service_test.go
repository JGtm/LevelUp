package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

// mockExclusionRepo implements port.MatchExclusionRepository.
type mockExclusionRepo struct {
	excluded map[string]bool
	registry map[string]domain.MatchRegistryInfo
	sets     int // nombre d'écritures du flag
}

func (m *mockExclusionRepo) SetExclusion(_ context.Context, matchID string, excl bool) (bool, error) {
	if m.excluded[matchID] == excl {
		return false, nil
	}
	m.sets++
	m.excluded[matchID] = excl
	return true, nil
}

func (m *mockExclusionRepo) ListExcluded(_ context.Context) ([]domain.ExcludedMatch, error) {
	var result []domain.ExcludedMatch
	for id, excl := range m.excluded {
		if excl {
			result = append(result, domain.ExcludedMatch{MatchID: id})
		}
	}
	return result, nil
}

func (m *mockExclusionRepo) GetMatchRegistryInfo(_ context.Context, matchID string) (domain.MatchRegistryInfo, error) {
	info, ok := m.registry[matchID]
	if !ok {
		return domain.MatchRegistryInfo{}, domain.ErrMatchNotFound
	}
	return info, nil
}

// mockRecomputer compte les appels au recompute pour assertions.
type mockRecomputer struct {
	calls    int
	lastID   string
	returnTo error
}

func (m *mockRecomputer) RecomputeAfterExclusion(_ context.Context, matchID string) error {
	m.calls++
	m.lastID = matchID
	return m.returnTo
}

func newSvc(t *testing.T) (*MatchExclusionService, *mockExclusionRepo, *mockRecomputer) {
	t.Helper()
	repo := &mockExclusionRepo{
		excluded: make(map[string]bool),
		registry: map[string]domain.MatchRegistryInfo{
			"m1":          {MatchID: "m1", IsRanked: false},
			"m2":          {MatchID: "m2", IsRanked: false},
			"m_ranked":    {MatchID: "m_ranked", IsRanked: true},
			"m_firefight": {MatchID: "m_firefight", IsRanked: false, IsFirefight: true},
		},
	}
	rec := &mockRecomputer{}
	return NewMatchExclusionService(repo, rec), repo, rec
}

func TestMatchExclusionService_SetAndList(t *testing.T) {
	svc, _, rec := newSvc(t)
	if err := svc.SetExclusion(context.Background(), "m1", true); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetExclusion(context.Background(), "m2", true); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetExclusion(context.Background(), "m2", false); err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListExcluded(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 excluded, got %d", len(list))
	}
	if list[0].MatchID != "m1" {
		t.Fatalf("expected m1, got %s", list[0].MatchID)
	}
	if rec.calls != 3 {
		t.Fatalf("expected 3 recompute calls (m1 set, m2 set, m2 unset), got %d", rec.calls)
	}
}

func TestMatchExclusionService_RankedRefused(t *testing.T) {
	svc, repo, rec := newSvc(t)
	err := svc.SetExclusion(context.Background(), "m_ranked", true)
	if !errors.Is(err, domain.ErrRankedMatchNotExcludable) {
		t.Fatalf("expected ErrRankedMatchNotExcludable, got %v", err)
	}
	if repo.excluded["m_ranked"] {
		t.Fatal("ranked match should not have been excluded")
	}
	if rec.calls != 0 {
		t.Fatalf("recompute should not run on refused exclusion, got %d calls", rec.calls)
	}
}

func TestMatchExclusionService_RankedReactivationAllowed(t *testing.T) {
	svc, _, _ := newSvc(t)
	// Réactiver (excluded=false) un match classé reste autorisé.
	if err := svc.SetExclusion(context.Background(), "m_ranked", false); err != nil {
		t.Fatalf("ranked reactivation should be allowed, got %v", err)
	}
}

func TestMatchExclusionService_MatchNotFound(t *testing.T) {
	svc, _, _ := newSvc(t)
	err := svc.SetExclusion(context.Background(), "ghost", true)
	if !errors.Is(err, domain.ErrMatchNotFound) {
		t.Fatalf("expected ErrMatchNotFound, got %v", err)
	}
}

func TestMatchExclusionService_RecomputeErrorPropagates(t *testing.T) {
	svc, repo, rec := newSvc(t)
	rec.returnTo = errors.New("disk full")
	err := svc.SetExclusion(context.Background(), "m1", true)
	if err == nil {
		t.Fatal("expected error from recomputer to propagate")
	}
	// Le flag a été UPSERT avant l'échec du recompute — l'utilisateur peut retenter,
	// le SetExclusion reste idempotent.
	if !repo.excluded["m1"] {
		t.Error("flag should be set even when recompute fails")
	}
	if rec.calls != 1 {
		t.Fatalf("recomputer should be called once, got %d", rec.calls)
	}
}

func TestMatchExclusionService_RecomputeReceivesMatchID(t *testing.T) {
	svc, _, rec := newSvc(t)
	if err := svc.SetExclusion(context.Background(), "m2", true); err != nil {
		t.Fatal(err)
	}
	if rec.lastID != "m2" {
		t.Fatalf("recomputer should receive matchID m2, got %q", rec.lastID)
	}
}

func TestMatchExclusionService_FirefightExcludable(t *testing.T) {
	// Un match Firefight n'est PAS classé : il doit pouvoir être exclu (les seuls
	// matchs protégés sont les ranked CSR).
	svc, repo, rec := newSvc(t)
	if err := svc.SetExclusion(context.Background(), "m_firefight", true); err != nil {
		t.Fatalf("firefight should be excludable, got %v", err)
	}
	if !repo.excluded["m_firefight"] {
		t.Fatal("firefight match should be flagged excluded")
	}
	if rec.calls != 1 {
		t.Fatalf("recomputer should run on firefight exclusion, got %d calls", rec.calls)
	}
}

func TestMatchExclusionService_RecomputerNil(t *testing.T) {
	repo := &mockExclusionRepo{
		excluded: make(map[string]bool),
		registry: map[string]domain.MatchRegistryInfo{
			"m1": {MatchID: "m1"},
		},
	}
	svc := NewMatchExclusionService(repo, nil)
	if err := svc.SetExclusion(context.Background(), "m1", true); err != nil {
		t.Fatalf("nil recomputer should not break the call, got %v", err)
	}
	if !repo.excluded["m1"] {
		t.Fatal("flag should still be set with nil recomputer")
	}
}

// Une exclusion identique à l'état courant ne change rien : aucune écriture du flag (la
// table est append-only), aucun recalcul de l'historique.
func TestMatchExclusionService_UnchangedStateIsANoOp(t *testing.T) {
	svc, repo, rec := newSvc(t)
	ctx := context.Background()

	// Démarquer un match jamais exclu : sans effet.
	if err := svc.SetExclusion(ctx, "m1", false); err != nil {
		t.Fatal(err)
	}
	if repo.sets != 0 || rec.calls != 0 {
		t.Fatalf("m1 jamais exclu puis démarqué : écritures=%d recalculs=%d, attendu 0/0", repo.sets, rec.calls)
	}

	// Exclure puis ré-exclure le même match : une seule écriture, un seul recalcul.
	for range 5 {
		if err := svc.SetExclusion(ctx, "m1", true); err != nil {
			t.Fatal(err)
		}
	}
	if repo.sets != 1 || rec.calls != 1 {
		t.Fatalf("5 exclusions identiques : écritures=%d recalculs=%d, attendu 1/1", repo.sets, rec.calls)
	}

	// Un vrai changement d'état recalcule.
	if err := svc.SetExclusion(ctx, "m1", false); err != nil {
		t.Fatal(err)
	}
	if repo.sets != 2 || rec.calls != 2 {
		t.Fatalf("démarquage effectif : écritures=%d recalculs=%d, attendu 2/2", repo.sets, rec.calls)
	}
}

// Un match classé déjà non exclu reste refusé à l'exclusion (la garde passe avant le no-op).
func TestMatchExclusionService_RankedStillRefusedBeforeNoOpCheck(t *testing.T) {
	svc, repo, _ := newSvc(t)
	repo.excluded["m_ranked"] = true // état hérité incohérent
	if err := svc.SetExclusion(context.Background(), "m_ranked", true); !errors.Is(err, domain.ErrRankedMatchNotExcludable) {
		t.Fatalf("err = %v, attendu ErrRankedMatchNotExcludable", err)
	}
}

// pendingRecomputer échoue puis réussit, et déclare un recalcul en attente après un échec.
type pendingRecomputer struct {
	mockRecomputer
	failNext bool
	pending  bool
}

func (p *pendingRecomputer) RecomputeAfterExclusion(ctx context.Context, matchID string) error {
	p.calls++
	if p.failNext {
		p.failNext = false
		p.pending = true
		return errors.New("db locked")
	}
	p.pending = false
	return nil
}

func (p *pendingRecomputer) RecomputePending() bool { return p.pending }

// Après un recalcul en échec, la même demande répétée recalcule (le flag est déjà posé :
// aucune seconde écriture) au lieu d'être tenue pour un no-op qui laisserait l'historique
// périmé.
func TestMatchExclusionService_RetryAfterFailedRecomputeRecomputes(t *testing.T) {
	repo := &mockExclusionRepo{
		excluded: make(map[string]bool),
		registry: map[string]domain.MatchRegistryInfo{"m1": {MatchID: "m1"}},
	}
	rec := &pendingRecomputer{failNext: true}
	svc := NewMatchExclusionService(repo, rec)
	ctx := context.Background()

	if err := svc.SetExclusion(ctx, "m1", true); err == nil {
		t.Fatal("le premier appel devait échouer (recalcul en échec)")
	}
	if err := svc.SetExclusion(ctx, "m1", true); err != nil {
		t.Fatalf("la reprise doit réussir: %v", err)
	}
	if rec.calls != 2 || repo.sets != 1 {
		t.Fatalf("recalculs=%d écritures=%d, attendu 2/1", rec.calls, repo.sets)
	}
	// Recalcul réussi : la demande suivante identique est de nouveau un no-op.
	if err := svc.SetExclusion(ctx, "m1", true); err != nil {
		t.Fatal(err)
	}
	if rec.calls != 2 || repo.sets != 1 {
		t.Fatalf("après succès: recalculs=%d écritures=%d, attendu 2/1", rec.calls, repo.sets)
	}
}
