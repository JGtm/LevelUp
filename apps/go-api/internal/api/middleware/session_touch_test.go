// Package middleware_test — session_touch_test.go : la persistance de fin de requête ne
// réécrit plus le fichier de session à chaque requête (plan perf du 2026-09-23, D3.5).
package middleware_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/session"
)

// TestWithSession_DeuxRequetesAUneSeconde_UneEcriture : deux requêtes à 1 s d'intervalle
// sur une session inchangée n'écrivent son fichier qu'UNE fois (la première rafraîchit un
// last_seen_at vieux de 10 min, la seconde s'abstient) ; une requête qui modifie la session
// l'écrit aussitôt.
func TestWithSession_DeuxRequetesAUneSeconde_UneEcriture(t *testing.T) {
	sessDir := filepath.Join(t.TempDir(), "sessions")
	clock := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	store := session.NewStore(sessDir, session.DefaultTTL, "test-secret-32bytesXXXXXXXXXXX",
		session.WithClock(func() time.Time { return clock }))
	sess := store.New()
	user := "alice"
	sess.Username = &user
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save : %v", err)
	}
	signed := store.SignCookie(sess.SessionID)
	path := filepath.Join(sessDir, sess.SessionID+".json")

	var mutate func(*domain.SessionData)
	handler := middleware.WithSession(store, middleware.SecureCookiePolicy{})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if mutate != nil {
				mutate(middleware.GetSession(r.Context()))
			}
			w.WriteHeader(http.StatusOK)
		}))
	serve := func() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: signed})
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}
	read := func() ([]byte, domain.SessionData) {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("lecture du fichier de session : %v", err)
		}
		var got domain.SessionData
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("fichier de session illisible : %v", err)
		}
		return raw, got
	}

	clock = clock.Add(10 * time.Minute)
	serve() // requête 1 : last_seen_at persisté vieux de 10 min → UNE écriture
	afterFirst, first := read()
	if first.LastSeenAt != clock.Unix() {
		t.Fatalf("requête 1 : last_seen_at = %d, attendu %d (le TTL glissant doit être rafraîchi)",
			first.LastSeenAt, clock.Unix())
	}

	clock = clock.Add(time.Second)
	serve() // requête 2, 1 s plus tard, session inchangée → AUCUNE écriture
	if afterSecond, _ := read(); !bytes.Equal(afterSecond, afterFirst) {
		t.Errorf("requête 2 (1 s après, session inchangée) a réécrit le fichier :\n avant %s\n après %s",
			afterFirst, afterSecond)
	}

	clock = clock.Add(time.Second)
	mutate = func(s *domain.SessionData) { s.Locale = "en" }
	serve() // requête 3 : la session change → écrite aussitôt
	if _, third := read(); third.Locale != "en" || third.LastSeenAt != clock.Unix() {
		t.Errorf("requête 3 (session modifiée) non écrite : locale %q, last_seen_at %d",
			third.Locale, third.LastSeenAt)
	}
}
