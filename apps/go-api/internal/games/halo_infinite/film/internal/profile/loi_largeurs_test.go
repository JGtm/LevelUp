package profile_test

// loi_largeurs_test.go — LA LOI DES LARGEURS D AXE, CONFRONTEE A TROIS PIECES INDEPENDANTES.
//
// C EST L INSTRUMENT DE RECHERCHE DU LOT 3.4 (volet preparation, 2026-09-16) PORTE EN
// PRODUCTION. Il vivait sous `//go:build research`, a cote d une SECONDE transcription de la
// meme loi ; la loi est desormais du code de production (`loi_largeurs.go`) et son test l est
// avec elle — une loi que seule une cible `-tags=research` verifie n est pas gardee.
//
// Les trois pieces ne viennent pas de la meme source :
//
//  1. le CATALOGUE COMMIS `data/titles/halo_infinite/reference/map_quant_bounds.json`, produit
//     hors ligne par `cmd/mapquant-build` depuis les `.module` du jeu — 79 cartes, chacune avec
//     ses bornes ET ses `axisWidths` ;
//  2. le releve memoire de la table DEFAUT `DAT_1445cc9e0`, consigne au journal le 2026-06-11
//     (`ce_prec_widths_1445cc9e0.bin`) : niveaux 0, 1, 2 = 6/6/6, 7/7/7, 8/8/8 ;
//  3. le releve memoire de la table PAR INDEX `DAT_1445ccbe0`, meme journal : 1/1/0, 2/2/1,
//     3/3/2 — dont le lot 3.4 preparation etablit qu il s agit de BAZAAR.
//
// Aucun film n est ouvert, aucune installation du jeu n est requise : ces tests lisent le
// catalogue versionne et des constantes.
//
// LA CONFRONTATION AVEC `himap` (la seconde copie de la loi, chez le producteur hors ligne) vit
// dans `loi_largeurs_himap_test.go`, derriere le tag `cgo` : `himap` tire `ooz`, et le vet de la CI
// sans CGO ne doit pas faire tomber les tests ci-dessous, qui n ouvrent rien.

import (
	"fmt"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/testutil"
)

func chargerCatalogueDesCartes(t *testing.T) *profile.MapQuantCatalog {
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
	if len(cat.Maps) < 70 {
		t.Fatalf("catalogue de bornes a %d entrees — balayage muet, au moins 70 attendues",
			len(cat.Maps))
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

// TestLaLoiRendLesLargeursDuCatalogue : sur les 79 cartes du catalogue commis, la loi recopiee
// de `FUN_140be9b88` au niveau du composant de position rend EXACTEMENT le champ `axisWidths`.
//
// C EST LA PIECE QUI DIT QUE LE LOT 3.4.1 N A AUCUNE VALEUR A SAISIR : les largeurs se
// derivent des bornes, et le catalogue les porte deja justes. C est aussi le CONTROLE du
// champ : une entree ajoutee a la main, ou un catalogue regenere par un outil qui aurait
// derive, rougit ici.
//
// Mutation qui doit le faire rougir (jouee au lot 3.4 preparation) : `C = 1/120` -> `1/121`
// rend `streets` en desaccord ([12 12 12] attendu, [12 12 13] rendu).
func TestLaLoiRendLesLargeursDuCatalogue(t *testing.T) {
	cat := chargerCatalogueDesCartes(t)
	var desaccords []string
	for nom, e := range cat.Maps {
		got := profile.LargeursAxeDuNiveau(bornesDe(e), profile.NiveauPositionDObjet)
		if got != e.AxisWidths {
			desaccords = append(desaccords, fmt.Sprintf("%s : catalogue %v, loi %v", nom, e.AxisWidths, got))
		}
	}
	if len(desaccords) > 0 {
		t.Fatalf("%d carte(s) en desaccord sur %d :\n  %v", len(desaccords), len(cat.Maps), desaccords)
	}
	t.Logf("accord loi / catalogue : %d cartes sur %d", len(cat.Maps), len(cat.Maps))
}

// TestPrecisionAbsolueProjetteLEntreeDeCatalogue : le descripteur du chemin absolu EST l entree
// de catalogue, projetee — pas une seconde table de valeurs.
func TestPrecisionAbsolueProjetteLEntreeDeCatalogue(t *testing.T) {
	cat := chargerCatalogueDesCartes(t)
	for _, nom := range []string{"Live Fire", "Bazaar", "Cliffhanger"} {
		e, err := cat.Lookup(nom)
		if err != nil {
			t.Fatalf("%s au catalogue : %v", nom, err)
		}
		p := e.PrecisionAbsolue()
		if p.AxisW != e.AxisWidths || p.IndexW != e.EffectiveRegionIndexBits() || p.Region != e.Region {
			t.Errorf("%s : descripteur %+v, entree axes %v / indexW %d / region %d",
				nom, p, e.AxisWidths, e.EffectiveRegionIndexBits(), e.Region)
		}
		// LE DECOUPAGE EN DERIVE, et c est ce qui interdit aux deux de diverger.
		lay := e.Layout()
		if lay.AxisW != p.AxisW || lay.Region != p.Region ||
			lay.GateBits != profile.I0SpineBits+profile.I0UseDefaultBits+int(p.IndexW) {
			t.Errorf("%s : le decoupage %s ne derive pas du descripteur %+v", nom, lay, p)
		}
	}
}

// TestTableDefautEgaleLeReleveMemoire : la table DEFAUT, calculee depuis les seules bornes
// `+/-20000` du build, reproduit le releve Cheat Engine du 2026-06-11 — ET rend 14 au niveau 8,
// c est-a-dire l origine de l uniforme que le lot 3.4.1 supprime : `14` etait une entree REELLE
// des tables du jeu, mais au mauvais NIVEAU.
func TestTableDefautEgaleLeReleveMemoire(t *testing.T) {
	attendus := map[int][3]uint{
		0:  {6, 6, 6},    // releve DAT_1445cc9e0 niveau 0
		1:  {7, 7, 7},    // releve niveau 1
		2:  {8, 8, 8},    // releve niveau 2
		8:  {14, 14, 14}, // l origine de l uniforme 14 du profil
		16: {22, 22, 22}, // le niveau REELLEMENT lu par le composant de position
		23: {26, 26, 26}, // le garde d epsilon (pas < 1e-4)
	}
	for niveau, attendu := range attendus {
		if got := profile.LargeursAxeParDefautDuBuild(niveau); got != attendu {
			t.Errorf("table defaut niveau %d : %v, attendu %v", niveau, got, attendu)
		}
	}
}

// TestTableParIndexEgaleBazaar : le releve de `DAT_1445ccbe0` (1/1/0, 2/2/1, 3/3/2 aux trois
// premiers niveaux) est celui de BAZAAR — donc la table PAR INDEX porte bien, ligne a ligne, la
// loi appliquee aux bornes de la carte chargee.
func TestTableParIndexEgaleBazaar(t *testing.T) {
	cat := chargerCatalogueDesCartes(t)
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
		if got := profile.LargeursAxeDuNiveau(bornesDe(e), niveau); got != attendu {
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
		if got := profile.LargeurIndexDePlage(nb); got != attendu {
			t.Errorf("%d plage(s) : largeur %d, attendu %d", nb, got, attendu)
		}
	}
	cat := chargerCatalogueDesCartes(t)
	e, err := cat.Lookup("Live Fire")
	if err != nil {
		t.Fatalf("live fire au catalogue : %v", err)
	}
	if e.EffectiveRegionIndexBits() != profile.LargeurIndexDePlage(4) {
		t.Errorf("live fire : catalogue %d bits, loi a 4 plages %d bits",
			e.EffectiveRegionIndexBits(), profile.LargeurIndexDePlage(4))
	}
}

// TestLesDeuxGardesDeLaLoiSontModelises — D2 (3.4), SUR UNE CARTE SYNTHETIQUE.
//
// Les deux gardes du remplisseur ne se declenchent sur AUCUNE carte reelle : au niveau 16 le
// garde de debordement demande une etendue de 69 905,1 unites monde quand la plus grande du
// catalogue est 2 707,4 (`recharge`), et le garde d epsilon ne concerne que les niveaux >= 23,
// que le composant de position n emprunte pas. Les laisser hors de la loi la rendait fausse
// pour un canevas Forge plus grand, EN SILENCE — d ou une carte fabriquee, ici, dont l etendue
// les franchit.
func TestLesDeuxGardesDeLaLoiSontModelises(t *testing.T) {
	// SEUIL EXACT du garde de debordement au niveau 16 : 2^22 casiers a 1/60 d unite.
	const seuil = float32(4194304) / 60
	if got := profile.LargeursAxeDuNiveau(
		[3][2]float32{{0, seuil * 0.99}, {0, seuil * 0.99}, {0, seuil * 0.99}},
		profile.NiveauPositionDObjet); got != [3]uint{22, 22, 22} {
		t.Errorf("juste SOUS le seuil de debordement : %v, attendu 22/22/22 "+
			"(ceilLog2 de ~4,15 millions de casiers)", got)
	}
	// AU-DELA le compte est FIGE a 2^22, donc la largeur reste 22 quelle que soit l etendue —
	// c est ce que le garde fait, et sans lui la loi rendrait 23, 24, ... a l infini.
	for _, facteur := range []float32{1.01, 4, 1000} {
		e := seuil * facteur
		if got := profile.LargeursAxeDuNiveau(
			[3][2]float32{{0, e}, {0, e}, {0, e}}, profile.NiveauPositionDObjet); got != [3]uint{22, 22, 22} {
			t.Errorf("etendue %.0f (x%.2f le seuil) : %v, attendu 22/22/22 — le garde 2^22 "+
				"n est pas applique", float64(e), facteur, got)
		}
	}
	// LE GARDE D EPSILON : des le niveau 23 le pas passe sous 1e-4 et les trois largeurs valent
	// 26 SANS que les bornes soient regardees — y compris pour une boite minuscule, qui sans lui
	// rendrait 0/0/0.
	minuscule := [3][2]float32{{0, 1e-3}, {0, 1e-3}, {0, 1e-3}}
	if got := profile.LargeursAxeDuNiveau(minuscule, 22); got == [3]uint{26, 26, 26} {
		t.Errorf("niveau 22 : le garde d epsilon ne doit PAS encore mordre (pas = %g)",
			profile.PasDuNiveau(22))
	}
	if got := profile.LargeursAxeDuNiveau(minuscule, 23); got != [3]uint{26, 26, 26} {
		t.Errorf("niveau 23 : %v, attendu 26/26/26 (pas = %g < 1e-4)",
			got, profile.PasDuNiveau(23))
	}
}
