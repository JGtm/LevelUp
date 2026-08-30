package handlers_test

import (
	"os"
	"testing"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/games/halo_infinite/skillchain"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/sync"
)

// TestMain câble le provider de steps title-owned (halo_infinite) AVANT tous les tests
// du package handlers — comme le boot (cmd/server/main.go). Depuis Phase 1.5 (voie B) les
// schémas shared_social (create_notifications_in_shared_social → player_notifications/
// notification_preferences/player_records) sont title-owned ; sans ce câblage les tests
// E2E DuckDB des notifications ne trouvent pas leurs tables via RunForDB.
func TestMain(m *testing.M) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	// MT-15 : des handlers atteignent sync.GetLUSRChain (placement match-history) →
	// câbler le classifier (fail-loud par design).
	sync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)
	sync.SetObjectiveFamilyClassifier(skillchain.IsObjectiveSubMode)
	os.Exit(m.Run())
}
