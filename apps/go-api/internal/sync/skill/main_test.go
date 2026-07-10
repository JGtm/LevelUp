package skill

import (
	"os"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/skillchain"
)

// TestMain câble le classifier LUSR title-owned avant les tests du package skill (sinon
// GetLUSRChain / ComputeSkillRatingsBatch paniquent — fail-loud par design, MT-15). Miroir
// du TestMain sync (migration_provider_test.go) pour le package skill extrait (K3c).
func TestMain(m *testing.M) {
	SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)
	os.Exit(m.Run())
}
