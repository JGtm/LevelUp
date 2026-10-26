// Package session — touch_after_test.go : la persistance de fin de requête ne réécrit
// jamais la copie chargée d'une session que la requête n'a pas modifiée.
package session_test

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/session"
)

func mustSave(t *testing.T, store *session.Store, sess *domain.SessionData) {
	t.Helper()
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save : %v", err)
	}
}

// TestTouchAfter_SessionInchangee_GardeLeContenuEcritEntreTemps : seul last_seen_at est
// rafraîchi, sur le contenu que porte le fichier (joueur courant posé par une requête
// concurrente), pas sur la copie chargée.
func TestTouchAfter_SessionInchangee_GardeLeContenuEcritEntreTemps(t *testing.T) {
	store, clock, dir := newClockStore(t)
	ancien, nouveau := "callousss", "alice_gt"
	charge := store.New()
	charge.CurrentPlayerSlug = &ancien
	mustSave(t, store, charge)
	loaded := session.FingerprintOf(charge)

	concurrente := *charge
	concurrente.CurrentPlayerSlug = &nouveau
	mustSave(t, store, &concurrente)

	clock.now = clock.now.Add(10 * time.Minute)
	if err := store.TouchAfter(charge, loaded); err != nil {
		t.Fatalf("TouchAfter : %v", err)
	}
	got := onDisk(t, dir, charge)
	if got.CurrentPlayerSlug == nil || *got.CurrentPlayerSlug != nouveau {
		t.Fatalf("joueur courant sur disque = %v, attendu %q", got.CurrentPlayerSlug, nouveau)
	}
	if got.LastSeenAt != clock.now.Unix() {
		t.Errorf("last_seen_at = %d, attendu %d", got.LastSeenAt, clock.now.Unix())
	}
}

// TestTouchAfter_SessionModifiee_EcriteEntiere : une session modifiée par la requête est
// écrite telle quelle, comme par Touch.
func TestTouchAfter_SessionModifiee_EcriteEntiere(t *testing.T) {
	store, _, dir := newClockStore(t)
	sess := store.New()
	mustSave(t, store, sess)
	loaded := session.FingerprintOf(sess)
	sess.Locale = "en"
	if err := store.TouchAfter(sess, loaded); err != nil {
		t.Fatalf("TouchAfter : %v", err)
	}
	if got := onDisk(t, dir, sess).Locale; got != "en" {
		t.Errorf("locale sur disque = %q, attendu en", got)
	}
}

// TestTouchAfter_FichierDisparu_NonRecree : une session supprimée pendant la requête
// (purge, suppression de compte) n'est pas recréée par une requête qui ne l'a pas modifiée.
func TestTouchAfter_FichierDisparu_NonRecree(t *testing.T) {
	store, clock, _ := newClockStore(t)
	sess := store.New()
	mustSave(t, store, sess)
	loaded := session.FingerprintOf(sess)
	if err := store.Delete(sess.SessionID); err != nil {
		t.Fatalf("Delete : %v", err)
	}
	clock.now = clock.now.Add(10 * time.Minute)
	if err := store.TouchAfter(sess, loaded); err != nil {
		t.Fatalf("TouchAfter : %v", err)
	}
	if got := store.Load(t.Context(), sess.SessionID); got != nil {
		t.Errorf("session recréée par TouchAfter : %+v", got)
	}
}
