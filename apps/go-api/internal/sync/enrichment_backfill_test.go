package sync

import (
	"context"
	"errors"
	"testing"
)

// TestBackfillEnrichmentFromShared_RejectsMissingSessionOptions : un appelant qui oublie la
// règle de session (valeur zéro) est refusé avant toute lecture ou écriture.
func TestBackfillEnrichmentFromShared_RejectsMissingSessionOptions(t *testing.T) {
	report, err := BackfillEnrichmentFromShared(context.Background(), nil, nil,
		EnrichmentBackfillPlayer{XUID: "x1", FriendGamertags: []string{"Ami"}}, false)
	if !errors.Is(err, ErrSessionOptionsMissing) {
		t.Fatalf("err = %v, attendu ErrSessionOptionsMissing", err)
	}
	if report.SessionsAssigned != 0 || report.BaselineRowsCreated != 0 || len(report.Errors) != 0 {
		t.Fatalf("aucune étape ne doit tourner : %+v", report)
	}
}
