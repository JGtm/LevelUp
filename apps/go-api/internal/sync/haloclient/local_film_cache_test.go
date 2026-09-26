package haloclient

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// TestLocalFilmCacheLoadChunk_MemeValidationQueFilmcache — le lecteur du client Halo lit le
// cache par `filmcache` : un chunk tronque rend la MEME erreur typee que `filmcache.Source`, un
// chunk entier ses octets, un chunk absent (nil, nil).
func TestLocalFilmCacheLoadChunk_MemeValidationQueFilmcache(t *testing.T) {
	racine := t.TempDir()
	if err := filmcache.EnsureDirs(racine); err != nil {
		t.Fatal(err)
	}
	const match = "0bad0202-1111-2222-3333-444455556666"
	court := title.FilmShortMatchID(match)
	if err := filmcache.Write(t.Context(), racine, court, []filmcache.WriteChunk{
		{Index: 0, ChunkType: 1, Data: []byte("header")},
		{Index: 1, ChunkType: 2, Data: []byte("replication")},
		{Index: 2, ChunkType: filmcache.ChunkTypeTempsForts, Data: []byte("killfeed")},
	}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	// Le nom des fichiers de chunks n'est declare que dans `filmcache` : on prend la deuxieme
	// entree du dossier (tri par nom), on ne recopie pas la convention.
	entrees, err := os.ReadDir(filmcache.ChunkDir(racine, court))
	if err != nil || len(entrees) != 3 {
		t.Fatalf("chunks ecrits = %d (err %v), attendu 3", len(entrees), err)
	}
	if err := os.WriteFile(filepath.Join(filmcache.ChunkDir(racine, court), entrees[1].Name()),
		[]byte("repl"), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := NewLocalFilmCache(racine)
	if data, err := cache.LoadChunk(match, 0); err != nil || string(data) != "header" {
		t.Errorf("LoadChunk(0) = %q (err %v), attendu header", data, err)
	}
	if data, err := cache.LoadChunk(match, 7); err != nil || data != nil {
		t.Errorf("LoadChunk(7) = %q (err %v), attendu (nil, nil) pour un chunk absent", data, err)
	}

	_, errCache := cache.LoadChunk(match, 1)
	src, _, err := filmcache.Open(racine, court)
	if err != nil {
		t.Fatal(err)
	}
	_, errFilmcache := src.Chunk(1)
	var parCache, parFilmcache *filmcache.ErrChunkTronque
	if !errors.As(errCache, &parCache) {
		t.Fatalf("LoadChunk(1) err = %v, attendu *filmcache.ErrChunkTronque", errCache)
	}
	if !errors.As(errFilmcache, &parFilmcache) {
		t.Fatalf("Source.Chunk(1) err = %v, attendu *filmcache.ErrChunkTronque", errFilmcache)
	}
	if *parCache != *parFilmcache {
		t.Errorf("validations divergentes : LoadChunk %+v, filmcache %+v", *parCache, *parFilmcache)
	}
}
