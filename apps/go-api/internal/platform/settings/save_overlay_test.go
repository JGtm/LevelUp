package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestSaveTitleOverlay_SparseWriteResolveGlobalUntouched : SaveTitleOverlay écrit
// UNIQUEMENT l'overlay (sparse), ResolveForTitle fait gagner l'overlay sur le
// global champ-présent-only, et le global app_settings.json reste INCHANGÉ
// (PMT-4 PR-3c — un PATCH sur un 2e titre ne fuit jamais vers Halo).
func TestSaveTitleOverlay_SparseWriteResolveGlobalUntouched(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "app_settings.json")
	if err := os.WriteFile(globalPath, []byte(`{
		"show_progression": false,
		"outcome_exclude_bot_matches_from_badges": false,
		"friend_gamertags": ["GlobalFriend"],
		"lang": "fr"
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewStore(globalPath)
	overlayPath := filepath.Join(dir, "titles", "synthetic_title_b", "settings.json")

	// Overlay sparse : seulement show_progression=true (les autres héritent du global).
	trueJSON, _ := json.Marshal(true)
	if err := s.SaveTitleOverlay(overlayPath, map[string]json.RawMessage{"show_progression": trueJSON}); err != nil {
		t.Fatalf("SaveTitleOverlay: %v", err)
	}

	// L'overlay sur disque est SPARSE (1 clé), pas une copie complète du global.
	raw, _ := os.ReadFile(overlayPath)
	var overlayMap map[string]json.RawMessage
	_ = json.Unmarshal(raw, &overlayMap)
	if len(overlayMap) != 1 {
		t.Errorf("overlay devrait être sparse (1 clé), got %d: %s", len(overlayMap), raw)
	}

	// ResolveForTitle : show_progression de l'overlay (true), reste hérité du global.
	resolved, err := s.ResolveForTitle(overlayPath)
	if err != nil {
		t.Fatalf("ResolveForTitle: %v", err)
	}
	if !resolved.ShowProgression {
		t.Error("show_progression devrait venir de l'overlay (true)")
	}
	if len(resolved.FriendGamertags) != 1 || resolved.FriendGamertags[0] != "GlobalFriend" {
		t.Errorf("friend_gamertags global devrait être préservé, got %v", resolved.FriendGamertags)
	}
	if resolved.Lang != "fr" {
		t.Errorf("lang global préservé, got %q", resolved.Lang)
	}

	// Le GLOBAL n'est PAS modifié : ResolveForTitle("") = base globale (false).
	base, err := s.ResolveForTitle("")
	if err != nil {
		t.Fatalf("ResolveForTitle(\"\"): %v", err)
	}
	if base.ShowProgression {
		t.Error("SaveTitleOverlay ne doit PAS modifier le global (show_progression resté false)")
	}

	// Merge incrémental : un 2e SaveTitleOverlay ajoute une clé sans perdre l'autre.
	if err := s.SaveTitleOverlay(overlayPath, map[string]json.RawMessage{"outcome_exclude_bot_matches_from_badges": trueJSON}); err != nil {
		t.Fatalf("SaveTitleOverlay 2: %v", err)
	}
	resolved2, _ := s.ResolveForTitle(overlayPath)
	if !resolved2.ShowProgression || !resolved2.OutcomeExcludeBotMatchesFromBadges {
		t.Error("les deux clés overlay devraient être présentes après merge incrémental")
	}
}
