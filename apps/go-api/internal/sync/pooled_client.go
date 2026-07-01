package sync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/time/rate"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/platform/auth/pool"
)

// defaultPooledRPS est le RPS fallback pour PooledHaloClient quand le pool
// n'expose pas de Lease.Limiter (cas des mocks/tests). En prod, le pool fournit
// son propre limiter par-token (Option 2 de l'audit 2026-05-21) — voir
// internal/platform/auth/pool/types.go::Lease.Limiter et PoolOptions.PerTokenRPS.
const defaultPooledRPS = 5

// PooledHaloClient implémente HaloClient en utilisant un pool de tokens partagés.
// Chaque appel Acquire() crée un HaloAPIClient avec un token frais du pool.
// PolicyAnyPublic pour endpoints publics, PolicyPinnedPlayer pour endpoints privacy.
//
// Rate-limiting (Option 2 de l'audit 2026-05-21) : chaque slot du pool a son
// propre *rate.Limiter (PerTokenRPS) — throughput global = PerTokenRPS × Size().
// newAPIClient() consulte Lease.Limiter à chaque requête ; si nil (mocks),
// fallback sur fallbackLimiter local.
type PooledHaloClient struct {
	p pool.Pool

	// Si non-vide : token pinned sur ce gamertag (pour endpoints privacy).
	pinnedGamertag string
	pinnedXUID     string

	// fallbackLimiter est utilisé uniquement quand Lease.Limiter est nil
	// (mocks de test). En prod, le pool fournit le limiter par-token.
	fallbackLimiter *rate.Limiter

	// localFilmCache (optionnel) — propage au HaloAPIClient construit par Acquire.
	localFilmCache *LocalFilmCache
}

// NewPooledHaloClient crée un client pooled.
// pinnedGamertag/pinnedXUID : si non-vides, les endpoints privacy utilisent ce token (ex: GetCareerRank).
// requestsPerSecond : utilisé uniquement comme fallback si le pool ne fournit
// pas de Lease.Limiter (≤ 0 → defaultPooledRPS). En prod, c'est PerTokenRPS
// configuré à NewPool() qui pilote le throughput.
func NewPooledHaloClient(p pool.Pool, pinnedGamertag, pinnedXUID string, requestsPerSecond int) *PooledHaloClient {
	if requestsPerSecond <= 0 {
		requestsPerSecond = defaultPooledRPS
	}
	return &PooledHaloClient{
		p:               p,
		pinnedGamertag:  pinnedGamertag,
		pinnedXUID:      pinnedXUID,
		fallbackLimiter: rate.NewLimiter(rate.Limit(requestsPerSecond), 1),
	}
}

// WithLocalFilmCache active le cache film local pour ce client poolé.
func (pc *PooledHaloClient) WithLocalFilmCache(cache *LocalFilmCache) *PooledHaloClient {
	pc.localFilmCache = cache
	return pc
}

// newAPIClient construit un HaloAPIClient éphémère qui utilise le rate-limiter
// du slot (lease.Limiter, Option 2). Fallback sur fallbackLimiter si nil.
func (pc *PooledHaloClient) newAPIClient(lease *pool.Lease) *HaloAPIClient {
	limiter := lease.Limiter
	if limiter == nil {
		limiter = pc.fallbackLimiter
	}
	c := NewHaloAPIClient(lease.Tokens.SpartanToken, lease.Tokens.ClearanceToken, defaultPooledRPS).
		WithLimiter(limiter)
	if pc.localFilmCache != nil {
		c = c.WithLocalFilmCache(pc.localFilmCache)
	}
	return c
}

// isAuthError indique si err est un refus d'authentification Halo (401/403).
func isAuthError(err error) bool {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == 401 || httpErr.StatusCode == 403
	}
	return false
}

// notifyPoolOnError signale au pool les erreurs HTTP qui nécessitent une action :
//   - 429 → cooldown PER-TOKEN (On429ForToken) sur le SLOT fautif (lease.Gamertag)
//     — les autres tokens continuent de servir (fini le scorched-earth global).
//   - 503 → cooldown GLOBAL (OnHTTPError) — signal serveur, tout le pool en pause.
//   - 401/403 → le SLOT fautif (lease.Gamertag) est marqué unhealthy
//     (MarkUnhealthy déclenche un Resolver.Refresh async) et le pool le skip.
//
// RC-1 (2026-06-04) : avant, seuls 429/503 étaient gérés et le 401/403 était
// ignoré — un token expiré/révoqué était re-servi en boucle (sync 18:19 →
// matches_inserted:0). On marque le bon slot via lease.Gamertag, JAMAIS le
// pinnedGamertag (vide pour les appels PolicyAnyPublic round-robin).
func (pc *PooledHaloClient) notifyPoolOnError(lease *pool.Lease, err error) {
	if err == nil {
		return
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		return
	}
	switch httpErr.StatusCode {
	case 429:
		// Rate-limit imputable AU token courant (lease.Gamertag) → cooldown
		// per-token, PAS de scorched-earth global. Les autres tokens continuent.
		// Sans gamertag (rare), On429ForToken retombe sur le filet global.
		gt := ""
		if lease != nil {
			gt = lease.Gamertag
		}
		slog.DebugContext(context.Background(), "pooled: 429 signalé (cooldown per-token)",
			"gamertag", gt, "retry_after_s", httpErr.RetryAfter.Seconds())
		pc.p.On429ForToken(gt, httpErr.RetryAfter)
	case 503:
		// 503 = signal SERVEUR (indisponibilité globale) → cooldown global légitime.
		slog.DebugContext(context.Background(), "pooled: 503 signalé (cooldown global)",
			"retry_after_s", httpErr.RetryAfter.Seconds())
		pc.p.OnHTTPError(503, httpErr.RetryAfter)
	case 401, 403:
		gt := ""
		if lease != nil {
			gt = lease.Gamertag
		}
		slog.WarnContext(context.Background(), "pooled: token refusé (auth) — slot marqué unhealthy + refresh",
			"statusCode", httpErr.StatusCode, "gamertag", gt)
		if gt != "" {
			pc.p.MarkUnhealthy(gt, err)
		}
	}
}

// doPublic exécute un appel sur un endpoint public (PolicyAnyPublic) avec
// recovery auth : sur 401/403, le slot fautif est marqué unhealthy
// (notifyPoolOnError) puis l'appel est retenté UNE seule fois avec un autre
// token (round-robin du pool). Borné à 2 tentatives — au-delà, on remonte la
// dernière erreur (token réellement révoqué, ou pool entièrement malsain).
func (pc *PooledHaloClient) doPublic(ctx context.Context, call func(c *HaloAPIClient) error) error {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		lease, err := pc.p.Acquire(ctx, pool.PolicyAnyPublic, "")
		if err != nil {
			return fmt.Errorf("pooled: Acquire failed: %w", err)
		}
		callErr := call(pc.newAPIClient(lease))
		pc.notifyPoolOnError(lease, callErr)
		lease.Release()
		if callErr == nil || !isAuthError(callErr) {
			return callErr
		}
		lastErr = callErr
	}
	return lastErr
}

// GetMatchHistory implémente HaloClient.GetMatchHistory() avec PolicyAnyPublic.
func (pc *PooledHaloClient) GetMatchHistory(
	ctx context.Context,
	gamertag, matchType string,
	start, count int,
) ([]MatchHistoryEntry, error) {
	callStart := time.Now()
	var result []MatchHistoryEntry
	err := pc.doPublic(ctx, func(c *HaloAPIClient) error {
		var e error
		result, e = c.GetMatchHistory(ctx, gamertag, matchType, start, count)
		return e
	})
	observeHaloCall(ctxkeys.TitleSlug(ctx), "match_history", gamertag, callStart, err)
	return result, err
}

// GetMatchStats implémente HaloClient.GetMatchStats() avec PolicyAnyPublic.
func (pc *PooledHaloClient) GetMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	callStart := time.Now()
	var result map[string]any
	err := pc.doPublic(ctx, func(c *HaloAPIClient) error {
		var e error
		result, e = c.GetMatchStats(ctx, matchID)
		return e
	})
	observeHaloCall(ctxkeys.TitleSlug(ctx), "match_stats", "", callStart, err)
	return result, err
}

// GetMatchSkill implémente HaloClient.GetMatchSkill() avec PolicyAnyPublic.
func (pc *PooledHaloClient) GetMatchSkill(
	ctx context.Context,
	matchID string,
	xuids []string,
) (map[string]*MatchSkillData, error) {
	callStart := time.Now()
	var result map[string]*MatchSkillData
	err := pc.doPublic(ctx, func(c *HaloAPIClient) error {
		var e error
		result, e = c.GetMatchSkill(ctx, matchID, xuids)
		return e
	})
	observeHaloCall(ctxkeys.TitleSlug(ctx), "match_skill", "", callStart, err)
	return result, err
}

// GetMatchFilm implémente HaloClient.GetMatchFilm() avec PolicyAnyPublic.
func (pc *PooledHaloClient) GetMatchFilm(ctx context.Context, matchID string) (map[int]filmChunkData, bool, error) {
	callStart := time.Now()
	var result map[int]filmChunkData
	var ok bool
	err := pc.doPublic(ctx, func(c *HaloAPIClient) error {
		var e error
		result, ok, e = c.GetMatchFilm(ctx, matchID)
		return e
	})
	observeHaloCall(ctxkeys.TitleSlug(ctx), "film", "", callStart, err)
	return result, ok, err
}

// GetHighlightEventsChunk implémente HaloClient.GetHighlightEventsChunk() avec PolicyAnyPublic.
func (pc *PooledHaloClient) GetHighlightEventsChunk(ctx context.Context, matchID string) ([]byte, int, bool, error) {
	callStart := time.Now()
	var result []byte
	var ver int
	var ok bool
	err := pc.doPublic(ctx, func(c *HaloAPIClient) error {
		var e error
		result, ver, ok, e = c.GetHighlightEventsChunk(ctx, matchID)
		return e
	})
	observeHaloCall(ctxkeys.TitleSlug(ctx), "film_chunk", "", callStart, err)
	return result, ver, ok, err
}

// GetCareerRank implémente HaloClient.GetCareerRank() avec PolicyPinnedPlayer.
// Retourne (nil, nil) si le token pinned est absent ou si la requête est 401/403 (privacy-gated).
// Note : HaloAPIClient.GetCareerRank gère déjà le silent-skip 401/403 en interne.
func (pc *PooledHaloClient) GetCareerRank(ctx context.Context, xuid string) (*CareerRankData, error) {
	// Si pas de token pinned, silent-skip.
	if pc.pinnedGamertag == "" {
		slog.DebugContext(ctx, "pooled: GetCareerRank skipped (no pinned token)",
			"xuid", xuid)
		return nil, nil
	}

	lease, err := pc.p.Acquire(ctx, pool.PolicyPinnedPlayer, pc.pinnedGamertag)
	if err != nil {
		// Token malsain ou absent → silent-skip (comportement identique à halo_client.go:434).
		slog.DebugContext(ctx, "pooled: GetCareerRank skipped (token unavailable)",
			"xuid", xuid, "gamertag", pc.pinnedGamertag, "err", err)
		return nil, nil
	}
	defer lease.Release()

	client := pc.newAPIClient(lease)
	// HaloAPIClient handles 401/403 internally (returns nil, nil)
	callStart := time.Now()
	rank, rankErr := client.GetCareerRank(ctx, xuid)
	observeHaloCall(ctxkeys.TitleSlug(ctx), "career_rank", xuid, callStart, rankErr)
	return rank, rankErr
}

// GetPlayerCSRs implémente HaloClient.GetPlayerCSRs() avec PolicyAnyPublic.
// Endpoint public (service token) — pas besoin du token pinned joueur.
func (pc *PooledHaloClient) GetPlayerCSRs(ctx context.Context, xuid, seasonID string) ([]PlayerPlaylistCSR, error) {
	lease, err := pc.p.Acquire(ctx, pool.PolicyAnyPublic, "")
	if err != nil {
		slog.DebugContext(ctx, "pooled: GetPlayerCSRs skipped (token unavailable)", "xuid", xuid, "err", err)
		return nil, nil
	}
	defer lease.Release()

	client := pc.newAPIClient(lease)
	callStart := time.Now()
	result, err := client.GetPlayerCSRs(ctx, xuid, seasonID)
	observeHaloCall(ctxkeys.TitleSlug(ctx), "player_csrs", xuid, callStart, err)
	pc.notifyPoolOnError(lease, err)
	return result, err
}

// GetPlaylistCsr implémente HaloClient.GetPlaylistCsr() avec PolicyAnyPublic.
// Endpoint public (service token) — pas besoin du token pinned joueur.
func (pc *PooledHaloClient) GetPlaylistCsr(ctx context.Context, playlistID, xuid, seasonID string) (*PlayerPlaylistCSR, error) {
	lease, err := pc.p.Acquire(ctx, pool.PolicyAnyPublic, "")
	if err != nil {
		slog.DebugContext(ctx, "pooled: GetPlaylistCsr skipped (token unavailable)", "xuid", xuid, "err", err)
		return nil, nil
	}
	defer lease.Release()

	client := pc.newAPIClient(lease)
	callStart := time.Now()
	result, err := client.GetPlaylistCsr(ctx, playlistID, xuid, seasonID)
	observeHaloCall(ctxkeys.TitleSlug(ctx), "playlist_csr", xuid, callStart, err)
	pc.notifyPoolOnError(lease, err)
	return result, err
}

// Vérifier que PooledHaloClient implémente l'interface HaloClient.
var _ HaloClient = (*PooledHaloClient)(nil)
