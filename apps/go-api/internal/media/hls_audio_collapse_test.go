package media

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollapseMasterToSingleAudio(t *testing.T) {
	master := strings.Join([]string{
		"#EXTM3U",
		"#EXT-X-VERSION:7",
		`#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="aud",NAME="game",DEFAULT=NO,URI="stream_game.m3u8"`,
		`#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="aud",NAME="voices",DEFAULT=NO,URI="stream_voices.m3u8"`,
		`#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="aud",NAME="full",DEFAULT=YES,URI="stream_full.m3u8"`,
		`#EXT-X-STREAM-INF:BANDWIDTH=211200,RESOLUTION=1920x1080,AUDIO="aud"`,
		"stream_0.m3u8",
	}, "\n")

	got, changed := collapseMasterToSingleAudio(master, "game")
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if n := strings.Count(got, "TYPE=AUDIO"); n != 1 {
		t.Errorf("%d TYPE=AUDIO, want 1 (game seul)\n%s", n, got)
	}
	if !strings.Contains(got, `NAME="game"`) {
		t.Errorf("rendition game absente\n%s", got)
	}
	if strings.Contains(got, `NAME="voices"`) || strings.Contains(got, `NAME="full"`) {
		t.Errorf("renditions voices/full non supprimées\n%s", got)
	}
	// game devient DEFAULT=YES.
	if !strings.Contains(got, `NAME="game",DEFAULT=YES`) {
		t.Errorf("game pas mis DEFAULT=YES\n%s", got)
	}
	// La variante vidéo et la sous-playlist restent intactes.
	if !strings.Contains(got, `#EXT-X-STREAM-INF:BANDWIDTH=211200,RESOLUTION=1920x1080,AUDIO="aud"`) {
		t.Errorf("ligne STREAM-INF altérée\n%s", got)
	}
	if !strings.Contains(got, "stream_0.m3u8") {
		t.Errorf("variante vidéo perdue\n%s", got)
	}
}

func TestCollapseMasterToSingleAudio_NoChange(t *testing.T) {
	// Master déjà mono-audio (game seul) → aucun changement.
	master := strings.Join([]string{
		"#EXTM3U",
		`#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="aud",NAME="game",DEFAULT=YES,URI="stream_game.m3u8"`,
		`#EXT-X-STREAM-INF:BANDWIDTH=100000,AUDIO="aud"`,
		"stream_0.m3u8",
	}, "\n")
	if _, changed := collapseMasterToSingleAudio(master, "game"); changed {
		t.Error("changed = true, want false (déjà mono-audio)")
	}
}

func TestSetDefaultYes(t *testing.T) {
	noLine := `#EXT-X-MEDIA:TYPE=AUDIO,NAME="game",DEFAULT=NO,URI="stream_game.m3u8"`
	if got := setDefaultYes(noLine); !strings.Contains(got, "DEFAULT=YES") || strings.Contains(got, "DEFAULT=NO") {
		t.Errorf("setDefaultYes(DEFAULT=NO) = %q", got)
	}
	absent := `#EXT-X-MEDIA:TYPE=AUDIO,NAME="game",URI="stream_game.m3u8"`
	if got := setDefaultYes(absent); !strings.Contains(got, "TYPE=AUDIO,DEFAULT=YES") {
		t.Errorf("setDefaultYes(sans DEFAULT) = %q", got)
	}
}

func TestForceCollapseHLSAudioTree(t *testing.T) {
	master := strings.Join([]string{
		"#EXTM3U",
		`#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="aud",NAME="game",DEFAULT=NO,URI="stream_game.m3u8"`,
		`#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="aud",NAME="voices",DEFAULT=NO,URI="stream_voices.m3u8"`,
		`#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="aud",NAME="full",DEFAULT=YES,URI="stream_full.m3u8"`,
		`#EXT-X-STREAM-INF:BANDWIDTH=211200,AUDIO="aud"`,
		"stream_0.m3u8",
	}, "\n")

	// Dry-run : rapporte le collapse sans écrire (master intact).
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "master.m3u8"), []byte(master), 0o644); err != nil {
		t.Fatalf("écriture master: %v", err)
	}
	res, err := ForceCollapseHLSAudioTree(dir, true)
	if err != nil || !res.Collapsed {
		t.Fatalf("dry-run : res=%+v err=%v, want Collapsed", res, err)
	}
	if n := strings.Count(readFile(t, filepath.Join(dir, "master.m3u8")), "TYPE=AUDIO"); n != 3 {
		t.Errorf("dry-run a modifié le master : %d TYPE=AUDIO, want 3", n)
	}

	// Écriture réelle : ne reste que la rendition game (DEFAULT=YES), SANS corrélation.
	res, err = ForceCollapseHLSAudioTree(dir, false)
	if err != nil || !res.Collapsed {
		t.Fatalf("collapse forcé : res=%+v err=%v, want Collapsed", res, err)
	}
	got := readFile(t, filepath.Join(dir, "master.m3u8"))
	if n := strings.Count(got, "TYPE=AUDIO"); n != 1 {
		t.Errorf("après collapse forcé : %d TYPE=AUDIO, want 1\n%s", n, got)
	}
	if !strings.Contains(got, `NAME="game",DEFAULT=YES`) {
		t.Errorf("game/DEFAULT=YES manquant\n%s", got)
	}

	// Sans rendition game : skip motivé, pas d'erreur.
	dir2 := t.TempDir()
	noGame := strings.ReplaceAll(master, "game", "a0") // slug dérivé de l'URI stream_<slug>.m3u8
	if err := os.WriteFile(filepath.Join(dir2, "master.m3u8"), []byte(noGame), 0o644); err != nil {
		t.Fatalf("écriture master: %v", err)
	}
	res, err = ForceCollapseHLSAudioTree(dir2, false)
	if err != nil || res.Collapsed || res.Skipped == "" {
		t.Errorf("sans game : res=%+v err=%v, want skip motivé", res, err)
	}
}

// --- Intégration (nécessite ffmpeg + ffprobe) ---

func TestCollapseRedundantHLSAudio_Integration(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent du PATH")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe absent du PATH")
	}
	dir := t.TempDir()

	// REDONDANT : 2 pistes source IDENTIQUES (même AM) → game≈voices → collapse.
	redundant := buildLegacyHLSTree(t, dir, "redundant", twinTrackMKV(t, dir, "twin.mkv", true))
	if n := strings.Count(readFile(t, filepath.Join(redundant, "master.m3u8")), "TYPE=AUDIO"); n != 3 {
		t.Fatalf("pré-condition : %d TYPE=AUDIO, want 3 (game/voices/full)", n)
	}
	res, err := CollapseRedundantHLSAudio(context.Background(), redundant, false)
	if err != nil {
		t.Fatalf("CollapseRedundantHLSAudio: %v", err)
	}
	if !res.Collapsed {
		t.Errorf("clip redondant non collapsé (corr=%.3f, skip=%q)", res.Corr, res.Skipped)
	}
	master := readFile(t, filepath.Join(redundant, "master.m3u8"))
	if n := strings.Count(master, "TYPE=AUDIO"); n != 1 {
		t.Errorf("après collapse : %d TYPE=AUDIO, want 1\n%s", n, master)
	}
	if !strings.Contains(master, `NAME="game"`) || !strings.Contains(master, "DEFAULT=YES") {
		t.Errorf("game/DEFAULT=YES manquant après collapse\n%s", master)
	}
	// Après collapse, le master n'expose plus qu'une rendition (game).
	if err := VerifyHLSPlayable(context.Background(), filepath.Join(redundant, "master.m3u8"), 1); err != nil {
		t.Errorf("VerifyHLSPlayable après collapse = %v, want nil", err)
	}

	// DISTINCT : 2 pistes source différentes (AM distincts) → game≠voices → NON collapsé.
	distinct := buildLegacyHLSTree(t, dir, "distinct", twinTrackMKV(t, dir, "distinct.mkv", false))
	res2, err := CollapseRedundantHLSAudio(context.Background(), distinct, false)
	if err != nil {
		t.Fatalf("CollapseRedundantHLSAudio (distinct): %v", err)
	}
	if res2.Collapsed {
		t.Errorf("clip jeu+voix distinct collapsé à tort (corr=%.3f)", res2.Corr)
	}
}

// twinTrackMKV produit un MKV vidéo + 2 pistes audio AM. identical=true → pistes
// identiques (clip redondant) ; false → AM distincts (clip jeu+voix).
func twinTrackMKV(t *testing.T, dir, name string, identical bool) string {
	t.Helper()
	out := filepath.Join(dir, name)
	args := ffmpegQuietArgs("-y",
		"-f", "lavfi", "-i", "aevalsrc="+amExpr("440", "0.7")+":d=3:s=8000")
	if identical {
		args = append(args, "-f", "lavfi", "-i", "testsrc=size=320x240:rate=15:duration=3",
			"-map", "1:v", "-map", "0:a", "-map", "0:a")
	} else {
		args = append(args, "-f", "lavfi", "-i", "aevalsrc="+amExpr("880", "1.3")+":d=3:s=8000",
			"-f", "lavfi", "-i", "testsrc=size=320x240:rate=15:duration=3",
			"-map", "2:v", "-map", "0:a", "-map", "1:a")
	}
	args = append(args, "-c:v", "libx264", "-preset", "ultrafast", "-c:a", "libopus", out)
	if o, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Fatalf("génération %s: %v\n%s", name, err, o)
	}
	return out
}

// buildLegacyHLSTree transcode src en arbre HLS multipiste avec le mapping
// HISTORIQUE game/voices/full (track0IsFullMix=false), reproduisant les clips
// legacy à corriger. Réutilise les internes de production (planHLS/buildHLSArgs).
func buildLegacyHLSTree(t *testing.T, dir, name, src string) string {
	t.Helper()
	streams, err := ProbeStreamsDetailed(context.Background(), src)
	if err != nil {
		t.Fatalf("ProbeStreamsDetailed: %v", err)
	}
	plan, err := planHLS(streams, audioLayout{})
	if err != nil {
		t.Fatalf("planHLS: %v", err)
	}
	outDir := filepath.Join(dir, name)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	args := buildHLSArgs(plan, src, outDir, 1)
	if o, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg legacy HLS: %v\n%s", err, o)
	}
	if err := rewriteMasterFile(filepath.Join(outDir, "master.m3u8"), plan.Audios); err != nil {
		t.Fatalf("rewriteMasterFile: %v", err)
	}
	return outDir
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lecture %s: %v", path, err)
	}
	return string(b)
}
