// Package presence — rest_client.go : client REST polling pour la présence Xbox.
//
// Pourquoi REST plutôt que RTA WebSocket :
//   - Le RTA `/titles/<TID>` ne push pas après le snapshot (confirmé 22 min
//     d'observation sans push, extinction du jeu non détectée).
//   - Le RTA `/richpresence` retourne status=3 avec notre RelyingParty XSTS
//     `http://xboxlive.com`. Microsoft/LucienHH s'authentifient comme
//     `Titles.XboxAppIOS` (borderline ToS).
//
// REST polling :
//   - Endpoint standard `/users/xuid(N)/presence` accepte notre XSTS standard.
//   - Latence ~30s en polling (acceptable car la sync se déclenche à fin de
//     match, et l'API Halo GetMatchHistory met 30-60s à exposer le match
//     terminé — le bottleneck end-to-end n'est pas la présence).
//   - Coût : 1 req / 30s / joueur tracké = négligeable.
//
// Le payload retourné est au même format `state + devices[].titles[]` que
// celui poussé par le RTA, donc on réutilise `ParsePresencePayload`.
package presence

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	// restPresenceURLFmt est l'URL REST de la présence courante d'un user.
	// Format confirmé par OpenXbox/xbox-webapi-python :
	// GET /users/xuid({xuid})?level={user|device|title|all}
	//
	// `level=all` retourne state global + devices[].titles[] (le format dont
	// on a besoin pour détecter le titre actif via le parser existant
	// ParsePresencePayload, cf. event_parser.go).
	//
	// Cf. https://learn.microsoft.com/en-us/gaming/gdk/docs/reference/live/rest/uri/presence/atoc-reference-presence
	restPresenceURLFmt = "https://userpresence.xboxlive.com/users/xuid(%s)?level=all"

	// restPresenceHTTPTimeout est le délai max d'une requête HTTP individuelle.
	// 10s couvre largement la latence Xbox typique (~200ms) avec marge réseau.
	restPresenceHTTPTimeout = 10 * time.Second
)

// PresenceClient interroge l'API REST Xbox Live pour récupérer la présence
// d'un joueur. Thread-safe : peut être partagé entre plusieurs goroutines.
type PresenceClient struct {
	authHeader atomic.Pointer[string] // mis à jour à chaud sur refresh XSTS
	httpClient *http.Client
}

// NewPresenceClient crée un client REST avec un header d'auth initial.
// authHeader doit être au format `XBL3.0 x=<userhash>;<xsts_token>`.
func NewPresenceClient(authHeader string) *PresenceClient {
	c := &PresenceClient{
		httpClient: &http.Client{Timeout: restPresenceHTTPTimeout},
	}
	c.authHeader.Store(&authHeader)
	return c
}

// UpdateAuth remplace le header XBL3.0 (appelé après refresh XSTS).
// Atomic : aucune lock côté GetPresence.
func (c *PresenceClient) UpdateAuth(authHeader string) {
	c.authHeader.Store(&authHeader)
}

// AuthHeader retourne le header courant (utile pour debug + tests).
func (c *PresenceClient) AuthHeader() string {
	if p := c.authHeader.Load(); p != nil {
		return *p
	}
	return ""
}

// GetPresence interroge Xbox pour récupérer la présence courante d'un user.
// Retourne un PresenceEvent (même type que RTA) pour réutiliser la chaîne
// de handlers existante. xuid doit être numérique sans préfixe "xuid()".
//
// Erreurs typiques :
//   - 401 : XSTS expiré → l'appelant doit refresh + UpdateAuth + retry
//   - 429 : rate-limit Xbox → backoff exponentiel à prévoir
//   - 5xx : transient Xbox → retry
//   - timeout : réseau intermittent → retry
//
// Cette fonction n'implémente pas le retry ; c'est la responsabilité du
// REST poller (cf. rest_poller.go) qui sait quand re-tenter selon l'erreur.
func (c *PresenceClient) GetPresence(ctx context.Context, xuid string) (PresenceEvent, error) {
	if xuid == "" {
		return PresenceEvent{}, fmt.Errorf("rest presence: xuid vide")
	}

	url := fmt.Sprintf(restPresenceURLFmt, xuid)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return PresenceEvent{}, fmt.Errorf("rest presence: build req: %w", err)
	}
	req.Header.Set("Authorization", c.AuthHeader())
	// x-xbl-contract-version : version de l'API Xbox pour la présence.
	// 3 est la version stable documentée (richpresence, devices[]) — cf.
	// header officiel utilisé par xbox-live-api C++ (XblPresence).
	req.Header.Set("x-xbl-contract-version", "3")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return PresenceEvent{}, fmt.Errorf("rest presence: do req: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		slog.DebugContext(ctx, "rest_presence: réponse non-OK",
			"xuid", xuid,
			"status", resp.StatusCode,
			"body", string(body),
		)
		return PresenceEvent{}, &HTTPError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return PresenceEvent{}, fmt.Errorf("rest presence: read body: %w", err)
	}

	event, err := ParsePresencePayload(json.RawMessage(raw), xuid)
	if err != nil {
		return PresenceEvent{}, fmt.Errorf("rest presence: parse: %w", err)
	}
	return event, nil
}

// HTTPError encapsule une réponse Xbox non-2xx (401, 429, 5xx). Permet à
// l'appelant de discriminer entre une vraie erreur réseau et une erreur HTTP
// récupérable.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("rest presence: HTTP %d: %s", e.StatusCode, truncateBody(e.Body, 200))
}

// IsAuthExpired retourne true si l'erreur HTTP indique un XSTS expiré (401).
// L'appelant doit appeler son refresh XSTS + UpdateAuth + retry.
func (e *HTTPError) IsAuthExpired() bool {
	return e.StatusCode == http.StatusUnauthorized
}

// IsRateLimited retourne true si Xbox a rate-limité (429). L'appelant doit
// backoff (typiquement plusieurs minutes).
func (e *HTTPError) IsRateLimited() bool {
	return e.StatusCode == http.StatusTooManyRequests
}

// IsTransient retourne true pour les 5xx Xbox (panne serveur). L'appelant
// peut retenter immédiatement avec un backoff modéré.
func (e *HTTPError) IsTransient() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

func truncateBody(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
