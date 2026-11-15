package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/JGtm/halo-infinite-map-images/himap"
)

// Une carte Forge sans blob au dépôt est laissée intacte : son sidecar n'est pas réécrit.
func TestAjoutePlansLaisseIntacteUneCarteSansBlob(t *testing.T) {
	if len(himap.CartesForge) == 0 {
		t.Skip("aucune carte Forge declaree")
	}
	carte := himap.CartesForge[0]
	sortie := t.TempDir()
	contenu := []byte(`{"schemaVersion":1,"module":"x","image":"x.webp","source":"t","generatedAt":"2026-08-12T23:00:51.9204482Z","style":"encre","calibration":{"metersPerPixel":1,"originX":0,"originY":0,"widthPx":1,"heightPx":1,"convention":"c"},"stats":{}}`)
	chemin := filepath.Join(sortie, carte.MapID+".json")
	if err := os.WriteFile(chemin, contenu, 0o644); err != nil {
		t.Fatal(err)
	}
	e := &environnement{repoRoot: t.TempDir(), sortieDir: sortie, filtre: map[string]bool{carte.MapID: true}, sansNatives: true}

	b := e.ajoutePlans(context.Background())
	if b.Ajoutes != 0 || b.Echecs != 0 || len(b.SansBlob) != 1 {
		t.Fatalf("bilan = %+v, attendu une carte sans blob et rien d'autre", b)
	}
	apres, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}
	if string(apres) != string(contenu) {
		t.Errorf("le sidecar d'une carte sans blob a ete reecrit :\n%s", apres)
	}
}
