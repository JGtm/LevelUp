// Package sync — halo_client.go : client HTTP pour l'API Halo Infinite stats.
//
// Portage de SPNKrAPIClient.get_match_history() + get_match_stats() (Python api_client.py).
// Ce client est stateless : instancié pour chaque session de sync avec les tokens du joueur.
//
// Endpoints utilisés :
//
//	GET https://halostats.svc.halowaypoint.com:443/hi/players/{gamertag}/matches
//	GET https://halostats.svc.halowaypoint.com:443/hi/matches/{match_id}/stats
//
// Headers requis (portage de spnkr/client.py) :
//
//	Accept: application/json
//	x-343-authorization-spartan: {SpartanToken}
//	343-clearance: {ClearanceToken}
package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	haloStatsHost = "https://halostats.svc.halowaypoint.com:443"
	// maxRetries est le nombre de tentatives avant abandon (portage de tries=4 Python).
	maxRetries = 4
	// retryBaseDelay est le délai de base pour le backoff exponentiel.
	retryBaseDelay = 800 * time.Millisecond
	// matchCountMax est le nombre maximum de matchs par page d'historique (API Halo).
	matchCountMax = 25
)

// validMatchTypes est l'ensemble des types de match valides.
var validMatchTypes = map[string]bool{
	"all":         true,
	"matchmaking": true,
	"custom":      true,
	"local":       true,
}

// rexUUID est l'expression régulière de validation UUID v4.
var rexUUID = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// HaloClient est l'interface abstraite du client HTTP Halo Infinite.
// Permet l'injection de dépendances et les mocks dans les tests de engine.go.
type HaloClient interface {
	// GetMatchHistory récupère une page d'historique de matchs d'un joueur.
	GetMatchHistory(ctx context.Context, gamertag, matchType string, start, count int) ([]MatchHistoryEntry, error)
	// GetMatchStats récupère les stats détaillées d'un match (JSON brut).
	GetMatchStats(ctx context.Context, matchID string) (map[string]any, error)
	// GetMatchFilm récupère les chunks du film d'un match.
	// Retourne (nil, false, nil) si le film est absent (404/410 — normal pour vieux matchs).
	GetMatchFilm(ctx context.Context, matchID string) (map[int]filmChunkData, bool, error)
	// GetCareerRank récupère la progression du rang carrière via l'API Economy.
	// Retourne (nil, nil) si le token est absent ou insuffisant.
	GetCareerRank(ctx context.Context, xuid string) (*CareerRankData, error)
}

// MatchHistoryEntry est un élément de l'historique des matchs.
// Portage de MatchHistoryItem (Python models.py).
type MatchHistoryEntry struct {
	MatchID   string
	StartTime string
}

// HaloAPIClient est le client HTTP pour l'API Halo Infinite stats.
// Thread-safe : utilise net/http.Client (concurrency-safe).
type HaloAPIClient struct {
	http           *http.Client
	spartanToken   string
	clearanceToken string
	economyBaseURL string
	// minInterval est l'intervalle minimum entre deux requêtes (rate limiting).
	minInterval time.Duration
	lastRequest time.Time
}

// NewHaloAPIClient crée un client authentifié avec les tokens Halo du joueur.
// requestsPerSecond contrôle le rate limiting (portage de requests_per_second Python).
func NewHaloAPIClient(spartanToken, clearanceToken string, requestsPerSecond int) *HaloAPIClient {
	if requestsPerSecond <= 0 {
		requestsPerSecond = 10
	}
	return &HaloAPIClient{
		http: &http.Client{
			Timeout: 20 * time.Second,
		},
		spartanToken:   spartanToken,
		clearanceToken: clearanceToken,
		economyBaseURL: "https://economy.svc.halowaypoint.com",
		minInterval:    time.Second / time.Duration(requestsPerSecond),
	}
}

// GetMatchHistory récupère une page de l'historique des matchs d'un joueur.
// Portage de SPNKrAPIClient.get_match_history() (Python api_client.py).
//
//   - gamertag  : gamertag ou xuid(1234) du joueur
//   - matchType : "all", "matchmaking", "custom", "local"
//   - start     : offset (0 = plus récent)
//   - count     : nombre de matchs (max 25)
func (c *HaloAPIClient) GetMatchHistory(
	ctx context.Context,
	gamertag, matchType string,
	start, count int,
) ([]MatchHistoryEntry, error) {
	if strings.TrimSpace(gamertag) == "" {
		return nil, errors.New("GetMatchHistory: gamertag vide")
	}
	if !validMatchTypes[matchType] {
		return nil, fmt.Errorf("GetMatchHistory: matchType invalide %q (attendu : all|matchmaking|custom|local)", matchType)
	}
	if start < 0 {
		return nil, fmt.Errorf("GetMatchHistory: start doit être ≥ 0 (reçu %d)", start)
	}
	if count < 1 || count > matchCountMax {
		return nil, fmt.Errorf("GetMatchHistory: count doit être entre 1 et %d (reçu %d)", matchCountMax, count)
	}
	endpoint := fmt.Sprintf("%s/hi/players/%s/matches", haloStatsHost, url.PathEscape(gamertag))
	params := url.Values{
		"start": {strconv.Itoa(start)},
		"count": {strconv.Itoa(count)},
		"type":  {matchType},
	}
	body, err := c.doGet(ctx, endpoint+"?"+params.Encode())
	if err != nil {
		return nil, fmt.Errorf("GetMatchHistory: %w", err)
	}

	var resp struct {
		Results []struct {
			MatchID   string `json:"MatchId"`
			MatchInfo struct {
				StartTime string `json:"StartTime"`
			} `json:"MatchInfo"`
		} `json:"Results"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("GetMatchHistory decode: %w", err)
	}

	entries := make([]MatchHistoryEntry, 0, len(resp.Results))
	for _, r := range resp.Results {
		entries = append(entries, MatchHistoryEntry{
			MatchID:   r.MatchID,
			StartTime: r.MatchInfo.StartTime,
		})
	}
	return entries, nil
}

// GetMatchStats récupère les stats détaillées d'un match.
// Retourne le JSON brut en map[string]any (portage de get_match_stats Python).
func (c *HaloAPIClient) GetMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	if !rexUUID.MatchString(matchID) {
		return nil, fmt.Errorf("GetMatchStats: matchID n'est pas un UUID valide %q", matchID)
	}
	endpoint := fmt.Sprintf("%s/hi/matches/%s/stats", haloStatsHost, url.PathEscape(matchID))
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("GetMatchStats(%s): %w", matchID, err)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("GetMatchStats decode(%s): %w", matchID, err)
	}
	return result, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Film API (Sprint 41 T2)
// ─────────────────────────────────────────────────────────────────────────────

const haloUGCHost = "https://discovery-infiniteugc.svc.halowaypoint.com"

// filmManifest représente la réponse JSON de l'endpoint film spectate.
type filmManifest struct {
	CustomData struct {
		FilmChunks []filmChunk `json:"FilmChunks"`
	} `json:"CustomData"`
}

// filmChunk décrit un segment binaire du film Halo.
type filmChunk struct {
	ChunkType                        int    `json:"ChunkType"`
	FileSize                         int    `json:"FileSize"`
	ChunkStartTimeOffsetMilliseconds int    `json:"ChunkStartTimeOffsetMilliseconds"`
	DurationMilliseconds             int    `json:"DurationMilliseconds"`
	ChunkURL                         string `json:"ChunkUrl"`
}

// GetMatchFilm télécharge le manifest film d'un match et retourne les chunks indexés.
// Retourne (chunks, true, nil) si le film est disponible.
// Retourne (nil, false, nil) si le film est absent (404/410) — normal pour vieux matchs.
func (c *HaloAPIClient) GetMatchFilm(ctx context.Context, matchID string) (map[int]filmChunkData, bool, error) {
	if !rexUUID.MatchString(matchID) {
		return nil, false, fmt.Errorf("GetMatchFilm: matchID n'est pas un UUID valide %q", matchID)
	}
	endpoint := fmt.Sprintf("%s/hi/films/matches/%s/spectate", haloUGCHost, url.PathEscape(matchID))
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		// 404/410 = film expiré ou non disponible, pas une erreur permanente.
		if isNotFoundErr(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("GetMatchFilm manifest(%s): %w", matchID, err)
	}

	var manifest filmManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, false, fmt.Errorf("GetMatchFilm decode(%s): %w", matchID, err)
	}

	chunks := manifest.CustomData.FilmChunks
	if len(chunks) == 0 {
		return nil, false, nil
	}

	result := make(map[int]filmChunkData, len(chunks))
	for i, chunk := range chunks {
		data, err := c.downloadBlob(ctx, chunk.ChunkURL)
		if err != nil {
			return nil, false, fmt.Errorf("GetMatchFilm chunk %d(%s): %w", i, matchID, err)
		}
		result[i] = filmChunkData{
			Data:       data,
			StartMS:    chunk.ChunkStartTimeOffsetMilliseconds,
			DurationMS: chunk.DurationMilliseconds,
		}
	}
	return result, true, nil
}

// filmChunkData encapsule les données binaires d'un chunk film.
type filmChunkData struct {
	Data       []byte
	StartMS    int
	DurationMS int
}

// isNotFoundErr vérifie si l'erreur est un 404 ou 410 (film absent).
func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return contains(s, "HTTP 404") || contains(s, "HTTP 410") || contains(s, "ressource absente")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// downloadBlob télécharge un blob Halo sans header d'auth (pre-signed URL).
func (c *HaloAPIClient) downloadBlob(ctx context.Context, blobURL string) ([]byte, error) {
	c.rateWait(ctx)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, blobURL, nil)
	if err != nil {
		return nil, fmt.Errorf("downloadBlob new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloadBlob: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloadBlob HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// doGet exécute un GET authentifié avec retry + backoff exponentiel.
// Portage de request_with_retries() (Python api_client.py).
func (c *HaloAPIClient) doGet(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Rate limiting : attendre l'intervalle minimum entre deux requêtes.
		c.rateWait(ctx)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("doGet new request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("x-343-authorization-spartan", c.spartanToken)
		req.Header.Set("343-clearance", c.clearanceToken)

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			c.backoff(ctx, attempt)
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
			c.backoff(ctx, attempt)
			continue
		}
		if readErr != nil {
			lastErr = readErr
			c.backoff(ctx, attempt)
			continue
		}
		return body, nil
	}
	return nil, fmt.Errorf("doGet %s: %d tentatives échouées — %w", rawURL, maxRetries, lastErr)
}

// rateWait bloque jusqu'à ce que l'intervalle minimum soit écoulé depuis la dernière requête.
func (c *HaloAPIClient) rateWait(ctx context.Context) {
	if c.lastRequest.IsZero() {
		c.lastRequest = time.Now()
		return
	}
	wait := c.minInterval - time.Since(c.lastRequest)
	if wait > 0 {
		select {
		case <-ctx.Done():
		case <-time.After(wait):
		}
	}
	c.lastRequest = time.Now()
}

// backoff attend un délai exponentiel avant de retenter.
func (c *HaloAPIClient) backoff(ctx context.Context, attempt int) {
	delay := retryBaseDelay * (1 << attempt)
	if delay > 10*time.Second {
		delay = 10 * time.Second
	}
	select {
	case <-ctx.Done():
	case <-time.After(delay):
	}
}

// GetCareerRank récupère la progression du rang carrière via l'API Economy player-gated.
// Retourne (nil, nil) si le token est absent ou insuffisant (401/403).
func (c *HaloAPIClient) GetCareerRank(ctx context.Context, xuid string) (*CareerRankData, error) {
	if strings.TrimSpace(xuid) == "" {
		return nil, errors.New("GetCareerRank: xuid vide")
	}
	progressURL := fmt.Sprintf(
		"%s/hi/players/xuid(%s)/rewardtracks/careerranks/careerrank1",
		c.economyHost(),
		url.PathEscape(xuid),
	)
	progressBody, ok, err := c.doPlayerGatedGet(ctx, progressURL)
	if err != nil {
		return nil, fmt.Errorf("GetCareerRank progression: %w", err)
	}
	if !ok {
		return nil, nil
	}

	data, err := parseCareerProgressPayload(progressBody, xuid)
	if err != nil {
		return nil, fmt.Errorf("GetCareerRank progression decode: %w", err)
	}
	if data == nil {
		return nil, nil
	}

	customizationURL := fmt.Sprintf(
		"%s/hi/players/xuid(%s)/customization?view=public",
		c.economyHost(),
		url.PathEscape(xuid),
	)
	customizationBody, ok, err := c.doPlayerGatedGet(ctx, customizationURL)
	if err == nil && ok {
		serviceTag, parseErr := parseCustomizationServiceTag(customizationBody)
		if parseErr == nil {
			data.SpartanID = serviceTag
		}
	}

	return data, nil
}

func (c *HaloAPIClient) economyHost() string {
	if strings.TrimSpace(c.economyBaseURL) == "" {
		return "https://economy.svc.halowaypoint.com"
	}
	return strings.TrimRight(c.economyBaseURL, "/")
}

func (c *HaloAPIClient) doPlayerGatedGet(ctx context.Context, rawURL string) ([]byte, bool, error) {
	body, err := c.doGet(ctx, rawURL)
	if err != nil {
		if isAuthErr(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return body, true, nil
}

func isAuthErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return contains(s, "HTTP 401") || contains(s, "HTTP 403")
}

func parseCareerProgressPayload(body []byte, xuid string) (*CareerRankData, error) {
	type progress struct {
		Rank              int  `json:"Rank"`
		PartialProgress   int  `json:"PartialProgress"`
		HasReachedMaxRank bool `json:"HasReachedMaxRank"`
	}
	type alternateTrack struct {
		Result struct {
			CurrentProgress *progress `json:"CurrentProgress"`
		} `json:"Result"`
	}
	var payload struct {
		CurrentProgress *progress        `json:"CurrentProgress"`
		RewardTracks    []alternateTrack `json:"RewardTracks"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	current := payload.CurrentProgress
	if current == nil {
		for _, track := range payload.RewardTracks {
			if track.Result.CurrentProgress != nil {
				current = track.Result.CurrentProgress
				break
			}
		}
	}
	if current == nil {
		return nil, nil
	}

	return &CareerRankData{
		XUID:        xuid,
		CurrentRank: current.Rank,
		CurrentXP:   current.PartialProgress,
		IsMaxRank:   current.HasReachedMaxRank,
	}, nil
}

func parseCustomizationServiceTag(body []byte) (string, error) {
	var payload struct {
		Appearance struct {
			ServiceTag string `json:"ServiceTag"`
		} `json:"Appearance"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	return strings.TrimSpace(payload.Appearance.ServiceTag), nil
}
