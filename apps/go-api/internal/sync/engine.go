// Package sync — engine.go : moteur de synchronisation delta/full.
//
// Portage du DuckDBSyncEngine Python (engine.py + mixins).
// Le moteur est instancié une fois par requête de sync et n'est pas réutilisable.
//
// Flux RunDelta :
//  1. Acquérir les write leases (player + shared)
//  2. Ouvrir les deux DBs en lecture/écriture
//  3. Charger les match_ids déjà connus (player_match_enrichment)
//  4. Paginer l'historique API jusqu'à maxMatches nouveaux ou fin
//  5. Pour chaque match nouveau : fetch stats → transform → insert shared + player
//  6. Mettre à jour sync_meta
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/dblease"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"

	"golang.org/x/sync/errgroup"
)

const (
	// historyPageSize est le nombre de matchs demandés par page API.
	historyPageSize = 25
)

// SyncEngine orchestre la synchronisation des données Halo d'un joueur.
// FriendsLoader retourne la liste courante des amis configurés (typiquement
// settings.FriendGamertags). Nil → feature désactivée (legacy / pas de wiring).
type FriendsLoader func() ([]string, error)

type SyncEngine struct {
	gamertag       string
	xuid           string
	titleSlug      string
	playerDBPath   string
	sharedDBPath   string
	globalDBPath   string // P5.3 : data/global/xbox_aliases.duckdb (mapping xuid→gamertag global)
	metadataDBPath string
	tokens         *domain.HaloTokens
	// provider est utilisé pour résoudre l'access_token Xbox Live (achievements).
	// Nil si non défini (les achievements seront ignorés).
	provider auth.TokenProvider
	// resolver est utilisé pour le pré-warming des images d'achievements (optionnel).
	resolver assets.Resolver
	// customClient est optionnel — si non-nil, utilisé à la place de NewHaloAPIClient.
	// Permet l'injection de PooledHaloClient ou autres implémentations HaloClient.
	customClient HaloClient
	// localFilmCache (optionnel) court-circuite l'API film en lisant le cache
	// disque hérité du projet Python. Utile pour récupérer manifestes +
	// chunks REPLICATION_DATA déjà téléchargés (~942 matchs en cache).
	localFilmCache *LocalFilmCache
	// prestigeHook est appelé après ingestion (best-effort, no-op si nil).
	// Reçoit (ctx, gamertag, titleSlug) — le hook se charge lui-même de
	// la résolution Prestige et du feature flag.
	prestigeHook func(ctx context.Context, gamertag, titleSlug string)
	// friendsLoader résout settings.FriendGamertags à la demande pour le
	// hook auto-recompute is_with_friends post-sync delta. Nil → feature off
	// (les nouveaux matchs resteront is_with_friends=FALSE jusqu'au prochain
	// recompute manuel via PATCH /settings ou CLI levelup recompute-friends).
	friendsLoader FriendsLoader
	// metaDB (optionnel) — connexion ouverte par run() au démarrage de la sync
	// pour permettre l'enrichissement post-Extract des MatchRegistryRow via
	// asset_translations (cf. EnrichRegistryFromMetadata, anti-régression UUIDs
	// bruts dans match_registry.playlist_name). Nil dans les tests unitaires
	// qui appellent processMatch directement → l'enrichissement devient no-op
	// et la sync reste fonctionnelle (UUID préservé comme avant).
	metaDB *sql.DB
	// csrSeasonID est l'identifiant de saison CSR courant (ex: "CsrSeason8").
	// Vide → runCSRSnapshotSync est skippé silencieusement.
	csrSeasonID string

	// sharedProvider (commit 8i) — si non-nil, le sync engine route ses
	// ouvertures RW de shared via Provider.AcquireWriter (mode B-swap).
	// Sinon, fallback OpenSharedDB direct (mode legacy, comportement
	// pre-sprint, conflit "different configuration" possible).
	//
	// Injecté via WithSharedProvider depuis main.go / scheduler en mode
	// flag-on. Cohérent avec PlayerPoolConfig.SharedReader côté pool.
	sharedProvider sharedprovider.Provider
}

// NewSyncEngine crée un moteur de sync pour un joueur.
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
	pr := titlePkg.NewPathResolver(repoRoot)
	return &SyncEngine{
		gamertag:       gamertag,
		xuid:           xuid,
		titleSlug:      titlePkg.DefaultSlug,
		playerDBPath:   pr.PlayerDBPath(titlePkg.DefaultSlug, gamertag),
		sharedDBPath:   pr.SharedDBPath(titlePkg.DefaultSlug),
		globalDBPath:   pr.GlobalXuidAliasesDBPath(),
		metadataDBPath: pr.MetadataDBPath(titlePkg.DefaultSlug),
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

// AcquirePlayerWriterStandalone est la variante package-level pour stats.duckdb
// d'un joueur. Prend le lease applicatif (Kind=Player) + ouvre via le pool.
// Retourne (*duckdbpkg.DB, releaseFunc, error) pour les callers qui consomment
// l'API ref-comptée (queries_auth.WriteOAuthRefreshToken et co).
//
// Sprint chore/ci-stabilization 2026-05-20 : respecte ADR 0013 (pas d'OpenReadWrite
// direct depuis service/handlers).
func AcquirePlayerWriterStandalone(ctx context.Context, playerDBPath string) (*duckdbpkg.DB, func(), error) {
	lease, err := dblease.AcquireWriterCtx(ctx, nil, playerDBPath, dblease.KindPlayer)
	if err != nil {
		return nil, nil, fmt.Errorf("AcquirePlayerWriterStandalone lease: %w", err)
	}
	handle, err := duckdbpkg.OpenReadWriteShared(playerDBPath)
	if err != nil {
		lease.Release()
		return nil, nil, fmt.Errorf("AcquirePlayerWriterStandalone open: %w", err)
	}
	return handle, func() {
		_ = handle.Close()
		lease.Release()
	}, nil
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

// RunDelta synchronise uniquement les matchs nouveaux depuis la dernière sync.
// S'arrête dès qu'un match connu est rencontré dans l'historique paginé.
// SetCustomClient injecte un client HaloClient personnalisé (ex: PooledHaloClient).
// Si défini, ce client sera utilisé à la place de NewHaloAPIClient.
func (e *SyncEngine) SetCustomClient(client HaloClient) {
	e.customClient = client
}

// SetLocalFilmCache injecte un cache disque film hérité du projet Python.
// Le cache sera consulté avant l'API pour les manifestes et chunks
// REPLICATION_DATA. Sans effet si nil ou si le repertoire est introuvable.
func (e *SyncEngine) SetLocalFilmCache(cache *LocalFilmCache) {
	e.localFilmCache = cache
}

func (e *SyncEngine) RunDelta(ctx context.Context, opts domain.SyncOptions) (domain.SyncResult, error) {
	return e.run(ctx, opts, true)
}

// RunFull synchronise tous les matchs jusqu'à opts.MaxMatches (peu importe l'historique connu).
func (e *SyncEngine) RunFull(ctx context.Context, opts domain.SyncOptions) (domain.SyncResult, error) {
	return e.run(ctx, opts, false)
}

// RunBackfill détecte les matchs avec données manquantes et retourne la liste.
// Le scope doit être Resolve() avant appel. Retourne la liste des match_ids manquants.
func (e *SyncEngine) RunBackfill(ctx context.Context, scope *SyncScope) ([]string, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return nil, fmt.Errorf("RunBackfill lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return nil, fmt.Errorf("RunBackfill OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// Sprint B1 commit 11b : acquireSharedWriter centralise lease + open
	// (Provider en B-swap, dblease+OpenSharedDB en legacy).
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctx)
	if err != nil {
		return nil, fmt.Errorf("RunBackfill: %w", err)
	}
	defer releaseShared()

	return FindMatchesMissingData(playerHandle.SQLDb(), sharedDB, e.xuid, scope)
}

// RunBackfillComebackBadges calcule et persiste le dominance_flag pour les
// matchs du joueur. Selectionne :
//   - tous les matchs si forceAll=true
//   - les matchs sans dominance_flag (ou flag=0) sinon
//
// Branche la fonction BackfillDominanceFlags (sync/comeback.go) au pipeline.
// Retourne le nombre de match_ids traites (et l'erreur infra si lease/open
// echoue).
//
// L'ingestion principale (RunDelta/RunFull) ne calcule PAS encore le flag a
// chaque match : ce backfill explicite est la voie d'entree pour peupler les
// dominance_flag (cf. PLAN_META_FOUNDATIONS_GO § 6.0.1, prerequis Phase 1
// pilote Squad/MatchView/Career).
// RunBackfillEngagementScores calcule et persiste le score d'engagement pour
// les matchs PvP du joueur (Phase 6 plan engagement). Si force=true, recalcule
// les scores existants ; sinon ne calcule que les manquants.
//
// Skip silencieux si la migration Phase 2 n'a pas ete appliquee (gating
// information_schema). Aucun appel API requis (calcul purement local depuis
// highlight_events deja synces).
func (e *SyncEngine) RunBackfillEngagementScores(ctx context.Context, force bool) (int, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillEngagementScores lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillEngagementScores OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// Sprint B1 commit 11b : acquireSharedWriter centralise lease + open.
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctx)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillEngagementScores: %w", err)
	}
	defer releaseShared()

	n, err := batchComputeEngagementScores(ctx, playerHandle.SQLDb(), sharedDB, e.xuid, force)
	if err != nil {
		return n, err
	}
	// Recompute des coefficients en queue : on a possiblement ajoute des
	// paces en DB, donc la mediane est a rafraichir.
	if nCoefs, errCoefs := batchRecomputeCoefficients(ctx, playerHandle.SQLDb(), e.xuid); errCoefs != nil {
		slog.WarnContext(ctx, "RunBackfillEngagementScores: recompute coefs failed",
			"xuid", e.xuid, "err", errCoefs)
	} else if nCoefs > 0 {
		slog.InfoContext(ctx, "RunBackfillEngagementScores: coefs updated",
			"xuid", e.xuid, "n_modes", nCoefs)
	}
	return n, nil
}

// RunBackfillEngagementCoefficients recompute UNIQUEMENT les coefficients
// d'engagement du joueur depuis les paces deja persistees (~5ms par joueur,
// 0 re-scan des matchs). A activer via SyncScope.EngagementCoefficients.
//
// Utile pour rafraichir apres un ajustement de formule sans devoir relancer
// le compute des scores. Skip silencieux si la migration des paces n'est
// pas appliquee (cf. batchRecomputeCoefficients).
//
// Retourne le nombre de modes_category mis a jour (0 a 2).
func (e *SyncEngine) RunBackfillEngagementCoefficients(ctx context.Context) (int, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillEngagementCoefficients lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillEngagementCoefficients OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	return batchRecomputeCoefficients(ctx, playerHandle.SQLDb(), e.xuid)
}

// RunBackfillLUSR recalcule le LUSR TrueSkill 2 pour tous les matchs du joueur.
// force=true : recalcule depuis zéro même si les matchs ont déjà un rating.
// Les poids des médailles (medal_exploit) sont chargés depuis la metadata DB (best-effort).
func (e *SyncEngine) RunBackfillLUSR(ctx context.Context, force bool) (int, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillLUSR lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillLUSR OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// Sprint B1 commit 11b : acquireSharedWriter centralise lease + open.
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctx)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillLUSR: %w", err)
	}
	defer releaseShared()

	medalMap := e.loadMedalExploitMapBestEffort(ctx, sharedDB)
	return batchComputeLUSR(playerHandle.SQLDb(), sharedDB, e.xuid, medalMap, force)
}

// RunBackfillCSR ré-importe les CSR par-match depuis l'API Halo skill pour
// tous les matchs classés du joueur qui n'ont pas encore de row CSR (ou tous
// si force=true). Cible : les matchs synchronisés AVANT la Phase B (sync
// nominal qui écrit déjà les CSR inline) et les cas où GetMatchSkill n'a pas
// retourné de RankRecap au moment du sync initial.
//
// Retourne le résumé d'exécution (matchs traités, restaurés, skippés, etc.).
func (e *SyncEngine) RunBackfillCSR(ctx context.Context, force bool) (CSRBackfillResult, error) {
	var empty CSRBackfillResult
	if e.tokens == nil || e.tokens.SpartanToken == "" {
		return empty, fmt.Errorf("RunBackfillCSR: tokens Halo absents (re-login requis)")
	}

	slog.InfoContext(ctx, "RunBackfillCSR: démarrage",
		"gamertag", e.gamertag, "xuid", e.xuid, "force", force)

	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return empty, fmt.Errorf("RunBackfillCSR lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return empty, fmt.Errorf("RunBackfillCSR OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// shared DB en read-only suffit : on ne fait que SELECT match_registry.
	// Sprint B1 commit 11b : passe par acquireSharedWriter pour cohérence
	// (Provider en B-swap, dblease+OpenSharedDB en legacy).
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctx)
	if err != nil {
		return empty, fmt.Errorf("RunBackfillCSR: %w", err)
	}
	defer releaseShared()

	var client HaloClient
	if e.customClient != nil {
		client = e.customClient
		slog.DebugContext(ctx, "RunBackfillCSR: utilisation client personnalisé (pool)")
	} else {
		client = NewHaloAPIClient(e.tokens.SpartanToken, e.tokens.ClearanceToken, 5)
		slog.DebugContext(ctx, "RunBackfillCSR: client Halo standard, 5 RPS")
	}

	res, err := BackfillCSRFromAPI(ctx, client, playerHandle.SQLDb(), sharedDB, e.xuid, force)
	if err != nil {
		slog.ErrorContext(ctx, "RunBackfillCSR: échec",
			"gamertag", e.gamertag, "err", err)
		return res, err
	}
	slog.InfoContext(ctx, "RunBackfillCSR: terminé",
		"gamertag", e.gamertag,
		"inserted", res.Inserted,
		"already_csr", res.AlreadyHadCSR,
		"skipped_no_recap", res.SkippedNoRankRecap,
		"skill_errors", res.SkillErrors,
	)
	return res, nil
}

// RunBackfillPerf recalcule le performance score relatif pour tous les matchs du joueur.
// force=true : recalcule même si les matchs ont déjà un score (utile après changement de formule).
func (e *SyncEngine) RunBackfillPerf(ctx context.Context, force bool) (int, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillPerf lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillPerf OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// Sprint B1 commit 11b : acquireSharedWriter centralise lease + open.
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctx)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillPerf: %w", err)
	}
	defer releaseShared()

	medalMap := e.loadMedalExploitMapBestEffort(ctx, sharedDB)
	return batchComputePerformanceScores(playerHandle.SQLDb(), sharedDB, e.xuid, medalMap, force)
}

// loadMedalExploitMapBestEffort charge les scores d'exploit médailles depuis la metadata DB.
// Retourne nil en cas d'erreur (le LUSR/Perf fonctionne sans données médailles).
func (e *SyncEngine) loadMedalExploitMapBestEffort(ctx context.Context, sharedDB *sql.DB) map[string]float64 {
	return loadMedalExploitMap(ctx, e.metadataDBPath, sharedDB, e.xuid)
}

// loadMedalExploitMap : variante package-level réutilisable hors SyncEngine
// (ex: MatchRecomputer). Best-effort : retourne nil si la metadata DB est
// indisponible ou si le calcul échoue — perf/LUSR fonctionnent sans.
func loadMedalExploitMap(ctx context.Context, metadataDBPath string, sharedDB *sql.DB, xuid string) map[string]float64 {
	if metadataDBPath == "" {
		return nil
	}
	metaDB, err := sql.Open("duckdb", metadataDBPath)
	if err != nil {
		slog.DebugContext(ctx, "loadMedalExploitMap: ouverture metaDB échouée", "err", err)
		return nil
	}
	defer metaDB.Close() //nolint:errcheck
	metaDB.SetMaxOpenConns(1)

	diffMap, err := LoadMedalDifficultyFromMeta(metaDB)
	if err != nil || len(diffMap) == 0 {
		slog.DebugContext(ctx, "loadMedalExploitMap: difficulty map vide", "err", err)
		return nil
	}
	result, err := ComputeMedalExploitByMatch(sharedDB, diffMap, xuid)
	if err != nil {
		slog.DebugContext(ctx, "loadMedalExploitMap: compute échoué", "err", err)
		return nil
	}
	return result
}

func (e *SyncEngine) RunBackfillComebackBadges(ctx context.Context, forceAll bool) (int, error) {
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	// Sprint B1 commit 11b : acquireSharedWriter centralise lease + open.
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctx)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges: %w", err)
	}
	defer releaseShared()

	matchIDs, err := selectMatchesForComebackBadges(ctx, playerHandle.SQLDb(), sharedDB, e.xuid, forceAll)
	if err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges select: %w", err)
	}
	if len(matchIDs) == 0 {
		slog.InfoContext(ctx, "comeback-badges: aucun match a traiter",
			"player", e.gamertag, "force_all", forceAll)
		return 0, nil
	}

	slog.InfoContext(ctx, "comeback-badges: backfill en cours",
		"player", e.gamertag, "match_count", len(matchIDs), "force_all", forceAll)
	if err := BackfillDominanceFlags(ctx, sharedDB, playerHandle.SQLDb(), e.xuid, matchIDs); err != nil {
		return 0, fmt.Errorf("RunBackfillComebackBadges backfill: %w", err)
	}
	return len(matchIDs), nil
}

// selectMatchesForComebackBadges retourne les match_ids du joueur a traiter
// pour le backfill dominance_flag.
//
// Si forceAll=true : tous les matchs du joueur dans shared.match_participants.
// Sinon : uniquement les matchs ou player_match_enrichment.dominance_flag est
// nul ou egal a 0 (cas par defaut "manquant").
func selectMatchesForComebackBadges(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	xuid string,
	forceAll bool,
) ([]string, error) {
	allIDs, err := loadAllMatchIDsForPlayer(ctx, sharedDB, xuid)
	if err != nil {
		return nil, fmt.Errorf("load all match_ids: %w", err)
	}
	if forceAll {
		return allIDs, nil
	}
	flagged, err := loadFlaggedMatchIDs(ctx, playerDB)
	if err != nil {
		return nil, fmt.Errorf("load flagged match_ids: %w", err)
	}
	flaggedSet := make(map[string]struct{}, len(flagged))
	for _, id := range flagged {
		flaggedSet[id] = struct{}{}
	}
	out := make([]string, 0, len(allIDs))
	for _, id := range allIDs {
		if _, ok := flaggedSet[id]; !ok {
			out = append(out, id)
		}
	}
	return out, nil
}

// loadAllMatchIDsForPlayer retourne tous les match_id du joueur (shared DB).
func loadAllMatchIDsForPlayer(ctx context.Context, sharedDB *sql.DB, xuid string) ([]string, error) {
	rows, err := sharedDB.QueryContext(ctx,
		`SELECT match_id FROM match_participants WHERE xuid = ? ORDER BY match_id`, xuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// loadFlaggedMatchIDs retourne les match_id deja peuples avec un flag
// non-nul et non-zero (player DB).
func loadFlaggedMatchIDs(ctx context.Context, playerDB *sql.DB) ([]string, error) {
	rows, err := playerDB.QueryContext(ctx,
		`SELECT match_id FROM player_match_enrichment
		 WHERE dominance_flag IS NOT NULL AND dominance_flag > 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// run est le cœur du moteur de sync. isDelta=true → stop dès un match connu.
func (e *SyncEngine) run(ctx context.Context, opts domain.SyncOptions, isDelta bool) (domain.SyncResult, error) {
	result := domain.SyncResult{StartedAt: time.Now()}
	mode := "full"
	if isDelta {
		mode = "delta"
	}

	// Sprint B1 commit 16 : crée un event_id qui sera ajouté à TOUS les logs
	// émis depuis ce ctx (cf. ContextHandler) — permet de grep cross-module
	// dans logs/{sync,provider,pool,...}.log pour reconstituer le timeline
	// d'un sync donné.
	ctx, eventID := logging.WithEvent(ctx, "sync.Run"+strings.Title(mode))
	slog.InfoContext(ctx, "sync: démarrage",
		"gamertag", e.gamertag,
		"xuid", e.xuid,
		"mode", mode,
		"match_type", opts.MatchType,
		"max_matches", opts.MaxMatches,
		"with_participants", opts.WithParticipants,
		"with_medals", opts.WithMedals,
		"rps", opts.RequestsPerSecond,
		"event", eventID,
	)

	// B8 : validation fail-fast des options avant tout accès réseau ou DB.
	if err := opts.Validate(); err != nil {
		slog.ErrorContext(ctx, "sync: options invalides", "err", err, "gamertag", e.gamertag)
		return result, fmt.Errorf("run: options invalides: %w", err)
	}

	// ─── Write leases ──────────────────────────────────────────────────────────
	slog.DebugContext(ctx, "sync: acquisition lease player DB", "gamertag", e.gamertag, "db", e.playerDBPath)
	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		slog.ErrorContext(ctx, "sync: lease player DB échouée", "gamertag", e.gamertag, "err", err)
		return result, fmt.Errorf("run: %w", err)
	}
	defer writerPlayer.Release()

	// Sprint B1 commit 11b : le dblease shared est désormais pris par
	// acquireSharedWriter (Provider ou legacy). Ne PAS le prendre ici sinon
	// auto-deadlock (cf. provider.go:231 + sync.Mutex non-réentrant).

	// ─── Ouverture des DBs ─────────────────────────────────────────────────────
	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		slog.ErrorContext(ctx, "sync: ouverture player DB échouée", "gamertag", e.gamertag, "db", e.playerDBPath, "err", err)
		return result, fmt.Errorf("run OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()
	playerDB := playerHandle.SQLDb()

	// Commit 8i : route via Provider en mode B-swap (coordonne avec le pool
	// joueur via Subscribe). Fallback OpenSharedDB direct si Provider nil.
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "sync: ouverture shared DB échouée", "gamertag", e.gamertag, "db", e.sharedDBPath, "err", err)
		return result, fmt.Errorf("run acquireSharedWriter: %w", err)
	}
	defer releaseShared()

	// P5.3 : DB globale xbox_aliases (mapping xuid→gamertag global Microsoft).
	globalDB, globalCleanup, err := openGlobalDB(e.globalDBPath)
	if err != nil {
		slog.WarnContext(ctx, "sync: ouverture global DB échouée — alias upsert désactivé",
			"db", e.globalDBPath, "err", err)
		globalDB = nil
	} else {
		defer globalCleanup()
	}

	// metaDB best-effort : utilisé par EnrichRegistryFromMetadata pour résoudre
	// les UUIDs bruts en noms canoniques EN avant l'INSERT match_registry.
	// Échec d'ouverture → enrichissement désactivé pour ce run, sync continue.
	if e.metadataDBPath != "" {
		metaDB, metaErr := sql.Open("duckdb", e.metadataDBPath+"?access_mode=read_only")
		if metaErr != nil {
			slog.WarnContext(ctx, "sync: ouverture metadata DB échouée — enrich registry désactivé",
				"db", e.metadataDBPath, "err", metaErr)
		} else {
			metaDB.SetMaxOpenConns(1)
			e.metaDB = metaDB
			defer func() {
				_ = metaDB.Close()
				e.metaDB = nil
			}()
		}
	}
	slog.DebugContext(ctx, "sync: DBs ouvertes", "gamertag", e.gamertag)

	// ─── Match IDs déjà connus (player DB) ───────────────────────────────────
	known, err := loadKnownMatchIDs(playerDB)
	if err != nil {
		slog.ErrorContext(ctx, "sync: chargement match_ids connus échoué", "gamertag", e.gamertag, "err", err)
		return result, fmt.Errorf("run loadKnownMatchIDs: %w", err)
	}
	slog.InfoContext(ctx, "sync: match_ids connus chargés", "gamertag", e.gamertag, "known_count", len(known))

	// ─── Client API ────────────────────────────────────────────────────────────
	var client HaloClient
	if e.customClient != nil {
		client = e.customClient
		slog.DebugContext(ctx, "sync: utilisation client personnalisé (pool)")
	} else {
		api := NewHaloAPIClient(e.tokens.SpartanToken, e.tokens.ClearanceToken, opts.RequestsPerSecond)
		if e.localFilmCache != nil {
			api = api.WithLocalFilmCache(e.localFilmCache)
			slog.InfoContext(ctx, "sync: cache film local actif", "gamertag", e.gamertag)
		}
		client = api
		slog.DebugContext(ctx, "sync: utilisation HaloAPIClient standard")
	}

	// ─── Pagination de l'historique ────────────────────────────────────────────
	processed := 0
	start := 0

	for processed < opts.MaxMatches {
		// Respecter le contexte d'annulation.
		if err := ctx.Err(); err != nil {
			break
		}

		slog.DebugContext(ctx, "sync: requête historique API",
			"gamertag", e.gamertag, "xuid", e.xuid, "start", start, "page_size", historyPageSize,
		)
		// L'endpoint /hi/players/{player}/matches exige strictement le format
		// xuid(NNN) (voir Grunt StatsModule.GetMatchHistory + SPNKr). Passer le
		// gamertag directement renvoie une réponse stale figée — symptôme du
		// "no inserts since 6 mai" diagnostiqué le 2026-05-20.
		entries, err := client.GetMatchHistory(ctx, fmt.Sprintf("xuid(%s)", e.xuid), opts.MatchType, start, historyPageSize)
		if err != nil {
			slog.WarnContext(ctx, "sync: GetMatchHistory échoué",
				"gamertag", e.gamertag, "start", start, "err", err,
			)
			result.AddWarning(fmt.Sprintf("GetMatchHistory(start=%d): %v", start, err))
			break
		}
		if len(entries) == 0 {
			slog.DebugContext(ctx, "sync: fin historique (page vide)", "gamertag", e.gamertag, "start", start)
			break // fin de l'historique
		}
		slog.DebugContext(ctx, "sync: page reçue",
			"gamertag", e.gamertag, "entries", len(entries), "start", start,
		)
		// Log INFO du 1er match retourné par l'API (seulement sur start=0).
		// Sentinelle de fraîcheur : si ce StartTime ne bouge pas entre 2 cycles
		// alors que le joueur a joué, on sait que l'API renvoie du stale
		// (cf. incident 2026-05-20, endpoint /hi/players/{gamertag}/matches sans
		// xuid(...) renvoyait du contenu figé).
		if start == 0 && len(entries) > 0 {
			slog.InfoContext(ctx, "sync: 1er match retourné par API",
				"gamertag", e.gamertag, "xuid", e.xuid,
				"first_match_id", entries[0].MatchID,
				"first_match_start_time", entries[0].StartTime,
			)
		}

		allKnown := true

		// ─── Phase 1 : Filtrer et préparer les matchs à fetcher ───
		var toFetch []string // MatchIDs à fetcher
		var fetchIndex []int // Index dans entries (pour order preservation)

		for i, entry := range entries {
			if processed >= opts.MaxMatches {
				break
			}
			if known[entry.MatchID] {
				result.MatchesSkipped++
				if isDelta {
					slog.InfoContext(ctx, "sync: match connu rencontré — arrêt delta",
						"gamertag", e.gamertag, "match_id", entry.MatchID,
						"processed", processed, "skipped", result.MatchesSkipped,
					)
					goto done
				}
				continue
			}
			allKnown = false
			toFetch = append(toFetch, entry.MatchID)
			fetchIndex = append(fetchIndex, i)
		}

		if len(toFetch) > 0 {
			// ─── Phase 2 : Fetch parallèle ───
			fetchedMatches := make([]*fetchedMatch, len(toFetch))
			fetchErrors := make([]error, len(toFetch))
			var mu sync.Mutex

			eg, egCtx := errgroup.WithContext(ctx)
			// Pas de SetLimit ici — RPS limité par HaloAPIClient.rateWait()
			for i, matchID := range toFetch {
				i, matchID := i, matchID // Capturer pour closure
				eg.Go(func() error {
					fm, err := e.fetchMatchData(egCtx, client, matchID, opts)
					mu.Lock()
					fetchedMatches[i] = fm
					fetchErrors[i] = err
					mu.Unlock()
					if err != nil {
						slog.WarnContext(egCtx, "sync: fetchMatchData échoué",
							"gamertag", e.gamertag, "match_id", matchID, "err", err,
						)
						result.AddWarning(fmt.Sprintf("fetchMatchData(%s): %v", matchID, err))
					}
					return nil // Non-fatal : continuer même si fetch échoue
				})
			}
			_ = eg.Wait() // Attendre tous les fetches (même si certains échouent)

			// ─── Phase 3 : Insert séquentiel (order-preserving) ───
			for i, fm := range fetchedMatches {
				if fetchErrors[i] != nil {
					// Fetch échoué, skip insert
					continue
				}
				if fm == nil {
					continue
				}

				if err := e.insertFetchedMatch(ctx, sharedDB, playerDB, globalDB, &result, fm); err != nil {
					slog.WarnContext(ctx, "sync: insertFetchedMatch échoué",
						"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
					)
					result.AddWarning(fmt.Sprintf("insertFetchedMatch(%s): %v", fm.MatchID, err))
				} else {
					processed++
					slog.InfoContext(ctx, "sync: match traité (parallèle)",
						"gamertag", e.gamertag, "match_id", fm.MatchID,
						"processed", processed, "inserted_total", result.MatchesInserted,
					)
				}
			}
		}

		if isDelta && allKnown {
			break
		}
		start += len(entries)
	}

done:
	slog.InfoContext(ctx, "sync: boucle pagination terminée",
		"gamertag", e.gamertag, "mode", mode,
		"inserted", result.MatchesInserted, "skipped", result.MatchesSkipped,
		"warnings", len(result.Warnings),
	)

	// ─── Pipeline post-sync ─────────────────────────────────────────────────────
	postResult := e.runConditionalPostSync(ctx, playerDB, sharedDB, client, result.MatchesInserted, result.InsertedMatchIDs)
	if result.MatchesInserted > 0 || postResult.AchievementsSynced {
		result.PostSync = &postResult
		slog.InfoContext(ctx, "sync: pipeline post-sync terminé",
			"gamertag", e.gamertag,
			"perf_scores", postResult.PerfScoresComputed,
			"lusr_updated", postResult.LUSRUpdated,
			"views_refreshed", postResult.ViewsRefreshed,
			"achievements_synced", postResult.AchievementsSynced,
		)
	}

	// ─── sync_meta ──────────────────────────────────────────────────────────────
	if err := SetSyncMeta(playerDB, "last_delta_sync", time.Now().UTC().Format(time.RFC3339)); err != nil {
		result.AddWarning(fmt.Sprintf("SetSyncMeta: %v", err))
	}

	// ─── Hook Prestige (post-sync) ──────────────────────────────────────────────
	// Best-effort : ré-évalue les défis Prestige actifs après ingestion.
	// No-op si feature flag PRESTIGE_ENABLED off ou si le hook n'est pas câblé.
	// Le hook ne propage jamais d'erreur pour ne pas casser le sync.
	if e.prestigeHook != nil {
		e.prestigeHook(ctx, e.gamertag, e.titleSlug)
	}

	result.FinishedAt = time.Now()
	result.DurationSeconds = result.FinishedAt.Sub(result.StartedAt).Seconds()

	slog.InfoContext(ctx, "sync: terminé",
		"gamertag", e.gamertag, "mode", mode,
		"inserted", result.MatchesInserted,
		"skipped", result.MatchesSkipped,
		"medals", result.MedalsInserted,
		"participants", result.ParticipantsDone,
		"warnings", len(result.Warnings),
		"duration_s", fmt.Sprintf("%.2f", result.DurationSeconds),
		"status", result.Status(),
	)
	return result, nil
}

// runConditionalPostSync exécute le pipeline complet si des matchs ont été insérés,
// sinon rafraîchit au moins la carrière pour mettre à jour le snapshot joueur.
func (e *SyncEngine) runConditionalPostSync(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	client HaloClient,
	matchesInserted int,
	insertedIDs []string,
) domain.PostSyncResult {
	if matchesInserted > 0 {
		slog.InfoContext(ctx, "sync: lancement pipeline post-sync", "gamertag", e.gamertag)
		return e.runPostSyncPipeline(ctx, playerDB, sharedDB, client, insertedIDs)
	}

	// Pas de nouveaux matchs : on tente d'abord un heal skill — si ça remplit
	// des champs (team_mmr, kills_expected), il faut quand même lancer le
	// pipeline post-sync complet pour recalculer perf/engagement/LUSR/citations
	// qui dépendent de ces champs.
	healed, healErr := healSkillForMissingMatches(ctx, sharedDB, client, e.xuid, 200)
	if healErr != nil {
		slog.WarnContext(ctx, "sync: skill heal échoué (no-insert path)", "gamertag", e.gamertag, "err", healErr)
	}
	// Détecter aussi les matchs avec scores manquants (engagement/perf NULL).
	// Si présents, on lance le PostSync complet pour les combler.
	needsScoreRefresh, _ := hasMatchesNeedingScoreRefresh(ctx, playerDB, sharedDB, e.xuid)
	if healed > 0 || needsScoreRefresh {
		slog.InfoContext(ctx, "sync: aucun match inséré — heal/scores → lancement post-sync complet",
			"gamertag", e.gamertag, "matches_healed", healed, "needs_score_refresh", needsScoreRefresh)
		return e.runPostSyncPipeline(ctx, playerDB, sharedDB, client, nil)
	}
	slog.DebugContext(ctx, "sync: aucun match inséré — refresh CSR + achievements seul (carrière live découplé)", "gamertag", e.gamertag)
	// Carrière (XP + Spartan ID) retirée du post-sync : service.CareerLiveService
	// la rafraîchit live à chaque chargement de /pages/home.
	e.runCSRSnapshotSync(ctx, playerDB, client)
	return domain.PostSyncResult{
		AchievementsSynced: e.runAchievementsSync(ctx, playerDB),
	}
}

// hasMatchesNeedingScoreRefresh indique si au moins un match a des scores
// manquants (performance OR engagement IS NULL) parmi les matchs joués par
// ce joueur. Heuristique pour décider si runPostSyncPipeline doit tourner
// même quand aucun nouveau match n'a été inséré.
func hasMatchesNeedingScoreRefresh(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) (bool, error) {
	_ = sharedDB // signature future-proof si on veut joindre shared.match_participants
	_ = xuid
	var n int
	err := playerDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM player_match_enrichment
		WHERE engagement_score IS NULL OR performance_score IS NULL
		LIMIT 1
	`).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// processMatch récupère, transforme et insère un match dans les deux DBs.
func (e *SyncEngine) processMatch(
	ctx context.Context,
	client HaloClient,
	sharedDB, playerDB, globalDB *sql.DB,
	result *domain.SyncResult,
	matchID string,
	opts domain.SyncOptions,
) error {
	start := time.Now()
	slog.DebugContext(ctx, "processMatch: début", "gamertag", e.gamertag, "match_id", matchID)

	matchJSON, err := client.GetMatchStats(ctx, matchID)
	if err != nil {
		slog.WarnContext(ctx, "processMatch: GetMatchStats échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return fmt.Errorf("GetMatchStats: %w", err)
	}

	// ─── match_registry ────────────────────────────────────────────────────────
	reg, err := ExtractRegistry(matchJSON, e.gamertag)
	if err != nil {
		slog.WarnContext(ctx, "processMatch: ExtractRegistry échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return fmt.Errorf("ExtractRegistry: %w", err)
	}
	// Enrichissement post-Extract : résout les UUIDs bruts en noms canoniques
	// via metadata.asset_translations[en-US] AVANT l'INSERT, pour ne pas
	// stocker `playlist_name = playlist_id` quand l'API Halo n'a pas retourné
	// de PublicName. Best-effort : nil metaDB → no-op (préserve le fallback
	// historique). Cf. thought_log 2026-05-09.
	if err := EnrichRegistryFromMetadata(ctx, e.metaDB, reg); err != nil {
		slog.WarnContext(ctx, "processMatch: EnrichRegistryFromMetadata non-bloquant",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
	}
	if err := InsertRegistryIfNotExists(sharedDB, *reg); err != nil {
		slog.ErrorContext(ctx, "processMatch: InsertRegistry échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return fmt.Errorf("InsertRegistry: %w", err)
	}

	// ─── match_participants ────────────────────────────────────────────────────
	if opts.WithParticipants {
		participants := ExtractParticipants(matchJSON)

		// Garantir gamertag sur la row du joueur synchronisé.
		ensureGamertagForSelf(participants, e.xuid, e.gamertag)

		// Skill API (séparé du stats endpoint) : team_mmr, enemy_mmr, kills/deaths_expected.
		// Non-bloquant : un échec produit un warning mais le sync continue.
		if xuids := ParticipantXUIDs(participants); len(xuids) > 0 {
			skillData, skillErr := client.GetMatchSkill(ctx, matchID, xuids)
			if skillErr != nil {
				slog.WarnContext(ctx, "processMatch: GetMatchSkill échoué (continuing without skill)",
					"gamertag", e.gamertag, "match_id", matchID, "err", skillErr,
				)
				result.Warnings = append(result.Warnings, fmt.Sprintf("skill %s: %v", matchID, skillErr))
			} else if len(skillData) > 0 {
				participants = MergeSkillIntoParticipants(participants, skillData)
				slog.DebugContext(ctx, "processMatch: skill merged",
					"match_id", matchID, "players_with_skill", len(skillData),
				)
				// CSR par-match : pour les matchs classés, le payload skill
				// contient RankRecap.PostMatchCsr. On persiste côté player DB.
				// Non-bloquant : tout échec laisse le sync continuer.
				if row := ExtractCSRRowIfRanked(reg, skillData[e.xuid]); row != nil {
					if csrErr := UpsertCSRRow(playerDB, row); csrErr != nil {
						slog.WarnContext(ctx, "processMatch: UpsertCSRRow échoué",
							"gamertag", e.gamertag, "match_id", matchID, "err", csrErr,
						)
					} else {
						slog.DebugContext(ctx, "processMatch: CSR row écrite",
							"match_id", matchID, "tier", row.Tier, "tier_label", row.TierLabel,
						)
					}
				}
			}
		}

		if err := InsertParticipants(sharedDB, participants); err != nil {
			slog.ErrorContext(ctx, "processMatch: InsertParticipants échoué",
				"gamertag", e.gamertag, "match_id", matchID, "count", len(participants), "err", err,
			)
			return fmt.Errorf("InsertParticipants: %w", err)
		}
		result.ParticipantsDone += len(participants)

		aliased := 0
		for _, p := range participants {
			if p.Gamertag != nil && *p.Gamertag != "" {
				// P5.3 : écriture dans la DB globale xbox_aliases.duckdb.
				if globalDB != nil {
					_ = UpsertXUIDAlias(globalDB, p.XUID, *p.Gamertag)
				}
				aliased++
			}
		}
		slog.DebugContext(ctx, "processMatch: participants insérés",
			"match_id", matchID, "participants", len(participants), "aliases_upserted", aliased,
		)
	}

	// ─── medals_earned ─────────────────────────────────────────────────────────
	if opts.WithMedals {
		medals := ExtractMedals(matchJSON)
		if err := InsertMedals(sharedDB, medals); err != nil {
			slog.ErrorContext(ctx, "processMatch: InsertMedals échoué",
				"gamertag", e.gamertag, "match_id", matchID, "count", len(medals), "err", err,
			)
			return fmt.Errorf("InsertMedals: %w", err)
		}
		result.MedalsInserted += len(medals)
		slog.DebugContext(ctx, "processMatch: médailles insérées",
			"match_id", matchID, "medals", len(medals),
		)
	}

	// ─── highlight_events + killer_victim_pairs ──────────────────────────────────────
	if opts.WithHighlightEvents {
		if err := ProcessHighlightEvents(ctx, client, sharedDB, globalDB, matchID, result); err != nil {
			// Non-bloquant : on logge et on continue (pas de return).
			slog.WarnContext(ctx, "processMatch: highlight_events non chargés",
				"gamertag", e.gamertag, "match_id", matchID, "err", err,
			)
			result.Warnings = append(result.Warnings, fmt.Sprintf("highlight_events %s: %v", matchID, err))
		}
	}

	// ─── player_match_enrichment (player DB) ───────────────────────────────────
	if err := UpsertPlayerEnrichment(playerDB, matchID, ""); err != nil {
		slog.ErrorContext(ctx, "processMatch: UpsertPlayerEnrichment échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return fmt.Errorf("UpsertPlayerEnrichment: %w", err)
	}

	// ─── personal_score_awards (player DB) ─────────────────────────────────────
	psaRows := ExtractPersonalScoreAwards(matchJSON, matchID, e.xuid)
	if len(psaRows) > 0 {
		if err := InsertPersonalScoreAwards(playerDB, matchID, e.xuid, psaRows); err != nil {
			slog.WarnContext(ctx, "processMatch: InsertPersonalScoreAwards échoué",
				"gamertag", e.gamertag, "match_id", matchID, "err", err,
			)
			result.Warnings = append(result.Warnings, fmt.Sprintf("psa %s: %v", matchID, err))
		}
	}

	result.MatchesInserted++
	result.InsertedMatchIDs = append(result.InsertedMatchIDs, matchID)
	slog.DebugContext(ctx, "processMatch: terminé",
		"gamertag", e.gamertag, "match_id", matchID,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}

// ─── Fetch phases (parallel fetch + sequential insert) ───

// fetchedMatch contient les données extraites d'un GetMatchStats, prêtes pour insertion.
// Utilisé pour paralléliser les fetches tout en gardant les inserts séquentiels.
type fetchedMatch struct {
	MatchID        string
	Registry       *MatchRegistryRow
	Participants   []ParticipantRow
	Medals         []MedalRow
	PSA            []PersonalScoreAwardRow // PersonalScores du joueur courant (player DB)
	HighlightData  []byte                  // Raw highlight events chunk (ou nil si absent)
	FilmMajorVer   int
	HasHighlights  bool
	HighlightError error // Non-bloquant si présent
	SkillError     error // Non-bloquant si présent
	// CSRRow : ligne CSR à insérer côté player DB. Renseignée uniquement
	// pour les matchs classés dont le payload skill contient RankRecap.
	// Inséré dans insertFetchedMatch.
	CSRRow *MatchCSRRow
}

// fetchMatchData exécute le fetch et l'extraction pour un match (pur, sans DB).
// Retourne les données extraites prêtes pour insertion séquentielle.
func (e *SyncEngine) fetchMatchData(
	ctx context.Context,
	client HaloClient,
	matchID string,
	opts domain.SyncOptions,
) (*fetchedMatch, error) {
	matchJSON, err := client.GetMatchStats(ctx, matchID)
	if err != nil {
		slog.WarnContext(ctx, "sync: GetMatchStats échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return nil, fmt.Errorf("GetMatchStats: %w", err)
	}

	fm := &fetchedMatch{
		MatchID: matchID,
	}

	// Extract registry (obligatoire).
	reg, err := ExtractRegistry(matchJSON, e.gamertag)
	if err != nil {
		slog.WarnContext(ctx, "sync: ExtractRegistry échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return nil, fmt.Errorf("ExtractRegistry: %w", err)
	}
	fm.Registry = reg

	// Extract optionnels.
	if opts.WithParticipants {
		fm.Participants = ExtractParticipants(matchJSON)

		// Garantir gamertag sur la row du joueur synchronisé : l'API renvoie
		// parfois Gamertag/PlayerName vide pour le joueur appelant.
		ensureGamertagForSelf(fm.Participants, e.xuid, e.gamertag)

		// Skill API : team_mmr, enemy_mmr, kills/deaths_expected.
		// Endpoint séparé du stats — non-bloquant : un échec produit un warning.
		if xuids := ParticipantXUIDs(fm.Participants); len(xuids) > 0 {
			skillData, skillErr := client.GetMatchSkill(ctx, matchID, xuids)
			if skillErr != nil {
				fm.SkillError = fmt.Errorf("GetMatchSkill: %w", skillErr)
			} else if len(skillData) > 0 {
				fm.Participants = MergeSkillIntoParticipants(fm.Participants, skillData)
				// CSR par-match : extraction depuis RankRecap si match classé.
				// L'écriture en player DB est différée à insertFetchedMatch.
				fm.CSRRow = ExtractCSRRowIfRanked(fm.Registry, skillData[e.xuid])
			}
		}
	}
	if opts.WithMedals {
		fm.Medals = ExtractMedals(matchJSON)
	}
	// PersonalScores du joueur courant — toujours extraits (pas de flag dédié,
	// même cycle de vie que les participants). La table n'est pas dans shared :
	// l'insertion se fera côté playerDB dans insertFetchedMatch.
	fm.PSA = ExtractPersonalScoreAwards(matchJSON, matchID, e.xuid)
	if opts.WithHighlightEvents {
		data, filmMajorVer, found, err := client.GetHighlightEventsChunk(ctx, matchID)
		fm.HasHighlights = found
		fm.FilmMajorVer = filmMajorVer
		if err != nil {
			fm.HighlightError = fmt.Errorf("GetHighlightEventsChunk: %w", err)
		} else if found {
			fm.HighlightData = data
		}
	}

	return fm, nil
}

// insertFetchedMatch insère les données fetchées d'un match (séquentiel, order-preserving).
func (e *SyncEngine) insertFetchedMatch(
	ctx context.Context,
	sharedDB, playerDB, globalDB *sql.DB,
	result *domain.SyncResult,
	fm *fetchedMatch,
) error {
	// Registry (obligatoire).
	if err := InsertRegistryIfNotExists(sharedDB, *fm.Registry); err != nil {
		slog.ErrorContext(ctx, "sync: InsertRegistry échoué",
			"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
		)
		return fmt.Errorf("InsertRegistry: %w", err)
	}

	// Participants.
	if len(fm.Participants) > 0 {
		if fm.SkillError != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("skill %s: %v", fm.MatchID, fm.SkillError))
		}
		if err := InsertParticipants(sharedDB, fm.Participants); err != nil {
			slog.ErrorContext(ctx, "sync: InsertParticipants échoué",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "count", len(fm.Participants), "err", err,
			)
			return fmt.Errorf("InsertParticipants: %w", err)
		}
		result.ParticipantsDone += len(fm.Participants)

		// Phase 2 du plan PLAN_BITMASKS_AUDIT_FIX : marquer le bit
		// participants pour que `levelup backfill --participants` ne re-traite
		// pas indéfiniment ce match.
		if markErr := MarkParticipantsDone(sharedDB, fm.MatchID); markErr != nil {
			slog.WarnContext(ctx, "sync: MarkParticipantsDone échoué",
				"match_id", fm.MatchID, "err", markErr)
		}

		// Phase 2 — skill bits : on ne marque que si l'API skill a renvoyé des
		// données (fm.SkillError nil ET team_mmr présent sur ≥1 participant).
		// MarkSkillLoaded filtre lui-même sur team_mmr IS NOT NULL côté SQL.
		if fm.SkillError == nil && hasAnyTeamMMR(fm.Participants) {
			if markErr := MarkSkillLoaded(sharedDB, fm.MatchID); markErr != nil {
				slog.WarnContext(ctx, "sync: MarkSkillLoaded échoué",
					"match_id", fm.MatchID, "err", markErr)
			}
		}

		// Upsert XUID aliases.
		aliased := 0
		for _, p := range fm.Participants {
			if p.Gamertag != nil && *p.Gamertag != "" {
				if globalDB != nil {
					_ = UpsertXUIDAlias(globalDB, p.XUID, *p.Gamertag)
				}
				aliased++
			}
		}
		slog.DebugContext(ctx, "sync: participants insérés",
			"match_id", fm.MatchID, "participants", len(fm.Participants), "aliases_upserted", aliased,
		)
	}

	// Medals.
	if len(fm.Medals) > 0 {
		if err := InsertMedals(sharedDB, fm.Medals); err != nil {
			slog.ErrorContext(ctx, "sync: InsertMedals échoué",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "count", len(fm.Medals), "err", err,
			)
			return fmt.Errorf("InsertMedals: %w", err)
		}
		result.MedalsInserted += len(fm.Medals)
		slog.DebugContext(ctx, "sync: médailles insérées",
			"match_id", fm.MatchID, "medals", len(fm.Medals),
		)
	}

	// Highlight events.
	if fm.HasHighlights && fm.HighlightData != nil {
		if err := insertHighlightEventsFromData(ctx, sharedDB, globalDB, fm.MatchID, fm.HighlightData, fm.FilmMajorVer, result); err != nil {
			slog.WarnContext(ctx, "sync: highlight_events insertion échouée",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
			)
			result.Warnings = append(result.Warnings, fmt.Sprintf("highlight_events %s: %v", fm.MatchID, err))
		}
	} else if fm.HighlightError != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("highlight_events %s: %v", fm.MatchID, fm.HighlightError))
	}

	// Player enrichment.
	if err := UpsertPlayerEnrichment(playerDB, fm.MatchID, ""); err != nil {
		slog.ErrorContext(ctx, "sync: UpsertPlayerEnrichment échoué",
			"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
		)
		return fmt.Errorf("UpsertPlayerEnrichment: %w", err)
	}

	// PersonalScoreAwards (player DB, par joueur synchronisé). Non-bloquant :
	// un échec produit un warning, le sync continue.
	if len(fm.PSA) > 0 {
		if err := InsertPersonalScoreAwards(playerDB, fm.MatchID, e.xuid, fm.PSA); err != nil {
			slog.WarnContext(ctx, "sync: InsertPersonalScoreAwards échoué",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
			)
			result.Warnings = append(result.Warnings, fmt.Sprintf("psa %s: %v", fm.MatchID, err))
		}
	}

	// CSR par-match (player DB). Renseigné par fetchMatchData uniquement pour
	// les matchs classés dont RankRecap était présent. Non-bloquant.
	if fm.CSRRow != nil {
		if err := UpsertCSRRow(playerDB, fm.CSRRow); err != nil {
			slog.WarnContext(ctx, "sync: UpsertCSRRow échoué",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err,
			)
			result.Warnings = append(result.Warnings, fmt.Sprintf("csr %s: %v", fm.MatchID, err))
		} else {
			slog.DebugContext(ctx, "sync: CSR row écrite",
				"match_id", fm.MatchID, "tier", fm.CSRRow.Tier, "tier_label", fm.CSRRow.TierLabel,
			)
		}
	}

	result.MatchesInserted++
	result.InsertedMatchIDs = append(result.InsertedMatchIDs, fm.MatchID)
	return nil
}

// insertHighlightEventsFromData parse et insère les highlight events à partir de données déjà fetchées.
// Helper utilisé par insertFetchedMatch pour injection de dépendance.
func insertHighlightEventsFromData(
	ctx context.Context,
	sharedDB, globalDB *sql.DB,
	matchID string,
	data []byte,
	filmMajorVersion int,
	result *domain.SyncResult,
) error {
	if len(data) == 0 {
		return nil // Pas de données — OK, pas d'erreur.
	}

	events, err := analysis.ParseHighlightEvents(data, filmMajorVersion)
	if err != nil {
		return fmt.Errorf("ParseHighlightEvents: %w", err)
	}
	if len(events) == 0 {
		// Anomalie : on a téléchargé un chunk non-vide mais le parser
		// n'a rien extrait. Avant le fix bit-aligné (mai 2026), ce cas
		// était silencieusement loggé en DEBUG et faisait perdre tout
		// l'historique highlight events. Désormais : WARN + compteur
		// expvar pour qu'une regression soit immédiatement visible.
		observability.IncCounter("highlight_events_parse_anomaly_total")
		slog.WarnContext(ctx, "highlight_events parse_anomaly: chunk non-vide mais 0 events extraits",
			"match_id", matchID,
			"film_version", filmMajorVersion,
			"data_size", len(data),
		)
		if result != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("highlight_events parse_anomaly %s: chunk %d bytes v%d → 0 events", matchID, len(data), filmMajorVersion))
		}
		return nil
	}

	n, err := InsertHighlightEvents(sharedDB, matchID, events)
	if err != nil {
		return fmt.Errorf("InsertHighlightEvents: %w", err)
	}

	// Upsert XUID aliases from events.
	if globalDB != nil {
		for _, ev := range events {
			if ev.XUID != 0 && ev.Gamertag != "" {
				_ = UpsertXUIDAlias(globalDB, strconv.FormatUint(ev.XUID, 10), ev.Gamertag)
			}
		}
	}

	if n > 0 {
		result.EventsInserted += n
		_ = MarkEventsLoaded(sharedDB, matchID)
	}

	// Fix Phase 1bis (mai 2026) : ne marquer MBitKillerVictim que si l'insert
	// a réellement réussi. Avant, l'insert + le mark étaient appelés
	// inconditionnellement avec `_ =` qui swallowait l'erreur — bit menteur
	// dormant, masqué tant que les events n'arrivaient pas (parser cassé).
	if pairsErr := InsertKillerVictimPairsFromEvents(sharedDB, matchID, events); pairsErr != nil {
		slog.WarnContext(ctx, "InsertKillerVictimPairs échoué", "match_id", matchID, "err", pairsErr)
		if result != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("killer_victim_pairs %s: %v", matchID, pairsErr))
		}
	} else {
		_ = MarkKillerVictimLoaded(sharedDB, matchID)
	}

	return nil
}

// ProcessHighlightEvents télécharge le chunk highlight events, le parse et
// insère les events + paires killer/victim dans la shared DB.
// Retourne une erreur uniquement en cas de défaillance fatale (non-nil = warning dans processMatch).
//
// Exposé (majuscule) pour les outils de replay : cmd/replay_highlight_events.
func ProcessHighlightEvents(
	ctx context.Context,
	client HaloClient,
	sharedDB, globalDB *sql.DB,
	matchID string,
	result *domain.SyncResult,
) error {
	data, filmMajorVersion, found, err := client.GetHighlightEventsChunk(ctx, matchID)
	if err != nil {
		return fmt.Errorf("GetHighlightEventsChunk: %w", err)
	}
	if !found || len(data) == 0 {
		slog.DebugContext(ctx, "processHighlightEvents: film absent ou chunk vide",
			"match_id", matchID, "found", found, "data_len", len(data),
		)
		// Marquer events_loaded=TRUE pour ne pas retenter à chaque sync : le
		// film 404 est définitif (Halo ne sauve pas le film de tous les matchs).
		if markErr := MarkEventsLoaded(sharedDB, matchID); markErr != nil {
			slog.DebugContext(ctx, "MarkEventsLoaded échoué (no-film)",
				"match_id", matchID, "err", markErr)
		}
		return nil
	}

	events, err := analysis.ParseHighlightEvents(data, filmMajorVersion)
	if err != nil {
		return fmt.Errorf("ParseHighlightEvents: %w", err)
	}
	if len(events) == 0 {
		// Anomalie : chunk téléchargé non-vide mais 0 event parsé.
		// Voir insertHighlightEventsFromData pour la justification.
		observability.IncCounter("highlight_events_parse_anomaly_total")
		slog.WarnContext(ctx, "highlight_events parse_anomaly: chunk non-vide mais 0 events extraits",
			"match_id", matchID,
			"film_version", filmMajorVersion,
			"data_size", len(data),
		)
		if result != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("highlight_events parse_anomaly %s: chunk %d bytes v%d → 0 events", matchID, len(data), filmMajorVersion))
		}
		return nil
	}

	n, err := InsertHighlightEvents(sharedDB, matchID, events)
	if err != nil {
		return fmt.Errorf("InsertHighlightEvents: %w", err)
	}

	// Upsert les gamertags extraits depuis le film (source la plus fiable).
	// P5.3 : ecriture dans la DB globale xbox_aliases.
	aliasCount := 0
	if globalDB != nil {
		for _, ev := range events {
			if ev.XUID != 0 && ev.Gamertag != "" {
				if uErr := UpsertXUIDAlias(globalDB, strconv.FormatUint(ev.XUID, 10), ev.Gamertag); uErr == nil {
					aliasCount++
				}
			}
		}
	}

	if n > 0 {
		result.EventsInserted += n
		if markErr := MarkEventsLoaded(sharedDB, matchID); markErr != nil {
			slog.WarnContext(ctx, "MarkEventsLoaded échoué", "match_id", matchID, "err", markErr)
		}
	}

	pairsErr := InsertKillerVictimPairsFromEvents(sharedDB, matchID, events)
	if pairsErr != nil {
		slog.WarnContext(ctx, "InsertKillerVictimPairs échoué", "match_id", matchID, "err", pairsErr)
		// Non-bloquant : on continue.
	} else {
		if markErr := MarkKillerVictimLoaded(sharedDB, matchID); markErr != nil {
			slog.WarnContext(ctx, "MarkKillerVictimLoaded échoué", "match_id", matchID, "err", markErr)
		}
	}

	slog.DebugContext(ctx, "processHighlightEvents: terminé",
		"match_id", matchID,
		"film_version", filmMajorVersion,
		"events_parsed", len(events),
		"events_inserted", n,
		"aliases_upserted", aliasCount,
		"killer_victim_err", pairsErr,
	)
	return nil
}

// hasAnyTeamMMR retourne true si au moins un participant a team_mmr renseigné.
// Utilisé pour décider si MarkSkillLoaded doit être appelé après
// MergeSkillIntoParticipants (Phase 2 plan PLAN_BITMASKS_AUDIT_FIX).
func hasAnyTeamMMR(parts []ParticipantRow) bool {
	for _, p := range parts {
		if p.TeamMMR != nil {
			return true
		}
	}
	return false
}

// loadKnownMatchIDs retourne l'ensemble des match_ids déjà présents dans
// player_match_enrichment (player DB).
func loadKnownMatchIDs(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT match_id FROM player_match_enrichment")
	if err != nil {
		// Table peut ne pas exister si le schéma vient d'être créé — OK.
		return map[string]bool{}, nil
	}
	defer rows.Close()

	known := make(map[string]bool, 256)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			known[id] = true
		}
	}
	return known, rows.Err()
}

// runPostSyncPipeline exécute le pipeline post-sync :
// 1. Performance scores
// 2. LUSR (TrueSkill 2)
// 3. Career rank
// 4. Aggregates (materialized views)
func (e *SyncEngine) runPostSyncPipeline(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	client HaloClient,
	insertedIDs []string,
) domain.PostSyncResult {
	var r domain.PostSyncResult

	// Sprint B1 commit 18 : event_id pour tracer le pipeline post-sync à
	// travers ses 14+ étapes (stats heal, skill heal, events heal, weapons,
	// bot teammate, sessions, perf scores, engagement, LUSR, citations, CSR,
	// friends, aggregates). Tous les sous-logs hériteront automatiquement.
	ctx, evID := logging.WithEvent(ctx, "sync.postSync:"+e.gamertag)
	slog.InfoContext(ctx, "post-sync: pipeline démarré",
		"gamertag", e.gamertag, "matches_inserted", len(insertedIDs), "event", evID)

	// -1.5 Stats re-extraction heal — comble max_killing_spree, grenade/melee/
	// power_weapon kills, time_played_seconds, avg_life_seconds, gamertag,
	// team_X_ps_score pour les matchs synchronisés avec un ancien binaire.
	// Détection via max_killing_spree IS NULL. Limit 10 pour amortir.
	if n, err := healStatsForRecentMatches(ctx, sharedDB, client, e.xuid, e.gamertag, 10); err != nil {
		slog.WarnContext(ctx, "post-sync: stats heal échoué", "gamertag", e.gamertag, "err", err)
	} else if n > 0 {
		slog.InfoContext(ctx, "post-sync: stats self-heal", "gamertag", e.gamertag, "matches_healed", n)
	}

	// -1. Skill self-heal — comble team_mmr/enemy_mmr/kills_expected/deaths_expected
	// pour les matchs synchronisés AVANT que GetMatchSkill ne soit câblé dans
	// processMatch (ou avec un échec transitoire). Idempotent : 0 appel API
	// si tout est déjà rempli. Doit tourner avant performance/LUSR qui
	// dépendent de team_mmr et kills_expected.
	if n, err := healSkillForMissingMatches(ctx, sharedDB, client, e.xuid, 200); err != nil {
		slog.WarnContext(ctx, "post-sync: skill heal échoué", "gamertag", e.gamertag, "err", err)
	} else if n > 0 {
		slog.InfoContext(ctx, "post-sync: skill self-heal", "gamertag", e.gamertag, "matches_healed", n)
	}

	// -0.5 Highlight events / killer_victim heal pour les matchs récents où
	// events_loaded=FALSE (matchs syncés avant que processHighlightEvents ne
	// soit câblé). Best-effort : films absents → 404 silencieux. globalDB nil
	// est OK : xuid_aliases déjà résolu pour ces matchs.
	// Limit 20 pour amortir : processHighlightEvents marque events_loaded=TRUE
	// même sur 404, donc converge en quelques syncs.
	if h, nf, err := healEventsForRecentMatches(ctx, sharedDB, nil, client, 20); err != nil {
		slog.WarnContext(ctx, "post-sync: events heal échoué", "gamertag", e.gamertag, "err", err)
	} else if h > 0 || nf > 0 {
		slog.InfoContext(ctx, "post-sync: events self-heal",
			"gamertag", e.gamertag, "healed", h, "no_film", nf)
	}

	// -0.4 Weapon kills heal pour les matchs récents où le pipeline n'a jamais
	// tourné (bit MBitWeaponKills absent dans match_registry). Dépend des
	// highlight_events ci-dessus (kills attribution lit highlight_events).
	// Limit 10 : weapon kills marque le bit MBitWeaponKills aussi sur no-film,
	// donc converge en quelques syncs.
	if h, nf, err := healWeaponKillsForRecentMatches(ctx, sharedDB, client, e.xuid, 10); err != nil {
		slog.WarnContext(ctx, "post-sync: weapon heal échoué", "gamertag", e.gamertag, "err", err)
	} else if h > 0 || nf > 0 {
		slog.InfoContext(ctx, "post-sync: weapon self-heal",
			"gamertag", e.gamertag, "healed", h, "no_film", nf)
	}

	// -0.3 had_bot_teammate — dérivé des participants (cheap SQL, pas d'API).
	// Idempotent : skip les rows déjà à TRUE.
	if n, err := computeAndPersistHadBotTeammate(ctx, playerDB, sharedDB, e.xuid); err != nil {
		slog.WarnContext(ctx, "post-sync: had_bot_teammate échoué", "gamertag", e.gamertag, "err", err)
	} else if n > 0 {
		slog.InfoContext(ctx, "post-sync: had_bot_teammate", "gamertag", e.gamertag, "rows_updated", n)
	}

	// 0. Session assignments — auto-recalc session_id pour les nouveaux matchs.
	// Best-effort : un échec ne bloque pas le pipeline. Les amis sont
	// résolus depuis le friendsLoader (settings.FriendGamertags). Sans loader
	// (legacy), on retombe en TeamChangeMode=teammates.
	{
		var friends []string
		if e.friendsLoader != nil {
			if fs, ferr := e.friendsLoader(); ferr == nil {
				friends = fs
			}
		}
		opts := analysis.DefaultSessionOptions()
		if n, err := recalculateSessionsInline(ctx, playerDB, sharedDB, e.xuid, opts, friends); err != nil {
			slog.WarnContext(ctx, "post-sync: sessions échoué", "gamertag", e.gamertag, "err", err)
		} else if n > 0 {
			r.SessionsAssigned = n
			slog.DebugContext(ctx, "post-sync: sessions recalculées", "gamertag", e.gamertag, "count", n)
		}
	}

	// 1. Performance scores
	slog.DebugContext(ctx, "post-sync: calcul perf scores", "gamertag", e.gamertag)
	if n, err := batchComputePerformanceScores(playerDB, sharedDB, e.xuid, nil, false); err != nil {
		slog.WarnContext(ctx, "post-sync: perf scores échoué", "gamertag", e.gamertag, "err", err)
	} else {
		r.PerfScoresComputed = n
		slog.DebugContext(ctx, "post-sync: perf scores calculés", "gamertag", e.gamertag, "count", n)
	}

	// 1.5 Engagement scores (Phase 3 plan engagement) — best-effort,
	// skip silencieux si migration Phase 2 non appliquee.
	slog.DebugContext(ctx, "post-sync: calcul engagement scores", "gamertag", e.gamertag)
	if n, err := batchComputeEngagementScores(ctx, playerDB, sharedDB, e.xuid, false); err != nil {
		slog.WarnContext(ctx, "post-sync: engagement scores échoué", "gamertag", e.gamertag, "err", err)
	} else if n > 0 {
		r.EngagementScoresComputed = n
		slog.DebugContext(ctx, "post-sync: engagement scores calculés", "gamertag", e.gamertag, "count", n)
	}

	// 1.5.b Recompute des engagement coefficients depuis la mediane glissante
	// des paces persistees ci-dessus. Sans ce recompute, coef_team_share reste
	// a 1.0 (cold-start) → pace_attendu = pace_team → courbes superposees a
	// l'ecran (cf. .ai/V7/PLAN_ENGAGEMENT_IMPLEMENTATION.md §4.4).
	if n, err := batchRecomputeCoefficients(ctx, playerDB, e.xuid); err != nil {
		slog.WarnContext(ctx, "post-sync: engagement coefs échoué", "gamertag", e.gamertag, "err", err)
	} else if n > 0 {
		r.EngagementCoefsUpdated = n
		slog.DebugContext(ctx, "post-sync: engagement coefs mis à jour", "gamertag", e.gamertag, "count", n)
	}

	// 1.52 Assists model — OLS per-mode, skip silencieux si migration absente.
	// force=false : ne recalcule que si player_assists_model est vide (cold-start).
	// Un nouveau sync peut amener des données → on recalcule si table vide.
	if n, err := batchComputePlayerAssistsModel(ctx, playerDB, sharedDB, e.xuid, false); err != nil {
		slog.WarnContext(ctx, "post-sync: assists model échoué", "gamertag", e.gamertag, "err", err)
	} else if n > 0 {
		slog.InfoContext(ctx, "post-sync: assists model calculé", "gamertag", e.gamertag, "n_modes", n)
	}

	// 1.55 Weapon kills — pipeline film pour les matchs nouvellement insérés.
	// Best-effort : films absents (404/410) sont normaux pour les vieux matchs
	// et n'échouent pas le sync. Limité aux nouveaux matchs (insertedIDs) pour
	// éviter de re-traiter l'historique à chaque sync.
	if len(insertedIDs) > 0 {
		done, noFilm, werr := processWeaponKillsInline(ctx, sharedDB, client, e.xuid, insertedIDs)
		if werr != nil {
			slog.WarnContext(ctx, "post-sync: weapon kills échoué", "gamertag", e.gamertag, "err", werr)
		}
		r.WeaponKillsProcessed = done
		r.WeaponKillsNoFilm = noFilm
		if done > 0 || noFilm > 0 {
			slog.InfoContext(ctx, "post-sync: weapon kills",
				"gamertag", e.gamertag, "done", done, "no_film", noFilm)
		}
	}

	// 1.6 Citations (best-effort) — calcul des deltas pour les matchs absents
	// de match_citations. Skip silencieux si metadata.duckdb introuvable ou si
	// citation_mappings vide. Ne propage jamais d'erreur (le sync ne doit pas
	// echouer a cause des citations).
	if n, err := e.runPostSyncCitations(ctx, playerDB, sharedDB); err != nil {
		slog.WarnContext(ctx, "post-sync: citations échoué", "gamertag", e.gamertag, "err", err)
	} else if n > 0 {
		slog.InfoContext(ctx, "post-sync: citations calculées",
			"gamertag", e.gamertag, "match_count", n)
	}

	// 2. LUSR (TrueSkill 2) — best-effort medal data depuis metadata DB
	slog.DebugContext(ctx, "post-sync: calcul LUSR", "gamertag", e.gamertag)
	medalMap := e.loadMedalExploitMapBestEffort(ctx, sharedDB)
	if n, err := batchComputeLUSR(playerDB, sharedDB, e.xuid, medalMap, false); err != nil {
		slog.WarnContext(ctx, "post-sync: LUSR échoué", "gamertag", e.gamertag, "err", err)
	} else {
		r.LUSRUpdated = n
		slog.DebugContext(ctx, "post-sync: LUSR mis à jour", "gamertag", e.gamertag, "count", n)
	}

	// 3. Career rank — DÉCOUPLÉ du post-sync depuis 2026-05-14.
	// Le flow XP + Spartan ID est désormais géré par service.CareerLiveService
	// (throttle 5 min / 6 h + fallback DB per-field), appelé depuis HomeService
	// à chaque chargement de /pages/home. Voir .ai/thought_log.md.
	// domain.PostSyncResult.CareerSynced reste dans le struct (compat e2e tests)
	// mais n'est plus jamais positionné à true ici.

	// 3.1 CSR snapshots (best-effort, skip silencieux si csrSeasonID vide).
	// Maintenu dans le post-sync : le CSR ne bouge que sur fin de match ranked,
	// donc le déclencheur "nouveau match" reste pertinent.
	e.runCSRSnapshotSync(ctx, playerDB, client)

	// 3.5 Friends recompute is_with_friends (best-effort).
	// Avant l'étape 4 (aggregates) pour éviter un double-refresh : on passe
	// refreshAggregates=false, le refresh natif de l'engine couvre les UPDATEs.
	// Skip silencieux si pas de loader (legacy) ou liste vide.
	if e.friendsLoader != nil {
		if friends, ferr := e.friendsLoader(); ferr != nil {
			slog.WarnContext(ctx, "post-sync: friends loader échoué", "gamertag", e.gamertag, "err", ferr)
		} else if len(friends) > 0 {
			slog.DebugContext(ctx, "post-sync: friends recompute", "gamertag", e.gamertag, "friends_count", len(friends))
			fres, err := RecomputeIsWithFriendsCore(ctx, playerDB, sharedDB, e.xuid, friends, false)
			if err != nil {
				slog.WarnContext(ctx, "post-sync: friends recompute échoué", "gamertag", e.gamertag, "err", err)
			} else if fres.MatchesPromoted > 0 {
				r.MatchesPromotedFriends = fres.MatchesPromoted
				slog.InfoContext(ctx, "post-sync: matchs reclasses comme escouade-amis",
					"gamertag", e.gamertag,
					"promoted", fres.MatchesPromoted,
				)
			}
		}
	}

	// 4. Aggregates (materialized views)
	slog.DebugContext(ctx, "post-sync: refresh aggregates player", "gamertag", e.gamertag)
	if n, err := refreshAggregates(playerDB); err != nil {
		slog.WarnContext(ctx, "post-sync: aggregates échoué", "gamertag", e.gamertag, "err", err)
	} else {
		r.ViewsRefreshed = n
	}
	slog.DebugContext(ctx, "post-sync: refresh shared views", "gamertag", e.gamertag)
	if n, err := refreshSharedViews(sharedDB); err != nil {
		slog.WarnContext(ctx, "post-sync: shared views échoué", "gamertag", e.gamertag, "err", err)
	} else {
		r.ViewsRefreshed += n
	}

	// 5. Achievements Xbox (fire-and-forget, non bloquant en cas d'erreur token)
	r.AchievementsSynced = e.runAchievementsSync(ctx, playerDB)

	return r
}

// runCSRSnapshotSync récupère les classements CSR du joueur pour la saison courante
// et les persiste dans player_csr_snapshots. Best-effort : skippé si csrSeasonID vide.
func (e *SyncEngine) runCSRSnapshotSync(ctx context.Context, playerDB *sql.DB, client HaloClient) {
	if strings.TrimSpace(e.csrSeasonID) == "" {
		return
	}
	slog.DebugContext(ctx, "post-sync: sync CSR snapshots", "gamertag", e.gamertag, "season", e.csrSeasonID)
	n, err := syncPlayerCSRs(ctx, client, playerDB, e.xuid, e.csrSeasonID)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: CSR snapshots échoué", "gamertag", e.gamertag, "err", err)
		return
	}
	slog.DebugContext(ctx, "post-sync: CSR snapshots sauvegardés", "gamertag", e.gamertag, "count", n)
}

// runAchievementsSync récupère les achievements Xbox pour le joueur et les persiste.
// Retourne true si la sync a réussi, false en cas d'erreur (non bloquante).
// Nécessite e.provider non nil ; skippé silencieusement sinon.
func (e *SyncEngine) runAchievementsSync(ctx context.Context, playerDB *sql.DB) bool {
	if e.provider == nil {
		slog.DebugContext(ctx, "achievements: provider nil — sync ignorée", "gamertag", e.gamertag)
		return false
	}

	// Résoudre l'access_token depuis sync_meta DuckDB.
	accessToken, err := resolveAccessTokenFromDB(ctx, playerDB, e.gamertag, e.provider)
	if err != nil {
		slog.WarnContext(ctx, "achievements: échec résolution access_token",
			"gamertag", e.gamertag, "err", err)
		return false
	}
	if accessToken == "" {
		slog.InfoContext(ctx, "achievements: aucun access_token disponible — sync ignorée",
			"gamertag", e.gamertag)
		return false
	}

	// Obtenir un XSTS token pour Xbox Live.
	xstsResult, err := auth.AcquireXSTSForRTA(ctx, accessToken)
	if err != nil {
		slog.WarnContext(ctx, "achievements: échec acquisition XSTS",
			"gamertag", e.gamertag, "err", err)
		return false
	}

	// Ouvrir la DB metadata (lecture-écriture pour l'upsert).
	metadataDB, err := sql.Open("duckdb", e.metadataDBPath)
	if err != nil {
		slog.WarnContext(ctx, "achievements: ouverture metadata DB échouée",
			"gamertag", e.gamertag, "err", err)
		return false
	}
	defer metadataDB.Close() //nolint:errcheck
	metadataDB.SetMaxOpenConns(1)

	client := NewXboxHTTPClient(xstsResult, titlePkg.XboxTitleIDFor(e.titleSlug))
	if err := SyncAchievements(ctx, client, e.resolver, metadataDB, playerDB, e.xuid, e.titleSlug); err != nil {
		slog.WarnContext(ctx, "achievements: sync échouée",
			"gamertag", e.gamertag, "err", err)
		return false
	}

	slog.InfoContext(ctx, "achievements: sync terminée avec succès", "gamertag", e.gamertag)
	return true
}

// RunAchievementsOnly synchronise uniquement les achievements Xbox du joueur,
// indépendamment du sync des matchs. Utilisé par le CLI sync-achievements pour
// le backfill admin one-shot. Best-effort : retourne false sur erreur (logguée).
//
// Acquiert le dblease sur la player DB pour éviter les collisions avec un sync
// concurrent. Le provider doit être non nil ; sinon retourne false silencieusement.
func (e *SyncEngine) RunAchievementsOnly(ctx context.Context) bool {
	if e.provider == nil {
		slog.WarnContext(ctx, "achievements: provider nil — sync ignorée",
			"gamertag", e.gamertag)
		return false
	}

	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		slog.ErrorContext(ctx, "achievements: lease player DB échoué",
			"gamertag", e.gamertag, "err", err)
		return false
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		slog.ErrorContext(ctx, "achievements: ouverture player DB échouée",
			"gamertag", e.gamertag, "err", err)
		return false
	}
	defer playerHandle.Close() //nolint:errcheck

	return e.runAchievementsSync(ctx, playerHandle.SQLDb())
}

// resolveAccessTokenFromDB lit le cache MSAL et le refresh token depuis sync_meta (DB déjà ouverte),
// puis tente TrySilentRefresh ou TryOAuthRefresh selon ce qui est disponible.
// Retourne ("", nil) si aucun token n'est disponible (non fatal).
//
//nolint:unparam // contrat documenté : second retour non-nil est réservé aux futures erreurs fatales (DB)
func resolveAccessTokenFromDB(
	ctx context.Context,
	playerDB *sql.DB,
	gamertag string,
	provider auth.TokenProvider,
) (string, error) {
	var cacheJSON, refreshToken string
	if err := playerDB.QueryRowContext(ctx,
		"SELECT value FROM sync_meta WHERE key = 'msal_token_cache'").Scan(&cacheJSON); err != nil {
		slog.DebugContext(ctx, "achievements: msal_token_cache absent", "gamertag", gamertag)
	}
	if err := playerDB.QueryRowContext(ctx,
		"SELECT value FROM sync_meta WHERE key = 'oauth_refresh_token'").Scan(&refreshToken); err != nil {
		slog.DebugContext(ctx, "achievements: oauth_refresh_token absent", "gamertag", gamertag)
	}

	// Fallback env var SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>
	if refreshToken == "" && gamertag != "" {
		key := strings.ToUpper(strings.NewReplacer(" ", "_", "-", "_", ".", "_").Replace(gamertag))
		if v := os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_" + key); v != "" {
			refreshToken = v
		}
	}

	if cacheJSON != "" {
		token, err := provider.TrySilentRefresh(ctx, cacheJSON)
		if err == nil && token != "" {
			return token, nil
		}
	}

	if refreshToken != "" {
		token, err := provider.TryOAuthRefresh(ctx, refreshToken)
		if err == nil && token != "" {
			return token, nil
		}
	}

	return "", nil
}
