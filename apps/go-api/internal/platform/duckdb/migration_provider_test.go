package duckdb

import (
	"os"
	"testing"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

// TestMain câble le provider de steps title-owned (halo_infinite) AVANT tous les
// tests du package — exactement comme le boot (cmd/server/main.go:1368). Sans ce
// câblage, migration.RunForDB(...) ne voit que le registre global et les steps
// relocalisés en voie B (Phase 1.5, ex. famille citation_mappings, battlepass,
// challenge…) sont absents → tables manquantes dans les tests repo.
func TestMain(m *testing.M) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	os.Exit(m.Run())
}
