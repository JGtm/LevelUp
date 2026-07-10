package prestige

import (
	"os"
	"testing"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

// TestMain câble le provider de steps title-owned (halo_infinite) AVANT tous les
// tests du sous-package prestige — exactement comme le boot (cmd/server/main.go) et
// comme le TestMain du package duckdb-root (migration_provider_test.go). Sans ce
// câblage, migration.RunForDB(TargetSharedSocial) ne voit que le registre global :
// les tables title-owned relocalisées en voie B (squad_challenge, battlepass…) sont
// absentes → INSERT échoue dans les tests repo Prestige. Ce TestMain a été perdu par
// l'extraction du sous-package (K3a) : les tests, jadis dans le binaire de test
// duckdb-root, héritaient de son câblage ; ils l'embarquent désormais eux-mêmes.
func TestMain(m *testing.M) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	os.Exit(m.Run())
}
