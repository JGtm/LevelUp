package scheduler_test

import (
	"os"
	"testing"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

// TestMain câble le provider de steps title-owned (halo_infinite) AVANT tous les tests
// du package scheduler — exactement comme le boot (cmd/server/main.go). Sans ce câblage,
// migration.RunForDB(...) ne voit que le registre global ; depuis Phase 1.5 (voie B) le
// god-file shared (create_base_shared_schema → match_registry/match_participants…) est
// title-owned → tables manquantes dans les tests E2E (data_health_check).
func TestMain(m *testing.M) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	os.Exit(m.Run())
}
