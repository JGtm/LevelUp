// Package session — store_limits_test.go : plafond de taille d'une session
// sérialisée et durée de vie courte des sessions sans identité authentifiée.
package session_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/session"
)

func TestSave_RefusesSessionOverSizeLimit(t *testing.T) {
	store, _, dir := newClockStore(t)
	sess := store.New()
	sess.PendingInviteCode = strings.Repeat("x", session.MaxSessionBytes)

	buf := captureWarnLogs(t)
	err := store.Save(sess)
	if !errors.Is(err, session.ErrSessionTooLarge) {
		t.Fatalf("Save d'une session > %d octets : err = %v, attendu ErrSessionTooLarge", session.MaxSessionBytes, err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, sess.SessionID+".json")); !os.IsNotExist(statErr) {
		t.Errorf("un refus ne doit rien écrire, stat err = %v", statErr)
	}
	if !strings.Contains(buf.String(), "session") {
		t.Errorf("le refus doit être journalisé, journal = %q", buf.String())
	}
}

func TestTouch_RefusesSessionOverSizeLimit(t *testing.T) {
	store, _, _ := newClockStore(t)
	sess := store.New()
	sess.PendingInviteCode = strings.Repeat("x", session.MaxSessionBytes)
	if err := store.Touch(sess); !errors.Is(err, session.ErrSessionTooLarge) {
		t.Fatalf("Touch d'une session > plafond : err = %v, attendu ErrSessionTooLarge", err)
	}
}

func TestSave_AcceptsAuthenticatedSessionWithHaloTokens(t *testing.T) {
	store, _, _ := newClockStore(t)
	sess := store.New()
	user := "joueur"
	sess.Username = &user
	sess.HaloTokens = &domain.HaloTokens{
		SpartanToken:   strings.Repeat("s", 2500),
		ClearanceToken: strings.Repeat("c", 2500),
	}
	if err := store.Save(sess); err != nil {
		t.Fatalf("session authentifiée avec jetons Halo (≈5 Kio) refusée : %v", err)
	}
}

func TestAnonymousSession_ExpiresAfter24h(t *testing.T) {
	store, clock, _ := newClockStore(t)
	anon := store.New()
	anon.Locale = "en" // session « significative » : persistée
	if err := store.Save(anon); err != nil {
		t.Fatalf("Save : %v", err)
	}

	clock.now = clock.now.Add(23 * time.Hour)
	if store.Load(context.Background(), anon.SessionID) == nil {
		t.Fatal("une session anonyme vue il y a 23 h doit être chargée")
	}

	clock.now = clock.now.Add(2 * time.Hour) // 25 h après le dernier passage
	if store.Load(context.Background(), anon.SessionID) != nil {
		t.Fatal("une session anonyme vue il y a 25 h doit être expirée (TTL anonyme 24 h)")
	}
}

func TestAuthenticatedSession_KeepsFullTTL(t *testing.T) {
	store, clock, _ := newClockStore(t)
	for name, mark := range map[string]func(*domain.SessionData){
		"compte local": func(s *domain.SessionData) { u := "joueur"; s.Username = &u },
		"device-code":  func(s *domain.SessionData) { s.AuthReady = true },
		"identite Xbox": func(s *domain.SessionData) {
			s.LinkedHaloIdentity = &domain.HaloIdentity{Gamertag: "JGtm", XUID: "1"}
		},
	} {
		sess := store.New()
		mark(sess)
		if err := store.Save(sess); err != nil {
			t.Fatalf("%s : Save : %v", name, err)
		}
		clock.now = clock.now.Add(3 * 24 * time.Hour)
		if store.Load(context.Background(), sess.SessionID) == nil {
			t.Errorf("%s : une session authentifiée vue il y a 3 jours doit rester valide (TTL %v)", name, session.DefaultTTL)
		}
		clock.now = clock.now.Add(-3 * 24 * time.Hour)
	}
}

func TestPurgeExpired_RemovesAnonymousSessionsAfter24h(t *testing.T) {
	store, clock, _ := newClockStore(t)
	anon := store.New()
	anon.Locale = "en"
	authed := store.New()
	u := "joueur"
	authed.Username = &u
	for _, s := range []*domain.SessionData{anon, authed} {
		if err := store.Save(s); err != nil {
			t.Fatalf("Save : %v", err)
		}
	}
	clock.now = clock.now.Add(25 * time.Hour)
	if removed := store.PurgeExpired(); removed != 1 {
		t.Fatalf("PurgeExpired = %d, attendu 1 (la session anonyme seule)", removed)
	}
	if store.Load(context.Background(), authed.SessionID) == nil {
		t.Error("la session authentifiée ne doit pas être purgée à 25 h")
	}
}
