package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Le like d'un média est l'action du VISITEUR, attribuée à son propre profil (ADR 0029,
// exception nommée de l'accès famille en lecture seule) : un co-membre qui peut lire le
// profil d'un ami peut y liker un média. Les autres écritures médias restent refusées.
func TestRequirePlayerOwnership_FamilyMayLikeMedia(t *testing.T) {
	mw := RequirePlayerOwnership(NewPlayerAccess(false, "password", ownershipProfiles, ownershipUsers, aliceShareGroupWithBob))
	cases := []struct {
		name, method, slug, sub string
		want                    int
	}{
		{"like sur le profil d'un ami", http.MethodPatch, "bob", mediaLikesSubPath, http.StatusOK},
		{"like sur son profil", http.MethodPatch, "alice", mediaLikesSubPath, http.StatusOK},
		{"suppression sur le profil d'un ami", http.MethodDelete, "bob", "/media", http.StatusForbidden},
		{"association sur le profil d'un ami", http.MethodPost, "bob", "/media/associate", http.StatusForbidden},
		{"autre verbe sur le chemin du like", http.MethodPost, "bob", mediaLikesSubPath, http.StatusForbidden},
		{"chemin du like suivi d'un segment", http.MethodPatch, "bob", "/media/likes/x", http.StatusForbidden},
		{"chemin du like ambigu", http.MethodPatch, "bob", "/media%2Flikes", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			mw(okHandler).ServeHTTP(rr, familyRequest(tc.method, tc.slug, tc.sub))
			if rr.Code != tc.want {
				t.Fatalf("%s %s : statut %d, attendu %d (%s)", tc.method, tc.sub, rr.Code, tc.want, rr.Body.String())
			}
		})
	}
}

// TestRequirePlayerOwnership_LikeOnStrangerProfileRefused : l'exception ne vaut que pour
// un profil LISIBLE ; le profil d'un inconnu reste refusé, like compris.
func TestRequirePlayerOwnership_LikeOnStrangerProfileRefused(t *testing.T) {
	mw := RequirePlayerOwnership(NewPlayerAccess(false, "password", ownershipProfiles, ownershipUsers, nil))
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/players/bob/media/likes", strings.NewReader("{}"))
	req = withSession(withURLParam(req, "player_slug", "bob"), userSession("alice"))
	rr := httptest.NewRecorder()
	mw(okHandler).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("statut %d, attendu 403", rr.Code)
	}
}
