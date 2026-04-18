package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

// mockExclusionRepo implements port.MatchExclusionRepository.
type mockExclusionRepo struct {
	excluded map[string]bool
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

func TestMatchExclusionService_SetAndList(t *testing.T) {
	repo := &mockExclusionRepo{excluded: make(map[string]bool)}
	svc := NewMatchExclusionService(repo)

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
}
