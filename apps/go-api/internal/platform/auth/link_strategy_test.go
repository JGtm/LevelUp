// Package auth_test — link_strategy_test.go : tests PasswordLinkStrategy.
package auth_test

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
)

// mockUserLinker capture les appels LinkIdentity pour assertion.
type mockUserLinker struct {
	calls   []linkCall
	failErr error
}

type linkCall struct {
	username string
	gamertag string
	xuid     string
}

func (m *mockUserLinker) LinkIdentity(username, gamertag, xuid string) error {
	m.calls = append(m.calls, linkCall{username, gamertag, xuid})
	return m.failErr
}

func TestPasswordLinkStrategy_OnAuthSuccess_Success(t *testing.T) {
	mock := &mockUserLinker{}
	s := auth.NewPasswordLinkStrategy(mock)

	username := "alice"
	sess := &domain.SessionData{Username: &username}
	attempt := &auth.Attempt{Gamertag: "AliceGT", XUID: "xuid-alice"}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("OnAuthSuccess: %v", err)
	}

	if len(mock.calls) != 1 {
		t.Fatalf("LinkIdentity calls = %d, want 1", len(mock.calls))
	}
	got := mock.calls[0]
	if got.username != "alice" || got.gamertag != "AliceGT" || got.xuid != "xuid-alice" {
		t.Errorf("LinkIdentity call = %+v", got)
	}
	if sess.CurrentPlayerSlug == nil || *sess.CurrentPlayerSlug != "AliceGT" {
		t.Errorf("CurrentPlayerSlug = %v, want AliceGT", sess.CurrentPlayerSlug)
	}
}

func TestPasswordLinkStrategy_OnAuthSuccess_GamertagEmpty(t *testing.T) {
	mock := &mockUserLinker{}
	s := auth.NewPasswordLinkStrategy(mock)

	username := "alice"
	sess := &domain.SessionData{Username: &username}
	attempt := &auth.Attempt{Gamertag: "", XUID: ""}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("OnAuthSuccess gamertag vide: %v", err)
	}
	if len(mock.calls) != 0 {
		t.Errorf("LinkIdentity ne devrait pas être appelé si gamertag vide")
	}
}

func TestPasswordLinkStrategy_OnAuthSuccess_NoUsername(t *testing.T) {
	mock := &mockUserLinker{}
	s := auth.NewPasswordLinkStrategy(mock)

	sess := &domain.SessionData{Username: nil}
	attempt := &auth.Attempt{Gamertag: "AliceGT", XUID: "xuid-alice"}

	err := s.OnAuthSuccess(context.Background(), attempt, sess)
	if !errors.Is(err, auth.ErrSessionNotAuthenticated) {
		t.Errorf("err = %v, want ErrSessionNotAuthenticated", err)
	}
	if len(mock.calls) != 0 {
		t.Errorf("LinkIdentity ne devrait pas être appelé sans username")
	}
}

func TestPasswordLinkStrategy_OnAuthSuccess_LinkError(t *testing.T) {
	mock := &mockUserLinker{failErr: errors.New("disk full")}
	s := auth.NewPasswordLinkStrategy(mock)

	username := "alice"
	sess := &domain.SessionData{Username: &username}
	attempt := &auth.Attempt{Gamertag: "AliceGT", XUID: "xuid-alice"}

	err := s.OnAuthSuccess(context.Background(), attempt, sess)
	if err == nil {
		t.Fatal("attendu erreur du LinkIdentity, got nil")
	}
	// CurrentPlayerSlug ne doit PAS être set si LinkIdentity a échoué.
	if sess.CurrentPlayerSlug != nil {
		t.Errorf("CurrentPlayerSlug ne devrait pas être set si LinkIdentity a échoué")
	}
}
