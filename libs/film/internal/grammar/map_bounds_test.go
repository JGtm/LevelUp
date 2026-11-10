package grammar

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

func TestNormalizeMapName(t *testing.T) {
	cases := map[string]string{
		"Cliffhanger":       "cliffhanger",
		"Launch Site":       "launch site",
		"Streets - Ranked":  "streets",
		"  Live   Fire  ":   "live fire",
		"Aquarius - Ranked": "aquarius",
		// Variantes de sandbox : même level_id que la base, donc mêmes bornes monde
		// (mesuré le 2026-08-08 — cf. variantSuffixes). Mutation qui doit faire rougir :
		// retirer " heavies" de variantSuffixes.
		"Fragmentation Heavies": "fragmentation",
		"Highpower Heavies":     "highpower",
		"Breaker Heavies":       "breaker",
		// Un nom qui CONTIENT le mot sans le porter en suffixe n'est pas touché.
		"Heavies Bay": "heavies bay",
	}
	for in, want := range cases {
		if got := profile.NormalizeMapName(in); got != want {
			t.Errorf("profile.NormalizeMapName(%q) = %q, attendu %q", in, got, want)
		}
	}
}

func TestMapQuantCatalogLookup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "map_quant_bounds.json")
	blob := `{"schemaVersion":1,"source":"test","maps":{"cliffhanger":{"module":"ridgeline",
	  "min":[-41.10293,-56.60673,-84.37054],"max":[72.10938,57.21198,53.17965],"axisWidths":[13,13,14]}}}`
	if err := os.WriteFile(path, []byte(blob), 0o644); err != nil {
		t.Fatal(err)
	}
	cat, err := profile.LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	e, err := cat.Lookup("Cliffhanger - Ranked")
	if err != nil {
		t.Fatalf("carte connue non trouvée: %v", err)
	}
	if e.Module != "ridgeline" || e.AxisWidths != [3]uint{13, 13, 14} {
		t.Errorf("entrée inattendue: %+v", e)
	}
	if rng := e.Range(); rng[0].Min != -41.10293 || rng[2].Max != 53.17965 {
		t.Errorf("Range() incohérent: %+v", rng)
	}
	// Carte absente : erreur explicite, jamais une valeur de repli.
	if _, err := cat.Lookup("Prism"); !errors.Is(err, profile.ErrUnknownMapBounds) {
		t.Errorf("carte inconnue: err = %v, attendu profile.ErrUnknownMapBounds", err)
	}
	// Catalogue nil : même refus.
	var nilCat *profile.MapQuantCatalog
	if _, err := nilCat.Lookup("Cliffhanger"); !errors.Is(err, profile.ErrUnknownMapBounds) {
		t.Errorf("catalogue nil: err = %v, attendu profile.ErrUnknownMapBounds", err)
	}
}

// TestScanFilmBipedPositionsRefusesWithoutBounds : sans bornes de carte, le décodeur REFUSE
// plutôt que de produire des coordonnées fausses (le comportement historique appliquait les
// bornes de Cliffhanger à toutes les cartes).
func TestScanFilmBipedPositionsRefusesWithoutBounds(t *testing.T) {
	opt := DefaultScanFilmOptions()
	if _, err := ScanFilmBipedPositions(t.TempDir(), opt); !errors.Is(err, profile.ErrUnknownMapBounds) {
		t.Errorf("err = %v, attendu profile.ErrUnknownMapBounds", err)
	}
}

// TestMapQuantCatalogShipped vérifie le catalogue RÉELLEMENT livré : la largeur d'axe
// déduite des bornes doit être celle stockée (donc celle mesurée dans les films). Le test
// est ignoré si le catalogue n'est pas dans l'arbre courant (worktree sans data/) ou hors dépôt hôte.
func TestMapQuantCatalogShipped(t *testing.T) {
	path := bornesLivreesParLHote(t)
	if _, err := os.Stat(path); err != nil {
		t.Skip("catalogue de bornes absent de cet arbre")
	}
	cat, err := profile.LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Maps) == 0 {
		t.Fatal("catalogue vide")
	}
	for name, e := range cat.Maps {
		for ax := range 3 {
			extent := float64(e.Max[ax]) - float64(e.Min[ax])
			if extent <= 0 {
				t.Errorf("%s axe %d : étendue non positive (%g)", name, ax, extent)
				continue
			}
			if got := referenceAxisWidth(extent, 16); got != e.AxisWidths[ax] {
				t.Errorf("%s axe %d : largeur déduite %d, stockée %d", name, ax, got, e.AxisWidths[ax])
			}
			// Le quantum au niveau 16 est structurellement dans ]1/120 ; 1/60].
			q := extent / float64(uint64(1)<<e.AxisWidths[ax])
			if q <= 1.0/120 || q > 1.0/60 {
				t.Errorf("%s axe %d : quantum %.6f hors de ]1/120 ; 1/60]", name, ax, q)
			}
		}
	}
}

// TestLesBornesDeTestSontLeCatalogueLivre : la copie que la brique porte dans `testdata/` (lue par
// les tests qui ont besoin des bornes de carte) est, octet a octet, le catalogue que l'hote livre.
// Une regeneration du catalogue de l'hote qui ne remet pas la copie a jour rougit ici. Hors depot
// hote, il n'y a rien a comparer : le test saute.
func TestLesBornesDeTestSontLeCatalogueLivre(t *testing.T) {
	livre, err := os.ReadFile(bornesLivreesParLHote(t))
	if err != nil {
		t.Skipf("catalogue de bornes absent du depot hote : %v", err)
	}
	copie, err := os.ReadFile(bornesDeCartes(t))
	if err != nil {
		t.Fatalf("copie de test illisible : %v", err)
	}
	if !bytes.Equal(livre, copie) {
		t.Errorf("%s differe du catalogue livre par l'hote : recopier le catalogue dans testdata/", bornesDeCartes(t))
	}
}
