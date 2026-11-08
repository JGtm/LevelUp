//go:build cgo

package skill

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// WithMatchFailureCounter compte le match dont l'écriture de la note échoue (journalisé sans arrêter
// la boucle) ; un rejeu sain ne compte rien, et sans l'option le rejeu ne change pas (aucune erreur).
func TestRunLUSRV2Shadow_MatchFailureCounter(t *testing.T) {
	sharedDB := openShadowTestDB(t)
	seedCanonical2v2(t, sharedDB, "m_old", time.Date(2025, 9, 1, 12, 0, 0, 0, time.UTC))
	seedCanonical2v2(t, sharedDB, "m_new", time.Date(2025, 9, 1, 13, 0, 0, 0, time.UTC))
	ctx := context.Background()

	broken := openCanonicalPlayerTestDBWithCheck(t, "FALSE")
	if _, err := RunLUSRV2Shadow(ctx, broken, newPinnedSharedAccessor(sharedDB), "owner"); err != nil {
		t.Fatalf("sans option : %v", err)
	}

	var failures atomic.Int64
	if _, err := RunLUSRV2Shadow(ctx, broken, newPinnedSharedAccessor(sharedDB), "owner",
		WithMatchFailureCounter(&failures)); err != nil {
		t.Fatalf("base refusant les notes : %v", err)
	}
	if got := failures.Load(); got != 1 {
		t.Fatalf("échecs comptés = %d, attendu 1 (m_old échoue, m_new est sauté, groupe tenu)", got)
	}

	var healthyFailures atomic.Int64
	healthy := openCanonicalPlayerTestDB(t)
	processed, err := RunLUSRV2Shadow(ctx, healthy, newPinnedSharedAccessor(sharedDB), "owner",
		WithMatchFailureCounter(&healthyFailures))
	if err != nil || processed != 2 {
		t.Fatalf("rejeu sain : processed=%d err=%v, attendu 2, nil", processed, err)
	}
	if got := healthyFailures.Load(); got != 0 {
		t.Fatalf("échecs comptés = %d sur un rejeu sain, attendu 0", got)
	}
}
