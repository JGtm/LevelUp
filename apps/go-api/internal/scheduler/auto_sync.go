// Package scheduler — auto_sync.go : synchronisation delta automatique et périodique.
//
// AutoSyncScheduler lit app_settings.json au démarrage et à chaque tick pour :
//   - vérifier que spnkr_auto_sync_enabled est vrai avant d'agir
//   - adapter l'intervalle si spnkr_auto_sync_interval_hours a changé
//
// Pour chaque joueur configuré dans db_profiles.json, il :
//  1. Lit stats.duckdb via tokenReader pour extraire le cache MSAL ou le refresh_token OAuth
//  2. Obtient un access_token via TokenProvider (TrySilentRefresh → TryOAuthRefresh)
//  3. Échange l'access_token contre des SpartanToken + ClearanceToken (provider.Exchange)
//  4. Lance DeltaRunner.RunDelta avec ces tokens
//
// Si un joueur n'a pas de token persisté, sa sync est silencieusement ignorée (INFO log).
// Les erreurs par joueur sont loggées et n'interrompent pas les joueurs suivants.
//
// Points de diagnostic :
//   - "auto_sync: DB joueur absente"      → stats.duckdb n'existe pas encore (sync initiale jamais faite ?)
//   - "auto_sync: aucun token en cache"   → DB présente mais sync_meta vide ou clés absentes
//   - "auto_sync: MSAL silent refresh OK" → renouvellement MSAL réussi
//   - "auto_sync: OAuth fallback OK"      → MSAL échoué, OAuth v2 utilisé
//   - "auto_sync: exchange échoué"        → access_token obtenu mais SPNKr Exchange a refusé
//   - "auto_sync: sync échouée"           → tokens OK mais RunDelta a retourné une erreur
package scheduler

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	duckdb "levelup/go-api/internal/platform/duckdb"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/sync"
)

const defaultIntervalHours = 6

// PlayerActivityChecker est satisfait par watcher.StateProvider.
// Défini ici pour éviter une dépendance circulaire scheduler→watcher.
type PlayerActivityChecker interface {
	IsPlayerActive(gamertag string) bool
}

// DeltaRunner abstrait l'exécution d'une sync delta (mockable dans les tests).
type DeltaRunner interface {
	RunDelta(ctx context.Context, opts domain.SyncOptions) (domain.SyncResult, error)
}

// EngineFactory crée un DeltaRunner pour un joueur donné.
// La factory par défaut utilise sync.NewSyncEngine.
type EngineFactory func(repoRoot, gamertag, xuid string, tokens *domain.HaloTokens, provider auth.TokenProvider) DeltaRunner

// TokenReader lit l'access_token depuis la DB d'un joueur (via sync_meta) et/ou
// depuis l'environnement (SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>).
// Retourne ("", nil) si aucun token utilisable n'est trouvé.
// Abstraction nécessaire pour mocker dans les tests sans créer de vraie DB DuckDB.
type TokenReader func(ctx context.Context, dbPath string, gamertag string, provider auth.TokenProvider) (string, error)

// RunOnceResult agrège les compteurs d'un cycle de sync.
type RunOnceResult struct {
	Total    int // joueurs dans db_profiles
	Synced   int // sync delta déclenchée avec succès
	Skipped  int // token absent → ignoré
	Failed   int // erreur pendant la sync
	Duration time.Duration
}

// AutoSyncScheduler orchestre la sync delta périodique de tous les joueurs.
type AutoSyncScheduler struct {
	cfg      *config.AppConfig
	settings *settings_platform.Store
	provider auth.TokenProvider
	// EngineFactory est exporté pour injection dans les tests.
	EngineFactory EngineFactory
	// TokenReader est exporté pour injection dans les tests.
	TokenReader TokenReader
	// ActivityChecker est optionnel. S'il est défini, le scheduler saute le tick
	// pour les joueurs dont le watcher est en état Watching/Syncing/Cooling.
	// Doit être défini avant d'appeler Run.
	ActivityChecker PlayerActivityChecker
}

// New crée un AutoSyncScheduler avec les implémentations de production.
func New(
	cfg *config.AppConfig,
	settings *settings_platform.Store,
	provider auth.TokenProvider,
) *AutoSyncScheduler {
	return &AutoSyncScheduler{
		cfg:           cfg,
		settings:      settings,
		provider:      provider,
		EngineFactory: defaultEngineFactory,
		TokenReader:   defaultTokenReader,
	}
}

// Run démarre la boucle périodique. Doit être lancé dans une goroutine.
// Se termine proprement à l'annulation de ctx.
func (s *AutoSyncScheduler) Run(ctx context.Context) {
	interval := s.CurrentInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.InfoContext(ctx, "auto_sync: scheduler démarré",
		"interval", interval,
		"interval_hours", int(interval.Hours()),
	)

	for {
		select {
		case <-ticker.C:
			cfg, err := s.settings.Load()
			if err != nil {
				slog.WarnContext(ctx, "auto_sync: lecture settings échouée — tick ignoré", "err", err)
				continue
			}
			if !cfg.SpnkrAutoSyncEnabled {
				slog.DebugContext(ctx, "auto_sync: désactivé dans les settings, tick ignoré")
				continue
			}
			if newInterval := resolveInterval(cfg.SpnkrAutoSyncIntervalMinutes, cfg.SpnkrAutoSyncIntervalHours); newInterval != interval {
				slog.InfoContext(ctx, "auto_sync: intervalle mis à jour",
					"old", interval,
					"new", newInterval,
				)
				interval = newInterval
				ticker.Reset(interval)
			}
			res := s.RunOnce(ctx)
			slog.InfoContext(ctx, "auto_sync: cycle terminé",
				"total", res.Total,
				"synced", res.Synced,
				"skipped", res.Skipped,
				"failed", res.Failed,
				"duration", res.Duration.Round(time.Millisecond),
			)
			if res.Failed > 0 {
				slog.WarnContext(ctx, "auto_sync: des joueurs ont échoué — consulter les logs ci-dessus",
					"failed_count", res.Failed,
				)
			}
			if res.Total > 0 && res.Skipped == res.Total {
				slog.WarnContext(ctx, "auto_sync: aucun token valide trouvé pour aucun joueur",
					"hint", "relancer une authentification interactive via /api/v1/auth/device-flow",
				)
			}

		case <-ctx.Done():
			slog.InfoContext(ctx, "auto_sync: arrêt du scheduler (contexte annulé)")
			return
		}
	}
}

// RunOnce exécute un cycle de sync pour tous les joueurs configurés.
// Peut être appelé manuellement (debug, endpoint) sans attendre un tick.
func (s *AutoSyncScheduler) RunOnce(ctx context.Context) *RunOnceResult {
	start := time.Now()
	res := &RunOnceResult{}

	players, err := s.cfg.LoadPlayers()
	if err != nil {
		slog.ErrorContext(ctx, "auto_sync: chargement des joueurs échoué", "err", err)
		return res
	}
	res.Total = len(players)
	slog.InfoContext(ctx, "auto_sync: démarrage du cycle", "player_count", res.Total)

	for _, p := range players {
		outcome := s.syncPlayer(ctx, p)
		switch outcome {
		case outcomeOK:
			res.Synced++
		case outcomeSkipped:
			res.Skipped++
		case outcomeFailed:
			res.Failed++
		}
	}

	res.Duration = time.Since(start)
	return res
}

// syncOutcome encode le résultat d'une tentative de sync par joueur.
type syncOutcome int

const (
	outcomeOK      syncOutcome = iota // sync delta réussie
	outcomeSkipped                    // pas de token → ignoré sans erreur
	outcomeFailed                     // erreur bloquante
)

// syncPlayer effectue la résolution de tokens puis la sync delta pour un joueur.
func (s *AutoSyncScheduler) syncPlayer(ctx context.Context, p domain.PlayerSummary) syncOutcome {
	slog.DebugContext(ctx, "auto_sync: traitement joueur", "gamertag", p.Gamertag, "xuid", p.XUID)

	// Si le watcher est actif sur ce joueur (Watching/Syncing/Cooling),
	// on cède la priorité pour éviter deux sync concurrentes sur la même DB.
	if s.ActivityChecker != nil && s.ActivityChecker.IsPlayerActive(p.Gamertag) {
		slog.InfoContext(ctx, "auto_sync: watcher actif sur ce joueur — tick cédé",
			"gamertag", p.Gamertag,
		)
		return outcomeSkipped
	}

	dbPath := titlePkg.NewPathResolver(s.cfg.RepoRoot).PlayerDBPath(titlePkg.DefaultSlug, p.Gamertag)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		slog.InfoContext(ctx, "auto_sync: DB joueur absente, joueur ignoré",
			"gamertag", p.Gamertag,
			"db_path", dbPath,
			"hint", "la sync initiale n'a peut-être jamais été lancée pour ce joueur",
		)
		return outcomeSkipped
	}

	accessToken, err := s.TokenReader(ctx, dbPath, p.Gamertag, s.provider)
	if err != nil {
		slog.ErrorContext(ctx, "auto_sync: lecture token échouée",
			"gamertag", p.Gamertag,
			"db_path", dbPath,
			"err", err,
		)
		return outcomeFailed
	}
	if accessToken == "" {
		slog.InfoContext(ctx, "auto_sync: aucun token en cache, joueur ignoré",
			"gamertag", p.Gamertag,
			"hint", "l'utilisateur doit se reconnecter via /api/v1/auth/device-flow",
		)
		return outcomeSkipped
	}

	slog.DebugContext(ctx, "auto_sync: échange access_token → tokens Halo", "gamertag", p.Gamertag)
	exchangeStart := time.Now()
	result, err := s.provider.Exchange(ctx, accessToken)
	if err != nil {
		slog.ErrorContext(ctx, "auto_sync: exchange SPNKr échoué",
			"gamertag", p.Gamertag,
			"exchange_duration", time.Since(exchangeStart).Round(time.Millisecond),
			"err", err,
			"hint", "token Microsoft peut-être révoqué — relancer device-flow",
		)
		return outcomeFailed
	}
	slog.DebugContext(ctx, "auto_sync: exchange OK",
		"gamertag", p.Gamertag,
		"exchange_duration", time.Since(exchangeStart).Round(time.Millisecond),
		"spartan_token_prefix", safePrefix(result.Tokens.SpartanToken, 8),
	)

	slog.InfoContext(ctx, "auto_sync: démarrage sync delta", "gamertag", p.Gamertag)
	runner := s.EngineFactory(s.cfg.RepoRoot, p.Gamertag, p.XUID, result.Tokens, s.provider)
	// §7 Hook auto-recompute is_with_friends post-sync delta : si la factory
	// produit un *sync.SyncEngine concret (cas prod), on l'enrichit avec un
	// loader pointant sur s.settings.FriendGamertags. Les factories de test
	// (mocks) ne supportent pas ce hook → skip silencieux.
	if engine, ok := runner.(*sync.SyncEngine); ok && s.settings != nil {
		engine.WithFriendsLoader(func() ([]string, error) {
			cfg, lerr := s.settings.Load()
			if lerr != nil {
				return nil, lerr
			}
			return cfg.FriendGamertags, nil
		})
	}
	syncResult, err := runner.RunDelta(ctx, domain.DefaultSyncOptions())
	if err != nil {
		slog.ErrorContext(ctx, "auto_sync: RunDelta échoué",
			"gamertag", p.Gamertag,
			"err", err,
		)
		return outcomeFailed
	}

	if len(syncResult.Errors) > 0 {
		slog.WarnContext(ctx, "auto_sync: sync terminée avec erreurs partielles",
			"gamertag", p.Gamertag,
			"inserted", syncResult.MatchesInserted,
			"skipped", syncResult.MatchesSkipped,
			"error_count", len(syncResult.Errors),
			"first_error", syncResult.Errors[0],
			"duration_s", syncResult.DurationSeconds,
			"status", syncResult.Status(),
		)
	} else {
		slog.InfoContext(ctx, "auto_sync: sync delta réussie",
			"gamertag", p.Gamertag,
			"inserted", syncResult.MatchesInserted,
			"skipped", syncResult.MatchesSkipped,
			"medals_inserted", syncResult.MedalsInserted,
			"duration_s", syncResult.DurationSeconds,
			"status", syncResult.Status(),
		)
	}
	return outcomeOK
}

// CurrentInterval retourne l'intervalle courant depuis les settings.
// Utilise _minutes en priorité si > 0, sinon fallback sur _hours.
// Retourne defaultIntervalHours en cas d'erreur de lecture.
// Exporté pour les tests.
func (s *AutoSyncScheduler) CurrentInterval() time.Duration {
	cfg, err := s.settings.Load()
	if err != nil {
		return intervalFromHours(0)
	}
	return resolveInterval(cfg.SpnkrAutoSyncIntervalMinutes, cfg.SpnkrAutoSyncIntervalHours)
}

// resolveInterval retourne la durée à utiliser.
// minutes > 0 → utiliser les minutes. Sinon fallback sur hours.
func resolveInterval(minutes, hours int) time.Duration {
	if minutes > 0 {
		return time.Duration(minutes) * time.Minute
	}
	return intervalFromHours(hours)
}

// intervalFromHours convertit un nombre d'heures en Duration.
// Retourne defaultIntervalHours si h <= 0.
func intervalFromHours(h int) time.Duration {
	if h <= 0 {
		return defaultIntervalHours * time.Hour
	}
	return time.Duration(h) * time.Hour
}

// safePrefix retourne les n premiers caractères d'une string, sans paniquer.
// Utilisé pour logger un préfixe de token sans l'exposer entièrement.
func safePrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// ---------------------------------------------------------------------------
// Implémentations de production des fonctions injectable
// ---------------------------------------------------------------------------

// defaultEngineFactory crée un sync.SyncEngine réel.
func defaultEngineFactory(repoRoot, gamertag, xuid string, tokens *domain.HaloTokens, provider auth.TokenProvider) DeltaRunner {
	return sync.NewSyncEngine(repoRoot, gamertag, xuid, tokens, provider)
}

// defaultTokenReader lit sync_meta depuis stats.duckdb du joueur et rafraîchit l'access_token.
// Ordre de recherche du refresh_token :
//  1. sync_meta.oauth_refresh_token (DuckDB)
//  2. Variable d'environnement SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> (.env.local)
//
// Tente TrySilentRefresh (MSAL) d'abord, puis TryOAuthRefresh sur le refresh_token trouvé.
// La connexion DB est fermée avant de retourner.
func defaultTokenReader(ctx context.Context, dbPath string, gamertag string, provider auth.TokenProvider) (string, error) {
	db, err := duckdb.OpenReadOnly(dbPath)
	if err != nil {
		return "", err
	}
	defer db.Close() //nolint:errcheck // best-effort à la fermeture

	var cacheJSON, refreshToken string
	if err := db.SQLDb().QueryRowContext(ctx,
		"SELECT value FROM sync_meta WHERE key = 'msal_token_cache'").Scan(&cacheJSON); err != nil {
		slog.DebugContext(ctx, "auto_sync: msal_token_cache absent de sync_meta", "db", dbPath)
	}
	if err := db.SQLDb().QueryRowContext(ctx,
		"SELECT value FROM sync_meta WHERE key = 'oauth_refresh_token'").Scan(&refreshToken); err != nil {
		slog.DebugContext(ctx, "auto_sync: oauth_refresh_token absent de sync_meta", "db", dbPath)
	}

	// Fallback : variable d'environnement SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>
	// (chargée depuis .env.local au démarrage via config.Load).
	if refreshToken == "" && gamertag != "" {
		key := strings.ToUpper(gamertag)
		key = strings.Map(func(r rune) rune {
			if r == ' ' || r == '-' || r == '.' {
				return '_'
			}
			return r
		}, key)
		if v := os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_" + key); v != "" {
			refreshToken = v
			slog.DebugContext(ctx, "auto_sync: refresh_token lu depuis env var", "gamertag", gamertag)
		}
	}

	if cacheJSON != "" {
		token, err := provider.TrySilentRefresh(ctx, cacheJSON)
		if err != nil {
			slog.WarnContext(ctx, "auto_sync: TrySilentRefresh a retourné une erreur, tentative OAuth fallback", "err", err, "db", dbPath)
		} else if token != "" {
			slog.DebugContext(ctx, "auto_sync: MSAL silent refresh OK", "db", dbPath)
			return token, nil
		} else {
			slog.InfoContext(ctx, "auto_sync: MSAL silent refresh impossible (cache expiré ou invalide), tentative OAuth fallback", "db", dbPath)
		}
	} else {
		slog.DebugContext(ctx, "auto_sync: cache MSAL absent, tentative OAuth directe", "db", dbPath)
	}

	if refreshToken != "" {
		token, err := provider.TryOAuthRefresh(ctx, refreshToken)
		if err != nil {
			slog.WarnContext(ctx, "auto_sync: TryOAuthRefresh a retourné une erreur", "err", err, "db", dbPath)
			return "", nil //nolint:nilerr // erreur OAuth non fatale → joueur simplement ignoré
		}
		if token != "" {
			slog.InfoContext(ctx, "auto_sync: OAuth v2 fallback OK", "db", dbPath)
			return token, nil
		}
		slog.InfoContext(ctx, "auto_sync: OAuth v2 fallback impossible (refresh token révoqué ?)", "db", dbPath)
	}

	return "", nil
}
