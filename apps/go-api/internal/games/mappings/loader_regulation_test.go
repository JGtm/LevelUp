package mappings

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestLoadRegulationTOMLsFromRepo — smoke test sur les VRAIS fichiers du repo :
// halo_infinite doit porter les 9 variantes mesurées à 720 s, halo_5 doit être
// vide (aucun temps réglementaire mesuré → aucun flag « Prolongation »).
func TestLoadRegulationTOMLsFromRepo(t *testing.T) {
	t.Parallel()
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")

	hi, err := LoadRegulationFromFile(filepath.Join(repoRoot, "config", "titles", "halo_infinite", "mappings", "regulation.toml"))
	if err != nil {
		t.Fatalf("halo_infinite regulation.toml: %v", err)
	}
	if n := len(hi.SecondsMap()); n != 9 {
		t.Errorf("halo_infinite : %d variantes, want 9", n)
	}
	for _, variant := range []string{"CTF:Arena", "Slayer:Arena", "Strongholds:Arena", "Arena:Team Slayer"} {
		if secs, ok := hi.Seconds(variant); !ok || secs != 720 {
			t.Errorf("halo_infinite %q = (%d, %v), want (720, true)", variant, secs, ok)
		}
	}

	h5, err := LoadRegulationFromFile(filepath.Join(repoRoot, "config", "titles", "halo_5", "mappings", "regulation.toml"))
	if err != nil {
		t.Fatalf("halo_5 regulation.toml: %v", err)
	}
	if n := len(h5.SecondsMap()); n != 0 {
		t.Errorf("halo_5 : %d variantes, want 0 (aucune mesure)", n)
	}
}

func TestLoadRegulation_Valid(t *testing.T) {
	raw := []byte(`
[meta]
title_slug     = "halo_infinite"
schema_version = 1

[regulation_seconds]
"CTF:Arena"         = 720
"Team Slayer:Arena" = 720
`)
	set, err := LoadRegulationFromBytes("regulation.toml", raw)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if set.TitleSlug() != "halo_infinite" {
		t.Errorf("title_slug: got %q", set.TitleSlug())
	}
	if set.SchemaVersion() != 1 {
		t.Errorf("schema_version: got %d", set.SchemaVersion())
	}
	if secs, ok := set.Seconds("CTF:Arena"); !ok || secs != 720 {
		t.Errorf("CTF:Arena: got (%d, %v), want (720, true)", secs, ok)
	}
	// Variante inconnue → jamais de flag (dégradation sûre).
	if secs, ok := set.Seconds("BTB:Slayer"); ok || secs != 0 {
		t.Errorf("variante inconnue: got (%d, %v), want (0, false)", secs, ok)
	}
	if n := len(set.SecondsMap()); n != 2 {
		t.Errorf("SecondsMap len: got %d, want 2", n)
	}
}

// Table VIDE = valide (titre sans temps réglementaire mesuré, ex. Halo 5).
func TestLoadRegulation_EmptyTableIsValid(t *testing.T) {
	raw := []byte(`
[meta]
title_slug     = "halo_5"
schema_version = 1

[regulation_seconds]
`)
	set, err := LoadRegulationFromBytes("regulation.toml", raw)
	if err != nil {
		t.Fatalf("une table vide doit être valide : %v", err)
	}
	if n := len(set.SecondsMap()); n != 0 {
		t.Errorf("SecondsMap len: got %d, want 0", n)
	}
	if _, ok := set.Seconds("CTF:Arena"); ok {
		t.Error("aucune variante ne doit être connue sur une table vide")
	}
}

func TestRegulationSet_NilIsSafe(t *testing.T) {
	var set *RegulationSet
	if secs, ok := set.Seconds("CTF:Arena"); ok || secs != 0 {
		t.Errorf("nil Seconds: got (%d, %v), want (0, false)", secs, ok)
	}
	if n := len(set.SecondsMap()); n != 0 {
		t.Errorf("nil SecondsMap must be empty: got %d", n)
	}
	if set.TitleSlug() != "" || set.SchemaVersion() != 0 {
		t.Error("nil accessors must return zero values")
	}
}

func TestLoadRegulation_Invalid(t *testing.T) {
	cases := map[string]string{
		"meta manquant": `
[regulation_seconds]
"CTF:Arena" = 720
`,
		"schema_version zero": `
[meta]
title_slug = "halo_infinite"
[regulation_seconds]
"CTF:Arena" = 720
`,
		"temps negatif": `
[meta]
title_slug     = "halo_infinite"
schema_version = 1
[regulation_seconds]
"CTF:Arena" = -1
`,
		"temps zero": `
[meta]
title_slug     = "halo_infinite"
schema_version = 1
[regulation_seconds]
"CTF:Arena" = 0
`,
	}
	for name, raw := range cases {
		if _, err := LoadRegulationFromBytes("t.toml", []byte(raw)); err == nil {
			t.Errorf("%s: attendu une erreur", name)
		}
	}
}
