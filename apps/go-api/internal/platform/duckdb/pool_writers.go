// Package duckdb — pool_writers.go : exposition des writers leasés par-DB.
//
// Le commit 1 du refactor "leased-writer-enforcement" introduit *dblease.LeasedWriter
// comme garantie compile-time. Ce fichier ajoute les méthodes d'acquisition à
// *PlayerDB pour que les services et le sync engine n'aient jamais à manipuler
// directement un *string path — ils manipulent l'abstraction par-DB existante.
//
// Référence : .ai/PLAN_DB_WRITE_CONCURRENCY.md commits 0-7.
package duckdb

import (
	"context"
	"errors"
	"time"

	"levelup/go-api/internal/platform/dblease"
)

// ErrSharedSocialUnavailable est retourné quand pdb.SharedSocial est nil
// (DB shared_social.duckdb absente ou non initialisée). Permet aux callers
// de dégrader proprement.
var ErrSharedSocialUnavailable = errors.New("duckdb: shared_social DB not available on this PlayerDB")

// AcquirePlayerWriter acquiert un writer exclusif sur stats.duckdb du joueur.
// Bloque jusqu'à acquisition ou annulation du contexte. Préférer cette variante
// pour les flux pilotés par contexte (sync long, opération HTTP avec deadline).
func (pdb *PlayerDB) AcquirePlayerWriter(ctx context.Context) (*dblease.LeasedWriter, error) {
	return dblease.AcquireWriterCtx(ctx, pdb.Player.SQLDb(), pdb.Player.Path(), dblease.KindPlayer)
}

// AcquirePlayerWriterTimeout acquiert un writer avec un timeout dur.
// Adapté aux handlers HTTP courts qui veulent retourner 503 rapidement.
func (pdb *PlayerDB) AcquirePlayerWriterTimeout(timeout time.Duration) (*dblease.LeasedWriter, error) {
	return dblease.AcquireWriter(pdb.Player.SQLDb(), pdb.Player.Path(), dblease.KindPlayer, timeout)
}

// AcquireSharedSocialWriter acquiert un writer sur shared_social.duckdb.
// Retourne ErrSharedSocialUnavailable si la DB n'est pas disponible (ex. avant
// migration initiale).
func (pdb *PlayerDB) AcquireSharedSocialWriter(ctx context.Context) (*dblease.LeasedWriter, error) {
	if pdb.SharedSocial == nil {
		return nil, ErrSharedSocialUnavailable
	}
	return dblease.AcquireWriterCtx(ctx, pdb.SharedSocial.SQLDb(), pdb.SharedSocial.Path(), dblease.KindSharedSocial)
}

// AcquireSharedSocialWriterTimeout — variante timeout pour HTTP.
func (pdb *PlayerDB) AcquireSharedSocialWriterTimeout(timeout time.Duration) (*dblease.LeasedWriter, error) {
	if pdb.SharedSocial == nil {
		return nil, ErrSharedSocialUnavailable
	}
	return dblease.AcquireWriter(pdb.SharedSocial.SQLDb(), pdb.SharedSocial.Path(), dblease.KindSharedSocial, timeout)
}

// AcquireMetadataWriter acquiert un writer sur metadata.duckdb.
// Note : metadata.duckdb est aussi écrite par le DuckDBIndexStore via WriteQueue
// (asset_index). Les deux mécanismes coexistent — le LeasedWriter protège les
// autres écritures (career enrichment, etc.).
func (pdb *PlayerDB) AcquireMetadataWriter(ctx context.Context) (*dblease.LeasedWriter, error) {
	return dblease.AcquireWriterCtx(ctx, pdb.Metadata.SQLDb(), pdb.Metadata.Path(), dblease.KindMetadata)
}

// AcquireMetadataWriterTimeout — variante timeout.
func (pdb *PlayerDB) AcquireMetadataWriterTimeout(timeout time.Duration) (*dblease.LeasedWriter, error) {
	return dblease.AcquireWriter(pdb.Metadata.SQLDb(), pdb.Metadata.Path(), dblease.KindMetadata, timeout)
}

// PlayerDBPath retourne le chemin du fichier stats.duckdb. Utile pour logging
// structuré et migrations qui nécessitent encore le chemin brut.
func (pdb *PlayerDB) PlayerDBPath() string {
	if pdb.Player == nil {
		return ""
	}
	return pdb.Player.Path()
}

// SharedSocialDBPath retourne le chemin de shared_social.duckdb, ou "" si
// la DB n'est pas disponible.
func (pdb *PlayerDB) SharedSocialDBPath() string {
	if pdb.SharedSocial == nil {
		return ""
	}
	return pdb.SharedSocial.Path()
}
