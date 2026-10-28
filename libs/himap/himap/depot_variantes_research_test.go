//go:build research

package himap

// depot_variantes_research_test.go — la lecture du DEPOT DE VARIANTES (`DepotVariantesCarte`)
// par les instruments de recherche.
//
// Le depot est le dump de retro-ingenierie `.ai/re_dump/mapvar` (non versionne) : la SOURCE des
// preuves level_id, des sondes de canevas et des rendus Forge. L'installation du jeu absente
// fait sauter l'instrument avec sa raison ; le depot absent, ou une variante declaree
// manquante, le fait ECHOUER avec sa source nommee (`lireVarianteDuDepot`) — jamais un « saute »
// qui passerait pour une mesure rejouee. Un instrument qui PARCOURT le dump s'en passe plus
// doucement (`dossierDeVariantes` saute).

import (
	"os"
	"path/filepath"
	"testing"
)

// lireVarianteDuDepot rend les octets du `.mvar` `fichier` du depot de variantes, ou fait
// echouer le test en nommant la source manquante.
func lireVarianteDuDepot(t *testing.T, fichier string) []byte {
	t.Helper()
	depot, err := cheminDepuisDepot(DepotVariantesCarte)
	if err != nil {
		t.Fatalf("depot de variantes absent (%v) : les instruments du tag research ne se rejouent pas "+
			"sans lui — le reconstituer sous %s avant de lancer le corpus", err, DepotVariantesCarte)
	}
	brut, err := os.ReadFile(filepath.Join(depot, fichier)) //nolint:gosec // chemin de test, lecture seule
	if err != nil {
		t.Fatalf("variante %s absente du depot : %v", fichier, err)
	}
	return brut
}

// dossierDeVariantes rend le dossier du depot de variantes (`DepotVariantesCarte`), ou fait
// SAUTER l'instrument quand le dump de retro-ingenierie n'est pas la : un instrument qui
// parcourt le dump n'a rien a mesurer sans lui, et ce n'est pas une preuve a faire echouer.
func dossierDeVariantes(t *testing.T) string {
	t.Helper()
	depot, err := cheminDepuisDepot(DepotVariantesCarte)
	if err != nil {
		t.Skipf("depot de variantes absent : %v", err)
	}
	return depot
}
