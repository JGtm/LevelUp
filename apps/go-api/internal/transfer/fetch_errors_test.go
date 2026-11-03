package transfer

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func fetchTo(t *testing.T, base string, client *http.Client) error {
	t.Helper()
	return Fetch(context.Background(), FetchInput{BaseURL: base, Token: "t", Dest: filepath.Join(t.TempDir(), "x.zip"),
		Client: client, Timeout: 5 * time.Second})
}

// TestFetch_DistinguishesWhyTheSiteCannotBeReached : une instance injoignable, un certificat refusé et un nom
// introuvable donnent trois codes distincts (T7), pas un « import_failed » unique.
func TestFetch_DistinguishesWhyTheSiteCannotBeReached(t *testing.T) {
	// Instance qui ne répond pas : le port est fermé.
	closed := httptest.NewServer(http.NotFoundHandler())
	url := closed.URL
	closed.Close()
	err := fetchTo(t, url, nil)
	if !errors.Is(err, ErrSiteUnreachable) || CodeOf(err) != CodeSiteUnreachable {
		t.Errorf("port fermé : %v (code %q), attendu %q", err, CodeOf(err), CodeSiteUnreachable)
	}

	// Certificat auto-signé : le client par défaut ne le connaît pas.
	tlsSrv := httptest.NewTLSServer(http.NotFoundHandler())
	defer tlsSrv.Close()
	err = fetchTo(t, tlsSrv.URL, nil)
	if !errors.Is(err, ErrSiteTLS) || CodeOf(err) != CodeSiteTLS {
		t.Errorf("certificat refusé : %v (code %q), attendu %q", err, CodeOf(err), CodeSiteTLS)
	}

	// Nom inexistant : le résolveur répond « introuvable ».
	notFound := &http.Client{Transport: &http.Transport{DialContext: func(context.Context, string, string) (net.Conn, error) {
		return nil, &net.DNSError{Err: "no such host", Name: "exemple.invalid", IsNotFound: true}
	}}}
	err = fetchTo(t, "https://exemple.invalid", notFound)
	if !errors.Is(err, ErrSiteNotFound) || CodeOf(err) != CodeSiteNotFound {
		t.Errorf("nom introuvable : %v (code %q), attendu %q", err, CodeOf(err), CodeSiteNotFound)
	}
}
