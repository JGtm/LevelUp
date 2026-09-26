package sync_test

import (
	"os"
	"testing"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/games/halo_infinite/skillchain"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/sync"
)

// TestMain câble le provider de steps title-owned (halo_infinite) AVANT tous les
// tests du package sync — exactement comme le boot (cmd/server/main.go) et comme
// les TestMain de scheduler/handlers. Sans ce câblage, migration.RunForDB(...) que
// les tests d'intégration sync utilisent pour bâtir leur schéma ne voit que le
// registre global ; depuis Phase 1.5 (voie B) le god-file shared
// (create_base_shared_schema → match_registry/match_participants…) + les schémas
// player/social sont title-owned → tables manquantes (« match_registry does not
// exist »). Comble le trou laissé par la relocation (scheduler/handlers/duckdb
// câblés, sync oublié).
func TestMain(m *testing.M) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	// MT-15 : câble le classifier LUSR title-owned avant les tests sync (sinon
	// GetLUSRChain panique — fail-loud par design). Couvre TestGetLUSRChain,
	// TestGetPerformanceChain et tous les tests d'intégration de scoring.
	sync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)
	// Scission ranked par famille : classifier de famille objectif, consommé par
	// GetPerformanceChain (ranked_slayer / ranked_objectif). Sans lui, le fallback
	// classerait tout le classé en ranked_slayer et TestGetPerformanceChain
	// échouerait — c'est voulu : les tests vérifient le régime CÂBLÉ.
	sync.SetObjectiveFamilyClassifier(skillchain.IsObjectiveSubMode)
	os.Exit(m.Run())
}
