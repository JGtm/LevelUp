package haloadapter

// client.go — le client de synchronisation Halo Infinite : jetons d'UN joueur (instance), limiteur
// (propre ou partagé par le pool), politique de reprise `infinite.RetryHaloClient` (429 rendu
// aussitôt avec sa fenêtre Retry-After : le pool gère le refroidissement), cache disque des films.
// Historique et stats des matchs ici ; carrière et apparence (client_career.go), skill et CSR
// (client_skill.go), playlists, variantes de carte et films dans leurs fichiers.

import (
	"context"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/haloapi/infinite"
	"levelup/go-api/internal/sync/filmfetch"
)

const (
	// clientHTTPTimeout : délai de chaque requête du client (porté par son client HTTP).
	clientHTTPTimeout = 20 * time.Second
	// clientDefaultRPS : débit par défaut quand l'appelant n'en donne pas.
	clientDefaultRPS = 10
)

// HTTPError : réponse non-200 d'un service Halo, typée (statut, adresse, fenêtre Retry-After) ;
// lue par le pool (`errors.As`) pour marquer un jeton ou refroidir.
type HTTPError = infinite.HTTPError

// HaloClient est l'interface du client HTTP Halo Infinite, consommée par la synchronisation (et
// ses doublures de test).
type HaloClient interface {
	// GetMatchHistory récupère une page d'historique de matchs d'un joueur.
	GetMatchHistory(ctx context.Context, gamertag, matchType string, start, count int) ([]MatchHistoryEntry, error)
	// GetMatchStats récupère les stats détaillées d'un match (JSON brut).
	GetMatchStats(ctx context.Context, matchID string) (map[string]any, error)
	// GetMatchSkill récupère le skill d'un match pour les xuid humains ; map vide (pas d'erreur)
	// si le service répond 404/410.
	GetMatchSkill(ctx context.Context, matchID string, xuids []string) (map[string]*MatchSkillData, error)
	// GetMatchFilm récupère les chunks REPLICATION_DATA du film d'un match ; (nil, false, nil) si
	// le film est absent (404/410 — normal pour les vieux matchs).
	GetMatchFilm(ctx context.Context, matchID string) (map[int]filmfetch.FilmChunkData, bool, error)
	// GetHighlightEventsChunk télécharge le chunk des temps forts du film ; (nil, 0, false, nil) si
	// le film est absent ou sans ce chunk.
	GetHighlightEventsChunk(ctx context.Context, matchID string) ([]byte, int, bool, error)
	// GetCareerRank récupère la progression du rang carrière ; (nil, nil) si le jeton est absent
	// ou insuffisant.
	GetCareerRank(ctx context.Context, xuid string) (*domain.CareerRankSnapshot, error)
	// GetPlayerCSRs récupère les classements CSR du joueur pour toutes les playlists classées
	// jouées d'une saison ; tranche vide (pas d'erreur) sans classement.
	GetPlayerCSRs(ctx context.Context, xuid, seasonID string) ([]PlayerPlaylistCSR, error)
	// GetPlaylistCsr récupère le CSR d'un joueur pour UNE playlist classée, même jamais jouée ;
	// (nil, nil) sans entrée.
	GetPlaylistCsr(ctx context.Context, playlistID, xuid, seasonID string) (*PlayerPlaylistCSR, error)
}

// MatchHistoryEntry est un élément de l'historique des matchs.
type MatchHistoryEntry struct {
	MatchID   string
	StartTime string
}

// Client est le client de synchronisation Halo Infinite. Sûr en concurrence (`*http.Client` et
// `*rate.Limiter` le sont).
type Client struct {
	// http émet les requêtes (derrière la garde netguard de la surface) et porte le délai.
	http           *http.Client
	spartanToken   string
	clearanceToken string
	// Surcharges d'hôte d'instance (tests) : vides en production.
	economyBaseURL string
	gameCMSBaseURL string
	// limiter borne le débit (seau à jetons, sûr en concurrence) ; nil : aucun.
	limiter *rate.Limiter
	// localFilmCache est consulté avant l'API pour les manifestes et chunks de film ; nil :
	// désactivé.
	localFilmCache *filmfetch.LocalFilmCache
	// endpoints : résolveur d'hôtes d'instance (tests) prioritaire sur le résolveur partagé du
	// boot ; nil : le résolveur partagé, puis les adresses par défaut.
	endpoints games.EndpointResolver
}

// NewClient crée un client authentifié par les jetons du joueur ; requestsPerSecond borne son
// débit (<= 0 : 10).
func NewClient(spartanToken, clearanceToken string, requestsPerSecond int) *Client {
	if requestsPerSecond <= 0 {
		requestsPerSecond = clientDefaultRPS
	}
	return &Client{
		http:           &http.Client{Timeout: clientHTTPTimeout},
		spartanToken:   spartanToken,
		clearanceToken: clearanceToken,
		limiter:        rate.NewLimiter(rate.Limit(requestsPerSecond), 1),
	}
}

// WithLocalFilmCache active le cache disque des films (manifestes et chunks) ; nil le laisse
// désactivé.
func (c *Client) WithLocalFilmCache(cache *filmfetch.LocalFilmCache) *Client {
	c.localFilmCache = cache
	return c
}

// WithHTTPClient remplace le client HTTP (benchs, tests d'intégration vers un httptest.Server par
// un transport réécrivant) ; nil est ignoré.
func (c *Client) WithHTTPClient(httpClient *http.Client) *Client {
	if httpClient != nil {
		c.http = httpClient
	}
	return c
}

// WithEndpoints pose un résolveur d'hôtes d'instance, prioritaire sur le résolveur partagé du
// boot ; nil restaure le résolveur partagé.
func (c *Client) WithEndpoints(r games.EndpointResolver) *Client {
	c.endpoints = r
	return c
}

// WithLimiter remplace le limiteur par un limiteur externe partagé (le pool applique un débit
// commun à tous ses clients éphémères, sinon chacun démarrerait seau plein) ; nil est ignoré.
func (c *Client) WithLimiter(limiter *rate.Limiter) *Client {
	if limiter != nil {
		c.limiter = limiter
	}
	return c
}

// Limiter rend le limiteur des requêtes de ce client (celui du créneau du pool quand il en a reçu
// un).
func (c *Client) Limiter() *rate.Limiter { return c.limiter }

// adresses rend les adresses de la brique pour le titre du contexte : surcharges d'instance,
// puis résolveur d'instance, puis résolveur partagé, puis défauts.
func (c *Client) adresses(ctx context.Context, cles ...infinite.Cle) infinite.Endpoints {
	res := c.endpoints
	if res == nil {
		res = games.DefaultEndpointResolver()
	}
	s := surcharges{
		infinite.CleEconomy: strings.TrimSpace(c.economyBaseURL),
		infinite.CleGameCMS: strings.TrimSpace(c.gameCMSBaseURL),
	}
	return s.appliquer(EndpointsDepuisResolveur(ctx, res, ctxkeys.TitleSlug(ctx), s.restantes(cles)...))
}

// api rend le client de la brique pour un appel : sortie gardée sous `surface`, politique
// `RetryHaloClient`, limiteur du client, adresses des clés `cles`.
func (c *Client) api(ctx context.Context, surface string, cles ...infinite.Cle) *infinite.Client {
	api := infinite.New(netguardDoer{surface: surface, next: c.http}, infinite.RetryHaloClient()).
		WithEndpoints(c.adresses(ctx, cles...))
	if c.limiter != nil {
		api = api.WithLimiter(c.limiter)
	}
	return api
}

// auth rend les jetons du client.
func (c *Client) auth() infinite.Auth {
	return infinite.Auth{Spartan: c.spartanToken, Clearance: c.clearanceToken}
}

// GetMatchHistory récupère une page de l'historique des matchs d'un joueur.
//
//   - gamertag  : DOIT être au format xuid(NNN) — `/players/{Gamertag}/matches` rend une réponse
//     périmée figée, pas un 404 ;
//   - matchType : "all", "matchmaking", "custom", "local" ;
//   - start     : décalage (0 = plus récent) ; count : 1..25.
func (c *Client) GetMatchHistory(ctx context.Context, gamertag, matchType string, start, count int) ([]MatchHistoryEntry, error) {
	dto, err := c.api(ctx, surfaceAPIGet, infinite.CleStats).GetMatchHistory(ctx, c.auth(), gamertag, matchType, start, count)
	if err != nil {
		return nil, err
	}
	entries := make([]MatchHistoryEntry, 0, len(dto.Results))
	for _, r := range dto.Results {
		entries = append(entries, MatchHistoryEntry{MatchID: r.MatchID, StartTime: r.MatchInfo.StartTime})
	}
	return entries, nil
}

// GetMatchStats récupère les stats détaillées d'un match (JSON brut).
func (c *Client) GetMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	return c.api(ctx, surfaceAPIGet, infinite.CleStats).GetMatchStats(ctx, c.auth(), matchID)
}
