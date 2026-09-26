// Package v2 — post_sync_runner.go : adapter PostSyncRunner V2 qui délègue
// à V1.RunPostSyncForV2 (cf. engine_v2bridge.go).
//
// Stratégie de duplication ciblée (cf. discussion D6.4) : au lieu de
// dupliquer les 14 sous-étapes du post-sync (~600-1000 lignes), V2
// appelle un wrapper exporté de V1 qui invoque la même logique
// runPostSyncPipeline interne. Bénéfice : parité parfaite V1↔V2 sur le
// post-sync, bug fix dans V1 → V2 en bénéficie automatiquement.
//
// L'adapter construit un SyncEngine V1 éphémère par joueur (via le
// EngineFactory injecté) et appelle son RunPostSyncForV2. Le SyncEngine
// doit être configuré avec WithCSRSeasonID, WithFriendsLoader, WithResolver,
// WithMediaScanHook — fait par EngineFactory en runtime.
package v2

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	syncpkg "levelup/go-api/internal/sync"
)

// SyncEngineFactory construit un *sync.SyncEngine configuré pour un joueur
// donné. Le runtime (main.go D6.5) wrappe NewSyncEngine + tous les WithX
// utilisés en prod. Les tests injectent une factory qui retourne un
// engine minimaliste (gamertag + xuid seulement).
type SyncEngineFactory func(ctx context.Context, p PlayerProfile) (*syncpkg.SyncEngine, error)

// SharedDBAcquirer acquiert un *sql.DB shared en READ-WRITE pour la durée du
// post-sync (heals events/skill/weapon/registry écrivent dans shared). En mode
// B-swap, route via Provider.AcquireWriter ; en legacy, OpenSharedDB. Le release
// ferme/rend le writer (RW→RO) et DOIT être appelé après le post-sync.
//
// Distinct du `getSharedDB func() *sql.DB` (RO) utilisé par KnownLoader pour des
// lectures : ce dernier renvoie le handle RO caché et n'a pas de release. Les
// confondre était la cause racine RC-A (heal recevait un handle RO → toutes les
// écritures combat échouaient "attached in read-only mode", silencieusement avant
// le fail-fast Phase 1a). Même classe de bug que le fix player DB RO→RW du
// 2026-05-25 (cf. sync_v2_wiring.go), mais côté shared, jamais corrigé jusqu'ici.
type SharedDBAcquirer func(ctx context.Context) (db *sql.DB, release func(), err error)

// postSyncRunnerV2 implémente PostSyncRunner via le wrapper V1.
type postSyncRunnerV2 struct {
	engineFactory  SyncEngineFactory
	openPlayerDB   PlayerDBOpener
	acquireSharedW SharedDBAcquirer // RW writer pour les heals (cf. SharedDBAcquirer)
	clientFactory  HaloClientFactory
	// readSharedRO (optionnel, étape 1 contention) : lecteur RO du shared
	// (provider.Get + release) pour les segments Read du pipeline en mode bursts.
	// nil → SharedAccess retombe sur des bursts Write pour les lectures (legacy).
	readSharedRO SharedDBAcquirer
}

// WithSharedReader câble le lecteur RO du shared (mode bursts, étape 1
// contention). Optionnel — sans lui, les lectures passent par des bursts Write.
func (r *postSyncRunnerV2) WithSharedReader(read SharedDBAcquirer) *postSyncRunnerV2 {
	r.readSharedRO = read
	return r
}

// NewPostSyncRunner construit un PostSyncRunner V2.
//
// Paramètres :
//   - engineFactory  : construit un SyncEngine V1 configuré pour un joueur.
//   - openPlayerDB   : ouvre stats.duckdb du joueur en read-write (heals écrivent dedans).
//   - acquireSharedW : acquiert le shared en READ-WRITE (writer provider + release).
//     Les heals post-sync écrivent dans shared ; un handle RO ferait échouer toutes
//     leurs écritures (RC-A). Le release est appelé en fin de post-sync.
//   - clientFactory  : construit un HaloClient pinné sur le joueur (pour les heals API).
//
// insertedIDs est passé per-call par l'orchestrator (cf. cycle.go Phase 6).
func NewPostSyncRunner(
	engineFactory SyncEngineFactory,
	openPlayerDB PlayerDBOpener,
	acquireSharedW SharedDBAcquirer,
	clientFactory HaloClientFactory,
) *postSyncRunnerV2 {
	return &postSyncRunnerV2{
		engineFactory:  engineFactory,
		openPlayerDB:   openPlayerDB,
		acquireSharedW: acquireSharedW,
		clientFactory:  clientFactory,
	}
}

// RunPostSync délègue à V1.RunPostSyncForV2 après avoir ouvert les DB
// et construit le client + engine.
//
// INVARIANT (Phase 6, ADR 0016 B-swap) : en mode bursts (défaut), aucun
// write-lease shared n'est tenu pour toute la durée de la phase 6 — chaque
// écriture shared ouvre une fenêtre RW courte puis la relâche (mesure étape 0 :
// ~13s/joueur tenues pour rien). Le fallback LEVELUP_POSTSYNC_BURST=0 acquiert
// encore le writer upfront.
func (r *postSyncRunnerV2) RunPostSync(ctx context.Context, p PlayerProfile, insertedIDs []string) (PlayerPostSyncResult, error) {
	engine, err := r.engineFactory(ctx, p)
	if err != nil {
		return PlayerPostSyncResult{}, fmt.Errorf("engineFactory %s: %w", p.Gamertag, err)
	}

	playerDB, releasePlayer, err := r.openPlayerDB(ctx, p.Gamertag)
	if err != nil {
		return PlayerPostSyncResult{}, fmt.Errorf("open player DB %s: %w", p.Gamertag, err)
	}
	defer releasePlayer()

	var client syncpkg.HaloClient
	if r.clientFactory != nil {
		if c := r.clientFactory(p.Gamertag, p.XUID); c != nil {
			// c est notre HaloClient narrow (v2) — il satisfait aussi sync.HaloClient
			// si le wrapper runtime renvoie un *sync.PooledHaloClient (qui implémente
			// les deux). On le passe tel quel.
			if cs, ok := c.(syncpkg.HaloClient); ok {
				client = cs
			}
		}
	}

	// Étape 1 contention (défaut) : mode BURSTS PARESSEUX — le runner n'acquiert
	// PLUS le writer upfront ; le pipeline ouvre des segments Read RO et des
	// bursts Write courts par écriture shared (fenêtre RW ~0 en stationnaire,
	// mesure étape 0 : 13s/joueur tenues pour rien). L'échec d'acquisition est
	// géré PAR ÉTAPE dans le pipeline (dégradation, plus de skip global).
	if syncpkg.PostSyncBurstEnabled() {
		shared := syncpkg.NewBurstSharedAccess(
			func(c context.Context) (*sql.DB, func(), error) { return r.acquireSharedW(c) },
			readFnOrNil(r.readSharedRO),
			"sync_v2_postsync")
		v1Res := engine.RunPostSyncForV2Shared(ctx, playerDB, shared, client, insertedIDs)
		return mapV1PostSyncResult(p.PlayerSlug, v1Res), nil
	}

	// ROLLBACK (LEVELUP_POSTSYNC_BURST=0) : chemin pinned historique — RC-A fix :
	// acquérir le shared en READ-WRITE (writer provider), pas le handle RO caché.
	sharedDB, releaseShared, err := r.acquireSharedW(ctx)
	if err != nil {
		slog.WarnContext(ctx, "sync.v2 post-sync: acquisition shared RW échouée — skip",
			"event", "sync.v2.postsync.shared_acquire_failed", "player", p.PlayerSlug, "err", err)
		return PlayerPostSyncResult{PlayerSlug: p.PlayerSlug}, nil
	}
	if sharedDB == nil {
		if releaseShared != nil {
			releaseShared()
		}
		slog.WarnContext(ctx, "sync.v2 post-sync: shared DB indisponible — skip",
			"event", "sync.v2.postsync.shared_unavailable", "player", p.PlayerSlug)
		return PlayerPostSyncResult{PlayerSlug: p.PlayerSlug}, nil
	}
	defer releaseShared()

	v1Res := engine.RunPostSyncForV2(ctx, playerDB, sharedDB, client, insertedIDs)

	return mapV1PostSyncResult(p.PlayerSlug, v1Res), nil
}

// readFnOrNil convertit le SharedDBAcquirer RO optionnel en closure pour
// NewBurstSharedAccess (nil reste nil → fallback legacy dans SharedAccess).
func readFnOrNil(read SharedDBAcquirer) func(context.Context) (*sql.DB, func(), error) {
	if read == nil {
		return nil
	}
	return func(c context.Context) (*sql.DB, func(), error) { return read(c) }
}

// mapV1PostSyncResult transforme un domain.PostSyncResult (V1) en
// PlayerPostSyncResult (V2). Les champs disponibles côté V1 sont mappés
// 1:1 ; les autres restent à zéro.
func mapV1PostSyncResult(slug string, v1 any) PlayerPostSyncResult {
	out := PlayerPostSyncResult{PlayerSlug: slug}
	// V1's domain.PostSyncResult a des champs CitationsComputed, etc.
	// On utilise la réflexion légère via interface ; pour rester portable
	// on tape directement quand l'import est dispo.
	// Cf. internal/domain/sync.go.
	// Champs mappés : CitationsComputed, DominanceFlagsComputed,
	// AchievementsSynced, FatalErrors.
	type v1Result interface {
		// Phantom marker : on consomme un alias structuré dans le main.go
		// wiring qui adapte le type concret.
	}
	_ = v1Result(nil)
	// Le mapping concret se fait dans le caller (post_sync_runner_mapping.go)
	// pour garder ce fichier découplé de domain.PostSyncResult.
	if mapper := postSyncResultMapper; mapper != nil {
		return mapper(slug, v1)
	}
	slog.Debug("mapV1PostSyncResult: postSyncResultMapper nil — mapping skip", "slug", slug)
	return out
}

// postSyncResultMapper est positionné par post_sync_runner_mapping.go pour
// éviter une dépendance directe à internal/domain dans ce fichier (lever
// les imports cycliques potentiels en cas de futur refactor).
var postSyncResultMapper func(slug string, v1 any) PlayerPostSyncResult
