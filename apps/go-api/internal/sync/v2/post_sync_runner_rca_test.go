//go:build cgo

// Package v2 — post_sync_runner_rca_test.go : garde-fou anti-régression RC-A.
//
// RC-A (2026-06-01) : le PostSyncRunner V2 recevait son sharedDB via un handle
// READ-ONLY caché (LookupCachedDB), donc tous les heals post-sync (events/skill/
// weapon/registry) échouaient "attached in read-only mode" — silencieusement
// avant le fail-fast Phase 1a. Le fix route le shared via un SharedDBAcquirer qui
// rend un writer RW + release.
//
// Ces tests verrouillent le CONTRAT : le runner DOIT appeler acquireSharedW,
// libérer le writer, et dégrader proprement si l'acquisition échoue ou rend nil.
// Les deux chemins testés court-circuitent AVANT l'engineFactory (jamais appelée).
package v2

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	syncpkg "levelup/go-api/internal/sync"
)

func mkRCAProfile(slug string) PlayerProfile {
	return PlayerProfile{Gamertag: slug, XUID: "x-" + slug, PlayerSlug: slug}
}

func noopPlayerOpener(_ context.Context, _ string) (*sql.DB, func(), error) {
	return &sql.DB{}, func() {}, nil
}

// cheapEngineFactory construit un SyncEngine minimal. Jamais EXÉCUTÉ dans les
// tests RC-A (RunPostSync retourne avant RunPostSyncForV2 sur les chemins
// acquire-échec / shared-nil), mais l'ordre réel est engineFactory → openPlayerDB
// → acquireSharedW, donc la factory doit réussir.
func cheapEngineFactory() SyncEngineFactory {
	return func(_ context.Context, p PlayerProfile) (*syncpkg.SyncEngine, error) {
		return syncpkg.NewSyncEngine(".", p.Gamertag, p.XUID, &domain.HaloTokens{}, nil), nil
	}
}

// Sur acquisition shared en échec : le runner dégrade proprement (pas d'erreur,
// résultat keyed slug) ET a bien TENTÉ d'acquérir le writer (contrat RC-A :
// passer par l'acquirer, jamais un handle RO implicite).
func TestPostSyncRunner_SharedAcquireError_GracefulSkip(t *testing.T) {
	acquireCalled := false
	r := NewPostSyncRunner(
		cheapEngineFactory(),
		noopPlayerOpener,
		func(_ context.Context) (*sql.DB, func(), error) {
			acquireCalled = true
			return nil, nil, errors.New("acquire shared writer failed (provider RO)")
		},
		nil,
	)

	res, err := r.RunPostSync(context.Background(), mkRCAProfile("alice"), nil)
	if err != nil {
		t.Fatalf("RunPostSync doit dégrader sans erreur, got %v", err)
	}
	if !acquireCalled {
		t.Error("acquireSharedW n'a pas été appelé — le runner court-circuite l'acquisition RW (RC-A)")
	}
	if res.PlayerSlug != "alice" {
		t.Errorf("PlayerSlug = %q, want alice", res.PlayerSlug)
	}
}

// Si l'acquirer rend (nil, release, nil) : le runner doit appeler release et
// skipper (pas de heal sur handle nil, pas de fuite de writer).
func TestPostSyncRunner_SharedNil_ReleasesAndSkips(t *testing.T) {
	released := false
	r := NewPostSyncRunner(
		cheapEngineFactory(),
		noopPlayerOpener,
		func(_ context.Context) (*sql.DB, func(), error) {
			return nil, func() { released = true }, nil
		},
		nil,
	)

	res, err := r.RunPostSync(context.Background(), mkRCAProfile("bob"), nil)
	if err != nil {
		t.Fatalf("RunPostSync nil shared : err = %v", err)
	}
	if !released {
		t.Error("release n'a pas été appelé sur le chemin sharedDB nil (fuite de writer)")
	}
	if res.PlayerSlug != "bob" {
		t.Errorf("PlayerSlug = %q, want bob", res.PlayerSlug)
	}
}
