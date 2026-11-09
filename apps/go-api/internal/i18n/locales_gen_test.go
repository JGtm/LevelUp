package i18n

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// localesConfig est la forme de config/locales.toml.
type localesConfig struct {
	Default  string `toml:"default"`
	Fallback string `toml:"fallback"`
	Locale   []struct {
		Code     string   `toml:"code"`
		Intl     string   `toml:"intl"`
		Halo     []string `toml:"halo"`
		Required bool     `toml:"required"`
	} `toml:"locale"`
}

// TestGeneratedTableMatchesLocalesTOML relit config/locales.toml, source unique des langues, et
// vérifie que locales_gen.go lui correspond. Le miroir web est tenu par
// apps/web/src/lib/i18n/generatedLocalesFresh.guard.test.ts, qui compare son module généré au même
// fichier : les deux listes ne peuvent pas diverger sans qu'un des deux tests échoue.
// Réparation : node apps/web/scripts/build_i18n_manifests.mjs
func TestGeneratedTableMatchesLocalesTOML(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "config", "locales.toml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lecture de %s : %v", path, err)
	}
	var cfg localesConfig
	if err := toml.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("décodage de %s : %v", path, err)
	}
	if len(cfg.Locale) == 0 {
		t.Fatal("config/locales.toml sans entrée [[locale]]")
	}

	if string(Default) != cfg.Default {
		t.Errorf("Default = %q, locales.toml : %q", Default, cfg.Default)
	}
	if string(Fallback) != cfg.Fallback {
		t.Errorf("Fallback = %q, locales.toml : %q", Fallback, cfg.Fallback)
	}
	if len(table) != len(cfg.Locale) {
		t.Fatalf("table : %d langues, locales.toml : %d (relancer le générateur)", len(table), len(cfg.Locale))
	}
	for i, want := range cfg.Locale {
		got := table[i]
		if string(got.code) != want.Code || got.intl != want.Intl ||
			!reflect.DeepEqual(got.halo, want.Halo) || got.required != want.Required {
			t.Errorf("entrée %d : table %+v, locales.toml %+v (relancer le générateur)", i, got, want)
		}
	}
}
