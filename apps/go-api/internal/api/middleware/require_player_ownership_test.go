package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
)

// fakeUserLookup implémente authz.UserLookup pour les tests d'ownership.
type fakeUserLookup struct {
	byName map[string]*domain.User
	byXUID map[string]*domain.User
}

func (f fakeUserLookup) Get(username string) (*domain.User, error) {
	if u, ok := f.byName[username]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}

func (f fakeUserLookup) GetByXUID(xuid string) (*domain.User, error) {
	if u, ok := f.byXUID[xuid]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}

// withURLParam injecte un paramètre d'URL chi (player_slug) dans la requête.
func withURLParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// profilesXUID renvoie un resolver slug → xuid statique pour les tests.
func profilesXUID(m map[string]string) PlayerXUIDResolver {
	return func(_ context.Context, slug string) (string, bool) {
		x, ok := m[slug]
		return x, ok
	}
}

func ownershipRequest(slug string, sess *domain.SessionData) *http.Request {
	req := httptest.NewRequest("GET", "/players/"+slug+"/home", nil)
	req = withURLParam(req, "player_slug", slug)
	if sess != nil {
		req = withSession(req, sess)
	}
	return req
}

func userSession(username string) *domain.SessionData {
	return &domain.SessionData{SessionID: "s", Username: &username}
}

// fixtures partagées
var (
	ownershipProfiles = profilesXUID(map[string]string{"alice": "222", "bob": "999"})
	ownershipUsers    = fakeUserLookup{
		byName: map[string]*domain.User{
			"alice": {Username: "alice", Role: domain.RoleUser, XUID: "222"},
			"boss":  {Username: "boss", Role: domain.RoleAdmin, XUID: "111"},
		},
	}
)

func runOwnership(demoMode bool, authMode, slug string, sess *domain.SessionData) *httptest.ResponseRecorder {
	mw := RequirePlayerOwnership(demoMode, authMode, ownershipProfiles, ownershipUsers)
	rr := httptest.NewRecorder()
	mw(okHandler).ServeHTTP(rr, ownershipRequest(slug, sess))
	return rr
}

func TestRequirePlayerOwnership_NotEnforced_PassThrough(t *testing.T) {
	// auth_mode=none → transparent même sans session ni propriété.
	if rr := runOwnership(false, "none", "bob", nil); rr.Code != http.StatusOK {
		t.Errorf("none mode: status = %d, want 200", rr.Code)
	}
	// demo mode → transparent.
	if rr := runOwnership(true, "password", "bob", nil); rr.Code != http.StatusOK {
		t.Errorf("demo mode: status = %d, want 200", rr.Code)
	}
}

func TestRequirePlayerOwnership_NoSession_PassThrough(t *testing.T) {
	// Contexte sans session (background) → laissé passer (jamais le cas derrière RequireAuth).
	if rr := runOwnership(false, "password", "bob", nil); rr.Code != http.StatusOK {
		t.Errorf("no session: status = %d, want 200", rr.Code)
	}
}

func TestRequirePlayerOwnership_UnknownSlug_PassThrough(t *testing.T) {
	// Slug inconnu → laissé passer (le handler répondra 404 player_not_found).
	if rr := runOwnership(false, "password", "ghost", userSession("alice")); rr.Code != http.StatusOK {
		t.Errorf("unknown slug: status = %d, want 200", rr.Code)
	}
}

func TestRequirePlayerOwnership_Owner_200(t *testing.T) {
	// alice (xuid 222) accède au slug alice (profil xuid 222).
	if rr := runOwnership(false, "password", "alice", userSession("alice")); rr.Code != http.StatusOK {
		t.Errorf("owner: status = %d, want 200", rr.Code)
	}
}

func TestRequirePlayerOwnership_ForeignSlug_403(t *testing.T) {
	// alice tente d'accéder au slug bob (xuid 999) → 403.
	rr := runOwnership(false, "password", "bob", userSession("alice"))
	if rr.Code != http.StatusForbidden {
		t.Errorf("foreign slug: status = %d, want 403", rr.Code)
	}
	if body := rr.Body.String(); !contains(body, "player_forbidden") {
		t.Errorf("corps attendu avec player_forbidden, obtenu %s", body)
	}
}

func TestRequirePlayerOwnership_Admin_200(t *testing.T) {
	// boss (admin) accède à n'importe quel slug.
	if rr := runOwnership(false, "password", "bob", userSession("boss")); rr.Code != http.StatusOK {
		t.Errorf("admin: status = %d, want 200", rr.Code)
	}
}

func TestRequirePlayerOwnership_UnlinkedUser_403(t *testing.T) {
	// Utilisateur connu de session mais absent du user store (non lié) → ne possède rien → 403.
	if rr := runOwnership(false, "password", "alice", userSession("charlie")); rr.Code != http.StatusForbidden {
		t.Errorf("unlinked user: status = %d, want 403", rr.Code)
	}
}

// Test d'intégration avec un VRAI routeur chi : prouve que chi.URLParam(player_slug)
// est bien résolu dans le middleware monté sur le groupe (sinon slug vide →
// pass-through silencieux = trou de sécurité). Reproduit le montage de server.go.
func TestRequirePlayerOwnership_RealChiRouter(t *testing.T) {
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Use(RequirePlayerOwnership(false, "password", ownershipProfiles, ownershipUsers))
		r.Get("/home", okHandler)
	})

	// alice (xuid 222) tente le slug bob (xuid 999) → 403.
	reqForeign := httptest.NewRequest("GET", "/players/bob/home", nil)
	reqForeign = withSession(reqForeign, userSession("alice"))
	rrForeign := httptest.NewRecorder()
	r.ServeHTTP(rrForeign, reqForeign)
	if rrForeign.Code != http.StatusForbidden {
		t.Fatalf("slug étranger via routeur chi: status = %d, want 403", rrForeign.Code)
	}

	// alice → son propre slug → 200.
	reqOwn := httptest.NewRequest("GET", "/players/alice/home", nil)
	reqOwn = withSession(reqOwn, userSession("alice"))
	rrOwn := httptest.NewRecorder()
	r.ServeHTTP(rrOwn, reqOwn)
	if rrOwn.Code != http.StatusOK {
		t.Fatalf("slug possédé via routeur chi: status = %d, want 200", rrOwn.Code)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
