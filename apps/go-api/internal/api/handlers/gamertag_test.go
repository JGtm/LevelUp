// Package handlers — gamertag_test.go : tests unitaires GamertagHandler.Query
// avec mock service. Le routage HTTP (200/500/503, query-param) est testé côté
// api via le golden Huma TestRegisterGamertagHuma_ContractPreserved (Phase 3b).
package handlers_test

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

type mockGamertagSearchService struct {
	results []domain.GamertagSearchResult
	err     error
}

func (m *mockGamertagSearchService) Search(_ context.Context, _ string) ([]domain.GamertagSearchResult, error) {
	return m.results, m.err
}

// compile-time : le mock satisfait le port.
var _ port.GamertagSearchService = (*mockGamertagSearchService)(nil)

func TestGamertagHandler_Query_OK(t *testing.T) {
	expected := []domain.GamertagSearchResult{{Gamertag: testGamertag, XUID: "123"}}
	h := handlers.NewGamertagHandler(&mockGamertagSearchService{results: expected})

	resp, err := h.Query(context.Background(), "test")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Gamertag != testGamertag {
		t.Errorf("unexpected items: %+v", resp.Items)
	}
	if resp.Query != "test" {
		t.Errorf("Query = %q, want test", resp.Query)
	}
}

func TestGamertagHandler_Query_Empty(t *testing.T) {
	// svc ne doit pas être appelé sur une query vide.
	h := handlers.NewGamertagHandler(&mockGamertagSearchService{err: errors.New("ne doit pas être appelé")})

	resp, err := h.Query(context.Background(), "   ")
	if err != nil {
		t.Fatalf("Query vide ne doit pas renvoyer d'erreur: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected empty items, got %v", resp.Items)
	}
	if resp.Items == nil {
		t.Error("Items ne doit jamais être nil (contrat JSON [])")
	}
}

func TestGamertagHandler_Query_ServiceError(t *testing.T) {
	h := handlers.NewGamertagHandler(&mockGamertagSearchService{err: errors.New("db error")})

	if _, err := h.Query(context.Background(), "abc"); err == nil {
		t.Fatal("erreur attendue, got nil")
	}
}

func TestGamertagHandler_Query_Unavailable(t *testing.T) {
	h := handlers.NewGamertagHandler(nil)

	if _, err := h.Query(context.Background(), "abc"); !errors.Is(err, handlers.ErrGamertagSearchUnavailable) {
		t.Fatalf("err = %v, want ErrGamertagSearchUnavailable", err)
	}
}
