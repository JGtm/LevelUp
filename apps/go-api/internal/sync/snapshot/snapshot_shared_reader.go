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
//
// LA CONSTRUCTION D'UNE VERSION N'EST JAMAIS SUR LE CHEMIN D'UNE REQUÊTE. Matérialiser les
// tables d'une version (match_kill_events dépasse 100 Mio de Parquet) prend plusieurs
// secondes sur un petit serveur. Sous le verrou du reader et avec le contexte d'une requête,
// elle bloquerait toutes les lectures concurrentes jusqu'à l'échéance de la plus courte
// (5 s pour la vérification de participation), et un échec sur cette échéance serait
// mémorisé comme une version inexploitable. Elle tourne donc en tâche de fond, détachée de
// l'annulation de la requête et bornée par sa propre échéance ; pendant qu'elle tourne, Get
// sert le live.

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"

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

// snapshotBuildTimeout borne la construction d'une version en tâche de fond. Elle n'est
// liée à aucune requête : au-delà, la version est tenue pour inexploitable (repli live
// jusqu'au cut suivant) — un signal d'exploitation, pas une échéance de requête.
const snapshotBuildTimeout = 2 * time.Minute

// liveSharedReader : contrat minimal du SharedReader live (évite d'importer
// platform/duckdb ici ; la conformité à duckdb.SharedReader est vérifiée au câblage).
type liveSharedReader interface {
	Get(ctx context.Context) (*sql.DB, func(), error)
}

// openSnapshotFunc construit le querier de la version courante (ops.OpenSnapshotShared
// en production ; remplaçable par les tests).
type openSnapshotFunc func(ctx context.Context, paths *title.PathResolver, titleSlug string) (*ops.SnapshotQuerier, error)

// SnapshotPreferredSharedReader sert les lectures shared depuis le snapshot courant,
// sinon depuis le live.
type SnapshotPreferredSharedReader struct {
	paths     *title.PathResolver
	titleSlug string
	live      liveSharedReader
	open      openSnapshotFunc

	mu    sync.Mutex
	cache map[int64]*ops.SnapshotQuerier
	order []int64 // versions par ordre d'insertion (éviction FIFO au-delà de keep)
	// failedVersion : version pour laquelle la construction a échoué (ex.
	// ErrSnapshotIncomplete, échéance de construction). Negative-cache : on ne retente PAS
	// la reconstruction :memory à chaque Get (~17x/page) tant que la version courante reste
	// celle qui a échoué — on retombe direct sur le live. Reset implicite au changement de
	// version.
	failedVersion int64
	// building : version dont la construction tourne en tâche de fond (0 = aucune). Une
	// seule construction à la fois : deux versions matérialisées en parallèle doubleraient
	// le pic mémoire.
	building int64
	closed   bool
	// builds compte les constructions en cours (les tests l'attendent).
	builds sync.WaitGroup
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
		open:      ops.OpenSnapshotShared,
		cache:     make(map[int64]*ops.SnapshotQuerier),
	}
}

// Get retourne une connexion de lecture shared : snapshot courant si disponible (release
// no-op, querier caché), sinon repli live (release du live). Ne bloque jamais sur la
// construction d'une version.
func (r *SnapshotPreferredSharedReader) Get(ctx context.Context) (*sql.DB, func(), error) {
	if db := r.snapshotDB(ctx); db != nil {
		observability.IncCounterT(r.titleSlug, "snapshot_read_served_total")
		return db, func() {}, nil
	}
	observability.IncCounterT(r.titleSlug, "snapshot_read_live_fallback_total")
	return r.live.Get(ctx)
}

// snapshotDB retourne la *sql.DB du querier de la version courante s'il est déjà construit,
// sinon nil (→ repli live) après avoir lancé sa construction en tâche de fond.
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
	if version == r.failedVersion || r.closed {
		return nil // negative-cache : version déjà connue inexploitable → repli live direct
	}
	if r.building == 0 {
		r.building = version
		r.builds.Add(1)
		// WithoutCancel : la construction garde les valeurs du contexte (journal) mais
		// survit à la fin de la requête qui l'a déclenchée.
		go r.build(context.WithoutCancel(ctx), version)
	}
	return nil
}

// build matérialise la version courante et la publie dans le cache. Un échec la marque
// inexploitable (journalisé une fois par version : un schéma partiel persistant désactive
// la lecture-snapshot, c'est un signal opérateur à NE PAS perdre).
func (r *SnapshotPreferredSharedReader) build(ctx context.Context, version int64) {
	defer r.builds.Done()
	ctx, cancel := context.WithTimeout(ctx, snapshotBuildTimeout)
	defer cancel()
	start := time.Now()
	q, err := r.open(ctx, r.paths, r.titleSlug)

	r.mu.Lock()
	defer r.mu.Unlock()
	r.building = 0
	if err != nil {
		r.failedVersion = version
		snapshotLog.WarnContext(ctx, "snapshot: version inexploitable → lecture live (fallback)",
			"titleSlug", r.titleSlug, "version", version, "err", err,
			"duration_ms", time.Since(start).Milliseconds())
		return
	}
	if _, dup := r.cache[q.Version]; dup || r.closed {
		q.Close()
		return
	}
	r.cache[q.Version] = q
	r.order = append(r.order, q.Version)
	r.evictLocked(q.Version)
	snapshotLog.InfoContext(ctx, "snapshot: version prête pour la lecture",
		"titleSlug", r.titleSlug, "version", q.Version, "duration_ms", time.Since(start).Milliseconds())
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

// Close ferme tous les queriers cachés (shutdown). Idempotent. Une construction en cours
// ferme son querier à son terme au lieu de le publier.
func (r *SnapshotPreferredSharedReader) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	for v, q := range r.cache {
		q.Close()
		delete(r.cache, v)
	}
	r.order = nil
}
