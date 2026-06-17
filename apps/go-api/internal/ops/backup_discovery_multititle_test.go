package ops

// backup_discovery_multititle_test.go — PMT-13 / MT-24 : preuve que la découverte
// de backup énumère les DBs de PLUSIEURS titres sans collision de clés (la
// rétention reste une enveloppe globale unique — cf. toPkgConfig).

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/title"
)

func TestDiscoverLevelUpDBs_MultiTitle(t *testing.T) {
	root := t.TempDir()
	pr := title.NewPathResolver(root)

	// Crée un shared_matches_v2.duckdb pour 2 titres distincts.
	for _, slug := range []string{"halo_infinite", "synthetic_test_title"} {
		p := pr.SharedDBPath(slug)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(p), err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}

	targets, err := discoverLevelUpDBs(pr)
	if err != nil {
		t.Fatalf("discoverLevelUpDBs: %v", err)
	}
	keys := make(map[string]bool, len(targets))
	for _, tg := range targets {
		keys[tg.Key] = true
	}

	// Les 2 titres coexistent, clés préfixées par slug → aucune collision.
	for _, want := range []string{"halo_infinite:shared_matches_v2", "synthetic_test_title:shared_matches_v2"} {
		if !keys[want] {
			t.Errorf("target %q absente — découverte backup non multi-titre (clés: %v)", want, keys)
		}
	}
}
