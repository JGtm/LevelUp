package ops

import (
	"strings"
	"testing"
)

// Sortie simulée de `ffmpeg -hide_banner -encoders` (en-tête + légende +
// séparateur + lignes de composant). Inclut les 3 encodeurs requis plus du bruit
// réaliste (deux entrées libwebp, encodeurs non requis). Aucun ffmpeg réel n'est
// exécuté : le parsing est testé sur cette fixture injectée.
const sampleEncoders = `Encoders:
 V..... = Video
 A..... = Audio
 S..... = Subtitle
 .F.... = Frame-level multithreading
 ..S... = Slice-level multithreading
 ...X.. = Codec is experimental
 ....B. = Supports draw_horiz_band
 .....D = Supports direct rendering method 1
 ------
 V....D libx264              libx264 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10 (codec h264)
 V....D libx265              libx265 H.265 / HEVC (codec hevc)
 A....D aac                  AAC (Advanced Audio Coding)
 A....D libmp3lame           libmp3lame MP3 (MPEG audio layer 3) (codec mp3)
 V....D libwebp_anim         libwebp WebP muxer (codec webp)
 V....D libwebp              libwebp WebP image format (codec webp)
`

// Sortie simulée de `ffmpeg -hide_banner -muxers` : légende, séparateur, puis
// lignes de muxer (flags " E" pour mux-only, "DE" pour demux+mux).
const sampleMuxers = `Muxers:
 D. = Demuxing supported
 .E = Muxing supported
 --
  E hls             Apple HTTP Live Streaming
  E mp4             MP4 (MPEG-4 Part 14)
  E mov             QuickTime / MOV
 DE matroska        Matroska
  E webm            WebM
`

func TestParseFFmpegComponents_Encoders(t *testing.T) {
	got := parseFFmpegComponents(sampleEncoders)
	for _, want := range []string{"libx264", "aac", "libwebp", "libwebp_anim", "libx265", "libmp3lame"} {
		if !got[want] {
			t.Errorf("encodeur %q attendu absent du set parsé", want)
		}
	}
	// La légende et les en-têtes ne doivent JAMAIS produire de composant.
	for _, unwanted := range []string{"=", "Video", "Audio", "Subtitle", "Encoders:"} {
		if got[unwanted] {
			t.Errorf("entrée de légende/en-tête %q parsée à tort comme composant", unwanted)
		}
	}
}

func TestParseFFmpegComponents_Muxers(t *testing.T) {
	got := parseFFmpegComponents(sampleMuxers)
	for _, want := range []string{"hls", "mp4", "mov", "matroska", "webm"} {
		if !got[want] {
			t.Errorf("muxer %q attendu absent du set parsé", want)
		}
	}
	if got["="] || got["Muxers:"] {
		t.Error("légende/en-tête muxers parsée à tort comme composant")
	}
}

func TestParseFFmpegComponents_Empty(t *testing.T) {
	if len(parseFFmpegComponents("")) != 0 {
		t.Error("une sortie vide doit produire un set vide")
	}
}

func TestMissingComponents_AllPresent(t *testing.T) {
	present := parseFFmpegComponents(sampleEncoders)
	if missing := missingComponents(present, requiredEncoders); len(missing) != 0 {
		t.Errorf("aucun encodeur requis ne devrait manquer, obtenu %v", missing)
	}
	presentMux := parseFFmpegComponents(sampleMuxers)
	if missing := missingComponents(presentMux, requiredMuxers); len(missing) != 0 {
		t.Errorf("aucun muxer requis ne devrait manquer, obtenu %v", missing)
	}
}

func TestMissingComponents_DetectsAbsence(t *testing.T) {
	// ffmpeg compilé SANS libx264 (l'encodeur est absent de la liste).
	stripped := strings.ReplaceAll(sampleEncoders,
		" V....D libx264              libx264 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10 (codec h264)\n", "")
	missing := missingComponents(parseFFmpegComponents(stripped), requiredEncoders)
	if len(missing) != 1 || missing[0] != "libx264" {
		t.Fatalf("attendu [libx264] manquant, obtenu %v", missing)
	}
}

func TestMissingComponents_MuxerAbsence(t *testing.T) {
	stripped := strings.ReplaceAll(sampleMuxers, "  E mp4             MP4 (MPEG-4 Part 14)\n", "")
	missing := missingComponents(parseFFmpegComponents(stripped), requiredMuxers)
	if len(missing) != 1 || missing[0] != "mp4" {
		t.Fatalf("attendu [mp4] manquant, obtenu %v", missing)
	}
}

func TestFirstLine(t *testing.T) {
	cases := map[string]string{
		"ffmpeg version 5.1.6-0+deb12u1 Copyright (c) 2000-2024\nbuilt with gcc 12\n": "ffmpeg version 5.1.6-0+deb12u1 Copyright (c) 2000-2024",
		"single line no newline": "single line no newline",
		"":                       "",
	}
	for in, want := range cases {
		if got := firstLine(in); got != want {
			t.Errorf("firstLine(%q) = %q, attendu %q", in, got, want)
		}
	}
}

func TestMediaToolingReport_Summary_AllOK(t *testing.T) {
	r := MediaToolingReport{
		FFmpegFound: true, FFmpegPath: "/usr/bin/ffmpeg", FFmpegVersion: "ffmpeg version 5.1.6",
		FFprobeFound: true, FFprobePath: "/usr/bin/ffprobe",
		CapabilitiesProbed: true,
	}
	out := r.Summary()
	for _, want := range []string{"/usr/bin/ffmpeg", "ffmpeg version 5.1.6", "/usr/bin/ffprobe", "[OK] encodeurs requis", "[OK] muxers requis"} {
		if !strings.Contains(out, want) {
			t.Errorf("Summary devrait contenir %q, obtenu:\n%s", want, out)
		}
	}
	if strings.Contains(out, "MANQUANT") {
		t.Errorf("Summary complet ne devrait pas contenir MANQUANT:\n%s", out)
	}
}

func TestMediaToolingReport_Summary_MissingEncoder(t *testing.T) {
	r := MediaToolingReport{
		FFmpegFound: true, FFmpegPath: "/usr/bin/ffmpeg",
		FFprobeFound: true, FFprobePath: "/usr/bin/ffprobe",
		CapabilitiesProbed: true,
		MissingEncoders:    []string{"libx264"},
	}
	out := r.Summary()
	if !strings.Contains(out, "MANQUANT: libx264") {
		t.Errorf("Summary devrait signaler libx264 manquant, obtenu:\n%s", out)
	}
	if !strings.Contains(out, "[KO] encodeurs requis") {
		t.Errorf("Summary devrait marquer [KO] la ligne encodeurs, obtenu:\n%s", out)
	}
}

func TestMediaToolingReport_Summary_FFmpegAbsent(t *testing.T) {
	out := MediaToolingReport{}.Summary()
	if !strings.Contains(out, "[KO] ffmpeg") || !strings.Contains(out, "[KO] ffprobe") {
		t.Errorf("Summary sans ffmpeg/ffprobe devrait les marquer [KO], obtenu:\n%s", out)
	}
}
