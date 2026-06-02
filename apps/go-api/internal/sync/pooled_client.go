package sync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/time/rate"

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

// notifyPoolOnHTTPError inspecte l'erreur et signale OnHTTPError au pool si 429/503.
func (pc *PooledHaloClient) notifyPoolOnHTTPError(err error) {
	if err == nil {
		return
	}
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		if httpErr.StatusCode == 429 || httpErr.StatusCode == 503 {
			msg := "rate_limit_exceeded"
			if httpErr.StatusCode == 503 {
				msg = "service_unavailable"
			}
			slog.WarnContext(context.Background(), "pooled: pool global cooldown triggered",
				"statusCode", httpErr.StatusCode, "reason", msg, "gamertag", pc.pinnedGamertag,
				"retry_after_s", httpErr.RetryAfter.Seconds())
			pc.p.OnHTTPError(httpErr.StatusCode, httpErr.RetryAfter)
		}
	}
}

// GetMatchHistory implémente HaloClient.GetMatchHistory() avec PolicyAnyPublic.
func (pc *PooledHaloClient) GetMatchHistory(
	ctx context.Context,
	gamertag, matchType string,
	start, count int,
) ([]MatchHistoryEntry, error) {
	lease, err := pc.p.Acquire(ctx, pool.PolicyAnyPublic, "")
	if err != nil {
		return nil, fmt.Errorf("pooled: Acquire failed: %w", err)
	}
	defer lease.Release()

	client := pc.newAPIClient(lease)
	result, err := client.GetMatchHistory(ctx, gamertag, matchType, start, count)
	pc.notifyPoolOnHTTPError(err)
	return result, err
}

// GetMatchStats implémente HaloClient.GetMatchStats() avec PolicyAnyPublic.
func (pc *PooledHaloClient) GetMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	lease, err := pc.p.Acquire(ctx, pool.PolicyAnyPublic, "")
	if err != nil {
		return nil, fmt.Errorf("pooled: Acquire failed: %w", err)
	}
	defer lease.Release()

	client := pc.newAPIClient(lease)
	result, err := client.GetMatchStats(ctx, matchID)
	pc.notifyPoolOnHTTPError(err)
	return result, err
}

// GetMatchSkill implémente HaloClient.GetMatchSkill() avec PolicyAnyPublic.
func (pc *PooledHaloClient) GetMatchSkill(
	ctx context.Context,
	matchID string,
	xuids []string,
) (map[string]*MatchSkillData, error) {
	lease, err := pc.p.Acquire(ctx, pool.PolicyAnyPublic, "")
	if err != nil {
		return nil, fmt.Errorf("pooled: Acquire failed: %w", err)
	}
	defer lease.Release()

	client := pc.newAPIClient(lease)
	result, err := client.GetMatchSkill(ctx, matchID, xuids)
	pc.notifyPoolOnHTTPError(err)
	return result, err
}

// GetMatchFilm implémente HaloClient.GetMatchFilm() avec PolicyAnyPublic.
func (pc *PooledHaloClient) GetMatchFilm(ctx context.Context, matchID string) (map[int]filmChunkData, bool, error) {
	lease, err := pc.p.Acquire(ctx, pool.PolicyAnyPublic, "")
	if err != nil {
		return nil, false, fmt.Errorf("pooled: Acquire failed: %w", err)
	}
	defer lease.Release()

	client := pc.newAPIClient(lease)
	result, ok, err := client.GetMatchFilm(ctx, matchID)
	pc.notifyPoolOnHTTPError(err)
	return result, ok, err
}

// GetHighlightEventsChunk implémente HaloClient.GetHighlightEventsChunk() avec PolicyAnyPublic.
func (pc *PooledHaloClient) GetHighlightEventsChunk(ctx context.Context, matchID string) ([]byte, int, bool, error) {
	lease, err := pc.p.Acquire(ctx, pool.PolicyAnyPublic, "")
	if err != nil {
		return nil, 0, false, fmt.Errorf("pooled: Acquire failed: %w", err)
	}
	defer lease.Release()

	client := pc.newAPIClient(lease)
	result, ver, ok, err := client.GetHighlightEventsChunk(ctx, matchID)
	pc.notifyPoolOnHTTPError(err)
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
	return client.GetCareerRank(ctx, xuid)
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
	result, err := client.GetPlayerCSRs(ctx, xuid, seasonID)
	pc.notifyPoolOnHTTPError(err)
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
	result, err := client.GetPlaylistCsr(ctx, playlistID, xuid, seasonID)
	pc.notifyPoolOnHTTPError(err)
	return result, err
}

// Vérifier que PooledHaloClient implémente l'interface HaloClient.
var _ HaloClient = (*PooledHaloClient)(nil)
