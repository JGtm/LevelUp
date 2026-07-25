package wire

// post_sync_home_cache_test.go — régression de l'invalidation du cache Home
// après une sync (contre-revue V7.2, 2026-07-25).
//
// Bug attrapé : le hook post-sync ne reçoit qu'un PLAYER slug
// (sync_handler.postSync → PostSyncDeltaHook(ctx, req.PlayerSlug)) alors que
// HomeMatchesCache est keyé (xuid, TITLE slug). Invalidate(xuid, playerSlug)
// était donc un no-op permanent — la Home continuait de servir ses données
// jusqu'à l'expiration du TTL (45 s) tout en loguant « home cache invalidé ».

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"levelup/go-api/internal/service"
)

// seedHomeCacheAsHomeService pose une entrée EXACTEMENT comme le fait
// HomeService.fetchMatchesAndSessions : Set(s.xuid, s.titleSlug, …) où
// s.xuid = pdb.XUID (WithMatchesCache) et s.titleSlug = pdb.TitleSlug
// (WithPlayerMatchesRepo) — cf. registry_pages_home.HomeCtx.
func seedHomeCacheAsHomeService(cache *service.HomeMatchesCache, xuid, titleSlug string) {
	cache.Set(xuid, titleSlug, nil, nil)
}

// TestInvalidateHomeCacheAfterSync_UsesTitleSlug : l'entrée posée par
// HomeService DOIT disparaître, même quand le slug JOUEUR diffère du slug
// TITRE. Avant le fix (Invalidate(xuid, playerSlug)) ce test échoue.
func TestInvalidateHomeCacheAfterSync_UsesTitleSlug(t *testing.T) {
	const (
		xuid       = "2533274800000001"
		titleSlug  = "halo_infinite"
		playerSlug = "guillaume" // slug JOUEUR, volontairement != slug TITRE
	)
	cache := service.NewHomeMatchesCache()
	seedHomeCacheAsHomeService(cache, xuid, titleSlug)

	if _, _, hit := cache.Get(xuid, titleSlug); !hit {
		t.Fatal("préparation : l'entrée Home devrait être en cache avant l'invalidation")
	}

	invalidateHomeCacheAfterSync(context.Background(), cache, xuid, titleSlug, playerSlug)

	if _, _, hit := cache.Get(xuid, titleSlug); hit {
		t.Error("l'entrée Home (xuid, titleSlug) survit à la sync : l'invalidation a utilisé " +
			"une mauvaise clé (le slug JOUEUR au lieu du slug TITRE) — la Home sert des données périmées")
	}
}

// TestInvalidateHomeCacheAfterSync_OtherTitleUntouched : isolation par titre
// (V72-29) — synchroniser un titre ne doit pas vider la Home de l'autre.
func TestInvalidateHomeCacheAfterSync_OtherTitleUntouched(t *testing.T) {
	const xuid = "2533274800000002"
	cache := service.NewHomeMatchesCache()
	seedHomeCacheAsHomeService(cache, xuid, "halo_infinite")
	seedHomeCacheAsHomeService(cache, xuid, "halo_5")

	invalidateHomeCacheAfterSync(context.Background(), cache, xuid, "halo_5", "guillaume")

	if _, _, hit := cache.Get(xuid, "halo_5"); hit {
		t.Error("l'entrée du titre synchronisé (halo_5) devrait être invalidée")
	}
	if _, _, hit := cache.Get(xuid, "halo_infinite"); !hit {
		t.Error("l'entrée d'un AUTRE titre ne doit pas être invalidée (isolation V72-29)")
	}
}

// TestInvalidateHomeCacheAfterSync_Degrade : cache nil ou xuid vide (registre
// partiellement câblé / résolution incomplète) → no-op silencieux, jamais de panic.
func TestInvalidateHomeCacheAfterSync_Degrade(t *testing.T) {
	invalidateHomeCacheAfterSync(context.Background(), nil, "x", "halo_infinite", "guillaume")
	cache := service.NewHomeMatchesCache()
	seedHomeCacheAsHomeService(cache, "2533274800000003", "halo_infinite")
	invalidateHomeCacheAfterSync(context.Background(), cache, "", "halo_infinite", "guillaume")
	if _, _, hit := cache.Get("2533274800000003", "halo_infinite"); !hit {
		t.Error("xuid vide ne doit rien invalider")
	}
}

// homeCacheKeyWiringRE : la clé de cache Home côté HomeService est le TITLE slug
// (WithPlayerMatchesRepo(<repo>, <x>.TitleSlug, …)). Regex indépendante du nom du
// receiver et de la variable pdb.
var homeCacheKeyWiringRE = regexp.MustCompile(`WithPlayerMatchesRepo\([^\n]*\.TitleSlug`)

// TestHomeCacheKeyWiring_IsTitleSlug ancre l'hypothèse du test ci-dessus : si le
// wiring de HomeService changeait de clé, la parité serait à revalider ici plutôt
// que de laisser l'invalidation redevenir silencieusement un no-op.
func TestHomeCacheKeyWiring_IsTitleSlug(t *testing.T) {
	src, err := os.ReadFile("registry_pages_home.go")
	if err != nil {
		t.Fatalf("lecture registry_pages_home.go: %v", err)
	}
	if !homeCacheKeyWiringRE.Match(src) {
		t.Fatal("registry_pages_home.go ne câble plus HomeService sur <pdb>.TitleSlug : " +
			"la clé du HomeMatchesCache a changé, revalider invalidateHomeCacheAfterSync")
	}
}

// playerSlugInvalidateRE détecte le retour du bug : une invalidation du cache
// Home passant la variable `slug` (= PLAYER slug dans le hook post-sync).
var playerSlugInvalidateRE = regexp.MustCompile(`homeMatchesCache\.Invalidate\([^)]*,\s*slug\s*\)`)

// TestPostSyncHomeCache_NoPlayerSlugInvalidate : garde-rail textuel — aucun
// fichier du package ne doit invalider le cache Home avec le slug joueur.
func TestPostSyncHomeCache_NoPlayerSlugInvalidate(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du répertoire package: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, rerr := os.ReadFile(name)
		if rerr != nil {
			t.Fatalf("lecture %s: %v", name, rerr)
		}
		if playerSlugInvalidateRE.Match(src) {
			t.Errorf("%s invalide le cache Home avec `slug` (PLAYER slug) : la clé attend le TITLE slug "+
				"— utiliser invalidateHomeCacheAfterSync(ctx, cache, xuid, pdb.TitleSlug, slug)", name)
		}
	}
}
