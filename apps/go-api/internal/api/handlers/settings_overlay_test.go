package handlers

import (
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	settings_platform "levelup/go-api/internal/platform/settings"
)

// writeJSONFile écrit un fichier JSON de test (helper local).
func writeJSONFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestSessionComputeOptionsFor_OverlayPerTitle (PMT-4) verrouille la résolution
// per-titre des options de session : Halo SANS overlay déclaré == global
// byte-identique ; un titre AVEC overlay surcharge gap/split sans fuiter sur les
// autres titres. C'est l'oracle de la contrainte « overlay vide ⇒ global » au
// niveau du site d'usage (le seam ResolveForTitle a déjà son oracle primitive).
func TestSessionComputeOptionsFor_OverlayPerTitle(t *testing.T) {
	root := t.TempDir()
	// Global app_settings.json : gap=90, split=false.
	writeJSONFile(t, filepath.Join(root, "app_settings.json"),
		`{"session_gap_minutes":90,"session_split_on_ranked_change":false}`)
	store := settings_platform.NewStore(filepath.Join(root, "app_settings.json"))
	pr := titlePkg.NewPathResolver(root)

	// (a) Halo sans overlay → global byte-identique (gap=90, split=false).
	halo := sessionComputeOptionsFor(store, pr, "halo_infinite")
	if halo.GapMinutes != 90 || halo.SplitOnRankedChange {
		t.Fatalf("halo (overlay absent) attendu gap=90 split=false (global), obtenu %+v", halo)
	}

	// (b) Overlay synthetic_title_b : gap=30, split=true (champs surchargés).
	writeJSONFile(t, pr.TitleSettingsPath("synthetic_title_b"),
		`{"session_gap_minutes":30,"session_split_on_ranked_change":true}`)
	synth := sessionComputeOptionsFor(store, pr, "synthetic_title_b")
	if synth.GapMinutes != 30 || !synth.SplitOnRankedChange {
		t.Fatalf("synthetic_title_b (overlay) attendu gap=30 split=true, obtenu %+v", synth)
	}

	// (c) Isolation : Halo reste sur le global (pas de fuite depuis l'overlay synthetic).
	if halo2 := sessionComputeOptionsFor(store, pr, "halo_infinite"); halo2.GapMinutes != 90 {
		t.Fatalf("halo après overlay synthetic : attendu gap=90 (isolation), obtenu %d", halo2.GapMinutes)
	}
}

// TestSessionComputeOptionsFor_GapFallbackAndNilStore garde le fallback gap<=0 → 120
// et la dégradation store nil → Defaults().
func TestSessionComputeOptionsFor_GapFallbackAndNilStore(t *testing.T) {
	root := t.TempDir()
	// Global avec gap invalide (0) → fallback 120.
	writeJSONFile(t, filepath.Join(root, "app_settings.json"), `{"session_gap_minutes":0}`)
	store := settings_platform.NewStore(filepath.Join(root, "app_settings.json"))
	pr := titlePkg.NewPathResolver(root)

	if got := sessionComputeOptionsFor(store, pr, "halo_infinite"); got.GapMinutes != 120 {
		t.Fatalf("gap<=0 doit retomber sur 120, obtenu %d", got.GapMinutes)
	}
	// store nil → Defaults() (gap par défaut, jamais 0 brut).
	if got := sessionComputeOptionsFor(nil, pr, "halo_infinite"); got.GapMinutes <= 0 {
		t.Fatalf("store nil doit donner les Defaults() (gap>0), obtenu %d", got.GapMinutes)
	}
}
