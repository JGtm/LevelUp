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

// TestIsReplayArtifactFile : le prédicat unique du nom d'un artefact. Il reconnaît la forme
// compressée et la forme brute ancienne ; il refuse la marque de dérivation, l'en-tête, une
// sauvegarde manuelle et un fichier d'un autre type.
func TestIsReplayArtifactFile(t *testing.T) {
	cas := []struct {
		name string
		want bool
	}{
		{"000d5950.json.gz", true},
		{"000d5950.json", true},
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
		{"000d5950.json", "000d5950"},
		{filepath.Join("d", "000d5950.json.gz"), filepath.Join("d", "000d5950")},
		{"000d5950.meta", "000d5950.meta"},
		{"000d5950.derived.json", "000d5950.derived"},
	}
	for _, c := range cas {
		if got := ReplayArtifactStem(c.in); got != c.want {
			t.Errorf("ReplayArtifactStem(%q) = %q, attendu %q", c.in, got, c.want)
		}
	}
}

// TestReplayArtifactCandidates : un chemin d'artefact, sous l'une ou l'autre forme, désigne les
// deux fichiers possibles du même match, la forme compressée d'abord.
func TestReplayArtifactCandidates(t *testing.T) {
	for _, in := range []string{filepath.Join("d", "a1b2c3d4.json.gz"), filepath.Join("d", "a1b2c3d4.json")} {
		gz, brut := ReplayArtifactCandidates(in)
		if gz != filepath.Join("d", "a1b2c3d4.json.gz") || brut != filepath.Join("d", "a1b2c3d4.json") {
			t.Errorf("ReplayArtifactCandidates(%q) = %q, %q", in, gz, brut)
		}
	}
}

func TestReplayArtifactPath_SuffixeCompresse(t *testing.T) {
	p := NewPathResolver("/depot")
	if got := filepath.ToSlash(p.ReplayArtifactPath(DefaultSlug, "000d5950")); got != "/depot/data/cache/replays/halo_infinite/000d5950.json.gz" {
		t.Errorf("chemin = %s", got)
	}
}
