// boot_router.go — construction du routeur HTTP et câblage du registre de services, hooks de
// fin de démarrage, puis ouverture de la porte (le routeur n'est branché qu'une fois tout câblé).

package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"levelup/go-api/internal/api"
	"levelup/go-api/internal/api/wire"
	"levelup/go-api/internal/notifications/external"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/scheduler"
)

// buildRouter construit le routeur HTTP et branche le registre de services (jetons, journal
// d'actions, planificateur de santé, notifications de rejeux).
func buildRouter(st *bootState) {
	cfg, bootRepo, bootSvc, tokenProvider, groupStore := st.cfg, st.bootRepo, st.bootSvc, st.tokenProvider, st.groupStore
	autoScheduler, watcherDaemon, schedulerCtx, watcherCtrl := st.autoScheduler, st.watcherDaemon, st.schedulerCtx, st.watcherCtrl
	backupSched, autoSyncPool, adminActionJournal, healthScheduler := st.backupSched, st.autoSyncPool, st.adminActionJournal, st.healthScheduler

	// Routeur HTTP — le daemon peut être nil si le watcher est désactivé.
	// reg est assigné ici : la closure notifierGetter y accède de manière lazy (joueur actif après démarrage).
	//
	// routerCtx : on ne dérive les bgCtx des syncs HTTP de schedulerCtx (annulable
	// au shutdown) QUE si un vrai gate est branché (watcher actif) — alors le drain
	// gate.WaitInFlight() au shutdown les attend. Watcher off → context.Background()
	// (comportement legacy exact : NopSyncGate ne tracke rien, pas de WaitInFlight).
	routerCtx := context.Background()
	if watcherDaemon != nil {
		routerCtx = schedulerCtx
	}
	var reg *wire.ServiceRegistry
	var router http.Handler
	// 3e retour = document OpenAPI PARTAGÉ (V72-01 / H1) : non exposé en HTTP à ce
	// stade (DocsPath/OpenAPIPath désactivés), réservé à la génération hors-ligne (H6).
	router, reg, _ = api.NewRouter(routerCtx, cfg, bootRepo, bootSvc, watcherCtrl, tokenProvider, autoScheduler, backupSched, groupStore)
	st.router, st.reg = router, reg

	// Purge périodique des sessions expirées, sur le magasin du routeur. Gardée dans les quatre
	// profils (elle ne touche que le dossier des sessions du profil) ; non suivie : aucune base
	// à fermer derrière elle, la boucle s'arrête avec schedulerCtx.
	st.boot.launch(loopTask(taskSessionPurge, false, schedulerCtx, func(c context.Context) {
		scheduler.RunSessionPurgeLoop(c, reg.SessionStore())
	}))

	// Câble le hook global de re-dérivation des tokens per-player : le cache
	// expiry-aware (tokencache.ResolveFreshPlayerTokens) re-minte via le registry quand un
	// token est périmé. Sans ce câblage, le re-mint singleflighté est inopérant.
	reg.WireGlobalTokenRefresher()

	// A1 : câble le pool de tokens partagé comme source de fallback pour les
	// LECTURES PUBLIQUES de tiers (player-query Explorer). Quand le RT du profil
	// sélectionné est mort, un token SAIN du pool porte quand même les fetchs live
	// de la cible (ADR 0023 : aucune re-capture, lecture publique uniquement).
	// autoSyncPool peut être nil (aucun credential découvert) → no-op.
	reg.WithTokenPool(autoSyncPool)

	// Journal des actions globales (C2) : partagé avec le scheduler (câblé plus
	// haut). Les runners d'action y écrivent leur dernière exécution ; l'endpoint
	// GET /admin/actions/journal le sert (survit au reboot).
	reg.WithActionJournal(adminActionJournal)

	// Dashboard monitoring admin — câblage du HealthScheduler (créé plus haut,
	// avant NewRouter). Les runners monitoring le lisent lazily à chaque
	// requête : l'ordre boot est sûr, et l'overview expose le dernier audit
	// data health + l'action POST /admin/actions/data-health/run.
	reg.WithHealthScheduler(healthScheduler)

	// Notification Discord groupée « rejeux 2D prêts » (lot B v7.5) : câble le puits
	// d'artefacts de internal/replaybuild, par lequel passent la construction locale
	// (post-sync), la livraison d'un ouvrier distant et l'action admin. UN SEUL câblage
	// dans tout le dépôt (garde-rail archlint/no_second_artifact_sink_test.go) : un
	// second REMPLACERAIT celui-ci. Posé ici, donc avant le montage des routes
	// ouvrier et bien avant l'ouverture du port HTTP ; le scheduler d'auto-sync, lui,
	// démarre quelques lignes plus haut — au pire les tout premiers artefacts d'un cycle
	// déjà en cours au boot ne sont pas annoncés, jamais un artefact perdu.
	reg.InstallReplayNotify()
}

// wireSyncPipeline câble le runner post-sync, la file des rejeux, le hook Prestige et
// l'orchestrateur V2 sur le scheduler d'auto-sync.
func wireSyncPipeline(st *bootState) {
	cfg, pr, titleSlug, reg, tokenProvider := st.cfg, st.pr, st.titleSlug, st.reg, st.tokenProvider
	autoScheduler, autoSyncPool, autoBatchQueue, metaDB := st.autoScheduler, st.autoSyncPool, st.autoBatchQueue, st.metaDB
	sharedPath, settingsStore, friendStore := st.dbPaths.shared, st.settingsStore, st.friendStore

	// Phase 4 plan stabilisation 2026-05-22 — câblage post-sync runner sur
	// l'auto-sync scheduler. Avant ce fix, l'auto-sync court-circuitait
	// systématiquement le pipeline progression V2 (streaks/records/milestones/
	// notifications delta). Maintenant les 3 entry points (HTTP, auto-sync,
	// CLI futur) invoquent le MÊME runner via SyncEngine.WithPostSyncRunner.
	// Cf. AUDIT_ASCENSION_PIPELINE_DISCONNECTED_2026-05-21 §4 cause B.
	if postSyncRunner := wire.NewPostSyncRunner(reg); postSyncRunner != nil {
		autoScheduler.WithPostSyncRunner(postSyncRunner)
	}

	// Fil de l'eau des rejeux, chemin « ouvrier » (piste F) : la mise en file vit
	// dans le registre (elle seule a les tokens pour résoudre le manifeste et y
	// déposer les URL pré-signées). Injectée ici, comme le runner post-sync, parce
	// que le scheduler est construit avant le registre.
	autoScheduler.WithReplayEnqueuer(reg.EnqueueReplayBuildJob)

	// VF-1 / DC-4 — câblage du hook Prestige post-sync sur le chemin V1
	// (auto-sync + HTTP delta + watcher, tous via BuildEngine). Avant ce fix,
	// prestige.RunPostSyncHook ne tournait sur AUCUN chemin (stub qui jetait le
	// hook, engine.prestigeHook toujours nil). = PrestigeBundle.RunPostSync
	// (no-op si bundle nil ou flag Prestige off, cf. RunPostSync).
	var prestigePostSyncHook func(ctx context.Context, playerSlug, titleSlug string)
	if pb := reg.PrestigeBundle(); pb != nil {
		prestigePostSyncHook = pb.RunPostSync
		autoScheduler.WithPrestigeHook(prestigePostSyncHook)
	}

	// ADR 0027 — câblage pipeline V2, UNIQUE moteur de sync du cycle depuis la
	// suppression du pipeline V1 (lot D1c, DEC-2). Une fois l'orchestrator câblé
	// (bloc ci-dessous), le cycle auto-sync pilote les joueurs MOTEUR (Infinite)
	// par V2 et les LIVE-ONLY (Halo 5) par syncPlayer→liveRunner. Plus de flag
	// LEVELUP_SYNC_PIPELINE ni de fallback automatique.
	//
	// Pré-requis : autoSyncPool + autoBatchQueue + metaDB non-nil. Si l'un manque,
	// on skip le câblage : le cycle bascule sur le filet structurel syncPlayer de
	// boot (sync directe sans orchestrator).
	if autoSyncPool != nil && autoBatchQueue != nil && metaDB != nil {
		// Récupère le handle shared via le cache duckdb process-wide
		// (déjà ouvert plus tôt par le serveur en RO ou RW selon mode).
		var sharedSQLDB *sql.DB
		if cached, ok := duckdb.LookupCachedDB(sharedPath); ok {
			sharedSQLDB = cached.SQLDb()
		}
		// T1 — parity-complete : passe TOUTES les dépendances que
		// defaultRunnerFactory utilise pour câbler la SyncEngine V1.
		// Garantit que V2 a EXACTEMENT le même runtime que V1 sur le
		// post-sync (sessions, achievements, progression V2, media scan).
		v2PostSyncRunner := wire.NewPostSyncRunner(reg)
		v2Orch := buildSyncV2Orchestrator(SyncV2WiringDeps{
			Cfg:            cfg,
			PathResolver:   pr,
			TitleSlug:      titleSlug,
			TokenPool:      autoSyncPool,
			BatchQueue:     autoBatchQueue,
			MetaDB:         metaDB.SQLDb(),
			SharedDB:       sharedSQLDB,
			TokenProvider:  tokenProvider,
			Settings:       settingsStore,
			Friends:        friendStore,
			PostSyncRunner: v2PostSyncRunner,
			PrestigeHook:   prestigePostSyncHook,
			ReplayEnqueue:  reg.EnqueueReplayBuildJob,
		})
		if v2Orch != nil {
			autoScheduler.WithCycleOrchestrator(v2Orch)
			slog.Info("sync.v2: orchestrator câblé parity-complete (unique moteur de sync ; filet syncPlayer si non câblé)")
		}
	}
}

// wireTitleHooks injecte dans la config les hooks dépendant du registre (titre prêt,
// progression après sync), émet la notification de version et journalise le relais externe.
func wireTitleHooks(st *bootState) {
	cfg, reg, boot, schedulerCtx := st.cfg, st.reg, st.boot, st.schedulerCtx

	// MT-19 / axe E : notifier « titre prêt » injecté dans cfg (lu au runtime par le
	// Runner live h5 via cfg.TitleReadyNotifier). Posé APRÈS NewRouter (reg dispo) et
	// AVANT que les syncs scheduler/watcher tournent (runtime post-boot). Émet une
	// notif quand un titre live a des matchs, dans le flux du titre par défaut, hors
	// pipeline progression/prestige. Même pattern d'injection que cfg.SharedManager.
	cfg.TitleReadyNotifier = wire.BuildTitleReadyNotifier(reg, cfg)
	// Progression V2 title-agnostic : le Runner live d'un titre (Halo 5+) déclenche le
	// pipeline streaks/records/milestones/coach via ce hook après un cycle qui insère
	// des matchs (deps de base, SANS le PrestigeBundle mono-titre). Même pattern d'injection.
	cfg.ProgressionAfterSync = wire.BuildProgressionAfterSyncHook(reg, cfg)

	// app_release : émission asynchrone d'une notification in-app par joueur si la
	// version a changé depuis sync_meta.last_seen_app_version. Ne bloque pas le boot.
	boot.launch(appReleaseTask(schedulerCtx, func(c context.Context) {
		wire.EmitAppReleaseForAllPlayers(c, cfg, reg, cfg.AppVersion)
	}))

	// Relais coach externe (Discord webhook, opt-in) : une ligne d'état au boot,
	// comme les autres sous-systèmes. OFF par défaut (cf. internal/notifications/external).
	external.LogBootState(context.Background(), cfg.AppSettingsPath)
}

// openRouterAndHeartbeat branche le routeur sur la porte de démarrage, lance le battement de
// vie et journalise les tâches coupées en démo.
func openRouterAndHeartbeat(st *bootState) {
	ctx, bootSrv, router, boot, schedulerCtx := st.ctx, st.bootSrv, st.router, st.boot, st.schedulerCtx

	// --- 7. Routeur branché (le port écoute depuis 2 bis) + graceful shutdown ---
	bootSrv.openRouter(ctx, router)

	// Heartbeat 30s — sentinelle de vie du process. Si les logs cessent de
	// montrer "alive" mais que le binaire est toujours en mémoire → deadlock
	// ou hang. Si les logs s'arrêtent ET le process disparaît → crash (cf.
	// logs/server.crash.log + recover() dans post-sync). Diagnostic immédiat
	// au prochain incident type 2026-05-22 (silence total post 18:41:19).
	startedAt := time.Now()
	boot.launch(heartbeatTask(schedulerCtx, startedAt))
	// LE log Info de boot qui liste, en démo, ce qui est coupé (no-op hors démo).
	boot.logCut(ctx)
}
