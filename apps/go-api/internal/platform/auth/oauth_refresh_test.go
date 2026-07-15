// Package auth_test — oauth_refresh_test.go : tests unitaires ExchangeRefreshToken.
package auth_test

import (
	"context"
	"testing"

	"levelup/go-api/internal/platform/auth"
)

// TestExchangeRefreshToken_EmptyToken vérifie que ExchangeRefreshToken retourne
// ("", nil) immédiatement quand le token est vide (sans appel réseau).
func TestExchangeRefreshToken_EmptyToken(t *testing.T) {
	token, err := auth.ExchangeRefreshToken(context.Background(), "")
	if err != nil {
		t.Fatalf("attendu err=nil pour token vide, got: %v", err)
	}
	if token != "" {
		t.Fatalf("attendu access_token vide, got: %q", token)
	}
}

// TestMSALProvider_TryOAuthRefresh_EmptyToken vérifie que TryOAuthRefresh délègue
// correctement avec un token vide.
func TestMSALProvider_TryOAuthRefresh_EmptyToken(t *testing.T) {
	p := auth.NewSISUProvider()
	token, err := p.TryOAuthRefresh(context.Background(), "")
	if err != nil {
		t.Fatalf("attendu err=nil, got: %v", err)
	}
	if token != "" {
		t.Fatalf("attendu token vide, got: %q", token)
	}
}
