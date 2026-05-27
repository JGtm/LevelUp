package media

import "testing"

func TestIsWebNativeVideo(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ext  string
		want bool
	}{
		{".mp4", true},
		{".MP4", true}, // case insensitive
		{".webm", true},
		{".mov", true},
		{".mkv", false},
		{".avi", false},
		{".png", false},
		{"", false},
		{"mp4", false}, // sans le point
	}
	for _, c := range cases {
		if got := IsWebNativeVideo(c.ext); got != c.want {
			t.Errorf("IsWebNativeVideo(%q) = %v, want %v", c.ext, got, c.want)
		}
	}
}

func TestRequiresRemux(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ext  string
		want bool
	}{
		{".mkv", true},
		{".MKV", true},
		{".avi", true},
		{".mp4", false},
		{".webm", false},
		{".mov", false},
		{".png", false},
		{"", false},
	}
	for _, c := range cases {
		if got := RequiresRemux(c.ext); got != c.want {
			t.Errorf("RequiresRemux(%q) = %v, want %v", c.ext, got, c.want)
		}
	}
}

func TestRemuxAndWebNativeAreDisjoint(t *testing.T) {
	t.Parallel()
	for _, ext := range VideoExtensions {
		web := IsWebNativeVideo(ext)
		remux := RequiresRemux(ext)
		if web && remux {
			t.Errorf("ext %q is both web-native AND remux-required — must be disjoint", ext)
		}
	}
}

func TestStemAndExt(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in       string
		wantStem string
		wantExt  string
	}{
		{"foo.mp4", "foo", ".mp4"},
		{"path/to/Bar.MKV", "Bar", ".mkv"},
		{"no-ext", "no-ext", ""},
		{"Halo Infinite 2026-04-19 17-22-23.mp4", "Halo Infinite 2026-04-19 17-22-23", ".mp4"},
		// Note : filepath.Ext(".hidden") = ".hidden" en stdlib Go — comportement
		// accepté tel quel (les médias n'utilisent jamais de noms dotfile).
	}
	for _, c := range cases {
		stem, ext := StemAndExt(c.in)
		if stem != c.wantStem || ext != c.wantExt {
			t.Errorf("StemAndExt(%q) = (%q, %q), want (%q, %q)",
				c.in, stem, ext, c.wantStem, c.wantExt)
		}
	}
}

func TestChooseAudioMap_AllOpus(t *testing.T) {
	t.Parallel()
	got, err := chooseAudioMap([]string{"opus", "opus", "opus", "opus"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "0:a" {
		t.Errorf("got %q, want 0:a (toutes pistes copiées)", got)
	}
}

func TestChooseAudioMap_MixedFirstOpus(t *testing.T) {
	t.Parallel()
	got, err := chooseAudioMap([]string{"opus", "ac3"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "0:a:0" {
		t.Errorf("got %q, want 0:a:0 (fallback première piste)", got)
	}
}

func TestChooseAudioMap_FirstIncompatible(t *testing.T) {
	t.Parallel()
	if _, err := chooseAudioMap([]string{"aac", "opus"}); err == nil {
		t.Errorf("expected error when first audio track is not webm-compatible, got nil")
	}
}

func TestChooseAudioMap_NoAudio(t *testing.T) {
	t.Parallel()
	got, err := chooseAudioMap(nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "0:a?" {
		t.Errorf("got %q, want 0:a? (optionnel quand pas d'audio)", got)
	}
}

func TestWebMCompatibleVideoCodec(t *testing.T) {
	t.Parallel()
	if !isWebMCompatibleVideoCodec("av1") {
		t.Error("av1 should be webm-compatible")
	}
	if !isWebMCompatibleVideoCodec("vp9") {
		t.Error("vp9 should be webm-compatible")
	}
	if isWebMCompatibleVideoCodec("h264") {
		t.Error("h264 should NOT be webm-compatible")
	}
	if isWebMCompatibleVideoCodec("") {
		t.Error("empty codec should NOT be webm-compatible")
	}
}
