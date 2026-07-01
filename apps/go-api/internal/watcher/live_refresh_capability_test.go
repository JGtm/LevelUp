package watcher

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/games"
)

// capStubResolver implémente games.EndpointResolver + games.CapabilityResolver
// pour piloter le gating live-service du refresher sans charger de TOML réel.
type capStubResolver struct {
	caps map[string]games.CapabilityMap
}

func (capStubResolver) HostFor(string, games.EndpointKey) (string, bool) { return "", false }

func (s capStubResolver) CapabilitiesFor(slug string) (games.CapabilityMap, bool) {
	c, ok := s.caps[slug]
	return c, ok
}

// withDefaultResolver câble un resolver global le temps du test puis restaure
// l'ancien (nil en run de test) — le gating passe par games.DefaultEndpointResolver.
func withDefaultResolver(t *testing.T, r games.EndpointResolver) {
	t.Helper()
	prev := games.DefaultEndpointResolver()
	games.SetDefaultEndpointResolver(r)
	t.Cleanup(func() { games.SetDefaultEndpointResolver(prev) })
}

// TestPlayerLiveRefresher_TitreSansSurface_NeDemarrePasLeTicker : un titre qui
// déclare BP + Challenges not_exposed (Halo 5) → OnPresenceActive ne démarre PAS
// le ticker (aucune sonde economy/decks 404 périodique).
func TestPlayerLiveRefresher_TitreSansSurface_NeDemarrePasLeTicker(t *testing.T) {
	withDefaultResolver(t, capStubResolver{caps: map[string]games.CapabilityMap{
		"halo_5": {games.CapBattlePass: games.CapNotExposed, games.CapChallenges: games.CapNotExposed},
	}})

	r := NewPlayerLiveRefresher("H5Player", "xuid-h5", nil, nil).WithTitleSlug("halo_5")
	r.interval = 24 * time.Hour

	ctx := context.Background()
	r.OnPresenceActive(ctx)
	defer r.OnPresenceInactive(ctx)

	r.cancelMu.Lock()
	c := r.cancel
	r.cancelMu.Unlock()

	if c != nil {
		t.Fatal("le ticker ne doit PAS démarrer pour un titre sans BP ni Challenges (Halo 5)")
	}
}

// TestPlayerLiveRefresher_TitreAvecSurface_DemarreLeTicker : un titre qui déclare
// au moins une surface (BP supported) → le ticker démarre normalement.
func TestPlayerLiveRefresher_TitreAvecSurface_DemarreLeTicker(t *testing.T) {
	withDefaultResolver(t, capStubResolver{caps: map[string]games.CapabilityMap{
		"halo_infinite": {games.CapBattlePass: games.CapSupported, games.CapChallenges: games.CapSupported},
	}})

	r := NewPlayerLiveRefresher("HIPlayer", "xuid-hi", nil, nil).WithTitleSlug("halo_infinite")
	r.interval = 24 * time.Hour

	ctx := context.Background()
	r.OnPresenceActive(ctx)
	defer r.OnPresenceInactive(ctx)

	r.cancelMu.Lock()
	c := r.cancel
	r.cancelMu.Unlock()

	if c == nil {
		t.Fatal("le ticker doit démarrer pour un titre exposant au moins une surface live-service")
	}
}
