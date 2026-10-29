package filmfetch

// films_test.go — lecture d'un film par le manifeste et ses blobs (ex tests du client de synchronisation).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetMatchFilm_BasicPrefix(t *testing.T) {
	blob := zlibCompress(t, []byte("DATA"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			_ = json.NewEncoder(w).Encode(filmManifestJSON(
				"http://blobs.test/base/",
				[]map[string]any{
					filmChunkEntry(0, FilmChunkTypeReplicationData, "chunk0.bin"),
					filmChunkEntry(1, FilmChunkTypeHighlightEvents, "hev.bin"),
				},
			))
			return
		}
		_, _ = w.Write(blob)
	}))
	defer srv.Close()

	c := newFilmTestClient(srv)
	chunks, found, err := c.GetMatchFilm(context.Background(), testFilmMatchUUID)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected found=true")
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if string(chunks[0].Data) != "DATA" {
		t.Fatalf("unexpected chunk data: %q", chunks[0].Data)
	}
}

func TestGetMatchFilm_MultiChunk(t *testing.T) {
	blob := zlibCompress(t, []byte("CHUNK"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			_ = json.NewEncoder(w).Encode(filmManifestJSON(
				"http://blobs.test/",
				[]map[string]any{
					filmChunkEntry(0, FilmChunkTypeHeader, "header.bin"),
					filmChunkEntry(1, FilmChunkTypeReplicationData, "c1.bin"),
					filmChunkEntry(2, FilmChunkTypeReplicationData, "c2.bin"),
					filmChunkEntry(3, FilmChunkTypeHighlightEvents, "hev.bin"),
				},
			))
			return
		}
		_, _ = w.Write(blob)
	}))
	defer srv.Close()

	c := newFilmTestClient(srv)
	chunks, found, err := c.GetMatchFilm(context.Background(), testFilmMatchUUID)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected found=true")
	}
	// Seul ChunkType=2 (REPLICATION_DATA) est retourné ; header (type=1) ignoré.
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks (type=2 only), got %d", len(chunks))
	}
	if _, hasHeader := chunks[0]; hasHeader {
		t.Fatal("index 0 (header) ne devrait pas être dans le résultat")
	}
}

func TestGetMatchFilm_FilmAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c := newFilmTestClient(srv)
	_, found, err := c.GetMatchFilm(context.Background(), testFilmMatchUUID)
	if err != nil {
		t.Fatalf("expected nil error for 404, got %v", err)
	}
	if found {
		t.Fatal("expected found=false for 404")
	}
}

func TestGetMatchFilm_DownloadFails(t *testing.T) {
	// Le 500 du blob est désormais RETENTÉ (volet C) : backoff raccourci, sinon
	// ce cas coûte ~5,6 s de sommeil pour un verdict connu d'avance.
	avecRetryBaseDelayCourt(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			_ = json.NewEncoder(w).Encode(filmManifestJSON(
				"http://blobs.test/",
				[]map[string]any{
					filmChunkEntry(0, FilmChunkTypeReplicationData, "bad.bin"),
					filmChunkEntry(1, FilmChunkTypeHighlightEvents, "hev.bin"),
				},
			))
			return
		}
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := newFilmTestClient(srv)
	_, _, err := c.GetMatchFilm(context.Background(), testFilmMatchUUID)
	if err == nil {
		t.Fatal("expected error when blob download returns 500")
	}
}

// ─── TestGetHighlightEventsChunk ─────────────────────────────────────────────

func TestGetHighlightEventsChunk_Found(t *testing.T) {
	const payload = "HEV_BYTES"
	blob := zlibCompress(t, []byte(payload))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			manifest := filmManifestJSON("http://blobs.test/", []map[string]any{
				filmChunkEntry(0, FilmChunkTypeReplicationData, "c0.bin"),
				filmChunkEntry(1, FilmChunkTypeHighlightEvents, "hev.bin"),
			})
			manifest["CustomData"].(map[string]any)["FilmMajorVersion"] = 3
			_ = json.NewEncoder(w).Encode(manifest)
			return
		}
		_, _ = w.Write(blob)
	}))
	defer srv.Close()

	c := newFilmTestClient(srv)
	data, version, found, err := c.GetHighlightEventsChunk(context.Background(), testFilmMatchUUID)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected found=true")
	}
	if version != 3 {
		t.Fatalf("expected version=3, got %d", version)
	}
	if string(data) != payload {
		t.Fatalf("unexpected data: %q", data)
	}
}

func TestGetHighlightEventsChunk_NoChunk(t *testing.T) {
	blob := zlibCompress(t, []byte("DATA"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			// Manifest sans ChunkType=3.
			_ = json.NewEncoder(w).Encode(filmManifestJSON(
				"http://blobs.test/",
				[]map[string]any{filmChunkEntry(0, FilmChunkTypeReplicationData, "c0.bin")},
			))
			return
		}
		_, _ = w.Write(blob)
	}))
	defer srv.Close()

	c := newFilmTestClient(srv)
	data, version, found, err := c.GetHighlightEventsChunk(context.Background(), testFilmMatchUUID)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("expected found=false (pas de ChunkType=3)")
	}
	if data != nil || version != 0 {
		t.Fatalf("expected nil/0, got data=%v version=%d", data, version)
	}
}

func TestGetHighlightEventsChunk_FilmAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c := newFilmTestClient(srv)
	_, _, found, err := c.GetHighlightEventsChunk(context.Background(), testFilmMatchUUID)
	if err != nil {
		t.Fatalf("expected nil error for absent film, got %v", err)
	}
	if found {
		t.Fatal("expected found=false for absent film")
	}
}
