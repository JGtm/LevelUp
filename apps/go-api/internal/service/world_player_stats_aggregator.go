// Package service — world_player_stats_aggregator.go : agrégateur one-shot
// MULTI-TOKENS des stats joueur du classement mondial (Phase C,
// PLAN_WORLD_LEADERBOARD_ENRICHED.md).
//
// Pipeline par joueur :
//  1. résolution gamertag -> xuid (PeopleHub, single-token RTA — bas volume)
//  2. pagination GetMatchHistory (matchmaking) via le PooledHaloClient
//     (PolicyAnyPublic = round-robin sur TOUS les tokens du pool → parallélisme)
//  3. GetMatchStats par match -> extraction pure (analysis.ExtractPlayerMatchStat)
//  4. accumulation par (saison CSR, playlist) -> analysis.AccumulateWorldStats
//
// Le fan-out entre joueurs est borné par Concurrency ; le RPS effectif global est
// déjà plafonné par le pool (PerTokenRPS × nb tokens). La persistance
// (InsertPlayerSeasonStats) est laissée au caller (cron/CLI) pour garder
// l'agrégateur testable sans DB.
package service

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	syncpkg "levelup/go-api/internal/sync"
)

// worldEnrichMatchSkipped compte les matchs ignorés (erreur persistante GetMatchStats
// après retries) SANS perdre le reste des stats du joueur (B2 : hardening par-match).
// Exposé sur /debug/vars pour mesurer l'attrition résiduelle.
var worldEnrichMatchSkipped = expvar.NewInt("world_enrich.match_skipped")

const worldMatchPageSize = 25 // plafond API Halo GetMatchHistory

// retryDelays : backoff sur erreurs transitoires (429/503/réseau). Le pool
// déclenche déjà un cooldown global sur 429 mais ne retente pas l'appel courant
// (doPublic ne retente que les erreurs d'auth) → l'agrégateur retente ici, sinon
// un 429 ferait perdre tout le joueur. Laisse le temps au rate limit de récupérer.
var retryDelays = []time.Duration{2 * time.Second, 5 * time.Second, 12 * time.Second}

// Garantit que le client multi-tokens du pool satisfait la surface attendue.
var _ WorldMatchSource = (*syncpkg.PooledHaloClient)(nil)

// WorldMatchSource : sous-ensemble de la surface client suffisant pour
// l'agrégation. Satisfait par *syncpkg.PooledHaloClient (multi-tokens).
type WorldMatchSource interface {
	GetMatchHistory(ctx context.Context, gamertag, matchType string, start, count int) ([]syncpkg.MatchHistoryEntry, error)
	GetMatchStats(ctx context.Context, matchID string) (map[string]any, error)
}

// WorldXUIDResolver résout gamertag -> xuid numérique (PeopleHub, single-token).
// Satisfait par *auth.PeopleHubResolver.
type WorldXUIDResolver interface {
	ResolveXUID(ctx context.Context, gamertag string) (string, error)
}

// WorldStatsAggregatorConfig paramètre l'agrégation.
type WorldStatsAggregatorConfig struct {
	// TargetSeasons : saisons CSR normalisées à conserver (ex {"csrseason13-2":true}).
	// Vide = conserver toutes les saisons rencontrées (backfill complet, Phase D).
	TargetSeasons map[string]bool
	// MaxPages : plafond de pages d'historique par joueur. 0 = défaut 40 (~1000 matchs).
	MaxPages int
	// StopAfterNonTarget : arrête la pagination après N matchs consécutifs hors
	// TargetSeasons (historique chronologique décroissant → une fois sous les saisons
	// cibles on ne remonte plus). 0 = défaut 50. **Négatif = désactivé** (scan
	// jusqu'à MaxPages — requis pour backfiller une VIEILLE saison, sinon l'arrêt
	// se déclenche sur les matchs récents avant d'atteindre la cible). Ignoré si
	// TargetSeasons vide.
	StopAfterNonTarget int
	// Concurrency : nb de joueurs traités en parallèle. 0 = défaut 8.
	Concurrency int
	// RankedPlaylists : si non vide, SEULS les matchs dont la playlist y figure sont
	// accumulés. L'historique matchmaking mêle classé ET social ; le classement
	// mondial étant CSR (classé), on ignore tout match hors de cet ensemble.
	RankedPlaylists map[string]bool
	// XUIDResolveDelay : délai entre deux résolutions PeopleHub (gamertag->xuid)
	// dans PrepareWorldPlayers. PeopleHub limite ~10 req/15s PAR compte (single-token)
	// → sans délai, résoudre 200+ joueurs d'affilée déclenche des 429 qui skippent les
	// joueurs en masse. 0 = pas de throttle (tests bas volume) ; le CLI/cron met ~1.6s.
	XUIDResolveDelay time.Duration
	// SeasonStart / SeasonEnd : fenêtre de dates de la saison CSR cible (depuis le
	// calendrier csr_placement_thresholds). Quand renseignée, collectPlayerMatches
	// FILTRE l'historique par StartTime AVANT de fetcher : il saute les matchs hors
	// fenêtre sans les fetcher. C'est LE fix des vieilles saisons — sinon on fetch le
	// match COMPLET de chaque match (des milliers de récents) juste pour lire sa saison,
	// et on n'atteint jamais la saison profonde. Zéro = pas de filtre date (fallback
	// historique : saison courante en tête, ou pas de calendrier).
	SeasonStart, SeasonEnd time.Time
	// ProbeDelay : délai entre deux sondes de la recherche dichotomique de l'offset
	// (findWindowStartOffset). Les ~log2(profondeur) sondes count=1 partent en rafale
	// au démarrage de CHAQUE joueur → elles brûlent le burst halostats (~10 req/15s) et
	// déclenchent des 429. Un léger espacement lisse la rafale. 0 = pas de throttle
	// (tests bas volume) ; le CLI met ~350ms.
	ProbeDelay time.Duration
}

func (c *WorldStatsAggregatorConfig) withDefaults() {
	if c.MaxPages <= 0 {
		c.MaxPages = 40
	}
	if c.StopAfterNonTarget == 0 {
		c.StopAfterNonTarget = 50 // négatif laissé tel quel = désactivé (backfill)
	}
	if c.Concurrency <= 0 {
		c.Concurrency = 8
	}
}

// cachedMatch : un match déjà fetché (GetMatchStats), extrait pour TOUS ses
// participants (indexé par xuid). Permet de ne fetcher chaque match qu'UNE fois même
// s'il concerne plusieurs de nos cibles (les tops s'affrontent en permanence) — et,
// l'extraction n'étant plus filtrée par un ensemble pré-résolu, l'attribution est
// indépendante de l'ordre où les joueurs sont résolus (cf. résolution paresseuse).
type cachedMatch struct {
	season  string
	players map[string]analysis.PlayerMatchStat // xuid -> stat (tous les participants)
}

// WorldStatsAggregator agrège les stats brutes par (saison, playlist) pour un
// ensemble de joueurs, via un client multi-tokens. Un cache de matchs partagé
// (mu/cache) dédoublonne les GetMatchStats entre joueurs concurrents.
type WorldStatsAggregator struct {
	src      WorldMatchSource
	resolver WorldXUIDResolver
	cfg      WorldStatsAggregatorConfig

	mu    sync.Mutex
	cache map[string]cachedMatch // matchID -> match extrait pour TOUS ses participants (dédup fetch)
	// xuidByGamertag : cache des résolutions gamertag->xuid (dédup intra-run). Rempli
	// paresseusement par AggregatePlayer (1 résolution/joueur, au moment où on le traite)
	// ou en amont par PrepareWorldPlayers (warm-up optionnel, utilisé par Run).
	xuidByGamertag map[string]string
	sf             singleflight.Group // dédup STRICT des fetchs concurrents par matchID
}

// NewWorldStatsAggregator construit l'agrégateur. `src` doit être un client
// multi-tokens (typiquement *syncpkg.PooledHaloClient construit avec PolicyAnyPublic).
func NewWorldStatsAggregator(src WorldMatchSource, resolver WorldXUIDResolver, cfg WorldStatsAggregatorConfig) *WorldStatsAggregator {
	cfg.withDefaults()
	return &WorldStatsAggregator{
		src: src, resolver: resolver, cfg: cfg,
		cache:          map[string]cachedMatch{},
		xuidByGamertag: map[string]string{},
	}
}

// SeedKnownXUIDs pré-remplit le cache gamertag->xuid avec des correspondances DÉJÀ
// connues (typiquement les xuid scrapés du snapshot Waypoint, cf. B1). Court-circuite
// la résolution PeopleHub dans PrepareWorldPlayers ET AggregatePlayer : le résolveur
// n'est appelé QUE pour les gamertags sans xuid connu (lignes de snapshot antérieures
// à la persistance du xuid). Les entrées vides sont ignorées ; une correspondance déjà
// présente n'est pas écrasée. Retourne le nombre de xuid effectivement seedés.
func (a *WorldStatsAggregator) SeedKnownXUIDs(byGamertag map[string]string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	n := 0
	for gt, xuid := range byGamertag {
		if gt == "" || xuid == "" {
			continue
		}
		if _, ok := a.xuidByGamertag[gt]; ok {
			continue
		}
		a.xuidByGamertag[gt] = xuid
		n++
	}
	return n
}

// PrepareWorldPlayers résout EN AMONT les xuid de tous les gamertags cibles (warm-up
// OPTIONNEL). L'extraction d'un match récupère désormais TOUS ses participants, donc
// l'attribution ne dépend plus d'un ensemble pré-résolu : ce batch n'est plus requis
// pour la correction (AggregatePlayer résout paresseusement). Il reste utilisé par
// Run (résolution groupée + collecte des erreurs en amont). Le backfill CLI, lui,
// privilégie l'opération lourde (le fetch) et résout joueur par joueur sans bloquer.
// Best-effort : retourne les erreurs de résolution.
func (a *WorldStatsAggregator) PrepareWorldPlayers(ctx context.Context, gamertags []string) []error {
	var errs []error
	first := true
	for _, gt := range gamertags {
		a.mu.Lock()
		_, done := a.xuidByGamertag[gt]
		a.mu.Unlock()
		if done {
			continue
		}
		// Throttle PeopleHub (~10 req/15s/compte single-token) : espacer les
		// résolutions évite les 429 qui skippent des joueurs en masse. 1er appel
		// immédiat. XUIDResolveDelay=0 (tests) → pas d'attente.
		if !first && a.cfg.XUIDResolveDelay > 0 {
			select {
			case <-ctx.Done():
				return errs
			case <-time.After(a.cfg.XUIDResolveDelay):
			}
		}
		first = false
		xuid, err := a.resolver.ResolveXUID(ctx, gt)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", gt, err))
			continue
		}
		a.mu.Lock()
		a.xuidByGamertag[gt] = xuid
		a.mu.Unlock()
	}
	return errs
}

// AggregatePlayer collecte l'historique du joueur et accumule ses stats par
// (saison, playlist). Résout le xuid PARESSEUSEMENT (cache intra-run + résolveur) :
// pour fetcher les matchs de X il suffit du xuid de X, donc on ne bloque pas sur la
// résolution des autres. L'attribution depuis un match déjà caché reste correcte même
// si X est résolu APRÈS coup : le cache contient la stat de TOUS les participants.
func (a *WorldStatsAggregator) AggregatePlayer(ctx context.Context, gamertag string) ([]domain.WorldPlayerSeasonStats, error) {
	a.mu.Lock()
	xuid, ok := a.xuidByGamertag[gamertag]
	a.mu.Unlock()
	if !ok {
		x, err := a.resolver.ResolveXUID(ctx, gamertag)
		if err != nil {
			return nil, fmt.Errorf("resolve xuid %q: %w", gamertag, err)
		}
		a.mu.Lock()
		a.xuidByGamertag[gamertag] = x
		a.mu.Unlock()
		xuid = x
	}
	stats, err := a.collectPlayerMatches(ctx, xuid)
	if err != nil {
		return nil, err
	}
	return analysis.AccumulateWorldStats(ctxkeys.TitleSlug(ctx), gamertag, stats), nil
}

// getMatch retourne le match (extrait pour tous les joueurs mondiaux) depuis le
// cache partagé, ou le fetch UNE fois (GetMatchStats) puis le met en cache. Une
// rare double-récupération concurrente reste correcte (chaque joueur lit SA stat).
func (a *WorldStatsAggregator) getMatch(ctx context.Context, matchID string) (cachedMatch, error) {
	a.mu.Lock()
	cm, ok := a.cache[matchID]
	a.mu.Unlock()
	if ok {
		return cm, nil
	}
	// singleflight : un seul fetch par matchID même si N joueurs le demandent en
	// parallèle ; les autres attendent et lisent le même résultat.
	v, err, _ := a.sf.Do(matchID, func() (any, error) {
		a.mu.Lock()
		if cm, ok := a.cache[matchID]; ok {
			a.mu.Unlock()
			return cm, nil
		}
		a.mu.Unlock()
		var raw map[string]any
		if e := a.withRetry(ctx, func() error {
			var e error
			raw, e = a.src.GetMatchStats(ctx, matchID)
			return e
		}); e != nil {
			return cachedMatch{}, fmt.Errorf("match stats %s: %w", matchID, e)
		}
		// nil = extraire TOUS les participants : le cache devient indépendant de
		// l'ordre de résolution (résolution paresseuse joueur par joueur).
		season, players := analysis.ExtractWorldPlayersFromMatch(raw, nil)
		fresh := cachedMatch{season: season, players: players}
		a.mu.Lock()
		a.cache[matchID] = fresh
		a.mu.Unlock()
		return fresh, nil
	})
	if err != nil {
		return cachedMatch{}, err
	}
	return v.(cachedMatch), nil
}

// collectPlayerMatches pagine l'historique matchmaking et collecte la stat du
// joueur via le cache de matchs partagé (1 fetch / match, jusqu'à 8 joueurs
// traités d'un coup). Un set local de matchs vus élimine le double-comptage par
// overlap de pagination. S'arrête à MaxPages, fin d'historique, ou StopAfterNonTarget.
func (a *WorldStatsAggregator) collectPlayerMatches(ctx context.Context, xuid string) ([]analysis.PlayerMatchStat, error) {
	player := "xuid(" + xuid + ")"
	var collected []analysis.PlayerMatchStat
	seen := map[string]bool{} // dédup overlap de pagination (intra-joueur)
	nonTarget := 0
	belowWindow := 0 // matchs consécutifs SOUS la fenêtre date (→ arrêt)
	// DICHOTOMIE : si on connaît la fenêtre de la saison, on saute DIRECTEMENT à
	// l'offset où l'historique entre dans la fenêtre (~log requêtes) au lieu de
	// paginer linéairement des centaines de pages de matchs récents pour atteindre
	// une vieille saison. Le filtre date dans la boucle gère ensuite les bords précis.
	startPage := 0
	if !a.cfg.SeasonStart.IsZero() && !a.cfg.SeasonEnd.IsZero() {
		off, err := a.findWindowStartOffset(ctx, player, a.cfg.MaxPages*worldMatchPageSize)
		if err != nil {
			if ctx.Err() != nil {
				return collected, ctx.Err()
			}
			// B2 hardening : la dichotomie a échoué (API) — on retombe sur le scan
			// linéaire (startPage=0) au lieu de perdre tout le joueur.
			slog.DebugContext(ctx, "collectPlayerMatches: dichotomie offset échouée — scan linéaire",
				"xuid", xuid, "err", err)
		} else {
			startPage = off / worldMatchPageSize
		}
	}
	for page := startPage; page < a.cfg.MaxPages; page++ {
		if err := ctx.Err(); err != nil {
			return collected, err
		}
		var hist []syncpkg.MatchHistoryEntry
		err := a.withRetry(ctx, func() error {
			var e error
			hist, e = a.src.GetMatchHistory(ctx, player, "matchmaking", page*worldMatchPageSize, worldMatchPageSize)
			return e
		})
		if err != nil {
			if ctx.Err() != nil {
				return collected, ctx.Err()
			}
			// B2 hardening : erreur d'historique (après retries 429/503). Si on a DÉJÀ
			// collecté des matchs, on arrête la pagination mais on CONSERVE le partiel
			// (une page en échec n'annule plus les précédentes). Si rien n'a été collecté
			// (échec dès la 1re page tentée), on remonte l'erreur — signal préservé.
			if len(collected) > 0 {
				slog.WarnContext(ctx, "collectPlayerMatches: GetMatchHistory échoué — pagination arrêtée, stats partielles conservées",
					"xuid", xuid, "page", page, "err", err)
				break
			}
			return collected, fmt.Errorf("match history xuid(%s) page %d: %w", xuid, page, err)
		}
		if len(hist) == 0 {
			break
		}
		for _, h := range hist {
			if seen[h.MatchID] {
				continue
			}
			seen[h.MatchID] = true
			// FILTRE DATE (le fix vieilles saisons) : l'entrée d'historique porte le
			// StartTime du match. Si on connaît la fenêtre de la saison cible, on saute
			// les matchs hors fenêtre SANS les fetcher — au lieu de fetcher le match
			// complet de chaque match juste pour lire sa saison (ce qui faisait fetcher
			// des milliers de matchs récents et ne jamais atteindre la saison profonde).
			if mt, ok := parseMatchStart(h.StartTime); ok {
				if !a.cfg.SeasonEnd.IsZero() && mt.After(a.cfg.SeasonEnd) {
					continue // trop récent : pas encore entré dans la fenêtre
				}
				if !a.cfg.SeasonStart.IsZero() && mt.Before(a.cfg.SeasonStart) {
					// Historique chronologique décroissant : une fois SOUS la fenêtre on
					// n'y remontera plus. Une page entière sous la fenêtre = fini.
					belowWindow++
					if belowWindow >= worldMatchPageSize {
						return collected, nil
					}
					continue // trop vieux
				}
				belowWindow = 0
			}
			cm, err := a.getMatch(ctx, h.MatchID)
			if err != nil {
				if ctx.Err() != nil {
					return collected, ctx.Err()
				}
				// B2 hardening : un match illisible (403/404/timeout après retries) est
				// IGNORÉ — on n'annule plus tout le joueur pour un seul match. C'est LE
				// fix des trous d'enrichissement (un 403 ne fait plus perdre 366 matchs).
				worldEnrichMatchSkipped.Add(1)
				slog.DebugContext(ctx, "collectPlayerMatches: getMatch échoué — match ignoré",
					"match", h.MatchID, "err", err)
				continue
			}
			if len(a.cfg.TargetSeasons) > 0 && !a.cfg.TargetSeasons[cm.season] {
				nonTarget++
				continue
			}
			nonTarget = 0
			if st, present := cm.players[xuid]; present {
				// Ranked-only : on ignore les matchs social/non-classés (la playlist
				// n'est pas dans l'ensemble classé). nonTarget déjà reset car le match
				// est bien dans la saison cible (juste pas classé).
				if len(a.cfg.RankedPlaylists) == 0 || a.cfg.RankedPlaylists[st.PlaylistID] {
					collected = append(collected, st)
				}
			}
		}
		if a.cfg.StopAfterNonTarget > 0 && len(a.cfg.TargetSeasons) > 0 && nonTarget >= a.cfg.StopAfterNonTarget {
			break
		}
		if len(hist) < worldMatchPageSize {
			break
		}
	}
	return collected, nil
}

// findWindowStartOffset trouve par DICHOTOMIE le plus petit offset d'historique dont
// le match a StartTime <= SeasonEnd — c.-à-d. la 1re entrée dans la fenêtre de la
// saison (l'historique étant récent-d'abord, StartTime décroît avec l'offset). Évite
// de paginer linéairement tous les matchs récents pour atteindre une vieille saison :
// ~log2(maxOffset) requêtes au lieu de ~(profondeur/25) pages. Borné à maxOffset
// (fenêtre au-delà du scan → offset = maxOffset → boucle vide → 0 collecté, attendu).
// Sonde GetMatchHistory(start=mid, count=1) : 1 requête légère par étape. Un léger
// délai (cfg.ProbeDelay) espace les sondes pour ne pas brûler le burst halostats.
func (a *WorldStatsAggregator) findWindowStartOffset(ctx context.Context, player string, maxOffset int) (int, error) {
	lo, hi := 0, maxOffset
	first := true
	for lo < hi {
		if err := ctx.Err(); err != nil {
			return lo, err
		}
		// Throttle : 1re sonde immédiate, puis espacement. Évite la rafale de
		// ~log2(profondeur) count=1 qui déclenchait des 429 au démarrage du joueur.
		if !first && a.cfg.ProbeDelay > 0 {
			select {
			case <-ctx.Done():
				return lo, ctx.Err()
			case <-time.After(a.cfg.ProbeDelay):
			}
		}
		first = false
		mid := (lo + hi) / 2
		var hist []syncpkg.MatchHistoryEntry
		if e := a.withRetry(ctx, func() error {
			var err error
			hist, err = a.src.GetMatchHistory(ctx, player, "matchmaking", mid, 1)
			return err
		}); e != nil {
			return 0, e
		}
		if len(hist) == 0 {
			hi = mid // au-delà de la fin de l'historique → la fenêtre est plus haut
			continue
		}
		mt, ok := parseMatchStart(hist[0].StartTime)
		if !ok {
			return lo, nil // date illisible → scan linéaire de secours depuis lo
		}
		if mt.After(a.cfg.SeasonEnd) {
			lo = mid + 1 // match trop récent → la fenêtre est plus profonde
		} else {
			hi = mid // <= SeasonEnd → candidat, chercher un offset plus petit (bord haut)
		}
	}
	return lo, nil
}

// parseMatchStart parse le StartTime d'un match (RFC3339, avec ou sans nanosecondes).
// ok=false si vide/format inconnu → le caller ne filtre alors pas par date (fallback sûr).
func parseMatchStart(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// withRetry exécute fn en retentant sur erreur transitoire (429/503/réseau) avec
// backoff (retryDelays). S'arrête immédiatement sur succès, erreur non transitoire,
// ou ctx annulé. Indispensable au backfill : le pool ne retente pas les 429.
func (a *WorldStatsAggregator) withRetry(ctx context.Context, fn func() error) error {
	var err error
	for attempt := 0; ; attempt++ {
		err = fn()
		if err == nil || ctx.Err() != nil || !isTransientErr(err) || attempt >= len(retryDelays) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelays[attempt]):
		}
	}
}

// isTransientErr classe une erreur de fetch comme retentable : HTTP 429/503/5xx,
// ou erreur réseau (non-HTTP) hors annulation de contexte.
func isTransientErr(err error) bool {
	if err == nil {
		return false
	}
	var he *syncpkg.HTTPError
	if errors.As(err, &he) {
		return he.StatusCode == 429 || he.StatusCode >= 500
	}
	return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}

// Run résout d'abord tous les xuid (PrepareWorldPlayers → extraction multi-joueurs
// par match), puis agrège chaque gamertag en parallèle (borné par Concurrency) en
// partageant le cache de matchs. Un joueur en échec n'interrompt pas le batch
// (erreur collectée dans `errs`, best-effort).
func (a *WorldStatsAggregator) Run(ctx context.Context, gamertags []string) ([]domain.WorldPlayerSeasonStats, []error) {
	type playerResult struct {
		stats []domain.WorldPlayerSeasonStats
		err   error
		gt    string
	}
	errs := a.PrepareWorldPlayers(ctx, gamertags)
	results := make([]playerResult, len(gamertags))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(a.cfg.Concurrency)
	for i, gt := range gamertags {
		i, gt := i, gt
		a.mu.Lock()
		_, resolved := a.xuidByGamertag[gt]
		a.mu.Unlock()
		if !resolved {
			continue // xuid non résolu (déjà dans errs via PrepareWorldPlayers)
		}
		g.Go(func() error {
			s, err := a.AggregatePlayer(gctx, gt)
			results[i] = playerResult{stats: s, err: err, gt: gt}
			return nil // best-effort : on ne propage pas l'échec d'un joueur
		})
	}
	_ = g.Wait()

	var all []domain.WorldPlayerSeasonStats
	for _, r := range results {
		if r.gt == "" {
			continue // slot non traité (xuid non résolu)
		}
		if r.err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", r.gt, r.err))
			continue
		}
		all = append(all, r.stats...)
	}
	return all, errs
}
