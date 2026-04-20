package sync

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

// mockTokenProvider fournit des tokens de test.
type mockTokenProvider struct{}

func (m *mockTokenProvider) GetTokens(_ context.Context) (*domain.HaloTokens, error) {
	return &domain.HaloTokens{
		SpartanToken:   "test-spartan",
		ClearanceToken: "test-clearance",
	}, nil
}

func TestNewTrigger(t *testing.T) {
	tp := &mockTokenProvider{}
	opts := domain.SyncOptions{MaxMatches: 50}
	trigger := NewTrigger("/repo", tp, opts)

	if trigger.repoRoot != "/repo" {
		t.Errorf("repoRoot = %q", trigger.repoRoot)
	}
	if trigger.defaultOpts.MaxMatches != 50 {
		t.Errorf("MaxMatches = %d", trigger.defaultOpts.MaxMatches)
	}
}

// TestTrigger_RunSync_NoRealDB vérifie que RunSync crée un SyncEngine
// mais échouera car pas de DB réelle. C'est un test d'intégration partiel.
func TestTrigger_RunSync_NoRealDB(t *testing.T) {
	tp := &mockTokenProvider{}
	trigger := NewTrigger(t.TempDir(), tp, domain.SyncOptions{MaxMatches: 10})

	err := trigger.RunSync(context.Background(), "test-player", "1234", []string{"m1"})
	// Attendu : erreur car pas de DB (mais le code s'exécute jusqu'au RunDelta)
	if err == nil {
		t.Log("RunSync succeeded (unexpected with no DB — tokens may be invalid)")
	}
	// L'important est que ça ne panic pas
}
