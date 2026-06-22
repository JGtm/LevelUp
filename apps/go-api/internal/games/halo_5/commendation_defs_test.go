package halo_5

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/games/canonical"
)

// fakeCommDefs — CommendationDefSource de test : retourne err si posée, sinon les
// définitions connues (les IDs absents ne sont simplement pas dans la map).
type fakeCommDefs struct {
	defs map[string]canonical.CommendationDefinition
	err  error
}

func (f *fakeCommDefs) LookupCommendations(_ context.Context, ids []string) (map[string]canonical.CommendationDefinition, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[string]canonical.CommendationDefinition)
	for _, id := range ids {
		if d, ok := f.defs[id]; ok {
			out[id] = d
		}
	}
	return out, nil
}

// TestEnrichCommendations_PopulatesNameAndIcon : une définition connue peuple
// Name + IconURL ; un UUID inconnu reste brut (Name="", IconURL=nil).
func TestEnrichCommendations_PopulatesNameAndIcon(t *testing.T) {
	icon := "https://cdn.test/spartan-slayer.png"
	a := NewDataAdapter(nil, nil).WithCommendationDefs(&fakeCommDefs{
		defs: map[string]canonical.CommendationDefinition{
			"uuid-1": {ID: "uuid-1", Name: "Spartan Slayer", IconURL: icon},
		},
	})
	detail := &canonical.MatchDetail{Commendations: []canonical.Commendation{
		{ID: "uuid-1", Count: 3},
		{ID: "uuid-2", Count: 1}, // inconnu → reste brut
	}}
	a.enrichCommendations(context.Background(), detail)

	if got := detail.Commendations[0]; got.Name != "Spartan Slayer" || got.IconURL == nil || *got.IconURL != icon {
		t.Errorf("commendation[0] mal enrichie: %+v (icon=%v)", got, got.IconURL)
	}
	if got := detail.Commendations[1]; got.Name != "" || got.IconURL != nil {
		t.Errorf("commendation inconnue doit rester brute: %+v", got)
	}
}

// TestEnrichCommendations_NilSource_NoChange : sans source injectée, no-op.
func TestEnrichCommendations_NilSource_NoChange(t *testing.T) {
	a := NewDataAdapter(nil, nil) // pas de WithCommendationDefs
	detail := &canonical.MatchDetail{Commendations: []canonical.Commendation{{ID: "x", Count: 1}}}
	a.enrichCommendations(context.Background(), detail)
	if detail.Commendations[0].Name != "" || detail.Commendations[0].IconURL != nil {
		t.Error("sans source, les commendations doivent rester brutes")
	}
}

// TestEnrichCommendations_LookupError_LeavesRaw : une erreur de lookup laisse les
// commendations brutes (best-effort, pas de panique).
func TestEnrichCommendations_LookupError_LeavesRaw(t *testing.T) {
	a := NewDataAdapter(nil, nil).WithCommendationDefs(&fakeCommDefs{err: errors.New("boom")})
	detail := &canonical.MatchDetail{Commendations: []canonical.Commendation{{ID: "x", Count: 1}}}
	a.enrichCommendations(context.Background(), detail)
	if detail.Commendations[0].Name != "" {
		t.Error("erreur de lookup → commendations brutes")
	}
}

// TestEnrichCommendations_EmptyDetail_NoPanic : detail nil / commendations vides.
func TestEnrichCommendations_EmptyDetail_NoPanic(t *testing.T) {
	a := NewDataAdapter(nil, nil).WithCommendationDefs(&fakeCommDefs{})
	a.enrichCommendations(context.Background(), nil)
	a.enrichCommendations(context.Background(), &canonical.MatchDetail{})
}
