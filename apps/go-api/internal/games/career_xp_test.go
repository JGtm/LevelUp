package games

import (
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/games/mappings"
)

// TestCareerXPEras_TitleAwareAndDefault — un titre déclare ses [[career_xp_eras]]
// dans constants.toml et elles routent VRAIMENT (multiplicateur custom 3.0, pas le
// défaut 2.0) ; un titre inconnu / resolver nil / resolver legacy retombe sur
// DefaultCareerXPEras (×1 avant 2025-11-18, ×2 depuis).
func TestCareerXPEras_TitleAwareAndDefault(t *testing.T) {
	t.Parallel()
	const slug = "synthetic_xp_title"

	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "config", "titles", slug, "mappings"), "fields.toml", minimalFieldsTOML(slug))
	writeFile(t, filepath.Join(tmp, "config", "titles", slug), "constants.toml", `
[meta]
title_slug = "`+slug+`"
schema_version = 1

[endpoints]
stats = "https://stats.example.test"

[[career_xp_eras]]
from = ""
to = "2025-11-18"
multiplier = 1.0

[[career_xp_eras]]
from = "2025-11-18"
to = ""
multiplier = 3.0
`)
	reg := mappings.NewRegistry()
	if errs := reg.LoadFromConfigDir(tmp, []string{slug}, nil); len(errs) != 0 {
		t.Fatalf("LoadFromConfigDir errs: %v", errs)
	}
	res := NewMappingsEndpointResolver(reg, "halo_infinite")

	// (1) Les éras déclarées routent (multiplicateur 3.0 custom, dates parsées).
	eras := CareerXPErasFromResolver(res, slug)
	if len(eras) != 2 {
		t.Fatalf("éras = %d, want 2", len(eras))
	}
	cut := time.Date(2025, 11, 18, 0, 0, 0, 0, time.UTC)
	if !eras[1].From.Equal(cut) || eras[1].Multiplier != 3.0 {
		t.Errorf("éra 2 = %+v, want From=%s Multiplier=3.0", eras[1], cut)
	}
	if !eras[0].From.IsZero() || !eras[0].To.Equal(cut) || eras[0].Multiplier != 1.0 {
		t.Errorf("éra 1 = %+v, want From ouvert, To=%s, Multiplier=1.0", eras[0], cut)
	}

	// (2) Titre inconnu du registre → défaut (×2 depuis cutover).
	def := CareerXPErasFromResolver(res, "never_loaded")
	if len(def) != 2 || def[1].Multiplier != 2.0 {
		t.Errorf("défaut (inconnu) = %+v, want 2 éras avec ×2 en 2e", def)
	}

	// (3) Resolver legacy (sans extension) → défaut.
	if got := CareerXPErasFromResolver(hostOnlyResolver{}, slug); len(got) != 2 || got[1].Multiplier != 2.0 {
		t.Errorf("resolver legacy = %+v, want défaut", got)
	}

	// (4) Resolver nil → défaut.
	if got := CareerXPErasFromResolver(nil, slug); len(got) != 2 || got[1].Multiplier != 2.0 {
		t.Errorf("resolver nil = %+v, want défaut", got)
	}
}

// TestProvidesCareerXPEstimate — la capability analytics.career_xp_estimate est
// opt-in STRICT (défaut false) : true seulement si le titre la déclare supported/
// degraded ; un titre déclarant ses capabilities SANS la clé, un resolver sans
// extension, ou nil → false.
func TestProvidesCareerXPEstimate(t *testing.T) {
	t.Parallel()

	withCap := &stubCapResolver{caps: map[string]CapabilityMap{
		"with_cap": {CapAnalyticsCareerXPEstimate: CapSupported},
	}}
	if !ProvidesCareerXPEstimateFromResolver(withCap, "with_cap") {
		t.Error("titre avec la capability doit exposer l'XP estimée")
	}

	without := &stubCapResolver{caps: map[string]CapabilityMap{
		"h5like": {CapMatchHistory: CapSupported}, // pas de CapAnalyticsCareerXPEstimate
	}}
	if ProvidesCareerXPEstimateFromResolver(without, "h5like") {
		t.Error("titre sans la clé NE doit PAS exposer l'XP estimée")
	}

	if ProvidesCareerXPEstimateFromResolver(hostOnlyResolver{}, "x") {
		t.Error("resolver legacy → false (opt-in strict)")
	}
	if ProvidesCareerXPEstimateFromResolver(nil, "x") {
		t.Error("resolver nil → false (opt-in strict)")
	}
}

// TestCareerXP_RealConfig — oracle bout-en-bout sur la config RÉELLE : halo_infinite
// déclare la capability ET ses 2 éras (×2 depuis 2025-11-18) ; halo_5 (Spartan Rank,
// système d'XP distinct) NE déclare PAS la capability.
func TestCareerXP_RealConfig(t *testing.T) {
	t.Parallel()
	reg := mappings.NewRegistry()
	root := repoConfigRoot(t)
	if errs := reg.LoadFromConfigDir(root, []string{"halo_5", "halo_infinite"}, nil); len(errs) != 0 {
		t.Fatalf("LoadFromConfigDir(halo_5, halo_infinite) errs: %v", errs)
	}
	res := NewMappingsEndpointResolver(reg, "halo_infinite")

	if !ProvidesCareerXPEstimateFromResolver(res, "halo_infinite") {
		t.Error("halo_infinite doit exposer analytics.career_xp_estimate")
	}
	eras, ok := res.CareerXPErasFor("halo_infinite")
	if !ok || len(eras) != 2 {
		t.Fatalf("halo_infinite éras = %v ok=%v, want 2 déclarées", eras, ok)
	}
	if eras[1].Multiplier != 2.0 {
		t.Errorf("halo_infinite éra 2 multiplier = %v, want 2.0", eras[1].Multiplier)
	}

	if ProvidesCareerXPEstimateFromResolver(res, "halo_5") {
		t.Error("halo_5 NE doit PAS exposer l'XP de carrière estimée (Spartan Rank distinct)")
	}
}
