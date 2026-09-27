//go:build integration

package killcollector

// carte_integration_test.go — LES TESTS D INTEGRATION NOMMENT LA CARTE DE LEUR FILM (2026-09-27).
//
// Depuis ce jour, un film sans carte resolue est MIS DE COTE (« pas de repli » : jamais un decodage
// aux largeurs d une autre carte). Un test qui veut des lignes ecrites doit donc cabler la
// resolution de carte de son collecteur sur la carte de son film.

import "testing"

// carteDe9b191a7f : la carte du film de reference des tests de `collector_test.go`.
const carteDe9b191a7f = "Bazaar"

// sousLaCarte cable la resolution de carte du collecteur sur UNE carte du catalogue commis.
func sousLaCarte(t *testing.T, col *KillSourceCollector, carte string) *KillSourceCollector {
	t.Helper()
	return col.WithPositionCapture(nomsDeCarteFixes{noms: []string{carte}}, catalogueDuDepot(t))
}
