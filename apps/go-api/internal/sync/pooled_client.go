package sync

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/platform/auth/pool"
)

// PooledHaloClient implémente HaloClient en utilisant un pool de tokens partagés.
// Chaque appel Acquire() crée un HaloAPIClient avec un token frais du pool.
// PolicyAnyPublic pour endpoints publics, PolicyPinnedPlayer pour endpoints privacy.
type PooledHaloClient struct {
	p pool.Pool

	// Si non-vide : token pinned sur ce gamertag (pour endpoints privacy).
	pinnedGamertag string
	pinnedXUID     string
}

// NewPooledHaloClient crée un client pooled.
// pinnedGamertag/pinnedXUID : si non-vides, les endpoints privacy utilisent ce token (ex: GetCareerRank).
func NewPooledHaloClient(p pool.Pool, pinnedGamertag, pinnedXUID string) *PooledHaloClient {
	return &PooledHaloClient{
		p:              p,
		pinnedGamertag: pinnedGamertag,
		pinnedXUID:     pinnedXUID,
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

	client := NewHaloAPIClient(lease.Tokens.SpartanToken, lease.Tokens.ClearanceToken, 1)
	return client.GetMatchHistory(ctx, gamertag, matchType, start, count)
}

// GetMatchStats implémente HaloClient.GetMatchStats() avec PolicyAnyPublic.
func (pc *PooledHaloClient) GetMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	lease, err := pc.p.Acquire(ctx, pool.PolicyAnyPublic, "")
	if err != nil {
		return nil, fmt.Errorf("pooled: Acquire failed: %w", err)
	}
	defer lease.Release()

	client := NewHaloAPIClient(lease.Tokens.SpartanToken, lease.Tokens.ClearanceToken, 1)
	return client.GetMatchStats(ctx, matchID)
}

// GetMatchFilm implémente HaloClient.GetMatchFilm() avec PolicyAnyPublic.
func (pc *PooledHaloClient) GetMatchFilm(ctx context.Context, matchID string) (map[int]filmChunkData, bool, error) {
	lease, err := pc.p.Acquire(ctx, pool.PolicyAnyPublic, "")
	if err != nil {
		return nil, false, fmt.Errorf("pooled: Acquire failed: %w", err)
	}
	defer lease.Release()

	client := NewHaloAPIClient(lease.Tokens.SpartanToken, lease.Tokens.ClearanceToken, 1)
	return client.GetMatchFilm(ctx, matchID)
}

// GetHighlightEventsChunk implémente HaloClient.GetHighlightEventsChunk() avec PolicyAnyPublic.
func (pc *PooledHaloClient) GetHighlightEventsChunk(ctx context.Context, matchID string) ([]byte, int, bool, error) {
	lease, err := pc.p.Acquire(ctx, pool.PolicyAnyPublic, "")
	if err != nil {
		return nil, 0, false, fmt.Errorf("pooled: Acquire failed: %w", err)
	}
	defer lease.Release()

	client := NewHaloAPIClient(lease.Tokens.SpartanToken, lease.Tokens.ClearanceToken, 1)
	return client.GetHighlightEventsChunk(ctx, matchID)
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

	client := NewHaloAPIClient(lease.Tokens.SpartanToken, lease.Tokens.ClearanceToken, 1)
	// HaloAPIClient handles 401/403 internally (returns nil, nil)
	return client.GetCareerRank(ctx, xuid)
}

// Vérifier que PooledHaloClient implémente l'interface HaloClient.
var _ HaloClient = (*PooledHaloClient)(nil)
