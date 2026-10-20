package session_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/platform/session"
)

// TestRevoke_DeletesAndForbidsRewrite : une session révoquée n'existe plus et une
// copie chargée avant la révocation (requête concurrente) ne peut pas la réécrire.
func TestRevoke_DeletesAndForbidsRewrite(t *testing.T) {
	store := newTestStore(t)
	sess := store.New()
	u := "jgtm"
	sess.Username = &u
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save : %v", err)
	}
	inFlight := store.Load(t.Context(), sess.SessionID)

	if err := store.Revoke(sess.SessionID); err != nil {
		t.Fatalf("Revoke : %v", err)
	}
	if store.Load(t.Context(), sess.SessionID) != nil {
		t.Error("session encore chargeable après révocation")
	}
	if err := store.Touch(inFlight); !errors.Is(err, session.ErrSessionRevoked) {
		t.Errorf("Touch d'une copie en vol = %v, attendu ErrSessionRevoked", err)
	}
	if err := store.Save(inFlight); !errors.Is(err, session.ErrSessionRevoked) {
		t.Errorf("Save d'une copie en vol = %v, attendu ErrSessionRevoked", err)
	}
	if store.Load(t.Context(), sess.SessionID) != nil {
		t.Error("session ressuscitée par une requête concurrente")
	}
	if successor, revoked := store.Revoked(sess.SessionID); !revoked || successor != "" {
		t.Errorf("Revoked = (%q, %v), attendu (\"\", true)", successor, revoked)
	}
}

// TestRevoke_ForgottenAfterTTLAndBounded : la mémoire des révocations est bornée en
// temps et en taille.
func TestRevoke_ForgottenAfterTTLAndBounded(t *testing.T) {
	now := time.Now()
	store := session.NewStore(t.TempDir(), time.Hour, "test-secret-32-bytesXXXXXXXXXX",
		session.WithClock(func() time.Time { return now }))
	if err := store.Revoke("old-id"); err != nil {
		t.Fatalf("Revoke : %v", err)
	}
	now = now.Add(2 * time.Hour)
	if _, revoked := store.Revoked("old-id"); revoked {
		t.Error("révocation encore connue après son délai")
	}
	for i := range 10_050 {
		now = now.Add(time.Millisecond)
		if err := store.Revoke(fmt.Sprintf("id-%d", i)); err != nil {
			t.Fatalf("Revoke : %v", err)
		}
	}
	if _, revoked := store.Revoked("id-10049"); !revoked {
		t.Error("la révocation la plus récente a été oubliée")
	}
	if _, revoked := store.Revoked("id-0"); revoked {
		t.Error("la table des révocations n'est pas bornée")
	}
}

// TestRotate_NewIDSameStateOldRevokedWithSuccessor : nouvel identifiant, même état,
// ancien supprimé et révoqué avec le nouveau pour successeur.
func TestRotate_NewIDSameStateOldRevokedWithSuccessor(t *testing.T) {
	store := newTestStore(t)
	sess := store.New()
	sess.CurrentTitleSlug = "halo_5"
	u := "jgtm"
	sess.Username = &u
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save : %v", err)
	}
	oldID, err := store.Rotate(sess)
	if err != nil {
		t.Fatalf("Rotate : %v", err)
	}
	if oldID == sess.SessionID {
		t.Fatal("identifiant inchangé")
	}
	if store.Load(t.Context(), oldID) != nil {
		t.Error("ancienne session encore chargeable")
	}
	fresh := store.Load(t.Context(), sess.SessionID)
	if fresh == nil || fresh.CurrentTitleSlug != "halo_5" || fresh.Username == nil || *fresh.Username != "jgtm" {
		t.Errorf("état perdu à la rotation : %+v", fresh)
	}
	if successor, revoked := store.Revoked(oldID); !revoked || successor != sess.SessionID {
		t.Errorf("Revoked(ancien) = (%q, %v), attendu (%q, true)", successor, revoked, sess.SessionID)
	}
}
