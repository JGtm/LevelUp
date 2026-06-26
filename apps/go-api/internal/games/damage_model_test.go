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

// TestOffensiveConversionP80_TitleAwareAndFallback — un titre déclare son
// offensive_conversion_p80 (frontière élite OC) et la valeur route (pas le défaut
// 0.90 Infinite) ; titre inconnu / resolver nil → fallback DefaultOffensiveConversionP80.
func TestOffensiveConversionP80_TitleAwareAndFallback(t *testing.T) {
	t.Parallel()
	const slug = "synthetic_p80_title"

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
offensive_conversion_p80 = 1.264
`)
	reg := mappings.NewRegistry()
	if errs := reg.LoadFromConfigDir(tmp, []string{slug}, nil); len(errs) != 0 {
		t.Fatalf("LoadFromConfigDir errs: %v", errs)
	}
	res := NewMappingsEndpointResolver(reg, "halo_infinite")

	if got := OffensiveConversionP80FromResolver(res, slug); got != 1.264 {
		t.Errorf("OffensiveConversionP80(%q) = %v, want 1.264", slug, got)
	}
	if got := OffensiveConversionP80FromResolver(res, "never_loaded"); got != DefaultOffensiveConversionP80 {
		t.Errorf("inconnu = %v, want %v (défaut)", got, DefaultOffensiveConversionP80)
	}
	if got := OffensiveConversionP80FromResolver(nil, slug); got != DefaultOffensiveConversionP80 {
		t.Errorf("resolver nil = %v, want %v (défaut)", got, DefaultOffensiveConversionP80)
	}
}

// TestProvidesNativeKDA — un titre déclarant no_native_kda=true (Halo 5) route
// false ; un titre sans le flag / inconnu / resolver legacy / resolver nil retombe
// sur true (défaut Infinite). Garantit qu'on ne fabrique jamais de KDA pour un
// titre qui n'en fournit pas via son API (règle absolue).
func TestProvidesNativeKDA(t *testing.T) {
	t.Parallel()
	const slug = "synthetic_nokda_title"

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
no_native_kda = true
`)
	reg := mappings.NewRegistry()
	if errs := reg.LoadFromConfigDir(tmp, []string{slug}, nil); len(errs) != 0 {
		t.Fatalf("LoadFromConfigDir errs: %v", errs)
	}
	res := NewMappingsEndpointResolver(reg, "halo_infinite")

	if ProvidesNativeKDAFromResolver(res, slug) {
		t.Errorf("ProvidesNativeKDA(%q) = true, want false (no_native_kda=true)", slug)
	}
	if !ProvidesNativeKDAFromResolver(res, "never_loaded") {
		t.Error("titre inconnu : ProvidesNativeKDA devrait être true (défaut)")
	}
	if !ProvidesNativeKDAFromResolver(hostOnlyResolver{}, slug) {
		t.Error("resolver legacy : ProvidesNativeKDA devrait être true (défaut)")
	}
	if !ProvidesNativeKDAFromResolver(nil, slug) {
		t.Error("resolver nil : ProvidesNativeKDA devrait être true (défaut)")
	}
}

// TestProvidesMMR — un titre déclarant no_mmr=true (Halo 5) route false (MMR non
// fourni → surfaces MMR omises) ; sans le flag / inconnu / resolver nil → true.
func TestProvidesMMR(t *testing.T) {
	t.Parallel()
	const slug = "synthetic_nommr_title"

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
no_mmr = true
`)
	reg := mappings.NewRegistry()
	if errs := reg.LoadFromConfigDir(tmp, []string{slug}, nil); len(errs) != 0 {
		t.Fatalf("LoadFromConfigDir errs: %v", errs)
	}
	res := NewMappingsEndpointResolver(reg, "halo_infinite")

	if ProvidesMMRFromResolver(res, slug) {
		t.Errorf("ProvidesMMR(%q) = true, want false (no_mmr=true)", slug)
	}
	if !ProvidesMMRFromResolver(res, "never_loaded") {
		t.Error("titre inconnu : ProvidesMMR devrait être true (défaut)")
	}
}

// TestProvidesDamageTaken — un titre déclarant no_damage_taken=true (Halo 5) route
// false (résistance défensive non calculable → surfaces DR neutralisées) ; sans le
// flag / inconnu / resolver nil → true (défaut Infinite).
func TestProvidesDamageTaken(t *testing.T) {
	t.Parallel()
	const slug = "synthetic_nodt_title"

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
no_damage_taken = true
`)
	reg := mappings.NewRegistry()
	if errs := reg.LoadFromConfigDir(tmp, []string{slug}, nil); len(errs) != 0 {
		t.Fatalf("LoadFromConfigDir errs: %v", errs)
	}
	res := NewMappingsEndpointResolver(reg, "halo_infinite")

	if ProvidesDamageTakenFromResolver(res, slug) {
		t.Errorf("ProvidesDamageTaken(%q) = true, want false (no_damage_taken=true)", slug)
	}
	if !ProvidesDamageTakenFromResolver(res, "never_loaded") {
		t.Error("titre inconnu : ProvidesDamageTaken devrait être true (défaut)")
	}
	if !ProvidesDamageTakenFromResolver(nil, slug) {
		t.Error("resolver nil : ProvidesDamageTaken devrait être true (défaut)")
	}
}
