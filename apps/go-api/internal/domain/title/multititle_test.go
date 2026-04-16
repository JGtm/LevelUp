package title

import (
	"path/filepath"
	"testing"
)

// TestMultiTitle_PathIsolation vérifie que deux titres différents produisent
// des chemins complètement distincts — aucun croisement de données.
func TestMultiTitle_PathIsolation(t *testing.T) {
	r := NewRegistry()
	r.Register(&TitleDescriptor{
		Slug:     "halo_mcc",
		Name:     "Halo MCC",
		Provider: "halo_mcc",
		Status:   StatusComingSoon,
	})
	pr := NewPathResolver("/repo", r)

	cases := []struct {
		name   string
		pathHI string
		pathMC string
	}{
		{
			"SharedDB",
			pr.SharedDBPath("halo_infinite"),
			pr.SharedDBPath("halo_mcc"),
		},
		{
			"MetadataDB",
			pr.MetadataDBPath("halo_infinite"),
			pr.MetadataDBPath("halo_mcc"),
		},
		{
			"PlayerDB",
			pr.PlayerDBPath("halo_infinite", "TestPlayer"),
			pr.PlayerDBPath("halo_mcc", "TestPlayer"),
		},
		{
			"PlayerDir",
			pr.PlayerDir("halo_infinite", "TestPlayer"),
			pr.PlayerDir("halo_mcc", "TestPlayer"),
		},
		{
			"WarehouseDir",
			pr.WarehouseDir("halo_infinite"),
			pr.WarehouseDir("halo_mcc"),
		},
		{
			"BackupDir",
			pr.BackupDir("halo_infinite", "TestPlayer"),
			pr.BackupDir("halo_mcc", "TestPlayer"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.pathHI == tc.pathMC {
				t.Errorf("paths should differ between titles, both got %q", tc.pathHI)
			}
		})
	}
}

// TestMultiTitle_TitleDataDir_Structure vérifie le pattern d'arborescence :
// data/titles/{slug}/ pour chaque titre enregistré.
func TestMultiTitle_TitleDataDir_Structure(t *testing.T) {
	r := NewRegistry()
	r.Register(&TitleDescriptor{Slug: "halo_mcc", Name: "Halo MCC", Status: StatusComingSoon})
	pr := NewPathResolver("/repo", r)

	want := map[string]string{
		"halo_infinite": filepath.Join("/repo", "data", "titles", "halo_infinite"),
		"halo_mcc":      filepath.Join("/repo", "data", "titles", "halo_mcc"),
	}

	for slug, expected := range want {
		got := pr.TitleDataDir(slug)
		if got != expected {
			t.Errorf("TitleDataDir(%q): got %q, want %q", slug, got, expected)
		}
	}
}

// TestMultiTitle_SameGamertagDifferentTitles vérifie qu'un même gamertag
// dans deux titres différents aboutit à des chemins DB distincts.
func TestMultiTitle_SameGamertagDifferentTitles(t *testing.T) {
	r := NewRegistry()
	r.Register(&TitleDescriptor{Slug: "halo_mcc", Name: "Halo MCC", Status: StatusComingSoon})
	pr := NewPathResolver("/repo", r)

	gamertag := "SharedPlayer"

	hiPath := pr.PlayerDBPath("halo_infinite", gamertag)
	mccPath := pr.PlayerDBPath("halo_mcc", gamertag)

	if hiPath == mccPath {
		t.Fatalf("same gamertag in different titles should have different DB paths, both %q", hiPath)
	}

	// Chacun doit contenir son slug de titre
	if !containsSegment(hiPath, "halo_infinite") {
		t.Errorf("halo_infinite path should contain 'halo_infinite': %q", hiPath)
	}
	if !containsSegment(mccPath, "halo_mcc") {
		t.Errorf("halo_mcc path should contain 'halo_mcc': %q", mccPath)
	}
}

// TestMultiTitle_RegistryIsolation vérifie qu'un titre non enregistré
// ne produit pas de chemins qui croisent un titre existant.
func TestMultiTitle_RegistryIsolation(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)

	// Un titre non enregistré produit quand même un chemin (PathResolver ne valide pas).
	// Mais ValidateTitle doit le rejeter.
	if err := pr.ValidateTitle("halo_mcc"); err == nil {
		t.Error("expected error for unregistered title")
	}

	// Le chemin calculé ne doit pas intercepter halo_infinite
	unknownPath := pr.SharedDBPath("halo_mcc")
	hiPath := pr.SharedDBPath("halo_infinite")
	if unknownPath == hiPath {
		t.Error("unregistered title path should not collide with halo_infinite")
	}
}

func containsSegment(path, segment string) bool {
	// Vérifie que le segment est un composant du chemin (pas juste une sous-chaîne)
	for _, part := range filepath.SplitList(path) {
		if part == segment {
			return true
		}
	}
	// Fallback : vérification simple de sous-chaîne pour les chemins non-séparés par SplitList
	return len(path) > 0 && len(segment) > 0 &&
		filepath.Clean(path) != filepath.Clean(path) || // toujours false, placeholder
		containsSubstring(path, string(filepath.Separator)+segment+string(filepath.Separator))
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && searchSubstring(s, sub)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
