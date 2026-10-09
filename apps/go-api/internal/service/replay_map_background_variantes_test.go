package service

// replay_map_background_variantes_test.go — LES CARTES JOUÉES SOUS UN ASSET SANS FOND PROPRE.
//
// Quatre cartes jouées depuis le gel de l'inventaire (`map_ids_joues_20260827.json`) n'ont pas de
// fond publié sous LEUR map_id : « Argyle - Ranked », « Serenity - Ranked » et « Vacancy - Ranked »
// sont des assets distincts de leur base, dont seule la base a un fond. « Interference » a le sien
// sous son propre map_id. La chaîne RÉELLE du service doit rendre, pour chacune, le fond de la
// carte qu'elle est : la clé map_id quand le fond existe, sinon l'héritage variante -> base de
// l'index des fonds (`replay.MapBackgroundIndex.Lookup`).
//
// Les identités passées sont celles que `ReplayMapRepo` rend pour ces matchs : le map_id joué et
// le nom affiché du registre.

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/testutil"
)

func TestFondsDesCartesJoueesSousUnAssetSansFondPropre(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	cas := []struct {
		nom, mapIDJoue, cleAttendue string
	}{
		// Héritage variante -> base : l'asset classé n'a pas de fond, sa base en a un.
		{"Argyle - Ranked", "78677080-db7a-429e-84fe-f041d8342c37", "dd600260-d91c-4d77-9990-3f35873c90a1"},
		{"Serenity - Ranked", "1de0bf60-e446-4fb9-970f-d0e54fc6c74a", "b4d13418-c0b5-47dc-9515-931dfda77d9f"},
		{"Vacancy - Ranked", "6a1e8432-88ae-4430-8f7d-9ffefc97cc8d", "4fb5b69f-5104-450b-9ed0-a232f997e8f9"},
		// Clé map_id : le fond est publié sous l'asset joué.
		{"Interference", "654dff62-d618-496a-8914-06ab73d991e3", "654dff62-d618-496a-8914-06ab73d991e3"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			svc := NewReplayService(title.DefaultSlug, root,
				&mapNamesStub{mapID: c.mapIDJoue, names: []string{c.nom}})
			bg, err := svc.MapBackground(context.Background(), "m")
			if err != nil {
				t.Fatalf("aucun fond pour %s (%s) : %v", c.nom, c.mapIDJoue, err)
			}
			if bg.Module != c.cleAttendue {
				t.Errorf("fond de %s = %q, veut %q", c.nom, bg.Module, c.cleAttendue)
			}
		})
	}
}
