// Package scheduler — auto_sync_engine.go : factory du moteur de sync (BuildEngine, la
// source unique de wiring engine) + résolution des runners (default / live-title) et du
// slug de titre. Extrait de auto_sync.go (K2c, 2026-07-06) pour scinder le god-file
// (scheduler / factory engine / métriques-convergence).
package scheduler

import (
	"context"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_5/livesync"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/sync"
)

// BuildEngine construit un *sync.SyncEngine fully-configured pour un
// (gamertag, xuid) donné. C'est l'UNIQUE source of truth du wiring engine
// pour le serveur — utilisée à la fois par defaultRunnerFactory (path
// scheduler / auto_sync) et par cmd/server/main.go pour câbler le path
// watcher via syncTrigger.WithEngineFactory.
//
// Toute évolution du wiring (nouveau hook, nouveau provider, etc.) ne doit
// se faire QUE dans cette méthode — c'est ce qui garantit que watcher path
// et scheduler path restent en parité runtime. Cf. incident 2026-05-26 :
// trigger.go faisait un NewSyncEngine direct sans WithSharedProvider, le
// path watcher tombait en legacy → conflit "different configuration".
//
// Sémantique des nils :
//   - s.cfg.SharedProvider == nil → engine en mode legacy (OpenSharedDB direct)
//   - s.settings == nil → pas de FriendsLoader, pas de MediaScanHook
//   - s.pool == nil → pas de PooledHaloClient (le moteur tombera back sur le
//     client default si pas .SetCustomClient — non-recommandé en prod)
//   - s.postSyncRunner == nil → post-sync runner V1 désactivé
//   - s.batchQueue == nil → batch mode synchrone (sans WAL durable)
func (s *AutoSyncScheduler) BuildEngine(ctx context.Context, gamertag, xuid string) *sync.SyncEngine {
	// MT-11 / PMT-3 : le titre du joueur est porté par le ctx (posé par les 3
	// points d'entrée : scheduler syncPlayer, watcher/HTTP). BuildEngine reste
	// l'UNIQUE funnel de construction (utilisé comme valeur de fonction par
	// WithEngineFactory/WithEngineBuilder + appel direct) ; sa signature ne change
	// donc PAS — le slug transite par ctxkeys (carrier titre établi par PMT-10),
	// pas par un param qui rippler­ait EngineFactory/SyncRunner/Coordinator.
	// Fallback DefaultSlug si le ctx ne porte pas de titre → byte-identique halo.
	titleSlug := ctxkeys.TitleSlug(ctx)
	engine := sync.NewSyncEngineForTitle(s.cfg.RepoRoot, titleSlug, gamertag, xuid, &domain.HaloTokens{}, s.provider)
	// Commit 8i : en mode B-swap (LEVELUP_USE_SHARED_PROVIDER=1), router les
	// ouvertures RW de shared via Provider.AcquireWriter au lieu d'OpenSharedDB
	// direct. Coordonne avec le pool joueur via Subscribe (DETACH/REATTACH).
	if s.cfg.SharedProvider != nil {
		engine.WithSharedProvider(s.cfg.SharedProvider)
	}
	if s.settings != nil {
		engine.WithFriendsLoader(func() ([]string, error) {
			cfg, lerr := s.settings.Load()
			if lerr != nil {
				return nil, lerr
			}
			return cfg.FriendGamertags, nil
		})
	}
	if s.pool != nil {
		pooledClient := sync.NewPooledHaloClient(s.pool, gamertag, xuid, 0) // 0 = defaultPooledRPS
		engine.SetCustomClient(pooledClient)
	}
	// Phase 4 plan stabilisation 2026-05-22 : injecter le runner post-sync
	// (notifications delta + pipeline progression V2). Sans ça, auto-sync
	// sautait TOUT le pipeline → page Ascension vide + notifications muettes.
	// gamertag = PlayerSlug (cf. config.go:284 — PlayerSlug = gamertag dans
	// db_profiles.json).
	if s.postSyncRunner != nil {
		engine.WithPostSyncRunner(s.postSyncRunner, gamertag)
	}
	// Media scan post-sync : indexe les captures présentes sur disque et les
	// associe aux matchs fraîchement synchronisés. Les 2 closures lisent les
	// settings live à chaque tick : MediaCapturesBaseDir + UserTimezone.
	// Sans timezone, parseCaptureTimeFromFilename retourne nil → 0 associations
	// (bug observé 2026-05-25, cf. thought_log).
	if s.settings != nil {
		engine.WithMediaScanHook(service.BuildMediaScanHook(s.cfg.RepoRoot, gamertag,
			func() string {
				cfg, _ := s.settings.Load()
				if cfg != nil {
					return cfg.MediaCapturesBaseDir
				}
				return ""
			},
			func() string {
				cfg, _ := s.settings.Load()
				if cfg != nil {
					return cfg.UserTimezone
				}
				return ""
			},
		))
	}
	// Le batch INSERT-only est le seul chemin d'écriture depuis D1b. Phase 4.9 :
	// le layer async reste optionnel. Set LEVELUP_PERSIST_BATCH_ASYNC=0 pour un
	// fallback synchrone direct Persister. La queue est injectée par main.go au
	// boot si activé (cf. autoBatchQueue init — cycle de vie du kill-switch).
	// Valeur résolue au boot dans config.AppConfig (source unique — CR A6).
	if s.batchQueue != nil && s.cfg.PersistBatchAsync {
		engine.WithBatchQueue(s.batchQueue)
	}
	if s.cfg.CurrentCSRSeasonID != "" {
		engine.WithCSRSeasonID(s.cfg.CurrentCSRSeasonID)
	}
	// Résolution autonome des noms d'assets au sync (primary write) — chemin V1
	// engine.run (fallback quand V2 off + CLI). On passe le POOL UNIFIÉ (s.pool,
	// la même source que tous les syncs) ; GameCMS exige un token. Gate
	// LEVELUP_SYNC_RESOLVE_ASSETS appliqué côté sync. Le chemin prod V2 est câblé
	// séparément (persister).
	engine.WithAssetNameResolution(s.pool)
	// Hook Prestige post-sync (best-effort). Fire à engine.run() après le pipeline
	// post-sync, pendant que le lease player/shared est encore tenu — instance directe
	// non-lease (invariant deadlock-free, cf. wire/prestige_setup.go). gamertag ==
	// PlayerSlug pour les joueurs réels (cf. WithPostSyncRunner). Couvre les 3 entrées
	// V1 qui passent par BuildEngine : auto-sync, HTTP delta (engineBuilder), watcher.
	if s.prestigeHook != nil {
		hook := s.prestigeHook
		engine.WithPrestigeHook(func(hookCtx context.Context, playerSlug, ts string) {
			hook(hookCtx, playerSlug, ts)
		})
	}
	return engine
}

// defaultRunnerFactory adapte BuildEngine vers l'interface DeltaRunner
// attendue par le scheduler (RunDelta-only). *sync.SyncEngine satisfait
// DeltaRunner via sa méthode RunDelta.
//
// Garde un nom historique pour ne pas casser l'API interne du scheduler
// (champ RunnerFactory, tests existants).
func (s *AutoSyncScheduler) defaultRunnerFactory(ctx context.Context, gamertag, xuid string) DeltaRunner {
	return s.BuildEngine(ctx, gamertag, xuid)
}

// acquireLiveTitleRunner est l'implémentation prod de liveTitleRunnerResolver pour
// les titres live-only (Halo 5+). Délègue à livesync.AcquireRunner (source unique
// pool→runner, partagée avec le path watcher) : le pool fournit le SpartanToken
// (là où le path HTTP le tient de la session), posé dans le ctx pour que l'adapter
// du titre le lise. Le lease est tenu jusqu'au release (différé par syncPlayer).
func (s *AutoSyncScheduler) acquireLiveTitleRunner(ctx context.Context, slug, gamertag, xuid string) (DeltaRunner, context.Context, func(), error) {
	// *Runner est nil-able : en cas d'erreur on retourne l'interface DeltaRunner
	// explicitement nil (évite le piège du typed-nil).
	r, runCtx, release, err := livesync.AcquireRunner(ctx, s.pool, s.cfg, slug, gamertag, xuid)
	if err != nil {
		return nil, ctx, release, err
	}
	return r, runCtx, release, nil
}

// resolveTitleSlug retourne le slug du profil joueur, avec fallback DefaultSlug
// (profils flat implicitement halo_infinite, ou TitleSlug vide). Source unique de
// la règle de fallback (MT-11 / PMT-3), partagée par syncPlayer + la précondition.
func resolveTitleSlug(p domain.PlayerSummary) string {
	if p.TitleSlug == "" {
		return titlePkg.DefaultSlug
	}
	return p.TitleSlug
}
