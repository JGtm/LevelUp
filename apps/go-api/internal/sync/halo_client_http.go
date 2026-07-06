// Package sync - halo_client_http.go : helpers HTTP (downloadBlob, doGet,
// rate limiting, backoff). Decoupe de halo_client.go (god-file split,
// refactor 2026-05-27).
package sync

import (
	"bytes"
	"compress/zlib"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

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
			// Échec réseau transitoire : retry silencieux (DEBUG). L'abandon après
			// maxRetries est loggé une seule fois en ERROR plus bas.
			slog.DebugContext(ctx, "halo_api: GET échec réseau (retry)",
				"url", rawURL, "attempt", attempt+1, "err", err)
			c.backoff(ctx, attempt, 0)
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
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
			httpErr := &HTTPError{StatusCode: resp.StatusCode, URL: rawURL, Err: fmt.Errorf("HTTP %d", resp.StatusCode), RetryAfter: retryAfter}
			// 429 (rate limit) : NE PAS retenter en interne. Re-taper l'API sous 10s
			// alors qu'on est rate-limité ne fait qu'ajouter des 429 et noyer les logs.
			// On rend l'HTTPError immédiatement : le PooledHaloClient déclenche le
			// cooldown GLOBAL du pool (OnHTTPError, dedup) et le caller retentera au
			// cycle suivant, fenêtre Retry-After respectée. Cf. boot stampede 2026-06-27.
			if resp.StatusCode == http.StatusTooManyRequests {
				slog.DebugContext(ctx, "halo_api: GET 429 (rate limit) — abandon immédiat, cooldown pool",
					"url", rawURL, "retry_after_s", retryAfter.Seconds())
				return nil, httpErr
			}
			lastErr = httpErr
			// Autres erreurs (ex: 503) : retry borné, en respectant Retry-After si fourni.
			slog.DebugContext(ctx, "halo_api: GET HTTP error (retry)",
				"url", rawURL, "status", resp.StatusCode, "attempt", attempt+1)
			c.backoff(ctx, attempt, retryAfter)
			continue
		}
		if readErr != nil {
			lastErr = readErr
			c.backoff(ctx, attempt, 0)
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

// backoff attend avant de retenter. Délai de base exponentiel
// (retryBaseDelay·2^attempt) ; si le serveur a fourni un Retry-After (retryAfter>0),
// on respecte AU MOINS cette fenêtre. Borné à backoffCeiling pour ne pas bloquer une
// requête de fond : les fenêtres plus longues sont gérées par le cooldown global du pool.
func (c *HaloAPIClient) backoff(ctx context.Context, attempt int, retryAfter time.Duration) {
	delay := retryBaseDelay * (1 << attempt)
	if retryAfter > delay {
		delay = retryAfter
	}
	if delay > backoffCeiling {
		delay = backoffCeiling
	}
	select {
	case <-ctx.Done():
	case <-time.After(delay):
	}
}

// SpartanCustomizationData : alias vers domain.SpartanCustomizationData (promu K1k).
// Identité visuelle Spartan (ServiceTag + images banner/emblem/backdrop) résolue depuis
// /customization?view=public. Le type canonique vit dans domain (découplage service→sync).
type SpartanCustomizationData = domain.SpartanCustomizationData

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

// maxRetryAfter borne la valeur du header Retry-After pour éviter qu'une valeur
// API aberrante ne fige le pool indéfiniment.
const maxRetryAfter = 5 * time.Minute

// parseRetryAfter parse le header Retry-After (RFC 7231) : soit un entier de
// secondes (delta-seconds), soit une date HTTP. Retourne une durée bornée dans
// [0, maxRetryAfter]. 0 si absent ou non parsable.
func parseRetryAfter(h string, now time.Time) time.Duration {
	if h == "" {
		return 0
	}
	if secs, err := strconv.Atoi(strings.TrimSpace(h)); err == nil {
		if secs <= 0 {
			return 0
		}
		if d := time.Duration(secs) * time.Second; d <= maxRetryAfter {
			return d
		}
		return maxRetryAfter
	}
	if t, err := http.ParseTime(h); err == nil {
		d := t.Sub(now)
		if d <= 0 {
			return 0
		}
		if d > maxRetryAfter {
			return maxRetryAfter
		}
		return d
	}
	return 0
}
