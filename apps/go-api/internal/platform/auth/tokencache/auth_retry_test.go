// Package tokencache — auth_retry_test.go : le filet 401 (defense-in-depth) piloté par le
// prédicat de l'appelant.
package tokencache

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

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
	ctx := ctxkeys.WithHaloAuth(context.Background(), &domain.HaloTokens{SpartanToken: "spartan_test", ClearanceToken: "clear_test"}, "xuid-custom")
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
