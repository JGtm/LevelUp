//go:build integration

package sync_test

// schema_index_retired_log_test.go — le retrait convergent d'un index par
// EnsurePlayerSchema est JOURNALISÉ (revue adversariale du lot B3, constat C6 ; lot B-C5
// du backlog 2026-09-26).
//
// Contrat de sync/schema.go : « toute action réelle est journalisée (schema_drift_healed) ».
// Le soin retire les index ART retirés (MSR, PSA) qu'un binaire plus ancien aurait
// recréés ; ce retrait était SILENCIEUX, car schemadrift.Report ne comparait que les
// créations. Il doit émettre un WARN schema_drift_healed par index réellement retiré, et
// RIEN sur une base à jour.

import (
	"context"
	"strings"
	"testing"

	"levelup/go-api/internal/sync"
)

// retraitsJournalises : lignes schema_drift_healed qui désignent le retrait de l'index.
func retraitsJournalises(lines []string, index string) []string {
	var out []string
	for _, l := range lines {
		if strings.Contains(l, "object="+index) && strings.Contains(l, "action=dropped") {
			out = append(out, l)
		}
	}
	return out
}

func TestEnsurePlayerSchema_RetraitIndexJournalise(t *testing.T) {
	db := freshMigratedPlayerDB(t)
	// Binaire ancien : il recrée un index MSR et un index PSA retirés.
	for _, stmt := range []string{retiredMSRIndexes["idx_msr_playlist"], retiredPSAIndexes["idx_psa_match"]} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("recréation (binaire ancien) : %v", err)
		}
	}
	cap := captureDriftWarnings(t)
	if err := sync.EnsurePlayerSchema(context.Background(), db); err != nil {
		t.Fatalf("EnsurePlayerSchema : %v", err)
	}
	lines := cap.snapshot()
	for _, idx := range []string{"idx_msr_playlist", "idx_psa_match"} {
		if got := retraitsJournalises(lines, idx); len(got) != 1 {
			t.Errorf("retrait de %s : %d ligne(s) schema_drift_healed action=dropped, attendu 1 ; "+
				"WARN capturés = %v", idx, len(got), lines)
		}
	}
	// Seuls les deux retraits réels sont journalisés : aucun autre objet.
	if len(lines) != 2 {
		t.Errorf("WARN schema_drift_healed = %d, attendu 2 (les deux retraits réels) : %v", len(lines), lines)
	}
}

func TestEnsurePlayerSchema_BaseAJour_AucunRetraitJournalise(t *testing.T) {
	db := freshMigratedPlayerDB(t)
	cap := captureDriftWarnings(t)
	if err := sync.EnsurePlayerSchema(context.Background(), db); err != nil {
		t.Fatalf("EnsurePlayerSchema : %v", err)
	}
	if lines := cap.snapshot(); len(lines) != 0 {
		t.Errorf("base à jour : aucun WARN attendu, reçu %v", lines)
	}
}
