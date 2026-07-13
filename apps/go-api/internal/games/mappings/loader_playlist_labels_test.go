package mappings

import "testing"

func TestLoadPlaylistLabels_Valid(t *testing.T) {
	raw := []byte(`
[meta]
title_slug     = "halo_5"
schema_version = 1

[overrides]
"Super Fiesta Fête" = "Super Fiesta"
"Team Slayer Classé" = "Team Slayer"
`)
	set, err := LoadPlaylistLabelsFromBytes("playlist_labels.toml", raw)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := set.Display("Super Fiesta Fête"); got != "Super Fiesta" {
		t.Errorf("override Super Fiesta: got %q", got)
	}
	if got := set.Display("Team Slayer Classé"); got != "Team Slayer" {
		t.Errorf("override Team Slayer: got %q", got)
	}
	// Nom non mappé → inchangé (aucune heuristique).
	if got := set.Display("Big Team Battle"); got != "Big Team Battle" {
		t.Errorf("passthrough: got %q", got)
	}
	if got := set.SchemaVersion(); got != 1 {
		t.Errorf("schema_version: got %d", got)
	}
	if n := len(set.OverridesMap()); n != 2 {
		t.Errorf("overrides count: got %d", n)
	}
}

func TestPlaylistLabelSet_NilIsNoOp(t *testing.T) {
	var set *PlaylistLabelSet
	if got := set.Display("Super Fiesta Fête"); got != "Super Fiesta Fête" {
		t.Errorf("nil set must passthrough: got %q", got)
	}
	if m := set.OverridesMap(); len(m) != 0 {
		t.Errorf("nil OverridesMap must be empty: got %v", m)
	}
}

func TestLoadPlaylistLabels_Invalid(t *testing.T) {
	cases := map[string]string{
		"meta manquant": `
[overrides]
"A" = "B"
`,
		"schema_version zero": `
[meta]
title_slug = "halo_5"
[overrides]
"A" = "B"
`,
		"override vide": `
[meta]
title_slug     = "halo_5"
schema_version = 1
[overrides]
"A" = ""
`,
	}
	for name, raw := range cases {
		if _, err := LoadPlaylistLabelsFromBytes("t.toml", []byte(raw)); err == nil {
			t.Errorf("%s: attendu une erreur", name)
		}
	}
}
