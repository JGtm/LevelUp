package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// liveSpy enregistre si le repli live (appel Xbox avec un jeton réel) était armé.
type liveSpy struct{ live []bool }

func (s *liveSpy) Search(ctx context.Context, _ string) ([]domain.GamertagSearchResult, error) {
	s.live = append(s.live, ctxkeys.GamertagLiveSearch(ctx))
	return nil, nil
}

// ?live=1 n'arme le repli live que pour un appelant autorisé (session authentifiée, ou
// instance sans authentification) : un anonyme est servi en local seulement, sans erreur.
func TestGamertagSearch_LiveIgnoredWithoutAuthenticatedSession(t *testing.T) {
	for _, tc := range []struct {
		name    string
		allowed func(context.Context) bool
		want    bool
	}{
		{"appelant anonyme", func(context.Context) bool { return false }, false},
		{"appelant authentifié", func(context.Context) bool { return true }, true},
		{"aucun prédicat : fermé par défaut", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spy := &liveSpy{}
			r := chi.NewRouter()
			registerGamertagHuma(newHumaAPI(r), handlers.NewGamertagHandler(spy), tc.allowed)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/directory/gamertags/search?q=abcd&live=1", nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("statut %d, attendu 200: %s", rec.Code, rec.Body.String())
			}
			if len(spy.live) != 1 || spy.live[0] != tc.want {
				t.Fatalf("repli live armé = %v, attendu [%v]", spy.live, tc.want)
			}
		})
	}
}
