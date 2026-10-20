package replaylabels

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// copieDesMappings recopie les deux TOML du titre sous une racine temporaire.
func copieDesMappings(t *testing.T, racine string) {
	t.Helper()
	src := filepath.Join(repoRoot(t), "config", "titles", "halo_infinite", "mappings")
	dst := filepath.Join(racine, "config", "titles", "halo_infinite", "mappings")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"weapon_names.toml", "replay_labels.toml"} {
		b, err := os.ReadFile(filepath.Join(src, f))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, f), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestCatalogue_LuUneFoisParProcessus : le second appel ne relit pas les fichiers (retirés
// entre-temps) et rend le même catalogue que Load.
func TestCatalogue_LuUneFoisParProcessus(t *testing.T) {
	racine := t.TempDir()
	copieDesMappings(t, racine)
	attendu, err := Load(racine, "halo_infinite")
	if err != nil {
		t.Fatalf("Load : %v", err)
	}
	premier, err := Catalogue(racine, "halo_infinite")
	if err != nil {
		t.Fatalf("Catalogue : %v", err)
	}
	if !reflect.DeepEqual(premier, attendu) {
		t.Fatal("Catalogue doit rendre exactement le catalogue de Load")
	}
	if err := os.RemoveAll(filepath.Join(racine, "config")); err != nil {
		t.Fatal(err)
	}
	second, err := Catalogue(racine, "halo_infinite")
	if err != nil {
		t.Fatalf("second appel : les fichiers ont été relus (%v)", err)
	}
	if !reflect.DeepEqual(second, attendu) {
		t.Fatal("le second appel doit servir le catalogue mémorisé")
	}
}

// TestCatalogue_EchecNonMemorise : une racine sans mappings échoue, puis réussit dès que les
// fichiers existent — l'échec n'est pas figé pour la vie du processus.
func TestCatalogue_EchecNonMemorise(t *testing.T) {
	racine := t.TempDir()
	if _, err := Catalogue(racine, "halo_infinite"); err == nil {
		t.Fatal("racine sans mappings : erreur attendue")
	}
	copieDesMappings(t, racine)
	if _, err := Catalogue(racine, "halo_infinite"); err != nil {
		t.Fatalf("mappings posés : %v", err)
	}
}
