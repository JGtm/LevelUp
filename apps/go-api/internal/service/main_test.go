package service

import (
	"os"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/skillchain"
	"levelup/go-api/internal/sync"
)

// TestMain câble le classifier LUSR title-owned avant les tests du package service
// (MT-15). Certains tests — match_history_placement (applyLUSRPlacements /
// applyMatchPlacements) — atteignent sync.GetLUSRChain, qui panique si le
// classifier n'est pas posé (fail-loud par design). Mirroir du boot serveur.
func TestMain(m *testing.M) {
	sync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)
	os.Exit(m.Run())
}
