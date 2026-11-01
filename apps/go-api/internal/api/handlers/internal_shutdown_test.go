package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
)

const testShutdownSecret = "secret-d-arret-de-test" // pragma: allowlist secret

// shutdownRouter monte la route comme server_apiv1.go : LoopbackOnly puis le handler.
func shutdownRouter(secret string, request func(string)) http.Handler {
	r := chi.NewRouter()
	h := handlers.NewInternalShutdownHandler(secret, request)
	r.With(middleware.LoopbackOnly).Post("/api/v1/internal/shutdown", h.Handle)
	return r
}

func postShutdown(h http.Handler, remoteAddr, secret string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/shutdown", nil)
	req.RemoteAddr = remoteAddr
	if secret != "" {
		req.Header.Set(handlers.ShutdownSecretHeader, secret)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestInternalShutdown(t *testing.T) {
	tests := []struct {
		name       string
		configured string // secret configuré côté serveur
		remote     string
		sent       string // secret envoyé
		wantStatus int
		wantCalled bool
	}{
		{"bon secret depuis la boucle locale", testShutdownSecret, "127.0.0.1:50000", testShutdownSecret, http.StatusAccepted, true},
		{"bon secret depuis ::1", testShutdownSecret, "[::1]:50000", testShutdownSecret, http.StatusAccepted, true},
		{"mauvais secret", testShutdownSecret, "127.0.0.1:50000", "autre-secret", http.StatusForbidden, false},
		{"secret absent", testShutdownSecret, "127.0.0.1:50000", "", http.StatusForbidden, false},
		{"préfixe du bon secret", testShutdownSecret, "127.0.0.1:50000", testShutdownSecret[:5], http.StatusForbidden, false},
		{"hors boucle locale avec le bon secret", testShutdownSecret, "203.0.113.7:50000", testShutdownSecret, http.StatusForbidden, false},
		{"secret non configuré", "", "127.0.0.1:50000", testShutdownSecret, http.StatusNotFound, false},
		{"secret non configuré et en-tête vide", "", "127.0.0.1:50000", "", http.StatusNotFound, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			reasons := make(chan string, 1)
			h := shutdownRouter(tt.configured, func(reason string) {
				calls.Add(1)
				reasons <- reason
			})
			rec := postShutdown(h, tt.remote, tt.sent)
			if rec.Code != tt.wantStatus {
				t.Fatalf("statut %d, attendu %d (corps %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.sent != "" && strings.Contains(rec.Body.String(), tt.sent) {
				t.Error("le corps de la réponse reprend le secret envoyé")
			}
			if tt.wantCalled {
				select {
				case reason := <-reasons:
					if reason != "http" {
						t.Errorf("cause %q, attendu \"http\"", reason)
					}
				case <-time.After(2 * time.Second):
					t.Fatal("callback d'arrêt jamais appelé")
				}
				return
			}
			time.Sleep(30 * time.Millisecond) // le callback est asynchrone : laisser une chance de l'appeler à tort
			if n := calls.Load(); n != 0 {
				t.Errorf("callback d'arrêt appelé %d fois sur une demande refusée", n)
			}
		})
	}
}

// Sans callback (CLI, tests), la route répond 404 même avec le bon secret.
func TestInternalShutdown_SansCallbackRepond404(t *testing.T) {
	rec := postShutdown(shutdownRouter(testShutdownSecret, nil), "127.0.0.1:50000", testShutdownSecret)
	if rec.Code != http.StatusNotFound {
		t.Errorf("statut %d, attendu 404", rec.Code)
	}
}

// La réponse part avant l'arrêt : un callback lent ne retarde pas le 202.
func TestInternalShutdown_ReponseAvantLArret(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	h := shutdownRouter(testShutdownSecret, func(string) { <-release })
	done := make(chan int, 1)
	go func() { done <- postShutdown(h, "127.0.0.1:50000", testShutdownSecret).Code }()
	select {
	case code := <-done:
		if code != http.StatusAccepted {
			t.Errorf("statut %d, attendu 202", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("la réponse attend la fin du callback d'arrêt")
	}
}
