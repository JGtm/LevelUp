package halo_5_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
)

const slug = "halo_5"

// repoRoot remonte de ce fichier (apps/go-api/internal/games/halo_5/) à la racine
// du repo (5 niveaux), où vit config/titles/.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
}

// TestHalo5_Manifest : title.toml = coming_soon, xbox_title_id réel, coarse
// capabilities attendues (et absence des coarse non déclarées).
func TestHalo5_Manifest(t *testing.T) {
	desc, err := title.LoadTitleManifest(repoRoot(t), slug)
	if err != nil {
		t.Fatalf("LoadTitleManifest(%s): %v", slug, err)
	}
	if desc.Status != title.StatusComingSoon {
		t.Errorf("status = %q, want coming_soon", desc.Status)
	}
	if desc.XboxTitleID != "219630713" {
		t.Errorf("xbox_title_id = %q, want 219630713", desc.XboxTitleID)
	}
	for _, c := range []title.Capability{
		title.CapMatchmaking, title.CapRanked, title.CapCareer,
		title.CapAssetImages, title.CapAchievements, title.CapEngagement, title.CapLUSR,
	} {
		if !desc.HasCapability(c) {
			t.Errorf("coarse capability %q manquante", c)
		}
	}
	for _, c := range []title.Capability{
		title.CapFirefight, title.CapForge, title.CapMedia, title.CapWorldLeaderboard,
	} {
		if desc.HasCapability(c) {
			t.Errorf("coarse capability %q ne devrait PAS être déclarée pour Halo 5", c)
		}
	}
}

// TestHalo5_FineCapabilities : capabilities.toml charge, toutes les clés sont
// connues, et la matrice reflète la surface RÉELLEMENT câblée en Phase 1a (seul
// career.progression ; le reste = not_exposed tant que stub, remonte en Phase 2).
// La matrice optimiste cible vit dans le handoff §2, pas dans ce fichier.
func TestHalo5_FineCapabilities(t *testing.T) {
	path := filepath.Join(repoRoot(t), "config", "titles", slug, "mappings", "capabilities.toml")
	set, err := mappings.LoadCapabilitiesFromFile(path)
	if err != nil {
		t.Fatalf("LoadCapabilitiesFromFile: %v", err)
	}
	cm, err := games.CapabilityMapFromMappings(set)
	if err != nil {
		t.Fatalf("CapabilityMapFromMappings (clé inconnue ?): %v", err)
	}
	want := map[games.CapabilityKey]games.CapabilityStatus{
		games.CapMatchHistory:         games.CapNotExposed,
		games.CapMatchDetailCore:      games.CapNotExposed,
		games.CapScoreboardExtra:      games.CapNotExposed,
		games.CapMatchSkillSnapshot:   games.CapNotExposed,
		games.CapCareerProgression:    games.CapSupported,
		games.CapTimeseries:           games.CapNotExposed,
		games.CapEngagement:           games.CapNotExposed,
		games.CapCitationsEngine:      games.CapNotExposed,
		games.CapPveFirefight:         games.CapNotExposed,
		games.CapBattlePass:           games.CapNotExposed,
		games.CapChallenges:           games.CapNotExposed,
		games.CapMatchEventsTimeline:  games.CapSupported,
		games.CapMatchKillfeedPerKill: games.CapSupported,
		games.CapMatchEventsSpatial:   games.CapSupported,
	}
	if len(cm) != len(want) {
		t.Fatalf("capabilities = %d clés, want %d", len(cm), len(want))
	}
	for k, w := range want {
		if cm[k] != w {
			t.Errorf("capability %q = %q, want %q", k, cm[k], w)
		}
	}
}

// TestHalo5_MappingsLoad : fields/assets/outcomes/constants chargent proprement.
func TestHalo5_MappingsLoad(t *testing.T) {
	base := filepath.Join(repoRoot(t), "config", "titles", slug)
	if _, err := mappings.LoadFieldsFromFile(filepath.Join(base, "mappings", "fields.toml")); err != nil {
		t.Errorf("fields.toml: %v", err)
	}
	if _, err := mappings.LoadAssetsFromFile(filepath.Join(base, "mappings", "assets.toml")); err != nil {
		t.Errorf("assets.toml: %v", err)
	}
	if _, err := mappings.LoadOutcomesFromFile(filepath.Join(base, "mappings", "outcomes.toml")); err != nil {
		t.Errorf("outcomes.toml: %v", err)
	}
	if _, err := mappings.LoadEndpointsFromFile(filepath.Join(base, "constants.toml")); err != nil {
		t.Errorf("constants.toml: %v", err)
	}
}

// TestHalo5_RegistryDiscovery : le registre config découvre Halo 5 en coming_soon
// (donc PAS provisionné -> absent de Active()).
func TestHalo5_RegistryDiscovery(t *testing.T) {
	reg := title.NewRegistryFromConfig(repoRoot(t), nil)
	desc := reg.Get(slug)
	if desc == nil {
		t.Fatal("halo_5 non découvert par NewRegistryFromConfig")
	}
	if desc.IsActive() {
		t.Error("halo_5 ne doit pas être actif (coming_soon)")
	}
	for _, a := range reg.Active() {
		if a.Slug == slug {
			t.Error("halo_5 (coming_soon) ne doit PAS être dans Active()")
		}
	}
}
