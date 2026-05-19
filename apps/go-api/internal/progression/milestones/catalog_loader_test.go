package milestones

import (
	"strings"
	"testing"
)

// catalog_loader_test.go — parsing TOML + validation.

func TestParseCatalog_ValidEntries(t *testing.T) {
	in := []byte(`
[meta]
schema_version = 1
title_slug = "halo_infinite"

[[milestones]]
id = "halo_infinite.matches.100"
metric = "matches_played"
threshold = 100
title_fr = "Centurion"
title_en = "Centurion"
icon = "milestone_100_matches"

[[milestones]]
id = "halo_infinite.wins.50"
metric = "wins"
threshold = 50
title_fr = "Vainqueur"
title_en = "Winner"
`)

	out, err := parseCatalogBytes(in)
	if err != nil {
		t.Fatalf("parseCatalogBytes: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if out[0].TitleSlug != "halo_infinite" {
		t.Errorf("TitleSlug propagation failed: got %q", out[0].TitleSlug)
	}
	if out[0].Threshold != 100 {
		t.Errorf("Threshold = %v, want 100", out[0].Threshold)
	}
	if out[1].TitleFR != "Vainqueur" {
		t.Errorf("TitleFR = %q, want Vainqueur", out[1].TitleFR)
	}
}

func TestParseCatalog_MissingTitleSlug(t *testing.T) {
	in := []byte(`
[[milestones]]
id = "x"
metric = "m"
threshold = 1
title_fr = "fr"
title_en = "en"
`)
	_, err := parseCatalogBytes(in)
	if err == nil || !strings.Contains(err.Error(), "title_slug") {
		t.Errorf("expected error mentioning title_slug, got %v", err)
	}
}

func TestParseCatalog_MissingFields(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			"missing id",
			`[meta]
title_slug = "halo_infinite"
[[milestones]]
metric = "m"
threshold = 1
title_fr = "fr"
title_en = "en"`,
			"missing id",
		},
		{
			"missing metric",
			`[meta]
title_slug = "halo_infinite"
[[milestones]]
id = "x"
threshold = 1
title_fr = "fr"
title_en = "en"`,
			"missing metric",
		},
		{
			"missing title_en",
			`[meta]
title_slug = "halo_infinite"
[[milestones]]
id = "x"
metric = "m"
threshold = 1
title_fr = "fr"`,
			"missing title_en or title_fr",
		},
		{
			"zero threshold",
			`[meta]
title_slug = "halo_infinite"
[[milestones]]
id = "x"
metric = "m"
threshold = 0
title_fr = "fr"
title_en = "en"`,
			"threshold must be > 0",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := parseCatalogBytes([]byte(c.in))
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %q does not contain %q", err.Error(), c.want)
			}
		})
	}
}

func TestParseCatalog_InvalidTOML(t *testing.T) {
	in := []byte(`this is not toml = == invalid`)
	_, err := parseCatalogBytes(in)
	if err == nil {
		t.Errorf("expected parse error")
	}
}
