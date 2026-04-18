// Package halo — provider.go : client HTTP pour l'API Halo Infinite.
//
// Ce fichier contient le provider Halo Infinite avec :
//   - Rate limiting (max 60 req/min) via token bucket.
//   - Retry exponentiel (max 3 tentatives).
//   - HTTP client configuré (timeout 15s).
//   - Appels live Battle Pass et Challenges (tokens lus depuis le contexte via ctxkeys).
package halo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// Rate limiter (token bucket)
// ---------------------------------------------------------------------------

// rateLimiter implémente un token bucket pour limiter les requêtes.
type rateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens par seconde
	lastRefill time.Time
}

func newRateLimiter(maxPerMinute int) *rateLimiter {
	return &rateLimiter{
		tokens:     float64(maxPerMinute),
		maxTokens:  float64(maxPerMinute),
		refillRate: float64(maxPerMinute) / 60.0,
		lastRefill: time.Now(),
	}
}

// tryConsume tente de consommer un token. Ne bloque pas.
func (r *rateLimiter) tryConsume() (bool, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens += elapsed * r.refillRate
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}
	r.lastRefill = now

	if r.tokens >= 1.0 {
		r.tokens--
		return true, 0
	}
	wait := time.Duration((1.0-r.tokens)/r.refillRate*1000) * time.Millisecond
	return false, wait
}

// Wait bloque jusqu'à ce qu'un token soit disponible ou que ctx soit annulé.
func (r *rateLimiter) Wait(ctx context.Context) error {
	for {
		ok, wait := r.tryConsume()
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

// ---------------------------------------------------------------------------
// Constantes des endpoints Halo
// ---------------------------------------------------------------------------

const (
	defaultEconomyHost    = "https://economy.svc.halowaypoint.com"
	defaultChallengesHost = "https://halostats.svc.halowaypoint.com"

	providerMaxRetries  = 3
	providerRetryBase   = 800 * time.Millisecond
	providerHTTPTimeout = 15 * time.Second
)

// ---------------------------------------------------------------------------
// HaloProvider
// ---------------------------------------------------------------------------

// HaloProvider est le client HTTP pour l'API Halo Infinite (343 Industries).
// Thread-safe : une seule instance par processus.
// Les tokens et le XUID sont lus depuis le contexte via ctxkeys.HaloTokens / ctxkeys.HaloXUID.
type HaloProvider struct {
	client     *http.Client
	limiter    *rateLimiter
	maxRetries int
	// Overridables pour les tests : vides en production (→ fallback vers les constantes).
	battlePassBaseURL string
	challengesBaseURL string
	// Sprint 54 B5 : cache process-level de la privacy par xuid.
	privacyCache privacyTTLCache
}

// DefaultHaloProvider est l'instance globale du provider (60 req/min, 3 retries).
var DefaultHaloProvider = NewHaloProvider()

// NewHaloProvider crée un provider Halo avec les paramètres par défaut.
func NewHaloProvider() *HaloProvider {
	return &HaloProvider{
		client: &http.Client{
			Timeout: providerHTTPTimeout,
		},
		limiter:    newRateLimiter(60),
		maxRetries: providerMaxRetries,
	}
}

// ---------------------------------------------------------------------------
// GetBattlePass
// ---------------------------------------------------------------------------

// GetBattlePass retourne les infos de l'opération Battle Pass active.
// Les tokens et le XUID sont lus depuis ctx via ctxkeys.
// Retourne available=false, error_hint="auth_required" si l'auth est absente du contexte.
func (p *HaloProvider) GetBattlePass(ctx context.Context) domain.BattlePassResponse {
	tokens := ctxkeys.HaloTokens(ctx)
	xuid := ctxkeys.HaloXUID(ctx)
	if tokens == nil || xuid == "" {
		hint := "auth_required"
		return domain.BattlePassResponse{Available: false, ErrorHint: &hint}
	}

	resp, err := p.fetchBattlePass(ctx, tokens, xuid)
	if err != nil {
		slog.WarnContext(ctx, "halo_provider: battle_pass fetch failed", "xuid", xuid, "err", err)
		hint := "fetch_error"
		return domain.BattlePassResponse{Available: false, ErrorHint: &hint}
	}
	return resp
}

// battlePassProgress contient la progression dans un palier de Battle Pass.
type battlePassProgress struct {
	Rank            int `json:"Rank"`
	PartialProgress int `json:"PartialProgress"`
}

// battlePassTrack représente une opération dans la réponse de l'endpoint rewardtracks.
type battlePassTrack struct {
	RewardTrackPath string             `json:"RewardTrackPath"`
	CurrentProgress battlePassProgress `json:"CurrentProgress"`
}

// fetchBattlePass appelle l'endpoint economy operations et parse la réponse.
func (p *HaloProvider) fetchBattlePass(ctx context.Context, tokens *domain.HaloTokens, xuid string) (domain.BattlePassResponse, error) {
	base := p.battlePassBaseURL
	if base == "" {
		base = defaultEconomyHost
	}
	url := fmt.Sprintf("%s/hi/players/xuid(%s)/rewardtracks/operations", base, xuid)

	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		return domain.BattlePassResponse{}, err
	}

	var raw struct {
		ActiveOperationRewardTrackPath string            `json:"ActiveOperationRewardTrackPath"`
		OperationRewardTracks          []battlePassTrack `json:"OperationRewardTracks"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return domain.BattlePassResponse{}, fmt.Errorf("battle_pass decode: %w", err)
	}

	rank, progress, trackPath := parseBattlePassTrack(raw.ActiveOperationRewardTrackPath, raw.OperationRewardTracks)
	slog.DebugContext(ctx, "halo_provider: battle_pass fetched", "xuid", xuid, "rank", rank, "track", trackPath)
	return domain.BattlePassResponse{
		Available:   true,
		Rank:        &rank,
		Progress:    &progress,
		RewardTrack: &trackPath,
	}, nil
}

// parseBattlePassTrack extrait rank, progress et trackPath depuis les opérations.
func parseBattlePassTrack(activePath string, tracks []battlePassTrack) (rank, progress int, trackPath string) {
	trackPath = activePath
	for _, t := range tracks {
		if t.RewardTrackPath == activePath {
			return t.CurrentProgress.Rank, t.CurrentProgress.PartialProgress, activePath
		}
	}
	// ActivePath non trouvé dans la liste → utiliser le premier si disponible.
	if len(tracks) > 0 {
		return tracks[0].CurrentProgress.Rank, tracks[0].CurrentProgress.PartialProgress, tracks[0].RewardTrackPath
	}
	return 0, 0, activePath
}

// ---------------------------------------------------------------------------
// GetChallenges
// ---------------------------------------------------------------------------

// GetChallenges retourne le résumé des défis actifs du joueur.
// Les tokens et le XUID sont lus depuis ctx via ctxkeys.
// Retourne available=false, error_hint="auth_required" si l'auth est absente du contexte.
func (p *HaloProvider) GetChallenges(ctx context.Context) domain.ChallengesResponse {
	tokens := ctxkeys.HaloTokens(ctx)
	xuid := ctxkeys.HaloXUID(ctx)
	if tokens == nil || xuid == "" {
		hint := "auth_required"
		return domain.ChallengesResponse{Available: false, ErrorHint: &hint}
	}

	resp, err := p.fetchChallenges(ctx, tokens, xuid)
	if err != nil {
		slog.WarnContext(ctx, "halo_provider: challenges fetch failed", "xuid", xuid, "err", err)
		hint := "fetch_error"
		return domain.ChallengesResponse{Available: false, ErrorHint: &hint}
	}
	return resp
}

// fetchChallenges appelle l'endpoint decks et parse la réponse.
func (p *HaloProvider) fetchChallenges(ctx context.Context, tokens *domain.HaloTokens, xuid string) (domain.ChallengesResponse, error) {
	base := p.challengesBaseURL
	if base == "" {
		base = defaultChallengesHost
	}
	url := fmt.Sprintf("%s/hi/players/xuid(%s)/decks", base, xuid)

	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		return domain.ChallengesResponse{}, err
	}

	var raw struct {
		AssignedDecks []struct {
			Expiration struct {
				ISO8601Date string `json:"ISO8601Date"`
			} `json:"Expiration"`
			ActiveChallenges    []json.RawMessage `json:"ActiveChallenges"`
			CompletedChallenges []json.RawMessage `json:"CompletedChallenges"`
		} `json:"AssignedDecks"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return domain.ChallengesResponse{}, fmt.Errorf("challenges decode: %w", err)
	}

	total, completed, xpAvail, nextExpiry := aggregateChallenges(raw.AssignedDecks)
	slog.InfoContext(ctx, "halo_provider: challenges fetched", "xuid", xuid, "total", total, "completed", completed)
	resp := domain.ChallengesResponse{
		Available:   true,
		Total:       &total,
		Completed:   &completed,
		XPAvailable: &xpAvail,
	}
	if nextExpiry != "" {
		resp.NextExpiry = &nextExpiry
	}
	return resp, nil
}

// aggregateChallenges calcule les totaux depuis les decks assignés.
func aggregateChallenges(decks []struct {
	Expiration struct {
		ISO8601Date string `json:"ISO8601Date"`
	} `json:"Expiration"`
	ActiveChallenges    []json.RawMessage `json:"ActiveChallenges"`
	CompletedChallenges []json.RawMessage `json:"CompletedChallenges"`
}) (total, completed, xpAvail int, nextExpiry string) {
	for _, deck := range decks {
		total += len(deck.ActiveChallenges)
		completed += len(deck.CompletedChallenges)
		for _, raw := range deck.ActiveChallenges {
			var ch struct {
				XPReward int `json:"XPReward"`
			}
			if err := json.Unmarshal(raw, &ch); err == nil {
				xpAvail += ch.XPReward
			}
		}
		if exp := deck.Expiration.ISO8601Date; exp != "" {
			if nextExpiry == "" || exp < nextExpiry {
				nextExpiry = exp
			}
		}
	}
	return
}

// ---------------------------------------------------------------------------
// doGet — helper HTTP avec retry + rate limiting
// ---------------------------------------------------------------------------

// doGet exécute un GET authentifié avec retry + backoff exponentiel.
// Pattern identique à halo_client.go:doGet.
func (p *HaloProvider) doGet(ctx context.Context, rawURL string, tokens *domain.HaloTokens) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < p.maxRetries; attempt++ {
		if err := p.limiter.Wait(ctx); err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("doGet new request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("x-343-authorization-spartan", tokens.SpartanToken)
		if tokens.ClearanceToken != "" {
			req.Header.Set("343-clearance", tokens.ClearanceToken)
		}

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = err
			p.backoff(ctx, attempt)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Erreurs d'auth : inutile de retry.
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("doGet %s: HTTP %d (tokens invalides/expirés)", rawURL, resp.StatusCode)
		}
		// Ressource absente : ne pas retry.
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
			return nil, fmt.Errorf("doGet %s: HTTP %d (ressource absente)", rawURL, resp.StatusCode)
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("doGet %s: HTTP %d", rawURL, resp.StatusCode)
			p.backoff(ctx, attempt)
			continue
		}
		if readErr != nil {
			lastErr = readErr
			p.backoff(ctx, attempt)
			continue
		}
		return body, nil
	}
	return nil, fmt.Errorf("doGet %s: %d tentatives échouées — %w", rawURL, p.maxRetries, lastErr)
}

// backoff attend un délai exponentiel avant de retenter.
func (p *HaloProvider) backoff(ctx context.Context, attempt int) {
	delay := providerRetryBase * (1 << attempt)
	if delay > 10*time.Second {
		delay = 10 * time.Second
	}
	select {
	case <-ctx.Done():
	case <-time.After(delay):
	}
}
