package replay

// journal_de_publication_test.go — LES AVERTISSEMENTS PROPRES AU DOCUMENT PUBLIE SE TAISENT SUR LE
// DOCUMENT INTERNE ([niveauDePublication], [Options.documentInterne]).
//
//	JDP-INTERNE  le document que `PortagesAuSync` assemble pour un Oddball (ni table du film, ni horloge
//	             du film, ni equipes, ni capacites balayees) n'ecrit aucun des six avertissements ;
//	JDP-PUBLIE   les memes entrees republiees depuis les faits les ecrivent tous les six en WARN ;
//	JDP-NIVEAU   les deux avertissements qu'aucun de ces deux documents n'atteint (la base contredit
//	             les equipes du film, l'origine lue est contredite par le fil des morts) suivent le
//	             niveau qu'on leur donne : WARN publie, rien en interne.

import (
	"context"
	"testing"
)

// avertissementsDePublication : un extrait de chacun des six avertissements qui disent un defaut du
// document publie.
var avertissementsDePublication = []string{
	"table du film NON EMPLOYEE",
	"origine d'horloge non etablie",
	"aucune origine etablie",
	"equipes NON LUES",
	"impulsions de capacite NON BALAYEES",
	"charges d equipement NON BALAYEES",
}

func TestPortagesAuSync_LeDocumentInterneTaitLesAvertissementsDePublication(t *testing.T) { // JDP-INTERNE
	e := EntreePorteursAuSync{MatchID: "temoin", Variante: "Oddball:Arena", Identite: lecturesSansEquipe(),
		lireStatborg: statborgVide}
	journal := journalDe(t, func() { _, _ = PortagesAuSync(context.Background(), e) })
	for _, extrait := range avertissementsDePublication {
		if aUnEnregistrement(journal, "WARN", extrait) {
			t.Errorf("avertissement %q ecrit par le document interne des porteurs au sync : il dit un defaut "+
				"du document publie", extrait)
		}
	}
}

func TestBuildFromFacts_LeDocumentPublieEcritLesAvertissementsDePublication(t *testing.T) { // JDP-PUBLIE
	lu := lecturesSansEquipe()
	faits := &FilmFactsFile{Facts: FilmFacts{Film: "temoin", FilmInputs: FilmInputs{Positions: lu.Positions,
		BipedCreations: lu.BipedCreations, PlayerIndices: lu.PlayerIndices}}}
	journal := journalDe(t, func() { _ = BuildFromFacts(context.Background(), "temoin", "halo_infinite", faits, Options{}) })
	for _, extrait := range avertissementsDePublication {
		if !aUnEnregistrement(journal, "WARN", extrait) {
			t.Errorf("avertissement %q absent du journal du document publie : %v", extrait, journal)
		}
	}
}

func TestAvertissementsSansTemoinSuiventLeNiveauDePublication(t *testing.T) { // JDP-NIVEAU
	jouer := func(interne bool) [][2]string {
		return journalDe(t, func() {
			niveau := niveauDePublication(interne)
			logTeamCoverage(context.Background(), "m", TeamCoverage{Read: true, Contradiction: 1}, niveau)
			// Temoin decale de 5 s sur 90 morts : l'origine lue est refusee.
			resolveOriginMs(context.Background(), 4_521_507_487, 4_517_903_087,
				temoinDuFil{offsetMS: 4_512_847, appariees: 90}, niveau)
		})
	}
	for _, extrait := range []string{"la base CONTREDIT le film", "origine LUE contredite"} {
		if !aUnEnregistrement(jouer(false), "WARN", extrait) {
			t.Errorf("%q : absent en WARN du document publie", extrait)
		}
		if aUnEnregistrement(jouer(true), "WARN", extrait) {
			t.Errorf("%q : ecrit en WARN par le document interne", extrait)
		}
	}
}
