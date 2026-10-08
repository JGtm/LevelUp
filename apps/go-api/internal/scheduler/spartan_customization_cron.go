// Package scheduler — spartan_customization_cron.go : cron leger qui
// rafraichit la customisation Spartan (banniere, emblem, backdrop,
// spartan_id / service tag) de tous les profils suivis toutes les N heures,
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
//   - La sequence COMMUNE (choix du token, lease pinned, ctx auth, timeout) reste ici :
//     le refresher recoit un ctx DEJA muni d'un token utilisable — celui du joueur,
//     ou celui d'un porteur quand le sien ne l'est pas (spartan_customization_bearer.go)
//     — avec le joueur comme sujet, et ne fait QUE l'appel metier specifique au titre.
//   - Le WIRING CONCRET des refreshers se fait dans cmd/server/main.go :
//   - halo_infinite -> CareerLiveService.GetSpartanIdentityFor(p.XUID) (chemin
//     live unifie, MEME path que la visite home → kickoffBackgroundRefresh →
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
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/platform/duckdb"
)

// SpartanIdentityFetcher abstrait CareerLiveService pour le mocking.
type SpartanIdentityFetcher interface {
	// GetSpartanIdentityFor résout l'identité du SUJET passé explicitement (finding
	// ID4) : le cron fournit le xuid du joueur rafraîchi (p.XUID), jamais une valeur
	// ambiante du contexte.
	GetSpartanIdentityFor(ctx context.Context, xuid string) (*domain.HomeSpartanIdentityRow, error)
}

// CareerLiveServiceProvider retourne un fetcher per-player.
// Implémenté par api.ServiceRegistry.CareerLiveCtx (signature adaptée).
type CareerLiveServiceProvider func(ctx context.Context, slug string) (SpartanIdentityFetcher, error)

// CustomizationRefresher rafraichit la customisation Spartan d'UN joueur d'UN
// titre donne. Le ctx fourni porte DEJA un token utilisable (celui du joueur, ou
// celui d'un porteur, cf. readerContext), le joueur comme sujet (ctxkeys.HaloXUID)
// et un timeout : le refresher ne fait QUE l'appel metier specifique au titre (live
// career identity, fetch appearance…), toujours pour p, jamais pour le porteur.
// Abstraction title-agnostic : le scheduler ne depend d'AUCUN package de titre ;
// chaque titre injecte son implementation au boot (cf. cmd/server/main.go).
type CustomizationRefresher func(ctx context.Context, p domain.PlayerSummary) error

// SpartanCustomizationCron itere sur tous les titres actifs et, pour chacun,
// sur tous ses profils suivis toutes les N heures, en appelant le refresher
// enregistre pour CE titre avec un token utilisable en context (celui du joueur,
// sinon celui d'un porteur, cf. acquireReaderToken). Cela declenche
// le path de rafraichissement propre au titre.
type SpartanCustomizationCron struct {
	cfg        *config.AppConfig
	pool       pool.Pool
	registry   *titlePkg.Registry
	refreshers map[string]CustomizationRefresher
	interval   time.Duration
	// accounts : comptes de l'instance, d'où le compte de l'utilisateur (rôle admin) que
	// le choix du porteur préfère. nil = aucun compte préféré (cf. WithAccounts).
	accounts AccountsReader
}

// DefaultSpartanCustomizationInterval est l'intervalle par defaut (8h)
// — choisi pour couvrir l'evolution Spartan ID/banniere d'un joueur actif
// sans saturer l'API Halo.
const DefaultSpartanCustomizationInterval = 8 * time.Hour

// NewSpartanCustomizationCron construit le cron. Si interval == 0,
// DefaultSpartanCustomizationInterval est utilise.
//
// titleSlug + svcProvider câblent le refresher du titre HISTORIQUE (Halo Infinite
// par defaut) : la closure appelle svcProvider(ctx, slug).GetSpartanIdentityFor(p.XUID).
// Les AUTRES titres (Halo 5+) s'enregistrent ensuite via WithRefresher (cf. main.go).
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
// appelle GetSpartanIdentityFor avec le SUJET EXPLICITE p.XUID (finding ID4)
// (→ kickoffBackgroundRefresh → persistPartial). Le ctx posé par refreshOne a p.XUID
// pour sujet (HaloXUID) quel que soit le compte qui porte le token : la persistance
// du chemin live (réservée au sujet == HaloXUID) écrit donc dans la base de p.
func careerIdentityRefresher(svcProvider CareerLiveServiceProvider) CustomizationRefresher {
	return func(ctx context.Context, p domain.PlayerSummary) error {
		svc, err := svcProvider(ctx, p.PlayerSlug)
		if err != nil {
			return err
		}
		_, err = svc.GetSpartanIdentityFor(ctx, p.XUID)
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

// WithAccounts branche les comptes de l'instance : quand le token d'un joueur est
// inutilisable, sa lecture est portée d'abord par le token du xuid lié à un compte de
// rôle admin (celui de l'utilisateur), par les autres comptes seulement à défaut.
// nil-safe. Le wiring (cmd/server) passe le store des comptes, *userstore.Store.
func (c *SpartanCustomizationCron) WithAccounts(r AccountsReader) *SpartanCustomizationCron {
	if c != nil && r != nil {
		c.accounts = r
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
	// Statut des crons (A6/DC-5, décision D1) : le cycle rapporte l'erreur AGRÉGÉE
	// réelle par titre — un cycle partiellement échoué = échec avec cause.
	start := time.Now()
	var errs []error
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
		if err := c.runOnceForTitle(ctx, desc.Slug); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", desc.Slug, err))
		}
	}
	observability.ReportCronRun("spartan_customization", start, errors.Join(errs...), time.Since(start).Milliseconds())
}

// runOnceForTitle execute un cycle pour tous les joueurs configures d'UN titre.
// Skip propre (slog Debug) si aucun refresher n'est enregistre pour ce titre —
// degradation gracieuse, jamais une erreur (le scheduler ne connait pas les
// titres concrets : un titre actif sans source de customisation est legitime).
func (c *SpartanCustomizationCron) runOnceForTitle(ctx context.Context, titleSlug string) error {
	refresher, ok := c.refreshers[titleSlug]
	if !ok {
		slog.DebugContext(ctx, "spartan_cron: titre sans refresher de customisation — skip",
			"titleSlug", titleSlug)
		return nil
	}

	start := time.Now()
	allPlayers, err := c.cfg.LoadPlayers(titleSlug)
	if err != nil {
		slog.ErrorContext(ctx, "spartan_cron: load players failed",
			"titleSlug", titleSlug, "err", err)
		return fmt.Errorf("load players: %w", err)
	}
	// Filtre CANONIQUE des chemins de refresh (domain.SyncablePlayers) : exclut les
	// titres en pause ET les profils AuthOnly. Un profil AuthOnly n'existe que pour
	// fournir des refresh tokens au pool (comptes DankerGlue/QuiteSiren/UppedJoker/
	// GeleJugefi/Trimbutton) : il n'a PAS de player DB (stats.duckdb absent), donc le
	// refresher échoue systématiquement en "No such file or directory" et polluait les
	// cycles d'un WARN par joueur. Le refresh de customisation est un chemin de refresh
	// au même titre que le sync : on applique le même filtre à la SOURCE.
	players := domain.SyncablePlayers(allPlayers)
	if skippedProfiles := len(allPlayers) - len(players); skippedProfiles > 0 {
		slog.DebugContext(ctx, "spartan_cron: profils non-refreshables ignorés (AuthOnly / titre en pause)",
			"titleSlug", titleSlug, "skipped_profiles", skippedProfiles)
	}

	var tally cycleTally
	var candidates []bearerCandidate
	if c.pool != nil && len(players) > 0 {
		candidates = c.bearerCandidates(ctx)
	}
	for _, p := range players {
		tally.record(p.Gamertag, c.refreshOne(ctx, p, refresher, candidates))
	}
	tally.reportLocked(ctx, titleSlug)
	slog.InfoContext(ctx, "spartan_cron: cycle done",
		"titleSlug", titleSlug, "players", len(players),
		"ok", tally.succeeded, "skipped", tally.skipped, "failed", tally.failed,
		"locked", len(tally.lockedDBs), "via_bearer", tally.viaBearer,
		"duration", time.Since(start))
	return tally.err(len(players))
}

type refreshOutcome int

const (
	refreshOK refreshOutcome = iota
	refreshSkipped
	refreshFailed
)

// refreshReport est l'issue du rafraîchissement d'un joueur. viaBearer dit si la
// lecture a été portée par le token d'un autre compte.
type refreshReport struct {
	outcome   refreshOutcome
	viaBearer bool
	err       error
}

// cycleTally est le bilan d'un cycle de rafraîchissement d'un titre.
type cycleTally struct {
	succeeded, skipped, failed, nonLockFailed, viaBearer int
	lockedDBs                                            []string
	firstNonLockErr                                      error
}

// record compte l'issue du rafraîchissement d'un joueur. Un verrou concurrent sur
// la player DB (second écrivain air/worktree/CLI) est une contention transitoire de
// poste de dev qui se résorbe seule : comptée à part, elle ne fait pas échouer le
// cycle. Toute autre cause, dont l'absence de tout token utilisable, est un échec
// réel (D1).
func (t *cycleTally) record(gamertag string, r refreshReport) {
	if r.viaBearer {
		t.viaBearer++
	}
	switch r.outcome {
	case refreshOK:
		t.succeeded++
	case refreshSkipped:
		t.skipped++
	case refreshFailed:
		t.failed++
		if duckdb.IsFileLockError(r.err) {
			t.lockedDBs = append(t.lockedDBs, gamertag)
			return
		}
		t.nonLockFailed++
		if t.firstNonLockErr == nil {
			t.firstNonLockErr = r.err
		}
	}
}

// reportLocked émet UNE ligne WARN agrégée par cycle pour les player DB tenues par
// un autre process (les lignes par joueur sont en Debug) et le compteur expvar
// associé : une cause connue et locale, pas un incident serveur.
func (t *cycleTally) reportLocked(ctx context.Context, titleSlug string) {
	if len(t.lockedDBs) == 0 {
		return
	}
	observability.AddInt("spartan_cron_player_db_locked_total", int64(len(t.lockedDBs)))
	slog.WarnContext(ctx, "spartan_cron: player DB(s) verrouillée(s) par un autre process — "+
		"un writer concurrent (CLI backfill / 2e instance serveur / Air pas encore libéré) tient le fichier RW ; "+
		"ces joueurs restent dégradés jusqu'à sa fermeture (DuckDB est mono-writer par fichier)",
		"titleSlug", titleSlug, "locked_players", t.lockedDBs, "count", len(t.lockedDBs))
}

// err rend l'échec du cycle : un échec partiel est un échec avec cause (D1), les
// échecs par verrou concurrent en sont exclus.
func (t *cycleTally) err(players int) error {
	if t.nonLockFailed == 0 {
		return nil
	}
	if t.firstNonLockErr != nil {
		return fmt.Errorf("%d/%d joueurs en échec de refresh (ex: %w)", t.nonLockFailed, players, t.firstNonLockErr)
	}
	return fmt.Errorf("%d/%d joueurs en échec de refresh", t.nonLockFailed, players)
}

// refreshOne rafraîchit la customisation d'UN joueur : choisit le token qui lira
// (le sien, sinon celui d'un porteur, cf. acquireReaderToken), le pose dans le ctx
// avec le joueur comme sujet, puis délègue au refresher du titre. Best-effort, ne
// bloque pas le cycle. Aucun profil suivi n'est sauté faute de token propre : seule
// l'absence de tout token utilisable empêche la lecture, et elle compte un échec.
func (c *SpartanCustomizationCron) refreshOne(
	ctx context.Context,
	p domain.PlayerSummary,
	refresher CustomizationRefresher,
	candidates []bearerCandidate,
) refreshReport {
	if c.pool == nil || p.XUID == "" || p.Gamertag == "" {
		return refreshReport{outcome: refreshSkipped}
	}
	lease, readerXUID, err := c.acquireReaderToken(ctx, p, candidates)
	if err != nil {
		slog.WarnContext(ctx, "spartan_cron: apparence non lue — aucun token utilisable",
			"gamertag", p.Gamertag, "xuid", p.XUID, "slug", p.TitleSlug, "err", err)
		return refreshReport{outcome: refreshFailed, err: err}
	}
	defer lease.Release()
	viaBearer := readerXUID != p.XUID

	playerCtx := readerContext(ctx, lease.Tokens, readerXUID, p.XUID)
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
				"gamertag", p.Gamertag, "slug", p.TitleSlug, "via_bearer", viaBearer, "err", err)
		}
		return refreshReport{outcome: refreshFailed, viaBearer: viaBearer, err: err}
	}
	return refreshReport{outcome: refreshOK, viaBearer: viaBearer}
}
