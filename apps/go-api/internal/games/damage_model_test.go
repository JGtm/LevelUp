package games

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/games/mappings"
)

// repoConfigRoot résout la racine du repo (qui contient config/titles/...) depuis
// l'emplacement de ce fichier de test (apps/go-api/internal/games → 4 niveaux au-
// dessus). Permet de charger la VRAIE config halo_5 / halo_infinite (oracle e2e).
func repoConfigRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué (impossible de localiser le fichier de test)")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "..")
	if _, err := os.Stat(filepath.Join(root, "config", "titles", "halo_5", "constants.toml")); err != nil {
		t.Fatalf("config halo_5 introuvable depuis la racine résolue %q: %v", root, err)
	}
	return root
}

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

// TestProvidesTeamMMR — un titre déclarant no_team_mmr=true (Halo 5) route false
// (colonne MMR du tableau Escouade/Explorer masquée) ; sans le flag / inconnu /
// resolver legacy / resolver nil → true (défaut Infinite).
func TestProvidesTeamMMR(t *testing.T) {
	t.Parallel()
	const slug = "synthetic_noteammmr_title"

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
no_team_mmr = true
`)
	reg := mappings.NewRegistry()
	if errs := reg.LoadFromConfigDir(tmp, []string{slug}, nil); len(errs) != 0 {
		t.Fatalf("LoadFromConfigDir errs: %v", errs)
	}
	res := NewMappingsEndpointResolver(reg, "halo_infinite")

	if ProvidesTeamMMRFromResolver(res, slug) {
		t.Errorf("ProvidesTeamMMR(%q) = true, want false (no_team_mmr=true)", slug)
	}
	if !ProvidesTeamMMRFromResolver(res, "never_loaded") {
		t.Error("titre inconnu : ProvidesTeamMMR devrait être true (défaut)")
	}
	if !ProvidesTeamMMRFromResolver(hostOnlyResolver{}, slug) {
		t.Error("resolver legacy : ProvidesTeamMMR devrait être true (défaut)")
	}
	if !ProvidesTeamMMRFromResolver(nil, slug) {
		t.Error("resolver nil : ProvidesTeamMMR devrait être true (défaut)")
	}
}

// TestProvidesMaxKillingSpree — le support de la « folie meurtrière max » est DÉRIVÉ
// de la capability events-timeline (CapMatchEventsTimeline), pas d'un flag du modèle
// de dégâts. Un titre qui sert des kills horodatés (events-timeline supported/degraded)
// → true (spree CALCULABLE) ; un titre déclarant ses capabilities SANS events-timeline
// → false (vraie absence de substrat) ; titre inconnu / resolver sans extension /
// resolver nil → true (défaut Infinite).
func TestProvidesMaxKillingSpree(t *testing.T) {
	t.Parallel()

	// (1) Titre avec events-timeline supported → spree calculable → true.
	withEvents := &stubCapResolver{
		caps: map[string]CapabilityMap{
			"with_events": {CapMatchEventsTimeline: CapSupported},
		},
	}
	if !ProvidesMaxKillingSpreeFromResolver(withEvents, "with_events") {
		t.Error("titre avec events-timeline supported doit fournir la spree (calculable)")
	}

	// (1b) events-timeline DEGRADED (Infinite reconstitue depuis highlight_events) → true.
	degraded := &stubCapResolver{
		caps: map[string]CapabilityMap{
			"degraded_events": {CapMatchEventsTimeline: CapDegraded},
		},
	}
	if !ProvidesMaxKillingSpreeFromResolver(degraded, "degraded_events") {
		t.Error("events-timeline degraded doit aussi fournir la spree (Has() couvre degraded)")
	}

	// (2) Titre déclarant ses capabilities SANS events-timeline → false (pas de substrat
	// d'events horodatés → ni valeur native ni calcul possible).
	noEvents := &stubCapResolver{
		caps: map[string]CapabilityMap{
			"no_events": {CapMatchHistory: CapSupported}, // pas de CapMatchEventsTimeline
		},
	}
	if ProvidesMaxKillingSpreeFromResolver(noEvents, "no_events") {
		t.Error("titre sans events-timeline NE doit PAS fournir la spree")
	}

	// (3) Titre inconnu (found=false) → défaut sûr true.
	unknown := &stubCapResolver{caps: map[string]CapabilityMap{}, found: map[string]bool{"x": false}}
	if !ProvidesMaxKillingSpreeFromResolver(unknown, "x") {
		t.Error("titre sans capabilities déclarées : ProvidesMaxKillingSpree devrait être true (défaut)")
	}

	// (4) Resolver sans l'extension CapabilityResolver → true (défaut Infinite).
	if !ProvidesMaxKillingSpreeFromResolver(hostOnlyResolver{}, "x") {
		t.Error("resolver legacy : ProvidesMaxKillingSpree devrait être true (défaut)")
	}

	// (5) Resolver nil → true (défaut Infinite).
	if !ProvidesMaxKillingSpreeFromResolver(nil, "x") {
		t.Error("resolver nil : ProvidesMaxKillingSpree devrait être true (défaut)")
	}
}

// TestH5ProvidesTeamMMRAndMaxKillingSpree — oracle bout-en-bout sur la config RÉELLE :
// halo_5 ne fournit PAS de team MMR (no_team_mmr=true → false) mais SUPPORTE la spree
// (match.events.timeline = supported → true, calculée depuis les kills horodatés).
// Infinite reste true sur les deux axes.
func TestH5ProvidesTeamMMRAndMaxKillingSpree(t *testing.T) {
	t.Parallel()
	reg := mappings.NewRegistry()
	root := repoConfigRoot(t)
	if errs := reg.LoadFromConfigDir(root, []string{"halo_5", "halo_infinite"}, nil); len(errs) != 0 {
		t.Fatalf("LoadFromConfigDir(halo_5, halo_infinite) errs: %v", errs)
	}
	res := NewMappingsEndpointResolver(reg, "halo_infinite")

	if ProvidesTeamMMRFromResolver(res, "halo_5") {
		t.Error("halo_5 : ProvidesTeamMMR doit être false (no_team_mmr=true)")
	}
	if !ProvidesMaxKillingSpreeFromResolver(res, "halo_5") {
		t.Error("halo_5 : ProvidesMaxKillingSpree doit être true (events-timeline supported → spree calculée)")
	}
	if !ProvidesTeamMMRFromResolver(res, "halo_infinite") {
		t.Error("halo_infinite : ProvidesTeamMMR doit être true (défaut, flag absent)")
	}
	if !ProvidesMaxKillingSpreeFromResolver(res, "halo_infinite") {
		t.Error("halo_infinite : ProvidesMaxKillingSpree doit être true (events-timeline degraded)")
	}
}
