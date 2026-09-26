package haloclient

// halo_client_film_finalise_test.go — UN FILM NON FINALISE N'EST NI ABSENT NI EN PANNE (lot L3,
// 2026-09-23).
//
// Le serveur Halo publie le morceau des temps forts (type 3) environ une minute apres la fin du
// match. Un manifeste servi AVANT decrit un film en cours de publication : le telecharger et le
// rendre comme un film complet a fait cuire `ab526724` sur 34 morceaux sur 37. Le chemin commun
// de telechargement rend desormais [filmcache.ErrFilmNonFinalise], sans rien telecharger.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// serveurDeFilm sert un manifeste donne et compte les blobs demandes.
func serveurDeFilm(t *testing.T, chunks []map[string]any) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var blobs atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			_ = json.NewEncoder(w).Encode(filmManifestJSON("http://blobs.test/", chunks))
			return
		}
		blobs.Add(1)
		_, _ = w.Write(zlibCompress(t, []byte("blob-"+r.URL.Path)))
	}))
	t.Cleanup(srv.Close)
	return srv, &blobs
}

// entreesEnCours : un film en cours de publication — en-tete et replication, pas de temps forts.
func entreesEnCours() []map[string]any {
	return []map[string]any{
		filmChunkEntry(0, FilmChunkTypeHeader, "c0.bin"),
		filmChunkEntry(1, FilmChunkTypeReplicationData, "c1.bin"),
		filmChunkEntry(2, FilmChunkTypeReplicationData, "c2.bin"),
	}
}

// entreesFinalisees : le meme film, finalise.
func entreesFinalisees() []map[string]any {
	return append(entreesEnCours(), filmChunkEntry(3, FilmChunkTypeHighlightEvents, "c3.bin"))
}

// TestGetFilmChunks_ManifesteAPINonFinalise_ErreurTypee : ni 404 (le film n'est pas perdu), ni
// panne (rien n'est casse) — une erreur TYPEE, et aucun blob telecharge.
func TestGetFilmChunks_ManifesteAPINonFinalise_ErreurTypee(t *testing.T) {
	srv, blobs := serveurDeFilm(t, entreesEnCours())
	c := newFilmTestClient(srv)

	chunks, found, err := c.GetFilmChunks(context.Background(), testFilmMatchUUID)
	if !errors.Is(err, filmcache.ErrFilmNonFinalise) {
		t.Fatalf("err = %v, attendu filmcache.ErrFilmNonFinalise", err)
	}
	if found || chunks != nil {
		t.Errorf("film non finalise rendu comme present : found=%v, %d morceaux", found, len(chunks))
	}
	if IsFilmGoneErr(err) {
		t.Error("film non finalise classe PERDU (404/410) : le marqueur terminal serait pose")
	}
	if n := blobs.Load(); n != 0 {
		t.Errorf("%d blobs telecharges pour un film non finalise", n)
	}
}

// TestGetMatchFilm_ManifesteAPINonFinalise_ErreurTypee : la vue « replication seule » passe par
// le meme chemin commun, donc par la meme regle.
func TestGetMatchFilm_ManifesteAPINonFinalise_ErreurTypee(t *testing.T) {
	srv, _ := serveurDeFilm(t, entreesEnCours())
	_, found, err := newFilmTestClient(srv).GetMatchFilm(context.Background(), testFilmMatchUUID)
	if !errors.Is(err, filmcache.ErrFilmNonFinalise) || found {
		t.Fatalf("GetMatchFilm : found=%v err=%v, attendu ErrFilmNonFinalise", found, err)
	}
}

// TestGetFilmChunks_ManifesteAPIVide_ResteAbsent : un manifeste a ZERO morceau est un film
// ABSENT (expire), comme avant le lot — pas un film non finalise.
func TestGetFilmChunks_ManifesteAPIVide_ResteAbsent(t *testing.T) {
	srv, _ := serveurDeFilm(t, []map[string]any{})
	chunks, found, err := newFilmTestClient(srv).GetFilmChunks(context.Background(), testFilmMatchUUID)
	if err != nil || found || chunks != nil {
		t.Fatalf("manifeste vide : (%d, %v, %v), attendu (0, false, nil)", len(chunks), found, err)
	}
}

// TestGetFilmChunks_ManifesteFinalise_RendLeFilm : le controle negatif.
func TestGetFilmChunks_ManifesteFinalise_RendLeFilm(t *testing.T) {
	srv, _ := serveurDeFilm(t, entreesFinalisees())
	chunks, found, err := newFilmTestClient(srv).GetFilmChunks(context.Background(), testFilmMatchUUID)
	if err != nil || !found || len(chunks) != 4 {
		t.Fatalf("film finalise : (%d, %v, %v), attendu (4, true, nil)", len(chunks), found, err)
	}
}

// TestGetFilmChunkURLs_ManifesteNonFinalise_ErreurTypee : la mise en file (VPS) ne confie pas a
// un ouvrier un film que le writer refuserait.
func TestGetFilmChunkURLs_ManifesteNonFinalise_ErreurTypee(t *testing.T) {
	srv, _ := serveurDeFilm(t, entreesEnCours())
	refs, found, err := newFilmTestClient(srv).GetFilmChunkURLs(context.Background(), testFilmMatchUUID)
	if !errors.Is(err, filmcache.ErrFilmNonFinalise) || found || refs != nil {
		t.Fatalf("GetFilmChunkURLs : (%d, %v, %v), attendu ErrFilmNonFinalise", len(refs), found, err)
	}
}

// TestGetFilmChunks_ManifesteLocalPartiel_RepliSurLAPI : un manifeste du CACHE sans temps forts
// (ecrit avant le lot, `ab526724`) ne masque plus l'API : le manifeste servi par le serveur
// prend le relais, les morceaux deja sur disque sont relus, les autres telecharges.
func TestGetFilmChunks_ManifesteLocalPartiel_RepliSurLAPI(t *testing.T) {
	racine := t.TempDir()
	court := testFilmMatchUUID[:8]
	if err := filmcache.EnsureDirs(racine); err != nil {
		t.Fatal(err)
	}
	dossier := filmcache.ChunkDir(racine, court)
	if err := os.MkdirAll(dossier, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"chunk_00.bin", "chunk_01.bin", "chunk_02.bin"} {
		if err := os.WriteFile(filepath.Join(dossier, n), []byte("disque"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	partiel, err := json.Marshal(CachedManifest{Chunks: []CachedChunk{
		{Index: 0, ChunkType: FilmChunkTypeHeader},
		{Index: 1, ChunkType: FilmChunkTypeReplicationData},
		{Index: 2, ChunkType: FilmChunkTypeReplicationData},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filmcache.ManifestPath(racine, court), partiel, 0o644); err != nil {
		t.Fatal(err)
	}
	srv, blobs := serveurDeFilm(t, entreesFinalisees())
	c := newFilmTestClient(srv).WithLocalFilmCache(NewLocalFilmCache(racine))

	chunks, found, err := c.GetFilmChunks(context.Background(), testFilmMatchUUID)
	if err != nil || !found || len(chunks) != 4 {
		t.Fatalf("repli API : (%d, %v, %v), attendu (4, true, nil)", len(chunks), found, err)
	}
	if !filmcache.Finalise(chunks, func(c FilmChunk) int { return c.ChunkType }) {
		t.Error("le film rendu ne porte pas ses temps forts")
	}
	if n := blobs.Load(); n != 1 {
		t.Errorf("%d blobs telecharges, attendu 1 (seul le morceau des temps forts manque)", n)
	}
}
