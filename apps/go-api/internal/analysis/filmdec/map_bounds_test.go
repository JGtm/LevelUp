package filmdec

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeMapName(t *testing.T) {
	cases := map[string]string{
		"Cliffhanger":       "cliffhanger",
		"Launch Site":       "launch site",
		"Streets - Ranked":  "streets",
		"  Live   Fire  ":   "live fire",
		"Aquarius - Ranked": "aquarius",
	}
	for in, want := range cases {
		if got := NormalizeMapName(in); got != want {
			t.Errorf("NormalizeMapName(%q) = %q, attendu %q", in, got, want)
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
	cat, err := LoadMapQuantCatalog(path)
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
	if _, err := cat.Lookup("Prism"); !errors.Is(err, ErrUnknownMapBounds) {
		t.Errorf("carte inconnue: err = %v, attendu ErrUnknownMapBounds", err)
	}
	// Catalogue nil : même refus.
	var nilCat *MapQuantCatalog
	if _, err := nilCat.Lookup("Cliffhanger"); !errors.Is(err, ErrUnknownMapBounds) {
		t.Errorf("catalogue nil: err = %v, attendu ErrUnknownMapBounds", err)
	}
}

// TestScanFilmBipedPositionsRefusesWithoutBounds : sans bornes de carte, le décodeur REFUSE
// plutôt que de produire des coordonnées fausses (le comportement historique appliquait les
// bornes de Cliffhanger à toutes les cartes).
func TestScanFilmBipedPositionsRefusesWithoutBounds(t *testing.T) {
	opt := DefaultScanFilmOptions()
	if _, err := ScanFilmBipedPositions(t.TempDir(), opt); !errors.Is(err, ErrUnknownMapBounds) {
		t.Errorf("err = %v, attendu ErrUnknownMapBounds", err)
	}
}

// TestMapQuantCatalogShipped vérifie le catalogue RÉELLEMENT livré : la largeur d'axe
// déduite des bornes doit être celle stockée (donc celle mesurée dans les films). Le test
// est ignoré si le catalogue n'est pas dans l'arbre courant (worktree sans data/).
func TestMapQuantCatalogShipped(t *testing.T) {
	path := ""
	for _, up := range []string{"../../../..", "../../../../.."} {
		p := filepath.Join(up, "data", "titles", "halo_infinite", "reference", "map_quant_bounds.json")
		if _, err := os.Stat(p); err == nil {
			path = p
			break
		}
	}
	if path == "" {
		t.Skip("catalogue de bornes absent de cet arbre")
	}
	cat, err := LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Maps) == 0 {
		t.Fatal("catalogue vide")
	}
	for name, e := range cat.Maps {
		for ax := 0; ax < 3; ax++ {
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
