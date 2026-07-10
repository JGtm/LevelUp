// halo_client_endpoints_test.go — oracle PMT-1 Contract axe 1 (stats/history).
//
// Prouve que GetMatchHistory/GetMatchStats routent leur host via le
// EndpointResolver title-aware : (a) golden de parité Halo (resolver réel chargé
// du vrai constants.toml → host byte-identique aux const Go), (b) routing
// synthétique piloté par le slug du ctx, (c) fallback legacy si aucun resolver.

package haloclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
)

// stubEndpoints route uniquement (slug, key) → host ; tout le reste → ok=false.
type stubEndpoints struct {
	slug string
	key  games.EndpointKey
	host string
}

func (s stubEndpoints) HostFor(slug string, key games.EndpointKey) (string, bool) {
	if slug == s.slug && key == s.key {
		return s.host, true
	}
	return "", false
}

// realHaloResolver charge le vrai constants.toml du repo (chemin complet
// file→loader→registry→resolver) pour le golden de parité.
func realHaloResolver(t *testing.T) games.EndpointResolver {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
	reg := mappings.NewRegistry()
	if errs := reg.LoadFromConfigDir(repoRoot, []string{"halo_infinite"}, nil); len(errs) != 0 {
		t.Fatalf("load halo registry: %v", errs)
	}
	return games.NewMappingsEndpointResolver(reg, "halo_infinite")
}

// captureStatsHost exécute GetMatchHistory et retourne l'host effectivement visé.
func captureStatsHost(t *testing.T, ctx context.Context, configure func(*HaloAPIClient)) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"Results": []any{}})
	}))
	defer srv.Close()

	client, ct := newDryRunClient(srv)
	if configure != nil {
		configure(client)
	}
	if _, err := client.GetMatchHistory(ctx, "xuid(2535469190789936)", "matchmaking", 0, 25); err != nil {
		t.Fatalf("GetMatchHistory: %v", err)
	}
	captured := ct.last()
	if captured == nil {
		t.Fatal("aucune requête capturée")
	}
	return captured.URL.Host
}

func TestHostFor_StatsAxis(t *testing.T) {
	// Isoler l'état global : restaurer le resolver partagé en sortie.
	prev := games.DefaultEndpointResolver()
	t.Cleanup(func() { games.SetDefaultEndpointResolver(prev) })

	const haloHost = "halostats.svc.halowaypoint.com:443"

	t.Run("instance resolver halo = parité byte-identique", func(t *testing.T) {
		games.SetDefaultEndpointResolver(nil)
		host := captureStatsHost(t, context.Background(), func(c *HaloAPIClient) {
			c.WithEndpoints(realHaloResolver(t))
		})
		if host != haloHost {
			t.Errorf("host = %q, want %q (parité const Go)", host, haloHost)
		}
	})

	t.Run("routing synthétique piloté par le slug du ctx", func(t *testing.T) {
		games.SetDefaultEndpointResolver(nil)
		ctx := ctxkeys.WithTitleSlug(context.Background(), "synthetic_x")
		host := captureStatsHost(t, ctx, func(c *HaloAPIClient) {
			c.WithEndpoints(stubEndpoints{slug: "synthetic_x", key: games.EndpointStats, host: "https://stats.example.test"})
		})
		if host != "stats.example.test" {
			t.Errorf("host = %q, want stats.example.test (le seam route via le ctx)", host)
		}
	})

	t.Run("aucun resolver câblé → fallback const legacy", func(t *testing.T) {
		games.SetDefaultEndpointResolver(nil)
		host := captureStatsHost(t, context.Background(), nil)
		if host != haloHost {
			t.Errorf("host = %q, want %q (fallback legacy)", host, haloHost)
		}
	})

	t.Run("resolver partagé de boot (halo) → parité", func(t *testing.T) {
		games.SetDefaultEndpointResolver(realHaloResolver(t))
		host := captureStatsHost(t, context.Background(), nil)
		if host != haloHost {
			t.Errorf("host = %q, want %q (resolver partagé)", host, haloHost)
		}
	})

	t.Run("endpoint absent pour le titre → warn + fallback legacy", func(t *testing.T) {
		games.SetDefaultEndpointResolver(nil)
		ctx := ctxkeys.WithTitleSlug(context.Background(), "title_without_stats")
		// stub ne connaît que "synthetic_x" → HostFor("title_without_stats") = false.
		host := captureStatsHost(t, ctx, func(c *HaloAPIClient) {
			c.WithEndpoints(stubEndpoints{slug: "synthetic_x", key: games.EndpointStats, host: "https://stats.example.test"})
		})
		if host != haloHost {
			t.Errorf("host = %q, want %q (dégradation sur fallback)", host, haloHost)
		}
	})
}

// captureHostVia exécute `call` (un appel HaloAPIClient) et retourne l'host visé.
func captureHostVia(t *testing.T, ctx context.Context, configure func(*HaloAPIClient), call func(context.Context, *HaloAPIClient)) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	client, ct := newDryRunClient(srv)
	if configure != nil {
		configure(client)
	}
	call(ctx, client) // l'host est capturé avant la réécriture locale, succès indifférent
	captured := ct.last()
	if captured == nil {
		t.Fatal("aucune requête capturée")
	}
	return captured.URL.Host
}

func TestHostFor_SkillAxis(t *testing.T) {
	prev := games.DefaultEndpointResolver()
	t.Cleanup(func() { games.SetDefaultEndpointResolver(prev) })

	const skillHost = "skill.svc.halowaypoint.com:443"
	// GetPlayerCSRs vise l'host skill ; capture indifférente au parsing.
	callCSRs := func(ctx context.Context, c *HaloAPIClient) {
		_, _ = c.GetPlayerCSRs(ctx, "2535469190789936", "CsrSeason8")
	}

	t.Run("resolver halo = parité byte-identique", func(t *testing.T) {
		games.SetDefaultEndpointResolver(nil)
		host := captureHostVia(t, context.Background(), func(c *HaloAPIClient) {
			c.WithEndpoints(realHaloResolver(t))
		}, callCSRs)
		if host != skillHost {
			t.Errorf("host = %q, want %q", host, skillHost)
		}
	})

	t.Run("routing synthétique via ctx slug", func(t *testing.T) {
		games.SetDefaultEndpointResolver(nil)
		ctx := ctxkeys.WithTitleSlug(context.Background(), "synthetic_x")
		host := captureHostVia(t, ctx, func(c *HaloAPIClient) {
			c.WithEndpoints(stubEndpoints{slug: "synthetic_x", key: games.EndpointSkill, host: "https://skill.example.test"})
		}, callCSRs)
		if host != "skill.example.test" {
			t.Errorf("host = %q, want skill.example.test", host)
		}
	})

	t.Run("aucun resolver → fallback const legacy", func(t *testing.T) {
		games.SetDefaultEndpointResolver(nil)
		host := captureHostVia(t, context.Background(), nil, callCSRs)
		if host != skillHost {
			t.Errorf("host = %q, want %q (fallback)", host, skillHost)
		}
	})
}

func TestHostFor_FilmAndDiscoveryAxis(t *testing.T) {
	prev := games.DefaultEndpointResolver()
	t.Cleanup(func() { games.SetDefaultEndpointResolver(prev) })

	const ugcHost = "discovery-infiniteugc.svc.halowaypoint.com"
	callFilm := func(ctx context.Context, c *HaloAPIClient) {
		_, _, _ = c.GetMatchFilm(ctx, "11111111-1111-1111-1111-000000000001")
	}
	callPlaylist := func(ctx context.Context, c *HaloAPIClient) {
		_, _ = c.GetPlaylistConfig(ctx, "playlist-asset-id", "v1")
	}

	t.Run("film (UGCFilm) parité halo", func(t *testing.T) {
		games.SetDefaultEndpointResolver(nil)
		host := captureHostVia(t, context.Background(), func(c *HaloAPIClient) {
			c.WithEndpoints(realHaloResolver(t))
		}, callFilm)
		if host != ugcHost {
			t.Errorf("film host = %q, want %q", host, ugcHost)
		}
	})

	t.Run("film (UGCFilm) routing synthétique", func(t *testing.T) {
		games.SetDefaultEndpointResolver(nil)
		ctx := ctxkeys.WithTitleSlug(context.Background(), "synthetic_x")
		host := captureHostVia(t, ctx, func(c *HaloAPIClient) {
			c.WithEndpoints(stubEndpoints{slug: "synthetic_x", key: games.EndpointUGCFilm, host: "https://film.example.test"})
		}, callFilm)
		if host != "film.example.test" {
			t.Errorf("film host = %q, want film.example.test", host)
		}
	})

	t.Run("playlist (DiscoveryUGC) parité halo", func(t *testing.T) {
		games.SetDefaultEndpointResolver(nil)
		host := captureHostVia(t, context.Background(), func(c *HaloAPIClient) {
			c.WithEndpoints(realHaloResolver(t))
		}, callPlaylist)
		if host != ugcHost {
			t.Errorf("playlist host = %q, want %q", host, ugcHost)
		}
	})

	t.Run("playlist (DiscoveryUGC) routing synthétique", func(t *testing.T) {
		games.SetDefaultEndpointResolver(nil)
		ctx := ctxkeys.WithTitleSlug(context.Background(), "synthetic_x")
		host := captureHostVia(t, ctx, func(c *HaloAPIClient) {
			c.WithEndpoints(stubEndpoints{slug: "synthetic_x", key: games.EndpointDiscoveryUGC, host: "https://discovery.example.test"})
		}, callPlaylist)
		if host != "discovery.example.test" {
			t.Errorf("playlist host = %q, want discovery.example.test", host)
		}
	})
}
