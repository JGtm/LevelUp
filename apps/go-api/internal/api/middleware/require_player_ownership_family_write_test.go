package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Accès famille en LECTURE seule (ADR 0029, décision utilisateur du 2026-10-15) : un
// co-membre de groupe lit le profil d'un ami et n'y écrit pas. La classification
// lecture / écriture est celle de IsMutatingRequest.

// aliceShareGroupWithBob : alice (222) et bob (999) partagent un groupe.
func aliceShareGroupWithBob(context.Context) map[string]bool {
	return map[string]bool{"222": true, "999": true}
}

func familyRequest(method, slug, sub string) *http.Request {
	req := httptest.NewRequest(method, "/api/v1/players/"+slug+sub, strings.NewReader("{}"))
	req = withURLParam(req, "player_slug", slug)
	return withSession(req, userSession("alice"))
}

func TestRequirePlayerOwnership_FamilyIsReadOnly(t *testing.T) {
	mw := RequirePlayerOwnership(NewPlayerAccess(false, "password", ownershipProfiles, ownershipUsers, aliceShareGroupWithBob))
	cases := []struct {
		method, slug, sub string
		want              int
	}{
		{http.MethodGet, "bob", "/home", http.StatusOK},
		{http.MethodPost, "bob", "/pages/media", http.StatusOK}, // POST de lecture
		{http.MethodPost, "bob", "/sync", http.StatusForbidden},
		{http.MethodPatch, "bob", "/matches/m1/exclusion", http.StatusForbidden},
		{http.MethodDelete, "bob", "/media", http.StatusForbidden},
		{http.MethodPut, "bob", "/media/audio-config", http.StatusForbidden},
		{http.MethodPost, "alice", "/sync", http.StatusOK}, // son propre profil
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.slug+tc.sub, func(t *testing.T) {
			rr := httptest.NewRecorder()
			mw(okHandler).ServeHTTP(rr, familyRequest(tc.method, tc.slug, tc.sub))
			if rr.Code != tc.want {
				t.Fatalf("statut %d, attendu %d (%s)", rr.Code, tc.want, rr.Body.String())
			}
		})
	}
}

// TestRequirePlayerOwnership_AdminWritesFriendProfile : l'admin garde l'écriture sur
// tout profil.
func TestRequirePlayerOwnership_AdminWritesFriendProfile(t *testing.T) {
	mw := RequirePlayerOwnership(NewPlayerAccess(false, "password", ownershipProfiles, ownershipUsers, aliceShareGroupWithBob))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/players/bob/sync", nil)
	req = withSession(withURLParam(req, "player_slug", "bob"), userSession("boss"))
	rr := httptest.NewRecorder()
	mw(okHandler).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("admin : statut %d, attendu 200", rr.Code)
	}
}

// TestActorGuard_FamilyActorReadOnly : la garde d'acteur Prestige (slug lu dans le
// corps ou la query, pas dans le chemin) décide avec la classification posée par la
// garde de propriété : agir au nom d'un ami est permis en lecture, refusé en écriture.
func TestActorGuard_FamilyActorReadOnly(t *testing.T) {
	access := NewPlayerAccess(false, "password", ownershipProfiles, ownershipUsers, aliceShareGroupWithBob)
	cases := []struct {
		method string
		want   bool
	}{
		{http.MethodGet, true},
		{http.MethodPost, false},
		{http.MethodDelete, false},
	}
	for _, tc := range cases {
		var granted, sawIntent bool
		probe := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			sawIntent = true
			granted, _ = access.Allows(r.Context(), GetSession(r.Context()), "bob", RequestWrites(r.Context()))
		})
		rr := httptest.NewRecorder()
		RequirePlayerOwnership(access)(probe).ServeHTTP(rr, familyRequest(tc.method, "alice", "/prestige/challenges"))
		if !sawIntent {
			t.Fatalf("%s : la garde n'a pas laissé passer le profil d'alice (%d)", tc.method, rr.Code)
		}
		if granted != tc.want {
			t.Errorf("%s : acteur bob accordé=%v, attendu %v", tc.method, granted, tc.want)
		}
	}
	// Sans classification (requête qui n'est pas passée par la garde) : écriture.
	if !RequestWrites(context.Background()) {
		t.Fatal("RequestWrites sans classification doit valoir écriture")
	}
}
