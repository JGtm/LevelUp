// Package scheduler — spartan_customization_cron.go : cron leger qui
// rafraichit la customisation Spartan (banniere, emblem, backdrop,
// spartan_id / service tag) pour tous les joueurs du pool toutes les N heures,
// POUR TOUS LES TITRES ACTIFS (title-aware, capability-driven).
//
// Pourquoi : meme apres Phase 4-7 V2 (live a chaque visite home avec
// INSERT partial field-aware), un joueur qui n'ouvre JAMAIS l'app ne verra
// jamais sa customisation populee. Le cron garantit que tous les joueurs
// configures ont au moins une tentative de fetch toutes les 8h (defaut),
// independamment de l'usage de l'UI — quel que soit le titre.
//
// Architecture title-aware (refactor h5-capability-unification) :
//   - Le cron NE connait AUCUN titre concret (pas d'import internal/games/*).
//     Il itere sur les titres du registre (DefaultRegistry().All()) et delegue
//     le refresh A UN REFRESHER ENREGISTRE PAR TITRE (map[slug]CustomizationRefresher).
//   - La sequence COMMUNE (lookup pool, lease pinned, ctx auth, timeout) reste ici :
//     le refresher recoit un ctx DEJA muni des tokens du joueur et ne fait QUE
//     l'appel metier specifique au titre.
//   - Le WIRING CONCRET des refreshers se fait dans cmd/server/main.go :
//   - halo_infinite -> CareerLiveService.GetSpartanIdentity (chemin live
//     unifie, MEME path que la visite home → kickoffBackgroundRefresh →
//     persistPartial field-aware) ;
//   - halo_5         -> livesync.PersistAppearance (fetch /h5/profiles/{gt}/
//     {appearance,spartan,emblem} + persist service_tag/banner/emblem).
//   - Un titre actif SANS refresher enregistre est SKIPPE proprement (slog Debug),
//     jamais une erreur, jamais de panic.
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
	"levelup/go-api/internal/observability"
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

// CustomizationRefresher rafraichit la customisation Spartan d'UN joueur d'UN
// titre donne. Le ctx fourni porte DEJA les tokens d'auth du joueur (poses par
// le cron via ctxkeys.WithHaloAuth) et un timeout : le refresher ne fait QUE
// l'appel metier specifique au titre (live career identity, fetch appearance…).
// Abstraction title-agnostic : le scheduler ne depend d'AUCUN package de titre ;
// chaque titre injecte son implementation au boot (cf. cmd/server/main.go).
type CustomizationRefresher func(ctx context.Context, p domain.PlayerSummary) error

// SpartanCustomizationCron itere sur tous les titres actifs et, pour chacun,
// sur tous ses joueurs du pool toutes les N heures, en appelant le refresher
// enregistre pour CE titre avec les tokens du joueur en context. Cela declenche
// le path de rafraichissement propre au titre.
type SpartanCustomizationCron struct {
	cfg        *config.AppConfig
	pool       pool.Pool
	registry   *titlePkg.Registry
	refreshers map[string]CustomizationRefresher
	interval   time.Duration
}

// DefaultSpartanCustomizationInterval est l'intervalle par defaut (8h)
// — choisi pour couvrir l'evolution Spartan ID/banniere d'un joueur actif
// sans saturer l'API Halo.
const DefaultSpartanCustomizationInterval = 8 * time.Hour

// NewSpartanCustomizationCron construit le cron. Si interval == 0,
// DefaultSpartanCustomizationInterval est utilise.
//
// titleSlug + svcProvider câblent le refresher du titre HISTORIQUE (Halo Infinite
// par defaut) : la closure appelle svcProvider(ctx, slug).GetSpartanIdentity. Les
// AUTRES titres (Halo 5+) s'enregistrent ensuite via WithRefresher (cf. main.go).
// svcProvider nil ⇒ aucun refresher HINF (le cron reste sain, simplement no-op pour
// ce titre tant qu'aucun refresher n'est enregistre).
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
	c := &SpartanCustomizationCron{
		cfg:        cfg,
		pool:       tokenPool,
		registry:   titlePkg.DefaultRegistry(),
		refreshers: make(map[string]CustomizationRefresher),
		interval:   interval,
	}
	if svcProvider != nil {
		c.refreshers[titleSlug] = careerIdentityRefresher(svcProvider)
	}
	return c
}

// careerIdentityRefresher adapte un CareerLiveServiceProvider (chemin live unifie
// Halo Infinite) en CustomizationRefresher : resout le fetcher per-player puis
// appelle GetSpartanIdentity (→ kickoffBackgroundRefresh → persistPartial). Le ctx
// porte deja l'auth du joueur (pose par refreshOne). Comportement HINF identique
// a l'historique.
func careerIdentityRefresher(svcProvider CareerLiveServiceProvider) CustomizationRefresher {
	return func(ctx context.Context, p domain.PlayerSummary) error {
		svc, err := svcProvider(ctx, p.PlayerSlug)
		if err != nil {
			return err
		}
		_, err = svc.GetSpartanIdentity(ctx)
		return err
	}
}

// WithRegistry remplace le registre de titres itere par le cron (defaut :
// DefaultRegistry()). Retourne le cron pour chainage. nil-safe. Utile pour le
// wiring (registre piloté par config) et les tests (registre a titres controles).
func (c *SpartanCustomizationCron) WithRegistry(reg *titlePkg.Registry) *SpartanCustomizationCron {
	if c != nil && reg != nil {
		c.registry = reg
	}
	return c
}

// WithRefresher enregistre le refresher de customisation d'un titre supplementaire
// (ex. halo_5 → livesync.PersistAppearance) et retourne le cron pour chainage.
// nil-safe : slug vide ou refresher nil est ignore. Le wiring (cmd/server) appelle
// ceci pour CHAQUE titre live-only avec ses deps deja construites au boot.
func (c *SpartanCustomizationCron) WithRefresher(titleSlug string, r CustomizationRefresher) *SpartanCustomizationCron {
	if c == nil || titleSlug == "" || r == nil {
		return c
	}
	if c.refreshers == nil {
		c.refreshers = make(map[string]CustomizationRefresher)
	}
	c.refreshers[titleSlug] = r
	return c
}

// Run lance le cron : un premier tick immediat (pour ne pas attendre N heures
// au boot), puis toutes les `interval`. Bloque jusqu'a ctx.Done().
func (c *SpartanCustomizationCron) Run(ctx context.Context) {
	if c == nil || c.cfg == nil || len(c.refreshers) == 0 {
		slog.WarnContext(ctx, "spartan_cron: noop (cfg nil ou aucun refresher enregistre)")
		return
	}
	slog.InfoContext(ctx, "spartan_cron: started", "interval", c.interval, "titles", len(c.refreshers))

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

// RunOnce execute un cycle pour TOUS les titres du registre. Exporte pour les
// endpoints admin (force-refresh) et les tests. Title-aware : itere les titres,
// charge les joueurs de CHAQUE titre et delegue au refresher enregistre. Un titre
// sans refresher est skippe proprement.
func (c *SpartanCustomizationCron) RunOnce(ctx context.Context) {
	if c == nil || c.cfg == nil || len(c.refreshers) == 0 {
		return
	}
	reg := c.registry
	if reg == nil {
		reg = titlePkg.DefaultRegistry()
	}
	// Itère les titres ACTIFS (parité avec world_leaderboard_cron, le pattern de
	// référence de ce sprint) : un titre archivé ne doit jamais être rafraîchi,
	// et un refresher n'est de toute façon câblé que pour des titres actifs. Le
	// gate FORT reste la présence d'un refresher enregistré (cf. runOnceForTitle).
	for _, desc := range reg.Active() {
		if desc == nil {
			continue
		}
		c.runOnceForTitle(ctx, desc.Slug)
	}
}

// runOnceForTitle execute un cycle pour tous les joueurs configures d'UN titre.
// Skip propre (slog Debug) si aucun refresher n'est enregistre pour ce titre —
// degradation gracieuse, jamais une erreur (le scheduler ne connait pas les
// titres concrets : un titre actif sans source de customisation est legitime).
func (c *SpartanCustomizationCron) runOnceForTitle(ctx context.Context, titleSlug string) {
	refresher, ok := c.refreshers[titleSlug]
	if !ok {
		slog.DebugContext(ctx, "spartan_cron: titre sans refresher de customisation — skip",
			"titleSlug", titleSlug)
		return
	}

	start := time.Now()
	players, err := c.cfg.LoadPlayers(titleSlug)
	if err != nil {
		slog.ErrorContext(ctx, "spartan_cron: load players failed",
			"titleSlug", titleSlug, "err", err)
		return
	}

	var succeeded, skipped, failed int
	var lockedDBs []string
	for _, p := range players {
		outcome, rerr := c.refreshOne(ctx, p, refresher)
		switch outcome {
		case refreshOK:
			succeeded++
		case refreshSkipped:
			skipped++
		case refreshFailed:
			failed++
			if duckdb.IsFileLockError(rerr) {
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
		// B3.3 : lock concurrent player DB = bruit LOCAL à cause connue (2e writer
		// air/worktree/CLI), self-healing à la fermeture du writer. Une seule ligne
		// WARN agrégée par cycle (les per-joueur sont en Debug) + compteur expvar
		// (DC-B2) — pas d'ERROR : ce n'est pas un incident serveur, c'est une
		// contention de poste de dev qui se résorbe seule.
		observability.AddInt("spartan_cron_player_db_locked_total", int64(len(lockedDBs)))
		slog.WarnContext(ctx, "spartan_cron: player DB(s) verrouillée(s) par un autre process — "+
			"un writer concurrent (CLI backfill / 2e instance serveur / Air pas encore libéré) tient le fichier RW ; "+
			"ces joueurs restent dégradés jusqu'à sa fermeture (DuckDB est mono-writer par fichier)",
			"titleSlug", titleSlug, "locked_players", lockedDBs, "count", len(lockedDBs))
	}
	slog.InfoContext(ctx, "spartan_cron: cycle done",
		"titleSlug", titleSlug, "players", len(players),
		"ok", succeeded, "skipped", skipped, "failed", failed, "locked", len(lockedDBs),
		"duration", time.Since(start))
}

type refreshOutcome int

const (
	refreshOK refreshOutcome = iota
	refreshSkipped
	refreshFailed
)

// refreshOne refresh la customisation d'UN joueur en posant ses tokens dans le ctx
// puis en deleguant au refresher du titre. Best-effort, ne bloque pas le cycle. La
// sequence COMMUNE (lookup pool, lease pinned, ctx auth, timeout) est ici ; le
// refresher ne fait que l'appel metier specifique au titre.
func (c *SpartanCustomizationCron) refreshOne(ctx context.Context, p domain.PlayerSummary, refresher CustomizationRefresher) (refreshOutcome, error) {
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

	// Construit un ctx avec les tokens du joueur pour que le refresher du titre
	// puisse appeler l'API Halo en son nom.
	playerCtx := ctxkeys.WithHaloAuth(ctx, lease.Tokens, p.XUID)
	playerCtx, cancel := context.WithTimeout(playerCtx, 30*time.Second)
	defer cancel()

	if err := refresher(playerCtx, p); err != nil {
		// Lock concurrent sur la player DB : pas de WARN par-joueur (la cause
		// racine est loggée une seule fois, agrégée, en fin de cycle par runOnceForTitle).
		if duckdb.IsFileLockError(err) {
			slog.DebugContext(ctx, "spartan_cron: refresher failed (player DB lock — agrégé en fin de cycle)",
				"gamertag", p.Gamertag, "err", err)
		} else {
			slog.WarnContext(ctx, "spartan_cron: refresher failed",
				"gamertag", p.Gamertag, "slug", p.TitleSlug, "err", err)
		}
		return refreshFailed, err
	}
	return refreshOK, nil
}
