// Package service — xbox_auth_service_test.go : tests XboxSSOLinkStrategy.
package service_test

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/service"
)

func newXboxStore(t *testing.T) *userstore.Store {
	t.Helper()
	return userstore.NewStore(filepath.Join(t.TempDir(), "users.json"))
}

func TestXboxSSOLinkStrategy_FirstLogin_CreatesUser(t *testing.T) {
	users := newXboxStore(t)
	s := service.NewXboxSSOLinkStrategy(users)

	sess := &domain.SessionData{}
	attempt := &auth.Attempt{
		Gamertag: "Spartan42",
		XUID:     "xuid-spartan-42",
	}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("OnAuthSuccess: %v", err)
	}

	// User créé.
	created, err := users.GetByXUID("xuid-spartan-42")
	if err != nil {
		t.Fatalf("user pas créé : %v", err)
	}
	if created.Role != domain.RoleUser {
		t.Errorf("role = %q, want user", created.Role)
	}
	if created.Gamertag != "Spartan42" {
		t.Errorf("gamertag = %q, want Spartan42", created.Gamertag)
	}

	// Session wirée.
	if sess.Username == nil || *sess.Username != "spartan42" {
		t.Errorf("Username = %v, want spartan42 (slug)", sess.Username)
	}
	if sess.Role == nil || *sess.Role != "user" {
		t.Errorf("Role = %v, want user", sess.Role)
	}
	if sess.CurrentPlayerSlug == nil || *sess.CurrentPlayerSlug != "Spartan42" {
		t.Errorf("CurrentPlayerSlug = %v, want Spartan42 (gamertag original)", sess.CurrentPlayerSlug)
	}
	if sess.LinkedHaloIdentity == nil || sess.LinkedHaloIdentity.XUID != "xuid-spartan-42" {
		t.Errorf("LinkedHaloIdentity = %v", sess.LinkedHaloIdentity)
	}
}

func TestXboxSSOLinkStrategy_ExistingUser_AuthenticatesAndTouchesLastLogin(t *testing.T) {
	users := newXboxStore(t)
	// Pré-créer un user via CreateFromXbox.
	pre, _ := users.CreateFromXbox("Spartan42", "xuid-spartan-42")
	if pre.LastLoginAt != "" {
		t.Fatal("LastLoginAt devrait être vide juste après création")
	}

	s := service.NewXboxSSOLinkStrategy(users)
	sess := &domain.SessionData{}
	attempt := &auth.Attempt{
		Gamertag: "Spartan42",
		XUID:     "xuid-spartan-42",
	}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("OnAuthSuccess: %v", err)
	}

	// LastLoginAt touché.
	updated, _ := users.GetByXUID("xuid-spartan-42")
	if updated.LastLoginAt == "" {
		t.Error("LastLoginAt devrait être touché pour un user existant")
	}

	// Session wirée pareil.
	if sess.Username == nil || *sess.Username != "spartan42" {
		t.Errorf("Username = %v, want spartan42", sess.Username)
	}
}

func TestXboxSSOLinkStrategy_MissingXUID_ReturnsError(t *testing.T) {
	users := newXboxStore(t)
	s := service.NewXboxSSOLinkStrategy(users)

	sess := &domain.SessionData{}
	attempt := &auth.Attempt{Gamertag: "Spartan42", XUID: ""}

	err := s.OnAuthSuccess(context.Background(), attempt, sess)
	if err == nil {
		t.Fatal("attendu erreur pour XUID vide, got nil")
	}
}

func TestXboxSSOLinkStrategy_MissingGamertag_ReturnsError(t *testing.T) {
	users := newXboxStore(t)
	s := service.NewXboxSSOLinkStrategy(users)

	sess := &domain.SessionData{}
	attempt := &auth.Attempt{Gamertag: "", XUID: "xuid-1"}

	err := s.OnAuthSuccess(context.Background(), attempt, sess)
	if err == nil {
		t.Fatal("attendu erreur pour gamertag vide, got nil")
	}
}

func TestXboxSSOLinkStrategy_CollisionWithPasswordUser_FallbackXbox(t *testing.T) {
	users := newXboxStore(t)
	// Pré-créer un user password avec slug "alice".
	_, _ = users.Create("alice", "Pa55w0rd!", domain.RoleUser)

	s := service.NewXboxSSOLinkStrategy(users)
	sess := &domain.SessionData{}
	attempt := &auth.Attempt{
		Gamertag: "Alice",
		XUID:     "xuid-alice-xbox",
	}

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("OnAuthSuccess collision: %v", err)
	}

	// User xbox créé avec suffixe.
	created, err := users.GetByXUID("xuid-alice-xbox")
	if err != nil {
		t.Fatalf("user xbox pas créé : %v", err)
	}
	if created.Username != "alice_xbox" {
		t.Errorf("username = %q, want alice_xbox (fallback)", created.Username)
	}

	// Session pointe vers le user xbox, pas le password.
	if sess.Username == nil || *sess.Username != "alice_xbox" {
		t.Errorf("session Username = %v, want alice_xbox", sess.Username)
	}
}
