package title

import (
	"path/filepath"
	"testing"
)

// Les chemins que produisent les deux gabarits de base joueur (PathResolver et DemoLayout) sont
// reconnus ; les autres bases d'un titre et les formes voisines ne le sont pas.
func TestIsPlayerDBPath(t *testing.T) {
	root := t.TempDir()
	pr := NewPathResolver(root)
	demo := NewDemoLayout(filepath.Join(root, "demo"))
	oui := []string{
		pr.PlayerDBPath(DefaultSlug, "Chocoboflor"),
		pr.PlayerDBPath("halo_5", "Joueur Avec Espace"),
		pr.PlayerDBPath(DefaultSlug, "players"), // gamertag homonyme du répertoire
		demo.PlayerDBPath(DefaultSlug, "DEMO"),
		filepath.Join("data", "titles", DefaultSlug, "players", "GT", "stats.duckdb"),
	}
	for _, p := range oui {
		if !IsPlayerDBPath(p) {
			t.Errorf("IsPlayerDBPath(%q) = false, attendu true", p)
		}
	}
	non := []string{
		pr.SharedDBPath(DefaultSlug),
		pr.MetadataDBPath(DefaultSlug),
		pr.SharedSocialDBPath(DefaultSlug),
		pr.SharedPVEDBPath(DefaultSlug),
		filepath.Join(pr.PlayersRootDir(DefaultSlug), "stats.duckdb"), // .../players/stats.duckdb : pas de dossier de joueur
		filepath.Join(pr.PlayerDir(DefaultSlug, "GT"), "archive", "stats.duckdb"),
		filepath.Join(pr.PlayerDir(DefaultSlug, "GT"), "autre.duckdb"),
		filepath.Join(root, "stats.duckdb"),
		"stats.duckdb",
		"",
	}
	for _, p := range non {
		if IsPlayerDBPath(p) {
			t.Errorf("IsPlayerDBPath(%q) = true, attendu false", p)
		}
	}
}
