package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// Une bascule de la base partagée pendant le recalcul remonte telle quelle (enveloppée) :
// le handler la reconnaît et répond 503 au lieu de 500.
func TestMatchExclusionService_SharedSwapTimeoutIsWrapped(t *testing.T) {
	svc, _, rec := newSvc(t)
	rec.returnTo = fmt.Errorf("MatchRecomputer.RecomputeAfterExclusion: %w", sharedprovider.ErrSwapTimeout)
	err := svc.SetExclusion(context.Background(), "m1", true)
	if !errors.Is(err, sharedprovider.ErrSwapTimeout) {
		t.Fatalf("err = %v, attendu une erreur qui enveloppe sharedprovider.ErrSwapTimeout", err)
	}
}
