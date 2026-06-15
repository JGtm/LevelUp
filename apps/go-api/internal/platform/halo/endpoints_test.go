package halo

import (
	"context"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
)

// stubResolver route (slug, key) → host ; le reste → ok=false.
type stubResolver struct {
	slug string
	key  games.EndpointKey
	host string
}

func (s stubResolver) HostFor(slug string, key games.EndpointKey) (string, bool) {
	if slug == s.slug && key == s.key {
		return s.host, true
	}
	return "", false
}

func TestHaloProvider_hostFor(t *testing.T) {
	prev := games.DefaultEndpointResolver()
	t.Cleanup(func() { games.SetDefaultEndpointResolver(prev) })

	p := NewHaloProvider()
	const legacy = "https://gamecms-hacs.svc.halowaypoint.com"

	t.Run("override d'instance prioritaire", func(t *testing.T) {
		games.SetDefaultEndpointResolver(stubResolver{slug: "halo_infinite", key: games.EndpointGameCMS, host: "https://resolver.example.test"})
		got := p.hostFor(context.Background(), games.EndpointGameCMS, "https://override.test", legacy)
		if got != "https://override.test" {
			t.Errorf("got %q, want override", got)
		}
	})

	t.Run("resolver routing via ctx slug", func(t *testing.T) {
		games.SetDefaultEndpointResolver(stubResolver{slug: "synthetic_x", key: games.EndpointGameCMS, host: "https://cms.example.test"})
		ctx := ctxkeys.WithTitleSlug(context.Background(), "synthetic_x")
		got := p.hostFor(ctx, games.EndpointGameCMS, "", legacy)
		if got != "https://cms.example.test" {
			t.Errorf("got %q, want cms.example.test", got)
		}
	})

	t.Run("aucun resolver → legacy", func(t *testing.T) {
		games.SetDefaultEndpointResolver(nil)
		got := p.hostFor(context.Background(), games.EndpointGameCMS, "", legacy)
		if got != legacy {
			t.Errorf("got %q, want legacy", got)
		}
	})

	t.Run("endpoint absent pour le titre → legacy (dégradation)", func(t *testing.T) {
		games.SetDefaultEndpointResolver(stubResolver{slug: "other", key: games.EndpointGameCMS, host: "https://x.test"})
		ctx := ctxkeys.WithTitleSlug(context.Background(), "title_without_cms")
		got := p.hostFor(ctx, games.EndpointGameCMS, "", legacy)
		if got != legacy {
			t.Errorf("got %q, want legacy (fallback)", got)
		}
	})

	t.Run("halo_infinite resolver = parité (gamecms sans :443)", func(t *testing.T) {
		games.SetDefaultEndpointResolver(stubResolver{slug: "halo_infinite", key: games.EndpointGameCMS, host: legacy})
		got := p.hostFor(context.Background(), games.EndpointGameCMS, "", legacy)
		if got != legacy {
			t.Errorf("got %q, want %q (parité)", got, legacy)
		}
	})
}
