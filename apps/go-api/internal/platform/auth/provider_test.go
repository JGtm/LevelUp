// Package auth_test — provider_test.go : tests unitaires et de conformité de TokenProvider.
package auth_test

import (
	"context"
	"testing"

	"levelup/go-api/internal/platform/auth"
)

// Vérification compile-time : MSALProvider implémente TokenProvider.
var _ auth.TokenProvider = (*auth.MSALProvider)(nil)

// TestMSALProvider_TrySilentRefresh_EmptyCacheJSON vérifie que TrySilentRefresh
// retourne immédiatement ("", nil) sans appeler MSAL quand le cache est vide.
func TestMSALProvider_TrySilentRefresh_EmptyCacheJSON(t *testing.T) {
	p := auth.NewMSALProvider()
	token, err := p.TrySilentRefresh(context.Background(), "")
	if err != nil {
		t.Fatalf("attendu err=nil pour cacheJSON vide, got: %v", err)
	}
	if token != "" {
		t.Fatalf("attendu token vide pour cacheJSON vide, got: %q", token)
	}
}

// TestMSALProvider_TrySilentRefresh_InvalidJSON vérifie que TrySilentRefresh
// ne panique pas sur un JSON invalide (MSAL retourne token vide ou erreur).
func TestMSALProvider_TrySilentRefresh_InvalidJSON(t *testing.T) {
	p := auth.NewMSALProvider()
	// JSON invalide → NewInMemoryCacheAccessorFromJSON l'accepte (opaque string),
	// AcquireTokenSilent échoue silencieusement et retourne ("", nil).
	token, err := p.TrySilentRefresh(context.Background(), "{invalid}")
	if err != nil {
		// Certaines versions MSAL peuvent retourner une erreur de désérialisation —
		// acceptable : l'important est de ne pas paniquer.
		t.Logf("TrySilentRefresh({invalid}): err=%v (non-bloquant)", err)
	}
	if token != "" {
		t.Fatalf("attendu token vide pour JSON invalide, got: %q", token)
	}
}
