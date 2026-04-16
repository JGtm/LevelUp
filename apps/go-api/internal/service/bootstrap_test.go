// Tests nécessitant CGO (dépendance transitive via service → config → duckdb).
// CGO_ENABLED=1 go test ./internal/service/ -run TestBootstrap -v

//go:build cgo

package service_test

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/service"
)

// TestBootstrapAuthState vérifie que Build() calcule l'AuthState correctement
// en fonction de la session fournie.
func TestBootstrapAuthState(t *testing.T) {
	tests := []struct {
		name     string
		sess     *domain.SessionData
		expected string
	}{
		{
			name:     "nil session → missing",
			sess:     nil,
			expected: "missing",
		},
		{
			name:     "session sans auth → missing",
			sess:     &domain.SessionData{AuthReady: false},
			expected: "missing",
		},
		{
			name:     "auth ready sans identité → partial",
			sess:     &domain.SessionData{AuthReady: true},
			expected: "partial",
		},
		{
			name:     "auth ready avec identité vide → partial",
			sess:     &domain.SessionData{AuthReady: true, LinkedHaloIdentity: &domain.HaloIdentity{}},
			expected: "partial",
		},
		{
			name: "auth ready avec identité complète → ready",
			sess: &domain.SessionData{
				AuthReady:          true,
				LinkedHaloIdentity: &domain.HaloIdentity{Gamertag: "TestPlayer", XUID: "1234"},
			},
			expected: "ready",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := service.ResolveAuthState(tc.sess)
			if got != tc.expected {
				t.Errorf("ResolveAuthState() = %q, want %q", got, tc.expected)
			}
		})
	}
}

// TestBootstrapLinkedIdentity vérifie l'extraction de l'identité Halo depuis la session.
func TestBootstrapLinkedIdentity(t *testing.T) {
	tests := []struct {
		name      string
		sess      *domain.SessionData
		expectNil bool
		expectGT  string
	}{
		{
			name:      "nil session → nil",
			sess:      nil,
			expectNil: true,
		},
		{
			name:      "session sans identité → nil",
			sess:      &domain.SessionData{},
			expectNil: true,
		},
		{
			name: "session avec identité → résolu",
			sess: &domain.SessionData{
				LinkedHaloIdentity: &domain.HaloIdentity{Gamertag: "GT1", XUID: "X1"},
			},
			expectNil: false,
			expectGT:  "GT1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := service.ResolveLinkedIdentity(tc.sess)
			if tc.expectNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil identity")
			}
			if got.Gamertag != tc.expectGT {
				t.Errorf("Gamertag = %q, want %q", got.Gamertag, tc.expectGT)
			}
		})
	}
}
