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
game_prefix = "h5"

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

// hostOnlyResolver implémente EndpointResolver SANS GamePrefixResolver : prouve
// que GamePrefixFromResolver dégrade vers le défaut "hi" quand le resolver ne
// supporte pas l'extension (resolver legacy / stub de test).
type hostOnlyResolver struct{}

func (hostOnlyResolver) HostFor(string, EndpointKey) (string, bool) { return "", false }

// TestGamePrefixResolver_SyntheticRouting — oracle MT-01 : le game_prefix d'un
// titre route VRAIMENT vers sa valeur déclarée (pas cosmétique). Le titre
// synthétique déclare "h5" ; un titre sans préfixe / inconnu / resolver legacy /
// resolver nil retombent sur DefaultGamePrefix ("hi", byte-identique Halo).
func TestGamePrefixResolver_SyntheticRouting(t *testing.T) {
	t.Parallel()
	const slug = "synthetic_test_title"
	reg := loadSyntheticRegistry(t, slug)
	res := NewMappingsEndpointResolver(reg, "halo_infinite")

	// (1) Le préfixe déclaré "h5" est rendu tel quel (route réelle, pas "hi").
	if p, ok := res.GamePrefixFor(slug); !ok || p != "h5" {
		t.Errorf("GamePrefixFor(%q) = %q ok=%v, want \"h5\" true", slug, p, ok)
	}
	if got := GamePrefixFromResolver(res, slug); got != "h5" {
		t.Errorf("GamePrefixFromResolver(%q) = %q, want \"h5\"", slug, got)
	}

	// (2) Titre inconnu du registre (le défaut halo_infinite n'est pas chargé ici)
	// → pas de préfixe déclaré → fallback "hi".
	if _, ok := res.GamePrefixFor("never_loaded_title"); ok {
		t.Errorf("GamePrefixFor(inconnu) devrait être ok=false")
	}
	if got := GamePrefixFromResolver(res, "never_loaded_title"); got != DefaultGamePrefix {
		t.Errorf("GamePrefixFromResolver(inconnu) = %q, want %q", got, DefaultGamePrefix)
	}

	// (3) Resolver legacy (HostFor seulement) et resolver nil → fallback "hi".
	if got := GamePrefixFromResolver(hostOnlyResolver{}, slug); got != DefaultGamePrefix {
		t.Errorf("GamePrefixFromResolver(legacy) = %q, want %q", got, DefaultGamePrefix)
	}
	if got := GamePrefixFromResolver(nil, slug); got != DefaultGamePrefix {
		t.Errorf("GamePrefixFromResolver(nil) = %q, want %q", got, DefaultGamePrefix)
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

// capabilitiesTOML — fixture capabilities valide (clés du vocabulaire produit
// games.AllCapabilityKeys, statuts admis). Prouve que CapabilitiesFor convertit
// VRAIMENT le set TOML en games.CapabilityMap typée.
func capabilitiesTOML(slug string) string {
	return `
[meta]
title_slug = "` + slug + `"
schema_version = 1

[capabilities]
"match.history"        = "supported"
"match.skill.snapshot" = "degraded"
"analytics.timeseries" = "not_exposed"
`
}

// invalidCapabilitiesTOML — fixture syntaxiquement valide mais portant une clé HORS
// vocabulaire produit. Le set se charge (mappings est title-agnostic et ne valide
// pas les clés), mais CapabilityMapFromMappings doit ÉCHOUER la conversion →
// CapabilitiesFor dégrade en ok=false.
func invalidCapabilitiesTOML(slug string) string {
	return `
[meta]
title_slug = "` + slug + `"
schema_version = 1

[capabilities]
"capability.totalement.inconnue" = "supported"
`
}

// loadRegistryWithCapabilities écrit fields + constants + capabilities.toml dans un
// temp dir et charge la vraie Registry (chemin file→loader→registry).
func loadRegistryWithCapabilities(t *testing.T, slug, capsContent string) *mappings.Registry {
	t.Helper()
	tmp := t.TempDir()
	mappingsDir := filepath.Join(tmp, "config", "titles", slug, "mappings")
	titleDir := filepath.Join(tmp, "config", "titles", slug)
	writeFile(t, mappingsDir, "fields.toml", minimalFieldsTOML(slug))
	writeFile(t, titleDir, "constants.toml", syntheticConstantsTOML(slug))
	writeFile(t, mappingsDir, "capabilities.toml", capsContent)

	r := mappings.NewRegistry()
	if errs := r.LoadFromConfigDir(tmp, []string{slug}, nil); len(errs) != 0 {
		t.Fatalf("LoadFromConfigDir errs: %v", errs)
	}
	return r
}

// TestCapabilitiesFor_HappyPath : un titre déclarant une capabilities.toml valide
// est résolu en games.CapabilityMap typée (chemin nominal Phase 1.7a).
func TestCapabilitiesFor_HappyPath(t *testing.T) {
	t.Parallel()
	const slug = "synthetic_test_title"
	reg := loadRegistryWithCapabilities(t, slug, capabilitiesTOML(slug))
	res := NewMappingsEndpointResolver(reg, "halo_infinite")

	caps, ok := res.CapabilitiesFor(slug)
	if !ok {
		t.Fatalf("CapabilitiesFor(%q) ok=false, want true", slug)
	}
	if got := caps[CapMatchHistory]; got != CapSupported {
		t.Errorf("match.history = %q, want supported", got)
	}
	if got := caps[CapMatchSkillSnapshot]; got != CapDegraded {
		t.Errorf("match.skill.snapshot = %q, want degraded", got)
	}
	if got := caps[CapTimeseries]; got != CapNotExposed {
		t.Errorf("analytics.timeseries = %q, want not_exposed", got)
	}
	// Has() applique la sémantique supported/degraded → exposé, not_exposed → non.
	if !caps.Has(CapMatchHistory) || !caps.Has(CapMatchSkillSnapshot) {
		t.Errorf("Has() devrait être true pour supported/degraded")
	}
	if caps.Has(CapTimeseries) {
		t.Errorf("Has(not_exposed) devrait être false")
	}
}

// TestCapabilitiesFor_EmptySlugUsesDefault : un slug vide (ctx sans titre) route vers
// le défaut, comme HostFor.
func TestCapabilitiesFor_EmptySlugUsesDefault(t *testing.T) {
	t.Parallel()
	const slug = "synthetic_test_title"
	reg := loadRegistryWithCapabilities(t, slug, capabilitiesTOML(slug))
	res := NewMappingsEndpointResolver(reg, slug) // défaut = la fixture

	caps, ok := res.CapabilitiesFor("")
	if !ok {
		t.Fatalf("CapabilitiesFor(\"\") devrait router vers le défaut, ok=false")
	}
	if caps[CapMatchHistory] != CapSupported {
		t.Errorf("défaut: match.history = %q", caps[CapMatchHistory])
	}
}

// TestCapabilitiesFor_Degradations couvre les chemins de dégradation : titre inconnu
// (pas de set), conversion échouée (clé hors vocabulaire produit), resolver/registry
// nil. Dans tous les cas → ok=false, jamais de panique ni de fallback cross-titre.
func TestCapabilitiesFor_Degradations(t *testing.T) {
	t.Parallel()

	t.Run("slug_inconnu_du_registre", func(t *testing.T) {
		t.Parallel()
		reg := loadRegistryWithCapabilities(t, "synthetic_test_title", capabilitiesTOML("synthetic_test_title"))
		res := NewMappingsEndpointResolver(reg, "halo_infinite")
		if caps, ok := res.CapabilitiesFor("never_loaded_title"); ok || caps != nil {
			t.Errorf("slug inconnu → (nil, false), got (%v, %v)", caps, ok)
		}
	})

	t.Run("conversion_echouee_cle_hors_vocabulaire", func(t *testing.T) {
		t.Parallel()
		const slug = "synthetic_bad_caps"
		reg := loadRegistryWithCapabilities(t, slug, invalidCapabilitiesTOML(slug))
		res := NewMappingsEndpointResolver(reg, "halo_infinite")
		// Le set existe (mappings ne valide pas les clés) MAIS CapabilityMapFromMappings
		// échoue → CapabilitiesFor doit dégrader en (nil, false), pas paniquer.
		if caps, ok := res.CapabilitiesFor(slug); ok || caps != nil {
			t.Errorf("conversion échouée → (nil, false), got (%v, %v)", caps, ok)
		}
	})

	t.Run("resolver_nil", func(t *testing.T) {
		t.Parallel()
		var nilRes *MappingsEndpointResolver
		if caps, ok := nilRes.CapabilitiesFor("x"); ok || caps != nil {
			t.Errorf("resolver nil → (nil, false) sans paniquer, got (%v, %v)", caps, ok)
		}
	})

	t.Run("registry_nil", func(t *testing.T) {
		t.Parallel()
		res := NewMappingsEndpointResolver(nil, "halo_infinite")
		if caps, ok := res.CapabilitiesFor("x"); ok || caps != nil {
			t.Errorf("registry nil → (nil, false), got (%v, %v)", caps, ok)
		}
	})
}
