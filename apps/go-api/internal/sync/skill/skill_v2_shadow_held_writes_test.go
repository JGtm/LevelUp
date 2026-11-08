//go:build cgo

package skill

// skill_v2_shadow_held_writes_test.go — un cycle dont une écriture canonical échoue (filigrane
// tenu, match sans note) ne rend une erreur que pour l'appelant qui l'a demandée
// (WithErrorOnHeldWrites) : le rejeu complet n'a pas de cycle suivant pour retenter.

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunLUSRV2Shadow_HeldWrites_ErreurSeulementAvecLOption(t *testing.T) {
	sharedDB := openShadowTestDB(t)
	seedCanonical2v2(t, sharedDB, "m_held", time.Date(2025, 8, 1, 14, 0, 0, 0, time.UTC))
	ctx := context.Background()
	access := newPinnedSharedAccessor(sharedDB)

	// Player DB qui refuse toute écriture de note : l'écriture canonical échoue, le filigrane
	// reste tenu, donc chaque cycle retrouve le même match à retenter.
	brokenPlayerDB := openCanonicalPlayerTestDBWithCheck(t, "FALSE")

	processed, err := RunLUSRV2Shadow(ctx, brokenPlayerDB, access, "owner")
	if err != nil {
		t.Fatalf("sans l'option, un cycle à écriture tenue reste nil (retry au cycle suivant) : %v", err)
	}
	if processed != 0 {
		t.Errorf("sans l'option : processed = %d, want 0", processed)
	}

	processed, err = RunLUSRV2Shadow(ctx, brokenPlayerDB, access, "owner", WithErrorOnHeldWrites())
	if !errors.Is(err, ErrCanonicalWriteHeld) {
		t.Fatalf("avec l'option : err = %v, want ErrCanonicalWriteHeld", err)
	}
	if processed != 0 {
		t.Errorf("avec l'option : processed = %d, want 0 (le décompte reste rendu avec l'erreur)", processed)
	}

	// Écriture saine : l'option ne change rien, le match tenu passe et le cycle rend nil.
	goodPlayerDB := openCanonicalPlayerTestDB(t)
	processed, err = RunLUSRV2Shadow(ctx, goodPlayerDB, access, "owner", WithErrorOnHeldWrites())
	if err != nil {
		t.Fatalf("écriture saine avec l'option : %v", err)
	}
	if processed != 1 {
		t.Errorf("écriture saine : processed = %d, want 1", processed)
	}
}

func TestRunLUSRV2ShadowOwnerOnly_HeldWrites_ErreurAvecLOption(t *testing.T) {
	sharedDB := openShadowTestDB(t)
	seedCanonical2v2(t, sharedDB, "m_held_oo", time.Date(2025, 8, 2, 14, 0, 0, 0, time.UTC))
	brokenPlayerDB := openCanonicalPlayerTestDBWithCheck(t, "FALSE")

	_, err := RunLUSRV2ShadowOwnerOnly(context.Background(), brokenPlayerDB, newPinnedSharedAccessor(sharedDB),
		"owner", WithErrorOnHeldWrites())
	if !errors.Is(err, ErrCanonicalWriteHeld) {
		t.Fatalf("owner-only avec l'option : err = %v, want ErrCanonicalWriteHeld", err)
	}
}
