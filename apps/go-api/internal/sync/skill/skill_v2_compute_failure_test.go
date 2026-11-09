package skill

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
)

// Le classement n'a d'effet que branché sur le chemin du calcul : aucune donnée de test ne fait
// diverger le solveur de façon stable, le branchement se vérifie donc sur la source.
func TestProcessOneShadowMatch_ClasseLEchecDeCalcul(t *testing.T) {
	src, err := os.ReadFile("skill_v2_shadow.go")
	if err != nil {
		t.Fatalf("lecture de la source: %v", err)
	}
	_, afterLog, found := strings.Cut(string(src), `"LUSR v2 shadow: compute échoué"`)
	if !found {
		t.Fatal("journalisation de l'échec de calcul introuvable")
	}
	branch, _, found := strings.Cut(afterLog, "return\n")
	if !found {
		t.Fatal("fin de la branche d'échec de calcul introuvable")
	}
	if !strings.Contains(branch, "c.noteComputeFailure(err, s)") || strings.Contains(branch, "c.noteMatchFailure()") {
		t.Fatalf("la branche d'échec de calcul doit passer par noteComputeFailure :\n%s", branch)
	}
}

// Une non-convergence du solveur, enveloppée comme sur le chemin réel, ne rend pas le rejeu
// partiel ; toute autre erreur de calcul, si.
func TestNoteComputeFailure(t *testing.T) {
	cases := []struct {
		name             string
		err              error
		wantFailures     int64
		wantNotConverged int
	}{
		{"non-convergence", fmt.Errorf("UpdateTwoTeamWithCountsEP: %w en 500 itérations", skillv2.ErrNotConverged), 0, 1},
		{"autre erreur", errors.New("LoadState: base indisponible"), 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var failures atomic.Int64
			var s shadowRunStats
			c := shadowRunContext{matchFailures: &failures}

			c.noteComputeFailure(tc.err, &s)

			if got := failures.Load(); got != tc.wantFailures {
				t.Errorf("échecs comptés = %d, attendu %d", got, tc.wantFailures)
			}
			if s.skippedNotConverged != tc.wantNotConverged {
				t.Errorf("non convergés = %d, attendu %d", s.skippedNotConverged, tc.wantNotConverged)
			}
		})
	}
}
