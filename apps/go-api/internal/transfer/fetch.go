package transfer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/platform/netguard"
)

// Réglages du téléchargement.
const (
	// ExportPath : la route d'export de l'instance hébergée.
	ExportPath = "/api/v1/export/desktop"
	// DefaultFetchTimeout : durée maximale d'un téléchargement.
	DefaultFetchTimeout = 30 * time.Minute
	// netSurface nomme la sortie dans le journal de netguard.
	netSurface = "site_import"
)

// Erreurs du téléchargement ; leur identité fixe le code d'API rendu.
var (
	ErrBadURL      = errors.New("transfer: adresse de l'instance invalide (https attendu)")
	ErrSiteRefused = errors.New("transfer: l'instance a refusé la demande")
	ErrSiteTooBig  = errors.New("transfer: l'archive dépasse la taille maximale")
)

// FetchInput décrit un téléchargement.
type FetchInput struct {
	// BaseURL : l'adresse de l'instance (https ; http seulement vers la boucle locale, pour les essais).
	BaseURL string
	// Token : le jeton d'export, envoyé en en-tête d'autorisation (jamais dans l'adresse).
	Token string
	// Dest : le fichier zip à écrire.
	Dest string
	// MaxBytes : taille maximale de l'archive (zéro = [DefaultMaxUncompressed]).
	MaxBytes int64
	// Timeout : durée maximale (zéro = [DefaultFetchTimeout]).
	Timeout time.Duration
	// Client : le client HTTP (tests) ; nil = un client sans redirection.
	Client *http.Client
}

// exportURL valide l'adresse de l'instance et rend celle de la route d'export.
func exportURL(base string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(base))
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", ErrBadURL
	}
	switch u.Scheme {
	case "https":
	case "http":
		if host := u.Hostname(); host != "localhost" && (net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback()) {
			return "", ErrBadURL
		}
	default:
		return "", ErrBadURL
	}
	return strings.TrimRight(u.Scheme+"://"+u.Host+u.Path, "/") + ExportPath, nil
}

// Fetch télécharge l'archive d'export de l'instance dans in.Dest. Aucune redirection n'est suivie (le jeton
// ne part qu'à l'adresse donnée), la taille est bornée, un échec ne laisse aucun fichier.
func Fetch(ctx context.Context, in FetchInput) error {
	target, err := exportURL(in.BaseURL)
	if err != nil {
		return err
	}
	if err := netguard.Check(ctx, netSurface); err != nil {
		return err
	}
	timeout := in.Timeout
	if timeout <= 0 {
		timeout = DefaultFetchTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return ErrBadURL
	}
	req.Header.Set("Authorization", "Bearer "+in.Token)
	req.Header.Set("Accept", "application/zip")
	client := in.Client
	if client == nil {
		client = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	resp, err := client.Do(req) //nolint:bodyclose // fermé ci-dessous
	if err != nil {
		return fmt.Errorf("transfer: instance injoignable : %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	switch {
	case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
		return ErrTokenInvalid
	case resp.StatusCode == http.StatusTooManyRequests:
		return ErrRateLimited
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("%w : état %d", ErrSiteRefused, resp.StatusCode)
	}
	limit := in.MaxBytes
	if limit <= 0 {
		limit = DefaultMaxUncompressed
	}
	if resp.ContentLength > limit {
		return ErrSiteTooBig
	}
	return writeLimited(in.Dest, resp.Body, limit)
}

// writeLimited écrit le corps dans dest, en échouant au-delà de limit (le fichier partiel est retiré).
func writeLimited(dest string, body io.Reader, limit int64) error {
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) //nolint:gosec // fichier de travail de l'import
	if err != nil {
		return err
	}
	n, err := io.Copy(out, io.LimitReader(body, limit+1))
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > limit {
		err = ErrSiteTooBig
	}
	if err != nil {
		_ = os.Remove(dest)
		return err
	}
	return nil
}
