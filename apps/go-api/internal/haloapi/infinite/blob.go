package infinite

// blob.go — téléchargement d'un blob du CDN des films (fragments, temps forts). Copié de
// `sync/haloclient/halo_client_http.go` (`downloadBlob`, `fetchBlobOnce`, `retryableBlobStatus`,
// `blobAbandon`) et `halo_client_film_taille.go` (contrôle de la taille annoncée). Le CDN est
// public : aucun en-tête de jeton. Les octets rendus sont COMPRESSÉS ; la décompression est à
// l'appelant.

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// drainBlobMax : ce qui est lu d'un corps non-200 avant fermeture (rendre la connexion au pool).
const drainBlobMax = 64 << 10

// retryableBlobStatus dit si un statut du CDN mérite une nouvelle tentative. Liste FERMÉE :
//   - 304 : artefact d'edge (aucun en-tête conditionnel n'est envoyé, le blob est vivant) ;
//   - 408/429/500/502/503/504 : transitoires côté serveur ou edge.
//
// Jamais retentés : 404/410 (blob absent, verdict définitif), 401/403 (sans objet sur un CDN
// public), tout autre 4xx.
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

// blobAttempt : l'issue d'UNE tentative de téléchargement de blob.
type blobAttempt struct {
	raw        []byte
	status     int // 0 si la requête n'a pas abouti, ou si le corps s'est coupé en route
	err        error
	retryAfter time.Duration
	// fatal : ne pas retenter (requête impossible à construire, ou erreur définitive du Doer).
	fatal bool
}

// DownloadBlob télécharge un blob du CDN et rend ses octets BRUTS (compressés). Reprise : les essais
// et délais de la politique du client ([Retry.Essais], [Retry.DelaiBase], [Retry.Plafond],
// [Retry.HonoreRetryAfter]) sur les statuts de [retryableBlobStatus] et les échecs de transport ;
// jamais d'attente après le dernier essai ; une fin de contexte pendant l'attente rend `ctx.Err()`.
// Après un 304, les essais suivants envoient `Cache-Control: no-cache`.
//
// `tailleAnnoncee` (> 0) est la taille du blob brut annoncée par le manifeste : un blob d'une autre
// taille rend [*ErrChunkIncomplet], sans nouvelle tentative (un CDN qui sert un blob tronqué le
// resservirait). 0 : aucun contrôle.
func (c *Client) DownloadBlob(ctx context.Context, blobURL string, tailleAnnoncee int) ([]byte, error) {
	start := time.Now()
	var lastStatus int
	var lastErr error
	revalider := false
	for n := 0; n < c.retry.Essais; n++ {
		at := c.fetchBlobOnce(ctx, blobURL, revalider)
		switch {
		case at.fatal:
			return nil, at.err
		case at.err != nil:
			lastErr, lastStatus = at.err, 0
			slog.DebugContext(ctx, "halo_api: downloadBlob échec réseau (retry)", "url", blobURL, "attempt", n+1, "err", at.err)
		case at.status == http.StatusOK:
			return c.blobRecu(ctx, blobURL, at.raw, tailleAnnoncee, n, revalider, start)
		case !retryableBlobStatus(at.status):
			slog.WarnContext(ctx, "halo_api: downloadBlob HTTP error", "url", blobURL, "status", at.status,
				"attempts", n+1, "duration_ms", time.Since(start).Milliseconds())
			return nil, &BlobHTTPError{StatusCode: at.status, URL: blobURL, Attempts: n + 1}
		default:
			lastErr, lastStatus = nil, at.status
			revalider = revalider || at.status == http.StatusNotModified
			slog.DebugContext(ctx, "halo_api: downloadBlob HTTP error (retry)", "url", blobURL, "status", at.status,
				"attempt", n+1, "retry_after_s", at.retryAfter.Seconds())
		}
		if n < c.retry.Essais-1 {
			select {
			case <-ctx.Done():
			case <-time.After(c.retry.delai(n, at.retryAfter)):
			}
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
		}
	}
	return nil, c.blobAbandon(ctx, blobURL, lastStatus, lastErr, start)
}

// fetchBlobOnce exécute UNE requête GET sur le CDN, sans jeton ; `revalider` pose
// `Cache-Control: no-cache` (seulement après un 304).
func (c *Client) fetchBlobOnce(ctx context.Context, blobURL string, revalider bool) blobAttempt {
	if c.limiter != nil {
		_ = c.limiter.Wait(ctx) // une fin de contexte fait échouer la requête qui suit
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, blobURL, nil)
	if err != nil {
		return blobAttempt{err: fmt.Errorf("downloadBlob new request: %w", err), fatal: true}
	}
	if revalider {
		req.Header.Set("Cache-Control", "no-cache")
	}
	resp, err := c.doer.Do(req)
	if err != nil {
		return blobAttempt{err: err, fatal: estDefinitif(err)}
	}
	defer func() { _ = resp.Body.Close() }() // corps lu ou drainé ; la fermeture ne change pas l'issue
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, drainBlobMax)) // drainage best-effort
		return blobAttempt{status: resp.StatusCode, retryAfter: ParseRetryAfter(resp.Header.Get("Retry-After"), time.Now())}
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return blobAttempt{err: fmt.Errorf("downloadBlob read: %w", err)}
	}
	return blobAttempt{raw: raw, status: resp.StatusCode}
}

// blobRecu traite un blob servi en 200 : contrôle de la taille annoncée (0 = inconnue), crochet de
// reprise réussie, journal de succès.
func (c *Client) blobRecu(ctx context.Context, blobURL string, raw []byte, tailleAnnoncee, n int, revalider bool, start time.Time) ([]byte, error) {
	if tailleAnnoncee > 0 && len(raw) != tailleAnnoncee {
		appeler(c.crochets.BlobIncomplet)
		slog.WarnContext(ctx, "halo_api: blob de film d'une autre taille que celle annoncée — refusé",
			"url", blobURL, "annonce", tailleAnnoncee, "recu", len(raw), "attempts", n+1)
		return nil, &ErrChunkIncomplet{URL: blobURL, Annonce: tailleAnnoncee, Recu: len(raw)}
	}
	if n > 0 {
		appeler(c.crochets.BlobRepriseReussie)
		if revalider {
			slog.InfoContext(ctx, "halo_api: downloadBlob 304 puis succès", "url", blobURL, "attempts", n+1)
		}
	}
	slog.DebugContext(ctx, "halo_api: downloadBlob succès", "url", blobURL, "bytes_compressed", len(raw),
		"attempts", n+1, "duration_ms", time.Since(start).Milliseconds())
	return raw, nil
}

// blobAbandon signale l'abandon après épuisement des essais et rend l'erreur définitive : typée
// ([*BlobHTTPError]) si le dernier essai portait un statut, sinon l'erreur de transport enveloppée.
func (c *Client) blobAbandon(ctx context.Context, blobURL string, status int, lastErr error, start time.Time) error {
	appeler(c.crochets.BlobEpuise)
	if status != 0 {
		slog.WarnContext(ctx, "halo_api: downloadBlob HTTP error", "url", blobURL, "status", status,
			"attempts", c.retry.Essais, "duration_ms", time.Since(start).Milliseconds())
		return &BlobHTTPError{StatusCode: status, URL: blobURL, Attempts: c.retry.Essais}
	}
	if c.crochets.JournalBlobReseau != nil {
		c.crochets.JournalBlobReseau(ctx, blobURL, c.retry.Essais, lastErr)
	} else {
		slog.WarnContext(ctx, "halo_api: downloadBlob échec réseau", "url", blobURL, "attempts", c.retry.Essais, "err", lastErr)
	}
	return fmt.Errorf("downloadBlob: %w", lastErr)
}

// appeler appelle un crochet non nil.
func appeler(f func()) {
	if f != nil {
		f()
	}
}
