//go:build cgo

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// TestHumaNestedSubrouterProbe (EXPÉRIENCE Phase 3b) : valide que Huma peut être
// monté sur un SOUS-routeur chi (r.Route("/players/{player_slug}", ...)) et :
//
//	(1) lit le path param PARENT {player_slug} via Input{path:"player_slug"} ;
//	(2) hérite du middleware du sous-groupe (ownership/title équivalent) ;
//	(3) ne panique pas à l'enregistrement bien que le param soit dans le mount
//	    parent et non dans le path relatif "/pages/probe".
//
// C'est le go/no-go pour migrer les ~70 routes sous /players/{player_slug}.
func TestHumaNestedSubrouterProbe(t *testing.T) {
	type probeInput struct {
		PlayerSlug string `path:"player_slug"`
	}
	type probeOutput struct {
		Body struct {
			Slug          string `json:"slug"`
			MiddlewareRan bool   `json:"middleware_ran"`
		}
	}

	r := chi.NewRouter()
	r.Route("/api/v1/players/{player_slug}", func(r chi.Router) {
		// Middleware du sous-groupe : pose un flag dans le contexte de requête.
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := context.WithValue(req.Context(), probeMWKey{}, true)
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})
		api := newHumaAPI(r)
		huma.Get(api, "/pages/probe", func(ctx context.Context, in *probeInput) (*probeOutput, error) {
			out := &probeOutput{}
			out.Body.Slug = in.PlayerSlug
			ran, _ := ctx.Value(probeMWKey{}).(bool)
			out.Body.MiddlewareRan = ran
			return out, nil
		})
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/players/Madina97294/pages/probe", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("200 attendu, got %d: %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Slug          string `json:"slug"`
		MiddlewareRan bool   `json:"middleware_ran"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("JSON invalide: %v", err)
	}
	if out.Slug != "Madina97294" {
		t.Errorf("slug = %q, want Madina97294 (path param parent non lu)", out.Slug)
	}
	if !out.MiddlewareRan {
		t.Error("middleware du sous-groupe NON hérité par la route Huma")
	}
}

type probeMWKey struct{}
