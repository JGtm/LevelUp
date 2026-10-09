//go:build gamefiles

package himap

// depot_variantes_gamefiles_test.go — la lecture du DEPOT DE VARIANTES (`DepotVariantesCarte`)
// par les preuves du corpus gamefiles.
//
// Sous ce tag, l'installation du jeu est la seule absence toleree (les tests la sautent avec sa
// raison) : le depot, lui, est la SOURCE des preuves level_id, des sondes de canevas et des
// rendus Forge. Son absence, ou celle d'une variante declaree, est un ECHEC nomme — jamais un
// « saute » qui, dans un run gamefiles vert, passerait pour une preuve rejouee.

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
		t.Fatalf("depot de variantes absent (%v) : les preuves du tag gamefiles ne se rejouent pas "+
			"sans lui — le reconstituer sous %s avant de lancer le corpus", err, DepotVariantesCarte)
	}
	brut, err := os.ReadFile(filepath.Join(depot, fichier)) //nolint:gosec // chemin de test, lecture seule
	if err != nil {
		t.Fatalf("variante %s absente du depot : %v", fichier, err)
	}
	return brut
}
