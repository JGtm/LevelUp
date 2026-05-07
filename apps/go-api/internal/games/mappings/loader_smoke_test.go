package mappings

import (
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/games/canonical"
)

// TestLoadHaloInfiniteFieldsTOML est un smoke test sur le vrai
// config/titles/halo_infinite/mappings/fields.toml du repo. Il garantit que
// le fichier source de vérité est valide à tout moment.
func TestLoadHaloInfiniteFieldsTOML(t *testing.T) {
	t.Parallel()
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
	tomlPath := filepath.Join(repoRoot, "config", "titles", "halo_infinite", "mappings", "fields.toml")

	set, err := LoadFieldsFromFile(tomlPath)
	if err != nil {
		t.Fatalf("LoadFieldsFromFile(%s): %v", tomlPath, err)
	}

	if set.TitleSlug() != "halo_infinite" {
		t.Errorf("TitleSlug = %q, want halo_infinite", set.TitleSlug())
	}
	if set.SchemaVersion() != 1 {
		t.Errorf("SchemaVersion = %d, want 1", set.SchemaVersion())
	}

	// Tous les FieldKey du canonique doivent être couverts par le titre par défaut.
	for _, k := range canonical.AllFieldKeys() {
		if _, ok := set.Get(k); !ok {
			t.Errorf("FieldKey %q absent du fields.toml HI", k)
		}
	}

	// Spot check : kills doit avoir les libellés FR/EN attendus.
	kills, ok := set.Get(canonical.FieldKills)
	if !ok {
		t.Fatalf("FieldKills introuvable")
	}
	labelFR, fallback := kills.Label("fr")
	if fallback || labelFR != "Frags" {
		t.Errorf("kills FR = %q (fallback=%v), want Frags", labelFR, fallback)
	}
	labelEN, fallback := kills.Label("en")
	if fallback || labelEN != "Kills" {
		t.Errorf("kills EN = %q (fallback=%v), want Kills", labelEN, fallback)
	}

	// Locale inconnue → fallback EN.
	_, fallback = kills.Label("xx")
	if !fallback {
		t.Errorf("locale inconnue devrait déclencher fallback")
	}

	// Conversion d'unité accuracy : ratio storage → percent display.
	acc, ok := set.Get(canonical.FieldAccuracy)
	if !ok {
		t.Fatalf("FieldAccuracy introuvable")
	}
	if acc.StorageUnit != UnitRatio || acc.DisplayUnit != UnitPercent {
		t.Errorf("accuracy units: storage=%s display=%s, want ratio→percent", acc.StorageUnit, acc.DisplayUnit)
	}
	out, ok := ConvertValue(0.42, acc.StorageUnit, acc.DisplayUnit)
	if !ok {
		t.Errorf("ConvertValue ratio→percent ok=false")
	}
	if out < 41.99 || out > 42.01 {
		t.Errorf("0.42 ratio = %.2f percent, want ≈42", out)
	}
}
