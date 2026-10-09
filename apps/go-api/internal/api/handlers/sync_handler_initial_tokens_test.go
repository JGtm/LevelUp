package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
)

// postInitialSync envoie une sync initiale valide, avec la session `sess` (nil = aucune).
func postInitialSync(t *testing.T, sess *domain.SessionData) *httptest.ResponseRecorder {
	t.Helper()
	r, _ := newSyncRouter(t, true)
	body, _ := json.Marshal(map[string]interface{}{"player_slug": "test-player", "max_matches": 100})
	req := httptest.NewRequest(http.MethodPost, "/sync/initial", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if sess != nil {
		req = req.WithContext(middleware.InjectSession(req.Context(), sess))
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// Sans session : 401 `auth_required` (la coquille recharge vers la connexion, c'est voulu).
func TestSyncHandler_InitialSync_SansSession401(t *testing.T) {
	w := postInitialSync(t, nil)
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), `"auth_required"`) {
		t.Fatalf("attendu 401 auth_required, obtenu %d : %s", w.Code, w.Body.String())
	}
}

// Session valide sans jetons Halo : 403 `halo_tokens_missing`, JAMAIS 401 — un 401
// `auth_required` ferait recharger la coquille vers la connexion alors que la session vit.
func TestSyncHandler_InitialSync_SessionSansJetonsHalo403(t *testing.T) {
	username := "alice"
	w := postInitialSync(t, &domain.SessionData{Username: &username})
	if w.Code != http.StatusForbidden {
		t.Fatalf("attendu 403, obtenu %d : %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"halo_tokens_missing"`) {
		t.Fatalf("code d'erreur attendu halo_tokens_missing : %s", w.Body.String())
	}
}
