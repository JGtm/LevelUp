package main

// job_test.go — le TRAVAIL de l'ouvrier exercé SANS décoder un vrai film : téléchargement
// des morceaux (URL pré-signées), décompression zlib du CDN, préservation des métadonnées et
// de l'ordre, et le nettoyage qui protège le cache-archive du dépôt. Le décodage réel est
// prouvé au volet B (E2E avec mini-film versionné) — ici, aucun octet de film.

import (
	"bytes"
	"compress/zlib"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// servirZlib rend un handler qui compresse en zlib ce que fabrique produce(nom-de-fichier) —
// exactement ce que fait le CDN Azure des films, et ce que l'ouvrier doit savoir défaire.
func servirZlib(produce func(nom string) []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		zw := zlib.NewWriter(&buf)
		_, _ = zw.Write(produce(filepath.Base(r.URL.Path)))
		_ = zw.Close()
		_, _ = w.Write(buf.Bytes())
	}
}

func TestDownloadChunk_DecompresseLeZlib(t *testing.T) {
	original := []byte("REPLICATION_DATA brut du morceau de film")
	srv := httptest.NewServer(servirZlib(func(string) []byte { return original }))
	defer srv.Close()

	got, err := downloadChunk(context.Background(), srv.Client(), srv.URL+"/chunk_00.bin")
	if err != nil {
		t.Fatalf("downloadChunk: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("morceau décompressé = %q, veut %q", got, original)
	}
}

func TestDownloadChunk_HTTPErreur_URLExpiree(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "expiré", http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := downloadChunk(context.Background(), srv.Client(), srv.URL+"/x")
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("err = %v, veut la mention HTTP 404 (URL pré-signée expirée)", err)
	}
}

func TestDownloadChunk_ZlibInvalide(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ceci n'est pas du zlib")) // pas d'en-tête zlib valide
	}))
	defer srv.Close()

	_, err := downloadChunk(context.Background(), srv.Client(), srv.URL+"/x")
	if err == nil || !strings.Contains(err.Error(), "zlib") {
		t.Fatalf("err = %v, veut la mention zlib", err)
	}
}

func TestFetchChunks_TelechargeDecompresseEtPreserve(t *testing.T) {
	// CDN : /<nom> -> zlib("data-<nom>"). L'ouvrier doit rendre les morceaux DANS L'ORDRE,
	// métadonnées (index, type, bornes) intactes, données décompressées.
	srv := httptest.NewServer(servirZlib(func(nom string) []byte { return []byte("data-" + nom) }))
	defer srv.Close()

	p := &domain.BuildQueuePayload{Chunks: []domain.BuildQueueChunk{
		{Index: 0, ChunkType: 2, StartMS: 0, DurationMS: 100, URL: srv.URL + "/c0"},
		{Index: 1, ChunkType: 6, StartMS: 100, DurationMS: 250, URL: srv.URL + "/c1"},
	}}
	got, err := (&worker{}).fetchChunks(context.Background(), p)
	if err != nil {
		t.Fatalf("fetchChunks: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got = %d morceaux, veut 2", len(got))
	}
	if got[0].Index != 0 || got[1].Index != 1 || got[1].ChunkType != 6 || got[1].DurationMS != 250 {
		t.Fatalf("métadonnées non préservées: %+v", got)
	}
	if string(got[0].Data) != "data-c0" || string(got[1].Data) != "data-c1" {
		t.Fatalf("données décompressées = %q / %q", got[0].Data, got[1].Data)
	}
}

func TestFetchChunks_ContexteAnnule_NeTelechargeRien(t *testing.T) {
	p := &domain.BuildQueuePayload{Chunks: []domain.BuildQueueChunk{
		{Index: 0, URL: "http://127.0.0.1:0/jamais-appele"},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // déjà annulé : fetchChunks doit rendre l'erreur AVANT tout appel réseau
	if _, err := (&worker{}).fetchChunks(ctx, p); err == nil {
		t.Fatal("fetchChunks devrait rendre l'erreur d'un contexte annulé")
	}
}

func TestCleanupFilm_SupprimeLesMorceaux(t *testing.T) {
	work := t.TempDir()
	const short = "abcd1234"
	dir := ecrireMorceauBidon(t, work, short)

	wk := &worker{workDir: work, keepsFilms: false}
	job := &domain.BuildQueueJob{MatchID: "m", Payload: &domain.BuildQueuePayload{ShortID: short}}
	wk.cleanupFilm(context.Background(), job)

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("le dossier de morceaux existe encore (%v) — cleanup devait le supprimer", err)
	}
}

func TestCleanupFilm_ConserveLeCacheDuDepot(t *testing.T) {
	// keepsFilms=true : le dossier de travail EST le cache film du dépôt, archive
	// IRREMPLAÇABLE (les films expirent côté serveur). L'ouvrier n'y touche jamais.
	work := t.TempDir()
	const short = "abcd1234"
	dir := ecrireMorceauBidon(t, work, short)

	wk := &worker{workDir: work, keepsFilms: true}
	job := &domain.BuildQueueJob{MatchID: "m", Payload: &domain.BuildQueuePayload{ShortID: short}}
	wk.cleanupFilm(context.Background(), job)

	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("le cache du dépôt (archive) a été touché: %v", err)
	}
}

// ecrireMorceauBidon crée le dossier de morceaux d'un film et y dépose un fichier, puis rend
// le chemin du dossier.
func ecrireMorceauBidon(t *testing.T, work, short string) string {
	t.Helper()
	dir := filmcache.ChunkDir(work, short)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "chunk_00.bin"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return dir
}
