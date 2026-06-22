package api

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
)

// TestMetadataDBPathFor vérifie la redirection démo du chemin metadata.duckdb.
//
// Régression (Référentiel / Asset Drawer) : en démo, le handler asset-metadata
// lisait la coquille vide data/titles/.../metadata.duckdb au lieu des fixtures
// data/demo/warehouse/metadata.duckdb → listes maps/armes vides → aucune image.
// Le helper unifie cette redirection (déjà appliquée au RankCatalog) sur tous les
// sites de lecture de la metadata du titre.
func TestMetadataDBPathFor(t *testing.T) {
	const repoRoot = "/repo"
	const fixturesDir = "/fixtures"

	titlePath := titlePkg.NewPathResolver(repoRoot).MetadataDBPath(titlePkg.DefaultSlug)
	demoPath := filepath.Join(fixturesDir, "warehouse", "metadata.duckdb")

	tests := []struct {
		name string
		cfg  *config.AppConfig
		want string
	}{
		{
			name: "démo + fixtures dir → metadata des fixtures démo",
			cfg:  &config.AppConfig{RepoRoot: repoRoot, DemoMode: true, DemoFixturesDir: fixturesDir},
			want: demoPath,
		},
		{
			name: "hors démo → metadata du titre",
			cfg:  &config.AppConfig{RepoRoot: repoRoot, DemoMode: false, DemoFixturesDir: fixturesDir},
			want: titlePath,
		},
		{
			name: "démo sans fixtures dir → repli sur metadata du titre",
			cfg:  &config.AppConfig{RepoRoot: repoRoot, DemoMode: true, DemoFixturesDir: ""},
			want: titlePath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := metadataDBPathFor(tt.cfg); got != tt.want {
				t.Errorf("metadataDBPathFor() = %q, want %q", got, tt.want)
			}
		})
	}
}
