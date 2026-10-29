package tokencache

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// TestTTLDeRepli_SansExpiryConnue : un jeton sans expiry connue est mis en cache pour 50 min
// (fallbackTokenTTL) et servi tant qu'il reste plus que la marge (5 min) : servi à 44 min de vie,
// plus servi à 46 min. L'horloge n'est pas injectable : l'expiry calculée est bornée par les deux
// lectures de l'horloge qui l'entourent, et le service est jugé sur une entrée posée à l'instant
// voulu.
func TestTTLDeRepli_SansExpiryConnue(t *testing.T) {
	resetPlayerTokenStore()
	t.Cleanup(resetPlayerTokenStore)
	tokens := &domain.HaloTokens{SpartanToken: "s", ClearanceToken: "c"}

	avant := time.Now()
	SetCachedPlayerTokens("x", tokens)
	apres := time.Now()
	playerTokenStore.mu.RLock()
	exp := playerTokenStore.store["x"].expiresAt
	playerTokenStore.mu.RUnlock()
	if exp.Before(avant.Add(50*time.Minute)) || exp.After(apres.Add(50*time.Minute)) {
		t.Fatalf("expiry de repli = %v après la pose, attendu 50 min", exp.Sub(avant))
	}

	poser := func(age time.Duration) {
		playerTokenStore.mu.Lock()
		playerTokenStore.store["x"] = cachedTokenEntry{tokens: tokens, expiresAt: time.Now().Add(50*time.Minute - age)}
		playerTokenStore.mu.Unlock()
	}
	poser(44 * time.Minute)
	if GetCachedPlayerTokens("x") == nil {
		t.Error("jeton de 44 min (repli 50 min, marge 5 min) : attendu servi")
	}
	poser(46 * time.Minute)
	if GetCachedPlayerTokens("x") != nil {
		t.Error("jeton de 46 min (repli 50 min, marge 5 min) : attendu périmé")
	}
}
