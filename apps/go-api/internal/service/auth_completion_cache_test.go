package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/tokencache"
	"levelup/go-api/internal/service"
)

// poseCacheDeJetons met dans le cache process une chaîne Halo fraîche pour dcXUID, que seule une
// invalidation retire.
func poseCacheDeJetons(t *testing.T) {
	t.Helper()
	tokencache.SetCachedPlayerTokens(dcXUID, &domain.HaloTokens{
		SpartanToken: "ancienne-chaine", SpartanExpiresAt: time.Now().Add(time.Hour),
	})
	t.Cleanup(func() { tokencache.InvalidateCachedPlayerTokens(dcXUID) })
	if tokencache.GetCachedPlayerTokens(dcXUID) == nil {
		t.Fatal("préparation : le cache doit servir la chaîne posée")
	}
}

// Après chaque écriture de jetons de l'AuthCompletion, la chaîne Halo en cache de l'ancienne
// connexion n'est plus servie (code d'appareil comme navigateur : les deux passent ici).
func TestAuthCompletion_InvalideLeCacheDesJetonsApresEcriture(t *testing.T) {
	ctx := context.Background()
	t.Run("Persist", func(t *testing.T) {
		poseCacheDeJetons(t)
		c := service.NewAuthCompletion(dcStore(t), nil)
		if ok, err := c.Persist(ctx, dcAttempt()); !ok || err != nil {
			t.Fatalf("Persist = (%v, %v)", ok, err)
		}
		if tokencache.GetCachedPlayerTokens(dcXUID) != nil {
			t.Error("Persist doit purger le cache process des HaloTokens")
		}
	})
	t.Run("CompleteForTrackedProfile", func(t *testing.T) {
		poseCacheDeJetons(t)
		tracked := true
		c := service.NewAuthCompletion(dcStore(t), gateFor(&tracked, nil))
		if !c.CompleteForTrackedProfile(ctx, dcAttempt()) {
			t.Fatal("profil suivi : écriture attendue")
		}
		if tokencache.GetCachedPlayerTokens(dcXUID) != nil {
			t.Error("CompleteForTrackedProfile doit purger le cache process des HaloTokens")
		}
	})
	t.Run("SeedCredentials", func(t *testing.T) {
		tracked := false
		c := service.NewAuthCompletion(dcStore(t), gateFor(&tracked, nil))
		c.CompleteForTrackedProfile(ctx, dcAttempt()) // mis en attente, rien écrit
		poseCacheDeJetons(t)
		c.SeedCredentials(ctx, dcXUID)
		if tokencache.GetCachedPlayerTokens(dcXUID) != nil {
			t.Error("SeedCredentials doit purger le cache process des HaloTokens")
		}
	})
}

// failingStore : un magasin dont l'écriture échoue.
type failingStore struct{ service.CredentialStore }

func (failingStore) Load(string) (*auth.UserTokens, error) { return nil, auth.ErrUserTokensNotFound }
func (failingStore) Upsert(*auth.UserTokens) error         { return errors.New("disque plein") }

// Une écriture échouée ne purge rien : l'ancienne chaîne reste la seule valide.
func TestAuthCompletion_EcritureEchouee_NePurgePasLeCache(t *testing.T) {
	poseCacheDeJetons(t)
	c := service.NewAuthCompletion(failingStore{}, nil)
	if ok, err := c.Persist(context.Background(), dcAttempt()); ok || err == nil {
		t.Fatalf("Persist = (%v, %v), attendu une erreur", ok, err)
	}
	if tokencache.GetCachedPlayerTokens(dcXUID) == nil {
		t.Error("aucune purge sans écriture réussie")
	}
}
