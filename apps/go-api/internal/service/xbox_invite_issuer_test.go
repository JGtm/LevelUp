package service_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/service"
)

// lockedInviteRig : instance verrouillée, invitation émise par le compte « owner ».
func lockedInviteRig(t *testing.T) (*service.XboxSSOLinkStrategy, *userstore.Store, *domain.InviteCode) {
	t.Helper()
	dir := t.TempDir()
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	invites := userstore.NewInviteStore(filepath.Join(dir, "invites.json"), users)
	if _, err := users.CreateFromXbox("Owner", "owner-x"); err != nil {
		t.Fatalf("CreateFromXbox owner : %v", err)
	}
	inv, err := invites.Generate("owner", 7, "")
	if err != nil {
		t.Fatalf("Generate : %v", err)
	}
	s := service.NewXboxSSOLinkStrategy(users).
		WithInstanceLock(func() bool { return true }).
		WithInviteStore(invites)
	return s, users, inv
}

// TestXboxSSO_InviteFromDeletedIssuer_LockedRefused : l'émetteur a été supprimé :
// l'invitation ne lève plus le verrou, aucun compte n'est créé.
func TestXboxSSO_InviteFromDeletedIssuer_LockedRefused(t *testing.T) {
	s, users, inv := lockedInviteRig(t)
	if err := users.Delete("owner"); err != nil {
		t.Fatalf("Delete owner : %v", err)
	}
	sess := &domain.SessionData{PendingInviteCode: inv.Code}
	err := s.OnAuthSuccess(context.Background(), &auth.Attempt{XUID: "newcomer-x", Gamertag: "Newcomer"}, sess)
	if !errors.Is(err, auth.ErrInstanceLocked) {
		t.Fatalf("err = %v, attendu ErrInstanceLocked", err)
	}
	if _, err := users.GetByXUID("newcomer-x"); err == nil {
		t.Error("compte créé avec l'invitation d'un émetteur supprimé")
	}
}

// TestXboxSSO_InviteSingleUseUnderConcurrency : six identités inconnues présentent le
// même code en même temps sur une instance verrouillée : un seul compte créé.
func TestXboxSSO_InviteSingleUseUnderConcurrency(t *testing.T) {
	s, users, inv := lockedInviteRig(t)
	const n = 6
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sess := &domain.SessionData{PendingInviteCode: inv.Code}
			gt := "Racer" + string(rune('A'+i))
			_ = s.OnAuthSuccess(context.Background(), &auth.Attempt{XUID: "racer-x-" + gt, Gamertag: gt}, sess)
		}()
	}
	wg.Wait()
	list, err := users.List()
	if err != nil {
		t.Fatalf("List : %v", err)
	}
	if created := len(list) - 1; created != 1 {
		t.Fatalf("comptes créés avec un seul code : %d, attendu 1", created)
	}
}
