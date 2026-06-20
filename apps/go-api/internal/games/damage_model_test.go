package games

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/mappings"
)

// TestEffectiveHpToKill_TitleAwareAndFallback — oracle damage-model Phase 0 : un
// titre déclare son effective_hp_to_kill dans constants.toml [damage_model] et la
// valeur route VRAIMENT (pas le 225 par défaut) ; un titre sans section / inconnu /
// resolver legacy / resolver nil retombe sur DefaultEffectiveHpToKill (225,
// byte-identique Halo Infinite).
func TestEffectiveHpToKill_TitleAwareAndFallback(t *testing.T) {
	t.Parallel()
	const slug = "synthetic_dm_title"

	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "config", "titles", slug, "mappings"), "fields.toml", minimalFieldsTOML(slug))
	writeFile(t, filepath.Join(tmp, "config", "titles", slug), "constants.toml", `
[meta]
title_slug = "`+slug+`"
schema_version = 1

[endpoints]
stats = "https://stats.example.test"

[damage_model]
effective_hp_to_kill = 115
`)

	reg := mappings.NewRegistry()
	if errs := reg.LoadFromConfigDir(tmp, []string{slug}, nil); len(errs) != 0 {
		t.Fatalf("LoadFromConfigDir errs: %v", errs)
	}
	res := NewMappingsEndpointResolver(reg, "halo_infinite")

	// (1) La valeur déclarée (115) route réellement, pas le défaut 225.
	if dm, ok := res.DamageModelFor(slug); !ok || dm.EffectiveHpToKill != 115 {
		t.Errorf("DamageModelFor(%q) = %+v ok=%v, want EffectiveHpToKill=115 true", slug, dm, ok)
	}
	if got := EffectiveHpToKillFromResolver(res, slug); got != 115 {
		t.Errorf("EffectiveHpToKillFromResolver(%q) = %v, want 115", slug, got)
	}

	// (2) Titre inconnu du registre → défaut 225.
	if got := EffectiveHpToKillFromResolver(res, "never_loaded"); got != DefaultEffectiveHpToKill {
		t.Errorf("inconnu = %v, want %v (défaut)", got, DefaultEffectiveHpToKill)
	}

	// (3) Resolver legacy (sans DamageModelResolver) → défaut 225.
	if got := EffectiveHpToKillFromResolver(hostOnlyResolver{}, slug); got != DefaultEffectiveHpToKill {
		t.Errorf("resolver legacy = %v, want %v (défaut)", got, DefaultEffectiveHpToKill)
	}

	// (4) Resolver nil → défaut 225.
	if got := EffectiveHpToKillFromResolver(nil, slug); got != DefaultEffectiveHpToKill {
		t.Errorf("resolver nil = %v, want %v (défaut)", got, DefaultEffectiveHpToKill)
	}
}

// TestDamageModel_AbsentSection — un titre sans [damage_model] expose ok=false et
// retombe sur le défaut (la fixture synthétique partagée n'a pas la section).
func TestDamageModel_AbsentSection(t *testing.T) {
	t.Parallel()
	const slug = "synthetic_no_dm"
	reg := loadSyntheticRegistry(t, slug)
	res := NewMappingsEndpointResolver(reg, "halo_infinite")

	if _, ok := res.DamageModelFor(slug); ok {
		t.Errorf("DamageModelFor sans [damage_model] devrait être ok=false")
	}
	if got := EffectiveHpToKillFromResolver(res, slug); got != DefaultEffectiveHpToKill {
		t.Errorf("sans section = %v, want %v (défaut)", got, DefaultEffectiveHpToKill)
	}
}
