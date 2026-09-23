package replay

// filmfacts_entites_test.go — LES OCCUPANTS DU MATCH TRAVERSENT LE FICHIER DE FAITS A L IDENTIQUE
// (lot M2.2, 2026-09-23).
//
// `TestCodecCouvreFilmInputs` ne demande qu une chose d un champ : qu il ne se relise pas VIDE.
// Ce test-ci demande l EGALITE : chaque entite (slot, index, designateur, rangs, trous comptes,
// instabilite), les instants des images-cles porteuses et le temoin `Scanned` — y compris le cas
// « balaye, personne », qui ne doit pas se relire « non balaye ».

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

func TestFaitsTransportentLesEntitesDesJoueurs(t *testing.T) {
	entry := goldenEntryPourTest(t)
	cas := map[string]grammar.PlayerEntityScan{
		"temoin b1ad85eb": {
			Scanned:     true,
			KeyframesUS: []uint64{1_576_675_905, 1_576_695_905, 1_576_715_905, 1_576_736_005},
			Entities: []grammar.PlayerEntity{
				{Slot: 1297, Index: 0, Team: 0, FirstKF: 0, LastKF: 3, Seen: 3},
				{Slot: 1530, Index: 8, Team: 0, FirstKF: 0, LastKF: 1, Seen: 2},
				{Slot: 2145, Index: 8, Team: 1, FirstKF: 3, LastKF: 3, Seen: 1},
				{Slot: 9, Index: 2, Team: grammar.TeamNone, FirstKF: 2, LastKF: 2, Seen: 1, Unstable: true},
			},
		},
		"balaye, personne": {Scanned: true},
		"non balaye":       {},
	}
	for nom, scan := range cas {
		t.Run(nom, func(t *testing.T) {
			f := &FilmFactsFile{Facts: FilmFacts{Film: goldenFilm, MapModule: entry.Module,
				AxisW: entry.AxisWidths}}
			f.Facts.PlayerEntities = scan
			blob, err := EncodeFilmFactsFile(f)
			if err != nil {
				t.Fatalf("encodage : %v", err)
			}
			relu, err := DecodeFilmFactsFile(blob, entry)
			if err != nil {
				t.Fatalf("relecture : %v", err)
			}
			if !reflect.DeepEqual(relu.Facts.PlayerEntities, scan) {
				t.Fatalf("entites relues :\n %+v\n attendu :\n %+v", relu.Facts.PlayerEntities, scan)
			}
		})
	}
}
