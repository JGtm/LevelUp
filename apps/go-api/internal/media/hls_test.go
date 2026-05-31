package media

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNeedsHLS(t *testing.T) {
	mono := []AVStreamDetail{{CodecType: "video", CodecName: "h264"}, {CodecType: "audio", CodecName: "aac"}}
	multi := []AVStreamDetail{{CodecType: "video", CodecName: "h264"}, {CodecType: "audio"}, {CodecType: "audio"}}
	cases := []struct {
		name    string
		ext     string
		streams []AVStreamDetail
		want    bool
	}{
		{"mp4 mono-piste → direct", ".mp4", mono, false},
		{"webm mono-piste → direct", ".webm", mono, false},
		{"mp4 multipiste → HLS", ".mp4", multi, true},
		{"mkv mono-piste → HLS (container)", ".mkv", mono, true},
		{"avi mono-piste → HLS (container)", ".avi", mono, true},
		{"MKV majuscule → HLS", ".MKV", mono, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NeedsHLS(tc.ext, tc.streams); got != tc.want {
				t.Errorf("NeedsHLS(%q) = %v, want %v", tc.ext, got, tc.want)
			}
		})
	}
}

func TestPlanHLS_CopyOpusMultiTrack(t *testing.T) {
	streams := []AVStreamDetail{
		{CodecType: "video", CodecName: "h264"},
		{CodecType: "audio", CodecName: "opus", Title: "Game", Language: "fra"},
		{CodecType: "audio", CodecName: "opus", Title: "Mic"},
	}
	plan, err := planHLS(streams)
	if err != nil {
		t.Fatalf("planHLS: %v", err)
	}
	if plan.VideoAction != actionCopy {
		t.Errorf("VideoAction = %v, want copy", plan.VideoAction)
	}
	if len(plan.Audios) != 2 {
		t.Fatalf("len(Audios) = %d, want 2", len(plan.Audios))
	}
	for i, a := range plan.Audios {
		if a.Action != actionCopy {
			t.Errorf("Audios[%d].Action = %v, want copy (Opus)", i, a.Action)
		}
	}
	if !plan.Audios[0].Default || plan.Audios[1].Default {
		t.Errorf("Default = [%v,%v], want [true,false]", plan.Audios[0].Default, plan.Audios[1].Default)
	}
	if plan.Audios[0].Display != "Game" || plan.Audios[1].Display != "Mic" {
		t.Errorf("Display = [%q,%q], want [Game,Mic]", plan.Audios[0].Display, plan.Audios[1].Display)
	}
	if plan.Audios[0].Language != "fra" {
		t.Errorf("Audios[0].Language = %q, want fra", plan.Audios[0].Language)
	}
}

func TestPlanHLS_ReencodeWhenIncompatible(t *testing.T) {
	streams := []AVStreamDetail{
		{CodecType: "video", CodecName: "vp9"},
		{CodecType: "audio", CodecName: "aac"},
		{CodecType: "audio", CodecName: "vorbis"},
	}
	plan, err := planHLS(streams)
	if err != nil {
		t.Fatalf("planHLS: %v", err)
	}
	if plan.VideoAction != actionReencode || plan.VideoCodec != "h264" {
		t.Errorf("video = (%v,%q), want (reencode,h264)", plan.VideoAction, plan.VideoCodec)
	}
	if plan.Audios[0].Action != actionCopy {
		t.Errorf("AAC should copy, got %v", plan.Audios[0].Action)
	}
	if plan.Audios[1].Action != actionReencode {
		t.Errorf("Vorbis should reencode, got %v", plan.Audios[1].Action)
	}
}

func TestPlanHLS_Errors(t *testing.T) {
	if _, err := planHLS([]AVStreamDetail{{CodecType: "audio", CodecName: "opus"}}); err == nil {
		t.Error("planHLS sans vidéo: erreur attendue")
	}
	if _, err := planHLS([]AVStreamDetail{{CodecType: "video", CodecName: "h264"}}); err == nil {
		t.Error("planHLS sans audio: erreur attendue")
	}
}

func TestPlanHLS_SingleVideoTrack(t *testing.T) {
	// Deux pistes vidéo (cas pathologique) → seule la première est retenue.
	streams := []AVStreamDetail{
		{CodecType: "video", CodecName: "h264"},
		{CodecType: "video", CodecName: "h264"},
		{CodecType: "audio", CodecName: "opus"},
	}
	plan, err := planHLS(streams)
	if err != nil {
		t.Fatalf("planHLS: %v", err)
	}
	if !strings.HasPrefix(plan.VarStreamMap, "v:0,agroup:aud ") {
		t.Errorf("VarStreamMap = %q, want une seule variante vidéo", plan.VarStreamMap)
	}
}

func TestBuildVarStreamMap(t *testing.T) {
	plan := hlsPlan{
		Audios: []audioRendition{
			{SrcIndex: 0, Slug: "a0", Language: "fra", Default: true},
			{SrcIndex: 1, Slug: "a1", Language: "eng"},
		},
	}
	got := buildVarStreamMap(plan)
	want := "v:0,agroup:aud a:0,agroup:aud,name:a0,default:yes,language:fra a:1,agroup:aud,name:a1,language:eng"
	if got != want {
		t.Errorf("buildVarStreamMap =\n  %q\nwant\n  %q", got, want)
	}
}

func TestBuildVarStreamMap_NoLanguage(t *testing.T) {
	plan := hlsPlan{Audios: []audioRendition{{SrcIndex: 0, Slug: "a0", Default: true}}}
	got := buildVarStreamMap(plan)
	want := "v:0,agroup:aud a:0,agroup:aud,name:a0,default:yes"
	if got != want {
		t.Errorf("buildVarStreamMap = %q, want %q", got, want)
	}
}

func TestRewriteMasterAudioNames(t *testing.T) {
	master := strings.Join([]string{
		"#EXTM3U",
		`#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="group_aud",NAME="audio_1",DEFAULT=YES,URI="stream_a0.m3u8"`,
		`#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="group_aud",NAME="audio_2",DEFAULT=NO,URI="stream_a1.m3u8"`,
		`#EXT-X-STREAM-INF:BANDWIDTH=200000,AUDIO="group_aud"`,
		"stream_0.m3u8",
	}, "\n")

	got := rewriteMasterAudioNames(master, []string{"Game", "Mic"})

	if !strings.Contains(got, `NAME="Game"`) || !strings.Contains(got, `NAME="Mic"`) {
		t.Errorf("NAME non réécrits:\n%s", got)
	}
	if strings.Contains(got, `NAME="audio_1"`) || strings.Contains(got, `NAME="audio_2"`) {
		t.Errorf("anciens NAME subsistent:\n%s", got)
	}
	// Les lignes non-audio doivent rester intactes.
	if !strings.Contains(got, `#EXT-X-STREAM-INF:BANDWIDTH=200000,AUDIO="group_aud"`) {
		t.Errorf("ligne STREAM-INF altérée:\n%s", got)
	}
	if !strings.Contains(got, `DEFAULT=YES`) {
		t.Errorf("attribut DEFAULT perdu:\n%s", got)
	}
}

func TestRewriteMasterAudioNames_FewerDisplays(t *testing.T) {
	// Si moins de displays que de pistes, les pistes restantes gardent leur NAME.
	master := strings.Join([]string{
		`#EXT-X-MEDIA:TYPE=AUDIO,NAME="audio_1",URI="a.m3u8"`,
		`#EXT-X-MEDIA:TYPE=AUDIO,NAME="audio_2",URI="b.m3u8"`,
	}, "\n")
	got := rewriteMasterAudioNames(master, []string{"Game"})
	if !strings.Contains(got, `NAME="Game"`) {
		t.Errorf("première piste non réécrite:\n%s", got)
	}
	if !strings.Contains(got, `NAME="audio_2"`) {
		t.Errorf("seconde piste devrait garder son NAME:\n%s", got)
	}
}

func TestAudioDisplay(t *testing.T) {
	cases := []struct {
		name   string
		stream AVStreamDetail
		idx    int
		want   string
	}{
		{"title prioritaire", AVStreamDetail{Title: "Game", Language: "eng"}, 0, "Game"},
		{"langue si pas de title", AVStreamDetail{Language: "eng"}, 1, "ENG"},
		{"fallback index", AVStreamDetail{}, 2, "Audio 3"},
		{"title espacé trimmé", AVStreamDetail{Title: "  Mic  "}, 0, "Mic"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := audioDisplay(tc.stream, tc.idx); got != tc.want {
				t.Errorf("audioDisplay = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSanitizeLanguage(t *testing.T) {
	cases := map[string]string{
		"fra":         "fra",
		"EN":          "en",
		"fr-FR":       "", // tiret non autorisé
		"":            "",
		"123":         "",
		"toolonglang": "",
	}
	for in, want := range cases {
		if got := sanitizeLanguage(in); got != want {
			t.Errorf("sanitizeLanguage(%q) = %q, want %q", in, got, want)
		}
	}
}

// --- Golden d'intégration (nécessite ffmpeg + ffprobe) ---

func TestBuildHLS_Integration(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent du PATH — golden d'intégration HLS ignoré")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe absent du PATH — golden d'intégration HLS ignoré")
	}

	dir := t.TempDir()
	src := generateTestMKV(t, dir)
	outDir := filepath.Join(dir, "hls")

	res, err := BuildHLS(context.Background(), src, outDir, HLSOptions{SegmentDuration: 1})
	if err != nil {
		t.Fatalf("BuildHLS: %v", err)
	}
	if res.AudioTracks != 2 {
		t.Errorf("AudioTracks = %d, want 2", res.AudioTracks)
	}
	if res.Segments == 0 {
		t.Error("Segments = 0, want > 0")
	}

	master, err := os.ReadFile(res.MasterPath)
	if err != nil {
		t.Fatalf("lecture master: %v", err)
	}
	ms := string(master)
	if n := strings.Count(ms, "TYPE=AUDIO"); n != 2 {
		t.Errorf("master: %d TYPE=AUDIO, want 2\n%s", n, ms)
	}
	if !strings.Contains(ms, `NAME="Game"`) || !strings.Contains(ms, `NAME="Mic"`) {
		t.Errorf("master: NAME réécrits manquants\n%s", ms)
	}
	if n := strings.Count(ms, "DEFAULT=YES"); n != 1 {
		t.Errorf("master: %d DEFAULT=YES, want 1", n)
	}

	// Preuve du copy : audio reste opus, vidéo reste h264. On probe les init
	// segments fMP4 (autonomes : leur moov porte la config codec) plutôt que les
	// sous-playlists. ffprobe traite les chemins comme des URL (séparateur '/') :
	// sur un chemin Windows en backslash, il ne résout pas les segments relatifs
	// d'une playlist. Un fichier autonome n'a pas ce souci.
	assertSegmentCodec(t, filepath.Join(outDir, "init_a0.mp4"), "audio", "opus")
	assertSegmentCodec(t, filepath.Join(outDir, "init_0.mp4"), "video", "h264")
}

// generateTestMKV produit un MKV synthétique H.264 + 2 pistes Opus (Game/Mic).
func generateTestMKV(t *testing.T, dir string) string {
	t.Helper()
	src := filepath.Join(dir, "source.mkv")
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=size=320x240:rate=15:duration=2",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=2",
		"-f", "lavfi", "-i", "sine=frequency=880:duration=2",
		"-map", "0:v", "-map", "1:a", "-map", "2:a",
		"-c:v", "libx264", "-preset", "ultrafast", "-c:a", "libopus",
		"-metadata:s:a:0", "title=Game", "-metadata:s:a:1", "title=Mic",
		src)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("génération MKV test: %v\n%s", err, out)
	}
	return src
}

// assertSegmentCodec vérifie via ProbeStreamsDetailed qu'un segment fMP4 (init)
// contient bien le codec attendu pour le type donné (preuve copy/réencode).
func assertSegmentCodec(t *testing.T, segmentPath, codecType, wantCodec string) {
	t.Helper()
	streams, err := ProbeStreamsDetailed(context.Background(), segmentPath)
	if err != nil {
		t.Fatalf("ProbeStreamsDetailed(%s): %v", segmentPath, err)
	}
	for _, s := range streams {
		if s.CodecType == codecType {
			if s.CodecName != wantCodec {
				t.Errorf("%s: codec %s = %q, want %q", filepath.Base(segmentPath), codecType, s.CodecName, wantCodec)
			}
			return
		}
	}
	t.Errorf("%s: aucune piste %s trouvée", filepath.Base(segmentPath), codecType)
}
