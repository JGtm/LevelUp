package filmfetch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"levelup/go-api/internal/observability"
)

// serveurReprise sert le manifeste `chunks` puis, pour chaque blob, un 503 au premier essai et
// `corps(chemin)` au second.
func serveurReprise(t *testing.T, chunks []map[string]any, corps func(chemin string) []byte) *httptest.Server {
	t.Helper()
	return serveurBlobs(t, chunks, corps, true)
}

// serveurDirect sert le manifeste `chunks` puis `corps(chemin)` dès le premier essai de chaque blob.
func serveurDirect(t *testing.T, chunks []map[string]any, corps func(chemin string) []byte) *httptest.Server {
	t.Helper()
	return serveurBlobs(t, chunks, corps, false)
}

// serveurBlobs : `repriseDAbord` fait répondre un 503 au premier essai de chaque blob.
func serveurBlobs(t *testing.T, chunks []map[string]any, corps func(chemin string) []byte, repriseDAbord bool) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	vus := map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			_ = json.NewEncoder(w).Encode(filmManifestJSON("http://blobs.test/", chunks))
			return
		}
		mu.Lock()
		deja := vus[r.URL.Path]
		vus[r.URL.Path] = true
		mu.Unlock()
		if repriseDAbord && !deja {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write(corps(r.URL.Path))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestCompteurRepriseReussie_ApresDecompression : `halo_api_blob_retry_success` compte un blob
// obtenu en reprise ET lisible. Un blob servi en reprise mais illisible (corps non zlib) n'est pas
// une reprise réussie : le compteur ne bouge pas, par le blob seul comme par le film.
func TestCompteurRepriseReussie_ApresDecompression(t *testing.T) {
	avecRetryBaseDelayCourt(t)
	nonZlib := func(string) []byte { return []byte("<html>erreur d edge</html>") }

	srv := serveurReprise(t, nil, nonZlib)
	avant := observability.LoadCounter(metricBlobRetrySuccess)
	if _, err := clientBlobTest(srv).downloadBlob(context.Background(), srv.URL+"/b", 0); err == nil {
		t.Fatal("blob non zlib : erreur attendue")
	}
	if d := observability.LoadCounter(metricBlobRetrySuccess) - avant; d != 0 {
		t.Errorf("blob seul, reprise illisible : delta = %d, attendu 0", d)
	}

	srv = serveurReprise(t, entreesFinalisees(), nonZlib)
	avant = observability.LoadCounter(metricBlobRetrySuccess)
	if _, _, err := newFilmTestClient(srv).GetFilmChunks(context.Background(), testFilmMatchUUID); err == nil {
		t.Fatal("film aux blobs non zlib : erreur attendue")
	}
	if d := observability.LoadCounter(metricBlobRetrySuccess) - avant; d != 0 {
		t.Errorf("film, reprises illisibles : delta = %d, attendu 0", d)
	}

	srv = serveurReprise(t, entreesFinalisees(), func(p string) []byte { return zlibCompress(t, []byte(p)) })
	avant = observability.LoadCounter(metricBlobRetrySuccess)
	chunks, _, err := newFilmTestClient(srv).GetFilmChunks(context.Background(), testFilmMatchUUID)
	if err != nil || len(chunks) != 4 {
		t.Fatalf("film en reprise lisible : %d morceaux, %v", len(chunks), err)
	}
	if d := observability.LoadCounter(metricBlobRetrySuccess) - avant; d != 4 {
		t.Errorf("film, quatre reprises lisibles : delta = %d, attendu 4", d)
	}
}

// TestCompteurRepriseReussie_PremierEssaiNeComptePas : un blob lisible obtenu du PREMIER essai n'est
// pas une reprise : le compteur ne bouge ni pour le blob seul ni pour le film (le seuil de
// `compterReprise` est « plus d'un essai », pas « au moins un »).
func TestCompteurRepriseReussie_PremierEssaiNeComptePas(t *testing.T) {
	avecRetryBaseDelayCourt(t)
	zlibDuChemin := func(p string) []byte { return zlibCompress(t, []byte(p)) }

	srv := serveurDirect(t, nil, zlibDuChemin)
	avant := observability.LoadCounter(metricBlobRetrySuccess)
	if _, err := clientBlobTest(srv).downloadBlob(context.Background(), srv.URL+"/b", 0); err != nil {
		t.Fatalf("blob lisible au premier essai : %v", err)
	}
	if d := observability.LoadCounter(metricBlobRetrySuccess) - avant; d != 0 {
		t.Errorf("blob seul, premier essai : delta = %d, attendu 0", d)
	}

	srv = serveurDirect(t, entreesFinalisees(), zlibDuChemin)
	avant = observability.LoadCounter(metricBlobRetrySuccess)
	chunks, _, err := newFilmTestClient(srv).GetFilmChunks(context.Background(), testFilmMatchUUID)
	if err != nil || len(chunks) != 4 {
		t.Fatalf("film lisible au premier essai : %d morceaux, %v", len(chunks), err)
	}
	if d := observability.LoadCounter(metricBlobRetrySuccess) - avant; d != 0 {
		t.Errorf("film, premier essai : delta = %d, attendu 0", d)
	}
}
