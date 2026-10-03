package main

import (
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// sync_report_test.go — le verdict rendu par la CLI. Ces tests rougissent si
// reportSyncResult redevient un `fmt.Printf` suivi d'un `return nil` : c'est exactement ce
// qui faisait sortir en code 0, sous le mot « OK », une passe arrêtée par un 429.

func TestReportSyncResult_Succes_AucuneErreur(t *testing.T) {
	var out strings.Builder
	r := &domain.SyncResult{MatchesInserted: 12, MatchesSkipped: 3, DurationSeconds: 4.5}

	if err := reportSyncResult(&out, "delta", "Nuzzles", r); err != nil {
		t.Fatalf("attendu nil sur une passe sans erreur, obtenu %v", err)
	}
	line := out.String()
	for _, want := range []string{"sync delta SUCCESS:", "gamertag=Nuzzles", "inserted=12", "status=success"} {
		if !strings.Contains(line, want) {
			t.Errorf("sortie %q sans %q", line, want)
		}
	}
	if strings.Contains(line, "first_error=") {
		t.Errorf("aucune erreur ne doit être annoncée: %q", line)
	}
}

func TestReportSyncResult_Failure_ErreurEtPremiereCause(t *testing.T) {
	var out strings.Builder
	r := &domain.SyncResult{}
	r.AddError("historique interrompu à start=225: HTTP 429")

	err := reportSyncResult(&out, "full", "Nuzzles", r)
	if err == nil {
		t.Fatal("attendu une erreur sur un statut failure, obtenu nil")
	}
	if !strings.Contains(err.Error(), "historique interrompu à start=225") {
		t.Errorf("l'erreur rendue doit porter la première cause: %v", err)
	}
	line := out.String()
	if !strings.Contains(line, "sync full FAILURE:") {
		t.Errorf("sortie %q sans le verdict FAILURE", line)
	}
	if !strings.Contains(line, "first_error=historique interrompu à start=225") {
		t.Errorf("sortie %q sans first_error", line)
	}
}

func TestReportSyncResult_PartialSuccess_RendUneErreur(t *testing.T) {
	var out strings.Builder
	r := &domain.SyncResult{MatchesInserted: 7}
	r.AddError("historique interrompu à start=50: HTTP 429")

	if err := reportSyncResult(&out, "full", "Nuzzles", r); err == nil {
		t.Fatal("une passe incomplète doit sortir en erreur, même avec des insertions")
	}
	if !strings.Contains(out.String(), "sync full PARTIAL_SUCCESS:") {
		t.Errorf("sortie %q sans le verdict PARTIAL_SUCCESS", out.String())
	}
}

func TestReportSyncResult_ResultatNil_RendUneErreur(t *testing.T) {
	var out strings.Builder
	if err := reportSyncResult(&out, "delta", "Nuzzles", nil); err == nil {
		t.Fatal("un résultat absent est une erreur, pas un succès muet")
	}
}
