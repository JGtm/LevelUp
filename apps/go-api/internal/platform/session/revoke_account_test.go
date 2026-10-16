package session_test

import (
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/session"
)

func saveSessionOf(t *testing.T, store *session.Store, username string) *domain.SessionData {
	t.Helper()
	sess := store.New()
	if username != "" {
		u := username
		sess.Username = &u
	}
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save : %v", err)
	}
	return sess
}

// TestRevokeAccount_RevokesEveryOtherSessionOfTheAccount : toutes les sessions du compte
// sauf celle à garder sont révoquées (fichier supprimé, réécriture refusée) ; celles des
// autres comptes et les sessions anonymes ne sont pas touchées.
func TestRevokeAccount_RevokesEveryOtherSessionOfTheAccount(t *testing.T) {
	store := newTestStore(t)
	keep := saveSessionOf(t, store, "jgtm")
	other1 := saveSessionOf(t, store, "jgtm")
	other2 := saveSessionOf(t, store, "JGtm")
	neighbour := saveSessionOf(t, store, "voisin")
	anonymous := saveSessionOf(t, store, "")
	inFlight := store.Load(t.Context(), other1.SessionID)

	n, err := store.RevokeAccount("jgtm", keep.SessionID)
	if err != nil {
		t.Fatalf("RevokeAccount : %v", err)
	}
	if n != 2 {
		t.Errorf("%d session(s) révoquée(s), attendu 2", n)
	}
	for _, s := range []*domain.SessionData{other1, other2} {
		if store.Load(t.Context(), s.SessionID) != nil {
			t.Errorf("session %s du compte encore chargeable", s.SessionID)
		}
		if _, revoked := store.Revoked(s.SessionID); !revoked {
			t.Errorf("session %s du compte non retenue comme révoquée", s.SessionID)
		}
	}
	if err := store.Save(inFlight); !errors.Is(err, session.ErrSessionRevoked) {
		t.Errorf("Save d'une copie en vol = %v, attendu ErrSessionRevoked", err)
	}
	for name, s := range map[string]*domain.SessionData{"gardée": keep, "autre compte": neighbour, "anonyme": anonymous} {
		if store.Load(t.Context(), s.SessionID) == nil {
			t.Errorf("session %s supprimée à tort", name)
		}
	}
}

// TestRevokeAccount_EmptyUsernameRevokesNothing : un nom vide ne désigne aucun compte (les
// sessions anonymes n'en portent pas).
func TestRevokeAccount_EmptyUsernameRevokesNothing(t *testing.T) {
	store := newTestStore(t)
	anonymous := saveSessionOf(t, store, "")
	n, err := store.RevokeAccount("", "")
	if err != nil || n != 0 {
		t.Fatalf("RevokeAccount(\"\") = (%d, %v), attendu (0, nil)", n, err)
	}
	if store.Load(t.Context(), anonymous.SessionID) == nil {
		t.Error("session anonyme supprimée")
	}
}
