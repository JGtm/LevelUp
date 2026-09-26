package haloclient

// halo_client_film_taille_test.go — « PRESENT » N'EST PAS « COMPLET » AU TELECHARGEMENT (J2.4 du
// plan de suite de l'audit du decodeur de film, 2026-09-26). Le manifeste de l'API annonce la
// taille de chaque blob (`ChunkSize`, octets du blob BRUT tel que servi par le CDN — cf. le
// temoin `testdata/jgtm_full_match/README.md` : « 0 mismatch vs manifest.ChunkSize » sur les
// blobs bruts). Un blob d'une autre taille n'est pas un film : erreur typee, rien n'est rendu.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"levelup/go-api/internal/observability"
)

// TestFetchFilmChunks_TailleAnnonceeDiffereRendErrChunkIncomplet — le CDN sert MOINS d'octets
// que le manifeste n'en annonce pour un chunk : erreur typee, film non rendu, compteur, et ce
// n'est pas un film perdu (aucun marqueur terminal : reprise au cycle suivant).
func TestFetchFilmChunks_TailleAnnonceeDiffereRendErrChunkIncomplet(t *testing.T) {
	blobs := map[string][]byte{}
	var entrees []map[string]any
	for i, typ := range []int{FilmChunkTypeHeader, FilmChunkTypeReplicationData, FilmChunkTypeHighlightEvents} {
		nom := "c" + string(rune('0'+i)) + ".bin"
		blobs["/"+nom] = zlibCompress(t, []byte("contenu-"+nom))
		entree := filmChunkEntry(i, typ, nom)
		entree["ChunkSize"] = len(blobs["/"+nom])
		entrees = append(entrees, entree)
	}
	entrees[1]["ChunkSize"] = len(blobs["/c1.bin"]) + 5 // le CDN en sert cinq de moins
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/spectate") {
			_ = json.NewEncoder(w).Encode(filmManifestJSON("http://blobs.test/", entrees))
			return
		}
		_, _ = w.Write(blobs[r.URL.Path])
	}))
	t.Cleanup(srv.Close)

	avant := observability.LoadCounter(compteurChunkIncomplet)
	chunks, found, err := newFilmTestClient(srv).GetFilmChunks(context.Background(), testFilmMatchUUID)
	var incomplet *ErrChunkIncomplet
	if !errors.As(err, &incomplet) {
		t.Fatalf("err = %v, attendu *ErrChunkIncomplet", err)
	}
	if incomplet.Annonce != incomplet.Recu+5 {
		t.Errorf("ErrChunkIncomplet = %+v, attendu Annonce = Recu + 5", *incomplet)
	}
	if found || chunks != nil {
		t.Errorf("film incomplet rendu : found=%v, %d chunks", found, len(chunks))
	}
	if IsFilmGoneErr(err) {
		t.Error("blob incomplet classe film PERDU : le marqueur terminal serait pose")
	}
	if d := observability.LoadCounter(compteurChunkIncomplet) - avant; d != 1 {
		t.Errorf("%s += %d, attendu 1", compteurChunkIncomplet, d)
	}
}
