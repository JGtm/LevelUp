package replay

// golden_catalog_test.go — LE CATALOGUE DE LIBELLÉS DU TITRE, TEL QUE LES GOLDENS LE LISENT.
//
// POURQUOI LE VRAI ET PAS UNE FIXTURE. Depuis le lot 3.2, les noms d'armes, de grenades et
// de capacités ne sont plus dans le code : ils vivent dans
// `config/titles/halo_infinite/mappings/`. Un golden bâti sur une fixture figerait des
// noms inventés pour le test — donc ne verrait PAS une régression de configuration (un
// libellé perdu, une langue vide, un effet de tir renommé).
//
// COMMENT LA BRIQUE LE LIT. Elle n'importe plus les chargeurs de libellés de LevelUp (plan D2,
// item 2.11) : le catalogue est un INSTANTANÉ, `testdata/golden_catalogue_titre.json`, produit par
// la jointure de la production à partir des vrais fichiers du titre. La garde qui exige que
// l'instantané soit celui des fichiers du titre — c'est-à-dire la garde de ces tables de
// configuration — vit côté hôte : `internal/filmcorpus/catalogue_titre_test.go`
// (`TestInstantaneDuCatalogueDuTitreEstFrais`). Un libellé qui change rougit d'abord cette garde.
//
// AUCUN OCTET DE FILM N'EST LU ICI.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// golden_catalogue_titre.json : l'instantané, relatif au paquet.
const instantaneCatalogueTitre = "golden_catalogue_titre.json"

// goldenCatalog charge le catalogue du titre de référence depuis l'instantané versionné.
func goldenCatalog(t *testing.T) LabelCatalog {
	t.Helper()
	chemin := filepath.Join(goldenDir, instantaneCatalogueTitre)
	raw, err := os.ReadFile(chemin) //nolint:gosec // chemin figé dans le code
	if err != nil {
		t.Fatalf("instantané du catalogue du titre illisible (%s) : %v — il se régénère côté hôte "+
			"(internal/filmcorpus/catalogue_titre_test.go)", chemin, err)
	}
	var cat LabelCatalog
	if err := json.Unmarshal(raw, &cat); err != nil {
		t.Fatalf("instantané du catalogue du titre non désérialisable (%s) : %v", chemin, err)
	}
	return cat
}
