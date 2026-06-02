package halo_infinite

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
)

// careerStub : CareerSource non-nil → career.progression non rétrogradée →
// CapSupported (le TOML déclare career.progression=supported, intention max).
type careerStub struct{}

func (careerStub) GetLatestRank(context.Context) (*domain.CareerRankData, error) { return nil, nil }
func (careerStub) GetEncounters(context.Context) ([]domain.EncounterRawRow, error) {
	return nil, nil
}

// TestCapabilitiesTOMLMatchesHardcoded est le garde-fou de la Phase 1.7a : le
// capabilities.toml versionné doit reproduire EXACTEMENT la CapabilityMap codée
// en dur dans adapter_data.go (career non-nil → supported). Cette parité prouve
// que le TOML est une extraction fidèle et rend sûr le câblage ultérieur de
// l'adapter sur le TOML (swap sans changement de comportement).
func TestCapabilitiesTOMLMatchesHardcoded(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
	tomlPath := filepath.Join(repoRoot, "config", "titles", "halo_infinite", "mappings", "capabilities.toml")

	set, err := mappings.LoadCapabilitiesFromFile(tomlPath)
	if err != nil {
		t.Fatalf("LoadCapabilitiesFromFile: %v", err)
	}
	fromTOML, err := games.CapabilityMapFromMappings(set)
	if err != nil {
		t.Fatalf("CapabilityMapFromMappings: %v", err)
	}

	hardcoded := NewDataAdapter(careerStub{}, nil).Capabilities()

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
