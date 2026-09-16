package filmprofile_test

// empreinte_registre_test.go — CE QUE LA TABLE DES EMPREINTES REFUSE, ET CE QU ELLE REND.
//
// Une table d empreintes de registre n a de valeur que si elle refuse plus largement qu elle
// n accepte : une empreinte tronquee, une clef qui ne veut rien dire pour un registre, un build
// que le catalogue ne connait pas par ailleurs, une ligne que personne n a mesuree — chacun de
// ces cas donnerait au decodeur une certitude qu il n a pas, et le rendrait MUET la ou il doit
// dire « grammaire inconnue ».
//
// Le fichier commis est verifie ici aussi : ses neuf clefs couvrent les sept builds de la table
// des builds et les deux familles de films sans section d identification.

import (
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/filmprofile"
)

// catalogueAvecEmpreintes rend un catalogue minimal PLUS une table d empreintes valide, dont
// les cas de refus derivent.
func catalogueAvecEmpreintes() filmprofile.Catalogue {
	cat := catalogueMinimalValide()
	cat.Entrees = append(cat.Entrees, filmprofile.Entree{
		Cle: "build=HI_1_13_0", Champ: "Slots.PersoBytes", Valeur: "1852",
		Source: filmprofile.ProvenanceRelue, Preuve: "FUN_1407edea8", Date: "2026-09-12",
	})
	cat.EmpreintesRegistre = []filmprofile.EmpreinteRegistre{{
		Cle: "build=HI_1_13_0", Empreinte: "0x36ca8c3d2a2f9b88", Blocs: 50, SlotsNommes: 1067,
		Statut: filmprofile.StatutConnue, Source: filmprofile.ProvenanceMesuree,
		Temoins: []string{"fb1a1a72"}, Preuve: "mini-bobine commise", Date: "2026-09-16",
	}, {
		Cle: "majeure=31", Empreinte: "0xba34fa35f781d1a7", Blocs: 49, SlotsNommes: 1029,
		Statut: filmprofile.StatutPresumee, Source: filmprofile.ProvenanceMesuree,
		Temoins: []string{"50247b26"}, Preuve: "film sans section d identification", Date: "2026-09-16",
	}}
	return cat
}

func TestValideAccepteLaTableDesEmpreintes(t *testing.T) {
	cat := catalogueAvecEmpreintes()
	if err := cat.Valide(); err != nil {
		t.Fatalf("table d empreintes valide refusee : %v", err)
	}
}

// TestSectionDesEmpreintesEstFacultative — un catalogue qui ne connait le registre d aucun
// build reste valide : il le DIT, au lieu d etre refuse.
func TestSectionDesEmpreintesEstFacultative(t *testing.T) {
	cat := catalogueMinimalValide()
	if err := cat.Valide(); err != nil {
		t.Fatalf("catalogue sans table d empreintes refuse : %v", err)
	}
}

// TestValideRefuseUneEmpreinte — un cas par regle, chacun derive de la table valide.
func TestValideRefuseUneEmpreinte(t *testing.T) {
	cas := map[string]func(*filmprofile.EmpreinteRegistre){
		"clef illisible":        func(e *filmprofile.EmpreinteRegistre) { e.Cle = "carte=aquarius" },
		"clef toutes":           func(e *filmprofile.EmpreinteRegistre) { e.Cle = "toutes" },
		"clef de format":        func(e *filmprofile.EmpreinteRegistre) { e.Cle = "format=27" },
		"build hors catalogue":  func(e *filmprofile.EmpreinteRegistre) { e.Cle = "build=HI_9_9_9" },
		"majeure en intervalle": func(e *filmprofile.EmpreinteRegistre) { e.Cle = "majeure>=41" },
		"empreinte sans prefixe": func(e *filmprofile.EmpreinteRegistre) {
			e.Empreinte = "36ca8c3d2a2f9b88"
		},
		"empreinte tronquee": func(e *filmprofile.EmpreinteRegistre) { e.Empreinte = "0x36ca8c3d" },
		"empreinte trop longue": func(e *filmprofile.EmpreinteRegistre) {
			e.Empreinte = "0x36ca8c3d2a2f9b880"
		},
		"empreinte en majuscules": func(e *filmprofile.EmpreinteRegistre) {
			e.Empreinte = "0x36CA8C3D2A2F9B88"
		},
		"empreinte non hexadecimale": func(e *filmprofile.EmpreinteRegistre) {
			e.Empreinte = "0x36ca8c3d2a2f9bzz"
		},
		"zero bloc":           func(e *filmprofile.EmpreinteRegistre) { e.Blocs = 0 },
		"zero slot nomme":     func(e *filmprofile.EmpreinteRegistre) { e.SlotsNommes = 0 },
		"statut invente":      func(e *filmprofile.EmpreinteRegistre) { e.Statut = "probable" },
		"statut inconnue":     func(e *filmprofile.EmpreinteRegistre) { e.Statut = "inconnue" },
		"provenance inventee": func(e *filmprofile.EmpreinteRegistre) { e.Source = "devinee" },
		"aucun temoin":        func(e *filmprofile.EmpreinteRegistre) { e.Temoins = nil },
		"temoin vide":         func(e *filmprofile.EmpreinteRegistre) { e.Temoins = []string{" "} },
		"temoin en double": func(e *filmprofile.EmpreinteRegistre) {
			e.Temoins = []string{"fb1a1a72", "fb1a1a72"}
		},
		"preuve vide":           func(e *filmprofile.EmpreinteRegistre) { e.Preuve = "" },
		"date qui n en est pas": func(e *filmprofile.EmpreinteRegistre) { e.Date = "hier" },
	}
	for nom, casser := range cas {
		t.Run(nom, func(t *testing.T) {
			cat := catalogueAvecEmpreintes()
			casser(&cat.EmpreintesRegistre[0])
			if err := cat.Valide(); err == nil {
				t.Errorf("accepte : %s", nom)
			}
		})
	}
}

// TestValideRefuseDeuxEmpreintesPourLaMemeClef — deux grammaires pour un meme build : aucune
// ne prime, et le decodeur appliquerait la premiere lue sans que personne ne le sache.
func TestValideRefuseDeuxEmpreintesPourLaMemeClef(t *testing.T) {
	cat := catalogueAvecEmpreintes()
	seconde := cat.EmpreintesRegistre[0]
	seconde.Empreinte = "0x0000000000000001"
	cat.EmpreintesRegistre = append(cat.EmpreintesRegistre, seconde)
	if err := cat.Valide(); err == nil {
		t.Error("deux empreintes pour la meme clef acceptees")
	}
}

// TestValideAccepteDeuxClefsQuiPartagentUneEmpreinte — l unicite porte sur la CLEF, jamais sur
// l empreinte : c est un FAIT MESURE que HI_1_12_0 et HI_1_13_0 partagent leur registre.
func TestValideAccepteDeuxClefsQuiPartagentUneEmpreinte(t *testing.T) {
	cat := catalogueAvecEmpreintes()
	cat.Entrees = append(cat.Entrees, filmprofile.Entree{
		Cle: "build=HI_1_12_0", Champ: "Slots.PersoBytes", Valeur: "1852",
		Source: filmprofile.ProvenanceMesuree, Preuve: "transposition 0 bit", Date: "2026-09-14",
	})
	jumelle := cat.EmpreintesRegistre[0]
	jumelle.Cle = "build=HI_1_12_0"
	jumelle.Temoins = []string{"bcb6d393"}
	cat.EmpreintesRegistre = append(cat.EmpreintesRegistre, jumelle)
	if err := cat.Valide(); err != nil {
		t.Errorf("deux builds au meme registre refuses : %v", err)
	}
}

// TestEmpreinteRegistrePourRendLaPlusSpecifique — un film qui ecrit un build est decrit par son
// build, meme quand une clef de majeure le selectionne aussi.
func TestEmpreinteRegistrePourRendLaPlusSpecifique(t *testing.T) {
	cat := catalogueAvecEmpreintes()
	cat.EmpreintesRegistre[1].Cle = "majeure=41" // selectionne le meme film que build=HI_1_13_0

	e, ok := cat.EmpreinteRegistrePour(filmprofile.ClesLues{Build: "HI_1_13_0", Majeure: 41})
	if !ok {
		t.Fatal("aucune empreinte rendue pour un film que la table couvre")
	}
	if e.Cle != "build=HI_1_13_0" {
		t.Errorf("clef %q rendue, la clef de build etait plus specifique", e.Cle)
	}

	// Film sans section : pas de build ecrit, la majeure repond.
	e, ok = cat.EmpreinteRegistrePour(filmprofile.ClesLues{Majeure: 41})
	if !ok || e.Cle != "majeure=41" {
		t.Errorf("clef de majeure attendue pour un film sans build, obtenu %q (trouve=%v)", e.Cle, ok)
	}

	// Grammaire dont le depot ne sait rien : c est une REPONSE, pas un repli sur une voisine.
	if _, ok := cat.EmpreinteRegistrePour(filmprofile.ClesLues{Build: "HI_9_9_9", Majeure: 99}); ok {
		t.Error("une empreinte rendue pour un build inconnu — le decodeur croirait la grammaire connue")
	}
}

// TestEmpreintesCommisesCouvrentLesBuildsDuCatalogue — LE FICHIER COMMIS.
//
// Mutation qui doit le faire rougir : retirer une des neuf clefs, ou ajouter une entree
// `build=` a `entries` sans poser son empreinte.
func TestEmpreintesCommisesCouvrentLesBuildsDuCatalogue(t *testing.T) {
	cat, _ := chargerCatalogueCommis(t)

	buildsAvecEmpreinte := map[string]bool{}
	var majeures int
	for _, e := range cat.EmpreintesRegistre {
		cle, err := filmprofile.ParseCle(e.Cle)
		if err != nil {
			t.Fatalf("clef %q refusee apres validation — incoherence : %v", e.Cle, err)
		}
		switch cle.Sorte {
		case filmprofile.SorteBuild:
			buildsAvecEmpreinte[cle.Build] = true
		case filmprofile.SorteMajeure:
			majeures++
		}
	}
	for _, e := range cat.Entrees {
		cle, err := filmprofile.ParseCle(e.Cle)
		if err != nil || cle.Sorte != filmprofile.SorteBuild {
			continue
		}
		if !buildsAvecEmpreinte[cle.Build] {
			t.Errorf("le catalogue connait le build %q (entree %q) mais pas son registre — "+
				"le decodeur alerterait `grammaire inconnue` sur un build que le depot connait",
				cle.Build, e.Champ)
		}
	}
	if majeures < 2 {
		t.Errorf("%d clef(s) de majeure, 2 attendues au moins (versions 31 et 33 : les films dont "+
			"chunk_00 ne porte aucune section d identification n ecrivent pas de build)", majeures)
	}
}

// TestEmpreintesCommisesSontDesGrandeursComparables — la raison d etre de la section : des
// valeurs sur lesquelles un test MORD, sans analyser une phrase.
func TestEmpreintesCommisesSontDesGrandeursComparables(t *testing.T) {
	cat, _ := chargerCatalogueCommis(t)
	const empreinteDeReference = "0x36ca8c3d2a2f9b88" // grammar.KnownRegistryFingerprint

	e, ok := cat.EmpreinteRegistrePour(filmprofile.ClesLues{Build: "HI_1_13_0", Majeure: 41})
	if !ok {
		t.Fatal("aucune empreinte pour le build de reference")
	}
	if e.Empreinte != empreinteDeReference || e.Blocs != 50 || e.SlotsNommes != 1067 {
		t.Errorf("build de reference : %s / %d blocs / %d slots, attendu %s / 50 / 1067",
			e.Empreinte, e.Blocs, e.SlotsNommes, empreinteDeReference)
	}
	if e.Statut != filmprofile.StatutConnue {
		t.Errorf("statut du build de reference %q, attendu %q", e.Statut, filmprofile.StatutConnue)
	}

	// Les films sans section d identification n ecrivent pas de build : la majeure repond.
	sans, ok := cat.EmpreinteRegistrePour(filmprofile.ClesLues{Majeure: 31})
	if !ok {
		t.Fatal("aucune empreinte pour les films de majeure 31 sans section d identification")
	}
	if !strings.HasPrefix(sans.Cle, "majeure=") {
		t.Errorf("clef %q rendue pour un film sans build", sans.Cle)
	}
	if sans.Empreinte == empreinteDeReference {
		t.Error("la plus ancienne grammaire du cache rend l empreinte de reference — mesure perdue")
	}
}
