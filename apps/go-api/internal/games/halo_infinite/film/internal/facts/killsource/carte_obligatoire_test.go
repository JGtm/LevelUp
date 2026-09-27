package killsource

// carte_obligatoire_test.go — SANS CARTE, PAS DE DECODAGE (2026-09-27).
//
// Regle utilisateur, fermee : « Le flux du film est la seule source fiable. Pas de repli. »
// Decoder un film aux largeurs d une AUTRE carte est un repli : la marche des morts s y
// desynchronise et le scan publie a sa place (enquete ENQUETE_MARCHE_KILLSOURCE_2026-09-27).
// [Decode] sans entree de catalogue rend donc [ErrCarteAbsente] ; seul un instrument de recherche
// peut le demander, et il le NOMME ([Options.RechercheSansCarte]), jamais par un nil.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : retirer la garde de [Decode] (le decodage aboutit aux
// largeurs par defaut) ; la faire ignorer une entree sans largeurs ; la faire ignorer l option de
// recherche (l instrument ne decode plus).

import (
	"errors"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// miniBobineCarte : la carte du film dont la bobine est tiree (000d5950, Cliffhanger — enquete du
// 2026-09-27, table film -> carte).
const miniBobineCarte = "Cliffhanger"

// optionsDeLaBobine : la configuration gelee sous la carte de la bobine — ce que tout decodage de
// la bobine doit passer depuis que la carte est obligatoire.
func optionsDeLaBobine(t *testing.T) *Options {
	t.Helper()
	carte := carteDuCatalogue(t, miniBobineCarte)
	opts := DefaultOptions()
	opts.Carte = &carte
	return &opts
}

func TestDecodeSansCarteMetLeFilmDeCote(t *testing.T) {
	src, err := source.LoadDir(miniBobineDir, nil)
	if err != nil {
		t.Fatalf("mini-bobine illisible : %v", err)
	}
	sansLargeurs := profile.MapQuantEntry{Module: "entree fabriquee a la main"}
	defauts := DefaultOptions()
	avecEntreeVide := DefaultOptions()
	avecEntreeVide.Carte = &sansLargeurs
	for _, cas := range []struct {
		nom  string
		opts *Options
	}{
		{"options nil (configuration gelee, sans carte)", nil},
		{"options par defaut, carte nil", &defauts},
		{"entree de catalogue sans largeurs", &avecEntreeVide},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			res, err := Decode(t.Context(), miniBobineFilm, src, cas.opts)
			if !errors.Is(err, ErrCarteAbsente) {
				t.Fatalf("Decode sans carte : err = %v, attendu ErrCarteAbsente — le film est decode "+
					"aux largeurs d une AUTRE carte (repli interdit)", err)
			}
			if res != nil {
				t.Errorf("un resultat est rendu avec l erreur : %d ligne(s) publiables", len(res.Kills))
			}
		})
	}
}

// TestDecodeAvecCarteDecode : le controle negatif — la bobine (Cliffhanger) sous SA carte decode,
// et la calibration dit que la carte est lue.
func TestDecodeAvecCarteDecode(t *testing.T) {
	src, err := source.LoadDir(miniBobineDir, nil)
	if err != nil {
		t.Fatalf("mini-bobine illisible : %v", err)
	}
	res, err := Decode(t.Context(), miniBobineFilm, src, optionsDeLaBobine(t))
	if err != nil {
		t.Fatalf("Decode sous la carte %q : %v", miniBobineCarte, err)
	}
	if len(res.Kills) == 0 {
		t.Fatal("aucune ligne publiee sous la carte : la garde ecarte aussi les films qu elle doit laisser passer")
	}
}

// TestDecodeRechercheSansCarteEstNommee : l instrument de recherche decode sans carte, et la
// calibration le DIT (« DEFAUT (carte absente) »). C est la seule porte vers les largeurs par
// defaut, et elle porte un nom.
func TestDecodeRechercheSansCarteEstNommee(t *testing.T) {
	src, err := source.LoadDir(miniBobineDir, nil)
	if err != nil {
		t.Fatalf("mini-bobine illisible : %v", err)
	}
	opts := DefaultOptions()
	opts.RechercheSansCarte = true
	res, err := Decode(t.Context(), miniBobineFilm, src, &opts)
	if err != nil {
		t.Fatalf("Decode en recherche sans carte : %v", err)
	}
	if want := "[DEFAUT (carte absente)]"; !strings.Contains(res.Calibration, want) {
		t.Errorf("calibration %q : la mention %q manque — l absence de carte ne se dit plus", res.Calibration, want)
	}
}
