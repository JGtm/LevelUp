package halo_infinite

import (
	"testing"

	"levelup/go-api/internal/games"
)

// TestCapabilities_UsesInjected vérifie que WithCapabilities prime sur le
// fallback codé (chemin nominal Phase 1.7a : la map TOML est la source).
func TestCapabilities_UsesInjected(t *testing.T) {
	injected := games.CapabilityMap{
		games.CapMatchHistory: games.CapSupported,
		games.CapTimeseries:   games.CapSupported, // diffère du fallback (not_exposed)
	}
	a := NewDataAdapter(careerStub{}, nil).WithCapabilities(injected)

	caps := a.Capabilities()
	if len(caps) != 2 {
		t.Fatalf("Capabilities() = %d entrées, want 2 (la map injectée, pas le fallback)", len(caps))
	}
	if !caps.Has(games.CapTimeseries) {
		t.Errorf("timeseries doit être supported (injecté), got %q", caps[games.CapTimeseries])
	}
}

// TestCapabilities_CareerDowngrade vérifie la rétrogradation runtime de
// career.progression quand aucune source carrière n'est câblée, sans jamais
// forcer au-dessus de l'intention déclarée.
func TestCapabilities_CareerDowngrade(t *testing.T) {
	caps := games.CapabilityMap{games.CapCareerProgression: games.CapSupported}

	// career == nil → not_exposed
	got := NewDataAdapter(nil, nil).WithCapabilities(caps).Capabilities()
	if got[games.CapCareerProgression] != games.CapNotExposed {
		t.Errorf("career sans source = %q, want not_exposed", got[games.CapCareerProgression])
	}

	// career != nil → reste supported
	got = NewDataAdapter(careerStub{}, nil).WithCapabilities(caps).Capabilities()
	if got[games.CapCareerProgression] != games.CapSupported {
		t.Errorf("career avec source = %q, want supported", got[games.CapCareerProgression])
	}
}
