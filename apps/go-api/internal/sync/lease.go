// Package sync — lease.go : façade sur internal/platform/dblease.
//
// Ce fichier délègue entièrement au package dblease pour éviter
// la duplication de la map de mutex et les cycles d'import entre
// internal/sync et internal/platform/duckdb (PersistSink).
//
// Graphe d'import garanti sans cycle :
//
//	internal/sync → internal/platform/dblease
//	internal/platform/duckdb → internal/platform/dblease
//	internal/platform/dblease → (stdlib uniquement)
//
// Usage :
//
//	release, err := AcquireLeaseCtx(ctx, dbPath)
//	if err != nil { return err }
//	defer release()
package sync

import (
	"context"
	"time"

	"levelup/go-api/internal/platform/dblease"
)

// AcquireLease est une façade sur dblease.AcquireLease.
// Préférer AcquireLeaseCtx pour les flux de sync longs.
func AcquireLease(path string, timeout time.Duration) (func(), error) {
	return dblease.AcquireLease(path, timeout)
}

// AcquireLeaseCtx est une façade sur dblease.AcquireLeaseCtx.
// L'attente s'arrête si le contexte est annulé.
func AcquireLeaseCtx(ctx context.Context, path string) (func(), error) {
	return dblease.AcquireLeaseCtx(ctx, path)
}
