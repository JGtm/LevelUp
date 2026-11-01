// boot_tasks.go — tâches de fond du démarrage : suivi des goroutines, pool de jetons, scheduler
// d'auto-sync, tâches périodiques, boucles des planificateurs. Les crons et le watcher vivent
// dans boot_crons.go et boot_watcher.go.

package main

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/adminstate"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/platform/auth/tokencache"
	"levelup/go-api/internal/scheduler"
	"levelup/go-api/internal/watcher"
)

// initBootTasks crée le suivi des tâches de fond puis rejoue les migrations de données du boot
// (amis par joueur, groupe par défaut).
func initBootTasks(st *bootState) {
	ctx, cfg, friendStore, groupStore := st.ctx, st.cfg, st.friendStore, st.groupStore
	// Tâches de fond et étapes de boot (cf. background_tasks.go) : en démo, celles qui
	// écrivent ou lisent comme état hors de la racine démo sont coupées (lot B5, D-7).
	// schedulerWG track les goroutines « suivies » pour que le shutdown les attende
	// (sur ctx.Done()) avant duckdb.CloseAll() — sans ce wait, un cycle RunOnce en cours
	// peut encore toucher metaDB après la fermeture.
	boot := newBootTasks(cfg.DemoMode, &st.schedulerWG)

	// Migrations boot-time (amis par joueur, puis groupe par défaut « Mon foyer ») —
	// idempotentes, ordre imposé ; cf. runBootMigrations.
	runBootMigrations(ctx, boot, cfg, friendStore, groupStore)
	st.boot = boot
}

// initAccountsPool construit le pool de tokens de l'auto-sync (nil si aucun credential).
func initAccountsPool(st *bootState) {
	ctx, cfg, gate, boot, tokenProvider := st.ctx, st.cfg, st.gate, st.boot, st.tokenProvider

	// ADR 0023 Phase 2 — Migration boot-time des tokens legacy vers MultiUserTokenStore.
	// Discovery + Resolver + Pool : tous les appels API Halo passent par là.
	// - Discovery scanne le MultiUserTokenStore (source unique ADR 0023).
	// - Resolver échange CredentialSource → ResolvedTokens (Spartan+Clearance)
	//   avec cache TTL ~3h30.
	// - Pool maintient les tokens vivants, gère 429/503 cooldown, et permet
	//   le round-robin (PolicyAnyPublic) ou pinned (PolicyPinnedPlayer).
	//
	// Le callback onRotated persiste le refresh_token rotaté par Microsoft dans le
	// MultiUserTokenStore — sans ça, le prochain refresh échouerait avec
	// invalid_grant (Microsoft rotate systématiquement le RT à chaque usage).
	gate.setStep(ctx, bootStepAccounts)
	var autoSyncPool pool.Pool
	if !boot.cutInDemo(stepTokenPool) {
		autoSyncPool = buildAutoSyncPool(ctx, cfg, tokenProvider)
	}
	gate.setStep(ctx, bootStepServices)
	if autoSyncPool != nil {
		slog.Info("auto_sync: pool initialisé", "size", autoSyncPool.Size())
	} else {
		slog.Warn("auto_sync: pool non initialisé — aucun credential découvert",
			"hint", "authentifier un joueur (SSO Xbox) ou lancer `go run ./cmd/token-capture/ <GT>`",
		)
	}
	st.autoSyncPool = autoSyncPool
}

// initScheduler crée le scheduler d'auto-sync, son contexte annulable et l'état admin
// persistant (instantané post-sync, journal des actions).
func initScheduler(st *bootState) {
	ctx, cfg, settingsStore, tokenProvider, autoSyncPool, friendStore := st.ctx, st.cfg, st.settingsStore, st.tokenProvider, st.autoSyncPool, st.friendStore

	autoScheduler := scheduler.New(cfg, settingsStore, tokenProvider, autoSyncPool).
		WithFriendStore(friendStore)
	schedulerCtx, cancelScheduler := context.WithCancel(ctx)

	// Lot C1/C2 — persistance JSON légère (HORS DuckDB) de l'état runtime admin,
	// pour guérir l'amnésie du dashboard au reboot : snapshot post-sync (timeline +
	// matrice + horodatage) réhydraté au boot (C1), et journal des actions globales
	// (dernière exécution / issue / déclencheur) partagé scheduler↔services (C2).
	// Chemins via le PathResolver (data/global/admin_state/*.json). Best-effort :
	// un échec de lecture/écriture est loggé et l'API sert l'état mémoire.
	// Démo : sous `<démo>/runtime/` (cfg.RuntimePaths, lot B5.5).
	runtimePaths := cfg.RuntimePaths()
	postSyncStore := adminstate.NewFileStore(runtimePaths.PostSyncSnapshotPath())
	adminActionJournal := adminstate.NewActionJournal(adminstate.NewFileStore(runtimePaths.ActionJournalPath()))
	if err := adminActionJournal.Load(ctx); err != nil {
		slog.ErrorContext(ctx, "adminstate: chargement du journal des actions échoué — démarrage à vide", "err", err)
	}
	autoScheduler.WithSnapshotStore(postSyncStore).WithActionJournal(adminActionJournal)
	autoScheduler.RehydrateFromDisk(ctx)
	st.autoScheduler, st.schedulerCtx, st.cancelScheduler, st.adminActionJournal = autoScheduler, schedulerCtx, cancelScheduler, adminActionJournal
}

// launchCoreTasks lance le janitor, la reprise des WAL, le CHECKPOINT social et le re-scan du pool.
func launchCoreTasks(st *bootState) {
	ctx, cfg, pr, boot, schedulerCtx := st.ctx, st.cfg, st.pr, st.boot, st.schedulerCtx
	autoBatchQueue, autoSyncPool, sharedSocialPath := st.autoBatchQueue, st.autoSyncPool, st.dbPaths.sharedSocial

	// Phase 4.7 closure : janitor périodique (1× / 24h) qui purge :
	//   - data/sync_cache/sync.RunDelta_* > 7 jours (fetch cache éphémère)
	//   - data/wal/*.json ACKés > 7 jours (résidus si BatchQueue active)
	// Best-effort, non-bloquant sur erreur.
	boot.launch(janitorTask(schedulerCtx, pr, autoBatchQueue))

	// Recovery périodique des WAL pending (PLAN_PERSIST_ROBUSTNESS Phase 1).
	// Avant : RecoverPending n'était appelé qu'au boot → un batch échoué
	// (DB busy au moment du persist) restait bloqué jusqu'au prochain reboot,
	// sur une webapp qui tourne des semaines. Désormais re-soumis toutes les
	// 10 min. Le dédup inFlight de la queue évite de re-pousser un batch déjà
	// en vol (seuls les WAL réellement bloqués sont rejoués).
	if autoBatchQueue != nil {
		boot.launch(walRecoveryTask(schedulerCtx, autoBatchQueue))
	}

	// CHECKPOINT périodique shared_social — vide le WAL toutes les 5 min sans
	// bloquer les writes per-opération.
	//
	// Pourquoi pas un CHECKPOINT systématique dans Persist() ? La connexion
	// shared_social est ouverte MaxOpenConns(4) (OpenReadWriteShared) — donc PAS
	// "une seule connexion" (revue 2026-06-01 SS-3 : l'ancien commentaire
	// MaxOpenConns(1) était faux). Le choix est de borner la fenêtre WAL à 5 min
	// via ce scheduler + le CHECKPOINT shutdown, plutôt que de checkpointer après
	// chaque commit (coût 100–500 ms par write user-facing). Les écritures
	// critiques (likes/favoris/associations média) peuvent flush immédiat via
	// CommitWithCheckpoint.
	//
	// Sérialisation : le commit-lock single-writer de DuckDB + le lease
	// KindSharedSocial sérialisent déjà les écritures ; ce tick réutilise le même
	// handle caché (LookupCachedDB) sans contention notable.
	//
	// LookupCachedDB : pas d'ouverture propre — on réutilise la connexion du
	// pool process-wide (même *sql.DB que SharedSocialPersister). Si le premier
	// joueur n'a pas encore été chargé, on skip le tick silencieusement.
	boot.launch(socialCheckpointTask(ctx, sharedSocialPath))

	// Phase 4.9 / PLAN_AUTH E.v2 (2026-05-24) : periodic Discovery re-scan
	// pour hot-add nouveaux tokens (nouveau SSO / token-capture écrivant dans
	// watcher_tokens/) sans reboot. Skip si pool nil (aucun credential au boot).
	if autoSyncPool != nil {
		boot.launch(poolRescanTask(schedulerCtx, cfg, autoSyncPool))
	}
}

// launchSchedulers lance l'auto-sync et le contrôle de santé des données, crée le planificateur
// de sauvegarde et la vue interface du daemon de présence.
func launchSchedulers(st *bootState) {
	cfg, pr, boot, schedulerCtx := st.cfg, st.pr, st.boot, st.schedulerCtx
	autoScheduler, watcherDaemon := st.autoScheduler, st.watcherDaemon

	// Scheduler d'auto-sync (goroutine suivie par schedulerWG).
	boot.launch(loopTask(taskAutoSync, true, schedulerCtx, autoScheduler.Run))

	// 2026-05-08 — Data health scheduler : audit périodique multi-DB
	// (UUIDs résiduels, bits menteurs, garbage URLs). Depuis 2026-05-20 les
	// compteurs sont uniquement loggés dans `logs/scheduler.log` (le diag
	// admin se fait via `cmd/diag_db_health` ou la lecture des logs) — plus
	// d'émission de notif `data_health_warning` (jargon dev sans intérêt
	// pour un end user lambda sur une app de stats).
	healthScheduler := scheduler.NewDataHealthScheduler(cfg.RepoRoot).WithSharedReader(healthSharedReader(cfg, pr))
	boot.launch(loopTask(taskDataHealth, true, schedulerCtx, healthScheduler.Run))

	// Backup restic des bases DuckDB (pkg/duckdbbackup).
	// La PLANIFICATION est externe (systemd timers côté serveur, cf.
	// scripts/systemd/levelup-restic-backup.timer) : l'application n'embarque plus
	// de planificateur périodique. Le scheduler est créé pour exposer le statut du
	// dernier backup dans l'UI settings, le déclenchement manuel (POST
	// /settings/backup/run) et la fraîcheur (admin monitoring).
	backupSched := ops.NewLevelUpBackupScheduler(cfg.Backup, pr)

	// Convertir en interface (nil safe : un *Daemon nil ne doit pas devenir une interface non-nil)
	var watcherCtrl watcher.DaemonController
	if watcherDaemon != nil {
		watcherCtrl = watcherDaemon
	}
	st.healthScheduler, st.backupSched, st.watcherCtrl = healthScheduler, backupSched, watcherCtrl
}

// buildAutoSyncPool construit le pool de tokens utilisé par l'AutoSyncScheduler.
// Pipeline :
//  1. Discovery scanne le MultiUserTokenStore (source unique ADR 0023) → []CredentialSource
//  2. Resolver échange ces sources en tokens Halo (cache TTL ~3h30) + callback
//     onRotated qui persiste le RT rotaté par Microsoft dans le store. Sans cette
//     persistance, le prochain refresh OAuth échouerait avec invalid_grant
//     (Microsoft rotate systématiquement le RT à chaque usage).
//  3. NewPool maintient les tokens vivants, gère cooldown 429/503, et expose
//     un Acquire round-robin + pinned.
//
// Retourne nil (et un log Warn) si aucun credential n'est trouvé — le
// scheduler tournera quand même mais tous les joueurs seront skipped avec
// raison explicite.
func buildAutoSyncPool(
	ctx context.Context,
	cfg *config.AppConfig,
	tokenProvider auth.TokenProvider,
) pool.Pool {
	pr := title.NewPathResolver(cfg.RepoRoot)
	multiUserStore := auth.NewMultiUserTokenStore(cfg.WatcherTokensDir())
	discovery := pool.NewDiscoveryWithStore(cfg, pr, title.DefaultSlug, multiUserStore)
	sources, err := discovery.Scan(ctx)
	if err != nil {
		slog.Error("auto_sync: pool discovery échoué", "err", err)
		return nil
	}
	if len(sources) == 0 {
		return nil // log Warn fait par le caller
	}

	resolver := pool.NewResolverWithCallbacks(tokenProvider, 0, poolCallbacks(cfg, multiUserStore)) // 0 = default TTL ~3h30
	p, err := pool.NewPool(ctx, resolver, sources, pool.PoolOptions{
		MaxSize:     0, // 0 = tous les sources découverts
		PerTokenRPS: 5, // Option 2 audit 2026-05-21 : aligné sur le sync manuel
		//                 (validé bench-rps real-multi : 0 × 429 à 5N RPS)
	})
	if err != nil {
		slog.Error("auto_sync: pool creation échouée", "err", err)
		return nil
	}
	return p
}

// poolCallbacks réunit les rappels du resolver : rotation du refresh token, transitions reauth,
// échecs OAuth permanents et provenance du préfixe RpsTicket.
func poolCallbacks(cfg *config.AppConfig, multiUserStore *auth.MultiUserTokenStore) pool.ResolverCallbacks {
	// Callback de persistance du RT rotaté (ADR 0023) : écriture dans le
	// MultiUserTokenStore, source unique (le double-write sync_meta a été retiré
	// en Phase 5, 2026-08-25).
	//
	// xuid résolu via store.LoadByGamertag (l'entrée a été créée par migration
	// Phase 2 ou par Discovery) — fallback config.LoadPlayers si pas en store.
	onRotated := func(ctx context.Context, gamertag, newRT string) error {
		xuid := resolveXUIDForRotation(ctx, cfg, multiUserStore, gamertag)
		if xuid == "" {
			return fmt.Errorf("onRotated: xuid introuvable pour %s — RT rotaté non persisté", gamertag)
		}
		if err := multiUserStore.UpdateOAuthRefreshToken(xuid, newRT); err != nil {
			return fmt.Errorf("onRotated: écriture store (%s): %w", gamertag, err)
		}
		// Le RT a tourné → la chaîne d'auth a changé : le cache process des
		// HaloTokens (Spartan/Clearance, TTL 50min) peut désormais servir des
		// tokens dérivés de l'ANCIENNE chaîne → 401 sur les fetchs live
		// token-gated (career rank/XP, challenges, battle pass, CSR, identité
		// Explorer). On le purge pour forcer une re-dérivation fraîche au
		// prochain enrichWithHaloTokens. Incident 2026-06-14 (post-consolidation
		// client ID : seuls les CLI token-capture/import invalidaient ce cache).
		tokencache.InvalidateCachedPlayerTokens(xuid)
		return nil
	}

	// Persistance des transitions reauth + du dernier échec OAuth permanent
	// (dashboard admin « Santé des tokens », plan anti-bruit 2026-06-11).
	// Best-effort : un échec d'écriture store ne casse pas le resolve.
	onReauth := func(ctx context.Context, gamertag, xuid string, required bool) {
		if xuid == "" {
			return
		}
		if required {
			if _, err := multiUserStore.MarkReauthRequired(xuid, gamertag); err != nil {
				slog.WarnContext(ctx, "onReauth: écriture store échouée",
					"gamertag", gamertag, "err", err)
			}
			return
		}
		if err := multiUserStore.ClearReauthRequired(xuid); err != nil {
			slog.WarnContext(ctx, "onReauth: clear store échoué",
				"gamertag", gamertag, "err", err)
		}
	}
	onAuthError := func(ctx context.Context, gamertag, xuid, class, msg string) {
		if xuid == "" {
			return
		}
		var err error
		if class == "" {
			err = multiUserStore.ClearAuthError(xuid)
		} else {
			err = multiUserStore.RecordAuthError(xuid, gamertag, class, msg)
		}
		if err != nil {
			slog.WarnContext(ctx, "onAuthError: écriture store échouée",
				"gamertag", gamertag, "err", err)
		}
	}

	return pool.ResolverCallbacks{
		OnRotated:        onRotated,
		OnReauth:         onReauth,
		OnAuthError:      onAuthError,
		OnFamilyObserved: familyObservedCallback(multiUserStore),
	}
}

// familyObservedCallback persiste la provenance du préfixe RpsTicket mesurée à l'échange XBL.
func familyObservedCallback(multiUserStore *auth.MultiUserTokenStore) func(ctx context.Context, gamertag, xuid, family string) error {
	// Provenance MESURÉE à l'échange XBL : persistée pour que le boot suivant
	// parte du bon préfixe RpsTicket (sinon 401 + retry à chaque échange pour les
	// comptes qui n'acceptent pas « d= »).
	return func(ctx context.Context, gamertag, xuid, family string) error {
		if xuid == "" {
			return nil
		}
		if err := multiUserStore.UpdateTokenClientFamily(xuid, family); err != nil {
			return fmt.Errorf("onFamilyObserved %s: %w", gamertag, err)
		}
		return nil
	}
}
