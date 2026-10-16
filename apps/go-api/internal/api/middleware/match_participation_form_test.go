package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Un {match_id} qui n'a pas la forme d'un identifiant de match Halo reçoit le 404 d'un
// match non joué, avant toute vérification et même sans vérification configurée (démo,
// auth inactive) : l'identifiant devient la clé d'un chemin d'artefact (rejeu, faits,
// raster tactique). Même réponse que l'absence : la frontière n'est pas un oracle.
func TestRequireMatchParticipation_RefusesMalformedMatchID(t *testing.T) {
	malformed := []string{
		"..", ".", "..%2F..%2Fetc", "..%5Cx", "a.b", "zzzzzzzz-zzzz", "m1",
		"000d5950%00", strings.Repeat("a", 65),
	}
	checks := map[string]MatchParticipationCheck{
		"sans vérification": nil,
		"vérification qui accepte": func(context.Context, string, string) (bool, error) {
			return true, nil
		},
	}
	for name, check := range checks {
		for _, id := range malformed {
			t.Run(name+"/"+id, func(t *testing.T) {
				called := false
				var wrapped MatchParticipationCheck
				if check != nil {
					wrapped = func(ctx context.Context, slug, match string) (bool, error) {
						called = true
						return check(ctx, slug, match)
					}
				}
				rr := httptest.NewRecorder()
				participationRouter(wrapped).ServeHTTP(rr,
					httptest.NewRequest(http.MethodGet, "/players/alice/matches/"+id+"/replay", nil))
				if rr.Code != http.StatusNotFound {
					t.Fatalf("match_id %q : statut %d, attendu 404", id, rr.Code)
				}
				if !strings.Contains(rr.Body.String(), "match_not_participant") {
					t.Fatalf("match_id %q : corps %q, attendu match_not_participant", id, rr.Body.String())
				}
				if called {
					t.Fatalf("match_id %q : vérification appelée avec un identifiant mal formé", id)
				}
			})
		}
	}
}

func TestRequireMatchParticipation_AcceptsHaloMatchIDs(t *testing.T) {
	for _, id := range []string{"000d5950-1234-4abc-9def-0123456789ab", "000d5950", "ABCDEF01-2345"} {
		for _, check := range []MatchParticipationCheck{nil, func(context.Context, string, string) (bool, error) { return true, nil }} {
			rr := httptest.NewRecorder()
			participationRouter(check).ServeHTTP(rr,
				httptest.NewRequest(http.MethodGet, "/players/alice/matches/"+id+"/replay", nil))
			if rr.Code != http.StatusOK {
				t.Fatalf("match_id %q : statut %d, attendu 200", id, rr.Code)
			}
		}
	}
}
