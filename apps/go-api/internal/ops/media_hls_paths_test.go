package ops

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestHLSPathsFor_Subfolders : deux sous-dossiers qui contiennent un clip de même nom ont
// chacun leur dossier de conversion ; la racine garde hls/{nom}.
func TestHLSPathsFor_Subfolders(t *testing.T) {
	capturesDir := filepath.Join("base", "GT")
	rootOut, rootRel := HLSPathsFor(capturesDir, "base", "GT", filepath.Join(capturesDir, "clip.mkv"))
	aOut, aRel := HLSPathsFor(capturesDir, "base", "GT", filepath.Join(capturesDir, "a", "clip.mkv"))
	bOut, bRel := HLSPathsFor(capturesDir, "base", "GT", filepath.Join(capturesDir, "b", "clip.mkv"))

	if rootOut != filepath.Join(capturesDir, "hls", "clip") || rootRel != "GT/hls/clip/master.m3u8" {
		t.Errorf("racine : (%q, %q), want hls/clip inchangé", rootOut, rootRel)
	}
	if aOut == bOut || aRel == bRel || aOut == rootOut {
		t.Errorf("dossiers non distincts : racine %q, a %q, b %q", rootOut, aOut, bOut)
	}
	for _, rel := range []string{aRel, bRel} {
		if !strings.HasPrefix(rel, "GT/hls/clip_") || !strings.HasSuffix(rel, "/master.m3u8") {
			t.Errorf("chemin stocké %q, want GT/hls/clip_<8 hex>/master.m3u8", rel)
		}
		suffix := strings.TrimSuffix(strings.TrimPrefix(rel, "GT/hls/clip_"), "/master.m3u8")
		if len(suffix) != 8 {
			t.Errorf("suffixe %q, want 8 caractères hexadécimaux", suffix)
		}
	}
}

// TestHLSPathsFor_SubfolderCaseInsensitive : le suffixe se calcule sur le dossier relatif
// en minuscules et séparateur « / ».
func TestHLSPathsFor_SubfolderCaseInsensitive(t *testing.T) {
	capturesDir := filepath.Join("base", "GT")
	_, lower := HLSPathsFor(capturesDir, "base", "GT", filepath.Join(capturesDir, "sess", "un", "clip.mkv"))
	_, upper := HLSPathsFor(capturesDir, "base", "GT", filepath.Join(capturesDir, "Sess", "UN", "clip.mkv"))
	if lower != upper {
		t.Errorf("%q et %q, want le même dossier (casse ignorée)", lower, upper)
	}
	_, other := HLSPathsFor(capturesDir, "base", "GT", filepath.Join(capturesDir, "sess", "deux", "clip.mkv"))
	if lower == other {
		t.Errorf("deux dossiers différents donnent le même chemin %q", lower)
	}
}
