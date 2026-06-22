// Package halo — auth_retry_test.go : garde-fou C3, filet 401 (defense-in-depth).
package halo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

const validBattlePassBody = `{"ActiveOperationRewardTrackPath":"track1",` +
	`"OperationRewardTracks":[{"RewardTrackPath":"track1","CurrentProgress":{"Rank":5,"PartialProgress":10}}]}`

// 401 puis 200 après re-mint : le filet invalide le cache, re-minte et réessaie UNE
// fois → succès. Exactement 2 hits serveur, 1 re-mint.
func TestRetryOnAuth_401ThenSuccess(t *testing.T) {
	resetPlayerTokenStore()
	prev := playerTokenRefresher
	defer SetPlayerTokenRefresher(prev)

	var serverHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&serverHits, 1) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(validBattlePassBody))
	}))
	defer srv.Close()

	var mintCount int32
	SetPlayerTokenRefresher(func(_ context.Context, _ string) (*domain.HaloTokens, error) {
		atomic.AddInt32(&mintCount, 1)
		return &domain.HaloTokens{SpartanToken: "fresh", SpartanExpiresAt: time.Now().Add(time.Hour)}, nil
	})

	p := newTestProvider(srv.URL, "")
	ctx := ctxWithAuth(testTokens(), "xuid-retry")

	resp, raw := p.GetBattlePassWithRaw(ctx)
	if !resp.Available {
		t.Fatalf("attendu Available=true après retry, got %+v", resp)
	}
	if len(raw) == 0 {
		t.Error("attendu un body brut après succès du retry")
	}
	if got := atomic.LoadInt32(&serverHits); got != 2 {
		t.Errorf("attendu 2 hits serveur (401 + retry 200), got %d", got)
	}
	if got := atomic.LoadInt32(&mintCount); got != 1 {
		t.Errorf("attendu 1 re-mint, got %d", got)
	}
}

// 401 puis ENCORE 401 (révocation réelle) : retry strictement 1× → exactement 2 hits,
// pas de boucle ; available=false (dégradation honnête en aval).
func TestRetryOnAuth_401Twice_NoLoop(t *testing.T) {
	resetPlayerTokenStore()
	prev := playerTokenRefresher
	defer SetPlayerTokenRefresher(prev)

	var serverHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&serverHits, 1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	SetPlayerTokenRefresher(func(_ context.Context, _ string) (*domain.HaloTokens, error) {
		return &domain.HaloTokens{SpartanToken: "still-rejected", SpartanExpiresAt: time.Now().Add(time.Hour)}, nil
	})

	p := newTestProvider(srv.URL, "")
	ctx := ctxWithAuth(testTokens(), "xuid-revoked")

	resp, _ := p.GetBattlePassWithRaw(ctx)
	if resp.Available {
		t.Error("attendu Available=false quand le retry échoue aussi")
	}
	if got := atomic.LoadInt32(&serverHits); got != 2 {
		t.Errorf("retry doit être STRICTEMENT 1× → 2 hits, got %d (boucle ?)", got)
	}
}

// RetryWithFreshTokens avec un predicate CUSTOM (chemin client sync : career/CSR/
// recent-matches passent sync.IsAuthError). Vérifie que le predicate pilote bien le retry.
func TestRetryWithFreshTokens_CustomPredicate(t *testing.T) {
	resetPlayerTokenStore()
	prev := playerTokenRefresher
	defer SetPlayerTokenRefresher(prev)

	var mintCount int32
	SetPlayerTokenRefresher(func(_ context.Context, _ string) (*domain.HaloTokens, error) {
		atomic.AddInt32(&mintCount, 1)
		return &domain.HaloTokens{SpartanToken: "fresh", SpartanExpiresAt: time.Now().Add(time.Hour)}, nil
	})

	sentinel := errors.New("custom auth boom")
	isAuth := func(err error) bool { return errors.Is(err, sentinel) }

	// Cas 1 : erreur qui matche le predicate → re-mint + retry → succès au 2e appel.
	var calls int32
	ctx := ctxWithAuth(testTokens(), "xuid-custom")
	got, err := RetryWithFreshTokens(ctx, isAuth, func(_ context.Context) (string, error) {
		if atomic.AddInt32(&calls, 1) == 1 {
			return "", sentinel
		}
		return "ok", nil
	})
	if err != nil || got != "ok" {
		t.Fatalf("attendu succès après retry, got %q err=%v", got, err)
	}
	if calls != 2 {
		t.Errorf("attendu 2 appels (1 échec auth + 1 retry), got %d", calls)
	}
	if atomic.LoadInt32(&mintCount) != 1 {
		t.Errorf("attendu 1 re-mint, got %d", mintCount)
	}

	// Cas 2 : erreur qui NE matche PAS le predicate → aucun retry.
	var calls2 int32
	other := errors.New("not auth")
	_, err = RetryWithFreshTokens(ctx, isAuth, func(_ context.Context) (string, error) {
		atomic.AddInt32(&calls2, 1)
		return "", other
	})
	if !errors.Is(err, other) {
		t.Errorf("erreur non-auth doit remonter telle quelle, got %v", err)
	}
	if calls2 != 1 {
		t.Errorf("erreur non-auth → pas de retry → 1 appel, got %d", calls2)
	}
}

// Sans refresher câblé (cas test/boot incomplet), aucun retry : 1 seul hit, comportement
// pré-fix conservé (nil-safe).
func TestRetryOnAuth_NoRefresher_NoRetry(t *testing.T) {
	resetPlayerTokenStore()
	prev := playerTokenRefresher
	defer SetPlayerTokenRefresher(prev)
	SetPlayerTokenRefresher(nil)

	var serverHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&serverHits, 1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := newTestProvider(srv.URL, "")
	ctx := ctxWithAuth(testTokens(), "xuid-no-refresher")

	resp, _ := p.GetBattlePassWithRaw(ctx)
	if resp.Available {
		t.Error("attendu Available=false sans refresher")
	}
	if got := atomic.LoadInt32(&serverHits); got != 1 {
		t.Errorf("sans refresher, pas de retry → 1 hit, got %d", got)
	}
}
