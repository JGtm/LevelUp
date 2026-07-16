package media

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMasterAudioRenditions(t *testing.T) {
	master := strings.Join([]string{
		"#EXTM3U",
		"#EXT-X-VERSION:7",
		`#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="group_aud",NAME="game",DEFAULT=NO,URI="stream_game.m3u8"`,
		`#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="group_aud",NAME="voices",DEFAULT=NO,URI="stream_voices.m3u8"`,
		`#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="group_aud",NAME="full",DEFAULT=YES,URI="stream_full.m3u8"`,
		`#EXT-X-STREAM-INF:BANDWIDTH=211200,RESOLUTION=1920x1080,AUDIO="group_aud"`,
		"stream_0.m3u8",
	}, "\n")

	got := parseMasterAudioRenditions(master)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3 (la variante vidéo STREAM-INF ne doit pas compter)", len(got))
	}
	wantSlugs := []string{"game", "voices", "full"}
	wantURIs := []string{"stream_game.m3u8", "stream_voices.m3u8", "stream_full.m3u8"}
	for i, r := range got {
		if r.Slug != wantSlugs[i] || r.URI != wantURIs[i] {
			t.Errorf("rendition[%d] = (%q,%q), want (%q,%q)", i, r.Slug, r.URI, wantSlugs[i], wantURIs[i])
		}
	}
}

func TestParseMasterAudioRenditions_NoAudio(t *testing.T) {
	// Master vidéo seul (pas de EXT-X-MEDIA) → aucune rendition audio.
	master := "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=100000\nstream_0.m3u8\n"
	if got := parseMasterAudioRenditions(master); len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

// --- Intégration (nécessite ffmpeg + ffprobe) ---

func TestMigrateHLSAudioToAAC_Integration(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent du PATH — migration audio HLS ignorée")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe absent du PATH — migration audio HLS ignorée")
	}

	dir := t.TempDir()
	outDir := generateMixedCodecHLS(t, dir) // game=opus (legacy), full=aac

	// Pré-condition : le groupe mélange les codecs (game Opus, full AAC).
	assertSegmentCodec(t, filepath.Join(outDir, "init_game.mp4"), "audio", "opus")
	assertSegmentCodec(t, filepath.Join(outDir, "init_full.mp4"), "audio", "aac")

	res, err := MigrateHLSAudioToAAC(context.Background(), outDir, false)
	if err != nil {
		t.Fatalf("MigrateHLSAudioToAAC: %v", err)
	}
	if len(res.Converted) != 1 || res.Converted[0] != "game" {
		t.Errorf("Converted = %v, want [game]", res.Converted)
	}
	if res.AlreadyAAC || res.NotMultiTrack {
		t.Errorf("flags inattendus: AlreadyAAC=%v NotMultiTrack=%v", res.AlreadyAAC, res.NotMultiTrack)
	}

	// Post-condition : groupe mono-codec AAC ; vidéo intacte ; pas de dossier temp résiduel.
	assertSegmentCodec(t, filepath.Join(outDir, "init_game.mp4"), "audio", "aac")
	assertSegmentCodec(t, filepath.Join(outDir, "init_full.mp4"), "audio", "aac")
	assertSegmentCodec(t, filepath.Join(outDir, "init_0.mp4"), "video", "h264")
	if entries, _ := filepath.Glob(filepath.Join(outDir, ".aacmig-*")); len(entries) != 0 {
		t.Errorf("dossiers temp résiduels: %v", entries)
	}
	if segs, _ := filepath.Glob(filepath.Join(outDir, "seg_game_*.m4s")); len(segs) == 0 {
		t.Error("aucun segment game après migration")
	}
	// L'arbre migré reste démultiplexable et garde ses 2 renditions (game + full).
	if err := VerifyHLSPlayable(context.Background(), filepath.Join(outDir, "master.m3u8"), 2); err != nil {
		t.Errorf("VerifyHLSPlayable après migration = %v, want nil", err)
	}

	// Idempotence : un 2e passage ne convertit rien (déjà tout AAC).
	res2, err := MigrateHLSAudioToAAC(context.Background(), outDir, false)
	if err != nil {
		t.Fatalf("2e MigrateHLSAudioToAAC: %v", err)
	}
	if !res2.AlreadyAAC || len(res2.Converted) != 0 {
		t.Errorf("2e passage : AlreadyAAC=%v Converted=%v, want true/[]", res2.AlreadyAAC, res2.Converted)
	}
}

func TestMigrateHLSAudioToAAC_DryRun(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent du PATH — migration audio HLS ignorée")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe absent du PATH — migration audio HLS ignorée")
	}
	dir := t.TempDir()
	outDir := generateMixedCodecHLS(t, dir)

	res, err := MigrateHLSAudioToAAC(context.Background(), outDir, true)
	if err != nil {
		t.Fatalf("MigrateHLSAudioToAAC dry-run: %v", err)
	}
	if len(res.Converted) != 1 || res.Converted[0] != "game" {
		t.Errorf("dry-run Converted = %v, want [game]", res.Converted)
	}
	// Dry-run : aucune écriture → game reste en Opus.
	assertSegmentCodec(t, filepath.Join(outDir, "init_game.mp4"), "audio", "opus")
}

// generateMixedCodecHLS produit un arbre HLS legacy à codecs MIXTES (game = Opus
// copié, full = amix AAC) — reproduit l'ancien comportement avant le fix codec
// unique, pour tester la migration. Retourne le dossier de l'arbre.
func generateMixedCodecHLS(t *testing.T, dir string) string {
	t.Helper()
	src := generateTestMKV(t, dir) // h264 + 2 pistes Opus (Game/Mic)
	outDir := filepath.Join(dir, "mixed")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir outDir: %v", err)
	}
	outSlash := filepath.ToSlash(outDir)
	cmd := exec.Command("ffmpeg", ffmpegQuietArgs("-y",
		"-i", src,
		"-filter_complex", "[0:a:0][0:a:1]amix=inputs=2:normalize=0:duration=longest[full]",
		"-map", "0:v:0", "-map", "0:a:0", "-map", "[full]",
		"-c:v", "copy", "-c:a:0", "copy", "-c:a:1", "aac", "-b:a:1", "192k",
		"-var_stream_map", "v:0,agroup:aud a:0,agroup:aud,name:game a:1,agroup:aud,name:full,default:yes",
		"-master_pl_name", "master.m3u8",
		"-f", "hls", "-hls_segment_type", "fmp4", "-hls_playlist_type", "vod",
		"-hls_time", "1", "-hls_flags", "independent_segments",
		"-hls_fmp4_init_filename", "init_%v.mp4",
		"-hls_segment_filename", outSlash+"/seg_%v_%03d.m4s",
		outSlash+"/stream_%v.m3u8",
	)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("génération HLS mixte: %v\n%s", err, out)
	}
	return outDir
}
