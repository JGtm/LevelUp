package filmdec

// film_context_catalogue_test.go — LA REGLE DU CATALOGUE, VERIFIEE SUR LE VRAI CATALOGUE.
//
// # CE QUE CE TEST VERROUILLE
//
// Lot 3 de PLAN_CUISSON_PERF (2026-09-03). Le decoupage d'i0 que `FilmContext` sert a TOUS les
// balayages suit desormais la regle des positions, ecrite une fois dans [resolveI0Layout] :
//
//	decoupage FORCE par l'appelant   maitre, toujours (instruments, tests de recherche) ;
//	entree de CATALOGUE valide       le decoupage du catalogue — il n'est pas lu dans le film ;
//	entree nil ou sans largeurs      repli sur l'AUTO-DETECTION, comme avant le lot 3.
//
// # POURQUOI LE VRAI CATALOGUE, ET PAS DES VALEURS INVENTEES
//
// La correction porte sur UNE propriete de donnee : Live Fire est la premiere carte dont l'arene
// n'est pas la region 0 et dont l'index de region tient sur DEUX bits (`regionIndexBits: 2`,
// `region: 1`), la ou toutes les autres cartes tiennent sur un bit a zero. Une entree ecrite a la
// main dans ce fichier prouverait que `MapQuantEntry.Layout()` sait additionner 4 + 2 ; elle ne
// prouverait pas que le catalogue VERSIONNE porte encore cette carte sous cette forme. Le jour ou
// `map_quant_bounds.json` est regenere sans le champ, c'est ici que ca doit tomber — pas dans une
// cuisson silencieuse.
//
// Le catalogue est un fichier VERSIONNE du depot (`data/titles/halo_infinite/reference/`), lu par
// le meme chemin relatif que `replay/golden_inputs_test.go` : ce test tourne donc en CI.

import (
	"path/filepath"
	"testing"
)

// cheminCatalogueBornes : le catalogue versionne, depuis `internal/analysis/filmdec`.
func cheminCatalogueBornes() string {
	return filepath.Join("..", "..", "..", "..", "..", "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json")
}

// entreeCatalogue rend l'entree VERSIONNEE d'une carte.
func entreeCatalogue(t *testing.T, carte string) MapQuantEntry {
	t.Helper()
	chemin := cheminCatalogueBornes()
	cat, err := LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatalf("catalogue de bornes %s illisible : %v", chemin, err)
	}
	entry, err := cat.Lookup(carte)
	if err != nil {
		t.Fatalf("carte %q absente du catalogue %s : %v", carte, chemin, err)
	}
	return entry
}

// TestRegleDuCatalogueSurLeVraiCatalogue — les DEUX formes d'entree que le depot porte
// aujourd'hui, et ce que le contexte en fait.
func TestRegleDuCatalogueSurLeVraiCatalogue(t *testing.T) {
	// La mini-bobine n'a NI slot bipede NI chunk_00 : l'auto-detection y ECHOUE (cf.
	// film_context_test.go). C'est exactement le film qu'il faut ici — un decoupage rendu sans
	// erreur ne peut alors venir que du catalogue, jamais d'une detection reussie par hasard.
	film := chargerMiniBobine(t)
	if _, _, err := DetectI0LayoutOf(film); err == nil {
		t.Fatal("l'auto-detection REUSSIT sur la mini-bobine : ce test ne prouverait plus rien")
	}

	for _, c := range []struct {
		carte  string
		gate   int
		region uint32
		axes   [3]uint
		note   string
	}{
		{"Live Fire", 6, 1, [3]uint{12, 12, 11},
			"4 regions declarees, arene en region 1 : index sur 2 bits, gate = 3+1+2"},
		{"Cliffhanger", 5, 0, [3]uint{13, 13, 14},
			"le cas de toutes les autres cartes : index sur 1 bit a zero, gate = 3+1+1"},
	} {
		t.Run(c.carte, func(t *testing.T) {
			entry := entreeCatalogue(t, c.carte)
			attendu := I0Layout{GateBits: c.gate, AxisW: c.axes, Region: c.region}
			if lay := entry.Layout(); lay != attendu {
				t.Fatalf("catalogue %s : %s, attendu %s (%s)", c.carte, lay, attendu, c.note)
			}
			fc := NewFilmContextForMap(film, &entry, nil)
			lay, err := fc.I0Layout()
			if err != nil {
				t.Fatalf("%s : le contexte refuse le decoupage du catalogue : %v", c.carte, err)
			}
			if lay != attendu {
				t.Fatalf("%s : le contexte sert %s, le catalogue impose %s", c.carte, lay, attendu)
			}
			if imp := fc.ImposedLayout(); imp == nil || *imp != attendu {
				t.Fatalf("%s : ImposedLayout rend %v, attendu %s", c.carte, imp, attendu)
			}
		})
	}
}

// TestRegleDuCatalogueRepliAutoDetection — entree nil, ou entree SANS largeurs : le contexte
// retombe sur l'auto-detection, avec l'erreur EXACTE qu'elle rend (la mini-bobine echoue).
func TestRegleDuCatalogueRepliAutoDetection(t *testing.T) {
	film := chargerMiniBobine(t)
	_, _, errAuto := DetectI0LayoutOf(film)
	if errAuto == nil {
		t.Fatal("l'auto-detection REUSSIT sur la mini-bobine : le repli ne serait pas mesurable")
	}

	// Une entree de catalogue ANTERIEURE au champ des largeurs : bornes presentes, axisWidths
	// absentes -> Layout() invalide. C'est le cas que le repli protege ; imposer ce decoupage
	// armerait des largeurs NULLES sur tout le film.
	sansLargeurs := MapQuantEntry{Module: "catalogue_anterieur_au_champ"}
	if sansLargeurs.Layout().Valid() {
		t.Fatal("une entree sans axisWidths rend un decoupage VALIDE : le cas de repli a disparu")
	}

	for _, c := range []struct {
		nom   string
		entry *MapQuantEntry
	}{
		{"entree nil (enveloppes D2, usages hors production)", nil},
		{"entree sans largeurs", &sansLargeurs},
	} {
		t.Run(c.nom, func(t *testing.T) {
			fc := NewFilmContextForMap(film, c.entry, nil)
			if imp := fc.ImposedLayout(); imp != nil {
				t.Fatalf("%s : un decoupage %s s'impose, aucun n'etait attendu", c.nom, imp)
			}
			_, err := fc.I0Layout()
			memeErreur(t, c.nom, err, errAuto)
		})
	}

	// `NewFilmContext` — la forme des enveloppes D2 — n'impose jamais rien.
	if imp := NewFilmContext(film).ImposedLayout(); imp != nil {
		t.Fatalf("NewFilmContext impose %s : les enveloppes D2 doivent garder l'auto-detection", imp)
	}
}

// TestRegleDuCatalogueDecoupageForceMaitre — un decoupage deja force par l'appelant l'emporte sur
// le catalogue, MEME sur Live Fire. C'est la precedence des positions
// (`replay.Options.Scan.Layout`), et elle tient parce que les instruments de recherche forcent un
// decoupage pour mesurer AUTRE CHOSE que ce que la carte declare.
func TestRegleDuCatalogueDecoupageForceMaitre(t *testing.T) {
	film := chargerMiniBobine(t)
	entry := entreeCatalogue(t, "Live Fire")
	force := I0Layout{GateBits: 7, AxisW: [3]uint{9, 10, 11}, Region: 3}
	if force == entry.Layout() {
		t.Fatal("le decoupage force EGALE celui du catalogue : la precedence ne serait pas mesuree")
	}

	fc := NewFilmContextForMap(film, &entry, &force)
	lay, err := fc.I0Layout()
	if err != nil {
		t.Fatalf("decoupage force : le contexte refuse %s : %v", force, err)
	}
	if lay != force {
		t.Fatalf("decoupage force : le contexte sert %s, l'appelant impose %s", lay, force)
	}

	// La valeur est COPIEE dans les deux sens : ni l'appelant ni un lecteur d'ImposedLayout ne
	// peut modifier le decoupage que le contexte sert aux balayages.
	force.GateBits = 42
	if lay2, _ := fc.I0Layout(); lay2.GateBits != 7 {
		t.Fatalf("le contexte suit les mutations de l'appelant : gate=%d apres coup", lay2.GateBits)
	}
	imp := fc.ImposedLayout()
	imp.GateBits = 99
	if lay3, _ := fc.I0Layout(); lay3.GateBits != 7 {
		t.Fatalf("ImposedLayout rend le pointeur interne : gate=%d apres mutation", lay3.GateBits)
	}
}
