package wire

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/observability"
)

func TestSignalerLigneIllisible_CompteParTitre(t *testing.T) {
	const compteur = "wire_rows_scan_failed_total"
	avant := observability.LoadCounterT("halo_5", compteur)
	signalerLigneIllisible(context.Background(), "halo_5", "test", errors.New("ligne illisible"))
	if d := observability.LoadCounterT("halo_5", compteur) - avant; d != 1 {
		t.Fatalf("compteur +%d, want +1", d)
	}
}
