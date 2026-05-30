package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

type stubMedalDefs struct {
	defs map[int64]port.MedalDefinitionRow
	err  error
}

func (s stubMedalDefs) LookupByIDs(_ context.Context, _ []int64, _ string) (map[int64]port.MedalDefinitionRow, error) {
	return s.defs, s.err
}

func TestBuildTargetTopMedals_SortAndEnrich(t *testing.T) {
	repo := stubMedalDefs{defs: map[int64]port.MedalDefinitionRow{
		10: {Label: "Double Kill", Description: "2 kills", Difficulty: "Normal", MedalType: "multikill"},
		20: {Label: "Killing Spree", Description: "5 kills", Difficulty: "Heroic", MedalType: "spree"},
	}}
	medals := []domain.RemoteMedalCount{
		{NameID: 10, Count: 3},
		{NameID: 20, Count: 50},
	}

	got := buildTargetTopMedals(context.Background(), repo, medals, "halo_infinite", "fr")
	if len(got) != 2 {
		t.Fatalf("attendu 2 médailles, got %d", len(got))
	}
	// Tri par count desc : Killing Spree (50) avant Double Kill (3).
	if got[0].MedalID != 20 || got[0].TotalCount != 50 {
		t.Errorf("attendu medal 20 (count 50) en tête, got %+v", got[0])
	}
	if got[0].Label != "Killing Spree" || got[0].Difficulty != "Heroic" {
		t.Errorf("enrichissement label/difficulty incorrect: %+v", got[0])
	}
	if got[0].ImageURL != "/static/medals/halo_infinite/20.png" {
		t.Errorf("image URL attendue /static/medals/halo_infinite/20.png, got %q", got[0].ImageURL)
	}
}

func TestBuildTargetTopMedals_CapAt20(t *testing.T) {
	medals := make([]domain.RemoteMedalCount, 0, 30)
	for i := 1; i <= 30; i++ {
		medals = append(medals, domain.RemoteMedalCount{NameID: int64(i), Count: i})
	}
	got := buildTargetTopMedals(context.Background(), stubMedalDefs{}, medals, "halo_infinite", "fr")
	if len(got) != explorerTopMedalsCap {
		t.Fatalf("attendu cap %d, got %d", explorerTopMedalsCap, len(got))
	}
	// Le plus gros count (30) doit être en tête.
	if got[0].TotalCount != 30 {
		t.Errorf("attendu top count 30, got %d", got[0].TotalCount)
	}
}

func TestBuildTargetTopMedals_LookupError(t *testing.T) {
	// LookupByIDs échoue → dégradation gracieuse : médailles rendues avec
	// image + compteur, mais label/description vides.
	repo := stubMedalDefs{err: errors.New("metadata db down")}
	medals := []domain.RemoteMedalCount{{NameID: 42, Count: 9}}

	got := buildTargetTopMedals(context.Background(), repo, medals, "halo_infinite", "fr")
	if len(got) != 1 {
		t.Fatalf("attendu 1 médaille malgré l'erreur lookup, got %d", len(got))
	}
	m := got[0]
	if m.MedalID != 42 || m.TotalCount != 9 {
		t.Errorf("compteur/id attendus (42, 9), got (%d, %d)", m.MedalID, m.TotalCount)
	}
	if m.ImageURL != "/static/medals/halo_infinite/42.png" {
		t.Errorf("image URL attendue malgré l'erreur, got %q", m.ImageURL)
	}
	if m.Label != "" || m.Description != "" {
		t.Errorf("label/description attendus vides en cas d'erreur lookup, got label=%q desc=%q", m.Label, m.Description)
	}
}

func TestBuildTargetTopMedals_NilSafe(t *testing.T) {
	if got := buildTargetTopMedals(context.Background(), nil, []domain.RemoteMedalCount{{NameID: 1, Count: 1}}, "t", "fr"); got != nil {
		t.Errorf("repo nil → nil attendu, got %v", got)
	}
	if got := buildTargetTopMedals(context.Background(), stubMedalDefs{}, nil, "t", "fr"); got != nil {
		t.Errorf("médailles vides → nil attendu, got %v", got)
	}
}
