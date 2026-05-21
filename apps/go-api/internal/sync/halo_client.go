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
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const (
	haloStatsHost   = "https://halostats.svc.halowaypoint.com:443"
	haloGameCMSHost = "https://gamecms-hacs.svc.halowaypoint.com"
	// maxRetries est le nombre de tentatives avant abandon (portage de tries=4 Python).
	maxRetries = 4
	// retryBaseDelay est le délai de base pour le backoff exponentiel.
	retryBaseDelay = 800 * time.Millisecond
	// matchCountMax est le nombre maximum de matchs par page d'historique (API Halo).
	matchCountMax = 25
)

// HTTPError encapsule une erreur HTTP avec statusCode exposé pour inspection.
type HTTPError struct {
	StatusCode int
	URL        string
	Err        error
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.URL)
}

func (e *HTTPError) Unwrap() error {
	return e.Err
}

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
	// GetMatchSkill récupère les données skill (team_mmr, enemy_mmr, kills/deaths
	// expected) d'un match pour les XUIDs humains. Endpoint séparé du stats.
	// Retourne map vide (pas d'erreur) si l'endpoint répond 404/410.
	GetMatchSkill(ctx context.Context, matchID string, xuids []string) (map[string]*MatchSkillData, error)
	// GetMatchFilm récupère les chunks REPLICATION_DATA du film d'un match.
	// Retourne (nil, false, nil) si le film est absent (404/410 — normal pour vieux matchs).
	GetMatchFilm(ctx context.Context, matchID string) (map[int]filmChunkData, bool, error)
	// GetHighlightEventsChunk télécharge le chunk highlight events (ChunkType=3) du film.
	// Retourne (data, filmMajorVersion, true, nil) si disponible.
	// Retourne (nil, 0, false, nil) si le film est absent ou sans chunk highlight events.
	GetHighlightEventsChunk(ctx context.Context, matchID string) ([]byte, int, bool, error)
	// GetCareerRank récupère la progression du rang carrière via l'API Economy.
	// Retourne (nil, nil) si le token est absent ou insuffisant.
	GetCareerRank(ctx context.Context, xuid string) (*CareerRankData, error)
	// GetPlayerCSRs récupère les classements CSR du joueur pour toutes les playlists
	// ranked d'une saison donnée (ex: "CsrSeason8"). Endpoint public (service token).
	// Retourne slice vide (pas d'erreur) si le joueur n'a aucun classement.
	GetPlayerCSRs(ctx context.Context, xuid, seasonID string) ([]PlayerPlaylistCSR, error)
}

// MatchHistoryEntry est un élément de l'historique des matchs.
// Portage de MatchHistoryItem (Python models.py).
type MatchHistoryEntry struct {
	MatchID   string
	StartTime string
}

// HaloAPIClient est le client HTTP pour l'API Halo Infinite stats.
// Thread-safe : net/http.Client + rate.Limiter sont concurrency-safe.
type HaloAPIClient struct {
	http           *http.Client
	spartanToken   string
	clearanceToken string
	economyBaseURL string
	gameCMSBaseURL string
	// limiter applique le rate-limiting (token bucket, thread-safe natif).
	limiter *rate.Limiter
	// localFilmCache est consulté avant l'API pour les manifestes et chunks
	// film. Nil = cache désactivé (comportement standard).
	localFilmCache *LocalFilmCache
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
		gameCMSBaseURL: haloGameCMSHost,
		limiter:        rate.NewLimiter(rate.Limit(requestsPerSecond), 1),
	}
}

// WithLocalFilmCache active le cache disque (manifestes + chunks REPLICATION_DATA)
// pour les méthodes GetMatchFilm / GetHighlightEventsChunk. Si nil ou si le
// répertoire n'existe pas, le cache reste désactivé.
func (c *HaloAPIClient) WithLocalFilmCache(cache *LocalFilmCache) *HaloAPIClient {
	c.localFilmCache = cache
	return c
}

// WithHTTPClient remplace le *http.Client interne — utilisé par les benchs et
// tests d'intégration pour rediriger les URLs Halo vers un httptest.Server via
// un http.RoundTripper custom. Passer nil restaure le client par défaut.
func (c *HaloAPIClient) WithHTTPClient(httpClient *http.Client) *HaloAPIClient {
	if httpClient != nil {
		c.http = httpClient
	}
	return c
}

// WithLimiter remplace le rate.Limiter interne par un limiter externe partagé.
// Utilisé par PooledHaloClient pour appliquer un rate-limit global commun à
// toutes les requêtes du pool (sinon chaque HaloAPIClient éphémère démarre
// avec son propre bucket plein → rate-limit inopérant). Passer nil est ignoré.
func (c *HaloAPIClient) WithLimiter(limiter *rate.Limiter) *HaloAPIClient {
	if limiter != nil {
		c.limiter = limiter
	}
	return c
}

// GetMatchHistory récupère une page de l'historique des matchs d'un joueur.
// Portage de SPNKrAPIClient.get_match_history() (Python api_client.py).
//
//   - gamertag  : DOIT être au format xuid(NNN) — pas un gamertag textuel.
//     Grunt StatsModule.GetMatchHistory + SPNKr et l'expérience de prod
//     confirment que /hi/players/{Gamertag}/matches retourne une réponse
//     stale figée (pas un 404). Toujours passer fmt.Sprintf("xuid(%s)", xuid).
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

// Constantes pour les types de chunks film Halo.
const (
	filmChunkTypeHeader          = 1
	filmChunkTypeReplicationData = 2
	filmChunkTypeHighlightEvents = 3
)

// filmManifest représente la réponse JSON de l'endpoint /hi/films/matches/{id}/spectate.
// Structure validée contre spnkr/models/discovery_ugc.py (FilmCustomData + FilmChunk).
type filmManifest struct {
	BlobStoragePathPrefix string `json:"BlobStoragePathPrefix"`
	CustomData            struct {
		FilmMajorVersion int         `json:"FilmMajorVersion"`
		Chunks           []filmChunk `json:"Chunks"`
	} `json:"CustomData"`
}

// filmChunk décrit un segment binaire du film Halo.
type filmChunk struct {
	Index                            int    `json:"Index"`
	ChunkType                        int    `json:"ChunkType"`
	ChunkSize                        int    `json:"ChunkSize"`
	ChunkStartTimeOffsetMilliseconds int    `json:"ChunkStartTimeOffsetMilliseconds"`
	DurationMilliseconds             int    `json:"DurationMilliseconds"`
	FileRelativePath                 string `json:"FileRelativePath"`
}

// buildChunkURL construit l'URL complète d'un chunk depuis le prefix et le chemin relatif.
func buildChunkURL(blobPrefix, fileRelativePath string) string {
	name := strings.TrimLeft(fileRelativePath, "/")
	if name == "" {
		return blobPrefix
	}
	if blobPrefix != "" && blobPrefix[len(blobPrefix)-1] != '/' {
		return blobPrefix + "/" + name
	}
	return blobPrefix + name
}

// fetchFilmManifest télécharge et décode le manifest film d'un match.
// Retourne (manifest, true, nil) si disponible, (nil, false, nil) si absent (404/410).
//
// Si un LocalFilmCache est configuré, on lit le manifest local avant l'API —
// le cache disque survit à l'expiration de l'endpoint manifest API (Halo
// purge les manifestes après quelques semaines/mois mais le cache local
// conserve les blob_prefixes valides plus longtemps via le CDN).
func (c *HaloAPIClient) fetchFilmManifest(ctx context.Context, matchID string) (*filmManifest, bool, error) {
	if !rexUUID.MatchString(matchID) {
		return nil, false, fmt.Errorf("fetchFilmManifest: matchID invalide %q", matchID)
	}

	// 1. Cache disque (Python legacy).
	if cm, err := c.localFilmCache.LoadManifest(matchID); err == nil && cm != nil {
		manifest := &filmManifest{
			BlobStoragePathPrefix: cm.BlobPrefix,
		}
		manifest.CustomData.FilmMajorVersion = 0 // legacy cache n'a pas la version
		manifest.CustomData.Chunks = make([]filmChunk, 0, len(cm.Chunks))
		for _, ch := range cm.Chunks {
			manifest.CustomData.Chunks = append(manifest.CustomData.Chunks, filmChunk{
				Index:                            ch.Index,
				ChunkType:                        ch.ChunkType,
				ChunkStartTimeOffsetMilliseconds: ch.StartMS,
				DurationMilliseconds:             ch.DurationMS,
				FileRelativePath:                 ch.FileRelativePath,
			})
		}
		return manifest, true, nil
	}

	// 2. API Halo.
	endpoint := fmt.Sprintf("%s/hi/films/matches/%s/spectate", haloUGCHost, url.PathEscape(matchID))
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		if isNotFoundErr(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("fetchFilmManifest(%s): %w", matchID, err)
	}
	var manifest filmManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, false, fmt.Errorf("fetchFilmManifest decode(%s): %w", matchID, err)
	}
	return &manifest, true, nil
}

// GetMatchFilm télécharge le manifest film d'un match et retourne les chunks REPLICATION_DATA.
// Seuls les chunks ChunkType==2 (REPLICATION_DATA) sont retournés — pour le weapon scanner.
// Retourne (chunks, true, nil) si le film est disponible.
// Retourne (nil, false, nil) si le film est absent (404/410) — normal pour vieux matchs.
func (c *HaloAPIClient) GetMatchFilm(ctx context.Context, matchID string) (map[int]filmChunkData, bool, error) {
	manifest, found, err := c.fetchFilmManifest(ctx, matchID)
	if err != nil || !found {
		return nil, found, err
	}

	result := make(map[int]filmChunkData)
	for _, chunk := range manifest.CustomData.Chunks {
		if chunk.ChunkType != filmChunkTypeReplicationData {
			continue
		}
		// Cache disque d'abord (Python legacy stocke les REPLICATION_DATA).
		if cached, cErr := c.localFilmCache.LoadChunk(matchID, chunk.Index); cErr == nil && cached != nil {
			result[chunk.Index] = filmChunkData{
				Data:       cached,
				StartMS:    chunk.ChunkStartTimeOffsetMilliseconds,
				DurationMS: chunk.DurationMilliseconds,
			}
			continue
		}
		chunkURL := buildChunkURL(manifest.BlobStoragePathPrefix, chunk.FileRelativePath)
		data, err := c.downloadBlob(ctx, chunkURL)
		if err != nil {
			return nil, false, fmt.Errorf("GetMatchFilm chunk %d(%s): %w", chunk.Index, matchID, err)
		}
		result[chunk.Index] = filmChunkData{
			Data:       data,
			StartMS:    chunk.ChunkStartTimeOffsetMilliseconds,
			DurationMS: chunk.DurationMilliseconds,
		}
	}
	if len(result) == 0 {
		return nil, false, nil
	}
	return result, true, nil
}

// GetHighlightEventsChunk télécharge le chunk highlight events (ChunkType=3) du film.
// Retourne (data, filmMajorVersion, true, nil) si disponible.
// Retourne (nil, 0, false, nil) si le film est absent ou sans chunk highlight events.
func (c *HaloAPIClient) GetHighlightEventsChunk(ctx context.Context, matchID string) ([]byte, int, bool, error) {
	manifest, found, err := c.fetchFilmManifest(ctx, matchID)
	if err != nil || !found {
		return nil, 0, found, err
	}

	for _, chunk := range manifest.CustomData.Chunks {
		if chunk.ChunkType != filmChunkTypeHighlightEvents {
			continue
		}
		// Cache disque d'abord (rarement présent — Python ne cache que
		// REPLICATION_DATA — mais on tente).
		if cached, cErr := c.localFilmCache.LoadChunk(matchID, chunk.Index); cErr == nil && cached != nil {
			return cached, manifest.CustomData.FilmMajorVersion, true, nil
		}
		chunkURL := buildChunkURL(manifest.BlobStoragePathPrefix, chunk.FileRelativePath)
		data, err := c.downloadBlob(ctx, chunkURL)
		if err != nil {
			// Fallback gracieux : si le manifest vient du cache local et que
			// le blob CDN a expiré, on retourne (nil, 0, false, nil) au lieu
			// d'une erreur — comportement equivalent à "film absent".
			if isNotFoundErr(err) {
				return nil, 0, false, nil
			}
			return nil, 0, false, fmt.Errorf("GetHighlightEventsChunk(%s): %w", matchID, err)
		}
		return data, manifest.CustomData.FilmMajorVersion, true, nil
	}
	return nil, 0, false, nil
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

// downloadBlob télécharge un blob Halo sans header d'auth (pre-signed URL)
// et le décompresse zlib (le CDN Azure des films Halo Infinite renvoie du zlib brut).
// Portage de download_film_chunk() (Python api_client.py:485-498).
func (c *HaloAPIClient) downloadBlob(ctx context.Context, blobURL string) ([]byte, error) {
	// Sprint B1 commit 18 : log download blob (film chunks, highlight events).
	// Volume potentiellement gros (>100 KB) — log les bytes en sortie pour
	// repérer les blobs anormalement gros.
	start := time.Now()
	c.rateWait(ctx)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, blobURL, nil)
	if err != nil {
		return nil, fmt.Errorf("downloadBlob new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		slog.WarnContext(ctx, "halo_api: downloadBlob échec réseau", "url", blobURL, "err", err)
		return nil, fmt.Errorf("downloadBlob: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.WarnContext(ctx, "halo_api: downloadBlob HTTP error",
			"url", blobURL, "status", resp.StatusCode, "duration_ms", time.Since(start).Milliseconds())
		return nil, fmt.Errorf("downloadBlob HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("downloadBlob read: %w", err)
	}
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("downloadBlob zlib header: %w", err)
	}
	defer zr.Close()
	out, err := io.ReadAll(zr)
	if err == nil {
		slog.DebugContext(ctx, "halo_api: downloadBlob succès",
			"url", blobURL, "bytes_compressed", len(raw), "bytes_inflated", len(out),
			"duration_ms", time.Since(start).Milliseconds())
	}
	return out, err
}

// doGet exécute un GET authentifié avec retry + backoff exponentiel.
// Portage de request_with_retries() (Python api_client.py).
func (c *HaloAPIClient) doGet(ctx context.Context, rawURL string) ([]byte, error) {
	// Sprint B1 commit 18 : log de chaque appel API sortant pour diag des
	// timeouts / 5xx / retries. event_id hérite du caller (sync.RunDelta,
	// sync.postSync, etc.) — pas besoin d'un WithEvent local. Le slog
	// inclut automatiquement event_id via ContextHandler.
	start := time.Now()
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
			slog.WarnContext(ctx, "halo_api: GET échec réseau",
				"url", rawURL, "attempt", attempt+1, "err", err)
			c.backoff(ctx, attempt)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Erreurs d'auth : inutile de retry.
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			slog.WarnContext(ctx, "halo_api: GET refusé auth (pas de retry)",
				"url", rawURL, "status", resp.StatusCode, "duration_ms", time.Since(start).Milliseconds())
			return nil, &HTTPError{StatusCode: resp.StatusCode, URL: rawURL, Err: errors.New("tokens invalides/expirés")}
		}
		// Ressource absente : ne pas retry.
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
			slog.DebugContext(ctx, "halo_api: GET ressource absente",
				"url", rawURL, "status", resp.StatusCode, "duration_ms", time.Since(start).Milliseconds())
			return nil, &HTTPError{StatusCode: resp.StatusCode, URL: rawURL, Err: errors.New("ressource absente")}
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = &HTTPError{StatusCode: resp.StatusCode, URL: rawURL, Err: fmt.Errorf("HTTP %d", resp.StatusCode)}
			slog.WarnContext(ctx, "halo_api: GET HTTP error",
				"url", rawURL, "status", resp.StatusCode, "attempt", attempt+1)
			c.backoff(ctx, attempt)
			continue
		}
		if readErr != nil {
			lastErr = readErr
			c.backoff(ctx, attempt)
			continue
		}
		slog.DebugContext(ctx, "halo_api: GET succès",
			"url", rawURL, "bytes", len(body), "duration_ms", time.Since(start).Milliseconds(), "attempt", attempt+1)
		return body, nil
	}
	slog.ErrorContext(ctx, "halo_api: GET échec définitif",
		"url", rawURL, "attempts", maxRetries, "duration_ms", time.Since(start).Milliseconds(), "err", lastErr)
	return nil, fmt.Errorf("doGet %s: %d tentatives échouées — %w", rawURL, maxRetries, lastErr)
}

// rateWait bloque jusqu'à ce qu'un token soit disponible dans le bucket.
// Thread-safe : rate.Limiter sérialise les accès concurrents nativement.
func (c *HaloAPIClient) rateWait(ctx context.Context) {
	if c.limiter == nil {
		return
	}
	_ = c.limiter.Wait(ctx)
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

// SpartanCustomizationData contient les données d'identité visuelle Spartan
// résolues depuis l'endpoint /customization?view=public (ServiceTag + images
// banner/emblem/backdrop). Les URLs sont vides si le resolve GameCMS a échoué
// (le front affiche un placeholder).
type SpartanCustomizationData struct {
	SpartanID        string
	BannerImageURL   string
	EmblemImageURL   string
	BackdropImageURL string
}

// GetCareerProgress récupère uniquement la progression de rang carrière
// (rang, XP courante, IsMaxRank) via l'API Economy player-gated.
// Retourne (nil, nil) si le token est absent/insuffisant (401/403) ou si la
// réponse ne contient pas de CurrentProgress.
//
// 2026-05-14 — endpoint mis à jour vers la nouvelle forme Halo Waypoint :
// `/hi/careerranks/careerRank1?players=xuid(X)` retourne un
// RewardTrackResultContainer avec RewardTracks[].Result.CurrentProgress.
// L'ancien path `/hi/players/xuid(X)/rewardtracks/careerranks/careerrank1`
// timeout depuis quelques jours (probablement déprécié). Source : projet
// Grunt API (github.com/dend/grunt — EconomyModule.GetPlayerCareerRank).
// Le parser parseCareerProgressPayload gère déjà cette forme (path alterne
// RewardTracks[].Result.CurrentProgress).
func (c *HaloAPIClient) GetCareerProgress(ctx context.Context, xuid string) (*CareerRankData, error) {
	if strings.TrimSpace(xuid) == "" {
		return nil, errors.New("GetCareerProgress: xuid vide")
	}
	progressURL := fmt.Sprintf(
		"%s/hi/careerranks/careerRank1?players=xuid(%s)",
		c.economyHost(),
		url.PathEscape(xuid),
	)
	progressBody, ok, err := c.doPlayerGatedGet(ctx, progressURL)
	if err != nil {
		return nil, fmt.Errorf("GetCareerProgress: %w", err)
	}
	if !ok {
		return nil, nil
	}
	data, err := parseCareerProgressPayload(progressBody, xuid)
	if err != nil {
		return nil, fmt.Errorf("GetCareerProgress decode: %w", err)
	}
	return data, nil
}

// GetSpartanCustomization récupère uniquement la customisation Spartan
// (ServiceTag, banner, emblem, backdrop) via l'API Economy player-gated.
// Retourne (nil, nil) si le token est absent/insuffisant (401/403) ou si la
// réponse est vide.
//
// 2026-05-14 — endpoint mis à jour vers `/customization/appearance` (l'ancien
// `/customization?view=public` timeout depuis quelques jours, probablement
// déprécié). Source : projet Grunt API (github.com/dend/grunt —
// EconomyModule.PlayerAppearanceCustomization). La réponse est de la forme
// {Status, Appearance:{ServiceTag, BackdropImagePath, Emblem:{EmblemPath},
// PlayerTitlePath, ...}} — exactement ce que parseCustomizationAppearance
// sait déjà décoder.
//
// 2026-05-08 — pattern Grunt strict : pas d'invention d'URL en cas d'échec
// resolve (ex-fallbackCustomization* retiraient la résolution canonique au
// profit d'URLs /Waypoint/file/images/... qui n'existent pas sur Microsoft
// GameCMS → 403). Si le resolve échoue, on log warn et on laisse vide.
func (c *HaloAPIClient) GetSpartanCustomization(ctx context.Context, xuid string) (*SpartanCustomizationData, error) {
	if strings.TrimSpace(xuid) == "" {
		return nil, errors.New("GetSpartanCustomization: xuid vide")
	}
	customizationURL := fmt.Sprintf(
		"%s/hi/players/xuid(%s)/customization/appearance",
		c.economyHost(),
		url.PathEscape(xuid),
	)
	customizationBody, ok, err := c.doPlayerGatedGet(ctx, customizationURL)
	if err != nil {
		return nil, fmt.Errorf("GetSpartanCustomization: %w", err)
	}
	if !ok {
		return nil, nil
	}
	appearance, err := parseCustomizationAppearance(customizationBody)
	if err != nil {
		return nil, fmt.Errorf("GetSpartanCustomization decode: %w", err)
	}
	if appearance == nil {
		return nil, nil
	}

	out := &SpartanCustomizationData{SpartanID: appearance.ServiceTag}
	if appearance.BannerImagePath != "" {
		if resolved, resolveErr := c.resolveCustomizationImageURL(ctx, appearance.BannerImagePath); resolveErr == nil {
			out.BannerImageURL = resolved
		} else {
			slog.WarnContext(ctx, "spartan_id: banner image resolve failed",
				"inventory_path", appearance.BannerImagePath, "err", resolveErr)
		}
	}
	// Fallback nameplate : Halo /customization/appearance ne retourne pas
	// toujours BannerImagePath. Port du flow Python `resolve_positive_emblem_cfg`
	// (ResolveNameplateURL dans spartan_nameplate_resolver.go) — fetch JSON
	// emblem CMS, parse AvailableConfigurations[], prend le 1er cfg > 0,
	// construit URL `/hi/Waypoint/file/images/nameplates/<stem>_<cfg>.png`.
	if out.BannerImageURL == "" && appearance.EmblemPath != "" {
		cfg, _ := strconv.ParseInt(strings.TrimSpace(appearance.EmblemConfigurationID), 10, 64)
		if url := ResolveNameplateURL(ctx, appearance.EmblemPath, cfg, c.spartanToken, c.clearanceToken); url != "" {
			out.BannerImageURL = url
		}
	}
	if appearance.EmblemPath != "" {
		if resolved, resolveErr := c.resolveCustomizationImageURL(ctx, appearance.EmblemPath); resolveErr == nil {
			out.EmblemImageURL = resolved
		} else {
			slog.WarnContext(ctx, "spartan_id: emblem image resolve failed",
				"inventory_path", appearance.EmblemPath, "err", resolveErr)
		}
	}
	if appearance.BackdropImagePath != "" {
		if resolved, resolveErr := c.resolveCustomizationImageURL(ctx, appearance.BackdropImagePath); resolveErr == nil {
			out.BackdropImageURL = resolved
		} else {
			slog.WarnContext(ctx, "spartan_id: backdrop image resolve failed",
				"inventory_path", appearance.BackdropImagePath, "err", resolveErr)
		}
	}
	return out, nil
}

// GetCareerRank récupère la progression du rang carrière combinée à la
// customisation Spartan (1 appel d'orchestration = 2 endpoints en série).
//
// Deprecated: appeler GetCareerProgress et GetSpartanCustomization séparément
// pour découpler les cadences de refresh (XP live throttled vs customization
// 6h TTL). Conservé pour compat avec PooledHaloClient et tests legacy.
func (c *HaloAPIClient) GetCareerRank(ctx context.Context, xuid string) (*CareerRankData, error) {
	data, err := c.GetCareerProgress(ctx, xuid)
	if err != nil || data == nil {
		return data, err
	}
	// Customization: échec silencieux (préserve la sémantique antérieure).
	if custom, cerr := c.GetSpartanCustomization(ctx, xuid); cerr == nil && custom != nil {
		if custom.SpartanID != "" {
			data.SpartanID = custom.SpartanID
		}
		data.BannerImageURL = custom.BannerImageURL
		data.EmblemImageURL = custom.EmblemImageURL
		data.BackdropImageURL = custom.BackdropImageURL
	}
	return data, nil
}

func (c *HaloAPIClient) economyHost() string {
	if strings.TrimSpace(c.economyBaseURL) == "" {
		return "https://economy.svc.halowaypoint.com"
	}
	return strings.TrimRight(c.economyBaseURL, "/")
}

func (c *HaloAPIClient) gameCMSHost() string {
	if strings.TrimSpace(c.gameCMSBaseURL) == "" {
		return haloGameCMSHost
	}
	return strings.TrimRight(c.gameCMSBaseURL, "/")
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

type customizationAppearance struct {
	ServiceTag            string
	BannerImagePath       string
	BackdropImagePath     string
	EmblemPath            string
	EmblemConfigurationID string
}

// Clés JSON Halo API customization payload — utilisées par
// parseCustomizationAppearance / extractCustomizationMediaPath pour
// naviguer dans la réponse imbriquée (Appearance.Banner.ImagePath, etc.).
const (
	jsonKeyAppearance  = "Appearance"
	jsonKeyEmblem      = "Emblem"
	jsonKeyEmblemPath  = "EmblemPath"
	jsonKeyCommonData  = "CommonData"
	jsonKeyImagePath   = "ImagePath"
	jsonKeyPath        = "Path"
	jsonKeyDisplayPath = "DisplayPath"
	jsonKeyMedia       = "Media"
	jsonKeyMediaURL    = "MediaUrl"
)

func parseCustomizationAppearance(body []byte) (*customizationAppearance, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	return &customizationAppearance{
		ServiceTag: firstNonEmptyPayloadString(payload,
			[]string{jsonKeyAppearance, "ServiceTag"},
		),
		BannerImagePath: firstNonEmptyPayloadString(payload,
			[]string{jsonKeyAppearance, "BannerImagePath"},
			[]string{jsonKeyAppearance, "NameplateImagePath"},
			[]string{jsonKeyAppearance, "PlayerTitlePath"},
			[]string{jsonKeyAppearance, "Nameplate", "NameplateImagePath"},
			[]string{jsonKeyAppearance, "Nameplate", jsonKeyImagePath},
			[]string{jsonKeyAppearance, "Nameplate", jsonKeyPath},
			[]string{jsonKeyAppearance, "Banner", "BannerImagePath"},
			[]string{jsonKeyAppearance, "Banner", jsonKeyImagePath},
			[]string{jsonKeyAppearance, "Banner", jsonKeyPath},
		),
		BackdropImagePath: firstNonEmptyPayloadString(payload,
			[]string{jsonKeyAppearance, "BackdropImagePath"},
		),
		EmblemPath: firstNonEmptyPayloadString(payload,
			[]string{jsonKeyAppearance, jsonKeyEmblem, jsonKeyEmblemPath},
			[]string{jsonKeyAppearance, jsonKeyEmblemPath},
		),
		EmblemConfigurationID: stringifyCustomizationConfigurationID(firstNonEmptyPayloadValue(payload,
			[]string{jsonKeyAppearance, jsonKeyEmblem, "ConfigurationId"},
			[]string{jsonKeyAppearance, jsonKeyEmblem, "ConfigurationID"},
		)),
	}, nil
}

func firstNonEmptyPayloadString(payload map[string]any, keySets ...[]string) string {
	for _, keys := range keySets {
		if value := nestedPayloadString(payload, keys...); value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmptyPayloadValue(payload map[string]any, keySets ...[]string) any {
	for _, keys := range keySets {
		if value := nestedPayloadValue(payload, keys...); value != nil {
			return value
		}
	}
	return nil
}

func stringifyCustomizationConfigurationID(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case json.Number:
		return v.String()
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func (c *HaloAPIClient) resolveCustomizationImageURL(ctx context.Context, inventoryPath string) (string, error) {
	trimmed := strings.TrimSpace(strings.TrimLeft(inventoryPath, "/"))
	if trimmed == "" {
		return "", fmt.Errorf("resolveCustomizationImageURL: inventory path vide")
	}

	endpoint := fmt.Sprintf("%s/hi/progression/file/%s", c.gameCMSHost(), trimmed)
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return "", err
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}

	mediaPath := extractCustomizationMediaPath(payload)
	if mediaPath == "" {
		return "", fmt.Errorf("resolveCustomizationImageURL: media path absent")
	}
	return buildCustomizationImageURL(c.gameCMSHost(), mediaPath), nil
}

func extractCustomizationMediaPath(payload map[string]any) string {
	paths := [][]string{
		{jsonKeyCommonData, jsonKeyDisplayPath, jsonKeyMedia, jsonKeyMediaURL, jsonKeyPath},
		{jsonKeyDisplayPath, jsonKeyMedia, jsonKeyMediaURL, jsonKeyPath},
		{jsonKeyImagePath, jsonKeyMedia, jsonKeyMediaURL, jsonKeyPath},
		{jsonKeyCommonData, jsonKeyImagePath, jsonKeyMedia, jsonKeyMediaURL, jsonKeyPath},
	}
	for _, keys := range paths {
		if value := nestedPayloadString(payload, keys...); value != "" {
			return value
		}
	}
	return ""
}

func nestedPayloadString(payload map[string]any, keys ...string) string {
	value, ok := nestedPayloadValue(payload, keys...).(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func nestedPayloadValue(payload map[string]any, keys ...string) any {
	current := any(payload)
	for _, key := range keys {
		asMap, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		next, ok := asMap[key]
		if !ok {
			return nil
		}
		current = next
	}
	return current
}

func buildCustomizationImageURL(baseURL, mediaPath string) string {
	trimmedBase := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	trimmedPath := strings.TrimSpace(mediaPath)
	if trimmedBase == "" || trimmedPath == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(trimmedPath), "http://") || strings.HasPrefix(strings.ToLower(trimmedPath), "https://") {
		return trimmedPath
	}
	trimmedPath = strings.TrimLeft(trimmedPath, "/")
	if strings.HasPrefix(strings.ToLower(trimmedPath), "hi/images/file/") {
		return trimmedBase + "/" + trimmedPath
	}
	if strings.HasPrefix(strings.ToLower(trimmedPath), "images/file/") {
		return trimmedBase + "/hi/" + trimmedPath
	}
	return trimmedBase + "/hi/images/file/" + trimmedPath
}

// (2026-05-08) Les fonctions `fallbackCustomization{Emblem,Backdrop,Banner}URL`
// + `fallbackCustomizationBannerFromEmblem` + `customizationInventoryStem` ont
// été SUPPRIMÉES. Elles inventaient des URLs au format
// `/hi/Waypoint/file/images/{kind}/{stem}.png` qui n'existent pas côté
// Microsoft GameCMS — l'API retourne 403 systématiquement. Le seul pattern
// correct, aligné sur Grunt API
// (https://github.com/dend/grunt — endpoints `GameCms_GetProgressionFile` +
// `GameCms_GetProgressionImage`), est :
//
//  1. Fetch JSON descriptor : `GET /hi/progression/file/{InventoryPath}` →
//     champ `CommonData.DisplayPath.Media.MediaUrl.Path` (ou variantes).
//  2. Fetch image : `GET /hi/images/file/{MediaPath}`.
//
// Implémenté par `resolveCustomizationImageURL` ci-dessus (ligne ~733). Quand
// le resolve échoue (auth absente, JSON malformé, asset Microsoft retiré),
// on stocke chaîne vide → le frontend affiche un placeholder "image
// indisponible" au lieu d'une URL inventée qui finirait en 403/502.
