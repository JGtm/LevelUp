package ops

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestPathKeyOn(t *testing.T) {
	dirty := filepath.Join("A", "B", "..", "Clip.MP4")
	clean := filepath.Join("A", "Clip.MP4")

	if got := pathKeyOn("linux", dirty); got != clean {
		t.Errorf("linux: pathKeyOn(%q) = %q, want %q (nettoyé, casse conservée)", dirty, got, clean)
	}
	want := filepath.Join("a", "clip.mp4")
	if got := pathKeyOn("windows", dirty); got != want {
		t.Errorf("windows: pathKeyOn(%q) = %q, want %q (nettoyé, sans casse)", dirty, got, want)
	}
}

func TestPathKey_UsesCurrentSystem(t *testing.T) {
	p := filepath.Join("A", "Clip.MP4")
	if got, want := pathKey(p), pathKeyOn(runtime.GOOS, p); got != want {
		t.Errorf("pathKey(%q) = %q, want %q", p, got, want)
	}
}
