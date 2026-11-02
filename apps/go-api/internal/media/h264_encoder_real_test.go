package media

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
)

// TestBuildHLS_RealEncoderOnTheResolvedFFmpeg transcode un petit clip (vidéo MPEG-4, que le plan
// réencode en H.264) avec le ffmpeg résolu et l'encodeur que SelectedH264Encoder retient pour
// lui, puis vérifie l'arbre HLS avec ffprobe. Sur un poste avec le build LGPL livré
// (LEVELUP_FFMPEG_DIR), cet encodeur est h264_mf ; avec un build GPL, libx264. Ignoré sans
// ffmpeg ni ffprobe.
func TestBuildHLS_RealEncoderOnTheResolvedFFmpeg(t *testing.T) {
	if _, err := ResolveFFmpeg(); err != nil {
		t.Skip("ffmpeg introuvable")
	}
	if _, err := ResolveFFprobe(); err != nil {
		t.Skip("ffprobe introuvable")
	}
	resetCapabilitiesCache()
	t.Cleanup(resetCapabilitiesCache)
	ctx := context.Background()
	enc, err := SelectedH264Encoder(ctx)
	if err != nil {
		t.Skipf("aucun encodeur H.264 sur ce ffmpeg : %v", err)
	}
	t.Logf("encodeur H.264 retenu : %s", enc)

	dir := t.TempDir()
	src := filepath.Join(dir, "src.mkv")
	var stderr bytes.Buffer
	gen := ffmpegQuietArgs("-f", "lavfi", "-i", "testsrc=size=640x360:rate=30:duration=6",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=6",
		"-c:v", "mpeg4", "-c:a", "aac", "-shortest", src)
	if err := RunFFmpeg(ctx, FFmpegJob{Args: gen, Stderr: &stderr}); err != nil {
		t.Skipf("source de test non générable (%v) : %s", err, stderr.String())
	}

	res, err := BuildHLS(ctx, src, filepath.Join(dir, "hls"), HLSOptions{SegmentDuration: 2})
	if err != nil {
		t.Fatalf("BuildHLS avec %s : %v", enc, err)
	}
	if res.Segments < 2 {
		t.Errorf("segments = %d, attendu au moins 2 pour 6 s à 2 s par segment", res.Segments)
	}
	if err := VerifyHLSPlayable(ctx, res.MasterPath, res.AudioTracks); err != nil {
		t.Errorf("arbre HLS illisible : %v", err)
	}
	streams, err := ProbeStreamsDetailed(ctx, res.MasterPath)
	if err != nil {
		t.Fatalf("ffprobe du master : %v", err)
	}
	for _, s := range streams {
		if s.CodecType == "video" && s.CodecName != "h264" {
			t.Errorf("codec vidéo = %q, attendu h264", s.CodecName)
		}
	}
}
