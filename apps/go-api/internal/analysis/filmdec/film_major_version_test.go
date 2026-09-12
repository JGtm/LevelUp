package filmdec

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// enTeteRegistre fabrique un en-tete de registre synthetique : version, second u32, puis le nom
// du premier composant — la forme exacte mesuree sur le cache.
func enTeteRegistre(version, second uint32) []byte {
	b := make([]byte, 8, 64)
	binary.LittleEndian.PutUint32(b[0:], version)
	binary.LittleEndian.PutUint32(b[4:], second)
	return append(b, "game-engine-team-mapping-component\x00"...)
}

func TestFilmMajorVersionFromHeader(t *testing.T) {
	cas := []struct {
		nom     string
		chunk   []byte
		attendu int
		ok      bool
	}{
		{"v40 (films 2025)", enTeteRegistre(40, 25), 40, true},
		{"v37 (temoin 2024)", enTeteRegistre(37, 24), 37, true},
		{"v41 (films recents)", enTeteRegistre(41, 27), 41, true},
		{"exactement quatre octets", []byte{0x27, 0, 0, 0}, 39, true},
		{"trois octets : illisible", []byte{0x28, 0, 0}, FilmMajorVersionUnknown, false},
		{"chunk vide", nil, FilmMajorVersionUnknown, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got, ok := FilmMajorVersionFromHeader(c.chunk)
			if ok != c.ok || got != c.attendu {
				t.Fatalf("FilmMajorVersionFromHeader = (%d, %v), attendu (%d, %v)", got, ok, c.attendu, c.ok)
			}
		})
	}
}

// TestFilmMajorVersionFilmSansRegistre : une bobine qui commence a `chunk_01` n'a pas de registre.
// La version n'est alors PAS lisible, et le helper le dit plutot que de rendre les octets du
// premier chunk de donnees.
func TestFilmMajorVersionFilmSansRegistre(t *testing.T) {
	metaAvecRegistre := []filmsource.ChunkMeta{{Index: 0}, {Index: 1}}
	f, err := filmsource.Load(filmsource.MemoryChunks{enTeteRegistre(40, 25), {1, 2, 3}}, metaAvecRegistre)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if v, ok := FilmMajorVersion(f); !ok || v != 40 {
		t.Fatalf("film avec registre : (%d, %v), attendu (40, true)", v, ok)
	}

	metaSansRegistre := []filmsource.ChunkMeta{{Index: 1}, {Index: 2}}
	sans, err := filmsource.Load(filmsource.MemoryChunks{{0x28, 0, 0, 0}, {1, 2, 3}}, metaSansRegistre)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if v, ok := FilmMajorVersion(sans); ok || v != FilmMajorVersionUnknown {
		t.Fatalf("film sans registre : (%d, %v), attendu (0, false)", v, ok)
	}

	if v, ok := FilmMajorVersion(nil); ok || v != FilmMajorVersionUnknown {
		t.Fatalf("film nil : (%d, %v), attendu (0, false)", v, ok)
	}
}

// TestFilmMajorVersionCacheReel lit la version des films du cache local. GARDE PAR ENV : la CI n'a
// pas de cache film.
//
//	FILMDEC_CACHE_CHUNKS=<racine>/film_chunks go test ./internal/analysis/filmdec/ -run TestFilmMajorVersionCacheReel -v
//
// Les trois attendus sont ceux verifies sur pieces le 2026-09-12 (cf. l'en-tete du fichier).
func TestFilmMajorVersionCacheReel(t *testing.T) {
	racine := strings.TrimSpace(os.Getenv("FILMDEC_CACHE_CHUNKS"))
	if racine == "" {
		t.Skip("FILMDEC_CACHE_CHUNKS absent : pas de cache film sur cette machine")
	}
	attendu := map[string]int{"e5adf7b2": 40, "a26dbcdb": 37, "5676a9ba": 41}
	for court, want := range attendu {
		raw, err := os.ReadFile(filepath.Join(racine, court, "chunk_00.bin"))
		if err != nil {
			t.Errorf("%s : %v", court, err)
			continue
		}
		got, ok := FilmMajorVersionFromHeader(filmsource.Inflate(raw))
		if !ok || got != want {
			t.Errorf("%s : version = (%d, %v), attendu (%d, true)", court, got, ok, want)
			continue
		}
		t.Logf("%s : FilmMajorVersion = %d", court, got)
	}
}
