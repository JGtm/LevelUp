// Package sync — engine_options.go : constructor + option setters du SyncEngine.
//
// Extrait de engine.go (refactor 2026-05-21). Regroupe NewSyncEngine et le
// pattern d'options (WithXxx + SetXxx) qui injectent les dépendances
// optionnelles : prestige hook, asset resolver, sharedprovider, friends loader,
// CSR season, custom client, local film cache. Comportement INCHANGÉ — pur
// déplacement.
//
// Voir engine.go (struct SyncEngine) pour le contexte.
package sync

import (
	"context"

	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/port"
)

// NewSyncEngine crée un moteur de sync pour un joueur sur le titre par défaut.
//
// Wrapper rétro-compatible (MT-11 / PMT-3) : délègue à NewSyncEngineForTitle
// avec titlePkg.DefaultSlug. Conservé pour les appelants legacy ; le nouveau code
// passe par NewSyncEngineForTitle avec le slug du profil.
//
//   - repoRoot    : racine du repo (cfg.RepoRoot)
//   - gamertag    : gamertag Halo du joueur
//   - xuid        : XUID numérique (sans "xuid()")
//   - tokens      : tokens Halo frais obtenus après Device Code Flow
//   - provider    : TokenProvider pour résoudre l'access_token Xbox Live
func NewSyncEngine(
	repoRoot, gamertag, xuid string,
	tokens *domain.HaloTokens,
	provider auth.TokenProvider,
) *SyncEngine {
	return NewSyncEngineForTitle(repoRoot, titlePkg.DefaultSlug, gamertag, xuid, tokens, provider)
}

// NewSyncEngineForTitle crée un moteur de sync pour un joueur sur un titre donné
// (MT-11 / PMT-3). TOUS les chemins DB (player/shared/metadata) sont résolus via
// PathResolver + titleSlug → le moteur écrit dans `data/titles/{titleSlug}/...`.
// Le champ `titleSlug` (déjà consommé par CSR/batch/scoring/asset-names/prestige/
// catalog) reçoit enfin sa vraie valeur au lieu de DefaultSlug systématique.
//
// titleSlug vide est traité comme DefaultSlug (garde défensive ; le caller
// scheduler applique déjà ce fallback via resolveTitleSlug).
func NewSyncEngineForTitle(
	repoRoot, titleSlug, gamertag, xuid string,
	tokens *domain.HaloTokens,
	provider auth.TokenProvider,
) *SyncEngine {
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	pr := titlePkg.NewPathResolver(repoRoot)
	return &SyncEngine{
		gamertag:       gamertag,
		xuid:           xuid,
		repoRoot:       repoRoot,
		titleSlug:      titleSlug,
		playerDBPath:   pr.PlayerDBPath(titleSlug, gamertag),
		sharedDBPath:   pr.SharedDBPath(titleSlug),
		metadataDBPath: pr.MetadataDBPath(titleSlug),
		pveDBPath:      pr.SharedPVEDBPath(titleSlug),
		syncCacheDir:   pr.SyncCacheDir(),
		tokens:         tokens,
		provider:       provider,
	}
}

// WithPrestigeHook attache un hook post-sync (best-effort).
//
// Le hook reçoit (ctx, gamertag, titleSlug) après que match_participants
// soit écrit. Il est responsable de gérer le feature flag et de ne jamais
// propager d'erreur (le sync ne doit pas échouer à cause de Prestige).
func (e *SyncEngine) WithPrestigeHook(hook func(ctx context.Context, gamertag, titleSlug string)) *SyncEngine {
	e.prestigeHook = hook
	return e
}

// WithResolver attache un Resolver pour le pré-warming des images d'achievements.
// Retourne le même engine pour permettre le chaînage.
func (e *SyncEngine) WithResolver(r assets.Resolver) *SyncEngine {
	e.resolver = r
	return e
}

// WithSharedProvider (commit 8i) attache un sharedprovider.Provider que le
// sync engine utilise pour ses ouvertures RW de shared (via AcquireWriter).
// En mode B-swap, c'est le chemin qui coordonne avec le pool joueur via
// les notifs Subscribe (PreSwapToRW / RWToRO).
//
// Si nil (mode legacy), le sync engine fait directement OpenSharedDB —
// comportement pre-sprint, avec risque de conflit "different configuration".
func (e *SyncEngine) WithSharedProvider(p sharedprovider.Provider) *SyncEngine {
	e.sharedProvider = p
	return e
}

// WithFriendsLoader attache un loader settings.FriendGamertags pour le hook
// auto-recompute is_with_friends post-sync delta. Sans ce hook, les nouveaux
// matchs sync restent is_with_friends=FALSE jusqu'au prochain recompute
// manuel (PATCH /settings ou CLI levelup recompute-friends).
//
// Le hook est idempotent (garde WHERE FALSE dans friends_recompute.go) et
// court-circuite si la liste est vide. Aucune erreur ne propage : un échec
// n'arrête pas le sync (best-effort).
func (e *SyncEngine) WithFriendsLoader(loader FriendsLoader) *SyncEngine {
	e.friendsLoader = loader
	return e
}

// WithCSRSeasonID configure l'identifiant de saison CSR (ex: "CsrSeason8").
// Requis pour que runCSRSnapshotSync appelle l'API — skip silencieux si absent.
func (e *SyncEngine) WithCSRSeasonID(id string) *SyncEngine {
	e.csrSeasonID = id
	return e
}

// WithBatchQueue branche une BatchQueue partagée serveur-wide. Quand non-nil,
// submitMatchAsBatch passe par queue.Submit (WAL + async worker) au lieu
// d'appeler les Persisters directement. À la fin du cycle, run() appelle
// queue.Drain pour attendre la fin des persists.
//
// Activé par cmd/server/main.go quand LEVELUP_PERSIST_BATCH_ASYNC != "0".
// Sans, l'engine reste synchrone (Persisters directs, Phase 2.3).
func (e *SyncEngine) WithBatchQueue(q *persist.BatchQueue) *SyncEngine {
	e.batchQueue = q
	return e
}

// WithPostSyncRunner branche le runner post-sync (Phase 4 plan stabilisation
// 2026-05-22). Le runner est invoqué dans runPostSyncPipeline avant + après
// la sync — il capture un snapshot before, attend la fin de la sync, puis
// émet les notifications delta + lance le pipeline progression V2.
//
// slug : identifiant URL du joueur (PlayerSlug), passé au runner.BeforeSync.
// Sert à la résolution ServiceRegistry.resolve(slug) côté runner.
//
// Nil runner → feature off, legacy behavior (sync sans hook).
// Avant ce sprint : seul SyncHandler HTTP avait le hook → auto-sync et CLI
// sautaient TOUT le post-sync delta + progression (notifications muettes,
// page Ascension vide).
func (e *SyncEngine) WithPostSyncRunner(runner port.PostSyncRunner, slug string) *SyncEngine {
	e.postSyncRunner = runner
	e.postSyncSlug = slug
	return e
}

// SetCustomClient injecte un client HaloClient personnalisé (ex: PooledHaloClient).
// Si défini, ce client sera utilisé à la place de NewHaloAPIClient.
func (e *SyncEngine) SetCustomClient(client HaloClient) {
	e.customClient = client
}

// WithMediaScanHook attache un hook d'indexation médias post-sync (best-effort).
// Le hook est appelé à la fin de runPostSyncPipeline : il scanne le répertoire
// captures/ du joueur, détecte les nouveaux fichiers et les associe aux matchs
// fraîchement insérés en DB. Nil → feature off.
// À construire via service.BuildMediaScanHook (injecté depuis scheduler + SyncHandler).
func (e *SyncEngine) WithMediaScanHook(hook func(ctx context.Context)) *SyncEngine {
	e.mediaHook = hook
	return e
}

// SetLocalFilmCache injecte un cache disque film hérité du projet Python.
// Le cache sera consulté avant l'API pour les manifestes et chunks
// REPLICATION_DATA. Sans effet si nil ou si le repertoire est introuvable.
func (e *SyncEngine) SetLocalFilmCache(cache *LocalFilmCache) {
	e.localFilmCache = cache
}
