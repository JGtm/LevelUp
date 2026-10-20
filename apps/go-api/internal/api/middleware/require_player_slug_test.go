package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// La garde du groupe /players/{player_slug} refuse un slug qui n'a pas la forme d'un
// gamertag avant toute autre décision, que la propriété soit appliquée ou non (démo,
// auth inactive) : un slug sert à construire des chemins disque.

var invalidPlayerSlugs = []string{
	"", ".", "..", "a/b", `a\b`, "../stats", `..\stats`, "..%2F..%2Fx", "a.b",
	"JGtm.", " JGtm", "JGtm ", "a\x00b", "a%20b", "a:b", "a*b",
	strings.Repeat("a", 65),
}

var legitimatePlayerSlugs = []string{
	"JGtm", "Madina97294", "XxDaemonGamerxX", "Gamer Tag", "I SiK killa 12",
	"test-player", "demo_player", "DEMO", "p1", "Élodie", "日本語タグ",
	strings.Repeat("a", 64),
}

func TestRequirePlayerOwnership_RefusesMalformedSlugInEveryMode(t *testing.T) {
	modes := []struct {
		name     string
		demo     bool
		authMode string
		sess     bool
	}{
		{"démo", true, "", false},
		{"auth inactive", false, "", false},
		{"propriété appliquée, admin", false, "password", true},
	}
	admin := userSession("boss")
	for _, m := range modes {
		for _, slug := range invalidPlayerSlugs {
			t.Run(m.name+"/"+slug, func(t *testing.T) {
				var rr = runSlugGuard(m.demo, m.authMode, slug, nil)
				if m.sess {
					rr = runSlugGuard(m.demo, m.authMode, slug, admin)
				}
				if rr.Code != http.StatusBadRequest {
					t.Fatalf("slug %q : status = %d, attendu 400", slug, rr.Code)
				}
				var body map[string]any
				if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
					t.Fatalf("corps non JSON : %v", err)
				}
				if body[errKeyCode] != "invalid_player_slug" {
					t.Fatalf("code = %v, attendu invalid_player_slug", body[errKeyCode])
				}
			})
		}
	}
}

func TestRequirePlayerOwnership_AcceptsGamertagSlugs(t *testing.T) {
	for _, slug := range legitimatePlayerSlugs {
		t.Run(slug, func(t *testing.T) {
			if rr := runSlugGuard(true, "", slug, nil); rr.Code != http.StatusOK {
				t.Fatalf("slug légitime %q refusé en démo : %d", slug, rr.Code)
			}
			if rr := runSlugGuard(false, "password", slug, userSession("boss")); rr.Code != http.StatusOK {
				t.Fatalf("slug légitime %q refusé à l'admin : %d", slug, rr.Code)
			}
		})
	}
}

// runSlugGuard passe par la garde réelle ; le chemin de la requête est fixe (un slug
// arbitraire n'est pas toujours une URL valide), seul le paramètre chi varie.
func runSlugGuard(demoMode bool, authMode, slug string, sess *domain.SessionData) *httptest.ResponseRecorder {
	mw := RequirePlayerOwnership(NewPlayerAccess(demoMode, authMode, ownershipProfiles, ownershipUsers, nil))
	req := withURLParam(httptest.NewRequest(http.MethodGet, "/players/x/home", nil), "player_slug", slug)
	if sess != nil {
		req = withSession(req, sess)
	}
	rr := httptest.NewRecorder()
	mw(okHandler).ServeHTTP(rr, req)
	return rr
}
