// boot_crons.go — crons et boucles lancés une fois le routeur construit : monitoring, surveillance
// disque, notifications de rejeux, catalogue, noms d'assets, customisation Spartan, classement
// mondial, purge des artefacts de rejeu.

package main

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/assetnames"
	"levelup/go-api/internal/domain"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/games/halo_5/livesync"
	"levelup/go-api/internal/games/halo_infinite/haloadapter"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/scheduler"
	"levelup/go-api/internal/worldenrich"
)

// launchMonitoring ouvre le store monitoring puis lance les boucles de détection, de surveillance
// disque, de notification des rejeux, de relance des citations, d'alignement des médias et de rejeu des KPI.
func launchMonitoring(st *bootState) {
	ctx, cfg, pr, boot, schedulerCtx, reg := st.ctx, st.cfg, st.pr, st.boot, st.schedulerCtx, st.reg

	// Store monitoring persistant (base globale data/global/monitoring.duckdb) —
	// survit au restart : détections avec cycle de vie, historique crons, dernier
	// audit data-health. Best-effort : un échec d'ouverture dégrade les sections
	// (jamais fatal). Le flush périodique du delta ErrorCollector tourne sur
	// schedulerCtx/schedulerWG (drainé AVANT duckdb.CloseAll — écriture sûre). En démo :
	// base EN MÉMOIRE, rien de persisté (openMonitoringStore, lot B5.3).
	if monStore, mErr := openMonitoringStore(ctx, cfg, pr); mErr != nil {
		slog.Warn("monitoring store: ouverture échouée — sections détections dégradées", "err", mErr)
	} else {
		reg.WithMonitoringStore(monStore)
		// Marqueur de démarrage (A5.1) : compteur de restarts persistant
		// (COUNT(server_boot) dans cron_runs — exposé par /monitoring/resources).
		if err := monStore.RecordCronRun(ctx, "server_boot", time.Now(), true, "", 0); err != nil {
			slog.Warn("monitoring store: marqueur server_boot non écrit", "err", err)
		}
		// Statut unifié des crons (A6/DC-5) : chaque ReportCronRun est relayé
		// vers cron_runs (persistance — l'historique survit au restart). Les
		// crons tournent sur schedulerCtx, drainé AVANT duckdb.CloseAll.
		observability.SetCronRunSink(func(name string, startedAt time.Time, ok bool, errStr string, durationMs int64) {
			if err := monStore.RecordCronRun(context.Background(), name, startedAt, ok, errStr, durationMs); err != nil {
				slog.Warn("monitoring store: cron run non persisté", "cron", name, "err", err)
			}
		})
		boot.launch(loopTask(taskDetectionFlush, true, schedulerCtx, reg.RunDetectionFlushLoop))
	}

	// Surveillance disque du volume data (lot ops 2026-07-13, suite incident
	// disque-plein VPS) : dépassement de seuil (A5.3 : 80 %/90 % ou 2 Go/500 Mo)
	// → log WARN/ERROR (détection persistée + badge admin) + notification Discord
	// si webhook configuré. Indépendante du monitoring store (log-driven).
	boot.launch(loopTask(taskDiskWatch, true, schedulerCtx, reg.RunDiskWatchLoop))

	// Flush des lots de rejeux prêts (lot B v7.5) : à chaque tick, les fenêtres de
	// groupement échues sortent en UN message chacune. Sur schedulerCtx/schedulerWG comme
	// la surveillance disque — la boucle ne fait que des lectures shared courtes, drainées
	// avant duckdb.CloseAll.
	boot.launch(loopTask(taskReplayNotify, true, schedulerCtx, reg.RunReplayNotifyLoop))
	// Citations du film recalculées dès la livraison d'un rejeu (délai du débrief).
	boot.launch(citationsRelanceTask(schedulerCtx, reg.NewCitationsRelance()))
	// Alignement des médias sur le dossier des captures : une passe, joueur par joueur, pour ce que
	// l'explorateur a changé pendant que l'application était éteinte.
	boot.launch(mediaReconcileTask(schedulerCtx, cfg, st.settingsStore))
	// Rejeu des KPI quand une formule a changé depuis le dernier démarrage : une passe, joueur par
	// joueur, sans effet quand la révision rangée est celle du binaire.
	boot.launch(scoringRevisionTask(schedulerCtx, reg.NewScoringRevision()))
}

// launchCatalogCrons lance le cron catalogue hebdomadaire et le balayage des noms d'assets.
func launchCatalogCrons(st *bootState) {
	ctx, boot, schedulerCtx, reg, autoSyncPool := st.ctx, st.boot, st.schedulerCtx, st.reg, st.autoSyncPool

	// Cron catalogue (hebdomadaire) : rafraîchit le catalogue (playlists / couples
	// map-mode / maps / modes) via le drain DiscoveryUGC testé (même chemin que l'action
	// admin catalog/ugc-drain). TOUJOURS actif (autonome, plus de flag). La
	// régularité vient du ticker hebdo, PAS d'un redémarrage.
	//
	// ART-safety (2026-06-19) : l'UPSERT vers les tables catalogue est SELECT-then-write
	// (upsertRowNoConflict), MAIS la queue catalog_fetch_queue était elle-même DELETE +
	// UPDATE per-row sur index ART (PK + idx secondaire) → corrompait metadata.duckdb à
	// chaque boot. Corrigé : la queue n'a plus aucune surface ART (cf. migration
	// rebuild_catalog_fetch_queue_drop_art_indexes + dédup NOT EXISTS à l'enqueue).
	catalogCron := scheduler.NewCatalogRefreshCron(func(cctx context.Context, ts string) (domain.CatalogUGCDrainResult, error) {
		// V2 — découverte « A à Z » : avant le drain, lire la config de chaque
		// playlist (discovery-infiniteugc) pour enfiler ses couples map-mode enfants
		// (même jamais joués) + stocker les poids. Best-effort : un échec d'expansion
		// n'empêche pas le drain de nommer ce qui a été joué.
		if n, eerr := reg.ExpandPlaylistChildren(cctx, ts); eerr != nil {
			slog.WarnContext(cctx, "catalog_refresh_cron: expansion playlists échouée (best-effort)", "module", logging.ModuleCatalog, "err", eerr)
		} else {
			slog.InfoContext(cctx, "catalog_refresh_cron: playlists expansées", "module", logging.ModuleCatalog, "children_enqueued", n)
		}
		// NB : le balayage des NOMS d'assets (asset_translations) vit dans
		// asset_name_sweep_cron (découplé, gaté LEVELUP_SYNC_RESOLVE_ASSETS).
		return reg.RunCatalogUGCDrain(cctx, ts)
	}, "", 0).
		// Gate RÉEL (remplace le proxy CapForge) : ne draine QUE les titres dont le
		// catalog adapter discovery-infiniteugc est résolvable (rules TOML +
		// halo_infinite.NewCatalogAdapter). HINF → adapter présent → run ; Halo 5 →
		// pas d'experience_rules.toml → skip propre. Comportement prod identique au
		// proxy, signal précis.
		WithCatalogAdapterCheck(reg.HasCatalogAdapter)
	if boot.launch(loopTask(taskCatalogCron, true, schedulerCtx, catalogCron.Run)) {
		slog.InfoContext(ctx, "catalog_refresh_cron: scheduled", "module", logging.ModuleCatalog, "interval", scheduler.DefaultCatalogRefreshInterval)
	}

	// Balayage de noms d'assets (filet de rattrapage de la traîne), distinct du cron
	// catalogue : ART-safe (asset_translations via ops.UpsertAssetTranslation), gaté par
	// LEVELUP_SYNC_RESOLVE_ASSETS (ON par défaut, kill-switch). 1er passage ~60s après le
	// boot (le temps que le pool de tokens se réchauffe), puis hebdomadaire. La résolution
	// PRIMAIRE des noms reste in-sync ; ce cron ne rattrape que les assets jamais rejoués.
	if haloadapter.AssetNameResolutionEnabled() {
		sweepCron := scheduler.NewAssetNameSweepCron(func(cctx context.Context, ts string) (assetnames.Result, error) {
			return reg.ResolveUnresolvedAssetNames(cctx, ts, autoSyncPool)
		}, "", 0).
			// Même gate RÉEL que le drain catalogue : le sweep de noms passe par le
			// même fetcher /hi/ hardcodé, donc même critère (catalog adapter résolvable).
			WithCatalogAdapterCheck(reg.HasCatalogAdapter)
		if boot.launch(loopTask(taskAssetNameSweep, true, schedulerCtx, sweepCron.Run)) {
			slog.InfoContext(ctx, "asset_name_sweep_cron: scheduled", "module", logging.ModuleSync, "interval", scheduler.DefaultAssetNameSweepInterval)
		}
	}
}

// launchSpartanCron lance le rafraîchissement périodique de la customisation Spartan.
func launchSpartanCron(st *bootState) {
	ctx, cfg, pr, titleSlug, us := st.ctx, st.cfg, st.pr, st.titleSlug, st.us
	boot, schedulerCtx, reg, autoSyncPool := st.boot, st.schedulerCtx, st.reg, st.autoSyncPool

	// PLAN_V2 Phase 8 (2026-05-26) : SpartanCustomizationCron tourne toutes
	// les 8h (DefaultSpartanCustomizationInterval) pour rafraîchir la
	// customisation Spartan de TOUS les joueurs configurés, indépendamment
	// de l'usage UI. Réutilise CareerLiveService.GetSpartanIdentityFor(p.XUID)
	// (même path que la visite home) → kickoffBackgroundRefresh → persistPartial
	// field-aware. Garantit qu'un joueur qui n'ouvre jamais l'app a quand
	// même sa customisation populée en DB, y compris quand ses propres jetons
	// sont morts (lecture par le token du compte admin, cf. acquireReaderToken).
	if autoSyncPool != nil && reg != nil {
		// Provider qui adapte la signature ServiceRegistry.CareerLiveCtx vers
		// celle attendue par le cron (retourne uniquement le SpartanIdentityFetcher).
		provider := func(ctx context.Context, slug string) (scheduler.SpartanIdentityFetcher, error) {
			svc, _, err := reg.CareerLiveCtx(ctx, slug)
			if err != nil {
				return nil, err
			}
			return svc, nil
		}
		// Comptes de l'instance (store partagé `us`) : un joueur aux jetons morts est lu
		// avec le token du compte de rôle admin, celui de l'utilisateur, et d'aucun autre.
		spartanCron := scheduler.NewSpartanCustomizationCron(
			cfg, autoSyncPool, provider, titleSlug, 0,
		).WithAccounts(us)
		// Title-aware (refactor h5-capability-unification) : enregistre le refresher
		// de customisation des AUTRES titres (Halo 5+). Le scheduler n'importe AUCUN
		// package de titre — c'est ICI (boot, qui importe déjà halo5/livesync) que la
		// closure title-spécifique est injectée. halo_5 → livesync.PersistAppearance
		// (fetch /h5/profiles/{gt}/{appearance,spartan,emblem} + persist service tag /
		// rendu Spartan / emblème dans career_progression h5, append-only). Le ctx
		// porte le token retenu par le cron (celui du joueur, sinon celui du compte admin
		// de l'instance, jamais un autre) → NewAppearanceSource le lit ; les profils h5 se lisent par
		// gamertag, la ligne va dans la base de p.
		// Best-effort : un échec source/fetch est remonté en err (loggé par le cron).
		spartanCron.WithRefresher(halo5.TitleSlug, func(rctx context.Context, p domain.PlayerSummary) error {
			src, err := halo5.NewAppearanceSource(rctx)
			if err != nil {
				return err
			}
			playerDBPath := pr.PlayerDBPath(halo5.TitleSlug, p.Gamertag)
			cacheRoot := pr.CacheRootDir()
			_, err = livesync.PersistAppearance(rctx, src, playerDBPath, cacheRoot, p.Gamertag, p.XUID)
			return err
		})
		if boot.launch(loopTask(taskSpartanCron, true, schedulerCtx, spartanCron.Run)) {
			slog.InfoContext(ctx, "spartan_cron: scheduled",
				"interval", scheduler.DefaultSpartanCustomizationInterval)
		}
	}
}

// launchWorldLeaderboardCron lance la capture quotidienne du classement CSR mondial.
func launchWorldLeaderboardCron(st *bootState) {
	ctx, cfg, boot, schedulerCtx := st.ctx, st.cfg, st.boot, st.schedulerCtx

	// Cron classement CSR mondial : capture quotidienne du leaderboard scrapé
	// depuis Halo Waypoint (saison active découverte automatiquement), persistée
	// en append-only via le SharedDBProvider (AcquireWriter, vidange expirée
	// retentée). Premier tick 2 min après le boot → peuple la prod sans manip
	// manuelle ; garde-fou fraîcheur 20h → pas de
	// re-scrape à chaque hot-reload. Gardé par cfg.SharedProvider != nil : en mode
	// legacy (RO direct), une 2e connexion RW sur shared entrerait en conflit
	// "different configuration" — on désactive donc le cron in-process (le CLI
	// snapshot-world-leaderboard reste la voie manuelle, serveur arrêté).
	if cfg.SharedProvider != nil {
		lbScraper := haloadapter.NewLeaderboardScraper(800 * time.Millisecond)
		worldLbCron := scheduler.NewWorldLeaderboardCron(cfg.SharedProvider, lbScraper, 0)
		worldLbTask := loopTask(taskWorldLeaderbd, true, schedulerCtx, func(sctx context.Context) {
			// Enrichissement des stats joueur (ranked-only, dédup match-centric,
			// append-only) : TOUJOURS actif, via le POOL multi-token (comptes db_profiles
			// round-robin — même chemin d'auth que le reste de l'app et que le CLI
			// -all-tokens, store-first ADR 0023). Construit ICI, dans la goroutine (hors
			// chemin de démarrage → boot non bloqué), en Eager:true : résout les tokens et
			// ÉLIMINE les comptes au RT mort (invalid_grant). Indispensable — sinon le
			// round-robin taperait des sources mortes et raterait des fetchs. Un build en
			// échec (aucun compte résolu) dégrade en scrape-only sans paniquer.
			enr, gts, eerr := worldenrich.BuildEnricher(sctx, cfg, worldenrich.EnricherOptions{
				RPS:   5,
				Eager: true, // filtre les comptes db_profiles dont le RT ne résout pas
			})
			switch {
			case sctx.Err() != nil:
				return // arrêt demandé pendant la résolution des comptes : rien à planifier
			case eerr != nil:
				slog.WarnContext(ctx, "world_leaderboard_cron: enricher non construit — cron en scrape-only",
					"err", eerr)
			default:
				worldLbCron.WithStatsEnricher(enr)
				slog.InfoContext(ctx, "world_leaderboard_cron: enrichissement actif (pool multi-token)",
					"token_accounts", gts)
			}
			worldLbCron.Run(sctx)
		})
		if boot.launch(worldLbTask) {
			slog.InfoContext(ctx, "world_leaderboard_cron: scheduled",
				"interval", scheduler.DefaultWorldLeaderboardInterval)
		}
	}
}

// launchReplayPurgeCron lance la purge récurrente des artefacts de rejeu hors fenêtre de rétention.
func launchReplayPurgeCron(st *bootState) {
	ctx, cfg, settingsStore, boot, schedulerCtx := st.ctx, st.cfg, st.settingsStore, st.boot, st.schedulerCtx

	// Purge récurrente des artefacts de rejeu 2D hors fenêtre replay_retention_months
	// (lot 6 v7.5). SUPPRIME DES ARTEFACTS, JAMAIS DES FILMS. La fenêtre est relue à
	// chaque tick (0 = illimité → tick no-op rapporté en succès). Visible sur
	// /admin/monitoring/crons (ReportCronRun).
	if settingsStore != nil {
		replayPurgeCron := scheduler.NewReplayPurgeCron(cfg.RepoRoot, func() int {
			s, err := settingsStore.Load()
			if err != nil {
				// LOGUE AVANT DE DÉGRADER (règle n°3) : sans cette ligne, un
				// app_settings.json illisible faisait passer la fenêtre à 0 EN SILENCE —
				// c'est-à-dire rétention illimitée, donc purge désactivée sans un mot.
				slog.WarnContext(ctx, "replay_purge_cron: settings illisibles, "+
					"rétention illimitée (purge désactivée)", "err", err)
				return 0
			}
			if s != nil {
				return s.ReplayRetentionMonths
			}
			return 0
		}, 0)
		if boot.launch(loopTask(taskReplayPurge, true, schedulerCtx, replayPurgeCron.Run)) {
			slog.InfoContext(ctx, "replay_purge_cron: scheduled",
				"interval", scheduler.DefaultReplayPurgeInterval)
		}
	}
}
