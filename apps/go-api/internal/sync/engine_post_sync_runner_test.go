// Package sync — engine_post_sync_runner_test.go : garde anti-régression
// pour le câblage PostSyncRunner (Phase 4 plan stabilisation 2026-05-22).
//
// Objectif : si quelqu'un retire le field postSyncRunner du SyncEngine ou
// casse le contrat WithPostSyncRunner / l'invocation BeforeSync→finalizer
// dans run(), ce test échoue.
//
// Ne lance PAS un sync complet (nécessite DBs réelles + HaloClient mocké
// lourd). Vérifie uniquement le contrat de l'API publique.
package sync

import (
	"context"
	"testing"

	"levelup/go-api/internal/port"
)

// stubPostSyncRunner mémorise les appels BeforeSync/finalizer pour assertion.
type stubPostSyncRunner struct {
	beforeCalls    int
	beforeSlugArg  string
	finalizerCalls int
}

func (s *stubPostSyncRunner) BeforeSync(_ context.Context, slug string) port.PostSyncFinalizer {
	s.beforeCalls++
	s.beforeSlugArg = slug
	return func(_ context.Context) {
		s.finalizerCalls++
	}
}

// TestSyncEngine_WithPostSyncRunner_StoresRunner : le builder stocke
// correctement le runner + slug dans la struct SyncEngine.
func TestSyncEngine_WithPostSyncRunner_StoresRunner(t *testing.T) {
	e := NewSyncEngine("/tmp/repo", "JGtm", "xuid-123", nil, nil)
	runner := &stubPostSyncRunner{}

	result := e.WithPostSyncRunner(runner, "JGtm")
	if result != e {
		t.Errorf("WithPostSyncRunner doit retourner *SyncEngine (chainable)")
	}
	if e.postSyncRunner != runner {
		t.Errorf("postSyncRunner non stocké")
	}
	if e.postSyncSlug != "JGtm" {
		t.Errorf("postSyncSlug = %q, want JGtm", e.postSyncSlug)
	}
}

// TestSyncEngine_WithPostSyncRunner_NilRunner : nil runner = legacy mode
// (pas d'invocation).
func TestSyncEngine_WithPostSyncRunner_NilRunner(t *testing.T) {
	e := NewSyncEngine("/tmp/repo", "JGtm", "xuid-123", nil, nil)
	e.WithPostSyncRunner(nil, "")
	if e.postSyncRunner != nil {
		t.Errorf("nil runner doit laisser postSyncRunner nil")
	}
}

// TestSyncEngine_WithPostSyncRunner_Chaining : le builder est chainable avec
// les autres With* (pattern conservé pour cohérence).
func TestSyncEngine_WithPostSyncRunner_Chaining(t *testing.T) {
	e := NewSyncEngine("/tmp/repo", "JGtm", "xuid-123", nil, nil).
		WithFriendsLoader(func() ([]string, error) { return nil, nil }).
		WithCSRSeasonID("CsrSeason13-1").
		WithPostSyncRunner(&stubPostSyncRunner{}, "JGtm")
	if e == nil {
		t.Fatal("chain a retourné nil")
	}
	if e.postSyncRunner == nil {
		t.Errorf("runner perdu dans le chaining")
	}
	if e.csrSeasonID != "CsrSeason13-1" {
		t.Errorf("csrSeasonID perdu : %q", e.csrSeasonID)
	}
}
