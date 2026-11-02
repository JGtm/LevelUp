package ops

import (
	"context"
	"errors"
	"strings"
	"testing"

	mediapkg "levelup/go-api/internal/media"
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
	got := mediapkg.ParseFFmpegComponents(sampleEncoders)
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
	got := mediapkg.ParseFFmpegComponents(sampleMuxers)
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
	if len(mediapkg.ParseFFmpegComponents("")) != 0 {
		t.Error("une sortie vide doit produire un set vide")
	}
}

func TestMissingComponents_AllPresent(t *testing.T) {
	present := mediapkg.ParseFFmpegComponents(sampleEncoders)
	if missing := missingComponents(present, requiredEncoders); len(missing) != 0 {
		t.Errorf("aucun encodeur requis ne devrait manquer, obtenu %v", missing)
	}
	presentMux := mediapkg.ParseFFmpegComponents(sampleMuxers)
	if missing := missingComponents(presentMux, requiredMuxers); len(missing) != 0 {
		t.Errorf("aucun muxer requis ne devrait manquer, obtenu %v", missing)
	}
}

func TestMissingComponents_DetectsAbsence(t *testing.T) {
	// ffmpeg compilé SANS aac (l'encodeur est absent de la liste).
	stripped := strings.ReplaceAll(sampleEncoders,
		" A....D aac                  AAC (Advanced Audio Coding)\n", "")
	missing := missingComponents(mediapkg.ParseFFmpegComponents(stripped), requiredEncoders)
	if len(missing) != 1 || missing[0] != "aac" {
		t.Fatalf("attendu [aac] manquant, obtenu %v", missing)
	}
}

func TestMissingComponents_MuxerAbsence(t *testing.T) {
	stripped := strings.ReplaceAll(sampleMuxers, "  E mp4             MP4 (MPEG-4 Part 14)\n", "")
	missing := missingComponents(mediapkg.ParseFFmpegComponents(stripped), requiredMuxers)
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
		MissingEncoders:    []string{"libwebp"},
	}
	out := r.Summary()
	if !strings.Contains(out, "MANQUANT: libwebp") {
		t.Errorf("Summary devrait signaler libwebp manquant, obtenu:\n%s", out)
	}
	if !strings.Contains(out, "[KO] encodeurs requis") {
		t.Errorf("Summary devrait marquer [KO] la ligne encodeurs, obtenu:\n%s", out)
	}
}

// TestMediaToolingReport_Summary_H264 : la ligne de l'encodeur H.264 nomme l'encodeur retenu, ou
// marque [KO] quand aucun n'est utilisable.
func TestMediaToolingReport_Summary_H264(t *testing.T) {
	base := MediaToolingReport{FFmpegFound: true, FFprobeFound: true, CapabilitiesProbed: true}

	retained := base
	retained.H264Candidates = []mediapkg.H264Encoder{mediapkg.EncoderMediaFoundation, mediapkg.EncoderOpenH264}
	retained.H264Retained = mediapkg.EncoderMediaFoundation
	out := retained.Summary()
	if !strings.Contains(out, "[OK] encodeur H.264 : h264_mf (candidats : h264_mf, libopenh264)") {
		t.Errorf("Summary devrait nommer l'encodeur retenu, obtenu:\n%s", out)
	}

	none := base
	none.H264Err = mediapkg.ErrNoH264Encoder
	out = none.Summary()
	if !strings.Contains(out, "[KO] encodeur H.264 : aucun utilisable") {
		t.Errorf("Summary sans encodeur devrait marquer [KO], obtenu:\n%s", out)
	}

	unknown := base
	unknown.H264Err = errors.New("délai dépassé")
	out = unknown.Summary()
	if !strings.Contains(out, "[??] encodeur H.264 : non vérifié") || strings.Contains(out, "[KO] encodeur H.264") {
		t.Errorf("une détection non aboutie n'est pas une absence d'encodeur, obtenu:\n%s", out)
	}
}

func TestMediaToolingReport_Summary_FFmpegAbsent(t *testing.T) {
	out := MediaToolingReport{}.Summary()
	if !strings.Contains(out, "[KO] ffmpeg") || !strings.Contains(out, "[KO] ffprobe") {
		t.Errorf("Summary sans ffmpeg/ffprobe devrait les marquer [KO], obtenu:\n%s", out)
	}
}

// TestMediaToolingReport_Summary_CapabilitiesProbeErr couvre la branche « ffmpeg
// présent mais interrogation des capacités échouée » : Summary doit afficher
// « capacités : non vérifiées » et surtout NE PAS rendre les lignes
// encodeurs/muxers (qui suggéreraient à tort « tout manquant »).
func TestMediaToolingReport_Summary_CapabilitiesProbeErr(t *testing.T) {
	r := MediaToolingReport{
		FFmpegFound: true, FFmpegPath: "/usr/bin/ffmpeg",
		FFprobeFound: true, FFprobePath: "/usr/bin/ffprobe",
		CapabilitiesProbeErr: errors.New("exec: -encoders a échoué"),
	}
	out := r.Summary()
	if !strings.Contains(out, "[??] capacités : non vérifiées") {
		t.Errorf("Summary devrait signaler les capacités non vérifiées, obtenu:\n%s", out)
	}
	if strings.Contains(out, "encodeurs requis") || strings.Contains(out, "muxers requis") {
		t.Errorf("probe en échec → les lignes encodeurs/muxers ne doivent PAS être rendues (faux « manquant »), obtenu:\n%s", out)
	}
}

// TestInspectMediaTooling_FFmpegAbsentGracefulDegradation exerce la sonde quand
// ffmpeg/ffprobe sont absents du PATH (dégradation gracieuse best-effort) : aucun
// exec n'est tenté, le rapport reflète l'absence sans erreur de sonde ni panic,
// et rien n'est marqué « manquant » (on ne SAIT pas — CapabilitiesProbed=false).
func TestInspectMediaTooling_FFmpegAbsentGracefulDegradation(t *testing.T) {
	// PATH pointant vers un répertoire vide : lookupBinary ne résout rien.
	t.Setenv("PATH", t.TempDir())
	r := InspectMediaTooling(context.Background())
	if r.FFmpegFound || r.FFprobeFound {
		t.Fatalf("PATH vide → ffmpeg/ffprobe ne doivent pas être résolus, obtenu %+v", r)
	}
	if r.CapabilitiesProbed {
		t.Error("sans ffmpeg, CapabilitiesProbed doit rester false")
	}
	if r.CapabilitiesProbeErr != nil {
		t.Errorf("sans ffmpeg, aucune erreur de sonde attendue (aucun exec tenté), obtenu %v", r.CapabilitiesProbeErr)
	}
	if len(r.MissingEncoders) != 0 || len(r.MissingMuxers) != 0 {
		t.Errorf("sans probe, rien ne doit être marqué manquant: enc=%v mux=%v", r.MissingEncoders, r.MissingMuxers)
	}
	if hs := r.ToHealthStatus(); hs.FFmpeg || hs.FFprobe || hs.FFmpegVersion != "" {
		t.Errorf("la projection /health doit refléter l'absence proprement, obtenu %+v", hs)
	}
}

func TestMediaToolingReport_ToHealthStatus(t *testing.T) {
	r := MediaToolingReport{
		FFmpegFound: true, FFmpegVersion: "ffmpeg version 5.1.6",
		FFprobeFound: true,
		// Le détail encodeurs/muxers n'est PAS projeté dans /health.
		MissingEncoders: []string{"libx264"},
	}
	got := r.ToHealthStatus()
	if !got.FFmpeg || !got.FFprobe {
		t.Errorf("attendu ffmpeg/ffprobe=true, obtenu %+v", got)
	}
	if got.FFmpegVersion != "ffmpeg version 5.1.6" {
		t.Errorf("version ffmpeg non propagée: %q", got.FFmpegVersion)
	}

	absent := MediaToolingReport{}.ToHealthStatus()
	if absent.FFmpeg || absent.FFprobe || absent.FFmpegVersion != "" {
		t.Errorf("rapport vide devrait projeter des faux/vide, obtenu %+v", absent)
	}
}
