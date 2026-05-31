// Package scheduler — spartan_customization_cron.go : cron leger qui
// rafraichit la customisation Spartan (banniere, emblem, backdrop,
// spartan_id) pour tous les joueurs du pool toutes les N heures.
//
// Pourquoi : meme apres Phase 4-7 V2 (live a chaque visite home avec
// INSERT partial field-aware), un joueur qui n'ouvre JAMAIS l'app ne verra
// jamais sa customisation populee. Le cron garantit que tous les joueurs
// configures ont au moins une tentative de fetch toutes les 8h (defaut),
// independamment de l'usage de l'UI.
//
// Architecture :
//   - Reutilise CareerLiveService.GetSpartanIdentity (chemin live unifie)
//   - Pour chaque joueur du pool : construit un ctx avec ses tokens, appelle
//     GetSpartanIdentity, ce qui declenche kickoffBackgroundRefresh →
//     persistPartial avec FetchStatus approprie
//   - Aucune duplication de logique : c'est le MEME path que la visite home
//
// Pas de retry interne : si l'API echoue, on aura un status 'failed' ou
// 'api_empty' en DB, et le cron prochain reessaiera. Best-effort.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/platform/duckdb"
)

// SpartanIdentityFetcher abstrait CareerLiveService pour le mocking.
type SpartanIdentityFetcher interface {
	GetSpartanIdentity(ctx context.Context) (*domain.HomeSpartanIdentityRow, error)
}

// CareerLiveServiceProvider retourne un fetcher per-player.
// Implémenté par api.ServiceRegistry.CareerLiveCtx (signature adaptée).
type CareerLiveServiceProvider func(ctx context.Context, slug string) (SpartanIdentityFetcher, error)

// SpartanCustomizationCron itere sur tous les joueurs du pool toutes les
// N heures et appelle CareerLiveService.GetSpartanIdentity pour chacun
// avec ses tokens en context. Cela declenche le path live habituel
// (kickoffBackgroundRefresh → persistPartial).
type SpartanCustomizationCron struct {
	cfg         *config.AppConfig
	pool        pool.Pool
	svcProvider CareerLiveServiceProvider
	titleSlug   string
	interval    time.Duration
}

// DefaultSpartanCustomizationInterval est l'intervalle par defaut (8h)
// — choisi pour couvrir l'evolution Spartan ID/banniere d'un joueur actif
// sans saturer l'API Halo.
const DefaultSpartanCustomizationInterval = 8 * time.Hour

// NewSpartanCustomizationCron construit le cron. Si interval == 0,
// DefaultSpartanCustomizationInterval est utilise.
func NewSpartanCustomizationCron(
	cfg *config.AppConfig,
	tokenPool pool.Pool,
	svcProvider CareerLiveServiceProvider,
	titleSlug string,
	interval time.Duration,
) *SpartanCustomizationCron {
	if interval <= 0 {
		interval = DefaultSpartanCustomizationInterval
	}
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	return &SpartanCustomizationCron{
		cfg:         cfg,
		pool:        tokenPool,
		svcProvider: svcProvider,
		titleSlug:   titleSlug,
		interval:    interval,
	}
}

// Run lance le cron : un premier tick immediat (pour ne pas attendre N heures
// au boot), puis toutes les `interval`. Bloque jusqu'a ctx.Done().
func (c *SpartanCustomizationCron) Run(ctx context.Context) {
	if c == nil || c.cfg == nil || c.svcProvider == nil {
		slog.WarnContext(ctx, "spartan_cron: noop (cfg/svc nil)")
		return
	}
	slog.InfoContext(ctx, "spartan_cron: started", "interval", c.interval)

	// Premier tick immediat — utile au boot et pour debug.
	c.RunOnce(ctx)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "spartan_cron: stopped (ctx done)")
			return
		case <-ticker.C:
			c.RunOnce(ctx)
		}
	}
}

// RunOnce execute un cycle pour tous les joueurs configures. Exporte pour
// les endpoints admin (force-refresh) et les tests.
func (c *SpartanCustomizationCron) RunOnce(ctx context.Context) {
	if c == nil || c.cfg == nil || c.svcProvider == nil {
		return
	}
	start := time.Now()
	players, err := c.cfg.LoadPlayers(c.titleSlug)
	if err != nil {
		slog.ErrorContext(ctx, "spartan_cron: load players failed", "err", err)
		return
	}

	var ok, skipped, failed int
	var lockedDBs []string
	for _, p := range players {
		outcome, err := c.refreshOne(ctx, p)
		switch outcome {
		case refreshOK:
			ok++
		case refreshSkipped:
			skipped++
		case refreshFailed:
			failed++
			if duckdb.IsFileLockError(err) {
				lockedDBs = append(lockedDBs, p.Gamertag)
			}
		}
	}
	// Diagnostic agrégé "fail-fast" : si une ou plusieurs player DB sont
	// verrouillées par un autre process (CLI backfill, 2e instance serveur,
	// hot-reload Air pas encore libéré), on émet UNE ligne ERROR claire et
	// actionnable plutôt que N WARN éparpillés qui noient la cause racine.
	// Pas d'abort du process : les locks au boot sont souvent transitoires
	// (cf. db.go IsFileLockError + commentaire Air post-SIGKILL).
	if len(lockedDBs) > 0 {
		slog.ErrorContext(ctx, "spartan_cron: player DB(s) verrouillée(s) par un autre process — "+
			"un writer concurrent (CLI backfill / 2e instance serveur / Air pas encore libéré) tient le fichier RW ; "+
			"ces joueurs restent dégradés jusqu'à sa fermeture (DuckDB est mono-writer par fichier)",
			"locked_players", lockedDBs, "count", len(lockedDBs))
	}
	slog.InfoContext(ctx, "spartan_cron: cycle done",
		"players", len(players),
		"ok", ok, "skipped", skipped, "failed", failed, "locked", len(lockedDBs),
		"duration", time.Since(start))
}

type refreshOutcome int

const (
	refreshOK refreshOutcome = iota
	refreshSkipped
	refreshFailed
)

// refreshOne refresh la customisation d'UN joueur en appelant le service
// live avec ses tokens dans le ctx. Best-effort, ne bloque pas le cycle.
func (c *SpartanCustomizationCron) refreshOne(ctx context.Context, p domain.PlayerSummary) (refreshOutcome, error) {
	if c.pool == nil {
		return refreshSkipped, nil
	}
	if p.XUID == "" || p.Gamertag == "" {
		return refreshSkipped, nil
	}
	if !c.pool.HasPlayer(p.Gamertag) {
		slog.DebugContext(ctx, "spartan_cron: skip (not in pool)",
			"gamertag", p.Gamertag)
		return refreshSkipped, nil
	}

	// Acquire un lease pinned sur ce joueur (customisation = endpoint privacy-gated).
	lease, err := c.pool.Acquire(ctx, pool.PolicyPinnedPlayer, p.Gamertag)
	if err != nil || lease == nil {
		slog.WarnContext(ctx, "spartan_cron: pool acquire failed",
			"gamertag", p.Gamertag, "err", err)
		return refreshFailed, err
	}
	defer lease.Release()

	// Construit un ctx avec les tokens du joueur pour que CareerLiveService
	// puisse appeler l'API Halo en son nom.
	playerCtx := ctxkeys.WithHaloAuth(ctx, lease.Tokens, p.XUID)
	playerCtx, cancel := context.WithTimeout(playerCtx, 30*time.Second)
	defer cancel()

	svc, err := c.svcProvider(playerCtx, p.PlayerSlug)
	if err != nil {
		// Lock concurrent sur la player DB : pas de WARN par-joueur (la cause
		// racine est loggée une seule fois, agrégée, en fin de cycle par RunOnce).
		if duckdb.IsFileLockError(err) {
			slog.DebugContext(ctx, "spartan_cron: svcProvider failed (player DB lock — agrégé en fin de cycle)",
				"gamertag", p.Gamertag, "err", err)
		} else {
			slog.WarnContext(ctx, "spartan_cron: svcProvider failed",
				"gamertag", p.Gamertag, "slug", p.PlayerSlug, "err", err)
		}
		return refreshFailed, err
	}
	if _, err := svc.GetSpartanIdentity(playerCtx); err != nil {
		slog.WarnContext(ctx, "spartan_cron: GetSpartanIdentity failed",
			"gamertag", p.Gamertag, "err", err)
		return refreshFailed, err
	}
	return refreshOK, nil
}
