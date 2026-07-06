// Package sync — halo_client.go : client HTTP pour l'API Halo Infinite stats.
//
// Portage de SPNKrAPIClient.get_match_history() + get_match_stats() (Python api_client.py).
// Ce client est stateless : instancie pour chaque session de sync avec les tokens du joueur.
//
// Endpoints utilises :
//
//	GET https://halostats.svc.halowaypoint.com:443/hi/players/{gamertag}/matches
//	GET https://halostats.svc.halowaypoint.com:443/hi/matches/{match_id}/stats
//
// Headers requis (portage de spnkr/client.py) :
//
//	Accept: application/json
//	x-343-authorization-spartan: {SpartanToken}
//	343-clearance: {ClearanceToken}
//
// Le code est decoupe en fichiers thematiques pour respecter la limite des
// 500 lignes par fichier (CLAUDE.md). Ce fichier contient les types core,
// le constructeur, les Withers et les endpoints match (history/stats).
// Les autres responsabilites vivent dans :
//
//   - halo_client_film.go   : film manifest + chunk download +
//     highlight events extraction
//   - halo_client_http.go   : helpers HTTP (downloadBlob, doGet,
//     rateWait, backoff)
//   - halo_client_career.go : endpoints career progression + spartan
//     customization + helpers de parsing
package haloclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"levelup/go-api/internal/games"
)

const (
	haloStatsHost   = "https://halostats.svc.halowaypoint.com:443"
	haloGameCMSHost = "https://gamecms-hacs.svc.halowaypoint.com"
	// maxRetries est le nombre de tentatives avant abandon (portage de tries=4 Python).
	maxRetries = 4
	// retryBaseDelay est le délai de base pour le backoff exponentiel.
	retryBaseDelay = 800 * time.Millisecond
	// backoffCeiling borne le délai d'un retry in-client : au-delà, c'est le
	// cooldown global du pool (OnHTTPError) qui prend le relais.
	backoffCeiling = 10 * time.Second
	// matchCountMax est le nombre maximum de matchs par page d'historique (API Halo).
	matchCountMax = 25
)

// HTTPError encapsule une erreur HTTP avec statusCode exposé pour inspection.
type HTTPError struct {
	StatusCode int
	URL        string
	Err        error
	// RetryAfter est la durée demandée par le header HTTP Retry-After (429/503),
	// déjà parsée et bornée. 0 = header absent/non parsable → le pool applique sa
	// politique de cooldown par défaut (globalCooldown + backoff exponentiel).
	RetryAfter time.Duration
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.URL)
}

func (e *HTTPError) Unwrap() error {
	return e.Err
}

// IsAuthError indique si err provient d'un refus d'authentification Waypoint (401/403)
// du client sync. Permet aux chemins live token-gated (career, CSR Explorer,
// recent-matches) de déclencher le filet defense-in-depth (re-mint + retry unique) via
// halo.RetryWithFreshTokens, comme le provider halo le fait avec son propre sentinel.
func IsAuthError(err error) bool {
	var he *HTTPError
	if errors.As(err, &he) {
		return he.StatusCode == http.StatusUnauthorized || he.StatusCode == http.StatusForbidden
	}
	return false
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
	GetMatchFilm(ctx context.Context, matchID string) (map[int]FilmChunkData, bool, error)
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
	// GetPlaylistCsr récupère le CSR d'un joueur pour UNE playlist classée (mécanisme
	// Grunt). Marche pour n'importe quelle playlist/saison — renvoie "Non classé"
	// (nil, nil) si jamais jouée. Permet d'afficher toutes les playlists classées
	// sans dériver de l'historique du joueur.
	GetPlaylistCsr(ctx context.Context, playlistID, xuid, seasonID string) (*PlayerPlaylistCSR, error)
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
	// endpoints override le resolver d'hosts partagé (MT-01). Nil = utilise le
	// resolver de boot (sharedEndpointResolver) puis fallback const legacy.
	// Injecté via WithEndpoints, surtout pour les tests de routing synthétique.
	endpoints games.EndpointResolver
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
		// economyBaseURL/gameCMSBaseURL laissés vides : la résolution d'host passe
		// par economyHost(ctx)/gameCMSHost(ctx) → EndpointResolver title-aware
		// (fallback const Halo). Non vides = override d'instance (tests).
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), 1),
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

// WithEndpoints injecte un resolver d'hosts d'instance (MT-01), prioritaire sur
// le resolver partagé de boot. Utilisé par les tests pour router vers des hosts
// synthétiques. Passer nil restaure le comportement par défaut (resolver partagé).
func (c *HaloAPIClient) WithEndpoints(r games.EndpointResolver) *HaloAPIClient {
	c.endpoints = r
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
	endpoint := fmt.Sprintf("%s/%s/players/%s/matches", c.hostFor(ctx, games.EndpointStats, haloStatsHost), c.gamePrefix(ctx), url.PathEscape(gamertag))
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
	endpoint := fmt.Sprintf("%s/%s/matches/%s/stats", c.hostFor(ctx, games.EndpointStats, haloStatsHost), c.gamePrefix(ctx), url.PathEscape(matchID))
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
