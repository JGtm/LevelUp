package title

import (
	"path/filepath"
	"testing"
)

func TestReplayArtifactFileName_EstCompresse(t *testing.T) {
	if got := ReplayArtifactFileName("000d5950-1234-4abc-9def-0123456789ab"); got != "000d5950.json.gz" {
		t.Errorf("ReplayArtifactFileName = %q, attendu 000d5950.json.gz", got)
	}
}

// TestIsReplayArtifactFile : le prédicat unique du nom d'un artefact. Il ne reconnaît que la forme
// compressée ; il refuse le `.json` brut d'une version ancienne, la marque de dérivation, l'en-tête,
// une sauvegarde manuelle et un fichier d'un autre type.
func TestIsReplayArtifactFile(t *testing.T) {
	cas := []struct {
		name string
		want bool
	}{
		{"000d5950.json.gz", true},
		{"000d5950.json", false},
		{"000d5950.derived.json", false},
		{"000d5950.meta", false},
		{"000d5950.json.ancien-11juillet", false},
		{"000d5950.json.tmp", false},
		{"000d5950.gz", false},
		{".json", false},
		{".json.gz", false},
		{"", false},
	}
	for _, c := range cas {
		if got := IsReplayArtifactFile(c.name); got != c.want {
			t.Errorf("IsReplayArtifactFile(%q) = %v, attendu %v", c.name, got, c.want)
		}
	}
}

func TestReplayArtifactStem(t *testing.T) {
	cas := []struct{ in, want string }{
		{"000d5950.json.gz", "000d5950"},
		{"000d5950.json", "000d5950.json"},
		{filepath.Join("d", "000d5950.json.gz"), filepath.Join("d", "000d5950")},
		{"000d5950.meta", "000d5950.meta"},
		{"000d5950.derived.json", "000d5950.derived.json"},
	}
	for _, c := range cas {
		if got := ReplayArtifactStem(c.in); got != c.want {
			t.Errorf("ReplayArtifactStem(%q) = %q, attendu %q", c.in, got, c.want)
		}
	}
}

func TestReplayArtifactPath_SuffixeCompresse(t *testing.T) {
	p := NewPathResolver("/depot")
	if got := filepath.ToSlash(p.ReplayArtifactPath(DefaultSlug, "000d5950")); got != "/depot/data/cache/replays/halo_infinite/000d5950.json.gz" {
		t.Errorf("chemin = %s", got)
	}
}
