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
	gosync "sync"
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
	Total    int           `json:"total"`    // joueurs dans db_profiles
	Synced   int           `json:"synced"`   // sync delta déclenchée avec succès
	Skipped  int           `json:"skipped"`  // token absent → ignoré
	Failed   int           `json:"failed"`   // erreur pendant la sync
	Duration time.Duration `json:"duration_ns"`
}

// PlayerOutcomeDetail capture le résultat détaillé d'une tentative de sync pour
// un joueur, pour exposition via l'endpoint admin /api/v1/admin/auto-sync/snapshot.
// Permet de diagnostiquer pourquoi un joueur ne sync pas sans avoir accès aux logs.
type PlayerOutcomeDetail struct {
	Gamertag        string    `json:"gamertag"`
	XUID            string    `json:"xuid"`
	Outcome         string    `json:"outcome"`         // "ok", "skipped", "failed"
	Reason          string    `json:"reason"`          // texte libre expliquant le résultat
	AttemptedAt     time.Time `json:"attempted_at"`
	DurationMs      int64     `json:"duration_ms"`
	MatchesInserted int       `json:"matches_inserted,omitempty"`
	MatchesSkipped  int       `json:"matches_skipped,omitempty"`
	MedalsInserted  int       `json:"medals_inserted,omitempty"`
	SyncStatus      string    `json:"sync_status,omitempty"` // SyncResult.Status()
	ErrorCount      int       `json:"error_count,omitempty"`
	FirstError      string    `json:"first_error,omitempty"`
}

// SchedulerSnapshot est exposé par l'endpoint admin pour diagnostic.
type SchedulerSnapshot struct {
	LastCycleAt     time.Time             `json:"last_cycle_at"`
	LastCycleResult *RunOnceResult        `json:"last_cycle_result,omitempty"`
	IntervalMinutes int                   `json:"interval_minutes"`
	Players         []PlayerOutcomeDetail `json:"players"`
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

	// Snapshot par joueur du dernier cycle — pour l'endpoint admin diagnostic.
	snapshotMu      gosync.RWMutex
	lastCycleAt     time.Time
	lastCycleResult *RunOnceResult
	playerOutcomes  map[string]PlayerOutcomeDetail // keyed by gamertag
}

// New crée un AutoSyncScheduler avec les implémentations de production.
func New(
	cfg *config.AppConfig,
	settings *settings_platform.Store,
	provider auth.TokenProvider,
) *AutoSyncScheduler {
	return &AutoSyncScheduler{
		cfg:            cfg,
		settings:       settings,
		provider:       provider,
		EngineFactory:  defaultEngineFactory,
		TokenReader:    defaultTokenReader,
		playerOutcomes: make(map[string]PlayerOutcomeDetail),
	}
}

// Snapshot retourne un cliché thread-safe du dernier cycle de sync, incluant
// le détail par joueur (raison du skip/failure, compteurs, erreurs).
// Utilisé par l'endpoint admin /api/v1/admin/auto-sync/snapshot pour permettre
// le diagnostic sans accès aux logs serveur.
func (s *AutoSyncScheduler) Snapshot() SchedulerSnapshot {
	s.snapshotMu.RLock()
	defer s.snapshotMu.RUnlock()

	players := make([]PlayerOutcomeDetail, 0, len(s.playerOutcomes))
	for _, d := range s.playerOutcomes {
		players = append(players, d)
	}
	intervalMinutes := int(s.CurrentInterval() / time.Minute)
	snap := SchedulerSnapshot{
		LastCycleAt:     s.lastCycleAt,
		IntervalMinutes: intervalMinutes,
		Players:         players,
	}
	if s.lastCycleResult != nil {
		copyRes := *s.lastCycleResult
		snap.LastCycleResult = &copyRes
	}
	return snap
}

// recordOutcome enregistre le résultat détaillé pour un joueur (thread-safe).
func (s *AutoSyncScheduler) recordOutcome(d PlayerOutcomeDetail) {
	s.snapshotMu.Lock()
	defer s.snapshotMu.Unlock()
	if s.playerOutcomes == nil {
		s.playerOutcomes = make(map[string]PlayerOutcomeDetail)
	}
	s.playerOutcomes[d.Gamertag] = d
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

	// Mémoriser pour l'endpoint admin diagnostic.
	s.snapshotMu.Lock()
	s.lastCycleAt = time.Now()
	copyRes := *res
	s.lastCycleResult = &copyRes
	s.snapshotMu.Unlock()

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
//
// Enregistre toujours un PlayerOutcomeDetail dans s.playerOutcomes (via defer)
// pour exposer le résultat via l'endpoint admin /api/v1/admin/auto-sync/snapshot.
func (s *AutoSyncScheduler) syncPlayer(ctx context.Context, p domain.PlayerSummary) syncOutcome {
	slog.DebugContext(ctx, "auto_sync: traitement joueur", "gamertag", p.Gamertag, "xuid", p.XUID)

	startedAt := time.Now()
	detail := PlayerOutcomeDetail{
		Gamertag:    p.Gamertag,
		XUID:        p.XUID,
		AttemptedAt: startedAt,
	}
	var outcome syncOutcome
	defer func() {
		switch outcome {
		case outcomeOK:
			detail.Outcome = "ok"
		case outcomeSkipped:
			detail.Outcome = "skipped"
		case outcomeFailed:
			detail.Outcome = "failed"
		}
		detail.DurationMs = time.Since(startedAt).Milliseconds()
		s.recordOutcome(detail)
	}()

	// Si le watcher est actif sur ce joueur (Watching/Syncing/Cooling),
	// on cède la priorité pour éviter deux sync concurrentes sur la même DB.
	if s.ActivityChecker != nil && s.ActivityChecker.IsPlayerActive(p.Gamertag) {
		slog.InfoContext(ctx, "auto_sync: watcher actif sur ce joueur — tick cédé",
			"gamertag", p.Gamertag,
		)
		detail.Reason = "watcher actif sur ce joueur (Watching/Syncing/Cooling) — tick cédé"
		outcome = outcomeSkipped
		return outcome
	}

	dbPath := titlePkg.NewPathResolver(s.cfg.RepoRoot).PlayerDBPath(titlePkg.DefaultSlug, p.Gamertag)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		slog.InfoContext(ctx, "auto_sync: DB joueur absente, joueur ignoré",
			"gamertag", p.Gamertag,
			"db_path", dbPath,
			"hint", "la sync initiale n'a peut-être jamais été lancée pour ce joueur",
		)
		detail.Reason = "DB joueur absente (" + dbPath + ") — sync initiale jamais effectuée ?"
		outcome = outcomeSkipped
		return outcome
	}

	accessToken, err := s.TokenReader(ctx, dbPath, p.Gamertag, s.provider)
	if err != nil {
		slog.ErrorContext(ctx, "auto_sync: lecture token échouée",
			"gamertag", p.Gamertag,
			"db_path", dbPath,
			"err", err,
		)
		detail.Reason = "lecture token échouée: " + err.Error()
		detail.FirstError = err.Error()
		outcome = outcomeFailed
		return outcome
	}
	if accessToken == "" {
		slog.InfoContext(ctx, "auto_sync: aucun token en cache, joueur ignoré",
			"gamertag", p.Gamertag,
			"hint", "l'utilisateur doit se reconnecter via /api/v1/auth/device-flow",
		)
		detail.Reason = "aucun token utilisable (MSAL cache + oauth_refresh_token + SPNKR_OAUTH_REFRESH_TOKEN_" + strings.ToUpper(p.Gamertag) + " tous vides ou refresh révoqué)"
		outcome = outcomeSkipped
		return outcome
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
		detail.Reason = "exchange SPNKr échoué (token Microsoft peut-être révoqué): " + err.Error()
		detail.FirstError = err.Error()
		outcome = outcomeFailed
		return outcome
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
		detail.Reason = "RunDelta échoué: " + err.Error()
		detail.FirstError = err.Error()
		outcome = outcomeFailed
		return outcome
	}

	detail.MatchesInserted = syncResult.MatchesInserted
	detail.MatchesSkipped = syncResult.MatchesSkipped
	detail.MedalsInserted = syncResult.MedalsInserted
	detail.SyncStatus = syncResult.Status()
	detail.ErrorCount = len(syncResult.Errors)

	if len(syncResult.Errors) > 0 {
		detail.FirstError = syncResult.Errors[0]
		detail.Reason = "sync terminée avec " + strings.TrimSpace(safePrefix(syncResult.Errors[0], 200)) + " (et " + itoa(len(syncResult.Errors)-1) + " autres erreurs)"
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
		if syncResult.MatchesInserted > 0 {
			detail.Reason = "sync delta réussie"
		} else {
			detail.Reason = "sync delta réussie — 0 nouveau match (déjà à jour ou API n'a rien renvoyé)"
		}
		slog.InfoContext(ctx, "auto_sync: sync delta réussie",
			"gamertag", p.Gamertag,
			"inserted", syncResult.MatchesInserted,
			"skipped", syncResult.MatchesSkipped,
			"medals_inserted", syncResult.MedalsInserted,
			"duration_s", syncResult.DurationSeconds,
			"status", syncResult.Status(),
		)
	}
	outcome = outcomeOK
	return outcome
}

// itoa retourne une représentation décimale d'un int (sans dépendre de strconv
// au niveau du fichier ; strconv reste libre pour les call sites ailleurs).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
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
// Sources de refresh_token (essayées dans l'ordre, première qui donne un access_token gagne) :
//  1. Variable d'environnement SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> (.env.local)
//  2. sync_meta.oauth_refresh_token (DuckDB)
//
// On essaie l'env var EN PREMIER parce que :
//   - Microsoft rotate le refresh_token à chaque usage ;
//   - sync_meta peut contenir un RT historique déjà révoqué par une rotation antérieure ;
//   - l'env var est la "source canonique" maintenue par l'utilisateur dans .env.local.
//
// À chaque refresh OAuth réussi, le RT rotaté est persisté dans
// sync_meta.oauth_refresh_token, de sorte que les ticks suivants utilisent
// toujours le dernier RT valide (Microsoft rotate, on garde le pas).
//
// Tente aussi TrySilentRefresh (MSAL cache) en parallèle si présent.
// La connexion DB est fermée avant de retourner.
//
// Note : on ouvre en OpenReadWriteShared (clé cache "rw:path") plutôt que OpenReadOnly
// pour partager l'instance DuckDB avec le pool joueur déjà ouvert par les handlers HTTP.
// DuckDB interdit deux handles avec des access_mode différents sur le même fichier.
// On a besoin du mode RW pour persister le RT rotaté via WriteOAuthRefreshToken.
func defaultTokenReader(ctx context.Context, dbPath string, gamertag string, provider auth.TokenProvider) (string, error) {
	db, err := duckdb.OpenReadWriteShared(dbPath)
	if err != nil {
		return "", err
	}
	defer db.Close() //nolint:errcheck // best-effort à la fermeture

	var cacheJSON, dbRefreshToken string
	if err := db.SQLDb().QueryRowContext(ctx,
		"SELECT value FROM sync_meta WHERE key = 'msal_token_cache'").Scan(&cacheJSON); err != nil {
		slog.DebugContext(ctx, "auto_sync: msal_token_cache absent de sync_meta", "db", dbPath)
	}
	if err := db.SQLDb().QueryRowContext(ctx,
		"SELECT value FROM sync_meta WHERE key = 'oauth_refresh_token'").Scan(&dbRefreshToken); err != nil {
		slog.DebugContext(ctx, "auto_sync: oauth_refresh_token absent de sync_meta", "db", dbPath)
	}

	envRefreshToken := ""
	if gamertag != "" {
		key := strings.ToUpper(gamertag)
		key = strings.Map(func(r rune) rune {
			if r == ' ' || r == '-' || r == '.' {
				return '_'
			}
			return r
		}, key)
		envRefreshToken = os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_" + key)
	}

	// 1) MSAL silent refresh (pas de rotation côté Microsoft : pas besoin de
	//    persister).
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

	// 2) OAuth v2 refresh — essayer env var d'abord, puis sync_meta.
	type rtSource struct {
		token  string
		origin string
	}
	candidates := []rtSource{
		{token: envRefreshToken, origin: "env_var"},
		{token: dbRefreshToken, origin: "sync_meta"},
	}

	for _, c := range candidates {
		if c.token == "" {
			continue
		}
		accessToken, rotatedRT, oerr := auth.ExchangeRefreshTokenWithRotation(ctx, c.token)
		if oerr != nil {
			slog.WarnContext(ctx, "auto_sync: ExchangeRefreshToken a retourné une erreur",
				"source", c.origin, "gamertag", gamertag, "err", oerr,
			)
			continue
		}
		if accessToken == "" {
			slog.InfoContext(ctx, "auto_sync: OAuth refresh impossible (token révoqué ?)",
				"source", c.origin, "gamertag", gamertag,
			)
			continue
		}
		// Succès — persister le RT rotaté dans sync_meta pour le prochain tick.
		// Si Microsoft n'a pas rotaté (rotatedRT == ""), on persiste le RT
		// utilisé tel quel pour qu'il devienne la "source de vérité" du DB.
		toPersist := rotatedRT
		if toPersist == "" {
			toPersist = c.token
		}
		if werr := duckdb.WriteOAuthRefreshToken(ctx, db, toPersist); werr != nil {
			slog.WarnContext(ctx, "auto_sync: persistance refresh_token rotaté échouée",
				"source", c.origin, "gamertag", gamertag, "err", werr,
			)
		} else {
			slog.InfoContext(ctx, "auto_sync: OAuth refresh OK, RT rotaté persisté",
				"source", c.origin, "gamertag", gamertag, "rotated", rotatedRT != "",
			)
		}
		return accessToken, nil
	}

	return "", nil
}
