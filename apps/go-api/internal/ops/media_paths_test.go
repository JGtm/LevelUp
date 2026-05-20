package ops

import (
	"path/filepath"
	"testing"
)

func TestMediaPathStore_ToRel_MultiPlayerLayout(t *testing.T) {
	base := filepath.FromSlash("C:/Users/Guillaume/Videos/Captures")
	store := MediaPathStore{CapturesBase: base}

	abs := filepath.FromSlash("C:/Users/Guillaume/Videos/Captures/Madina97294/Halo Infinite 2026-04-19 17-43-35.mp4")
	got := store.ToRel(abs, "Madina97294")
	want := "Madina97294/Halo Infinite 2026-04-19 17-43-35.mp4"
	if got != want {
		t.Errorf("ToRel multi-player: got %q, want %q", got, want)
	}
}

func TestMediaPathStore_ToRel_ThumbsSubdir(t *testing.T) {
	base := filepath.FromSlash("/srv/captures")
	store := MediaPathStore{CapturesBase: base}

	abs := filepath.FromSlash("/srv/captures/JGtm/thumbs/Replay 2026-03-27 23-17-43_bd811d72870d.webp")
	got := store.ToRel(abs, "JGtm")
	want := "JGtm/thumbs/Replay 2026-03-27 23-17-43_bd811d72870d.webp"
	if got != want {
		t.Errorf("ToRel thumbs: got %q, want %q", got, want)
	}
}

func TestMediaPathStore_ToRel_SinglePlayerLayoutPrefixesOwner(t *testing.T) {
	base := filepath.FromSlash("/captures")
	store := MediaPathStore{CapturesBase: base}

	// Fichier directement sous /captures sans sous-dossier owner_slug
	abs := filepath.FromSlash("/captures/Halo Infinite.mp4")
	got := store.ToRel(abs, "spartan")
	want := "spartan/Halo Infinite.mp4"
	if got != want {
		t.Errorf("ToRel single-player: got %q, want %q", got, want)
	}
}

func TestMediaPathStore_ToRel_OutsideBaseReturnsEmpty(t *testing.T) {
	store := MediaPathStore{CapturesBase: filepath.FromSlash("/srv/captures")}

	abs := filepath.FromSlash("/var/log/file.mp4")
	if got := store.ToRel(abs, "spartan"); got != "" {
		t.Errorf("ToRel outside base: got %q, want \"\"", got)
	}
}

func TestMediaPathStore_ToRel_EmptyBaseReturnsEmpty(t *testing.T) {
	store := MediaPathStore{CapturesBase: ""}
	abs := filepath.FromSlash("/srv/captures/spartan/x.mp4")
	if got := store.ToRel(abs, "spartan"); got != "" {
		t.Errorf("ToRel empty base: got %q, want \"\"", got)
	}
}

func TestMediaPathStore_ToAbs_RelativeJoinedWithBase(t *testing.T) {
	base := filepath.FromSlash("/srv/captures")
	store := MediaPathStore{CapturesBase: base}

	stored := "Madina97294/thumbs/x.webp"
	got := store.ToAbs(stored)
	want := filepath.Join(base, "Madina97294", "thumbs", "x.webp")
	if got != want {
		t.Errorf("ToAbs relative: got %q, want %q", got, want)
	}
}

func TestMediaPathStore_ToAbs_AbsolutePassThrough(t *testing.T) {
	store := MediaPathStore{CapturesBase: filepath.FromSlash("/srv/captures")}

	// Path absolu valide cross-platform via filepath.Abs (Windows: C:\..., Unix: /...).
	legacy, err := filepath.Abs(filepath.FromSlash("/old/layout/x.mp4"))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	if got := store.ToAbs(legacy); got != legacy {
		t.Errorf("ToAbs absolute pass-through: got %q, want %q", got, legacy)
	}
}

func TestMediaPathStore_ToAbs_EmptyBasePassThrough(t *testing.T) {
	store := MediaPathStore{CapturesBase: ""}
	rel := "spartan/x.mp4"
	if got := store.ToAbs(rel); got != rel {
		t.Errorf("ToAbs empty base: got %q, want %q", got, rel)
	}
}

func TestMediaPathStore_IsRel(t *testing.T) {
	store := MediaPathStore{CapturesBase: filepath.FromSlash("/srv/captures")}

	if !store.IsRel("spartan/x.mp4") {
		t.Error("expected spartan/x.mp4 to be relative")
	}
	abs, err := filepath.Abs(filepath.FromSlash("/old/x.mp4"))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	if store.IsRel(abs) {
		t.Errorf("expected absolute path %q to not be relative", abs)
	}
	if store.IsRel("") {
		t.Error("expected empty to not be relative")
	}
}
