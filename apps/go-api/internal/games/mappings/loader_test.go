package mappings

import (
	"strings"
	"testing"
)

const minimalValidTOML = `
[meta]
title_slug     = "test_title"
schema_version = 1

[fields.kills]
labels        = { en = "Kills", fr = "Éliminations" }
storage_unit  = "count"
display_unit  = "count"
format        = "integer"
display_order = 10
group         = "combat"
`

func TestLoadFieldsFromBytesValid(t *testing.T) {
	t.Parallel()
	set, err := LoadFieldsFromBytes("inline.toml", []byte(minimalValidTOML))
	if err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
	if set.TitleSlug() != "test_title" {
		t.Errorf("title_slug = %q", set.TitleSlug())
	}
	if got := set.SchemaVersion(); got != 1 {
		t.Errorf("schema_version = %d", got)
	}
}

func TestLoadFieldsFromBytesInvalid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		toml        string
		errContains string
	}{
		{
			name:        "empty file",
			toml:        ``,
			errContains: "title_slug manquant",
		},
		{
			name:        "missing schema_version",
			toml:        "[meta]\ntitle_slug = \"x\"\n",
			errContains: "schema_version doit être > 0",
		},
		{
			name:        "no fields section",
			toml:        "[meta]\ntitle_slug = \"x\"\nschema_version = 1\n",
			errContains: "aucune section",
		},
		{
			name: "unknown FieldKey",
			toml: `
[meta]
title_slug = "x"
schema_version = 1

[fields.not_a_real_field]
labels = { en = "X", fr = "X" }
storage_unit = "count"
display_unit = "count"
format = "integer"
display_order = 1
group = "combat"
`,
			errContains: "absent du canonique central",
		},
		{
			name: "missing labels.fr",
			toml: `
[meta]
title_slug = "x"
schema_version = 1

[fields.kills]
labels = { en = "Kills" }
storage_unit = "count"
display_unit = "count"
format = "integer"
display_order = 1
group = "combat"
`,
			errContains: "labels.fr manquant",
		},
		{
			name: "unknown format",
			toml: `
[meta]
title_slug = "x"
schema_version = 1

[fields.kills]
labels = { en = "Kills", fr = "Éliminations" }
storage_unit = "count"
display_unit = "count"
format = "fancy_format"
display_order = 1
group = "combat"
`,
			errContains: "format inconnu",
		},
		{
			name: "unknown storage_unit",
			toml: `
[meta]
title_slug = "x"
schema_version = 1

[fields.kills]
labels = { en = "Kills", fr = "Éliminations" }
storage_unit = "lightyears"
display_unit = "count"
format = "integer"
display_order = 1
group = "combat"
`,
			errContains: "storage_unit inconnue",
		},
		{
			name: "unsupported conversion",
			toml: `
[meta]
title_slug = "x"
schema_version = 1

[fields.kills]
labels = { en = "Kills", fr = "Éliminations" }
storage_unit = "seconds"
display_unit = "count"
format = "integer"
display_order = 1
group = "combat"
`,
			errContains: "non supportée",
		},
		{
			name: "display_order collision",
			toml: `
[meta]
title_slug = "x"
schema_version = 1

[fields.kills]
labels = { en = "Kills", fr = "Éliminations" }
storage_unit = "count"
display_unit = "count"
format = "integer"
display_order = 10
group = "combat"

[fields.deaths]
labels = { en = "Deaths", fr = "Morts" }
storage_unit = "count"
display_unit = "count"
format = "integer"
display_order = 10
group = "combat"
`,
			errContains: "collisionne",
		},
		{
			name: "description partial (only EN)",
			toml: `
[meta]
title_slug = "x"
schema_version = 1

[fields.kills]
labels = { en = "Kills", fr = "Éliminations" }
description = { en = "Total kills." }
storage_unit = "count"
display_unit = "count"
format = "integer"
display_order = 1
group = "combat"
`,
			errContains: "description.en et description.fr",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := LoadFieldsFromBytes("inline.toml", []byte(tc.toml))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.errContains)
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("err = %v, want contain %q", err, tc.errContains)
			}
		})
	}
}

func TestFieldMappingLabelFallback(t *testing.T) {
	t.Parallel()
	set, err := LoadFieldsFromBytes("x.toml", []byte(minimalValidTOML))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	m, ok := set.Get("kills")
	if !ok {
		t.Fatal("kills introuvable")
	}

	cases := []struct {
		locale       string
		wantLabel    string
		wantFallback bool
	}{
		{"fr", "Éliminations", false},
		{"en", "Kills", false},
		{"xx", "Kills", true}, // fallback EN
		{"", "Kills", true},   // fallback EN
	}
	for _, tc := range cases {
		got, fb := m.Label(tc.locale)
		if got != tc.wantLabel || fb != tc.wantFallback {
			t.Errorf("Label(%q) = (%q, %v), want (%q, %v)", tc.locale, got, fb, tc.wantLabel, tc.wantFallback)
		}
	}
}

func TestFieldMappingSetOrderedByGroup(t *testing.T) {
	t.Parallel()
	tomlBody := `
[meta]
title_slug = "x"
schema_version = 1

[fields.kills]
labels = { en = "Kills", fr = "Éliminations" }
storage_unit = "count"
display_unit = "count"
format = "integer"
display_order = 30
group = "combat"

[fields.deaths]
labels = { en = "Deaths", fr = "Morts" }
storage_unit = "count"
display_unit = "count"
format = "integer"
display_order = 10
group = "combat"

[fields.assists]
labels = { en = "Assists", fr = "Assistances" }
storage_unit = "count"
display_unit = "count"
format = "integer"
display_order = 20
group = "combat"
`
	set, err := LoadFieldsFromBytes("x.toml", []byte(tomlBody))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	groups := set.OrderedByGroup()
	combat, ok := groups["combat"]
	if !ok || len(combat) != 3 {
		t.Fatalf("groups[combat] = %v", combat)
	}
	if combat[0].Key != "deaths" || combat[1].Key != "assists" || combat[2].Key != "kills" {
		t.Errorf("ordre incorrect: %v %v %v", combat[0].Key, combat[1].Key, combat[2].Key)
	}
}
