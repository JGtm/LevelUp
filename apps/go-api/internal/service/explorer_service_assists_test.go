package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

// assistsFixture : trois relations mesurées, dont la cible. Le plus gros volume d'un
// sens est 120 (données de « mate-big ») — c'est la borne d'échelle attendue, PAS le
// volume de la cible.
func assistsFixture() map[string]domain.RelationAssists {
	return map[string]domain.RelationAssists{
		"target-x": {
			MatchesMeasured: 9,
			MyFrags:         80,
			PartnerFrags:    70,
			Received:        domain.AssistTiers{Total: 20, Low: 12, Mid: 5, High: 3},
			Given:           domain.AssistTiers{Total: 14, Low: 9, Mid: 3, High: 2},
		},
		"mate-big": {
			MatchesMeasured: 40,
			MyFrags:         300,
			PartnerFrags:    280,
			Received:        domain.AssistTiers{Total: 95, Low: 60, Mid: 25, High: 10},
			Given:           domain.AssistTiers{Total: 120, Low: 70, Mid: 35, High: 15},
		},
		"mate-small": {
			MatchesMeasured: 2,
			MyFrags:         10,
			PartnerFrags:    12,
			Received:        domain.AssistTiers{Total: 3, Low: 3},
			Given:           domain.AssistTiers{Total: 1, Low: 1},
		},
	}
}

// TestEnrichEncounterAssists : la cible présente dans la map alimente Assists, et la
// borne d'échelle est le max GLOBAL (toutes relations), pas celui de la paire.
func TestEnrichEncounterAssists(t *testing.T) {
	t.Parallel()
	prov := &mockExplorerRelations{assists: assistsFixture()}
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{Relations: prov})

	stats := &domain.ExplorerEncounterStats{CountTogether: 9}
	svc.enrichEncounterAssists(context.Background(), stats, "target-x")

	if stats.Assists == nil {
		t.Fatal("Assists nil, want la paire target-x")
	}
	if stats.Assists.Received.Total != 20 || stats.Assists.Given.Total != 14 {
		t.Errorf("volumes = %d/%d, want 20/14", stats.Assists.Received.Total, stats.Assists.Given.Total)
	}
	if stats.Assists.MatchesMeasured != 9 || stats.Assists.MyFrags != 80 || stats.Assists.PartnerFrags != 70 {
		t.Errorf("dénominateurs = %d/%d/%d, want 9/80/70",
			stats.Assists.MatchesMeasured, stats.Assists.MyFrags, stats.Assists.PartnerFrags)
	}
	if stats.AssistVolumeMax != 120 {
		t.Errorf("AssistVolumeMax = %d, want 120 (max global, pas celui de la paire)", stats.AssistVolumeMax)
	}
}

// TestEnrichEncounterAssists_TargetNotMeasured : aucun match ensemble avec film décodé
// → pas d'objet Assists du tout (le front affiche « — », jamais « 0 assistance »). La
// borne reste servie : les autres relations sont mesurées.
func TestEnrichEncounterAssists_TargetNotMeasured(t *testing.T) {
	t.Parallel()
	fixture := assistsFixture()
	delete(fixture, "target-x")
	prov := &mockExplorerRelations{assists: fixture}
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{Relations: prov})

	stats := &domain.ExplorerEncounterStats{CountTogether: 9}
	svc.enrichEncounterAssists(context.Background(), stats, "target-x")

	if stats.Assists != nil {
		t.Errorf("Assists = %+v, want nil (cible jamais mesurée)", stats.Assists)
	}
	if stats.AssistVolumeMax != 120 {
		t.Errorf("AssistVolumeMax = %d, want 120", stats.AssistVolumeMax)
	}
}

// TestEnrichEncounterAssists_Degraded : repo en erreur, map vide, xuid cible absent ou
// stats nil → aucune mutation, aucune panique. Le reste de la réponse est servi.
func TestEnrichEncounterAssists_Degraded(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		prov   *mockExplorerRelations
		target string
	}{
		{"repo en erreur", &mockExplorerRelations{assistsErr: errors.New("boom")}, "target-x"},
		{"aucune relation mesurée", &mockExplorerRelations{assists: map[string]domain.RelationAssists{}}, "target-x"},
		{"xuid cible vide", &mockExplorerRelations{assists: assistsFixture()}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := NewExplorerService(&mockExplorerRepo{}, "self").
				WithTargetProfileProviders(ExplorerTargetProfileDeps{Relations: tc.prov})
			stats := &domain.ExplorerEncounterStats{CountTogether: 1}
			svc.enrichEncounterAssists(context.Background(), stats, tc.target)
			if stats.Assists != nil || stats.AssistVolumeMax != 0 {
				t.Errorf("mutation inattendue : assists=%v volumeMax=%d", stats.Assists, stats.AssistVolumeMax)
			}
		})
	}
	// stats nil → no-op (pas de panic).
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{Relations: &mockExplorerRelations{}})
	svc.enrichEncounterAssists(context.Background(), nil, "target-x")
}

// TestExplorerAssistVolumeMax : le max porte sur les DEUX sens, map vide → 0.
func TestExplorerAssistVolumeMax(t *testing.T) {
	t.Parallel()
	if got := explorerAssistVolumeMax(nil); got != 0 {
		t.Errorf("map nil → %d, want 0", got)
	}
	if got := explorerAssistVolumeMax(assistsFixture()); got != 120 {
		t.Errorf("max = %d, want 120", got)
	}
	received := map[string]domain.RelationAssists{
		"a": {Received: domain.AssistTiers{Total: 77}, Given: domain.AssistTiers{Total: 5}},
	}
	if got := explorerAssistVolumeMax(received); got != 77 {
		t.Errorf("max côté reçues = %d, want 77", got)
	}
}
