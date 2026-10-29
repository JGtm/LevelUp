package infinite

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"levelup/go-api/internal/haloapi"
)

// En-têtes des services Halo.
const (
	enteteSpartan   = "x-343-authorization-spartan"
	enteteClearance = "343-clearance"
	enteteAccept    = "Accept"
	enteteLangue    = "Accept-Language"
	typeJSON        = "application/json"
)

// Limiter borne le débit des requêtes. `*rate.Limiter` (`golang.org/x/time/rate`) la satisfait ;
// le limiteur appartient à l'appelant (un limiteur partagé entre clients borne leur débit commun).
type Limiter interface {
	Wait(ctx context.Context) error
}

// Auth : les en-têtes d'authentification d'une requête de service, passés à CHAQUE appel. Spartan
// vide : aucun en-tête d'authentification (page ou CDN publics) ; Clearance vide : pas d'en-tête
// `343-clearance`.
type Auth struct {
	Spartan   string
	Clearance string
}

// String masque les jetons (un `%v` ne les révèle jamais).
func (a Auth) String() string {
	return "Auth{Spartan:" + masque(a.Spartan) + " Clearance:" + masque(a.Clearance) + "}"
}

// LogValue masque les jetons dans un journal `slog`.
func (a Auth) LogValue() slog.Value {
	return slog.GroupValue(slog.String("spartan", masque(a.Spartan)), slog.String("clearance", masque(a.Clearance)))
}

func masque(s string) string {
	if s == "" {
		return ""
	}
	return "***"
}

// Crochets : ce que la brique signale sans le compter elle-même (les compteurs expvar et la
// limitation des journaux appartiennent à l'appelant). Un crochet nil est ignoré.
type Crochets struct {
	// BlobRepriseReussie : un blob obtenu grâce à une nouvelle tentative.
	BlobRepriseReussie func()
	// BlobEpuise : abandon d'un blob après épuisement des essais.
	BlobEpuise func()
	// BlobIncomplet : un blob reçu d'une autre taille que celle annoncée, refusé.
	BlobIncomplet func()
	// JournalBlobReseau : journalise l'abandon d'un blob sur échec RÉSEAU (l'appelant y limite le
	// débit des journaux en cas de panne). nil : un `WARN` à chaque abandon.
	JournalBlobReseau func(ctx context.Context, url string, essais int, err error)
}

// Client : le client des services Halo Infinite. Valeur immuable ; les `With*` rendent une copie,
// sûre en concurrence tant que le [haloapi.Doer] et le [Limiter] reçus le sont.
type Client struct {
	doer      haloapi.Doer
	endpoints Endpoints
	retry     Retry
	limiter   Limiter
	crochets  Crochets
}

// New construit un client sur le `doer` reçu (obligatoire : la brique n'en construit aucun), avec
// la politique de reprise `retry` et les adresses par défaut ([DefaultEndpoints]).
func New(doer haloapi.Doer, retry Retry) *Client {
	return &Client{doer: doer, endpoints: DefaultEndpoints(), retry: retry}
}

// WithEndpoints rend une copie du client sur les adresses `e`.
func (c *Client) WithEndpoints(e Endpoints) *Client {
	clone := *c
	clone.endpoints = e
	return &clone
}

// WithLimiter rend une copie du client sur le limiteur `l` (nil : aucun limiteur).
func (c *Client) WithLimiter(l Limiter) *Client {
	clone := *c
	clone.limiter = l
	return &clone
}

// WithRetry rend une copie du client sur la politique de reprise `r`.
func (c *Client) WithRetry(r Retry) *Client {
	clone := *c
	clone.retry = r
	return &clone
}

// WithCrochets rend une copie du client sur les crochets `k`.
func (c *Client) WithCrochets(k Crochets) *Client {
	clone := *c
	clone.crochets = k
	return &clone
}

// Endpoints rend les adresses du client.
func (c *Client) Endpoints() Endpoints { return c.endpoints }

// requete : un GET de service.
type requete struct {
	url    string
	auth   Auth
	langue string // valeur d'Accept-Language ; "" = pas d'en-tête
}

// essai : l'issue d'UNE requête.
type essai struct {
	body       []byte
	err        error
	retryAfter time.Duration
	// definitif : ne pas retenter (requête impossible à construire, erreur [haloapi.ErrDefinitif],
	// 401/403/404/410, statut que la politique ne retente pas).
	definitif bool
}

// get exécute un GET de service sous la politique de reprise du client (voir [Retry]).
func (c *Client) get(ctx context.Context, rq requete) ([]byte, error) {
	r := c.retry
	if r.LimiteurUneFois {
		if err := c.attendreLimiteur(ctx); err != nil {
			return nil, err
		}
	}
	start := time.Now()
	var derniere error
	for n := 0; n < r.Essais; n++ {
		if !r.LimiteurUneFois {
			if err := c.attendreLimiteur(ctx); err != nil {
				return nil, err
			}
		}
		e := c.essayer(ctx, rq)
		if e.err == nil {
			slog.DebugContext(ctx, "halo_api: GET succès", "url", rq.url, "bytes", len(e.body),
				"duration_ms", time.Since(start).Milliseconds(), "attempt", n+1)
			return e.body, nil
		}
		if e.definitif {
			return nil, e.err
		}
		derniere = e.err
		slog.DebugContext(ctx, "halo_api: GET échec (retry)", "url", rq.url, "attempt", n+1, "err", e.err)
		if !r.attendre(ctx, n, e.retryAfter) {
			return nil, ctx.Err()
		}
	}
	slog.ErrorContext(ctx, "halo_api: GET échec définitif", "url", rq.url, "attempts", r.Essais,
		"duration_ms", time.Since(start).Milliseconds(), "err", derniere)
	return nil, fmt.Errorf("doGet %s: %d tentatives échouées — %w", rq.url, r.Essais, derniere)
}

// attendreLimiteur attend un jeton du limiteur ; une erreur est rendue, sauf si la politique
// l'ignore ([Retry.IgnoreErreurLimiteur]).
func (c *Client) attendreLimiteur(ctx context.Context) error {
	if c.limiter == nil {
		return nil
	}
	if err := c.limiter.Wait(ctx); err != nil && !c.retry.IgnoreErreurLimiteur {
		return fmt.Errorf("rate limiter: %w", err)
	}
	return nil
}

// essayer envoie UNE requête et classe son issue.
func (c *Client) essayer(ctx context.Context, rq requete) essai {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rq.url, nil)
	if err != nil {
		return essai{err: fmt.Errorf("doGet new request: %w", err), definitif: true}
	}
	poserEntetes(req, rq)
	resp, err := c.doer.Do(req)
	if err != nil {
		return essai{err: err, definitif: estDefinitif(err)}
	}
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close() // corps déjà lu ; une erreur de fermeture ne change pas l'issue
	return c.classer(ctx, rq.url, resp, body, readErr)
}

// classer rend l'issue d'une réponse reçue. Le statut est lu AVANT l'erreur de lecture du corps.
func (c *Client) classer(ctx context.Context, url string, resp *http.Response, body []byte, readErr error) essai {
	code := resp.StatusCode
	switch {
	case estStatutAuth(code):
		slog.WarnContext(ctx, "halo_api: GET refusé auth (pas de retry)", "url", url, "status", code)
		return essai{err: &HTTPError{StatusCode: code, URL: url, Err: errJetonsRefuses, Body: body}, definitif: true}
	case estStatutAbsent(code):
		slog.DebugContext(ctx, "halo_api: GET ressource absente", "url", url, "status", code)
		return essai{err: &HTTPError{StatusCode: code, URL: url, Err: errRessourceAbsente, Body: body}, definitif: true}
	case code != http.StatusOK:
		ra := ParseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
		he := &HTTPError{StatusCode: code, URL: url, Err: fmt.Errorf("HTTP %d", code), RetryAfter: ra, Body: body}
		if !c.retry.retente(code) {
			slog.DebugContext(ctx, "halo_api: GET HTTP error (pas de retry)", "url", url, "status", code,
				"retry_after_s", ra.Seconds())
			return essai{err: he, definitif: true}
		}
		return essai{err: he, retryAfter: ra}
	case readErr != nil:
		return essai{err: readErr}
	}
	return essai{body: body}
}

// poserEntetes pose Accept, Accept-Language (si donné) et les jetons (si donnés).
func poserEntetes(req *http.Request, rq requete) {
	req.Header.Set(enteteAccept, typeJSON)
	if rq.langue != "" {
		req.Header.Set(enteteLangue, rq.langue)
	}
	poserAuth(req, rq.auth)
}

// poserAuth pose les en-têtes de jetons non vides.
func poserAuth(req *http.Request, a Auth) {
	if a.Spartan != "" {
		req.Header.Set(enteteSpartan, a.Spartan)
	}
	if a.Clearance != "" {
		req.Header.Set(enteteClearance, a.Clearance)
	}
}

// estDefinitif dit si une erreur du Doer arrête toute reprise ([haloapi.ErrDefinitif]).
func estDefinitif(err error) bool { return errors.Is(err, haloapi.ErrDefinitif) }
