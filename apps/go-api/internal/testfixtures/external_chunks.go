package testfixtures

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// ExternalFilmDataDir retourne le chemin du dataset 942 matchs si la
// variable d'env LEVELUP_TEST_FILM_DATA_DIR est definie, sinon "".
//
// Convention : la var pointe vers le dossier qui contient {match_short}/chunk_NN.bin
// (cf. C:/Users/Guillaume/Downloads/film_chunks sur la machine dev).
func ExternalFilmDataDir() string {
	return os.Getenv("LEVELUP_TEST_FILM_DATA_DIR")
}

// ExternalFilmManifestsDir retourne le chemin du dossier manifests externes
// (cf. C:/Users/Guillaume/Downloads/film_manifests) si LEVELUP_TEST_FILM_MANIFESTS_DIR
// est definie, sinon "".
func ExternalFilmManifestsDir() string {
	return os.Getenv("LEVELUP_TEST_FILM_MANIFESTS_DIR")
}

// LoadExternalManifest charge un manifest depuis le dossier externe.
//
// shortID = 8 premiers caracteres hex du match_id (sans tirets).
// Skip si LEVELUP_TEST_FILM_MANIFESTS_DIR n'est pas defini ou si le manifest
// n'existe pas.
func LoadExternalManifest(t *testing.T, shortID string) FilmManifest {
	t.Helper()
	dir := ExternalFilmManifestsDir()
	if dir == "" {
		t.Skip("LEVELUP_TEST_FILM_MANIFESTS_DIR non defini — skip test external dataset")
	}
	path := filepath.Join(dir, shortID+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("manifest %s absent : %v", path, err)
	}
	var m FilmManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("LoadExternalManifest(%s): decode: %v", shortID, err)
	}
	return m
}

// LoadExternalChunk charge un chunk depuis le dossier externe.
//
// shortID = 8 premiers caracteres hex du match_id.
// chunkIdx : index du chunk (correspond a chunk_NN.bin avec NN sur 2 digits).
// Skip si LEVELUP_TEST_FILM_DATA_DIR n'est pas defini ou si le chunk n'existe.
func LoadExternalChunk(t *testing.T, shortID string, chunkIdx int) []byte {
	t.Helper()
	dir := ExternalFilmDataDir()
	if dir == "" {
		t.Skip("LEVELUP_TEST_FILM_DATA_DIR non defini — skip test external dataset")
	}
	path := filepath.Join(dir, shortID, fmt.Sprintf("chunk_%02d.bin", chunkIdx))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("chunk %s absent : %v", path, err)
	}
	return data
}

// ListExternalMatches retourne la liste des shortIDs des matchs presents dans
// le dossier externe (sous-dossiers de LEVELUP_TEST_FILM_DATA_DIR). Skip si
// la variable n'est pas definie.
//
// Utile pour les tests "stress" qui itereraient sur tout le dataset.
func ListExternalMatches(t *testing.T) []string {
	t.Helper()
	dir := ExternalFilmDataDir()
	if dir == "" {
		t.Skip("LEVELUP_TEST_FILM_DATA_DIR non defini")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("read dir %s: %v", dir, err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}
