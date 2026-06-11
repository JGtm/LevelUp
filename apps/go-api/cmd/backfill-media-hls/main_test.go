package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadCapturesBase : lecture de media_captures_base_dir depuis
// app_settings.json ; "" si fichier absent ou JSON invalide. La sélection des
// candidats (vidéos sans HLS) est désormais testée dans internal/ops
// (TestSelectPendingHLSCandidates) puisque la logique y a été centralisée.
func TestLoadCapturesBase(t *testing.T) {
	dir := t.TempDir()
	settings := filepath.Join(dir, "app_settings.json")
	if err := os.WriteFile(settings, []byte(`{"media_captures_base_dir":"C:/Captures"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := loadCapturesBase(settings); got != "C:/Captures" {
		t.Errorf("loadCapturesBase = %q, want C:/Captures", got)
	}
	if got := loadCapturesBase(filepath.Join(dir, "absent.json")); got != "" {
		t.Errorf("fichier absent: loadCapturesBase = %q, want \"\"", got)
	}
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := loadCapturesBase(bad); got != "" {
		t.Errorf("JSON invalide: loadCapturesBase = %q, want \"\"", got)
	}
}
