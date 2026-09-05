package haloclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/observability"
)

// Tests du retry de downloadBlob (volet C, 2026-09-05). Le blob CDN est public :
// un 304 d'edge Azure Front Door n'est pas un blob mort, il se retente.

// avecRetryBaseDelayCourt raccourcit le backoff le temps du test et restaure la
// valeur de production à la sortie. Sans lui, un cas « tentatives épuisées »
// dure une douzaine de secondes.
func avecRetryBaseDelayCourt(t *testing.T) {
	t.Helper()
	ancien := retryBaseDelay
	retryBaseDelay = time.Millisecond
	t.Cleanup(func() { retryBaseDelay = ancien })
}

// clientBlobTest : client minimal dont downloadBlob tape directement srv.
func clientBlobTest(srv *httptest.Server) *HaloAPIClient {
	return &HaloAPIClient{http: srv.Client(), limiter: fastLimiter()}
}

// (a) 304, 304, puis 200 : les données sont rendues, la 1re requête ne porte pas
// Cache-Control, les retentatives si.
func TestDownloadBlob_304PuisSucces(t *testing.T) {
	avecRetryBaseDelayCourt(t)
	charge := zlibCompress(t, []byte("BLOB_OK"))
	var recues int32
	var mu sync.Mutex
	var cacheControl []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&recues, 1)
		mu.Lock()
		cacheControl = append(cacheControl, r.Header.Get("Cache-Control"))
		mu.Unlock()
		if n <= 2 {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		_, _ = w.Write(charge)
	}))
	defer srv.Close()

	avant := observability.LoadCounter(metricBlobRetrySuccess)
	out, err := clientBlobTest(srv).downloadBlob(context.Background(), srv.URL+"/filmChunk16")
	if err != nil {
		t.Fatalf("downloadBlob: %v", err)
	}
	if string(out) != "BLOB_OK" {
		t.Fatalf("données rendues = %q, attendu %q", out, "BLOB_OK")
	}
	if got := atomic.LoadInt32(&recues); got != 3 {
		t.Fatalf("requêtes reçues = %d, attendu 3", got)
	}
	if cacheControl[0] != "" {
		t.Errorf("1re requête : Cache-Control = %q, attendu absent", cacheControl[0])
	}
	for i := 1; i < 3; i++ {
		if cacheControl[i] != "no-cache" {
			t.Errorf("requête %d : Cache-Control = %q, attendu \"no-cache\"", i+1, cacheControl[i])
		}
	}
	if delta := observability.LoadCounter(metricBlobRetrySuccess) - avant; delta != 1 {
		t.Errorf("compteur %s : delta = %d, attendu 1", metricBlobRetrySuccess, delta)
	}
}

// (b) 304 à chaque tentative : BlobHTTPError{304, Attempts: maxRetries}.
func TestDownloadBlob_304Epuise(t *testing.T) {
	avecRetryBaseDelayCourt(t)
	var recues int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&recues, 1)
		w.WriteHeader(http.StatusNotModified)
	}))
	defer srv.Close()

	avant := observability.LoadCounter(metricBlobRetryExhausted)
	_, err := clientBlobTest(srv).downloadBlob(context.Background(), srv.URL+"/filmChunk16")
	var blobErr *BlobHTTPError
	if !errors.As(err, &blobErr) {
		t.Fatalf("err = %v (%T), attendu *BlobHTTPError", err, err)
	}
	if blobErr.StatusCode != http.StatusNotModified || blobErr.Attempts != maxRetries {
		t.Errorf("err = %+v, attendu {StatusCode:304 Attempts:%d}", blobErr, maxRetries)
	}
	if got := atomic.LoadInt32(&recues); got != maxRetries {
		t.Errorf("requêtes reçues = %d, attendu %d", got, maxRetries)
	}
	if delta := observability.LoadCounter(metricBlobRetryExhausted) - avant; delta != 1 {
		t.Errorf("compteur %s : delta = %d, attendu 1", metricBlobRetryExhausted, delta)
	}
}

// (c) 404 : verdict définitif en UNE requête, aucun retry.
func TestDownloadBlob_404SansRetry(t *testing.T) {
	var recues int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&recues, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := clientBlobTest(srv).downloadBlob(context.Background(), srv.URL+"/filmChunk16")
	var blobErr *BlobHTTPError
	if !errors.As(err, &blobErr) {
		t.Fatalf("err = %v (%T), attendu *BlobHTTPError", err, err)
	}
	if blobErr.StatusCode != http.StatusNotFound || blobErr.Attempts != 1 {
		t.Errorf("err = %+v, attendu {StatusCode:404 Attempts:1}", blobErr)
	}
	if got := atomic.LoadInt32(&recues); got != 1 {
		t.Errorf("requêtes reçues = %d, attendu 1 (aucun retry sur 404)", got)
	}
}

// (d, moitié client) 403 : une seule requête. Le CDN est public — un refus
// d'auth n'y a aucun sens et ne changera pas. La moitié pool (notifyPoolOnError
// ne marque PAS le token) est dans internal/sync/pooled_client_test.go.
func TestDownloadBlob_403SansRetry(t *testing.T) {
	var recues int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&recues, 1)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := clientBlobTest(srv).downloadBlob(context.Background(), srv.URL+"/filmChunk16")
	var blobErr *BlobHTTPError
	if !errors.As(err, &blobErr) {
		t.Fatalf("err = %v (%T), attendu *BlobHTTPError", err, err)
	}
	if blobErr.StatusCode != http.StatusForbidden {
		t.Errorf("statut = %d, attendu 403", blobErr.StatusCode)
	}
	if got := atomic.LoadInt32(&recues); got != 1 {
		t.Errorf("requêtes reçues = %d, attendu 1 (aucun retry sur 403)", got)
	}
	// Le type ne doit JAMAIS être vu comme un *HTTPError : sinon un 403 du CDN
	// public marquerait un token Halo valide comme unhealthy.
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		t.Error("errors.As(*HTTPError) matche une erreur de blob — le pool serait poisonné")
	}
}

// (e) 503 puis succès.
func TestDownloadBlob_503PuisSucces(t *testing.T) {
	avecRetryBaseDelayCourt(t)
	charge := zlibCompress(t, []byte("APRES_503"))
	var recues int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&recues, 1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if r.Header.Get("Cache-Control") != "" {
			t.Errorf("Cache-Control posé après un 503, attendu seulement après un 304")
		}
		_, _ = w.Write(charge)
	}))
	defer srv.Close()

	out, err := clientBlobTest(srv).downloadBlob(context.Background(), srv.URL+"/filmChunk16")
	if err != nil {
		t.Fatalf("downloadBlob: %v", err)
	}
	if string(out) != "APRES_503" {
		t.Fatalf("données rendues = %q", out)
	}
	if got := atomic.LoadInt32(&recues); got != 2 {
		t.Errorf("requêtes reçues = %d, attendu 2", got)
	}
}

// (f) échec réseau (serveur fermé) : retenté jusqu'à épuisement, erreur de
// transport enveloppée (PAS un BlobHTTPError : aucun statut observé).
func TestDownloadBlob_EchecReseauRetente(t *testing.T) {
	avecRetryBaseDelayCourt(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	url := srv.URL + "/filmChunk16"
	client := clientBlobTest(srv)
	srv.Close() // le serveur ne répond plus : c.http.Do échoue

	_, err := client.downloadBlob(context.Background(), url)
	if err == nil {
		t.Fatal("attendu une erreur sur serveur fermé")
	}
	var blobErr *BlobHTTPError
	if errors.As(err, &blobErr) {
		t.Errorf("err = %v, attendu une erreur de transport et non un *BlobHTTPError", err)
	}
}

// (g) ctx annulé pendant le backoff : retour immédiat avec l'erreur de contexte.
func TestDownloadBlob_ContexteAnnulePendantBackoff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	debut := time.Now()
	_, err := clientBlobTest(srv).downloadBlob(ctx, srv.URL+"/filmChunk16")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, attendu context.Canceled", err)
	}
	// Backoff nominal : 0,8 + 1,6 + 3,2 s. Un retour sous 1 s prouve le
	// court-circuit sur ctx.Done().
	if ecoule := time.Since(debut); ecoule > time.Second {
		t.Errorf("retour après %v, attendu < 1s (annulation non propagée ?)", ecoule)
	}
}

// (h) isNotFoundErr : typé uniquement, le texte n'est plus une API.
func TestIsNotFoundErr_TypeUniquement(t *testing.T) {
	cas := []struct {
		nom    string
		err    error
		attend bool
	}{
		{"blob 404", &BlobHTTPError{StatusCode: 404, URL: "u", Attempts: 1}, true},
		{"blob 410", &BlobHTTPError{StatusCode: 410, URL: "u", Attempts: 1}, true},
		{"blob 304", &BlobHTTPError{StatusCode: 304, URL: "u", Attempts: maxRetries}, false},
		{"api 404", &HTTPError{StatusCode: 404, URL: "u"}, true},
		{"api 410", &HTTPError{StatusCode: 410, URL: "u"}, true},
		{"api 503", &HTTPError{StatusCode: 503, URL: "u"}, false},
		{"blob 404 enveloppé (errgroup)", fmt.Errorf("GetMatchFilm chunk 3: %w",
			&BlobHTTPError{StatusCode: 404, URL: "u", Attempts: 1}), true},
		{"erreur quelconque", errors.New("connection refused"), false},
		{"texte HTTP 404", errors.New("HTTP 404"), false},
		{"texte ressource absente", errors.New("ressource absente"), false},
		{"nil", nil, false},
	}
	for _, c := range cas {
		if got := isNotFoundErr(c.err); got != c.attend {
			t.Errorf("%s : isNotFoundErr = %v, attendu %v", c.nom, got, c.attend)
		}
	}
}
