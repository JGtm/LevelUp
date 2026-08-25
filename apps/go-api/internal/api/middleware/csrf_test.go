package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/api/middleware"
)

func TestCSRF_GET_Passthrough(t *testing.T) {
	handler := middleware.CSRF([]string{"http://localhost:5173"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bootstrap", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GET should pass through, got %d", rr.Code)
	}
}

func TestCSRF_POST_NoOrigin_Rejected(t *testing.T) {
	handler := middleware.CSRF([]string{"http://localhost:5173"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/session/context", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("POST without Origin should be 403, got %d", rr.Code)
	}
}

func TestCSRF_POST_ValidOrigin_Allowed(t *testing.T) {
	handler := middleware.CSRF([]string{"http://localhost:5173"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/session/context", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("POST with valid Origin should pass, got %d", rr.Code)
	}
}

func TestCSRF_POST_InvalidOrigin_Rejected(t *testing.T) {
	handler := middleware.CSRF([]string{"http://localhost:5173"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/session/context", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("POST with invalid Origin should be 403, got %d", rr.Code)
	}
}

// ─── Exemption ciblée de préfixes (2026-08-25, lot csrf-ouvrier) ─────────────
//
// Les cas ci-dessus décrivent le comportement PAR DÉFAUT (aucun préfixe exempté)
// et doivent rester verts inchangés : l'exemption est opt-in par argument
// variadique. Ceux ci-dessous décrivent la levée ciblée du contrôle d'origine.

const exemptPrefix = "/api/v1/internal"

// csrfExemptHandler : middleware CSRF configuré avec le préfixe du protocole
// ouvrier, sur un handler qui rend 200 s'il est atteint.
func csrfExemptHandler(prefixes ...string) http.Handler {
	return middleware.CSRF([]string{"http://localhost:5173"}, prefixes...)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
}

// Le cas du dry run du 2026-08-25 : un ouvrier (client net/http) n'envoie AUCUN
// Origin ni Referer. Sous le préfixe exempté, il doit atteindre le handler.
func TestCSRF_POST_ExemptPrefix_NoOrigin_Allowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, exemptPrefix+"/build-queue/claim", nil)
	rr := httptest.NewRecorder()
	csrfExemptHandler(exemptPrefix).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("POST sous préfixe exempté sans Origin devrait passer, got %d", rr.Code)
	}
}

// L'exemption lève le contrôle ENTIÈREMENT : une origine hostile ne change rien
// sous ce préfixe. C'est délibéré et sans conséquence — ces routes n'ont pas
// d'autorité ambiante (aucun cookie), leur seule auth est le jeton Bearer, et un
// en-tête Authorization cross-origin déclenche un préflight CORS qui échoue.
func TestCSRF_POST_ExemptPrefix_HostileOrigin_Allowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, exemptPrefix+"/build-queue/heartbeat", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rr := httptest.NewRecorder()
	csrfExemptHandler(exemptPrefix).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("POST sous préfixe exempté devrait passer quelle que soit l'origine, got %d", rr.Code)
	}
}

// Le préfixe exact (sans segment suivant) est exempté ; le préfixe suivi d'AUTRE
// CHOSE qu'un séparateur ne l'est pas — sinon "/api/v1/internalise" hériterait de
// l'exemption par simple préfixe de chaîne.
func TestCSRF_POST_ExemptPrefix_SegmentBoundary(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		wantCode int
	}{
		{"préfixe exact", exemptPrefix, http.StatusOK},
		{"préfixe + slash final", exemptPrefix + "/", http.StatusOK},
		{"sous-route", exemptPrefix + "/build-queue/complete", http.StatusOK},
		{"faux frère (pas une frontière de segment)", "/api/v1/internalise", http.StatusForbidden},
		{"préfixe au milieu du chemin", "/api/v1/admin" + exemptPrefix, http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, nil)
			rr := httptest.NewRecorder()
			csrfExemptHandler(exemptPrefix).ServeHTTP(rr, req)

			if rr.Code != tc.wantCode {
				t.Errorf("POST %q: attendu %d, got %d", tc.path, tc.wantCode, rr.Code)
			}
		})
	}
}

// Traversée : un chemin qui PORTE le préfixe exempté mais DÉSIGNE une route
// protégée ne doit pas hériter de l'exemption. path.Clean normalise avant de
// comparer (r.URL.Path est déjà pourcent-décodé, donc "%2e%2e" est couvert par
// le même chemin de code).
func TestCSRF_POST_ExemptPrefix_TraversalNotExempt(t *testing.T) {
	for _, p := range []string{
		exemptPrefix + "/../session/context",
		exemptPrefix + "/build-queue/../../session/context",
		exemptPrefix + "/%2e%2e/session/context",
	} {
		req := httptest.NewRequest(http.MethodPost, p, nil)
		rr := httptest.NewRecorder()
		csrfExemptHandler(exemptPrefix).ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("POST %q (traversée) devrait rester 403, got %d", p, rr.Code)
		}
	}
}

// Un préfixe vide, racine ou relatif exempterait TOUTE l'API : il doit être
// ignoré, pas appliqué. Faute de câblage → protection intacte.
func TestCSRF_POST_DegenerateExemptPrefix_Ignored(t *testing.T) {
	for _, prefix := range []string{"", "/", "   ", "api/v1/internal"} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/session/context", nil)
		rr := httptest.NewRecorder()
		csrfExemptHandler(prefix).ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("préfixe dégénéré %q ne doit rien exempter, got %d", prefix, rr.Code)
		}
	}
}

// Non-régression du reste de l'API quand une exemption est configurée : une route
// mutatrice hors préfixe reste soumise au contrôle d'origine.
func TestCSRF_POST_OutsideExemptPrefix_StillRejected(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/session/context", nil)
	rr := httptest.NewRecorder()
	csrfExemptHandler(exemptPrefix).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("POST hors préfixe exempté sans Origin devrait rester 403, got %d", rr.Code)
	}
}

func TestCSRF_POST_RefererFallback(t *testing.T) {
	handler := middleware.CSRF([]string{"http://localhost:5173"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/session/context", nil)
	req.Header.Set("Referer", "http://localhost:5173/some/page")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("POST with valid Referer should pass, got %d", rr.Code)
	}
}
