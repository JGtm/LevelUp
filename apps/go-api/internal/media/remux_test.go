package media

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// requireFFmpeg skip le test si ffmpeg ou ffprobe ne sont pas dans le PATH.
func requireFFmpeg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent du PATH — test remux ignoré")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe absent du PATH — test remux ignoré")
	}
}

// generateMKVAV1Opus crée un fichier MKV AV1+Opus de courte durée pour tester
// le remux sans dépendre d'un fixture binaire versionné. Utilise des sources
// synthétiques (testsrc + sine) — léger et déterministe.
func generateMKVAV1Opus(t *testing.T, path string, durationSec int) {
	t.Helper()
	args := []string{
		"-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc=size=320x240:rate=10:duration=" + itoa(durationSec),
		"-f", "lavfi", "-i", "sine=frequency=440:duration=" + itoa(durationSec),
		"-c:v", "libaom-av1", "-cpu-used", "8", "-b:v", "100k",
		"-c:a", "libopus", "-b:a", "32k",
		path,
	}
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("encodage MKV AV1+Opus indisponible (libaom-av1 manquant ?): %v\n%s", err, out)
	}
}

func itoa(n int) string {
	if n <= 0 {
		return "1"
	}
	switch n {
	case 1:
		return "1"
	case 2:
		return "2"
	case 3:
		return "3"
	}
	// Suffisant pour les tests : on n'utilise que de petites durées.
	return "1"
}

// webMMagic vérifie qu'un buffer commence par la signature EBML (0x1A45DFA3),
// shared par WebM et MKV. Un remux réussi doit produire un container EBML.
func looksLikeEBML(buf []byte) bool {
	return len(buf) >= 4 && buf[0] == 0x1A && buf[1] == 0x45 && buf[2] == 0xDF && buf[3] == 0xA3
}

func TestRemuxWebM_AV1Opus(t *testing.T) {
	requireFFmpeg(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "in.mkv")
	generateMKVAV1Opus(t, src, 1)

	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	plan, err := PlanRemuxWebM(ctx, src)
	if err != nil {
		t.Fatalf("PlanRemuxWebM: %v", err)
	}
	var out bytes.Buffer
	if err := StreamRemuxWebMPlan(ctx, src, plan, &out); err != nil {
		t.Fatalf("StreamRemuxWebMPlan: %v", err)
	}

	if out.Len() == 0 {
		t.Fatal("output WebM vide")
	}
	if !looksLikeEBML(out.Bytes()) {
		t.Errorf("output ne commence pas par la signature EBML (0x1A45DFA3) : % x", out.Bytes()[:min(8, out.Len())])
	}
}

func TestPlanRemuxWebM_IncompatibleVideoCodec(t *testing.T) {
	requireFFmpeg(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "in.mp4")

	// Encode un H.264 — incompatible WebM : le pré-flight doit le rejeter AVANT
	// tout octet, avec l'erreur typée qui pilote le 415 côté handler.
	args := []string{
		"-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc=size=320x240:rate=10:duration=1",
		"-c:v", "libx264", "-preset", "ultrafast", src,
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "ffmpeg", args...).CombinedOutput(); err != nil {
		t.Skipf("encodage H.264 indisponible: %v\n%s", err, out)
	}

	if _, err := PlanRemuxWebM(ctx, src); !errors.Is(err, ErrRemuxIncompatibleCodec) {
		t.Fatalf("PlanRemuxWebM(H.264) err = %v, want ErrRemuxIncompatibleCodec", err)
	}
}

func TestPlanRemuxWebM_FileMissing(t *testing.T) {
	requireFFmpeg(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	if _, err := PlanRemuxWebM(ctx, filepath.Join(t.TempDir(), "nope.mkv")); !errors.Is(err, ErrRemuxProbeFailed) {
		t.Fatalf("PlanRemuxWebM(missing) err = %v, want ErrRemuxProbeFailed", err)
	}
}

// TestProbeAVStreams_Sanity vérifie que probeAVStreams parse correctement la
// sortie JSON ffprobe sur une vidéo générée.
func TestProbeAVStreams_Sanity(t *testing.T) {
	requireFFmpeg(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "in.mkv")
	generateMKVAV1Opus(t, src, 1)

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	info, err := probeAVStreams(ctx, src)
	if err != nil {
		t.Fatalf("probeAVStreams: %v", err)
	}
	if info.VideoCodec != "av1" {
		t.Errorf("VideoCodec = %q, want av1", info.VideoCodec)
	}
	if len(info.AudioCodecs) == 0 || info.AudioCodecs[0] != "opus" {
		t.Errorf("AudioCodecs = %v, want [opus, ...]", info.AudioCodecs)
	}
}

// Helper assurant que les fixtures ne fuitent pas si un test plante.
func init() {
	// Nettoyage anti-poubelle pour les tests parallèles qui pourraient
	// laisser traîner des fichiers dans le TempDir.
	_ = os.RemoveAll(filepath.Join(os.TempDir(), "go-api-media-test"))
}
