// Package sync - halo_client_http.go : helpers HTTP (downloadBlob, doGet,
// rate limiting, backoff). Decoupe de halo_client.go (god-file split,
// refactor 2026-05-27).
package haloclient

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
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/netguard"
)

// Compteurs expvar du téléchargement de blobs (convention observability :
// "<categorie>_<sous_cle>" en snake_case).
const (
	// metricBlobRetrySuccess : blobs obtenus grâce à une retentative — sans elle,
	// c'était un film déclaré illisible.
	metricBlobRetrySuccess = "halo_api_blob_retry_success"
	// metricBlobRetryExhausted : abandons après maxRetries tentatives.
	metricBlobRetryExhausted = "halo_api_blob_retry_exhausted"
)

// BlobHTTPError est l'échec HTTP DÉFINITIF d'un téléchargement de blob sur le CDN
// des films Halo.
//
// Type DISTINCT de HTTPError, et surtout PAS un HTTPError enveloppé : le CDN des
// films est PUBLIC et non authentifié (URL sans query string, aucune en-tête
// d'auth envoyée). Un 401/403/503 venant de lui ne dit RIEN de la santé de nos
// tokens ni de l'API Halo — or notifyPoolOnError (internal/sync/pooled_client.go)
// marque un slot unhealthy sur un *HTTPError 401/403 et gèle TOUT le pool sur un
// 503. Si errors.As(err, &*HTTPError) matchait une erreur de blob, une panne
// d'edge CDN poisonnerait un token valide ou figerait le pool de l'API Halo.
type BlobHTTPError struct {
	StatusCode int
	URL        string
	// Attempts : nombre de requêtes réellement envoyées, dans 1..maxRetries. Un
	// statut non retentable rencontré à la 3e tentative (deux 503 puis un 404)
	// donne 3, pas 1 ; maxRetries quand les tentatives ont été épuisées.
	Attempts int
}

func (e *BlobHTTPError) Error() string {
	return fmt.Sprintf("downloadBlob HTTP %d après %d tentative(s): %s", e.StatusCode, e.Attempts, e.URL)
}

// retryableBlobStatus dit si un statut de blob CDN mérite une nouvelle tentative.
// Liste FERMÉE (mesure du 2026-09-05, volet C) :
//   - 304 : artefact d'edge Azure Front Door. Nous n'envoyons AUCUNE en-tête
//     conditionnelle — un « Not Modified » sur une requête inconditionnelle ne dit
//     rien du blob, qui est bien vivant (sondé sans token : HEAD 200, 870 Ko).
//     C'est le statut qui condamnait les mêmes 5-6 films à chaque passe, des mois
//     durant.
//   - 408/429/500/502/503/504 : transitoires côté serveur ou edge.
//
// JAMAIS retentés : 404/410 (blob réellement absent — verdict définitif lu par
// isNotFoundErr) et 401/403 (n'ont aucun sens sur un CDN public et ne changeront
// pas), comme tout autre 4xx.
func retryableBlobStatus(code int) bool {
	switch code {
	case http.StatusNotModified, http.StatusRequestTimeout, http.StatusTooManyRequests,
		http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// blobAttempt : résultat d'UNE tentative de téléchargement de blob.
type blobAttempt struct {
	raw    []byte
	status int // 0 si la requête n'a pas abouti, ou si le corps s'est coupé en route
	err    error
	// retryAfter : fenêtre demandée par le header Retry-After de cette réponse
	// (429/503), déjà parsée et bornée. 0 = header absent ou non parsable.
	retryAfter time.Duration
	// fatal : ne pas retenter. Un SEUL cas — la requête n'a pas pu être
	// construite (URL invalide) : la reconstruire à l'identique échouerait
	// pareil. Un corps coupé en route est, lui, un échec de TRANSPORT : il se
	// retente comme les autres (P1-a, ronde 1 de revue).
	fatal bool
}

// fetchBlobOnce exécute UNE requête GET sur le CDN, sans en-tête d'auth.
// revalider pose « Cache-Control: no-cache » pour forcer l'edge à revalider à
// l'origine — UNIQUEMENT sur une retentative qui suit un 304, jamais sur la
// première requête ni sur les autres statuts.
func (c *HaloAPIClient) fetchBlobOnce(ctx context.Context, blobURL string, revalider bool) blobAttempt {
	c.rateWait(ctx)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, blobURL, nil)
	if err != nil {
		return blobAttempt{err: fmt.Errorf("downloadBlob new request: %w", err), fatal: true}
	}
	if revalider {
		req.Header.Set("Cache-Control", "no-cache")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return blobAttempt{err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Drainer le peu qui reste avant la fermeture : sans ça net/http jette la
		// connexion au lieu de la rendre au pool (une rafale de 304 rouvrirait
		// autant de connexions TLS vers le CDN).
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		return blobAttempt{
			status:     resp.StatusCode,
			retryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()),
		}
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		// Corps coupé à mi-chemin (unexpected EOF, connection reset) : échec de
		// transport, retenté comme tel — status 0, jamais fatal.
		return blobAttempt{err: fmt.Errorf("downloadBlob read: %w", err)}
	}
	return blobAttempt{raw: raw, status: resp.StatusCode}
}

// inflateBlob décompresse le corps zlib d'un blob (le CDN Azure des films Halo
// Infinite renvoie du zlib brut).
func inflateBlob(raw []byte) ([]byte, error) {
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("downloadBlob zlib header: %w", err)
	}
	defer zr.Close()
	out, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("downloadBlob zlib: %w", err)
	}
	return out, nil
}

// blobAbandon journalise l'abandon après maxRetries tentatives et rend l'erreur
// définitive : typée (BlobHTTPError) si la dernière tentative portait un statut,
// sinon l'erreur de transport enveloppée.
func blobAbandon(ctx context.Context, blobURL string, status int, lastErr error, start time.Time) error {
	observability.AddInt(metricBlobRetryExhausted, 1)
	if status != 0 {
		slog.WarnContext(ctx, "halo_api: downloadBlob HTTP error",
			"url", blobURL, "status", status, "attempts", maxRetries,
			"duration_ms", time.Since(start).Milliseconds())
		return &BlobHTTPError{StatusCode: status, URL: blobURL, Attempts: maxRetries}
	}
	// Anti-flood B6.4 : une panne réseau fait échouer tous les téléchargements
	// de blobs en rafale → clé globale par cause, compteur expvar exact.
	if allow, since := observability.AllowThrottledLog(
		"log_throttle_halo_api_download_blob", observability.NetworkFloodWindow); allow {
		slog.WarnContext(ctx, "halo_api: downloadBlob échec réseau",
			"url", blobURL, "attempts", maxRetries, "err", lastErr,
			"throttled_since_last", since)
	}
	return fmt.Errorf("downloadBlob: %w", lastErr)
}

// downloadBlob télécharge un blob du CDN Halo (chunks de film, highlight events)
// et le décompresse. Le CDN est public : aucune en-tête d'auth n'est envoyée.
// Portage de download_film_chunk() (Python api_client.py:485-498).
//
// Retry (2026-09-05) : jusqu'à maxRetries tentatives sur les statuts de
// retryableBlobStatus et sur les échecs de transport, avec le backoff de doGet
// (fenêtre Retry-After honorée quand le CDN en fournit une, bornée par
// backoffCeiling).
// Avant, TOUT statut non-200 était un échec définitif de la tentative unique, et
// un seul 304 d'edge coûtait le film entier (errgroup : première erreur = film
// abandonné) à CHAQUE passe de rattrapage.
func (c *HaloAPIClient) downloadBlob(ctx context.Context, blobURL string) ([]byte, error) {
	// Mode démo : aucune sortie tierce (cf. internal/platform/netguard).
	if err := netguard.Check(ctx, "halo_api.download_blob"); err != nil {
		return nil, err
	}
	start := time.Now()
	var lastStatus int
	var lastErr error
	revalider := false // vrai dès qu'un 304 est vu : les retentatives forcent la revalidation
	for attempt := 0; attempt < maxRetries; attempt++ {
		at := c.fetchBlobOnce(ctx, blobURL, revalider)
		switch {
		case at.fatal:
			return nil, at.err
		case at.err != nil:
			// Transport : requête en erreur, ou corps coupé après un 200.
			lastErr, lastStatus = at.err, 0
			slog.DebugContext(ctx, "halo_api: downloadBlob échec réseau (retry)",
				"url", blobURL, "attempt", attempt+1, "err", at.err)
		case at.status == http.StatusOK:
			out, iErr := inflateBlob(at.raw)
			if iErr != nil {
				return nil, iErr
			}
			if attempt > 0 {
				observability.AddInt(metricBlobRetrySuccess, 1)
				if revalider {
					slog.InfoContext(ctx, "halo_api: downloadBlob 304 puis succès",
						"url", blobURL, "attempts", attempt+1)
				}
			}
			slog.DebugContext(ctx, "halo_api: downloadBlob succès",
				"url", blobURL, "bytes_compressed", len(at.raw), "bytes_inflated", len(out),
				"attempts", attempt+1, "duration_ms", time.Since(start).Milliseconds())
			return out, nil
		case !retryableBlobStatus(at.status):
			slog.WarnContext(ctx, "halo_api: downloadBlob HTTP error",
				"url", blobURL, "status", at.status, "attempts", attempt+1,
				"duration_ms", time.Since(start).Milliseconds())
			return nil, &BlobHTTPError{StatusCode: at.status, URL: blobURL, Attempts: attempt + 1}
		default:
			lastErr, lastStatus = nil, at.status
			revalider = revalider || at.status == http.StatusNotModified
			slog.DebugContext(ctx, "halo_api: downloadBlob HTTP error (retry)",
				"url", blobURL, "status", at.status, "attempt", attempt+1,
				"retry_after_s", at.retryAfter.Seconds())
		}
		// Pas de backoff après la DERNIÈRE tentative : il ne précède plus rien, et
		// un ctx qui expirait pendant ce sommeil masquait l'abandon (ni compteur
		// retry_exhausted, ni WARN). La sortie de boucle passe TOUJOURS par
		// blobAbandon.
		if attempt < maxRetries-1 {
			c.backoff(ctx, attempt, at.retryAfter)
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
		}
	}
	return nil, blobAbandon(ctx, blobURL, lastStatus, lastErr, start)
}

// doGet exécute un GET authentifié avec retry + backoff exponentiel.
// Portage de request_with_retries() (Python api_client.py).
func (c *HaloAPIClient) doGet(ctx context.Context, rawURL string) ([]byte, error) {
	// Mode démo : aucune sortie tierce. Placé AVANT la boucle de retry — c'est
	// elle qui coûtait ~12 s par appel sur les xuid factices de la fixture.
	if err := netguard.Check(ctx, "halo_api.get"); err != nil {
		return nil, err
	}
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
