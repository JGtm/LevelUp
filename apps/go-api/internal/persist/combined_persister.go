// Package persist — combined_persister.go : BatchPersister qui écrit shared + player
// en un seul appel, avec gestion des leases par batch.
//
// Utilisé par le Worker async (Phase 3 du refactor Collect→Persist) pour écrire
// les batches dans les deux DBs sans que le caller (main.go) ait à gérer les
// connexions lui-même.
//
// Pour éviter un cycle d'import (persist ↔ sync), l'acquisition de la connexion
// shared est injectée via SharedWriterFn — le caller (main.go) fournit la closure
// appropriée selon le mode (Provider B-swap ou legacy dblease).
//
// Chaque appel Persist ouvre ses propres connexions et les ferme à la fin.
// Acceptable pour notre volume (<50 matchs / cycle) — la durée d'une connexion
// DuckDB ≈ 1ms sur SSD local.

package persist

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/platform/dblease"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

// SharedWriterFn acquiert une connexion RW sur shared_matches_v2.duckdb avec
// le lease exclusif approprié (Provider B-swap ou dblease legacy).
// Le caller doit appeler la fonction release() retournée via defer.
type SharedWriterFn func(ctx context.Context) (*sql.DB, func(), error)

// OnPersistPhase (observabilité, optionnel) — hook branché par main.go pour
// mesurer les phases d'écriture SANS coupler persist à observability (même
// philosophie que Worker.OnPersistOK/OnPersistError). Posé une fois au boot,
// jamais muté ensuite. phase ∈ {shared_acquire, shared_write, player_lease,
// player_write}. NB : shared_write inclut les sous-batches PVE/metadata
// (portés par SharedPersister).
var OnPersistPhase func(phase string, d time.Duration, ok bool)

// observePersistPhase mesure une phase si le hook est branché (no-op sinon).
func observePersistPhase(phase string, start time.Time, ok bool) {
	if OnPersistPhase != nil {
		OnPersistPhase(phase, time.Since(start), ok)
	}
}

// CombinedPersister implémente BatchPersister pour shared + player en un seul
// appel. Conçu pour être utilisé par un Worker goroutine unique.
type CombinedPersister struct {
	acquireShared  SharedWriterFn
	playerDBPathFn func(titleSlug, gamertag string) string
}

// NewCombinedPersister construit un CombinedPersister.
//
//   - acquireShared  : fonction qui ouvre shared_matches_v2.duckdb en RW avec lease.
//     Construire depuis main.go avec le Provider (B-swap) ou OpenSharedDB (legacy).
//   - playerDBPathFn : retourne le chemin complet de stats.duckdb pour un
//     (titleSlug, gamertag). Le titleSlug vient du batch (batch.TitleSlug) — chaque
//     batch route vers la player DB de SON titre (multi-titres : ne JAMAIS figer le
//     slug au boot, sinon les batchs d'un 2e titre actif sont mal dirigés).
func NewCombinedPersister(acquireShared SharedWriterFn, playerDBPathFn func(titleSlug, gamertag string) string) *CombinedPersister {
	return &CombinedPersister{
		acquireShared:  acquireShared,
		playerDBPathFn: playerDBPathFn,
	}
}

// Persist écrit le batch dans shared puis dans player, chacun avec son propre
// lease et sa propre connexion. Atomique par DB (1 TX / DB), pas cross-DB.
//
// Ordre : shared AVANT player — si le shared fail, on ne touche pas player
// (le Worker loggue l'erreur et ne supprime pas le WAL → retry au prochain boot).
func (p *CombinedPersister) Persist(ctx context.Context, batch *MatchBatch) error {
	if batch == nil {
		return fmt.Errorf("CombinedPersister.Persist: batch nil")
	}

	// ── 1. Shared DB ─────────────────────────────────────────────────────────
	acquireStart := time.Now()
	sharedDB, releaseShared, err := p.acquireShared(ctx)
	observePersistPhase("shared_acquire", acquireStart, err == nil)
	if err != nil {
		return fmt.Errorf("CombinedPersister: acquire shared: %w", err)
	}

	writeStart := time.Now()
	// releaseShared sous defer d'une closure : la libération reste ANTICIPÉE
	// (writer rendu avant la phase player, comme avant), mais elle survit
	// désormais à un panic dans SharedPersister.Persist. Sans ce defer, un panic
	// pendant l'écriture laissait le writer RW tenu jusqu'au restart du process
	// (verrou shared pour tout le monde — scénario d'incident verrou réel).
	var sharedErr error
	func() {
		defer releaseShared()
		sharedErr = NewSharedPersister(sharedDB).Persist(ctx, batch)
		if sharedErr != nil {
			return
		}
		// Passe de décodage du film (match_kill_events, append-only). NO-OP tant
		// que batch.Shared.KillSource est nil — c'est le cas de la quasi-totalité
		// des batches : le film n'existe pas encore au sync primaire, le collecteur
		// écrit donc par le chemin direct PersistPass. Câblé quand même : un
		// SetKillSource() côté builder dont la charge serait silencieusement jetée
		// serait une perte de données MUETTE, exactement ce que ce package combat.
		// Dans la MÊME fenêtre de lease que le SharedPersister ; transaction
		// distincte — la table est append-only, elle n'a pas à partager
		// l'atomicité du registry.
		sharedErr = NewKillSourcePersister(sharedDB).Persist(ctx, batch)
		if sharedErr != nil {
			return
		}
		// Ventilation des tirs par arme (match_weapon_shots, append-only). Même
		// raisonnement : NO-OP tant que batch.Shared.WeaponShots est nil, câblé
		// quand même pour qu'un SetWeaponShots() ne puisse pas être silencieusement
		// jeté. Transaction distincte, même fenêtre de lease.
		sharedErr = NewWeaponShotsPersister(sharedDB).Persist(ctx, batch)
	}()
	observePersistPhase("shared_write", writeStart, sharedErr == nil)

	if sharedErr != nil {
		slog.ErrorContext(ctx, "CombinedPersister: shared persist échoué",
			"batch_id", batch.BatchID, "player", batch.Player, "err", sharedErr)
		return fmt.Errorf("CombinedPersister: shared: %w", sharedErr)
	}

	// ── 2. Player DB ──────────────────────────────────────────────────────────
	playerPath := p.playerDBPathFn(batch.TitleSlug, batch.Player)
	if playerPath == "" {
		slog.WarnContext(ctx, "CombinedPersister: playerDBPath vide, skip player persist",
			"batch_id", batch.BatchID, "player", batch.Player)
		return nil
	}

	playerCtx, playerCancel := context.WithTimeout(ctx, 30*time.Second)
	defer playerCancel()

	leaseStart := time.Now()
	playerLease, leaseErr := dblease.AcquireLeaseCtx(playerCtx, playerPath)
	observePersistPhase("player_lease", leaseStart, leaseErr == nil)
	if leaseErr != nil {
		return fmt.Errorf("CombinedPersister: player lease %s: %w", batch.Player, leaseErr)
	}
	defer playerLease()

	// Phase 1 du PLAN_FIX_SYNC_RELIABILITY_2026-05-24 : passage par le cache
	// process-level duckdbpkg.OpenReadWrite (DSN nu) au lieu de sql.Open direct
	// avec "?access_mode=READ_WRITE". Le DSN explicite causait un conflit
	// "Can't open a connection with a different configuration" avec les autres
	// sites qui ouvrent via le cache (engine.go::OpenPlayerDB). Cache key
	// "rw:"+path est partage entre tous les callers → 0 conflit possible.
	playerHandle, openErr := duckdbpkg.OpenReadWrite(playerPath)
	if openErr != nil {
		return fmt.Errorf("CombinedPersister: open player %s: %w", batch.Player, openErr)
	}
	defer playerHandle.Close()
	playerDB := playerHandle.SQLDb()

	playerWriteStart := time.Now()
	playerErr := NewPlayerPersister(playerDB).Persist(playerCtx, batch)
	observePersistPhase("player_write", playerWriteStart, playerErr == nil)
	if playerErr != nil {
		slog.ErrorContext(ctx, "CombinedPersister: player persist échoué",
			"batch_id", batch.BatchID, "player", batch.Player, "err", playerErr)
		return fmt.Errorf("CombinedPersister: player: %w", playerErr)
	}

	return nil
}
