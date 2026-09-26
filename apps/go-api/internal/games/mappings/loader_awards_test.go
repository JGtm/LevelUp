package mappings

import (
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/testutil"
)

// Tests unitaires du loader awards.toml (V2 §2).

func TestLoadAwardsFromBytes_Happy(t *testing.T) {
	raw := []byte(`
[meta]
title_slug     = "halo_infinite"
schema_version = 1

[awards.flag_captured]
axes   = ["objective"]
weight = 5.0

[awards.kill_assist]
axes   = ["support"]
weight = 1.0

[awards.flag_capture_assist]
axes   = ["support", "objective"]
weight = 2.0
`)
	set, err := LoadAwardsFromBytes("test.toml", raw)
	if err != nil {
		t.Fatalf("LoadAwardsFromBytes: %v", err)
	}
	if set.TitleSlug() != "halo_infinite" {
		t.Errorf("title_slug: got %q", set.TitleSlug())
	}
	if set.SchemaVersion() != 1 {
		t.Errorf("schema_version: got %d", set.SchemaVersion())
	}

	flag, ok := set.Get("flag_captured")
	if !ok {
		t.Fatal("flag_captured not found")
	}
	if flag.Weight != 5.0 {
		t.Errorf("weight: got %.1f, want 5.0", flag.Weight)
	}
	if len(flag.Axes) != 1 || flag.Axes[0] != "objective" {
		t.Errorf("axes: got %v, want [objective]", flag.Axes)
	}

	assist, _ := set.Get("flag_capture_assist")
	if len(assist.Axes) != 2 {
		t.Errorf("flag_capture_assist axes: got %v, want 2 axes", assist.Axes)
	}

	_, missing := set.Get("unknown_award")
	if missing {
		t.Error("unknown_award should return false")
	}
}

func TestLoadAwardsFromBytes_RejectsInvalidAxis(t *testing.T) {
	raw := []byte(`
[meta]
title_slug     = "halo_infinite"
schema_version = 1

[awards.bad_one]
axes   = ["nonexistent_axis"]
weight = 1.0
`)
	_, err := LoadAwardsFromBytes("test.toml", raw)
	if err == nil {
		t.Fatal("expected error for invalid axis")
	}
	if !strings.Contains(err.Error(), "axe inconnu") {
		t.Errorf("error message: got %q, want contains 'axe inconnu'", err.Error())
	}
}

func TestLoadAwardsFromBytes_RejectsZeroWeight(t *testing.T) {
	raw := []byte(`
[meta]
title_slug     = "halo_infinite"
schema_version = 1

[awards.bad_one]
axes   = ["combat"]
weight = 0.0
`)
	_, err := LoadAwardsFromBytes("test.toml", raw)
	if err == nil {
		t.Fatal("expected error for weight=0")
	}
	if !strings.Contains(err.Error(), "weight") {
		t.Errorf("error message should mention weight: %q", err.Error())
	}
}

func TestLoadAwardsFromBytes_RejectsEmptyAxes(t *testing.T) {
	raw := []byte(`
[meta]
title_slug     = "halo_infinite"
schema_version = 1

[awards.bad_one]
axes   = []
weight = 1.0
`)
	_, err := LoadAwardsFromBytes("test.toml", raw)
	if err == nil {
		t.Fatal("expected error for empty axes")
	}
}

func TestLoadAwardsFromBytes_RejectsMetaMissing(t *testing.T) {
	raw := []byte(`
[awards.flag_captured]
axes   = ["objective"]
weight = 5.0
`)
	_, err := LoadAwardsFromBytes("test.toml", raw)
	if err == nil {
		t.Fatal("expected error for missing [meta]")
	}
}

func TestAwardMappingSet_NilSafe(t *testing.T) {
	var set *AwardMappingSet
	if _, ok := set.Get("anything"); ok {
		t.Error("nil set.Get should return false")
	}
	if set.All() != nil {
		t.Error("nil set.All should return nil")
	}
}

// TestLoadAwardsFromFile_RealConfig vérifie que le fichier de prod parse OK
// et contient les awards critiques (objective + impact) — garde-fou contre
// les régressions sur awards.toml de halo_infinite.
func TestLoadAwardsFromFile_RealConfig(t *testing.T) {
	// awards.toml est VERSIONNÉ : son absence est une installation cassée, pas un cas à
	// skipper (revue ronde 1, R1-1 — un skip ici rendait ce garde-fou muet en CI).
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	path := filepath.Join(root, "config", "titles", "halo_infinite", "mappings", "awards.toml")
	set, err := LoadAwardsFromFile(path)
	if err != nil {
		t.Fatalf("awards.toml versionné illisible (%s) : %v", path, err)
	}
	criticalAwards := []struct {
		name     string
		mustHave string // axis qui DOIT être présent dans la liste
	}{
		{"flag_captured", "objective"},
		{"zone_captured", "objective"},
		{"destroyed_phantom", "impact"},
		{"revived_player", "support"},
		{"kill_assist", "support"},
		{"killed_player", "combat"},
	}
	for _, c := range criticalAwards {
		entry, ok := set.Get(c.name)
		if !ok {
			t.Errorf("award %q absent du config", c.name)
			continue
		}
		found := false
		for _, axis := range entry.Axes {
			if axis == c.mustHave {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("award %q n'a pas l'axe %q (got %v)", c.name, c.mustHave, entry.Axes)
		}
	}
}
