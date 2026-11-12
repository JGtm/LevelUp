package duckdb

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/platform/dblease"
)

// L'échec d'acquisition du writer est rendu tel quel : le handler reconnaît dblease.ErrDBLocked pour répondre
// 503 + Retry-After, et aucune transaction n'est ouverte.
func TestMediaTxOpener_EchecDAcquisitionRenduTelQuel(t *testing.T) {
	locked := errors.Join(errors.New("lease occupé"), dblease.ErrDBLocked)
	o := NewMediaTxOpener(nil, func() (*dblease.LeasedWriter, error) { return nil, locked })
	tx, err := o.BeginMediaTx(context.Background(), "media_like")
	if tx != nil || !errors.Is(err, dblease.ErrDBLocked) {
		t.Fatalf("tx=%v err=%v, attendu aucune transaction et ErrDBLocked", tx, err)
	}
}
