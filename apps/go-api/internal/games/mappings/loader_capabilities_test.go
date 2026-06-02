package mappings

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadCapabilities_Valid(t *testing.T) {
	raw := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 2

[capabilities]
"match.history"        = "supported"
"match.skill.snapshot" = "degraded"
"analytics.timeseries" = "not_exposed"
`)
	set, err := LoadCapabilitiesFromBytes("inline", raw)
	if err != nil {
		t.Fatalf("LoadCapabilitiesFromBytes: %v", err)
	}
	if set.TitleSlug() != "halo_infinite" || set.SchemaVersion() != 2 {
		t.Errorf("meta: got (%q, %d)", set.TitleSlug(), set.SchemaVersion())
	}
	// Les clés à points doivent être parsées comme clés plates (quotées dans le TOML).
	if st, ok := set.Status("match.skill.snapshot"); !ok || st != CapStatusDegraded {
		t.Errorf("Status(match.skill.snapshot) = (%q, %v), want (degraded, true)", st, ok)
	}
	if got := set.Keys(); len(got) != 3 {
		t.Errorf("Keys() = %v, want 3", got)
	}
}

func TestLoadCapabilities_InvalidStatus(t *testing.T) {
	raw := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[capabilities]
"match.history" = "enabled"
`)
	if _, err := LoadCapabilitiesFromBytes("inline", raw); err == nil {
		t.Fatalf("attendu une erreur pour statut inconnu 'enabled'")
	}
}

func TestLoadCapabilities_MissingMetaAndEmpty(t *testing.T) {
	raw := []byte(`
[capabilities]
`)
	if _, err := LoadCapabilitiesFromBytes("inline", raw); err == nil {
		t.Fatalf("attendu une erreur (meta manquante + capabilities vide)")
	}
}

// TestLoadHaloInfiniteCapabilitiesTOML charge le vrai fichier du repo et vérifie
// qu'il est valide + contient les capabilities attendues.
func TestLoadHaloInfiniteCapabilitiesTOML(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
	tomlPath := filepath.Join(repoRoot, "config", "titles", "halo_infinite", "mappings", "capabilities.toml")

	set, err := LoadCapabilitiesFromFile(tomlPath)
	if err != nil {
		t.Fatalf("LoadCapabilitiesFromFile(%s): %v", tomlPath, err)
	}
	for _, key := range []string{"match.history", "pve.firefight_stats", "engagement.score"} {
		if _, ok := set.Status(key); !ok {
			t.Errorf("capabilities.toml: clé %q absente", key)
		}
	}
}
