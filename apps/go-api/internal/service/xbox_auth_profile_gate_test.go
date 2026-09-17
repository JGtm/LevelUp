// Package service_test — xbox_auth_profile_gate_test.go : le SSO Xbox ne met
// personne sous surveillance sans profil suivi (ADR 0035 D3).
//
// Le 2026-07-23, OnAuthSuccess écrivait trois registres — compte, credentials,
// entrée live du watcher — et jamais le quatrième, le profil. Deux heures plus
// tard le watcher soumettait un sync pour un joueur qui n'existait pour personne.
// Compte et credentials RESTENT créés (le refresh token est ce qui rendra le
// profil utilisable) ; seule la prise en charge live est conditionnée.
package service_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/service"
)

// ssoAttempt construit une tentative SSO réussie portant des tokens RTA.
func ssoAttempt() *auth.Attempt {
	return &auth.Attempt{
		Gamertag:             "InconnuAuBataillon",
		XUID:                 "2533274796795729",
		MicrosoftAccessToken: "ms-access-token",
		OAuthRefreshToken:    "refresh-token",
		XSTSRTAToken:         "xsts-rta-token",
		XSTSRTAUserHash:      "rta-user-hash",
		XSTSRTAExpiresAt:     time.Now().Add(55 * time.Minute),
	}
}

// TestXboxSSO_SansProfilSuivi_PasDeWatcher — porte fermée : le login aboutit, le
// compte est créé, les tokens sont persistés, et AddPlayer n'est JAMAIS appelé.
func TestXboxSSO_SansProfilSuivi_PasDeWatcher(t *testing.T) {
	users := newXboxStore(t)
	tokenStore := auth.NewMultiUserTokenStore(filepath.Join(t.TempDir(), "watcher_tokens"))
	daemon := &mockDaemon{running: true}

	var vuTitre, vuXUID string
	s := service.NewXboxSSOLinkStrategy(users).
		WithTokenStore(tokenStore).
		WithDaemonGetter(func() service.WatcherDaemon { return daemon }).
		WithProfileGate(func(_ context.Context, titleSlug, xuid string) bool {
			vuTitre, vuXUID = titleSlug, xuid
			return false
		})

	sess := &domain.SessionData{}
	attempt := ssoAttempt()

	if err := s.OnAuthSuccess(context.Background(), attempt, sess); err != nil {
		t.Fatalf("le login doit aboutir même sans profil : %v", err)
	}

	if len(daemon.addCalls) != 0 {
		t.Fatalf("AddPlayer ne doit PAS être appelé sans profil suivi, reçu %d appels", len(daemon.addCalls))
	}
	if vuTitre != "halo_infinite" {
		t.Errorf("la porte doit être interrogée sur le titre par défaut, reçu %q", vuTitre)
	}
	if vuXUID != attempt.XUID {
		t.Errorf("la porte doit être interrogée par xuid, reçu %q", vuXUID)
	}

	// Le compte existe…
	if u, err := users.GetByXUID(attempt.XUID); err != nil || u == nil {
		t.Fatalf("le compte doit être créé malgré l'absence de profil : %v", err)
	}
	// …et les credentials aussi : c'est le refresh token qui rendra le profil
	// utilisable quand l'administrateur le déclarera.
	stored, err := tokenStore.Load(attempt.XUID)
	if err != nil {
		t.Fatalf("les tokens doivent être persistés : %v", err)
	}
	if stored.OAuthRefreshToken != "refresh-token" {
		t.Errorf("refresh token persisté = %q, want refresh-token", stored.OAuthRefreshToken)
	}
	// La session est bien câblée : l'utilisateur est connecté, simplement inerte.
	if sess.LinkedHaloIdentity == nil || sess.LinkedHaloIdentity.XUID != attempt.XUID {
		t.Error("la session doit porter l'identité Halo liée")
	}
}

// TestXboxSSO_AvecProfilSuivi_NotifieLeWatcher — porte ouverte : comportement
// d'origine.
func TestXboxSSO_AvecProfilSuivi_NotifieLeWatcher(t *testing.T) {
	users := newXboxStore(t)
	daemon := &mockDaemon{running: true}

	s := service.NewXboxSSOLinkStrategy(users).
		WithDaemonGetter(func() service.WatcherDaemon { return daemon }).
		WithProfileGate(func(_ context.Context, _, _ string) bool { return true })

	attempt := ssoAttempt()
	if err := s.OnAuthSuccess(context.Background(), attempt, &domain.SessionData{}); err != nil {
		t.Fatalf("OnAuthSuccess: %v", err)
	}

	if len(daemon.addCalls) != 1 {
		t.Fatalf("AddPlayer appelé %d fois, want 1", len(daemon.addCalls))
	}
	if got := daemon.addCalls[0].XUID; got != attempt.XUID {
		t.Errorf("AddPlayer XUID = %q, want %q", got, attempt.XUID)
	}
}

// TestXboxSSO_PorteNil_NotifieLeWatcher — seam documenté : sans porte câblée, le
// comportement historique est conservé (les tests existants du daemon getter en
// dépendent).
func TestXboxSSO_PorteNil_NotifieLeWatcher(t *testing.T) {
	users := newXboxStore(t)
	daemon := &mockDaemon{running: true}

	s := service.NewXboxSSOLinkStrategy(users).
		WithDaemonGetter(func() service.WatcherDaemon { return daemon })

	if err := s.OnAuthSuccess(context.Background(), ssoAttempt(), &domain.SessionData{}); err != nil {
		t.Fatalf("OnAuthSuccess: %v", err)
	}
	if len(daemon.addCalls) != 1 {
		t.Fatalf("porte nil ⇒ AddPlayer appelé %d fois, want 1", len(daemon.addCalls))
	}
}
