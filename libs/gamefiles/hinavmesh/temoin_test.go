package hinavmesh

import (
	"os"
	"path/filepath"
	"testing"
)

// decodeTemoin decode le navmesh verse en testdata pour un asset.
func decodeTemoin(t *testing.T, assetID string) *Maillage {
	t.Helper()
	chemin := filepath.Join("testdata", assetID+".navmesh.blob")
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin de test, versionne
	if err != nil {
		t.Fatalf("lecture de %s: %v", chemin, err)
	}
	m, err := Decode(blob)
	if err != nil {
		t.Fatalf("Decode(%s): %v", chemin, err)
	}
	return m
}
