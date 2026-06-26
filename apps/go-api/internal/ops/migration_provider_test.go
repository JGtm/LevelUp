package ops

import (
	"os"
	"testing"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

// TestMain câble le provider de steps title-owned (halo_infinite) AVANT tous les
// tests du package ops — exactement comme le boot serveur (cmd/server/main.go) et
// les migration_provider_test.go des autres packages (service, api, sync,
// platform/duckdb). Sans ce câblage, migration.RunForDB(TargetShared) ne voit que
// le registre global : les tables relocalisées en voie B (match_registry,
// match_participants, medals_earned, xuid_aliases…) ne sont pas créées et les
// tests d'intégration seed-demo échouent avec « Catalog Error: Table ... does not exist ».
func TestMain(m *testing.M) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	os.Exit(m.Run())
}
