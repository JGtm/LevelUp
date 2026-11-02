package service

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/platform/auth"
)

func pendingAttempt(xuid string) *auth.Attempt {
	return &auth.Attempt{Gamertag: "GT" + xuid, XUID: xuid, OAuthRefreshToken: "rt-" + xuid}
}

func closedGate(context.Context, string, string) bool { return false }

// Les jetons en attente expirent : passé le délai, la création du profil n'écrit rien.
func TestAuthCompletion_AttenteExpiree(t *testing.T) {
	store := auth.NewMultiUserTokenStore(filepath.Join(t.TempDir(), "watcher_tokens"))
	c := NewAuthCompletion(store, closedGate)
	now := time.Now()
	c.now = func() time.Time { return now }

	c.CompleteForTrackedProfile(context.Background(), pendingAttempt("2535400000000001"))
	now = now.Add(pendingCredentialsTTL + time.Minute)
	c.SeedCredentials(context.Background(), "2535400000000001")

	if _, err := store.Load("2535400000000001"); err == nil {
		t.Error("des jetons en attente expirés ne doivent pas être écrits")
	}
}

// L'attente est bornée : au plafond, l'identité la plus ancienne est oubliée.
func TestAuthCompletion_AttentePlafonnee(t *testing.T) {
	store := auth.NewMultiUserTokenStore(filepath.Join(t.TempDir(), "watcher_tokens"))
	c := NewAuthCompletion(store, closedGate)
	now := time.Now()
	c.now = func() time.Time { return now }

	oldest := fmt.Sprintf("25354%011d", 0)
	for i := 0; i < pendingCredentialsMax+1; i++ {
		c.CompleteForTrackedProfile(context.Background(), pendingAttempt(fmt.Sprintf("25354%011d", i)))
		now = now.Add(time.Second)
	}
	c.mu.Lock()
	got := len(c.pending)
	_, oldestKept := c.pending[oldest]
	c.mu.Unlock()
	if got != pendingCredentialsMax {
		t.Errorf("attente = %d identités, plafond %d", got, pendingCredentialsMax)
	}
	if oldestKept {
		t.Error("l'identité la plus ancienne doit être évincée au plafond")
	}
}
