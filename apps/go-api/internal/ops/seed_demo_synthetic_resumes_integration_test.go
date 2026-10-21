//go:build integration

package ops

// seed_demo_synthetic_resumes_integration_test.go — LA DÉMO SYNTHÉTIQUE ÉCRIT LES RÉSUMÉS PAR
// MATCH DE SON JOURNAL (ADR 0039). Le seed insère le kill-feed en SQL, hors de l'écrivain des
// passes : sans l'appel à `persist.PersistLatestKillPassSummaries` une démo n'aurait aucune
// tête, et un lecteur de résumés la montrerait vide.

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/sync/testutil"
)

func TestSeedDemoSynthetic_ResumesCoherents(t *testing.T) {
	setSyntheticProviders()
	out := t.TempDir()
	if _, err := SeedDemoSynthetic(context.Background(), SyntheticDemoOptions{OutDir: out, ServiceTag: "DEMO"}); err != nil {
		t.Fatalf("SeedDemoSynthetic: %v", err)
	}
	shared := openRO(t, filepath.Join(out, "warehouse", "shared_matches_v2.duckdb"))
	defer shared.Close()

	var matchsDuJournal, matchsAvecTete int
	if err := shared.QueryRow(`SELECT COUNT(DISTINCT match_id) FROM match_kill_events`).Scan(&matchsDuJournal); err != nil {
		t.Fatalf("matchs du journal : %v", err)
	}
	if err := shared.QueryRow(`SELECT COUNT(*) FROM match_kill_heads_latest`).Scan(&matchsAvecTete); err != nil {
		t.Fatalf("têtes : %v", err)
	}
	if matchsDuJournal == 0 {
		t.Fatal("la démo n'a aucun match au journal : le test ne mesure rien")
	}
	if matchsAvecTete != matchsDuJournal {
		t.Fatalf("%d match(s) au journal, %d tête(s) : chaque match semé doit avoir la sienne",
			matchsDuJournal, matchsAvecTete)
	}
	// Chaque tête résume la passe courante de son match, S1 en compte exactement les morts.
	if n := testutil.ExigerCoherenceDesResumes(t, shared); n != matchsDuJournal {
		t.Fatalf("%d match(s) vérifié(s), attendu %d", n, matchsDuJournal)
	}
}
