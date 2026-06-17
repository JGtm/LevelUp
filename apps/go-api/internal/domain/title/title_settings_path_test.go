package title

import (
	"path/filepath"
	"testing"
)

// TestPathResolver_TitleSettingsPath (PMT-4) : l'overlay de settings est
// namespacé par titre (data/titles/<slug>/settings.json) ; slug vide → défaut ;
// titres distincts → chemins distincts (isolation).
func TestPathResolver_TitleSettingsPath(t *testing.T) {
	pr := NewPathResolver("/repo")

	wantHalo := filepath.Join("/repo", "data", "titles", "halo_infinite", "settings.json")
	if got := pr.TitleSettingsPath("halo_infinite"); got != wantHalo {
		t.Errorf("TitleSettingsPath(halo_infinite) = %q, want %q", got, wantHalo)
	}
	if got := pr.TitleSettingsPath(""); got != wantHalo {
		t.Errorf("TitleSettingsPath(\"\") = %q, want %q (fallback défaut)", got, wantHalo)
	}

	wantOther := filepath.Join("/repo", "data", "titles", "synthetic_title_b", "settings.json")
	if got := pr.TitleSettingsPath("synthetic_title_b"); got != wantOther {
		t.Errorf("TitleSettingsPath(synthetic_title_b) = %q, want %q", got, wantOther)
	}
	if pr.TitleSettingsPath("synthetic_title_b") == pr.TitleSettingsPath("halo_infinite") {
		t.Error("chemins d'overlay non isolés entre titres")
	}
}
