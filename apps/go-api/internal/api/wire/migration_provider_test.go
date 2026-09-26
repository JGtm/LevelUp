package wire

import (
	"os"
	"testing"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/games/halo_infinite/skillchain"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/sync"
)

// TestMain câble le provider de steps title-owned (halo_infinite) + le classifier
// LUSR AVANT tous les tests du sous-package wire — comme le boot (cmd/server/main.go)
// et le TestMain du package api (migration_provider_test.go). Répliqué ici : les
// tests d'intégration progression (post_sync_progression_test.go) ont suivi la DI
// dans wire (K3d) ; sans ce câblage, migration.RunForDB(TargetShared) ne crée pas
// match_registry (tables shared title-owned depuis la voie B) et l'orchestrateur
// progression V2 n'atteint pas sync.GetLUSRChain (fail-loud).
func TestMain(m *testing.M) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	sync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)
	sync.SetObjectiveFamilyClassifier(skillchain.IsObjectiveSubMode)
	os.Exit(m.Run())
}
