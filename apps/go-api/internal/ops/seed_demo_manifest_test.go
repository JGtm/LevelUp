package ops

import (
	"os"
	"path/filepath"
	"testing"

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
