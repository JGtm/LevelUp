package profile

import (
	"os"
	"testing"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

// TestMain câble le provider de steps title-owned (halo_infinite) AVANT tous les
// tests du package profile — comme le boot (cmd/server/main.go) et exactement
// comme internal/api/handlers/migration_provider_test.go + internal/service/main_test.go.
//
// Depuis Phase 1.5 (voie B), les schémas shared (create_base_shared_schema →
// match_registry / match_participants / …) sont title-owned. Sans ce câblage,
// migration.RunForDB(TargetShared) dans setupProfileEnv ne crée pas ces tables
// → BuildProfile.countMatchesInWindow échoue car match_participants est absent.
func TestMain(m *testing.M) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	os.Exit(m.Run())
}
