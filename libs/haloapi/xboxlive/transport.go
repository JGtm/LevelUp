package xboxlive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	haloapi "github.com/JGtm/halo-api"
)

// En-têtes et valeurs du protocole des succès.
const (
	enteteAutorisation = "Authorization"
	enteteContrat      = "x-xbl-contract-version"
	versionContrat     = "2"
	enteteLangue       = "Accept-Language"
	enteteAccept       = "Accept"
	typeJSON           = "application/json"
)

// Auth : l'en-tête d'autorisation d'une requête, passé à CHAQUE appel. `Autorisation` est la valeur
// COMPLÈTE de l'en-tête `Authorization` (`XBL3.0 x=<hash>;<jeton>`), posée telle quelle.
type Auth struct {
	Autorisation string
}

// String masque le jeton (un `%v` ne le révèle jamais).
func (a Auth) String() string { return "Auth{Autorisation:" + haloapi.Masque(a.Autorisation) + "}" }

// LogValue masque le jeton dans un journal `slog`.
func (a Auth) LogValue() slog.Value {
	return slog.GroupValue(slog.String("autorisation", haloapi.Masque(a.Autorisation)))
}

// Page : la fenêtre demandée d'une liste de succès.
type Page struct {
	// TitleID restreint la liste à un titre Xbox (identifiant numérique en chaîne) ; vide : aucun filtre.
	TitleID string
	// SkipItems : nombre d'éléments déjà reçus, ignorés par l'API ; 0 : première page.
	SkipItems int
}

// Client : le client des succès Xbox Live. Valeur immuable ; `WithEndpoints` rend une copie, sûre en
// concurrence tant que le [haloapi.Doer] reçu l'est.
type Client struct {
	doer      haloapi.Doer
	endpoints Endpoints
}

// New construit un client sur le `Doer` reçu (obligatoire : la brique n'en construit aucun), à
// l'adresse par défaut ([EndpointsParDefaut]).
func New(doer haloapi.Doer) *Client {
	return &Client{doer: doer, endpoints: EndpointsParDefaut()}
}

// WithEndpoints rend une copie du client sur l'adresse `e`.
func (c *Client) WithEndpoints(e Endpoints) *Client {
	clone := *c
	clone.endpoints = e
	return &clone
}

// GetSuccesPage demande UNE page des succès du joueur `xuid`, libellés dans la langue `lang` (étiquette
// BCP-47, posée même vide). La garde du Doer ([haloapi.Garde]) est consultée d'abord : une sortie
// refusée rend son erreur telle quelle, sans requête. Un statut non-200 rend un [*HTTPError], un
// échec d'émission un [*ErreurTransport] ; une erreur de construction de la requête, de lecture du
// corps ou de décodage est rendue enveloppée (`parse URL`, `new request`, `read body`, `JSON decode`).
func (c *Client) GetSuccesPage(ctx context.Context, auth Auth, xuid, lang string, page Page) (*SuccesReponse, error) {
	if g, ok := c.doer.(haloapi.Garde); ok {
		if err := g.Verifie(ctx); err != nil {
			return nil, err
		}
	}
	adresse, err := c.urlSucces(xuid, page)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, adresse, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set(enteteAutorisation, auth.Autorisation)
	req.Header.Set(enteteContrat, versionContrat)
	req.Header.Set(enteteLangue, lang)
	req.Header.Set(enteteAccept, typeJSON)
	corps, err := c.lire(req)
	if err != nil {
		return nil, err
	}
	var rep SuccesReponse
	if err := json.Unmarshal(corps, &rep); err != nil {
		return nil, fmt.Errorf("JSON decode: %w", err)
	}
	return &rep, nil
}

// urlSucces rend l'URL de la page : le `xuid` est posé tel quel dans le chemin, puis `titleId` et
// `skipItems` (posés s'ils sont non nuls) dans la requête encodée par `url.Values.Encode`.
func (c *Client) urlSucces(xuid string, page Page) (string, error) {
	u, err := url.Parse(fmt.Sprintf("%s/users/xuid(%s)/achievements", c.endpoints.hoteSucces(), xuid))
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}
	q := u.Query()
	if page.TitleID != "" {
		q.Set("titleId", page.TitleID)
	}
	if page.SkipItems > 0 {
		q.Set("skipItems", strconv.Itoa(page.SkipItems))
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// lire émet la requête et rend le corps d'une réponse 200. Le corps est lu EN ENTIER avant le statut :
// un statut non-200 rend un [*HTTPError] qui le porte.
func (c *Client) lire(req *http.Request) ([]byte, error) {
	resp, err := c.doer.Do(req)
	if err != nil {
		if errors.Is(err, haloapi.ErrDefinitif) {
			return nil, err
		}
		return nil, &ErreurTransport{URL: req.URL.String(), Err: err}
	}
	defer func() { _ = resp.Body.Close() }() // la fermeture ne change pas l'issue
	corps, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{StatusCode: resp.StatusCode, URL: req.URL.String(), Body: corps}
	}
	return corps, nil
}
