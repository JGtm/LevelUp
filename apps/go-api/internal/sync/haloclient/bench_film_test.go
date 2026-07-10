//go:build integration

// Package haloclient — bench_film_test.go : benchmark du path parallèle GetMatchFilm.
// Extrait de sync/bench_perf_test.go lors de l'extraction du client (K3e) — il
// exerce le client (GetMatchFilm) + les fixtures film (haloclient), pas de sync.
package haloclient

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// zlibCompressForBench : variante zlib-compression pour *testing.B (le type B ne
// permet pas de réutiliser zlibCompress(t *testing.T)).
func zlibCompressForBench(b *testing.B, data []byte) []byte {
	b.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		b.Fatalf("zlib write: %v", err)
	}
	if err := w.Close(); err != nil {
		b.Fatalf("zlib close: %v", err)
	}
	return buf.Bytes()
}

// BenchmarkGetMatchFilm_20Chunks : 20 chunks × 50ms latence CDN simulée.
func BenchmarkGetMatchFilm_20Chunks(b *testing.B) {
	const nChunks = 20
	const latency = 50 * time.Millisecond

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			_ = json.NewEncoder(w).Encode(filmManyChunks("http://blobs.test/", nChunks))
			return
		}
		time.Sleep(latency)
		_, _ = w.Write(zlibCompressForBench(b, []byte("data-"+r.URL.Path)))
	}))
	b.Cleanup(srv.Close)

	c := newFilmTestClient(srv)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := c.GetMatchFilm(context.Background(), testFilmMatchUUID)
		if err != nil {
			b.Fatalf("GetMatchFilm: %v", err)
		}
	}
}
