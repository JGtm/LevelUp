package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func participationRouter(check MatchParticipationCheck) *chi.Mux {
	r := chi.NewRouter()
	r.With(RequireMatchParticipation(check)).Get("/players/{player_slug}/matches/{match_id}/replay", okHandler)
	r.With(RequireMatchParticipation(check)).Get("/players/{player_slug}/tactical/maps", okHandler)
	return r
}

func TestRequireMatchParticipation(t *testing.T) {
	cases := []struct {
		name  string
		check MatchParticipationCheck
		path  string
		want  int
	}{
		{"match joué", func(context.Context, string, string) (bool, error) { return true, nil }, "/players/alice/matches/000d5950-1234-4abc-9def-0123456789ab/replay", http.StatusOK},
		{"match non joué", func(context.Context, string, string) (bool, error) { return false, nil }, "/players/alice/matches/000d5950-1234-4abc-9def-0123456789ab/replay", http.StatusNotFound},
		{"vérification en échec : fermé", func(context.Context, string, string) (bool, error) { return false, errors.New("db") }, "/players/alice/matches/000d5950-1234-4abc-9def-0123456789ab/replay", http.StatusServiceUnavailable},
		{"route sans match", func(context.Context, string, string) (bool, error) { return false, nil }, "/players/alice/tactical/maps", http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotSlug, gotMatch string
			check := func(ctx context.Context, slug, match string) (bool, error) {
				gotSlug, gotMatch = slug, match
				return tc.check(ctx, slug, match)
			}
			rr := httptest.NewRecorder()
			participationRouter(check).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rr.Code != tc.want {
				t.Fatalf("statut %d, attendu %d (%s)", rr.Code, tc.want, rr.Body.String())
			}
			if tc.want != http.StatusOK && (gotSlug != "alice" || gotMatch != "000d5950-1234-4abc-9def-0123456789ab") {
				t.Fatalf("vérification sur (%q, %q), attendu (alice, 000d5950-1234-4abc-9def-0123456789ab)", gotSlug, gotMatch)
			}
		})
	}
}
