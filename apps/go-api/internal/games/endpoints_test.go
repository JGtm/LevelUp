package games

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/mappings"
)

// writeFile écrit un fichier dans dir (créé si besoin) pour les fixtures de test.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// minimalFieldsTOML — fields.toml valide minimal (requis pour que la Registry
// charge le reste, dont constants.toml).
func minimalFieldsTOML(slug string) string {
	return `
[meta]
title_slug = "` + slug + `"
schema_version = 1

[fields.kills]
labels = { en = "Kills", fr = "Frags" }
storage_unit = "count"
display_unit = "count"
format = "integer"
display_order = 10
group = "combat"
`
}

// syntheticConstantsTOML — fixture : hosts distincts (example.test) + endpoint
// `skill` volontairement ABSENT, pour exercer la dégradation.
func syntheticConstantsTOML(slug string) string {
	return `
[meta]
title_slug = "` + slug + `"
schema_version = 1

[endpoints]
stats = "https://stats.example.test"
gamecms = "https://cms.example.test"
economy = "https://economy.example.test"
ugc_film = "https://film.example.test"
discovery_ugc = "https://discovery.example.test"
challenges = "https://challenges.example.test"
nameplate = "https://nameplate.example.test"
`
}

// loadSyntheticRegistry écrit la fixture synthetic_test_title dans un temp dir et
// la charge via la vraie Registry.LoadFromConfigDir (chemin file→loader→registry).
func loadSyntheticRegistry(t *testing.T, slug string) *mappings.Registry {
	t.Helper()
	tmp := t.TempDir()
	mappingsDir := filepath.Join(tmp, "config", "titles", slug, "mappings")
	titleDir := filepath.Join(tmp, "config", "titles", slug)
	writeFile(t, mappingsDir, "fields.toml", minimalFieldsTOML(slug))
	writeFile(t, titleDir, "constants.toml", syntheticConstantsTOML(slug))

	r := mappings.NewRegistry()
	if errs := r.LoadFromConfigDir(tmp, []string{slug}, nil); len(errs) != 0 {
		t.Fatalf("LoadFromConfigDir errs: %v", errs)
	}
	return r
}

func TestEndpointResolver_SyntheticRouting(t *testing.T) {
	t.Parallel()
	const slug = "synthetic_test_title"
	reg := loadSyntheticRegistry(t, slug)
	res := NewMappingsEndpointResolver(reg, "halo_infinite")

	// (1) Le seam route VRAIMENT vers example.test (pas cosmétique).
	if h, ok := res.HostFor(slug, EndpointStats); !ok || h != "https://stats.example.test" {
		t.Errorf("HostFor(stats) = %q ok=%v, want example.test", h, ok)
	}
	if h, ok := res.HostFor(slug, EndpointDiscoveryUGC); !ok || h != "https://discovery.example.test" {
		t.Errorf("HostFor(discovery_ugc) = %q ok=%v", h, ok)
	}

	// (2) Endpoint absent (skill omis) → ok=false, dégradation propre (le caller
	// skip + warn ; pas de fallback silencieux vers l'host Halo).
	if h, ok := res.HostFor(slug, EndpointSkill); ok {
		t.Errorf("HostFor(skill) devrait être ok=false, got %q", h)
	}
}

func TestEndpointResolver_UnknownSlugNoFallback(t *testing.T) {
	t.Parallel()
	reg := loadSyntheticRegistry(t, "synthetic_test_title")
	res := NewMappingsEndpointResolver(reg, "halo_infinite")

	// Slug NON vide mais non chargé → ok=false (pas de fallback cross-titre,
	// même si halo_infinite est le défaut).
	if h, ok := res.HostFor("never_loaded_title", EndpointStats); ok {
		t.Errorf("slug inconnu devrait dégrader, got %q", h)
	}
}

func TestEndpointResolver_EmptySlugUsesDefault(t *testing.T) {
	t.Parallel()
	const slug = "synthetic_test_title"
	reg := loadSyntheticRegistry(t, slug)
	// defaultSlug = la fixture → un slug vide (ctx sans titre) route vers le défaut.
	res := NewMappingsEndpointResolver(reg, slug)

	if h, ok := res.HostFor("", EndpointStats); !ok || h != "https://stats.example.test" {
		t.Errorf("HostFor(\"\") devrait router vers le défaut, got %q ok=%v", h, ok)
	}
}

func TestEndpointResolver_NilSafe(t *testing.T) {
	t.Parallel()
	var nilRes *MappingsEndpointResolver
	if _, ok := nilRes.HostFor("x", EndpointStats); ok {
		t.Error("resolver nil devrait retourner ok=false sans paniquer")
	}
	res := NewMappingsEndpointResolver(nil, "halo_infinite")
	if _, ok := res.HostFor("x", EndpointStats); ok {
		t.Error("registry nil devrait retourner ok=false")
	}
}
