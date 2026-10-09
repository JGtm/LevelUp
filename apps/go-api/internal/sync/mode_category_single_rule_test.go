package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestModeCategoryRuleIsSingle verrouille qu'il n'existe qu'UNE règle de catégorie de mode :
// analysis/modelabel (InferCategory). Un classifieur local par sous-chaînes avait divergé de
// celle de l'interface sur un tiers du registre (mesure du 2026-10-09). Le garde interdit sa
// résurrection dans le paquet sync : ni fonction de ce nom, ni constantes locales de catégorie.
func TestModeCategoryRuleIsSingle(t *testing.T) {
	interdits := []string{
		"func determineModeCategory",
		"modeCategoryRanked",
		"modeCategoryFirefight",
		"modeCategoryAssassin",
		`strings.Contains(lower, "fiesta")`,
	}
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != "." {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, motif := range interdits {
			if strings.Contains(string(data), motif) {
				t.Errorf("%s contient %q : la catégorie de mode se calcule par modelabel.InferCategory", path, motif)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestExtractRegistry_CategorieSuitLeNomDeLaPaire : la catégorie écrite à l'insertion est celle
// de la règle canonique pour le nom de paire de l'API.
func TestExtractRegistry_CategorieSuitLeNomDeLaPaire(t *testing.T) {
	for pair, want := range map[string]string{
		"Ranked:Strongholds on Live Fire":           "Ranked",
		"Community:Team Slayer on Dynasty":          "Assassin",
		"Super Fiesta:Slayer on Catalyst - Forge":   "Super Fiesta",
		"BTB:Slayer on Deadlock":                    "BTB",
		"Gruntpocalypse:Fiesta on Fathom Firefight": "Firefight",
	} {
		match := map[string]any{
			"MatchId": "m1",
			"MatchInfo": map[string]any{
				"StartTime":           "2026-10-01T10:00:00Z",
				"PlaylistMapModePair": map[string]any{"AssetId": "p1", "PublicName": pair},
			},
		}
		row, err := ExtractRegistry(match, "test")
		if err != nil {
			t.Fatalf("%s: %v", pair, err)
		}
		if row.ModeCategory != want {
			t.Errorf("%q: catégorie %q, attendu %q", pair, row.ModeCategory, want)
		}
	}
}
