package service

import (
	"os"
	"testing"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/games/halo_infinite/skillchain"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/sync"
)

// TestMain câble le classifier LUSR title-owned avant les tests du package service
// (MT-15). Certains tests — match_history_placement (applyLUSRPlacements /
// applyMatchPlacements) — atteignent sync.GetLUSRChain, qui panique si le
// classifier n'est pas posé (fail-loud par design). Mirroir du boot serveur.
//
// Câble aussi le provider de steps title-owned (halo_infinite) AVANT tous les
// tests — exactement comme le boot (cmd/server/main.go) et migration_provider_test.go
// du package duckdb. Sans ce câblage, migration.RunForDB(TargetMetadata) ne voit
// que le registre global et les steps relocalisés en voie B (Phase 1.5, ex.
// xbox_achievement_definitions) sont absents → table manquante dans les tests
// d'intégration (achievements_integration_test.go).
func TestMain(m *testing.M) {
	sync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)
	sync.SetObjectiveFamilyClassifier(skillchain.IsObjectiveSubMode)
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	os.Exit(m.Run())
}
