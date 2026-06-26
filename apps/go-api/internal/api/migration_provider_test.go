package api

import (
	"os"
	"testing"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/games/halo_infinite/skillchain"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/sync"
)

// TestMain câble le provider de steps title-owned (halo_infinite) AVANT tous les
// tests du package api — comme le boot (cmd/server/main.go) et exactement comme
// internal/api/handlers/migration_provider_test.go + internal/service/main_test.go.
//
// Depuis Phase 1.5 (voie B), les schémas shared (create_base_shared_schema →
// match_registry / match_participants / medals_earned / …) sont title-owned. Sans
// ce câblage, migration.RunForDB(TargetShared) ne voit que le registre global et
// ces tables sont absentes → les tests d'intégration progression (post_sync_progression_test.go)
// échouent à l'INSERT sur match_registry (« Did you mean schema_migrations »).
//
// On câble aussi le classifier LUSR (fail-loud par design) : l'orchestrateur
// progression V2 (EvaluateProgressionAfterSync) atteint sync.GetLUSRChain.
func TestMain(m *testing.M) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	sync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)
	os.Exit(m.Run())
}
