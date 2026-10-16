package wire

// registry_analysis_pending_test.go — LA PORTE DE TITRE DE `analysis_pending`, rejouée sur les
// capabilities.toml LIVRÉS : Halo Infinite obtient un résolveur, Halo 5 aucun (le champ y reste
// faux, rien ne s'affiche). Aucune base : le dépôt est factice et ne doit jamais être lu par le
// test de porte.

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

type sondeNeutre struct{}

func (sondeNeutre) LoadAnalysisProbes(context.Context, []string) (map[string]domain.AnalysisProbe, error) {
	return nil, nil
}

func TestNewAnalysisPendingResolver_PorteDeTitre(t *testing.T) {
	cas := []struct {
		slug       string
		wantCable  bool
		wantReplay bool
	}{
		{"halo_infinite", true, true},
		{"halo_5", false, false},
	}
	for _, c := range cas {
		t.Run(c.slug, func(t *testing.T) {
			caps, err := games.LoadCapabilityMap(racineDepotDeCeTest(t), c.slug)
			if err != nil {
				t.Fatalf("LoadCapabilityMap(%s): %v", c.slug, err)
			}
			got := newAnalysisPendingResolver(caps, sondeNeutre{})
			if (got != nil) != c.wantCable {
				t.Fatalf("%s : résolveur câblé = %v, want %v", c.slug, got != nil, c.wantCable)
			}
			if c.wantReplay != caps.Has(games.CapFilmReplayArtifact) {
				t.Fatalf("%s : film.replay_artifact inattendu", c.slug)
			}
		})
	}
}

func TestNewAnalysisPendingResolver_CapabilitiesAbsentes(t *testing.T) {
	if newAnalysisPendingResolver(nil, sondeNeutre{}) != nil {
		t.Fatal("CapabilityMap nil : aucun résolveur")
	}
}
