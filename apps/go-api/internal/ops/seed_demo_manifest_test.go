package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
)

func TestDemoManifest_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "halo_infinite.json")
	m := &DemoManifest{
		Version:        demoManifestVersion,
		TitleSlug:      "halo_infinite",
		SourceGamertag: "JGtm",
		Corpus: DemoManifestCorpus{
			SoloMatchIDs:   []string{"a", "b"},
			SquadMatchIDs:  []string{"b", "c"},
			RankedMatchIDs: []string{"d"},
			MediaMatchIDs:  []string{"a", "e"},
		},
	}
	if err := writeDemoManifest(path, m); err != nil {
		t.Fatal(err)
	}
	got, found, err := LoadDemoManifest(path)
	if err != nil || !found {
		t.Fatalf("load: found=%v err=%v", found, err)
	}
	if got.TitleSlug != "halo_infinite" || got.SourceGamertag != "JGtm" {
		t.Errorf("métadonnées perdues au round-trip: %+v", got)
	}
	// union ordre/dedup solo→squad→ranked→media
	want := []string{"a", "b", "c", "d", "e"}
	if gotIDs := got.CorpusMatchIDs(); !strSliceEqual(gotIDs, want) {
		t.Errorf("CorpusMatchIDs = %v, want %v", gotIDs, want)
	}
}

func TestLoadDemoManifest_Missing(t *testing.T) {
	_, found, err := LoadDemoManifest(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("fichier absent ne doit pas être une erreur: %v", err)
	}
	if found {
		t.Error("found doit être false pour un fichier absent")
	}
}

func TestLoadDemoManifest_InvalidIsHardError(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"bad_json":     `{not json`,
		"missing_ver":  `{"title_slug":"x","corpus":{"solo_match_ids":["a"]}}`,
		"bad_ver":      `{"version":"99","corpus":{"solo_match_ids":["a"]}}`,
		"empty_corpus": `{"version":"1","corpus":{}}`,
	}
	for name, content := range cases {
		p := filepath.Join(dir, name+".json")
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, _, err := LoadDemoManifest(p); err == nil {
			t.Errorf("%s : un manifeste présent mais invalide doit être une erreur dure", name)
		}
	}
}

func TestDemoManifest_CorpusMatchIDs_Dedup(t *testing.T) {
	m := &DemoManifest{Corpus: DemoManifestCorpus{
		SoloMatchIDs:  []string{"x", "y"},
		SquadMatchIDs: []string{"y", "z"},
	}}
	if got := m.CorpusMatchIDs(); !strSliceEqual(got, []string{"x", "y", "z"}) {
		t.Errorf("dedup attendu, got %v", got)
	}
}

func TestPathResolver_DemoManifestPath(t *testing.T) {
	pr := titlePkg.NewPathResolver("/repo")
	got := pr.DemoManifestPath("JGtm", "halo_5")
	if want := filepath.Join("/repo", "config", "demo", "JGtm", "halo_5.json"); got != want {
		t.Errorf("DemoManifestPath = %q, want %q", got, want)
	}
	// slug vide → DefaultSlug
	gotDef := pr.DemoManifestPath("JGtm", "")
	if want := filepath.Join("/repo", "config", "demo", "JGtm", titlePkg.DefaultSlug+".json"); gotDef != want {
		t.Errorf("slug vide: %q, want %q", gotDef, want)
	}
}

func strSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Le manifeste émis est versionné : il ne porte ni l'identifiant réel du joueur source ni aucun
// champ d'identifiant de joueur, quel que soit le xuid des options.
func TestNewEmittedManifest_NEcritAucunIdentifiantDeJoueur(t *testing.T) {
	const xuidReel = "2533860000000001"
	opts := SeedDemoOptions{TitleSlug: "halo_infinite", SourceLabel: "JGtm", SourceXUID: xuidReel}
	dc := dynamicCorpus{solo: []string{"a"}, squad: []string{"b"}, ranked: []string{"c"}, media: []string{"d"}}

	m := newEmittedManifest(opts, dc, time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC))

	path := filepath.Join(t.TempDir(), "halo_infinite.json")
	if err := writeDemoManifest(path, m); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), xuidReel) {
		t.Errorf("le manifeste émis contient le xuid de la source :\n%s", raw)
	}
	if strings.Contains(strings.ToLower(string(raw)), "xuid") {
		t.Errorf("le manifeste émis porte un champ d'identifiant de joueur :\n%s", raw)
	}
	if m.SourceGamertag != "JGtm" || m.GeneratedAt != "2026-11-24T12:00:00Z" {
		t.Errorf("métadonnées de traçabilité perdues : %+v", m)
	}
}

// Un manifeste déjà versionné qui porte encore l'ancien champ se lit sans erreur.
func TestLoadDemoManifest_IgnoreUnAncienChampXUID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ancien.json")
	if err := os.WriteFile(path, []byte(`{"version":"1","title_slug":"x","source_xuid":"1","corpus":{"solo_match_ids":["a"]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, found, err := LoadDemoManifest(path); err != nil || !found {
		t.Fatalf("load: found=%v err=%v", found, err)
	}
}
