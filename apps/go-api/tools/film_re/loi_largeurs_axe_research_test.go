//go:build research

package filmre_test

// loi_largeurs_axe_research_test.go — LA CONFRONTATION DE LA LOI AUX PIECES (lot 3.4, 2026-09-16).
//
// Trois pieces, trois tests, et elles ne viennent pas de la meme source :
//
//  1. le CATALOGUE COMMIS `data/titles/halo_infinite/reference/map_quant_bounds.json`, produit
//     hors ligne par `cmd/mapquant-build` depuis les `.module` du jeu — 79 cartes, chacune avec
//     ses bornes ET ses `axisWidths` ;
//  2. le releve memoire de la table DEFAUT `DAT_1445cc9e0`, consigne au journal le 2026-06-11
//     (`ce_prec_widths_1445cc9e0.bin`) : niveaux 0, 1, 2 = 6/6/6, 7/7/7, 8/8/8 ;
//  3. le releve memoire de la table PAR INDEX `DAT_1445ccbe0`, meme journal : 1/1/0, 2/2/1,
//     3/3/2 — dont ce lot etablit qu il s agit de BAZAAR (cf. NOTE_3_4_*).
//
// Aucun film n est ouvert, aucune installation du jeu n est requise : le test lit le catalogue
// versionne et des constantes. Il tourne sous `go test -tags=research ./tools/film_re/`.

import (
	"fmt"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/profile"
	"levelup/go-api/internal/testutil"
	filmre "levelup/go-api/tools/film_re"
)

func chargerCatalogue(t *testing.T) *profile.MapQuantCatalog {
	t.Helper()
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	chemin := filepath.Join(racine, "data", "titles", "halo_infinite", "reference", "map_quant_bounds.json")
	cat, err := profile.LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	if len(cat.Maps) == 0 {
		t.Fatal("catalogue de bornes vide")
	}
	return cat
}

func bornesDe(e profile.MapQuantEntry) [3][2]float32 {
	var b [3][2]float32
	for axe := 0; axe < 3; axe++ {
		b[axe] = [2]float32{e.Min[axe], e.Max[axe]}
	}
	return b
}

// TestLoiDuRemplisseurEgaleLeCatalogue : sur les 79 cartes du catalogue commis, la loi recopiee
// de `FUN_140be9b88` au niveau 16 rend EXACTEMENT le champ `axisWidths`. C est la piece qui
// dit que le lot 3.4 n a AUCUNE valeur a saisir : les largeurs se derivent des bornes.
func TestLoiDuRemplisseurEgaleLeCatalogue(t *testing.T) {
	cat := chargerCatalogue(t)
	var desaccords []string
	for nom, e := range cat.Maps {
		got := filmre.LargeursAxe(bornesDe(e), filmre.NiveauPosition)
		if got != e.AxisWidths {
			desaccords = append(desaccords, fmt.Sprintf("%s : catalogue %v, loi %v", nom, e.AxisWidths, got))
		}
	}
	if len(desaccords) > 0 {
		t.Fatalf("%d carte(s) en desaccord sur %d :\n  %v", len(desaccords), len(cat.Maps), desaccords)
	}
	t.Logf("accord loi / catalogue : %d cartes sur %d", len(cat.Maps), len(cat.Maps))
}

// TestTableDefautEgaleLeReleveMemoire : la table DEFAUT (`DAT_1445cc9e0`), calculee depuis les
// seules bornes `+/-20000` du build, reproduit le relevé Cheat Engine du 2026-06-11 ET rend 14
// au niveau 8 — la valeur que `MovementProfile.AbsoluteAxisW` porte aujourd hui en uniforme,
// alors que le composant de position lit au niveau 16.
func TestTableDefautEgaleLeReleveMemoire(t *testing.T) {
	attendus := map[int][3]uint{
		0:  {6, 6, 6},    // relevé DAT_1445cc9e0 niveau 0
		1:  {7, 7, 7},    // relevé niveau 1
		2:  {8, 8, 8},    // relevé niveau 2
		8:  {14, 14, 14}, // l origine de l uniforme 14 du profil
		16: {22, 22, 22}, // le niveau REELLEMENT lu par le composant de position
		23: {26, 26, 26}, // le garde d epsilon (pas < 1e-4)
	}
	for niveau, attendu := range attendus {
		if got := filmre.LargeursAxe(filmre.BornesParDefautDuBuild, niveau); got != attendu {
			t.Errorf("table defaut niveau %d : %v, attendu %v", niveau, got, attendu)
		}
	}
}

// TestTableParIndexEgaleBazaar : le relevé de `DAT_1445ccbe0` (1/1/0, 2/2/1, 3/3/2 aux trois
// premiers niveaux) est celui de BAZAAR — donc la table PAR INDEX porte bien, ligne a ligne,
// la loi appliquee aux bornes de la carte chargee. C est la piece qui identifie la SOURCE du
// contenu de cette table.
func TestTableParIndexEgaleBazaar(t *testing.T) {
	cat := chargerCatalogue(t)
	e, err := cat.Lookup("Bazaar")
	if err != nil {
		t.Fatalf("bazaar au catalogue : %v", err)
	}
	attendus := map[int][3]uint{
		0:  {1, 1, 0},
		1:  {2, 2, 1},
		2:  {3, 3, 2},
		16: {17, 17, 16}, // = e.AxisWidths
	}
	for niveau, attendu := range attendus {
		if got := filmre.LargeursAxe(bornesDe(e), niveau); got != attendu {
			t.Errorf("bazaar niveau %d : %v, attendu %v", niveau, got, attendu)
		}
	}
}

// TestLargeurIndexDePlageSuitLeCatalogue : la loi de `DAT_144632be0` (1 quand il n y a qu une
// plage, sinon ceilLog2 du compte) rend les largeurs que `cmd/mapquant-build` inscrit dans
// `regionIndexBits`. Live Fire (4 regions declarees, bits = 2) est la seule carte du catalogue
// qui sorte du cas historique.
func TestLargeurIndexDePlageSuitLeCatalogue(t *testing.T) {
	cas := map[int]uint{1: 1, 2: 1, 3: 2, 4: 2, 5: 3, 8: 3, 9: 4}
	for nb, attendu := range cas {
		if got := filmre.LargeurIndexDePlage(nb); got != attendu {
			t.Errorf("%d plage(s) : largeur %d, attendu %d", nb, got, attendu)
		}
	}
	cat := chargerCatalogue(t)
	e, err := cat.Lookup("Live Fire")
	if err != nil {
		t.Fatalf("live fire au catalogue : %v", err)
	}
	if e.EffectiveRegionIndexBits() != filmre.LargeurIndexDePlage(4) {
		t.Errorf("live fire : catalogue %d bits, loi a 4 plages %d bits",
			e.EffectiveRegionIndexBits(), filmre.LargeurIndexDePlage(4))
	}
}
