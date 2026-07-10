package snapshot

// snapshot_shared_reader.go — SharedReader snapshot-préféré (Phase 3 cutover lecture).
//
// Implémente STRUCTURELLEMENT platform/duckdb.SharedReader (Get(ctx) → *sql.DB, release,
// err). Sert les lectures shared depuis la version COURANTE du snapshot immuable
// (read_parquet, hors fenêtre RW du B-swap → jamais stallée), avec FALLBACK GRACIEUX
// vers le SharedReader live si aucun snapshot / snapshot incomplet / erreur. Pas de flag :
// snapshot dès qu'il est disponible, live sinon (dégradation, pas demi-feature OFF).
//
// CACHE VERSIONNÉ : Get est appelé ~17×/page (loadMatchViewDataParallel). Rebâtir la
// DuckDB :memory' (≈10 vues read_parquet) à chaque appel serait absurde → on cache le
// SnapshotQuerier par version et on garde les `snapshotReaderKeep` dernières ouvertes.
// Une version ne change qu'à un cut (minutes d'intervalle) alors qu'une requête vit en
// millisecondes → la release no-op sur un querier caché est sûre (jamais fermé sous une
// requête en vol). Le fallback live, lui, retourne la release du live.

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
)

// snapshotLog : logger taggé module=snapshot → logs/snapshot.log (le package sync route
// sinon vers logs/sync.log). Diagnostic centralisé du sous-système snapshot côté sync
// (fallback lecture, bridge cut). Mémoire cross-package : ops a son propre snapshotLog.
var snapshotLog = slog.With("module", "snapshot")

// snapshotReaderKeep : nombre de versions de querier gardées ouvertes simultanément.
// 3 = marge confortable (la version active + 2 antérieures) pour qu'aucune requête en
// vol ne tombe sur un querier fermé même en cas de cuts rapprochés.
const snapshotReaderKeep = 3

// liveSharedReader : contrat minimal du SharedReader live (évite d'importer
// platform/duckdb ici ; la conformité à duckdb.SharedReader est vérifiée au câblage).
type liveSharedReader interface {
	Get(ctx context.Context) (*sql.DB, func(), error)
}

// SnapshotPreferredSharedReader sert les lectures shared depuis le snapshot courant,
// sinon depuis le live.
type SnapshotPreferredSharedReader struct {
	paths     *title.PathResolver
	titleSlug string
	live      liveSharedReader

	mu    sync.Mutex
	cache map[int64]*ops.SnapshotQuerier
	order []int64 // versions par ordre d'insertion (éviction FIFO au-delà de keep)
	// failedVersion : version pour laquelle OpenSnapshotShared a échoué (ex.
	// ErrSnapshotIncomplete). Negative-cache : on ne retente PAS la reconstruction
	// :memory à chaque Get (~17x/page) tant que la version courante reste celle qui a
	// échoué — on retombe direct sur le live. Reset implicite au changement de version.
	failedVersion int64
}

// NewSnapshotPreferredSharedReader construit le reader. `live` est le SharedReader de
// repli (provider B-swap ou legacy) ; jamais nil.
func NewSnapshotPreferredSharedReader(paths *title.PathResolver, titleSlug string, live liveSharedReader) *SnapshotPreferredSharedReader {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	return &SnapshotPreferredSharedReader{
		paths:     paths,
		titleSlug: titleSlug,
		live:      live,
		cache:     make(map[int64]*ops.SnapshotQuerier),
	}
}

// Get retourne une connexion de lecture shared : snapshot courant si disponible (release
// no-op, querier caché), sinon repli live (release du live).
func (r *SnapshotPreferredSharedReader) Get(ctx context.Context) (*sql.DB, func(), error) {
	if db := r.snapshotDB(ctx); db != nil {
		observability.IncCounterT(r.titleSlug, "snapshot_read_served_total")
		return db, func() {}, nil
	}
	observability.IncCounterT(r.titleSlug, "snapshot_read_live_fallback_total")
	return r.live.Get(ctx)
}

// snapshotDB retourne la *sql.DB du querier de la version courante (caché ou nouvellement
// construit), ou nil si aucun snapshot exploitable (→ repli live).
func (r *SnapshotPreferredSharedReader) snapshotDB(ctx context.Context) *sql.DB {
	version, err := ops.CurrentSnapshotVersion(r.paths, r.titleSlug)
	if err != nil || version == 0 {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if q, ok := r.cache[version]; ok {
		return q.DB
	}
	if version == r.failedVersion {
		return nil // negative-cache : version déjà connue inexploitable → repli live direct
	}
	q, err := ops.OpenSnapshotShared(ctx, r.paths, r.titleSlug)
	if err != nil {
		// Snapshot inexploitable (ErrSnapshotIncomplete = schéma partiel, ou erreur
		// d'ouverture) → repli live. Log UNE fois par version (negative-cache évite le
		// spam ~17x/page) car un schéma partiel persistant désactive silencieusement la
		// lecture-snapshot : c'est un signal opérateur à NE PAS perdre.
		r.failedVersion = version
		snapshotLog.WarnContext(ctx, "snapshot: version inexploitable → lecture live (fallback)",
			"titleSlug", r.titleSlug, "version", version, "err", err)
		return nil
	}
	r.cache[q.Version] = q
	r.order = append(r.order, q.Version)
	r.evictLocked(q.Version)
	return q.DB
}

// evictLocked ferme les queriers au-delà de snapshotReaderKeep (FIFO), sans jamais
// fermer `keepVersion` (la version qu'on vient de servir). Appelé sous r.mu.
func (r *SnapshotPreferredSharedReader) evictLocked(keepVersion int64) {
	for len(r.order) > snapshotReaderKeep {
		oldest := r.order[0]
		r.order = r.order[1:]
		if oldest == keepVersion {
			continue
		}
		if q, ok := r.cache[oldest]; ok {
			q.Close()
			delete(r.cache, oldest)
		}
	}
}

// Close ferme tous les queriers cachés (shutdown). Idempotent.
func (r *SnapshotPreferredSharedReader) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for v, q := range r.cache {
		q.Close()
		delete(r.cache, v)
	}
	r.order = nil
}
