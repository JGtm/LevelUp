package halo_5

import (
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
)

// TestCapabilitiesTOMLMatchesHardcoded : garde-fou de parite. Le capabilities.toml
// versionne de Halo 5 doit reproduire EXACTEMENT la CapabilityMap codee dans
// fallbackCapabilities() (adapter_data.go). Toute divergence (ex. on flip une
// capability dans le TOML sans mettre a jour le fallback) casse ce test.
func TestCapabilitiesTOMLMatchesHardcoded(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
	tomlPath := filepath.Join(repoRoot, "config", "titles", "halo_5", "mappings", "capabilities.toml")

	set, err := mappings.LoadCapabilitiesFromFile(tomlPath)
	if err != nil {
		t.Fatalf("LoadCapabilitiesFromFile: %v", err)
	}
	fromTOML, err := games.CapabilityMapFromMappings(set)
	if err != nil {
		t.Fatalf("CapabilityMapFromMappings: %v", err)
	}

	// Factory non-nil : on veut la map STATIQUE (la degradation runtime "factory nil
	// -> tout not_exposed" est testee ailleurs ; ici on compare au TOML).
	hardcoded := NewDataAdapter(srcFactory(&fakeSource{}), nil).Capabilities()

	if len(fromTOML) != len(hardcoded) {
		t.Fatalf("nombre de capabilities: TOML=%d, hardcoded=%d", len(fromTOML), len(hardcoded))
	}
	for key, want := range hardcoded {
		got, ok := fromTOML[key]
		if !ok {
			t.Errorf("capability %q absente du TOML", key)
			continue
		}
		if got != want {
			t.Errorf("capability %q: TOML=%q, hardcoded=%q", key, got, want)
		}
	}
}
