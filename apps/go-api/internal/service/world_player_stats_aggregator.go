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
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	syncpkg "levelup/go-api/internal/sync"
)

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

// cachedMatch : un match déjà fetché (GetMatchStats), extrait pour TOUS les joueurs
// mondiaux présents. Permet de ne fetcher chaque match qu'UNE fois même s'il
// concerne jusqu'à 8 de nos cibles (les tops s'affrontent en permanence).
type cachedMatch struct {
	season  string
	players map[string]analysis.PlayerMatchStat // xuid -> stat (joueurs mondiaux présents)
}

// WorldStatsAggregator agrège les stats brutes par (saison, playlist) pour un
// ensemble de joueurs, via un client multi-tokens. Un cache de matchs partagé
// (mu/cache) dédoublonne les GetMatchStats entre joueurs concurrents.
type WorldStatsAggregator struct {
	src      WorldMatchSource
	resolver WorldXUIDResolver
	cfg      WorldStatsAggregatorConfig

	mu             sync.Mutex
	cache          map[string]cachedMatch // matchID -> match extrait (dédup fetch)
	worldXuids     map[string]bool        // xuid des joueurs cibles (extraction multi)
	xuidByGamertag map[string]string      // gamertag -> xuid (pré-résolu, PrepareWorldPlayers)
	sf             singleflight.Group     // dédup STRICT des fetchs concurrents par matchID
}

// NewWorldStatsAggregator construit l'agrégateur. `src` doit être un client
// multi-tokens (typiquement *syncpkg.PooledHaloClient construit avec PolicyAnyPublic).
func NewWorldStatsAggregator(src WorldMatchSource, resolver WorldXUIDResolver, cfg WorldStatsAggregatorConfig) *WorldStatsAggregator {
	cfg.withDefaults()
	return &WorldStatsAggregator{
		src: src, resolver: resolver, cfg: cfg,
		cache:          map[string]cachedMatch{},
		worldXuids:     map[string]bool{},
		xuidByGamertag: map[string]string{},
	}
}

// PrepareWorldPlayers résout EN AMONT les xuid de tous les gamertags cibles et
// alimente l'ensemble worldXuids — indispensable pour que l'extraction d'un match
// récupère TOUS les joueurs mondiaux présents (et pas juste celui en cours).
// À appeler avant AggregatePlayer. Best-effort : retourne les erreurs de résolution.
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
		a.worldXuids[xuid] = true
		a.mu.Unlock()
	}
	return errs
}

// AggregatePlayer collecte l'historique du joueur et accumule ses stats par
// (saison, playlist). L'xuid doit avoir été pré-résolu (PrepareWorldPlayers) ;
// sinon il est résolu à la volée (le joueur ne profitera pas des matchs déjà
// cachés AVANT sa résolution — d'où l'intérêt du Prepare amont).
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
		a.worldXuids[x] = true
		a.mu.Unlock()
		xuid = x
	}
	stats, err := a.collectPlayerMatches(ctx, xuid)
	if err != nil {
		return nil, err
	}
	return analysis.AccumulateWorldStats(gamertag, stats), nil
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
		season, players := analysis.ExtractWorldPlayersFromMatch(raw, a.worldXuids)
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
	for page := 0; page < a.cfg.MaxPages; page++ {
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
				return collected, err
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
