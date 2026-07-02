package sync

// shared_access.go — accès à la DB partagée pour le pipeline post-sync, en
// « bursts » paresseux (étape 1 contention, cf. .ai/PLAN_POSTSYNC_BURST_LEASE.md).
//
// PROBLÈME MESURÉ (étape 0) : le post-sync tenait le writer RW pendant TOUT le
// pipeline (~13s/joueur, 99% du cycle), alors que seules 4 étapes écrivent
// réellement dans shared. SharedAccess remplace le *sql.DB RW acquis upfront :
//   - Read(ctx)  : lecture RO (provider.Get) — ne gate personne ;
//   - Write(ctx, step) : burst RW court, labellisé "<label>/<step>" pour la
//     ventilation par-détenteur (carte admin + watchdog étape 0).
// En cycle stationnaire (0 écriture shared), AUCUNE acquisition writer n'a lieu.
//
// Mode pinned (NewPinnedSharedAccess) : compat V1 non-batch — le caller tient
// déjà le writer primaire ; Read/Write retournent ce handle (release no-op),
// comportement byte-identique à l'existant.
//
// GARDE ANTI-DEADLOCK : Write refuse si des Read de CE SharedAccess sont encore
// en vol — AcquireWriter drainerait notre propre RO (auto-contention, cf.
// provider drain + engine.go « fix auto-contention shared »).

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"

	"levelup/go-api/internal/ctxkeys"
)

// SharedAccess arbitre les accès shared du pipeline post-sync.
type SharedAccess struct {
	// isPinned : mode handle déjà tenu par le caller (release no-op) — flag explicite
	// car le handle pinné peut être nil dans les tests minimaux.
	isPinned bool
	pinned   *sql.DB

	// acquire : burst writer RW (typiquement e.acquireSharedWriter / AcquireSharedWriterStandalone).
	acquire func(ctx context.Context) (*sql.DB, func(), error)
	// read : lecteur RO (typiquement SharedReadDB().Get). nil → fallback burst Write
	// (mode legacy sans provider : un handle RW sert aussi les lectures).
	read func(ctx context.Context) (*sql.DB, func(), error)
	// label : détenteur de base (ex. "sync_v2_postsync") — Write suffixe "/<step>".
	label string

	// outstandingReads : Read en vol de CE SharedAccess (garde anti-deadlock Write).
	outstandingReads atomic.Int32

	// probeOnce/probeErr : assertSharedWritable exécuté au 1er burst Write
	// (contrat RC-A : un shared read-only doit faire dégrader les heals, pas
	// échouer silencieusement). Memoïsé — un seul probe par pipeline.
	probeOnce sync.Once
	probeErr  error
}

// NewPinnedSharedAccess construit un SharedAccess sur un handle DÉJÀ TENU par le
// caller (V1 non-batch : writer primaire). Read/Write le retournent tel quel,
// release no-op — byte-identique au comportement historique.
func NewPinnedSharedAccess(db *sql.DB) *SharedAccess {
	return &SharedAccess{isPinned: true, pinned: db}
}

// NewBurstSharedAccess construit un SharedAccess en mode bursts paresseux.
// acquire est obligatoire ; read est optionnel (nil → les lectures passent par
// un burst Write, mode legacy).
func NewBurstSharedAccess(
	acquire func(ctx context.Context) (*sql.DB, func(), error),
	read func(ctx context.Context) (*sql.DB, func(), error),
	label string,
) *SharedAccess {
	return &SharedAccess{acquire: acquire, read: read, label: label}
}

// noopRelease évite d'allouer une closure par appel pinned.
func noopRelease() {}

// Read retourne un handle de LECTURE shared + release (à defer). En mode burst,
// passe par le lecteur RO (aucun writer acquis) ; en pinned, retourne le handle
// tenu. Le release DOIT être appelé avant tout Write (garde anti-deadlock).
func (a *SharedAccess) Read(ctx context.Context) (*sql.DB, func(), error) {
	if a.isPinned {
		return a.pinned, noopRelease, nil
	}
	if a.read == nil {
		// Legacy sans lecteur RO : un burst writer sert la lecture.
		return a.Write(ctx, "read_fallback")
	}
	db, release, err := a.read(ctx)
	if err != nil {
		return nil, nil, err
	}
	a.outstandingReads.Add(1)
	return db, sync.OnceFunc(func() {
		a.outstandingReads.Add(-1)
		release()
	}), nil
}

// Write retourne un handle WRITER shared + release (à defer), pour un burst
// d'écriture COURT (fetch réseau et compute restent HORS de ce burst). step
// suffixe le label de détenteur ("sync_v2_postsync/events") pour l'attribution.
func (a *SharedAccess) Write(ctx context.Context, step string) (*sql.DB, func(), error) {
	if a.isPinned {
		// PAS de probe en pinned : parité stricte avec l'historique — le pipeline
		// V1 sonde lui-même (assertSharedWritable) et CONTINUE en dégradé sur
		// RC-A (trackFatalErr), il n'aborte pas. Le probe bloquant est réservé
		// au mode burst (dégradation PAR ÉTAPE, chaque burst échoue proprement).
		return a.pinned, noopRelease, nil
	}
	if a.acquire == nil {
		return nil, nil, fmt.Errorf("SharedAccess.Write(%s): aucun acquirer câblé", step)
	}
	if n := a.outstandingReads.Load(); n > 0 {
		// AcquireWriter drainerait notre propre lecteur RO → deadlock jusqu'au
		// drain-timeout. Bug de séquencement du caller : release le Read d'abord.
		return nil, nil, fmt.Errorf(
			"SharedAccess.Write(%s): %d Read encore en vol — release avant un burst write (anti-deadlock drain)",
			step, n)
	}
	wctx := ctxkeys.WithDBWriterLabel(ctx, a.label+"/"+step)
	db, release, err := a.acquire(wctx)
	if err != nil {
		return nil, nil, err
	}
	a.probeOnce.Do(func() { a.probeErr = assertSharedWritable(ctx, db) })
	if a.probeErr != nil {
		release()
		return nil, nil, a.probeErr
	}
	return db, release, nil
}
