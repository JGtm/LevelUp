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
	"levelup/go-api/internal/service"
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

// localSearchStub : recherche locale minimale (aucun résultat) pour armer le
// décorateur live réel dans le test de repli.
type localSearchStub struct{}

func (localSearchStub) Search(_ context.Context, _ string) ([]domain.GamertagSearchResult, error) {
	return nil, nil
}

// resolverSpy compte les résolutions live (le round-trip Xbox que le repli doit
// éviter quand live=0).
type resolverSpy struct{ calls int }

func (r *resolverSpy) ResolveXUID(_ context.Context, _ string) (string, error) {
	r.calls++
	return "2533274800000001", nil
}

// TestGamertagHandler_Query_LiveGatesResolver : le paramètre live pilote le repli
// live via le décorateur réel. live=false (typeahead) ne touche JAMAIS le résolveur
// Xbox ; live=true (intention explicite) l'invoque. Challenge V72-24 (latence).
func TestGamertagHandler_Query_LiveGatesResolver(t *testing.T) {
	spy := &resolverSpy{}
	svc := service.NewLiveFallbackGamertagSearch(localSearchStub{}, spy)
	h := handlers.NewGamertagHandler(svc)

	if _, err := h.Query(context.Background(), "NeverSeenTag", false); err != nil {
		t.Fatalf("live=false: %v", err)
	}
	if spy.calls != 0 {
		t.Fatalf("live=false ne doit PAS appeler le résolveur (calls=%d)", spy.calls)
	}

	if _, err := h.Query(context.Background(), "NeverSeenTag", true); err != nil {
		t.Fatalf("live=true: %v", err)
	}
	if spy.calls != 1 {
		t.Fatalf("live=true doit appeler le résolveur une fois (calls=%d)", spy.calls)
	}
}

func TestGamertagHandler_Query_OK(t *testing.T) {
	expected := []domain.GamertagSearchResult{{Gamertag: testGamertag, XUID: "123"}}
	h := handlers.NewGamertagHandler(&mockGamertagSearchService{results: expected})

	resp, err := h.Query(context.Background(), "test", false)
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

	resp, err := h.Query(context.Background(), "   ", false)
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

	if _, err := h.Query(context.Background(), "abc", false); err == nil {
		t.Fatal("erreur attendue, got nil")
	}
}

func TestGamertagHandler_Query_Unavailable(t *testing.T) {
	h := handlers.NewGamertagHandler(nil)

	if _, err := h.Query(context.Background(), "abc", false); !errors.Is(err, handlers.ErrGamertagSearchUnavailable) {
		t.Fatalf("err = %v, want ErrGamertagSearchUnavailable", err)
	}
}
