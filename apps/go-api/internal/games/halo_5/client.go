// Package halo_5 — adapter multi-titre EXPERIMENTAL pour Halo 5: Guardians.
//
// client.go : l'adaptateur du client Halo 5 de la brique « API Halo »
// (`internal/haloapi/halo5`, qui porte la recette du protocole : hôtes, segment `h5`,
// en-têtes, `?auth=st`, reprise). Ici l'exploitation : le jeton Spartan v4 du joueur
// (le même que celui du pool Infinite), le client HTTP et son délai, le limiteur, la
// garde `netguard` (mode démo) par surface, la politique de reprise choisie.
//
// Instancier par session avec le SpartanToken du joueur. Thread-safe (net/http.Client
// + rate.Limiter concurrency-safe).
package halo_5

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/time/rate"

	"levelup/go-api/internal/haloapi/halo5"
	"levelup/go-api/internal/platform/netguard"
)

// TitleSlug est l'identite canonique du titre Halo 5 (constante de package
// title-specific, pas une comparaison de slug — archlint no_slug_comparison OK).
const TitleSlug = "halo_5"

const (
	// Surfaces sortantes : une par chemin de la brique (JSON, rendus PNG), chacune avec sa
	// ligne dans le journal de `netguard`.
	surfaceGet       = "halo5_api.get"
	surfaceGetBinary = "halo5_api.get_binary"
	// clientHTTPTimeout : délai de chaque requête (porté par le client HTTP).
	clientHTTPTimeout = 20 * time.Second
	// clientDefaultRPS : débit par défaut quand l'appelant n'en donne pas.
	clientDefaultRPS = 10
)

// Client est le client live Halo 5 d'UN joueur : la brique et le jeton passé à chaque appel.
type Client struct {
	api  *halo5.Client
	auth halo5.Auth
}

// NewClient cree un client authentifie avec le SpartanToken v4 du joueur.
// requestsPerSecond <= 0 -> defaut 10. PAS de clearanceToken (Halo 5 ne l'utilise pas).
func NewClient(spartanToken string, requestsPerSecond int) *Client {
	if requestsPerSecond <= 0 {
		requestsPerSecond = clientDefaultRPS
	}
	api := nouvelleAPI(&http.Client{Timeout: clientHTTPTimeout}).
		WithLimiter(rate.NewLimiter(rate.Limit(requestsPerSecond), 1))
	return &Client{api: api, auth: halo5.Auth{Spartan: spartanToken}}
}

// nouvelleAPI rend le client de la brique dont les deux chemins sortent par `next`, gardés par
// netguard sous leur surface, avec la politique de reprise de Halo 5.
func nouvelleAPI(next *http.Client) *halo5.Client {
	return halo5.New(
		netguard.Doer{Surface: surfaceGet, Next: next},
		netguard.Doer{Surface: surfaceGetBinary, Next: next},
		halo5.RetryHalo5(),
	)
}

// WithHTTPClient remplace le *http.Client des deux chemins (tests httptest via
// RoundTripper) ; adresses et limiteur sont conservés. nil ignore. Chainable.
func (c *Client) WithHTTPClient(h *http.Client) *Client {
	if h != nil {
		c.api = nouvelleAPI(h).WithEndpoints(c.api.Endpoints()).WithLimiter(c.api.Limiter())
	}
	return c
}

// WithLimiter partage un rate.Limiter externe (pool commun). nil ignore. Chainable.
func (c *Client) WithLimiter(l *rate.Limiter) *Client {
	if l != nil {
		c.api = c.api.WithLimiter(l)
	}
	return c
}

// WithStatsBaseURL override le host stats (tests httptest : rediriger vers srv.URL).
// Vide ignore. Chainable.
func (c *Client) WithStatsBaseURL(statsURL string) *Client {
	if statsURL != "" {
		e := c.api.Endpoints()
		e.Stats = statsURL
		c.api = c.api.WithEndpoints(e)
	}
	return c
}

// WithProfilesBaseURL override le host profils (tests httptest : rediriger vers
// srv.URL). Vide ignore. Chainable.
func (c *Client) WithProfilesBaseURL(profilesURL string) *Client {
	if profilesURL != "" {
		e := c.api.Endpoints()
		e.Profils = profilesURL
		c.api = c.api.WithEndpoints(e)
	}
	return c
}

// GetPlayerMatches recupere une page d'historique de matchs (halo5.Client.GetPlayerMatches).
func (c *Client) GetPlayerMatches(ctx context.Context, gamertag string, start, count int) (*halo5.H5MatchesResponse, error) {
	return c.api.GetPlayerMatches(ctx, c.auth, gamertag, start, count)
}

// GetServiceRecords recupere le service record agrege d'un joueur pour un mode
// (halo5.Client.GetServiceRecords).
func (c *Client) GetServiceRecords(ctx context.Context, gamertag, recordType string) (*halo5.H5ServiceRecordResponse, error) {
	return c.api.GetServiceRecords(ctx, c.auth, gamertag, recordType)
}

// GetMatchCarnage recupere le carnage report type d'un match (halo5.Client.GetMatchCarnage).
func (c *Client) GetMatchCarnage(ctx context.Context, matchID, mode string) (*halo5.H5CarnageResponse, error) {
	return c.api.GetMatchCarnage(ctx, c.auth, matchID, mode)
}

// GetMatchEvents recupere la timeline d'events native d'un match (halo5.Client.GetMatchEvents).
func (c *Client) GetMatchEvents(ctx context.Context, matchID string) (*halo5.H5MatchEventsResponse, error) {
	return c.api.GetMatchEvents(ctx, c.auth, matchID)
}

// GetAppearance recupere l'appearance identitaire d'un joueur (halo5.Client.GetAppearance).
func (c *Client) GetAppearance(ctx context.Context, gamertag string) (*halo5.H5Appearance, error) {
	return c.api.GetAppearance(ctx, c.auth, gamertag)
}

// GetSpartanRenderPNG recupere le rendu PNG du Spartan d'un joueur
// (halo5.Client.GetSpartanRenderPNG).
func (c *Client) GetSpartanRenderPNG(ctx context.Context, gamertag string) ([]byte, string, error) {
	return c.api.GetSpartanRenderPNG(ctx, c.auth, gamertag)
}

// GetEmblemPNG recupere le rendu PNG de l'emblème d'un joueur (halo5.Client.GetEmblemPNG).
func (c *Client) GetEmblemPNG(ctx context.Context, gamertag string) ([]byte, string, error) {
	return c.api.GetEmblemPNG(ctx, c.auth, gamertag)
}
