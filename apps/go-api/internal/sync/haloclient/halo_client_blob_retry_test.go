package haloclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
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

// ---------------------------------------------------------------------------
// Ronde 1 de revue adversariale (2026-09-05) : les cas que le lot C.3 n'avait
// pas couverts.
// ---------------------------------------------------------------------------

// (P1-a) Corps coupé à mi-chemin après un 200 : c'est un échec de TRANSPORT, il
// se retente. Avant, il était marqué fatal et condamnait le film comme un 404.
func TestDownloadBlob_CorpsCoupeEstRetente(t *testing.T) {
	avecRetryBaseDelayCourt(t)
	charge := zlibCompress(t, []byte("CORPS_ENTIER"))
	var recues int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&recues, 1) == 1 {
			// Content-Length annoncé plus grand que ce qui est écrit, en-têtes et
			// début de corps POUSSÉS sur le fil (Flush : sans lui le serveur
			// n'envoie rien et l'échec retombe sur le transport, pas sur la
			// lecture), puis connexion coupée : le client lit un unexpected EOF
			// APRÈS avoir reçu un 200.
			w.Header().Set("Content-Length", strconv.Itoa(len(charge)+64))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(charge[:4])
			w.(http.Flusher).Flush()
			panic(http.ErrAbortHandler)
		}
		_, _ = w.Write(charge)
	}))
	defer srv.Close()

	out, err := clientBlobTest(srv).downloadBlob(context.Background(), srv.URL+"/filmChunk16")
	if err != nil {
		t.Fatalf("downloadBlob: %v (un corps coupé doit être retenté, pas fatal)", err)
	}
	if string(out) != "CORPS_ENTIER" {
		t.Fatalf("données rendues = %q, attendu %q", out, "CORPS_ENTIER")
	}
	if got := atomic.LoadInt32(&recues); got != 2 {
		t.Fatalf("requêtes reçues = %d, attendu 2", got)
	}
}

// (P1-b) Le « no-cache » est COLLANT : une fois un 304 vu, toutes les
// retentatives suivantes le portent, même si le statut intermédiaire n'est plus
// un 304. Sans ça, la 3e requête retomberait sur le cache d'edge fautif.
func TestDownloadBlob_NoCacheCollantApres304(t *testing.T) {
	avecRetryBaseDelayCourt(t)
	charge := zlibCompress(t, []byte("APRES_304_PUIS_503"))
	var recues int32
	var mu sync.Mutex
	var cacheControl []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&recues, 1)
		mu.Lock()
		cacheControl = append(cacheControl, r.Header.Get("Cache-Control"))
		mu.Unlock()
		switch n {
		case 1:
			w.WriteHeader(http.StatusNotModified)
		case 2:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			_, _ = w.Write(charge)
		}
	}))
	defer srv.Close()

	avant := observability.LoadCounter(metricBlobRetrySuccess)
	out, err := clientBlobTest(srv).downloadBlob(context.Background(), srv.URL+"/filmChunk16")
	if err != nil {
		t.Fatalf("downloadBlob: %v", err)
	}
	if string(out) != "APRES_304_PUIS_503" {
		t.Fatalf("données rendues = %q", out)
	}
	if got := atomic.LoadInt32(&recues); got != 3 {
		t.Fatalf("requêtes reçues = %d, attendu 3", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if cacheControl[0] != "" {
		t.Errorf("1re requête : Cache-Control = %q, attendu absent", cacheControl[0])
	}
	for i := 1; i < 3; i++ {
		if cacheControl[i] != "no-cache" {
			t.Errorf("requête %d : Cache-Control = %q, attendu no-cache (il doit rester posé après le 503)",
				i+1, cacheControl[i])
		}
	}
	if delta := observability.LoadCounter(metricBlobRetrySuccess) - avant; delta != 1 {
		t.Errorf("compteur %s : delta = %d, attendu 1", metricBlobRetrySuccess, delta)
	}
}

// (P2-b) Retry-After du CDN honoré : la retentative n'arrive pas avant la
// fenêtre demandée, même avec un backoff de base raccourci.
func TestDownloadBlob_RetryAfterHonore(t *testing.T) {
	avecRetryBaseDelayCourt(t)
	charge := zlibCompress(t, []byte("APRES_429"))
	var mu sync.Mutex
	var dates []time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		dates = append(dates, time.Now())
		n := len(dates)
		mu.Unlock()
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write(charge)
	}))
	defer srv.Close()

	out, err := clientBlobTest(srv).downloadBlob(context.Background(), srv.URL+"/filmChunk16")
	if err != nil {
		t.Fatalf("downloadBlob: %v", err)
	}
	if string(out) != "APRES_429" {
		t.Fatalf("données rendues = %q", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(dates) != 2 {
		t.Fatalf("requêtes reçues = %d, attendu 2", len(dates))
	}
	if ecart := dates[1].Sub(dates[0]); ecart < time.Second {
		t.Errorf("2e requête après %v, attendu au moins 1s (Retry-After ignoré ?)", ecart)
	}
}

// (P2-e) La table retryableBlobStatus est couverte EN ENTIER, dans les deux
// sens : retirer un statut de la liste, ou y ajouter un 4xx définitif, fait
// rougir ce test.
func TestDownloadBlob_TableDesStatuts(t *testing.T) {
	avecRetryBaseDelayCourt(t)
	charge := zlibCompress(t, []byte("OK"))

	retentables := []int{
		http.StatusNotModified, http.StatusRequestTimeout, http.StatusTooManyRequests,
		http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout,
	}
	for _, statut := range retentables {
		t.Run("retente_"+strconv.Itoa(statut), func(t *testing.T) {
			var recues int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if atomic.AddInt32(&recues, 1) == 1 {
					w.WriteHeader(statut)
					return
				}
				_, _ = w.Write(charge)
			}))
			defer srv.Close()

			out, err := clientBlobTest(srv).downloadBlob(context.Background(), srv.URL+"/filmChunk16")
			if err != nil {
				t.Fatalf("statut %d : downloadBlob = %v, attendu un succès après retry", statut, err)
			}
			if string(out) != "OK" {
				t.Fatalf("statut %d : données rendues = %q", statut, out)
			}
			if got := atomic.LoadInt32(&recues); got != 2 {
				t.Errorf("statut %d : requêtes reçues = %d, attendu 2", statut, got)
			}
		})
	}

	definitifs := []int{
		http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden,
		http.StatusNotFound, http.StatusGone, http.StatusTeapot,
	}
	for _, statut := range definitifs {
		t.Run("definitif_"+strconv.Itoa(statut), func(t *testing.T) {
			var recues int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				atomic.AddInt32(&recues, 1)
				w.WriteHeader(statut)
			}))
			defer srv.Close()

			_, err := clientBlobTest(srv).downloadBlob(context.Background(), srv.URL+"/filmChunk16")
			var blobErr *BlobHTTPError
			if !errors.As(err, &blobErr) {
				t.Fatalf("statut %d : err = %v (%T), attendu *BlobHTTPError", statut, err, err)
			}
			if blobErr.StatusCode != statut || blobErr.Attempts != 1 {
				t.Errorf("statut %d : err = %+v, attendu {StatusCode:%d Attempts:1}", statut, blobErr, statut)
			}
			if got := atomic.LoadInt32(&recues); got != 1 {
				t.Errorf("statut %d : requêtes reçues = %d, attendu 1 (aucun retry)", statut, got)
			}
		})
	}
}

// (P2-f) 200 dont le corps n'est PAS du zlib (page d'erreur HTML servie en 200) :
// verdict immédiat, une seule requête — retenter coûterait 4 fois 870 Ko pour
// rien. Ce n'est pas un BlobHTTPError (le statut, lui, était bon) et aucun
// compteur de retry ne bouge.
func TestDownloadBlob_200CorpsNonZlib(t *testing.T) {
	var recues int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&recues, 1)
		_, _ = w.Write([]byte("<html>erreur d edge</html>"))
	}))
	defer srv.Close()

	avantSucces := observability.LoadCounter(metricBlobRetrySuccess)
	avantEpuise := observability.LoadCounter(metricBlobRetryExhausted)
	_, err := clientBlobTest(srv).downloadBlob(context.Background(), srv.URL+"/filmChunk16")
	if err == nil {
		t.Fatal("attendu une erreur sur un corps non-zlib")
	}
	var blobErr *BlobHTTPError
	if errors.As(err, &blobErr) {
		t.Errorf("err = %v, attendu une erreur de décompression et non un *BlobHTTPError", err)
	}
	if got := atomic.LoadInt32(&recues); got != 1 {
		t.Errorf("requêtes reçues = %d, attendu 1 (un corps illisible ne se retente pas)", got)
	}
	if d := observability.LoadCounter(metricBlobRetrySuccess) - avantSucces; d != 0 {
		t.Errorf("compteur %s : delta = %d, attendu 0", metricBlobRetrySuccess, d)
	}
	if d := observability.LoadCounter(metricBlobRetryExhausted) - avantEpuise; d != 0 {
		t.Errorf("compteur %s : delta = %d, attendu 0", metricBlobRetryExhausted, d)
	}
}

// (P2-g) Borne basse de retryBaseDelay. Le passage const vers var (C.3.4) a
// supprimé la garantie de compilation : ce test la remplace. Il lit la valeur
// SANS appeler avecRetryBaseDelayCourt — les surcharges des autres tests sont
// restaurées par leur t.Cleanup, et aucun test du paquet n'est parallèle.
func TestRetryBaseDelay_BorneBasse(t *testing.T) {
	const plancher = 500 * time.Millisecond
	if retryBaseDelay < plancher {
		t.Fatalf("retryBaseDelay = %v, attendu au moins %v — un backoff plus court transformerait "+
			"un retry borné en martèlement du CDN (ou une surcharge de test a fui)", retryBaseDelay, plancher)
	}
}

// transportBlob304 rend un 304 à chaque appel et annule le contexte PILE après
// la dernière tentative — le moment exact où l'ancien code partait dormir un
// backoff qui ne précédait plus rien, et rendait ctx.Err() : ni compteur
// retry_exhausted, ni WARN, l'abandon était invisible.
type transportBlob304 struct {
	appels int32
	annule context.CancelFunc
}

func (tr *transportBlob304) RoundTrip(r *http.Request) (*http.Response, error) {
	if int(atomic.AddInt32(&tr.appels, 1)) == maxRetries {
		tr.annule()
	}
	return &http.Response{
		StatusCode: http.StatusNotModified,
		Body:       http.NoBody,
		Header:     make(http.Header),
		Request:    r,
	}, nil
}

// (P2-a) Pas de backoff après la dernière tentative, et la sortie de boucle
// passe TOUJOURS par blobAbandon.
func TestDownloadBlob_AbandonNonMasqueParUnCtxExpire(t *testing.T) {
	avecRetryBaseDelayCourt(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr := &transportBlob304{annule: cancel}
	client := &HaloAPIClient{http: &http.Client{Transport: tr}, limiter: fastLimiter()}

	avant := observability.LoadCounter(metricBlobRetryExhausted)
	_, err := client.downloadBlob(ctx, "https://cdn.exemple.invalid/filmChunk16")
	var blobErr *BlobHTTPError
	if !errors.As(err, &blobErr) {
		t.Fatalf("err = %v (%T), attendu *BlobHTTPError — l'abandon ne doit plus être masqué "+
			"par un ctx qui expire pendant le backoff terminal", err, err)
	}
	if blobErr.StatusCode != http.StatusNotModified || blobErr.Attempts != maxRetries {
		t.Errorf("err = %+v, attendu {StatusCode:304 Attempts:%d}", blobErr, maxRetries)
	}
	if got := atomic.LoadInt32(&tr.appels); int(got) != maxRetries {
		t.Errorf("tentatives = %d, attendu %d", got, maxRetries)
	}
	if delta := observability.LoadCounter(metricBlobRetryExhausted) - avant; delta != 1 {
		t.Errorf("compteur %s : delta = %d, attendu 1", metricBlobRetryExhausted, delta)
	}
}
