package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// RequirePlayerSlugParam (routes d'administration et diagnostics hors du groupe joueur)
// applique la même forme que la garde du groupe : mêmes tables de slugs.
func TestRequirePlayerSlugParam_RefusesMalformedSlug(t *testing.T) {
	for _, param := range []string{"player_slug", "player"} {
		for _, slug := range invalidPlayerSlugs {
			t.Run(param+"/"+slug, func(t *testing.T) {
				rr := runSlugParamGuard(param, param, slug)
				if rr.Code != http.StatusBadRequest {
					t.Fatalf("slug %q : status = %d, attendu 400", slug, rr.Code)
				}
				var body map[string]any
				if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil || body[errKeyCode] != "invalid_player_slug" {
					t.Fatalf("corps %q, attendu le code invalid_player_slug", rr.Body.String())
				}
			})
		}
	}
}

func TestRequirePlayerSlugParam_AcceptsGamertagSlugs(t *testing.T) {
	for _, slug := range legitimatePlayerSlugs {
		if rr := runSlugParamGuard("player", "player", slug); rr.Code != http.StatusOK {
			t.Errorf("slug légitime %q refusé : %d", slug, rr.Code)
		}
	}
}

// Une route du même handler qui ne porte pas le paramètre (ex. GET /monitoring/lusr-gaps)
// n'est pas concernée.
func TestRequirePlayerSlugParam_RouteWithoutParamPasses(t *testing.T) {
	if rr := runSlugParamGuard("player", "title_id", "halo"); rr.Code != http.StatusOK {
		t.Fatalf("route sans paramètre joueur refusée : %d", rr.Code)
	}
}

// runSlugParamGuard passe par la garde réelle, le paramètre chi routeParam valant slug.
func runSlugParamGuard(guardParam, routeParam, slug string) *httptest.ResponseRecorder {
	req := withURLParam(httptest.NewRequest(http.MethodGet, "/admin/x", nil), routeParam, slug)
	rr := httptest.NewRecorder()
	RequirePlayerSlugParam(guardParam)(okHandler).ServeHTTP(rr, req)
	return rr
}
