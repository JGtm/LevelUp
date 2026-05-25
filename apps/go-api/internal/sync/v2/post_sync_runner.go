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

// postSyncRunnerV2 implémente PostSyncRunner via le wrapper V1.
type postSyncRunnerV2 struct {
	engineFactory SyncEngineFactory
	openPlayerDB  PlayerDBOpener
	sharedDB      *sql.DB
	clientFactory HaloClientFactory
}

// NewPostSyncRunner construit un PostSyncRunner V2.
//
// Paramètres :
//   - engineFactory : construit un SyncEngine V1 configuré pour un joueur.
//   - openPlayerDB  : ouvre la stats.duckdb du joueur en read-write (heals écrivent dedans).
//   - sharedDB      : connexion shared partagée (read-write — les heals écrivent aussi).
//   - clientFactory : construit un HaloClient pinné sur le joueur (pour les heals API).
//
// insertedIDs est passé per-call par l'orchestrator (cf. cycle.go Phase 6).
func NewPostSyncRunner(
	engineFactory SyncEngineFactory,
	openPlayerDB PlayerDBOpener,
	sharedDB *sql.DB,
	clientFactory HaloClientFactory,
) *postSyncRunnerV2 {
	return &postSyncRunnerV2{
		engineFactory: engineFactory,
		openPlayerDB:  openPlayerDB,
		sharedDB:      sharedDB,
		clientFactory: clientFactory,
	}
}

// RunPostSync délègue à V1.RunPostSyncForV2 après avoir ouvert les DB
// et construit le client + engine.
func (r *postSyncRunnerV2) RunPostSync(ctx context.Context, p PlayerProfile, insertedIDs []string) (PlayerPostSyncResult, error) {
	engine, err := r.engineFactory(ctx, p)
	if err != nil {
		return PlayerPostSyncResult{}, fmt.Errorf("engineFactory %s: %w", p.Gamertag, err)
	}

	playerDB, release, err := r.openPlayerDB(ctx, p.Gamertag)
	if err != nil {
		return PlayerPostSyncResult{}, fmt.Errorf("open player DB %s: %w", p.Gamertag, err)
	}
	defer release()

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

	v1Res := engine.RunPostSyncForV2(ctx, playerDB, r.sharedDB, client, insertedIDs)

	return mapV1PostSyncResult(p.PlayerSlug, v1Res), nil
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
	// AchievementsSynced, WeaponKillsProcessed, FatalErrors.
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
