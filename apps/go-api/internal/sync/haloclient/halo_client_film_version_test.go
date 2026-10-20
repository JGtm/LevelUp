package haloclient

// halo_client_film_version_test.go — LA VERSION DU FILM SURVIT AU CACHE DISQUE.
//
// CE QUE CE TEST FIGE. Le manifeste stocke sur disque ne porte pas `FilmMajorVersion` : ce champ
// etait donc pose a 0 des que le manifeste venait du cache, et `GetHighlightEventsChunk` servait
// 0 a tout le pipeline de synchronisation. Sur un film de version 39-40 (211 des 1 351 du cache),
// 0 fait lire le gamertag douze octets trop tot et effondre le roster du kill-feed. La version
// est desormais lue dans l en-tete du registre (`chunk_00`), qui est LE film : elle vaut donc
// pour tout le cache existant, sans migration.
//
// Le test ne touche AUCUN reseau : manifeste et chunks sont sur disque, la methode rend le chunk
// en cache sans jamais appeler le CDN.

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

const matchIDDeTest = "11111111-2222-3333-4444-555555555555"

// cacheAvecRegistre fabrique un cache film minimal : un manifeste, un registre (`chunk_00`)
// portant `version` en tete, et un chunk highlight. Rend la racine du cache.
func cacheAvecRegistre(t *testing.T, version uint32, avecRegistre bool) string {
	t.Helper()
	racine := t.TempDir()
	court := matchIDDeTest[:8]
	if err := filmcache.EnsureDirs(racine); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}
	dossierChunks := filmcache.ChunkDir(racine, court)
	if err := os.MkdirAll(dossierChunks, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if avecRegistre {
		registre := make([]byte, 64)
		binary.LittleEndian.PutUint32(registre[0:], version)
		binary.LittleEndian.PutUint32(registre[4:], 25)
		copy(registre[8:], "game-engine-team-mapping-component")
		if err := os.WriteFile(filepath.Join(dossierChunks, "chunk_00.bin"), registre, 0o644); err != nil {
			t.Fatalf("ecriture du registre: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(dossierChunks, "chunk_01.bin"), []byte("highlight"), 0o644); err != nil {
		t.Fatalf("ecriture du chunk highlight: %v", err)
	}
	manifeste := CachedManifest{Chunks: []CachedChunk{
		{Index: 0, ChunkType: FilmChunkTypeHeader},
		{Index: 1, ChunkType: FilmChunkTypeHighlightEvents},
	}}
	blob, err := json.Marshal(manifeste)
	if err != nil {
		t.Fatalf("serialisation du manifeste: %v", err)
	}
	if err := os.WriteFile(filmcache.ManifestPath(racine, court), blob, 0o644); err != nil {
		t.Fatalf("ecriture du manifeste: %v", err)
	}
	return racine
}

// TestGetHighlightEventsChunk_VersionDepuisLeRegistre : manifeste en cache, version lue.
func TestGetHighlightEventsChunk_VersionDepuisLeRegistre(t *testing.T) {
	racine := cacheAvecRegistre(t, 40, true)
	client := NewHaloAPIClient("s", "c", 10).WithLocalFilmCache(NewLocalFilmCache(racine))

	data, version, trouve, err := client.GetHighlightEventsChunk(context.Background(), matchIDDeTest)
	if err != nil || !trouve {
		t.Fatalf("GetHighlightEventsChunk = (_, _, %v, %v)", trouve, err)
	}
	if string(data) != "highlight" {
		t.Fatalf("chunk = %q, want \"highlight\"", data)
	}
	if version != 40 {
		t.Fatalf("FilmMajorVersion = %d, want 40 — la version du cache est de nouveau perdue", version)
	}
}

// TestGetHighlightEventsChunk_SansRegistre_VersionInconnue : bobine partielle, aucun registre.
// La methode rend 0 (comportement historique) plutot que d inventer une version.
func TestGetHighlightEventsChunk_SansRegistre_VersionInconnue(t *testing.T) {
	racine := cacheAvecRegistre(t, 40, false)
	client := NewHaloAPIClient("s", "c", 10).WithLocalFilmCache(NewLocalFilmCache(racine))

	_, version, trouve, err := client.GetHighlightEventsChunk(context.Background(), matchIDDeTest)
	if err != nil || !trouve {
		t.Fatalf("GetHighlightEventsChunk = (_, _, %v, %v)", trouve, err)
	}
	if version != 0 {
		t.Fatalf("FilmMajorVersion = %d, want 0", version)
	}
}
