package main

// carte_test.go — LA CLI RESOUT LA CARTE DU MATCH, OU REFUSE (2026-09-27).
//
// Regle utilisateur, fermee : « Le flux du film est la seule source fiable. Pas de repli. » La CLI
// decodait tout film sans carte, donc aux largeurs de Cliffhanger — la marche des morts s y
// desynchronise sur toute autre carte. Elle exige desormais `-carte <nom>` (un nom par film,
// separes par des virgules pour `comparer`) et le resout au catalogue de bornes du depot, par la
// meme voie que la production (`PathResolver.MapQuantBoundsPath` + `MapQuantCatalog.Lookup`).
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : accepter une option vide ; accepter un nom hors
// catalogue ; accepter moins de noms que de films.

import (
	"errors"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
)

func TestCarteObligatoire(t *testing.T) {
	if _, err := cartesDesFilms(options{}, 1); !errors.Is(err, errCarteObligatoire) {
		t.Fatalf("sans -carte : err = %v, attendu errCarteObligatoire — le film serait decode aux "+
			"largeurs d une autre carte", err)
	}
	// LA COMMANDE REFUSE AVANT DE LIRE LE FILM : le cache vide ne doit pas etre la raison du refus.
	if err := run([]string{"kills", "000d5950"}, options{cache: t.TempDir()}); !errors.Is(err, errCarteObligatoire) {
		t.Errorf("kills sans -carte : err = %v, attendu errCarteObligatoire", err)
	}
	if err := run([]string{"comparer", "000d5950", "fccc61cd"}, options{cache: t.TempDir()}); !errors.Is(err, errCarteObligatoire) {
		t.Errorf("comparer sans -carte : err = %v, attendu errCarteObligatoire", err)
	}
}

func TestCarteHorsCatalogueRefusee(t *testing.T) {
	_, err := cartesDesFilms(options{carte: "Carte Inexistante"}, 1)
	if !errors.Is(err, decfilm.ErrUnknownMapBounds) {
		t.Fatalf("carte hors catalogue : err = %v, attendu ErrUnknownMapBounds", err)
	}
}

func TestUneCarteParFilm(t *testing.T) {
	if _, err := cartesDesFilms(options{carte: "Bazaar"}, 2); err == nil {
		t.Error("comparer avec une seule carte pour deux films : accepte, attendu un refus")
	}
	cartes, err := cartesDesFilms(options{carte: "Bazaar, Launch Site"}, 2)
	if err != nil {
		t.Fatalf("deux cartes pour deux films : %v", err)
	}
	if len(cartes) != 2 || cartes[0].Module == cartes[1].Module {
		t.Fatalf("cartes = %+v, attendu Bazaar puis Launch Site", cartes)
	}
	for i, c := range cartes {
		if c.AxisWidths[0] == 0 || c.AxisWidths[1] == 0 || c.AxisWidths[2] == 0 {
			t.Errorf("carte %d sans largeurs : %+v", i, c)
		}
	}
}
