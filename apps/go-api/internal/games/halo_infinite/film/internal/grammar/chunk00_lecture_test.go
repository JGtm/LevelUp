package grammar

// chunk00_lecture_test.go — lecture de chunk_00.bin PARTAGEE entre les instruments `research`
// (`chunk00_*_research_test.go`, `e191*`) et les gardes qui tournent dans le build par defaut
// (`film_format_version_test.go`, `mpp_resolution_test.go`). Extraite de
// `chunk00_carte_research_test.go` au J12.7 : sans ce fichier non tague, le tag `research` de
// l'instrument aurait sorti ces deux gardes du build par defaut.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// readChunk00 lit et decompresse chunk_00.bin d'un repertoire de film.
func readChunk00(t *testing.T, dir string) (raw, data []byte) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "chunk_00.bin"))
	if err != nil {
		t.Fatalf("lecture chunk_00 de %s : %v", dir, err)
	}
	return b, source.Inflate(b)
}
