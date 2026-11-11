// boot_providers.go — fournisseurs du démarrage : lecture partagée et metadata DuckDB,
// saison CSR, filet ART, dépôts et services, réglages et fournisseur de jetons, file de
// persistance. Chaque étape lit ce que la précédente a posé dans bootState.
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/haloadapter"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/capturecli"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/platform/groupstore"
	"levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/service"
	syncpkg "levelup/go-api/internal/sync"
)

// bootOpenDatabases ouvre la lecture partagée et la base metadata, résout la saison CSR
// courante et passe le filet de garde ART.
func bootOpenDatabases(st *bootState) {
	st.gate.setStep(context.Background(), bootStepDatabases)
	openSharedReader(st)
	openMetadataDB(st)
	resolveCSRSeason(st)
	runBootARTGuard(st)
}

// openSharedReader ouvre le lecteur de la base partagée : un SharedDBProvider (ADR 0016, B-swap)
// par base partagée de titre.
//
// shared_matches_v2.duckdb est géré par un SharedDBProvider qui swap dynamiquement RO <-> RW
// autour des leases writer (sync engine). Les handlers HTTP attendent la fenêtre de sync
// (timeout borné, mappé en 503 Retry-After). Cela élimine l'erreur « different configuration »
// qui survenait quand le sync ouvrait shared en RW pendant qu'une instance RO globale existait.
//
// Le fournisseur est posé dans la configuration (SharedProvider, SharedManager) : les pools
// joueur créés ensuite en héritent, et un PlayerDB ne tient AUCUN handle sur le fichier shared
// (le provider de chaque titre en est l'unique détenteur). La lecture per-titre
// (player_resolver) et l'écriture per-titre (livesync Halo 5) résolvent le provider d'un titre
// par SharedManager.For(SharedDBPath(slug)) ; pour le titre par défaut, For() rend CE provider
// (caché par chemin). La fermeture ferme tous les providers via Manager.Close().
//
// `attachShared` (pool.go) reste en place pour les requêtes cross-DB de squad_repo
// (LoadTopTeammates, LoadSquadMatches, LoadTeammateMatches, LoadSynthesisMatches,
// LoadMapStatsForSquad).
func openSharedReader(st *bootState) {
	cfg, sharedPath := st.cfg, st.dbPaths.shared
	sharedMgr := sharedprovider.NewManager()
	provider, err := sharedMgr.For(sharedPath)
	if err != nil {
		slog.Error("ouverture shared_matches_v2 via provider échouée", "err", err)
		os.Exit(1)
	}
	cfg.SharedProvider = provider
	cfg.SharedManager = sharedMgr
	slog.Info("shared_matches_v2: mode sharedprovider (B-swap actif, un provider par base partagée)")
	st.sharedReader, st.closeShared = provider, sharedMgr.Close
}

// openMetadataDB ouvre metadata.duckdb en écriture partagée (reprise du verrou comprise).
func openMetadataDB(st *bootState) {
	metaPath := st.dbPaths.metadata

	// Reprise du verrou metadata (hot-reload, ancien process pas encore parti) : la MÊME reprise
	// que celle des migrations (retryOnFileLock) ; l'échec définitif reste ERROR + exit.
	// IMPORTANT : OpenReadWriteShared (et non OpenReadOnly) pour partager la même
	// instance DuckDB que le pool joueur (pool.go) et le DuckDBIndexStore (assets).
	// Sinon DuckDB rejette toute deuxième connexion sur le même fichier avec
	// "Can't open a connection to same database file with a different configuration".
	var metaDB *duckdb.DB
	if err := retryOnFileLock(context.Background(), "metadata", lockRetryAttempts, lockRetryDelay, func() (openErr error) {
		metaDB, openErr = duckdb.OpenReadWriteShared(metaPath)
		return openErr
	}); err != nil {
		slog.Error("ouverture metadata échouée", "attempts", lockRetryAttempts, "err", err)
		os.Exit(1)
	}
	slog.Debug("DuckDB ouvert")
	st.metaDB = metaDB
}

// resolveCSRSeason résout la saison CSR active depuis metadata.duckdb quand la config n'en fixe pas.
func resolveCSRSeason(st *bootState) {
	cfg, metaDB := st.cfg, st.metaDB

	// --- 3.ter. Résolution CSR season depuis metadata.duckdb ---
	// Si LEVELUP_CSR_SEASON_ID n'est pas défini (ni env, ni app_settings.json),
	// requête csr_season_calendars pour la saison active du jour. Doit être fait
	// AVANT l'initialisation des services qui utilisent cfg.CurrentCSRSeasonID.
	if cfg.CurrentCSRSeasonID == "" {
		if id := resolveCSRSeasonFromDB(metaDB.SQLDb()); id != "" {
			cfg.CurrentCSRSeasonID = id
			slog.Info("csr_season_id résolu depuis csr_season_calendars", "season_id", id)
		} else {
			slog.Warn("csr_season_id absent de csr_season_calendars — sync CSR désactivé")
		}
	}
}

// runBootARTGuard sonde les bases shared et metadata (détection seule, jamais bloquante).
func runBootARTGuard(st *bootState) {
	sharedReader, metaDB := st.sharedReader, st.metaDB

	// --- 3.bis. Filet de garde corruption ART (Phase 1, plan stabilisation
	// 2026-05-22). Scanne shared_matches_v2 + metadata pour détecter les
	// tables dont l'index ART est corrompu (filter pushdown qui rate des
	// rows — cf. INCIDENT_2026-05-20_match_participants_index.md). Non-
	// bloquant : log WARN + métrique expvar si divergence, le serveur démarre.
	// Sample 5 par table ; suffisant pour démasquer le bug qui dépend du
	// contenu de la liste IN, pas d'une valeur unique.
	//
	// Phase 4.1 (2026-05-23) : si match_participants montre une divergence
	// sur shared, auto-heal via swap CTAS (RebuildMatchParticipantsART).
	// Requiert le SharedProvider (mode B-swap) pour AcquireWriter; en mode
	// legacy on log seulement.
	{
		// Phase 5 cleanup (2026-05-24) : auto-heal supprimé — Phase 4 batch
		// INSERT-only path élimine la corruption ART à la racine. On garde la
		// DÉTECTION (BootARTGuard) pour alerte ops, mais le rebuild auto au
		// boot est remplacé par `levelup rebuild-pme-art` (player DB) ; la shared
		// se reconstruit par le step de migration (RebuildMatchParticipantsART). Cf. ADR 0019.
		bootCtx, bootCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer bootCancel()
		if sharedSQL, release, err := sharedReader.Get(bootCtx); err == nil {
			duckdb.BootARTGuard(bootCtx, sharedSQL, "shared", 5)
			release()
		} else {
			slog.WarnContext(bootCtx, "art_guard: shared reader indisponible pour probe boot",
				"err", err)
		}
		duckdb.BootARTGuard(bootCtx, metaDB.SQLDb(), "metadata", 5)
	}
}

// bootBuildServices construit le dépôt de bootstrap, son service et les magasins de comptes,
// groupes, amis et jetons dont il dépend.
func bootBuildServices(st *bootState) {
	cfg, sharedReader, metaDB := st.cfg, st.sharedReader, st.metaDB

	// --- 4. Repositories + services ---
	bootRepo := duckdb.NewBootstrapRepo(sharedReader, metaDB)
	bootSvc := service.NewBootstrapService(cfg, bootRepo).
		WithPrivacyProvider(haloadapter.DefaultHaloProvider).
		// Décompte de matchs TITLE-AWARE : compte le shared du titre COURANT (pas
		// celui figé Infinite du boot). Un switch Halo 5 n'attend plus le provider
		// Infinite tenu par le sync Infinite. Manager.For est caché par path →
		// DefaultSlug retourne le MÊME provider que sharedReader (aucune conn en plus).
		WithMatchCountResolver(func(ctx context.Context, titleSlug string) (int, error) {
			return duckdb.NewBootstrapRepo(cfg.SharedReaderForTitle(titleSlug), metaDB).GetMatchCount(ctx)
		})

	// Auth locale : user store partagé — filtrage ownership des joueurs (ADR 0029,
	// modes password + xbox) et check "first launch" (mode password).
	usersPath := cfg.UsersFilePath()
	us := userstore.NewStore(usersPath)
	bootSvc = bootSvc.WithUserLookup(us)
	if cfg.AuthMode == "password" {
		bootSvc = bootSvc.WithUserStoreEmpty(us.IsEmpty)
	}
	// Groupes/familles (accès mutuel aux données, ADR 0029 multi-groupes). Le set
	// co-membres pilote le filtrage ownership (available_players) et le switch de BDD.
	groupStore := groupstore.NewGroupStore(filepath.Join(cfg.AuthDir, "groups.json"))
	// Amis PAR PROFIL JOUEUR (data/global/player_friends.json) : remplace l'ancien
	// réglage global des amis dans app_settings. Migré une fois au boot.
	friendStore := bootFriendStore(cfg)
	bootSvc = bootSvc.WithCoMemberResolver(func(xuid string) map[string]bool {
		co, _ := groupStore.CoMemberXUIDs(xuid)
		return co
	})

	// PR-B : expose reauth_required (refresh_token mort) du joueur courant au front.
	// Lecture par-xuid dans le MultiUserTokenStore (data/auth/watcher_tokens/{xuid}.json).
	reauthStore := auth.NewMultiUserTokenStore(cfg.WatcherTokensDir())
	bootSvc = bootSvc.WithReauthChecker(reauthStore.IsReauthRequired).
		WithPoolSharingResolver(reauthStore.PoolSharingState)
	st.bootRepo, st.bootSvc, st.us, st.groupStore, st.friendStore = bootRepo, bootSvc, us, groupStore, friendStore
}

// validateBootTypes valide les types critiques côté DuckDB (échec fatal) et sonde career_ranks.
func validateBootTypes(st *bootState) {
	ctx, bootRepo := st.ctx, st.bootRepo
	// --- 5. Sprint 0 : validation des types critiques ---
	if err := bootRepo.ValidateTypes(ctx); err != nil {
		slog.Error("validation types DuckDB échouée", "err", err)
		os.Exit(1)
	}
	if _, err := bootRepo.GetCareerRanksSample(ctx); err != nil {
		slog.Warn("lecture career_ranks échouée", "err", err)
	}
}

// initSettingsAndAuth charge les réglages globaux et instancie le fournisseur de jetons.
func initSettingsAndAuth(st *bootState) {
	cfg := st.cfg

	// --- 6. Scheduler, watcher, puis routeur HTTP ---
	// WithEnforcedDefaults (ADR 0035 D5) : cf. internal/api/server.go — mêmes
	// défauts sûrs des deux côtés, sinon le boot et le routeur divergeraient.
	settingsStore := settings.NewStore(cfg.AppSettingsPath).
		WithEnforcedDefaults(authz.Enforced(cfg.DemoMode, cfg.AuthMode)).
		WithSystemDefaults(cfg.SystemLocale).
		WithAutoSyncDefault(cfg.Capabilities.AutoSyncByDefault)
	// Propage le réglage global "rendement sans assistances" au package analysis
	// dès le boot (sinon le défaut false s'applique jusqu'au premier PATCH).
	if s, lerr := settingsStore.Load(); lerr == nil {
		analysis.SetExcludeAssistsFromYield(s.RendementExcludeAssists)
		if s.RendementExcludeAssists {
			slog.Info("rendement combat : assistances EXCLUES (réglage rendement_exclude_assists actif)")
		}
	}
	tokenProvider := buildTokenProvider(settingsStore, title.DefaultHaloAuthDescriptor())
	st.settingsStore, st.tokenProvider = settingsStore, tokenProvider
}

// openPersistQueue crée la file de persistance asynchrone (WAL) et son worker, sauf en démo
// ou si le chemin synchrone est demandé.
func openPersistQueue(st *bootState) {
	ctx, cfg, pr, boot, autoScheduler := st.ctx, st.cfg, st.pr, st.boot, st.autoScheduler

	// Phase 4.9 (2026-05-24) : BatchQueue serveur-wide default ON. Set
	// LEVELUP_PERSIST_BATCH_ASYNC=0 pour fallback path synchrone (validé Phase 4.5).
	//
	// Cycle de vie du kill-switch (rollback vers le path synchrone) :
	//   - basculement défaut ON : 2026-05-24 (Phase 4.9) ;
	//   - date cible de retrait : >= 2026-Q4 (après >= 1 trimestre prod stable) ;
	//   - critère mesurable : aucun rollback `=0` activé + compteur
	//     `persist_wal_purged_total` stable (aucun WAL purgé sans persist) +
	//     recovery boot (RecoverPending) sans batch orphelin. Alors supprimer le
	//     chemin synchrone, le flag et ses 3 lecteurs (main.go, sync_v2_wiring.go,
	//     auto_sync.go).
	//
	// Path async : queue.Submit + worker + WAL durable (recovery au boot via
	// RecoverPending). Bénéfice : décorrélation sync/persist + résilience crash
	// mid-persist (le batch est journalisé sur disque AVANT le push channel).
	// Cycle de vie : créée 1× au boot, drainée + fermée à shutdown via
	// autoBatchQueue.Drain() + Close() AVANT duckdb.CloseAll() (ordre critique).
	var autoBatchQueue *persist.BatchQueue
	if !boot.isCut(stepPersistQueue) && cfg.PersistBatchAsync {
		walDir := pr.WALDir()
		q, qErr := persist.NewBatchQueue(persist.BatchQueueConfig{
			WALDir:      walDir,
			ChanBufSize: 1000, // cf. Q2 ADR 0019 — backpressure naturelle
		})
		if qErr != nil {
			slog.WarnContext(ctx, "persist: BatchQueue init échoué — fallback path synchrone",
				"err", qErr)
		} else {
			autoBatchQueue = q
			autoScheduler.WithBatchQueue(q)
			// Métrique : chaque WAL purgé sans avoir été persisté (perte
			// potentielle) incrémente persist_wal_purged_total (le log ERROR
			// est émis par PurgeOldWAL). Cf. PLAN_PERSIST_ROBUSTNESS Phase 1.
			q.OnWALPurged = func(info persist.PurgedWALInfo) {
				observability.IncCounter("persist_wal_purged_total")
			}
			slog.InfoContext(ctx, "persist: BatchQueue activée (async path)",
				"wal_dir", walDir)

			// Dashboard monitoring P4 : hook de chronométrage des phases
			// d'écriture (acquire/lease/write par DB) → expvar, sans coupler
			// persist à observability. Posé une fois avant le 1er batch.
			persist.OnPersistPhase = func(phase string, d time.Duration, ok bool) {
				observability.RecordDurationMS("persist_"+phase+"_ms", d.Milliseconds())
				if !ok {
					observability.IncCounter("persist_" + phase + "_err_total")
				}
			}

			startPersistWorker(st, q)
		}
	}
	st.autoBatchQueue = autoBatchQueue
}

// startPersistWorker démarre le worker combiné (shared + player par batch) puis rejoue les
// batches WAL laissés par un crash précédent.
func startPersistWorker(st *bootState, q *persist.BatchQueue) {
	ctx, cfg, pr := st.ctx, st.cfg, st.pr
	sharedPath := st.dbPaths.shared
	workerWG := &st.workerWG

	// Câblage Worker — CombinedPersister écrit shared + player par batch.
	// context.Background() : le Worker doit finir le batch en cours avant
	// de s'arrêter → ne doit pas être annulé par cancelScheduler().
	// Arrêt naturel via autoBatchQueue.Close() (channel close) au shutdown.
	combinedP := persist.NewCombinedPersister(
		func(workerCtx context.Context) (*sql.DB, func(), error) {
			// Étape 0 attribution : fenêtre RW courte de référence (1 batch/TX).
			workerCtx = ctxkeys.WithDBWriterLabel(workerCtx, "persist_worker")
			return syncpkg.AcquireSharedWriterStandalone(workerCtx, cfg.SharedProvider, sharedPath)
		},
		// titleSlug vient du batch (batch.TitleSlug) — chaque batch route vers
		// la player DB de SON titre (multi-titres). En mono-titre = identique.
		func(batchTitleSlug, gamertag string) string { return pr.PlayerDBPath(batchTitleSlug, gamertag) },
	)
	batchWorker := persist.NewWorker("combined", q, persist.TargetShared, combinedP)
	workerWG.Add(1)
	go func() {
		defer workerWG.Done()
		_ = batchWorker.Run(context.Background())
	}()
	slog.InfoContext(ctx, "persist: Worker combiné démarré")

	// Recovery boot : re-soumet les batches WAL pending d'un crash précédent.
	// Appelé APRES le démarrage du Worker pour traitement immédiat.
	if rerr := q.RecoverPending(); rerr != nil {
		slog.WarnContext(ctx, "persist: RecoverPending échoué (non-bloquant)",
			"err", rerr)
	}
}

// buildTokenProvider instancie le TokenProvider : SISUProvider, seul provider
// (login Xbox natif — device-flow login.live.com puis chaîne XBL classique —,
// ZÉRO app Azure : LevelUp est distribué à des self-hosters qui ne peuvent pas
// tous enregistrer une app Azure). L'ancien réglage
// app_settings.json:auth_provider="msal" est ignoré avec un warning.
func buildTokenProvider(settingsStore *settings.Store, authDesc title.AuthDescriptor) auth.TokenProvider {
	// MT-02 (PMT-2 leg 3) : les SISU app/title id viennent du descripteur du titre
	// (byte-identique au défaut Halo). Descripteur incomplet → garde NewSISUProvider().
	newSISU := func() auth.TokenProvider {
		if authDesc.SISUAppID != "" && authDesc.SISUTitleID != "" {
			return auth.NewSISUProviderWithIDs(authDesc.SISUAppID, authDesc.SISUTitleID)
		}
		return auth.NewSISUProvider()
	}
	if s, err := settingsStore.Load(); err != nil {
		slog.Warn("buildTokenProvider: lecture settings échouée, défaut SISU", "err", err)
	} else if s.AuthProvider != "" && s.AuthProvider != "sisu" {
		slog.Warn("buildTokenProvider: auth_provider obsolète ignoré (MSAL retiré 2026-07-15) — SISU",
			"value", s.AuthProvider)
	}
	slog.Info("buildTokenProvider: SISU provider activé",
		"sisu_title_id", authDesc.SISUTitleID)
	return newSISU()
}

// resolveCSRSeasonFromDB retourne le season_id CSR actif depuis csr_season_calendars.
// Retourne "" si la table est absente ou si aucune saison couvre la date courante.
func resolveCSRSeasonFromDB(db *sql.DB) string {
	var id string
	err := db.QueryRowContext(context.Background(), `
		SELECT season_id FROM csr_season_calendars
		WHERE title_id = 'halo_infinite'
		  AND start_date <= CURRENT_DATE
		  AND (end_date IS NULL OR end_date >= CURRENT_DATE)
		ORDER BY start_date DESC
		LIMIT 1
	`).Scan(&id)
	if err != nil {
		return ""
	}
	return id
}

// staticTokenProvider fournit les tokens Halo depuis le token store.
type staticTokenProvider struct {
	tokens auth.StoredTokens
}

func (s *staticTokenProvider) GetTokens(_ context.Context) (*domain.HaloTokens, error) {
	return &domain.HaloTokens{
		SpartanToken:   s.tokens.AccessToken,
		ClearanceToken: "",
	}, nil
}

// resolveXUIDForRotation retourne le xuid associé à un gamertag pour l'écriture
// du RT rotaté dans MultiUserTokenStore. Délègue à capturecli.ResolveXUIDForRotation
// (helper testable sans cgo). Best-effort : LoadPlayers échoué = "" silencieux.
func resolveXUIDForRotation(ctx context.Context, cfg *config.AppConfig, store *auth.MultiUserTokenStore, gamertag string) string {
	players, err := cfg.LoadPlayers()
	if err != nil {
		slog.DebugContext(ctx, "resolveXUIDForRotation: LoadPlayers erreur", "err", err)
		players = nil
	}
	return capturecli.ResolveXUIDForRotation(ctx, store, players, gamertag)
}
