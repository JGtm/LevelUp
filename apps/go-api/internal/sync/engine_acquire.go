// Package sync — engine_acquire.go : helpers d'acquisition writer lease+open.
//
// Extrait de engine.go (refactor 2026-05-21). Regroupe les helpers package-level
// AcquireSharedWriterStandalone / AcquireMetadataWriterStandalone + la méthode
// proxy e.acquireSharedWriter, qui centralisent la prise du dblease applicatif
// et l'ouverture RW des DBs shared et metadata.
//
// Voir engine.go (struct SyncEngine) pour le contexte.
package sync

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/platform/dblease"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// acquireSharedWriter retourne un *sql.DB RW sur shared + une fonction
// release à appeler via defer. Prend en charge le dblease applicatif des
// deux côtés — le caller n'a JAMAIS à prendre `dblease.KindSharedMatches`
// lui-même.
//
//   - Mode B-swap (e.sharedProvider != nil) : appelle Provider.AcquireWriter
//     qui déclenche le mécanisme PreSwap → pool DETACH → OpenReadWrite →
//     RWToRO → pool re-ATTACH. Le Provider prend le dblease en interne
//     (provider.go:231). Le release ferme RW et orchestre le retour en RO.
//   - Mode legacy (e.sharedProvider == nil) : prend explicitement le dblease
//     puis OpenSharedDB direct. Le release ferme le handle ET libère le
//     dblease. Pas de coordination avec le pool — bug "different configuration"
//     reste théoriquement possible (avant le sprint B1).
//
// Sprint B1 commit 11b : centralisation de la prise du dblease. Évite que
// les call sites (run, RunBackfill*) re-prennent le dblease eux-mêmes et
// causent un deadlock auto en mode Provider (sync.Mutex non-réentrant).
func (e *SyncEngine) acquireSharedWriter(ctx context.Context) (*sql.DB, func(), error) {
	return AcquireSharedWriterStandalone(ctx, e.sharedProvider, e.sharedDBPath)
}

// AcquireSharedWriterStandalone est la variante package-level utilisable par
// les fonctions sync qui ne vivent pas sur *SyncEngine (RecomputeIsWithFriends,
// RecalculatePlayerSessions, MatchRecomputer). Même contrat que la méthode
// e.acquireSharedWriter — voir sa godoc.
//
// provider peut être nil : alors fallback legacy (dblease + OpenSharedDB).
// Sprint B1 commit 13b : extraction de l'helper pour migrer les fonctions
// package-level vers le Provider sans dupliquer la logique conditional.
func AcquireSharedWriterStandalone(
	ctx context.Context,
	provider sharedprovider.Provider,
	sharedDBPath string,
) (*sql.DB, func(), error) {
	if provider != nil {
		w, err := provider.AcquireWriter(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("AcquireSharedWriterStandalone via Provider: %w", err)
		}
		return w.DB(), w.Release, nil
	}
	// Mode legacy : prendre le dblease APPLICATIF pour sérialiser les writers
	// concurrents (sans Provider, rien d'autre ne le ferait).
	lease, err := dblease.AcquireWriterCtx(ctx, nil, sharedDBPath, dblease.KindSharedMatches)
	if err != nil {
		return nil, nil, fmt.Errorf("AcquireSharedWriterStandalone legacy lease: %w", err)
	}
	handle, err := OpenSharedDB(sharedDBPath)
	if err != nil {
		lease.Release()
		return nil, nil, fmt.Errorf("AcquireSharedWriterStandalone legacy open: %w", err)
	}
	return handle.SQLDb(), func() {
		_ = handle.Close()
		lease.Release()
	}, nil
}

// AcquireMetadataWriterStandalone est la variante package-level pour metadata.duckdb.
// Prend le lease applicatif (Kind=Metadata) + ouvre via le pool process-level
// (OpenReadWriteShared). Utilisée par les services qui n'ont pas accès à un
// *PlayerDB struct et doivent écrire metadata.duckdb (post-import citations).
//
// Retourne (*sql.DB, releaseFunc, error). Le release ferme le handle ET libère
// le lease — caller via defer.
//
// Sprint chore/ci-stabilization 2026-05-20 : respecte ADR 0013 (pas d'OpenReadWrite
// direct depuis service/handlers).
func AcquireMetadataWriterStandalone(ctx context.Context, metadataPath string) (*sql.DB, func(), error) {
	lease, err := dblease.AcquireWriterCtx(ctx, nil, metadataPath, dblease.KindMetadata)
	if err != nil {
		return nil, nil, fmt.Errorf("AcquireMetadataWriterStandalone lease: %w", err)
	}
	handle, err := duckdbpkg.OpenReadWriteShared(metadataPath)
	if err != nil {
		lease.Release()
		return nil, nil, fmt.Errorf("AcquireMetadataWriterStandalone open: %w", err)
	}
	return handle.SQLDb(), func() {
		_ = handle.Close()
		lease.Release()
	}, nil
}
