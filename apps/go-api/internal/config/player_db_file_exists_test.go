package config

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/title"
)

// PlayerDBFileExists distingue le profil jamais synchronisé (aucun dossier, aucune base) de celui
// dont la base est là, sans jamais rien créer ; en démo elle rend true (les fixtures disent leur
// propre absence).
func TestPlayerDBFileExists(t *testing.T) {
	t.Parallel()
	racine := t.TempDir()
	cfg := &AppConfig{RepoRoot: racine}
	slug := title.DefaultSlug

	if PlayerDBFileExists(cfg, slug, "Jamais") {
		t.Error("profil jamais synchronisé : la base n'existe pas")
	}
	if _, err := os.Stat(filepath.Dir(PlayerDBPath(cfg, slug, "Jamais"))); err == nil {
		t.Error("la vérification ne doit rien créer")
	}

	chemin := PlayerDBPath(cfg, slug, "Synchro")
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		t.Fatal(err)
	}
	if PlayerDBFileExists(cfg, slug, "Synchro") {
		t.Error("dossier sans base : la base n'existe pas")
	}
	if err := os.WriteFile(chemin, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !PlayerDBFileExists(cfg, slug, "Synchro") {
		t.Error("base présente : doit rendre true")
	}

	demo := &AppConfig{RepoRoot: racine, DemoMode: true}
	if !PlayerDBFileExists(demo, slug, "Jamais") {
		t.Error("en démo : true (ResolvePlayer lit les fixtures et dit leur absence)")
	}
}
