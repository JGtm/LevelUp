package filmfetch

// aides_test.go — le montage commun des tests du paquet : un lecteur de films dont les deux
// clients de la brique émettent par un `*http.Client` de test (le Doer en production est celui,
// gardé par `netguard`, de l'adaptateur), sous la politique de reprise de production
// (`infinite.RetryHaloClient`) au délai de base raccourcissable.

import (
	"bytes"
	"compress/zlib"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"levelup/go-api/internal/haloapi/infinite"
)

// maxRetries : le nombre d'essais de la politique de production des films (vérifié à l'init : une
// constante sans type se compare aux compteurs des tests sans conversion).
const maxRetries = 4

func init() {
	if infinite.RetryHaloClient().Essais != maxRetries {
		panic("aides_test.go : maxRetries diffère des essais de infinite.RetryHaloClient")
	}
}

// retryBaseDelay : le délai de base de la politique des lecteurs construits par ce fichier. Il vaut
// celui de la production ; [avecRetryBaseDelayCourt] le raccourcit le temps d'un test (aucun test
// du paquet n'est parallèle).
var retryBaseDelay = infinite.RetryHaloClient().DelaiBase

// avecRetryBaseDelayCourt raccourcit le délai de base des lecteurs construits ENSUITE et le
// restaure à la sortie. Sans lui, un cas « tentatives épuisées » dure une douzaine de secondes.
func avecRetryBaseDelayCourt(t *testing.T) {
	t.Helper()
	ancien := retryBaseDelay
	retryBaseDelay = time.Millisecond
	t.Cleanup(func() { retryBaseDelay = ancien })
}

// politiqueTest : la politique de production, au délai de base courant de ce fichier.
func politiqueTest() infinite.Retry {
	r := infinite.RetryHaloClient()
	r.DelaiBase = retryBaseDelay
	return r
}

// fastLimiter retourne un rate.Limiter quasi-illimité (1000 RPS).
func fastLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Every(time.Millisecond), 1)
}

// filmsSur construit un lecteur dont les deux clients émettent par `doer`.
func filmsSur(doer *http.Client) *Films {
	api := infinite.New(doer, politiqueTest()).WithLimiter(fastLimiter())
	return New(api, api, infinite.Auth{Spartan: "s", Clearance: "c"})
}

// clientBlobTest : lecteur minimal dont les blobs tapent directement srv.
func clientBlobTest(srv *httptest.Server) *Films {
	return filmsSur(srv.Client())
}

// redirectTransport redirige toutes les requêtes HTTP vers un serveur httptest : les adresses du
// manifeste et des blobs restent celles de production.
type redirectTransport struct{ host string }

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = "http"
	req2.URL.Host = rt.host
	return http.DefaultTransport.RoundTrip(req2)
}

// newFilmTestClient crée un lecteur dont toutes les requêtes sont redirigées vers srv.
func newFilmTestClient(srv *httptest.Server) *Films {
	host := strings.TrimPrefix(srv.URL, "http://")
	return filmsSur(&http.Client{Transport: &redirectTransport{host: host}})
}

// newFilmTestClientHorsReseau crée un lecteur sans serveur (le cache disque répond seul).
func newFilmTestClientHorsReseau() *Films {
	return filmsSur(&http.Client{})
}

// zlibCompress imite les blobs compressés zlib du CDN des films.
func zlibCompress(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}
	return buf.Bytes()
}

const testFilmMatchUUID = "00000000-0000-0000-0000-000000000001"

// filmManifestJSON construit un manifeste JSON pour les tests film.
func filmManifestJSON(prefix string, chunks []map[string]any) map[string]any {
	return map[string]any{
		"BlobStoragePathPrefix": prefix,
		"CustomData": map[string]any{
			"FilmMajorVersion": 2,
			"Chunks":           chunks,
		},
	}
}

// filmChunkEntry : une entrée de manifeste SANS `ChunkSize` (taille non annoncée, aucun contrôle).
func filmChunkEntry(index, chunkType int, path string) map[string]any {
	return map[string]any{
		"Index": index, "ChunkType": chunkType,
		"ChunkStartTimeOffsetMilliseconds": index * 500,
		"DurationMilliseconds":             500, "FileRelativePath": path,
	}
}
