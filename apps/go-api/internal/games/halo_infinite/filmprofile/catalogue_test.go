package filmprofile_test

// catalogue_test.go — LA GRAMMAIRE DES CLEFS ET LE REFUS.
//
// Un catalogue de profil n a de valeur que si ce qu il refuse est plus large que ce qu il
// accepte : une clef qui ne selectionne jamais rien, une provenance inventee, une preuve vide
// donneraient au decodeur une certitude qu il n a pas. Ce fichier exerce les deux cotes.

import (
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/filmprofile"
)

// catalogueMinimalValide rend un catalogue accepte, dont les cas de refus derivent.
func catalogueMinimalValide() filmprofile.Catalogue {
	return filmprofile.Catalogue{
		SchemaVersion: filmprofile.SchemaVersionCourante,
		TitleSlug:     "halo_infinite",
		APropos:       "catalogue d essai",
		Derive: filmprofile.Derive{
			Catalogue:       "map_quant_bounds.json",
			SchemaVersion:   1,
			Cartes:          79,
			EmpreinteCartes: strings.Repeat("a", 64),
			Outil:           "go run ./cmd/mapquant-build",
		},
		Entrees: []filmprofile.Entree{{
			Cle: "format=27", Champ: "MPP", Valeur: "lead=9 index=5",
			Source: filmprofile.ProvenanceRelue, Preuve: "FUN_14080cfe8", Date: "2026-09-15",
		}},
	}
}

func TestParseCleAccepteLesQuatreSortes(t *testing.T) {
	cas := []struct {
		brut  string
		sorte filmprofile.SorteDeCle
		lues  filmprofile.ClesLues
	}{
		{"toutes", filmprofile.SorteToutes, filmprofile.ClesLues{}},
		{"format=27", filmprofile.SorteFormat, filmprofile.ClesLues{Format: 27}},
		{"format=20,21,24,25", filmprofile.SorteFormat, filmprofile.ClesLues{Format: 24}},
		{"build=HI_1_13_0", filmprofile.SorteBuild, filmprofile.ClesLues{Build: "HI_1_13_0"}},
		{"majeure<=38", filmprofile.SorteMajeure, filmprofile.ClesLues{Majeure: 38}},
		{"majeure=39,40", filmprofile.SorteMajeure, filmprofile.ClesLues{Majeure: 40}},
		{"majeure>=41", filmprofile.SorteMajeure, filmprofile.ClesLues{Majeure: 41}},
	}
	for _, c := range cas {
		t.Run(c.brut, func(t *testing.T) {
			cle, err := filmprofile.ParseCle(c.brut)
			if err != nil {
				t.Fatalf("refusee : %v", err)
			}
			if cle.Sorte != c.sorte {
				t.Errorf("sorte %q, attendu %q", cle.Sorte, c.sorte)
			}
			if !cle.Correspond(c.lues) {
				t.Errorf("ne selectionne pas %+v", c.lues)
			}
		})
	}
}

// TestCorrespondNeSelectionnePasAuHasard — l autre cote : une clef qui accepte tout ne dit rien.
func TestCorrespondNeSelectionnePasAuHasard(t *testing.T) {
	cas := []struct {
		brut string
		lues filmprofile.ClesLues
	}{
		{"format=27", filmprofile.ClesLues{Format: 24}},
		{"format=20,21,24,25", filmprofile.ClesLues{Format: 27}},
		{"build=HI_1_13_0", filmprofile.ClesLues{Build: "HI_1_12_0"}},
		{"majeure<=38", filmprofile.ClesLues{Majeure: 39}},
		{"majeure=39,40", filmprofile.ClesLues{Majeure: 41}},
		{"majeure>=41", filmprofile.ClesLues{Majeure: 40}},
	}
	for _, c := range cas {
		t.Run(c.brut, func(t *testing.T) {
			cle, err := filmprofile.ParseCle(c.brut)
			if err != nil {
				t.Fatalf("refusee : %v", err)
			}
			if cle.Correspond(c.lues) {
				t.Errorf("selectionne %+v alors qu elle ne devrait pas", c.lues)
			}
		})
	}
}

func TestParseCleRefuseCeQuElleNeComprendPas(t *testing.T) {
	refusees := []string{
		"", "toutes les cartes", "format=", "format=vingt-sept", "build=",
		"majeure<=38,39", "majeure>=", "carte=aquarius", "27",
	}
	for _, brut := range refusees {
		t.Run(brut, func(t *testing.T) {
			if _, err := filmprofile.ParseCle(brut); err == nil {
				t.Errorf("clef %q acceptee — une clef que personne ne comprend selectionne "+
					"zero film en silence", brut)
			}
		})
	}
}

func TestValideAccepteLeCatalogueMinimal(t *testing.T) {
	cat := catalogueMinimalValide()
	if err := cat.Valide(); err != nil {
		t.Fatalf("catalogue minimal refuse : %v", err)
	}
}

// TestValideRefuse — un cas par regle, chacun derive du catalogue minimal.
func TestValideRefuse(t *testing.T) {
	cas := map[string]func(*filmprofile.Catalogue){
		"schema inconnu":         func(c *filmprofile.Catalogue) { c.SchemaVersion = 2 },
		"titre vide":             func(c *filmprofile.Catalogue) { c.TitleSlug = "" },
		"about vide":             func(c *filmprofile.Catalogue) { c.APropos = "" },
		"aucune entree":          func(c *filmprofile.Catalogue) { c.Entrees = nil },
		"derive sans catalogue":  func(c *filmprofile.Catalogue) { c.Derive.Catalogue = "" },
		"derive sans outil":      func(c *filmprofile.Catalogue) { c.Derive.Outil = "" },
		"derive sans carte":      func(c *filmprofile.Catalogue) { c.Derive.Cartes = 0 },
		"empreinte tronquee":     func(c *filmprofile.Catalogue) { c.Derive.EmpreinteCartes = "abc" },
		"clef illisible":         func(c *filmprofile.Catalogue) { c.Entrees[0].Cle = "carte=aquarius" },
		"champ vide":             func(c *filmprofile.Catalogue) { c.Entrees[0].Champ = "" },
		"valeur vide":            func(c *filmprofile.Catalogue) { c.Entrees[0].Valeur = "" },
		"provenance inventee":    func(c *filmprofile.Catalogue) { c.Entrees[0].Source = "devinee" },
		"preuve vide":            func(c *filmprofile.Catalogue) { c.Entrees[0].Preuve = "" },
		"date qui n en est pas":  func(c *filmprofile.Catalogue) { c.Entrees[0].Date = "hier" },
		"deux fois la meme case": func(c *filmprofile.Catalogue) { c.Entrees = append(c.Entrees, c.Entrees[0]) },
	}
	for nom, casser := range cas {
		t.Run(nom, func(t *testing.T) {
			cat := catalogueMinimalValide()
			casser(&cat)
			if err := cat.Valide(); err == nil {
				t.Errorf("accepte : %s", nom)
			}
		})
	}
}

// TestDecoderRefuseUnChampInconnu — un champ mal orthographie serait une donnee saisie que
// personne ne lit ; le decodeur doit le dire, pas l ignorer.
func TestDecoderRefuseUnChampInconnu(t *testing.T) {
	const blob = `{"schemaVersion":1,"titleSlug":"halo_infinite","about":"x",
	"derived":{"catalogue":"map_quant_bounds.json","schemaVersion":1,"maps":79,
	"mapsSha256":"` + `0000000000000000000000000000000000000000000000000000000000000000` + `","tool":"t"},
	"entries":[{"key":"toutes","field":"F","value":"V","provenence":"relue","proof":"p","date":"2026-09-16"}]}`
	if _, err := filmprofile.Decoder([]byte(blob)); err == nil {
		t.Error("champ `provenence` accepte — une faute de frappe ferait disparaitre la provenance")
	}
}

func TestEntreesPourSelectionneEtRendCompteDUneClefIllisible(t *testing.T) {
	cat := catalogueMinimalValide()
	cat.Entrees = append(cat.Entrees, filmprofile.Entree{
		Cle: "toutes", Champ: "Keyframe.EnTeteBits", Valeur: "108",
		Source: filmprofile.ProvenanceRelue, Preuve: "FUN_142e2bfd0", Date: "2026-08-17",
	})
	sel, err := cat.EntreesPour(filmprofile.ClesLues{Format: 27, Build: "HI_1_13_0", Majeure: 41})
	if err != nil {
		t.Fatalf("selection : %v", err)
	}
	if len(sel) != 2 {
		t.Fatalf("%d entree(s) selectionnee(s), 2 attendues", len(sel))
	}
	sel, err = cat.EntreesPour(filmprofile.ClesLues{Format: 24})
	if err != nil {
		t.Fatalf("selection : %v", err)
	}
	if len(sel) != 1 || sel[0].Champ != "Keyframe.EnTeteBits" {
		t.Fatalf("selection inattendue pour un format non couvert : %+v", sel)
	}

	cat.Entrees[0].Cle = "carte=aquarius"
	if _, err := cat.EntreesPour(filmprofile.ClesLues{Format: 27}); err == nil {
		t.Error("clef illisible sautee en silence — le profil serait incomplet sans le dire")
	}
}
