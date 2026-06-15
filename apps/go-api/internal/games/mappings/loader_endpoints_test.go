package mappings

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadEndpointsFromBytes_Valid(t *testing.T) {
	t.Parallel()
	const doc = `
[meta]
title_slug = "x_title"
schema_version = 2

[endpoints]
stats = "https://stats.example.test:443"
gamecms = "https://cms.example.test"
`
	set, err := LoadEndpointsFromBytes("x.toml", []byte(doc))
	if err != nil {
		t.Fatalf("LoadEndpointsFromBytes: %v", err)
	}
	if set.TitleSlug() != "x_title" {
		t.Errorf("TitleSlug = %q", set.TitleSlug())
	}
	if set.SchemaVersion() != 2 {
		t.Errorf("SchemaVersion = %d", set.SchemaVersion())
	}
	if h, ok := set.Host(EndpointStats); !ok || h != "https://stats.example.test:443" {
		t.Errorf("stats host = %q ok=%v", h, ok)
	}
	if h, ok := set.Host(EndpointGameCMS); !ok || h != "https://cms.example.test" {
		t.Errorf("gamecms host = %q ok=%v", h, ok)
	}
	// Clé non déclarée → absente.
	if _, ok := set.Host(EndpointSkill); ok {
		t.Errorf("skill devrait être absent")
	}
	if got := set.Keys(); len(got) != 2 {
		t.Errorf("Keys = %v, want 2", got)
	}
}

func TestLoadEndpointsFromBytes_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		doc  string
		want string
	}{
		{
			name: "section absente",
			doc:  "[meta]\ntitle_slug = \"x\"\nschema_version = 1\n",
			want: "section [endpoints] absente ou vide",
		},
		{
			name: "meta manquant",
			doc:  "[endpoints]\nstats = \"https://a.test\"\n",
			want: "title_slug manquant",
		},
		{
			name: "clé inconnue",
			doc:  "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[endpoints]\nbogus = \"https://a.test\"\n",
			want: "clé inconnue",
		},
		{
			name: "host vide",
			doc:  "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[endpoints]\nstats = \"\"\n",
			want: "host vide",
		},
		{
			name: "host non-https",
			doc:  "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[endpoints]\nstats = \"http://a.test\"\n",
			want: "non-https",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadEndpointsFromBytes("x.toml", []byte(tc.doc))
			if err == nil {
				t.Fatalf("attendu une erreur")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %q, want contains %q", err.Error(), tc.want)
			}
		})
	}
}

// TestLoadHaloInfiniteEndpointsTOML — golden de parité : les hosts du vrai
// constants.toml du repo doivent correspondre byte-pour-byte aux const Go
// actuels de la couche d'ingestion (sync/platform). Toute dérive casse ce test.
func TestLoadHaloInfiniteEndpointsTOML(t *testing.T) {
	t.Parallel()
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
	tomlPath := filepath.Join(repoRoot, "config", "titles", "halo_infinite", "constants.toml")

	set, err := LoadEndpointsFromFile(tomlPath)
	if err != nil {
		t.Fatalf("LoadEndpointsFromFile(%s): %v", tomlPath, err)
	}
	if set.TitleSlug() != "halo_infinite" {
		t.Errorf("TitleSlug = %q", set.TitleSlug())
	}

	// Valeurs ATTENDUES = const Go actuels (cf. commentaires constants.toml).
	want := map[EndpointKey]string{
		EndpointStats:        "https://halostats.svc.halowaypoint.com:443",
		EndpointGameCMS:      "https://gamecms-hacs.svc.halowaypoint.com",
		EndpointEconomy:      "https://economy.svc.halowaypoint.com",
		EndpointSkill:        "https://skill.svc.halowaypoint.com:443",
		EndpointUGCFilm:      "https://discovery-infiniteugc.svc.halowaypoint.com",
		EndpointDiscoveryUGC: "https://discovery-infiniteugc.svc.halowaypoint.com",
		EndpointChallenges:   "https://halostats.svc.halowaypoint.com",
		EndpointNameplate:    "https://gamecms-hacs.svc.halowaypoint.com",
	}
	for key, wantHost := range want {
		got, ok := set.Host(key)
		if !ok {
			t.Errorf("endpoint %q absent du constants.toml HI", key)
			continue
		}
		if got != wantHost {
			t.Errorf("endpoint %q = %q, want %q (parité const Go)", key, got, wantHost)
		}
	}
	// Aucune clé inattendue (les 8 axes, ni plus ni moins).
	if got := len(set.Keys()); got != len(want) {
		t.Errorf("nombre d'endpoints = %d, want %d", got, len(want))
	}
}
