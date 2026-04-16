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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	haloStatsHost = "https://halostats.svc.halowaypoint.com:443"
	// maxRetries est le nombre de tentatives avant abandon (portage de tries=4 Python).
	maxRetries = 4
	// retryBaseDelay est le délai de base pour le backoff exponentiel.
	retryBaseDelay = 800 * time.Millisecond
)

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
	ChunkUrl                         string `json:"ChunkUrl"`
}

// GetMatchFilm télécharge le manifest film d'un match et retourne les chunks indexés.
// Retourne (chunks, true, nil) si le film est disponible.
// Retourne (nil, false, nil) si le film est absent (404/410) — normal pour vieux matchs.
func (c *HaloAPIClient) GetMatchFilm(ctx context.Context, matchID string) (map[int]filmChunkData, bool, error) {
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
		data, err := c.downloadBlob(ctx, chunk.ChunkUrl)
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
