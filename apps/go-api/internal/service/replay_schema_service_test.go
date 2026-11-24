package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

type censusFactice struct {
	counts map[string]domain.AdminReplaySchemaTitle
	failOn string
}

func (censusFactice) CurrentSchema() int { return 127 }

func (c censusFactice) CountReplaySchemas(_ context.Context, slug string) (domain.AdminReplaySchemaTitle, error) {
	if slug == c.failOn {
		return domain.AdminReplaySchemaTitle{}, errors.New("lecture impossible")
	}
	return c.counts[slug], nil
}

// TestReplaySchema_UnComptePartTitreDansLOrdreDesTitres : un décompte par titre qui range des rejeux, dans l'ordre
// que la capacité donne, avec le schéma courant.
func TestReplaySchema_UnComptePartTitreDansLOrdreDesTitres(t *testing.T) {
	census := censusFactice{counts: map[string]domain.AdminReplaySchemaTitle{
		"a": {TitleSlug: "a", Total: 3, Current: 1, Behind: 2},
		"b": {TitleSlug: "b", Total: 1, Unreadable: 1},
	}}
	svc := NewReplaySchemaService(ReplaySchemaDeps{Census: census, Titles: func() []string { return []string{"a", "b"} }})
	got, err := svc.ReplaySchema(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.CurrentSchema != 127 || len(got.Titles) != 2 || got.Titles[0].Behind != 2 || got.Titles[1].Unreadable != 1 {
		t.Fatalf("réponse = %+v", got)
	}
	if got.GeneratedAt == "" {
		t.Fatal("la réponse n'est pas datée")
	}
}

// TestReplaySchema_SansTitreLaListeEstVide : aucun titre ne range de rejeux, la liste est vide et non nulle (le
// contrat la sert comme `[]`).
func TestReplaySchema_SansTitreLaListeEstVide(t *testing.T) {
	svc := NewReplaySchemaService(ReplaySchemaDeps{Census: censusFactice{}, Titles: func() []string { return nil }})
	got, err := svc.ReplaySchema(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Titles == nil || len(got.Titles) != 0 {
		t.Fatalf("titres = %#v, attendu une liste vide non nulle", got.Titles)
	}
}

// TestReplaySchema_UneErreurNommeLeTitre : un titre dont le décompte échoue interrompt la réponse et se nomme.
func TestReplaySchema_UneErreurNommeLeTitre(t *testing.T) {
	svc := NewReplaySchemaService(ReplaySchemaDeps{
		Census: censusFactice{failOn: "b"}, Titles: func() []string { return []string{"a", "b"} },
	})
	_, err := svc.ReplaySchema(context.Background())
	if err == nil {
		t.Fatal("l'échec d'un titre doit remonter")
	}
}
