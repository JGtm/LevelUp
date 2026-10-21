// Package handlers_test — home_matches_test.go : GET /players/{slug}/pages/home/matches.
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

func homeMatchesFactory(mock *mockHomeService) func(context.Context, string) (port.HomeService, context.Context, string, string, error) {
	return func(ctx context.Context, slug string) (port.HomeService, context.Context, string, string, error) {
		if slug != testPlayerSlug {
			return nil, ctx, "", "", errors.New("player_not_found")
		}
		return mock, ctx, testXUID1, testGamertag, nil
	}
}

func getHomeMatches(t *testing.T, mock *mockHomeService, target string, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	r := newHomeRouter(homeMatchesFactory(mock), nil)
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHomeHandler_GetHomeMatches_OK(t *testing.T) {
	mock := &mockHomeService{matches: &domain.HomeMatchesPage{
		Items:      []domain.RecentMatchItem{{MatchID: "m1"}, {MatchID: "m2"}},
		HasMore:    true,
		NextBefore: "2026-10-20T21:00:00Z_m2",
	}}
	w := getHomeMatches(t, mock, "/players/test-player/pages/home/matches?before=abc&limit=45", map[string]string{"X-LevelUp-Locale": "en"})
	if w.Code != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d : %s", w.Code, w.Body.String())
	}
	if mock.matchesLocale != "en" {
		t.Errorf("locale = %q, attendu en", mock.matchesLocale)
	}
	want := domain.HomeMatchesRequest{Kind: "recent", Before: "abc", Limit: 45}
	if mock.matchesReq != want {
		t.Errorf("requête = %+v, attendu %+v", mock.matchesReq, want)
	}
	var got domain.HomeMatchesPage
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("corps illisible : %v", err)
	}
	if len(got.Items) != 2 || !got.HasMore || got.NextBefore != "2026-10-20T21:00:00Z_m2" {
		t.Errorf("corps = %+v", got)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "private, no-cache" {
		t.Errorf("Cache-Control = %q : la liste doit être revalidée, jamais resservie périmée", cc)
	}
	if w.Header().Get("ETag") == "" {
		t.Error("ETag attendu")
	}
}

func TestHomeHandler_GetHomeMatches_304(t *testing.T) {
	mock := &mockHomeService{matches: &domain.HomeMatchesPage{Items: []domain.RecentMatchItem{{MatchID: "m1"}}}}
	first := getHomeMatches(t, mock, "/players/test-player/pages/home/matches", nil)
	etag := first.Header().Get("ETag")
	w := getHomeMatches(t, mock, "/players/test-player/pages/home/matches", map[string]string{"If-None-Match": etag})
	if w.Code != http.StatusNotModified {
		t.Fatalf("attendu 304, obtenu %d", w.Code)
	}
}

func TestHomeHandler_GetHomeMatches_Favoris(t *testing.T) {
	mock := &mockHomeService{matches: &domain.HomeMatchesPage{Items: []domain.RecentMatchItem{}}}
	w := getHomeMatches(t, mock, "/players/test-player/pages/home/matches?kind=favorites", nil)
	if w.Code != http.StatusOK || mock.matchesReq.Kind != domain.HomeMatchesKindFavorites {
		t.Fatalf("code %d, requête %+v", w.Code, mock.matchesReq)
	}
}

func TestHomeHandler_GetHomeMatches_Refus(t *testing.T) {
	for _, c := range []struct {
		name, target string
		mock         *mockHomeService
		want         int
	}{
		{"curseur invalide -> 400", "/players/test-player/pages/home/matches?before=zzz", &mockHomeService{matchesErr: domain.ErrHomeCursorInvalid}, http.StatusBadRequest},
		{"genre inconnu -> 422", "/players/test-player/pages/home/matches?kind=tout", &mockHomeService{}, http.StatusUnprocessableEntity},
		{"limite hors bornes -> 422", "/players/test-player/pages/home/matches?limit=61", &mockHomeService{}, http.StatusUnprocessableEntity},
		{"limite nulle -> 422", "/players/test-player/pages/home/matches?limit=0", &mockHomeService{}, http.StatusUnprocessableEntity},
		{"joueur inconnu -> 404", "/players/inconnu/pages/home/matches", &mockHomeService{}, http.StatusNotFound},
		{"erreur service -> 500", "/players/test-player/pages/home/matches", &mockHomeService{matchesErr: errors.New("db_error")}, http.StatusInternalServerError},
		{"base fermée -> 503", "/players/test-player/pages/home/matches", &mockHomeService{matchesErr: errors.New("x: sql: database is closed")}, http.StatusServiceUnavailable},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := getHomeMatches(t, c.mock, c.target, nil)
			if w.Code != c.want {
				t.Errorf("attendu %d, obtenu %d : %s", c.want, w.Code, w.Body.String())
			}
		})
	}
}
