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

	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/netguard"

	"golang.org/x/sync/singleflight"
)

// Hints d'erreur retournés dans domain.*Response.ErrorHint pour signaler
// pourquoi un appel live a échoué côté provider Waypoint.
const (
	errHintAuthRequired = "auth_required"
	errHintFetchError   = "fetch_error"
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

// TrackDefinitionPersister est notifié après chaque fetch réussi d'une définition de
// Reward Track depuis GameCMS, pour la persister dans battlepass_track_definitions.
// Implémenté par platform/duckdb.PersistSink.
type TrackDefinitionPersister interface {
	UpsertTrackDefinition(ctx context.Context, trackPath string, raw []byte) error
}

// ItemDefinitionPersister est notifié après chaque fetch réussi d'un item Battle Pass
// depuis GameCMS, pour la persister dans battlepass_item_definitions + _translations.
// Implémenté par platform/duckdb.PersistSink.
type ItemDefinitionPersister interface {
	UpsertItemDefinition(ctx context.Context, itemPath string, raw []byte) error
}

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
	gameCMSBaseURL    string
	// assetResolver est le resolver unifié (P4/P5).
	assetResolver assets.Resolver
	// trackDefPersister persiste les définitions de tracks dans battlepass_track_definitions.
	trackDefPersister TrackDefinitionPersister
	// itemDefPersister persiste les définitions d'items dans battlepass_item_definitions.
	itemDefPersister ItemDefinitionPersister
	// titleSlug est le titre courant (ex: "halo_infinite").
	titleSlug string
	// Sprint 54 B5 : cache process-level de la privacy par xuid.
	// Pointeur pour permettre les clones de HaloProvider (With*) sans copier
	// le mutex interne — les clones partagent le même cache.
	privacyCache *privacyTTLCache
	// staticTokens est injecté par WithTokens pour les CLIs batch (pas de contexte HTTP).
	staticTokens *domain.HaloTokens
}

// DefaultHaloProvider est l'instance globale du provider (60 req/min, 3 retries).
var DefaultHaloProvider = NewHaloProvider()

// challengesFetchSFGroup déduplique les fetchs live concurrents des challenges.
// La clé inclut les paramètres d'enrichissement du provider pour ne partager
// que des réponses construites dans le même contexte metadata/assets.
var challengesFetchSFGroup singleflight.Group

// NewHaloProvider crée un provider Halo avec les paramètres par défaut.
func NewHaloProvider() *HaloProvider {
	return &HaloProvider{
		client: &http.Client{
			Timeout: providerHTTPTimeout,
		},
		limiter:      newRateLimiter(60),
		maxRetries:   providerMaxRetries,
		privacyCache: &privacyTTLCache{entries: make(map[string]privacyCacheEntry)},
	}
}

// WithRateLimit remplace le rate limiter par un nouveau limiter avec maxPerMinute.
// Utile pour les opérations batch (populate-playlists-catalog) sur des APIs publiques CDN.
func (p *HaloProvider) WithRateLimit(maxPerMinute int) *HaloProvider {
	if p == nil {
		return nil
	}
	clone := *p
	clone.limiter = newRateLimiter(maxPerMinute)
	return &clone
}

// WithTokens injecte des tokens Halo statiques dans le provider.
// Utilisé par les CLIs batch qui ne tournent pas dans le contexte HTTP du serveur.
// Les tokens sont ajoutés aux requêtes Discovery UGC (gamecms-hacs) qui nécessitent auth.
func (p *HaloProvider) WithTokens(tokens *domain.HaloTokens) *HaloProvider {
	if p == nil {
		return nil
	}
	clone := *p
	clone.staticTokens = tokens
	return &clone
}

// WithAssetResolver câble le resolver unifié (P4/P5).
// Quand non-nil, les opérations de fetch/cache de définitions et d'images
// sont déléguées au resolver au lieu d'écrire directement dans DuckDB.
func (p *HaloProvider) WithAssetResolver(resolver assets.Resolver) *HaloProvider {
	if p == nil {
		return nil
	}
	clone := *p
	clone.assetResolver = resolver
	return &clone
}

// WithTrackDefPersister câble le persister de définitions de tracks.
// Quand présent, chaque fetch réussi d'un Reward Track JSON est persisté dans
// battlepass_track_definitions pour que LoadSeasonPassTracks puisse servir les images.
func (p *HaloProvider) WithTrackDefPersister(persister TrackDefinitionPersister) *HaloProvider {
	if p == nil {
		return nil
	}
	clone := *p
	clone.trackDefPersister = persister
	return &clone
}

// WithItemDefPersister câble le persister de définitions d'items Battle Pass.
// Quand présent, chaque item résolu via KindBPItemDefinition est persisté dans
// battlepass_item_definitions et battlepass_item_translations.
func (p *HaloProvider) WithItemDefPersister(persister ItemDefinitionPersister) *HaloProvider {
	if p == nil {
		return nil
	}
	clone := *p
	clone.itemDefPersister = persister
	return &clone
}

// WithTitleSlug configure le slug du titre pour ce provider (ex: "halo_infinite").
// Si slug est vide, le DefaultSlug ("halo_infinite") est utilisé comme fallback.
func (p *HaloProvider) WithTitleSlug(slug string) *HaloProvider {
	if p == nil {
		return nil
	}
	clone := *p
	if slug == "" {
		clone.titleSlug = title.DefaultSlug
	} else {
		clone.titleSlug = slug
	}
	return &clone
}

// titleID retourne le titleSlug effectif, avec fallback sur DefaultSlug.
func (p *HaloProvider) titleID() string {
	if p.titleSlug == "" {
		return title.DefaultSlug
	}
	return p.titleSlug
}

// ---------------------------------------------------------------------------
// GetBattlePass
// ---------------------------------------------------------------------------

// GetBattlePass retourne les infos de l'opération Battle Pass active.
// Les tokens et le XUID sont lus depuis ctx via ctxkeys.
// Retourne available=false, error_hint="auth_required" si l'auth est absente du contexte.
func (p *HaloProvider) GetBattlePass(ctx context.Context) domain.BattlePassResponse {
	resp, _ := p.GetBattlePassWithRaw(ctx)
	return resp
}

// GetBattlePassWithRaw retourne la réponse Battle Pass et les bytes JSON bruts
// de Waypoint (pour persistance). Les bytes sont nil en cas d'erreur ou d'absence d'auth.
func (p *HaloProvider) GetBattlePassWithRaw(ctx context.Context) (domain.BattlePassResponse, []byte) {
	tokens := ctxkeys.HaloTokens(ctx)
	xuid := ctxkeys.HaloXUID(ctx)
	if tokens == nil || xuid == "" {
		hint := errHintAuthRequired
		return domain.BattlePassResponse{Available: false, ErrorHint: &hint}, nil
	}

	type bpResult struct {
		resp domain.BattlePassResponse
		raw  []byte
	}
	out, err := retryOnAuth(ctx, func(c context.Context) (bpResult, error) {
		r, raw, e := p.fetchBattlePass(c, ctxkeys.HaloTokens(c), xuid)
		return bpResult{resp: r, raw: raw}, e
	})
	if err != nil {
		slog.WarnContext(ctx, "halo_provider: battle_pass fetch failed", "xuid", xuid, "err", err)
		hint := errHintFetchError
		return domain.BattlePassResponse{Available: false, ErrorHint: &hint}, nil
	}
	return out.resp, out.raw
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
// Retourne la réponse domaine, les bytes JSON bruts (pour persistance), et une erreur éventuelle.
func (p *HaloProvider) fetchBattlePass(ctx context.Context, tokens *domain.HaloTokens, xuid string) (domain.BattlePassResponse, []byte, error) {
	base := p.hostFor(ctx, games.EndpointEconomy, p.battlePassBaseURL, defaultEconomyHost)
	url := fmt.Sprintf("%s/%s/players/xuid(%s)/rewardtracks/operations", base, p.gamePrefix(ctx), xuid)

	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		return domain.BattlePassResponse{}, nil, err
	}

	var raw struct {
		ActiveOperationRewardTrackPath string            `json:"ActiveOperationRewardTrackPath"`
		OperationRewardTracks          []battlePassTrack `json:"OperationRewardTracks"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return domain.BattlePassResponse{}, body, fmt.Errorf("battle_pass decode: %w", err)
	}

	rank, progress, trackPath := parseBattlePassTrack(raw.ActiveOperationRewardTrackPath, raw.OperationRewardTracks)
	slog.DebugContext(ctx, "halo_provider: battle_pass fetched", "xuid", xuid, "rank", rank, "track", trackPath)

	// Pré-cache fire-and-forget des définitions GameCMS pour tous les tracks via le resolver.
	if p.assetResolver != nil {
		var trackRefs []assets.Ref
		for _, t := range raw.OperationRewardTracks {
			if t.RewardTrackPath == "" {
				continue
			}
			trackRefs = append(trackRefs, assets.Ref{
				Kind:    assets.KindRewardTrackDefinition,
				TitleID: p.titleID(),
				ID:      t.RewardTrackPath,
			})
		}
		if len(trackRefs) > 0 {
			go p.assetResolver.Warm(context.Background(), trackRefs...)
		}
	}

	return domain.BattlePassResponse{
		Available:   true,
		Rank:        &rank,
		Progress:    &progress,
		RewardTrack: &trackPath,
	}, body, nil
}

// parseBattlePassTrack extrait rank, progress et trackPath depuis les opérations.
func parseBattlePassTrack(activePath string, tracks []battlePassTrack) (rank, progress int, trackPath string) {
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
	resp, _ := p.GetChallengesWithRaw(ctx)
	return resp
}

// GetChallengesWithRaw retourne la réponse défis et les bytes JSON bruts
// de Waypoint (pour persistance). Les bytes sont nil en cas d'erreur ou d'absence d'auth.
func (p *HaloProvider) GetChallengesWithRaw(ctx context.Context) (domain.ChallengesResponse, []byte) {
	tokens := ctxkeys.HaloTokens(ctx)
	xuid := ctxkeys.HaloXUID(ctx)
	if tokens == nil || xuid == "" {
		hint := errHintAuthRequired
		return domain.ChallengesResponse{Available: false, ErrorHint: &hint}, nil
	}

	type challengesFetchResult struct {
		response domain.ChallengesResponse
		rawBody  []byte
	}

	out, err := retryOnAuth(ctx, func(c context.Context) (challengesFetchResult, error) {
		v, ferr, _ := challengesFetchSFGroup.Do(p.challengesFetchKey(c, xuid), func() (interface{}, error) {
			resp, raw, fetchErr := p.fetchChallenges(c, ctxkeys.HaloTokens(c), xuid)
			if fetchErr != nil {
				return nil, fetchErr
			}
			return challengesFetchResult{response: resp, rawBody: raw}, nil
		})
		if ferr != nil {
			return challengesFetchResult{}, ferr
		}
		r, ok := v.(challengesFetchResult)
		if !ok {
			return challengesFetchResult{}, fmt.Errorf("challenges fetch invalid result type")
		}
		return r, nil
	})
	if err != nil {
		slog.WarnContext(ctx, "halo_provider: challenges fetch failed", "xuid", xuid, "err", err)
		hint := errHintFetchError
		return domain.ChallengesResponse{Available: false, ErrorHint: &hint}, nil
	}
	return out.response, out.rawBody
}

func (p *HaloProvider) challengesFetchKey(ctx context.Context, xuid string) string {
	base := p.hostFor(ctx, games.EndpointChallenges, p.challengesBaseURL, defaultChallengesHost)
	return xuid + "|" + base
}

// fetchChallenges appelle l'endpoint decks et parse la réponse.
// Retourne la réponse domaine, les bytes JSON bruts (pour persistance), et une erreur éventuelle.
func (p *HaloProvider) fetchChallenges(ctx context.Context, tokens *domain.HaloTokens, xuid string) (domain.ChallengesResponse, []byte, error) {
	base := p.hostFor(ctx, games.EndpointChallenges, p.challengesBaseURL, defaultChallengesHost)
	url := fmt.Sprintf("%s/%s/players/xuid(%s)/decks", base, p.gamePrefix(ctx), xuid)

	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		return domain.ChallengesResponse{}, nil, err
	}

	var raw struct {
		AssignedDecks []challengeDeckRaw `json:"AssignedDecks"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return domain.ChallengesResponse{}, body, fmt.Errorf("challenges decode: %w", err)
	}

	total, completed, xpAvail, nextExpiry := aggregateChallenges(raw.AssignedDecks)
	items := p.buildActiveChallengeItems(ctx, tokens, raw.AssignedDecks)
	slog.InfoContext(ctx, "halo_provider: challenges fetched", "xuid", xuid, "total", total, "completed", completed)
	resp := domain.ChallengesResponse{
		Available:   true,
		Total:       &total,
		Completed:   &completed,
		XPAvailable: &xpAvail,
		Items:       items,
	}
	if nextExpiry != "" {
		resp.NextExpiry = &nextExpiry
	}
	return resp, body, nil
}

// aggregateChallenges calcule les totaux depuis les decks assignés.
func aggregateChallenges(decks []challengeDeckRaw) (total, completed, xpAvail int, nextExpiry string) {
	for _, deck := range decks {
		activeCnt := len(deck.ActiveChallenges)
		completedCnt := len(deck.CompletedChallenges)
		total += activeCnt + completedCnt
		completed += completedCnt
		for _, ch := range deck.ActiveChallenges {
			if ch.XPReward != nil {
				xpAvail += *ch.XPReward
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
	// Mode démo : aucune sortie tierce (battle pass, défis). Les appelants
	// remontent l'échec en WARN et servent une réponse vide — comportement
	// déjà en place pour un compte sans token.
	if err := netguard.Check(ctx, "halo_provider.get"); err != nil {
		return nil, err
	}
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
		// Auth headers uniquement si tokens fournis (API publiques vs privées)
		if tokens != nil {
			req.Header.Set("x-343-authorization-spartan", tokens.SpartanToken)
			if tokens.ClearanceToken != "" {
				req.Header.Set("343-clearance", tokens.ClearanceToken)
			}
		}

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = err
			p.backoff(ctx, attempt)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Erreurs d'auth : inutile de retry AU NIVEAU HTTP (le re-mint se fait plus haut,
		// via retryOnAuth, qui détecte errHaloAuthFailure). On wrappe le sentinel.
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("doGet %s: HTTP %d (tokens invalides/expirés): %w", rawURL, resp.StatusCode, errHaloAuthFailure)
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
