// Package halo_5 — adapter multi-titre EXPERIMENTAL pour Halo 5: Guardians.
//
// client.go : client HTTP pour les endpoints internes Halo 5 (autorites cryptum).
// Recette CONFIRMEE par la sonde live 2026-06-19 (cmd/probe-h5, JGtm, 7/7 HTTP 200
// avec le SpartanToken v4 du pool Infinite) :
//
//   - Host stats/CSR/commendations : https://spartanstats.svc.halowaypoint.com
//   - Host profils/appearance      : https://haloplayer.svc.halowaypoint.com
//   - Segment de jeu               : "h5" (ex. /h5/players/{gamertag}/matches)
//   - Header auth                  : X-343-Authorization-Spartan: {SpartanToken v4}
//   - User-Agent                   : cpprestsdk/2.4.0 (requis ; gate cote 343)
//   - Query auth                   : ?auth=st sur chaque requete *.svc.halowaypoint.com
//   - PAS de 343-clearance         : Halo 5 ne l'utilise pas (ClearanceAware:false)
//   - Identite                     : GAMERTAG BRUT dans le path (divergence vs HI
//     qui exige xuid(NNN))
//
// Stateless : instancier par session avec le SpartanToken du joueur. Thread-safe
// (net/http.Client + rate.Limiter concurrency-safe). gzip transparent : on NE fixe
// PAS Accept-Encoding (sinon net/http desactive la decompression).
package halo_5

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/platform/netguard"

	"golang.org/x/time/rate"
)

// TitleSlug est l'identite canonique du titre Halo 5 (constante de package
// title-specific, pas une comparaison de slug — archlint no_slug_comparison OK).
const TitleSlug = "halo_5"

const (
	// Host interne confirme (cf. config/titles/halo_5/constants.toml [endpoints]).
	h5StatsHost = "https://spartanstats.svc.halowaypoint.com"

	// Host profils/appearance/emblem/spartan (Phase 2 profil). Confirme par la sonde
	// live 2026-06-25 (JGtm) : /h5/profiles/{gt}/appearance → 200 JSON,
	// /h5/profiles/{gt}/{spartan,emblem} → 302 vers image.halocdn.com (PNG signe).
	h5ProfilesHost = "https://haloplayer.svc.halowaypoint.com"

	h5UserAgent      = "cpprestsdk/2.4.0"
	h5MaxRetries     = 4
	h5RetryBaseDelay = 800 * time.Millisecond
	h5MaxBackoff     = 10 * time.Second
)

// HTTPError encapsule une erreur HTTP avec StatusCode expose (symetrique de
// sync.HTTPError, redeclare local pour eviter un couplage halo_5 -> sync).
type HTTPError struct {
	StatusCode int
	URL        string
	Err        error
}

func (e *HTTPError) Error() string { return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.URL) }
func (e *HTTPError) Unwrap() error { return e.Err }

// Client est le client HTTP stateless pour l'API interne Halo 5.
type Client struct {
	http            *http.Client
	spartanToken    string
	statsBaseURL    string // override d'instance (tests httptest) ; vide -> h5StatsHost
	profilesBaseURL string // override d'instance (tests httptest) ; vide -> h5ProfilesHost
	limiter         *rate.Limiter
}

// NewClient cree un client authentifie avec le SpartanToken v4 du joueur.
// requestsPerSecond <= 0 -> defaut 10. PAS de clearanceToken (Halo 5 ne l'utilise pas).
func NewClient(spartanToken string, requestsPerSecond int) *Client {
	if requestsPerSecond <= 0 {
		requestsPerSecond = 10
	}
	return &Client{
		http:         &http.Client{Timeout: 20 * time.Second},
		spartanToken: spartanToken,
		limiter:      rate.NewLimiter(rate.Limit(requestsPerSecond), 1),
	}
}

// WithHTTPClient remplace le *http.Client interne (tests httptest via RoundTripper).
// nil restaure le defaut. Chainable.
func (c *Client) WithHTTPClient(h *http.Client) *Client {
	if h != nil {
		c.http = h
	}
	return c
}

// WithLimiter partage un rate.Limiter externe (pool commun). nil ignore. Chainable.
func (c *Client) WithLimiter(l *rate.Limiter) *Client {
	if l != nil {
		c.limiter = l
	}
	return c
}

// WithStatsBaseURL override le host stats (tests httptest : rediriger vers srv.URL).
// Vide ignore. Chainable.
func (c *Client) WithStatsBaseURL(statsURL string) *Client {
	if statsURL != "" {
		c.statsBaseURL = statsURL
	}
	return c
}

func (c *Client) statsHost() string {
	if c.statsBaseURL != "" {
		return c.statsBaseURL
	}
	return h5StatsHost
}

// WithProfilesBaseURL override le host profils (tests httptest : rediriger vers
// srv.URL). Vide ignore. Chainable.
func (c *Client) WithProfilesBaseURL(profilesURL string) *Client {
	if profilesURL != "" {
		c.profilesBaseURL = profilesURL
	}
	return c
}

func (c *Client) profilesHost() string {
	if c.profilesBaseURL != "" {
		return c.profilesBaseURL
	}
	return h5ProfilesHost
}

// GetPlayerMatches recupere une page d'historique de matchs (tous modes confondus).
// GAMERTAG BRUT (pas xuid(NNN)). Path confirme sonde : /h5/players/{gt}/matches.
func (c *Client) GetPlayerMatches(ctx context.Context, gamertag string, start, count int) (*H5MatchesResponse, error) {
	if strings.TrimSpace(gamertag) == "" {
		return nil, errors.New("GetPlayerMatches: gamertag vide")
	}
	// include-times=true : SANS ce param, l'API met le composant heure de
	// MatchCompletedDate à 00:00:00 (fidelity 1, jour seulement) → StartedAtUTC
	// dérivé (fin − durée) tombe à minuit−durée (faux). AVEC, MatchCompletedDate
	// est horodaté à la ms (fidelity 2) → heure de début exacte. Requis pour
	// l'association media par timestamp + la qualité temporelle des matchs.
	endpoint := fmt.Sprintf("%s/h5/players/%s/matches?start=%d&count=%d&include-times=true",
		c.statsHost(), url.PathEscape(gamertag), start, count)
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("GetPlayerMatches(%s): %w", gamertag, err)
	}
	var resp H5MatchesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("GetPlayerMatches decode: %w", err)
	}
	return &resp, nil
}

// GetServiceRecords recupere le service record agrege d'un joueur pour un mode.
// recordType : "arena" | "warzone" | "campaign" | "custom". Path confirme sonde :
// /h5/servicerecords/{recordType}?players={gt}.
func (c *Client) GetServiceRecords(ctx context.Context, gamertag, recordType string) (*H5ServiceRecordResponse, error) {
	if strings.TrimSpace(gamertag) == "" {
		return nil, errors.New("GetServiceRecords: gamertag vide")
	}
	if strings.TrimSpace(recordType) == "" {
		return nil, errors.New("GetServiceRecords: recordType vide")
	}
	endpoint := fmt.Sprintf("%s/h5/servicerecords/%s?players=%s",
		c.statsHost(), url.PathEscape(recordType), url.QueryEscape(gamertag))
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("GetServiceRecords(%s,%s): %w", gamertag, recordType, err)
	}
	var resp H5ServiceRecordResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("GetServiceRecords decode: %w", err)
	}
	return &resp, nil
}

// GetMatchDetail recupere le carnage report complet d'un match (Phase 2 : scoreboard
// etendu + CSR pre/post). mode = "arena"|"warzone"|... (Links.StatsMatchDetails).
// Renvoie le JSON brut (le mapping fin carnage->canonical est Phase 2).
func (c *Client) GetMatchDetail(ctx context.Context, matchID, mode string) (map[string]any, error) {
	if strings.TrimSpace(matchID) == "" || strings.TrimSpace(mode) == "" {
		return nil, errors.New("GetMatchDetail: matchID/mode vide")
	}
	endpoint := fmt.Sprintf("%s/h5/%s/matches/%s",
		c.statsHost(), url.PathEscape(mode), url.PathEscape(matchID))
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("GetMatchDetail(%s): %w", matchID, err)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("GetMatchDetail decode: %w", err)
	}
	return out, nil
}

// GetMatchCarnage recupere le carnage report TYPE d'un match (roster complet +
// scoreboard etendu). mode = "arena"|"warzone"|... (cf. Links.StatsMatchDetails).
// Variante typee de GetMatchDetail, consommee par l'ingestion participants.
func (c *Client) GetMatchCarnage(ctx context.Context, matchID, mode string) (*H5CarnageResponse, error) {
	if strings.TrimSpace(matchID) == "" || strings.TrimSpace(mode) == "" {
		return nil, errors.New("GetMatchCarnage: matchID/mode vide")
	}
	endpoint := fmt.Sprintf("%s/h5/%s/matches/%s",
		c.statsHost(), url.PathEscape(mode), url.PathEscape(matchID))
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("GetMatchCarnage(%s): %w", matchID, err)
	}
	var resp H5CarnageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("GetMatchCarnage decode: %w", err)
	}
	return &resp, nil
}

// GetMatchEvents recupere la timeline d'events NATIVE d'un match (kill-feed +
// arme-par-kill + medailles + positions monde, horodates). Path CONFIRME sonde :
// /h5/matches/{id}/events — ⚠️ SANS segment de mode (le /h5/{mode}/matches/{id}/
// events renvoie 404). Cf. HANDOFF_HALO5_EXPERIMENTAL §0-quater.
func (c *Client) GetMatchEvents(ctx context.Context, matchID string) (*h5MatchEventsResponse, error) {
	if strings.TrimSpace(matchID) == "" {
		return nil, errors.New("GetMatchEvents: matchID vide")
	}
	endpoint := fmt.Sprintf("%s/h5/matches/%s/events", c.statsHost(), url.PathEscape(matchID))
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("GetMatchEvents(%s): %w", matchID, err)
	}
	var resp h5MatchEventsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("GetMatchEvents decode: %w", err)
	}
	return &resp, nil
}

// GetAppearance recupere l'appearance identitaire d'un joueur (service tag + emblem
// IDs/couleurs + customization armure). Host PROFILS (haloplayer). GAMERTAG BRUT.
// Path confirme sonde : /h5/profiles/{gt}/appearance → 200 JSON. On ne projette que
// les champs identitaires (cf. H5Appearance).
func (c *Client) GetAppearance(ctx context.Context, gamertag string) (*H5Appearance, error) {
	if strings.TrimSpace(gamertag) == "" {
		return nil, errors.New("GetAppearance: gamertag vide")
	}
	endpoint := fmt.Sprintf("%s/h5/profiles/%s/appearance", c.profilesHost(), url.PathEscape(gamertag))
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("GetAppearance(%s): %w", gamertag, err)
	}
	var resp H5Appearance
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("GetAppearance decode: %w", err)
	}
	return &resp, nil
}

// GetSpartanRenderPNG recupere le RENDU PNG du Spartan (corps/armure) d'un joueur.
// L'endpoint /h5/profiles/{gt}/spartan renvoie un 302 vers image.halocdn.com (URL
// signee, non reproductible cote client). doGetBinary suit le redirect et telecharge
// les octets PNG. Renvoie (bytes, contentType, err). Host PROFILS (haloplayer).
func (c *Client) GetSpartanRenderPNG(ctx context.Context, gamertag string) ([]byte, string, error) {
	if strings.TrimSpace(gamertag) == "" {
		return nil, "", errors.New("GetSpartanRenderPNG: gamertag vide")
	}
	endpoint := fmt.Sprintf("%s/h5/profiles/%s/spartan", c.profilesHost(), url.PathEscape(gamertag))
	b, ct, err := c.doGetBinary(ctx, endpoint)
	if err != nil {
		return nil, "", fmt.Errorf("GetSpartanRenderPNG(%s): %w", gamertag, err)
	}
	return b, ct, nil
}

// GetEmblemPNG recupere le rendu PNG de l'emblème d'un joueur. L'endpoint
// /h5/profiles/{gt}/emblem renvoie un 302 vers image.halocdn.com (path
// emblems/{EmblemId}_{ColorPrimary}_{ColorSecondary}_{ColorTertiary}, hash signe).
// doGetBinary suit le redirect et telecharge le PNG. Renvoie (bytes, contentType,
// err). Host PROFILS (haloplayer).
func (c *Client) GetEmblemPNG(ctx context.Context, gamertag string) ([]byte, string, error) {
	if strings.TrimSpace(gamertag) == "" {
		return nil, "", errors.New("GetEmblemPNG: gamertag vide")
	}
	endpoint := fmt.Sprintf("%s/h5/profiles/%s/emblem", c.profilesHost(), url.PathEscape(gamertag))
	b, ct, err := c.doGetBinary(ctx, endpoint)
	if err != nil {
		return nil, "", fmt.Errorf("GetEmblemPNG(%s): %w", gamertag, err)
	}
	return b, ct, nil
}

// doGet execute un GET authentifie facon Halo 5 : header Spartan v4 + UA cpprestsdk,
// query ?auth=st, gzip transparent, retry/backoff exponentiel borne. 401/403/404/410
// sont terminaux (pas de retry). Renvoie le corps JSON brut.
func (c *Client) doGet(ctx context.Context, rawURL string) ([]byte, error) {
	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	rawURL += sep + "auth=st"

	var lastErr error
	for attempt := 0; attempt < h5MaxRetries; attempt++ {
		if c.limiter != nil {
			if err := c.limiter.Wait(ctx); err != nil {
				return nil, err
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("doGet new request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-343-Authorization-Spartan", c.spartanToken)
		req.Header.Set("User-Agent", h5UserAgent)
		// PAS de 343-clearance. Accept-Encoding non fixe -> gzip transparent.

		// Mode démo : aucune sortie tierce (cf. internal/platform/netguard).
		if gErr := netguard.Check(ctx, "halo5_api.get"); gErr != nil {
			return nil, gErr
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			c.waitRetry(ctx, attempt, 0)
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		retryAfter := parseRetryAfterSeconds(resp.Header.Get("Retry-After"))
		resp.Body.Close()

		switch {
		case resp.StatusCode == http.StatusOK:
			if readErr != nil {
				lastErr = readErr
				c.waitRetry(ctx, attempt, 0)
				continue
			}
			return body, nil
		case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
			return nil, &HTTPError{StatusCode: resp.StatusCode, URL: rawURL, Err: errors.New("token invalide/expire")}
		case resp.StatusCode == http.StatusNotFound, resp.StatusCode == http.StatusGone:
			return nil, &HTTPError{StatusCode: resp.StatusCode, URL: rawURL, Err: errors.New("ressource absente")}
		default:
			// 429/503 + Retry-After : respecter le delai demande par 343 (plancher =
			// backoff exponentiel) pour ne pas re-cogner trop tot et se faire re-throttler.
			lastErr = &HTTPError{StatusCode: resp.StatusCode, URL: rawURL, Err: fmt.Errorf("HTTP %d", resp.StatusCode)}
			c.waitRetry(ctx, attempt, retryAfter)
		}
	}
	return nil, fmt.Errorf("doGet %s: %d tentatives echouees: %w", rawURL, h5MaxRetries, lastErr)
}

// doGetBinary execute un GET authentifie facon Halo 5 et SUIT les redirects (302)
// pour telecharger un corps binaire (image PNG). Mirroir de doGet pour l'auth/retry
// (header Spartan + UA + ?auth=st), mais ne suppose pas de JSON et accepte n'importe
// quel content-type d'image en 200. 401/403/404/410 terminaux. Renvoie (bytes,
// contentType, err). Le redirect est suivi par le *http.Client (politique par defaut).
func (c *Client) doGetBinary(ctx context.Context, rawURL string) ([]byte, string, error) {
	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	rawURL += sep + "auth=st"

	var lastErr error
	for attempt := 0; attempt < h5MaxRetries; attempt++ {
		if c.limiter != nil {
			if err := c.limiter.Wait(ctx); err != nil {
				return nil, "", err
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, "", fmt.Errorf("doGetBinary new request: %w", err)
		}
		req.Header.Set("Accept", "image/png,image/*")
		req.Header.Set("X-343-Authorization-Spartan", c.spartanToken)
		req.Header.Set("User-Agent", h5UserAgent)

		// Mode démo : aucune sortie tierce (cf. internal/platform/netguard).
		if gErr := netguard.Check(ctx, "halo5_api.get_binary"); gErr != nil {
			return nil, "", gErr
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			c.waitRetry(ctx, attempt, 0)
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		retryAfter := parseRetryAfterSeconds(resp.Header.Get("Retry-After"))
		contentType := resp.Header.Get("Content-Type")
		resp.Body.Close()

		switch {
		case resp.StatusCode == http.StatusOK:
			if readErr != nil {
				lastErr = readErr
				c.waitRetry(ctx, attempt, 0)
				continue
			}
			if len(body) == 0 {
				return nil, "", &HTTPError{StatusCode: resp.StatusCode, URL: rawURL, Err: errors.New("corps vide")}
			}
			return body, contentType, nil
		case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
			return nil, "", &HTTPError{StatusCode: resp.StatusCode, URL: rawURL, Err: errors.New("token invalide/expire")}
		case resp.StatusCode == http.StatusNotFound, resp.StatusCode == http.StatusGone:
			return nil, "", &HTTPError{StatusCode: resp.StatusCode, URL: rawURL, Err: errors.New("ressource absente")}
		default:
			lastErr = &HTTPError{StatusCode: resp.StatusCode, URL: rawURL, Err: fmt.Errorf("HTTP %d", resp.StatusCode)}
			c.waitRetry(ctx, attempt, retryAfter)
		}
	}
	return nil, "", fmt.Errorf("doGetBinary %s: %d tentatives echouees: %w", rawURL, h5MaxRetries, lastErr)
}

// waitRetry attend max(backoff exponentiel, retryAfter), borne a h5MaxBackoff,
// interruptible par ctx. retryAfter=0 -> backoff seul.
func (c *Client) waitRetry(ctx context.Context, attempt int, retryAfter time.Duration) {
	delay := h5RetryBaseDelay * time.Duration(1<<attempt)
	if retryAfter > delay {
		delay = retryAfter
	}
	if delay > h5MaxBackoff {
		delay = h5MaxBackoff
	}
	select {
	case <-ctx.Done():
	case <-time.After(delay):
	}
}

// parseRetryAfterSeconds lit un header Retry-After en SECONDES (forme la plus
// courante des services 343). Forme date HTTP non geree (rare ici) -> 0. Borne a
// h5MaxBackoff. Valeur invalide/negative -> 0.
func parseRetryAfterSeconds(h string) time.Duration {
	h = strings.TrimSpace(h)
	if h == "" {
		return 0
	}
	secs, err := strconv.Atoi(h)
	if err != nil || secs <= 0 {
		return 0
	}
	d := time.Duration(secs) * time.Second
	if d > h5MaxBackoff {
		d = h5MaxBackoff
	}
	return d
}
