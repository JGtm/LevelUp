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

// TestHalo5_Manifest : title.toml = active (activation 2026-06-20), xbox_title_id
// réel, coarse capabilities attendues (et absence des coarse non déclarées).
func TestHalo5_Manifest(t *testing.T) {
	desc, err := title.LoadTitleManifest(repoRoot(t), slug)
	if err != nil {
		t.Fatalf("LoadTitleManifest(%s): %v", slug, err)
	}
	if desc.Status != title.StatusActive {
		t.Errorf("status = %q, want active", desc.Status)
	}
	if desc.XboxTitleID != "219630713" {
		t.Errorf("xbox_title_id = %q, want 219630713", desc.XboxTitleID)
	}
	for _, c := range []title.Capability{
		title.CapMatchmaking, title.CapRanked, title.CapCareer,
		title.CapAssetImages, title.CapAchievements, title.CapEngagement, title.CapLUSR,
		// media = pipeline LOCAL (upload utilisateur + corrélation par timestamp ;
		// PAS d'UGC API). Activé axe D prod-gate 2026-06-22.
		title.CapMedia,
	} {
		if !desc.HasCapability(c) {
			t.Errorf("coarse capability %q manquante", c)
		}
	}
	for _, c := range []title.Capability{
		title.CapFirefight, title.CapForge, title.CapWorldLeaderboard,
		// team_mmr / damage_taken = miroir title-level des flags scalaires :
		// Halo 5 déclare no_team_mmr / no_damage_taken (constants.toml) donc
		// ProvidesTeamMMR/ProvidesDamageTaken == false → AUCUNE des deux caps.
		title.CapTeamMMR, title.CapDamageTaken,
	} {
		if desc.HasCapability(c) {
			t.Errorf("coarse capability %q ne devrait PAS être déclarée pour Halo 5", c)
		}
	}
}

// TestHalo5_FineCapabilities : capabilities.toml charge, toutes les clés sont
// connues, et la matrice reflète la surface RÉELLEMENT câblée (career.progression
// + match.events.* + match.detail.core via LoadMatchDetail/voie canonique Match
// View ; le reste = not_exposed tant que stub).
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
		games.CapMatchHistory:         games.CapSupported, // LoadMatchSummaries (shared h5 local → canonical)
		games.CapMatchDetailCore:      games.CapSupported, // LoadMatchDetail (carnage → canonical)
		games.CapScoreboardExtra:      games.CapNotExposed,
		games.CapMatchSkillSnapshot:   games.CapNotExposed,
		games.CapCareerProgression:    games.CapSupported,
		games.CapTimeseries:           games.CapNotExposed,
		games.CapEngagement:           games.CapSupported, // F7 E6b (2026-07-13) : gate humain validé → calibration=validated
		games.CapCitationsEngine:      games.CapNotExposed,
		games.CapCommendationsNative:  games.CapSupported, // commendations NATIVES par match (carnage → match_commendations)
		games.CapPveFirefight:         games.CapNotExposed,
		games.CapBattlePass:           games.CapNotExposed,
		games.CapChallenges:           games.CapNotExposed,
		games.CapMatchEventsTimeline:  games.CapSupported,
		games.CapMatchKillfeedPerKill: games.CapSupported,
		games.CapMatchEventsSpatial:   games.CapSupported,
		games.CapWeaponAccuracy:       games.CapSupported, // précision par arme (events weapon_drop → weapon_accuracy)
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

// TestHalo5_RegistryDiscovery : le registre config découvre Halo 5 ACTIF
// (activation 2026-06-20 -> provisionné -> présent dans Active()).
func TestHalo5_RegistryDiscovery(t *testing.T) {
	reg := title.NewRegistryFromConfig(repoRoot(t), nil)
	desc := reg.Get(slug)
	if desc == nil {
		t.Fatal("halo_5 non découvert par NewRegistryFromConfig")
	}
	if !desc.IsActive() {
		t.Error("halo_5 doit être actif (status=active)")
	}
	found := false
	for _, a := range reg.Active() {
		if a.Slug == slug {
			found = true
		}
	}
	if !found {
		t.Error("halo_5 (active) doit être présent dans Active()")
	}
}
