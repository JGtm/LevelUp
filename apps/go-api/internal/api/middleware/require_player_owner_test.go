package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Les notifications d'un profil sont réservées à leur propriétaire, LECTURE comprise
// (ADR 0029) : la garde de route RequirePlayerOwner refuse le co-membre que la garde de
// groupe laisse lire le reste du profil.
func TestRequirePlayerOwner_NotificationsPrivateToOwner(t *testing.T) {
	access := NewPlayerAccess(false, "password", ownershipProfiles, ownershipUsers, aliceShareGroupWithBob)
	chain := func(h http.Handler) http.Handler {
		return RequirePlayerOwnership(access)(RequirePlayerOwner(access)(h))
	}
	cases := []struct {
		name, method, user, slug, sub string
		want                          int
	}{
		{"co-membre lit la liste", http.MethodGet, "alice", "bob", "/notifications", http.StatusForbidden},
		{"co-membre lit le compteur", http.MethodGet, "alice", "bob", "/notifications/unread-count", http.StatusForbidden},
		{"co-membre lit les préférences", http.MethodGet, "alice", "bob", "/notifications/preferences", http.StatusForbidden},
		{"co-membre marque lu", http.MethodPost, "alice", "bob", "/notifications/mark-read", http.StatusForbidden},
		{"propriétaire lit la liste", http.MethodGet, "alice", "alice", "/notifications", http.StatusOK},
		{"propriétaire marque lu", http.MethodPost, "alice", "alice", "/notifications/mark-read", http.StatusOK},
		{"admin lit la liste d'un autre", http.MethodGet, "boss", "bob", "/notifications", http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := familyRequest(tc.method, tc.slug, tc.sub)
			req = withSession(req, userSession(tc.user))
			rr := httptest.NewRecorder()
			chain(okHandler).ServeHTTP(rr, req)
			if rr.Code != tc.want {
				t.Fatalf("%s %s par %s : statut %d, attendu %d (%s)", tc.method, tc.sub, tc.user, rr.Code, tc.want, rr.Body.String())
			}
		})
	}
}

// TestRequirePlayerOwner_TransparentWhenNotEnforced : sans propriété appliquée (démo,
// mono-utilisateur), la garde laisse tout passer.
func TestRequirePlayerOwner_TransparentWhenNotEnforced(t *testing.T) {
	access := NewPlayerAccess(true, "password", ownershipProfiles, ownershipUsers, aliceShareGroupWithBob)
	rr := httptest.NewRecorder()
	RequirePlayerOwner(access)(okHandler).ServeHTTP(rr, familyRequest(http.MethodGet, "bob", "/notifications"))
	if rr.Code != http.StatusOK {
		t.Fatalf("statut %d, attendu 200", rr.Code)
	}
}
