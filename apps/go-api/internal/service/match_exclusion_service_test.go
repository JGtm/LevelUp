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
}

func (m *mockExclusionRepo) SetExclusion(_ context.Context, matchID string, excl bool) error {
	m.excluded[matchID] = excl
	return nil
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
	if rec.calls != 2 {
		t.Fatalf("expected 2 recompute calls (set + unset), got %d", rec.calls)
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
