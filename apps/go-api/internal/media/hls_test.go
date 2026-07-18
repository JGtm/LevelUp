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

func TestPlanHLS_TwoTracksGameVoicesFull(t *testing.T) {
	// 2 pistes source (jeu Opus + micro Opus) → 3 renditions game/voices/full.
	streams := []AVStreamDetail{
		{CodecType: "video", CodecName: "h264"},
		{CodecType: "audio", CodecName: "opus", Title: "Game", Language: "fra"},
		{CodecType: "audio", CodecName: "opus", Title: "Mic"},
	}
	plan, err := planHLS(streams, audioLayout{})
	if err != nil {
		t.Fatalf("planHLS: %v", err)
	}
	if plan.VideoAction != actionCopy {
		t.Errorf("VideoAction = %v, want copy", plan.VideoAction)
	}
	if len(plan.Audios) != 3 {
		t.Fatalf("len(Audios) = %d, want 3 (game/voices/full)", len(plan.Audios))
	}
	g, v, f := plan.Audios[0], plan.Audios[1], plan.Audios[2]
	if g.Slug != "game" || v.Slug != "voices" || f.Slug != "full" {
		t.Errorf("slugs = [%q,%q,%q], want [game,voices,full]", g.Slug, v.Slug, f.Slug)
	}
	// Codec unique AAC sur tout le groupe → la bascule de piste marche sur
	// Firefox/Safari. Sources Opus : game (piste 0) et voices (piste 1, map
	// direct) sont réencodées AAC ; full = amix réencodé AAC.
	if g.MapSpec != "0:a:0" || g.Action != actionReencode {
		t.Errorf("game = (%q,%v), want (0:a:0, reencode AAC)", g.MapSpec, g.Action)
	}
	if v.MapSpec != "0:a:1" || v.Action != actionReencode {
		t.Errorf("voices = (%q,%v), want (0:a:1, reencode AAC)", v.MapSpec, v.Action)
	}
	if f.MapSpec != "[full]" || f.Action != actionReencode {
		t.Errorf("full = (%q,%v), want ([full], reencode)", f.MapSpec, f.Action)
	}
	// Seul full est DEFAULT (les deux toggles actifs par défaut côté lecteur).
	if g.Default || v.Default || !f.Default {
		t.Errorf("Default = [%v,%v,%v], want [false,false,true]", g.Default, v.Default, f.Default)
	}
	// Display = slugs machine pour la détection du layout 2-toggles côté lecteur.
	if g.Display != "game" || v.Display != "voices" || f.Display != "full" {
		t.Errorf("Display = [%q,%q,%q], want [game,voices,full]", g.Display, v.Display, f.Display)
	}
	// 2 pistes → seul full passe par amix (voices est un map direct). L'amix de
	// SORTIE est suivi d'alimiter (anti-écrêtage de la somme jeu+voix).
	wantFC := "[0:a:0][0:a:1]amix=inputs=2:normalize=0:duration=longest,alimiter=limit=0.98:level=false[full]"
	if plan.FilterComplex != wantFC {
		t.Errorf("FilterComplex =\n  %q\nwant\n  %q", plan.FilterComplex, wantFC)
	}
}

func TestPlanHLS_ThreeTracksVoicesAmix(t *testing.T) {
	// 3 pistes source (jeu + micro + discord) → voices = amix de a:1+a:2.
	streams := []AVStreamDetail{
		{CodecType: "video", CodecName: "h264"},
		{CodecType: "audio", CodecName: "opus", Title: "Game"},
		{CodecType: "audio", CodecName: "opus", Title: "Mic"},
		{CodecType: "audio", CodecName: "opus", Title: "Discord"},
	}
	plan, err := planHLS(streams, audioLayout{})
	if err != nil {
		t.Fatalf("planHLS: %v", err)
	}
	if len(plan.Audios) != 3 {
		t.Fatalf("len(Audios) = %d, want 3", len(plan.Audios))
	}
	v := plan.Audios[1]
	if v.MapSpec != "[voices]" || v.Action != actionReencode {
		t.Errorf("voices = (%q,%v), want ([voices], reencode)", v.MapSpec, v.Action)
	}
	wantFC := "[0:a:1][0:a:2]amix=inputs=2:normalize=0:duration=longest,alimiter=limit=0.98:level=false[voices];" +
		"[0:a:0][0:a:1][0:a:2]amix=inputs=3:normalize=0:duration=longest,alimiter=limit=0.98:level=false[full]"
	if plan.FilterComplex != wantFC {
		t.Errorf("FilterComplex =\n  %q\nwant\n  %q", plan.FilterComplex, wantFC)
	}
}

func TestPlanHLS_SingleAudioTrackLegacy(t *testing.T) {
	// 1 seule piste audio (ex. MKV mono remuxé) → 1 rendition directe, pas de
	// filtre, pas de layout toggle. Display = titre de la piste (legacy).
	streams := []AVStreamDetail{
		{CodecType: "video", CodecName: "h264"},
		{CodecType: "audio", CodecName: "vorbis", Title: "Game"},
	}
	plan, err := planHLS(streams, audioLayout{})
	if err != nil {
		t.Fatalf("planHLS: %v", err)
	}
	if len(plan.Audios) != 1 {
		t.Fatalf("len(Audios) = %d, want 1", len(plan.Audios))
	}
	a := plan.Audios[0]
	if a.Slug != "a0" || a.MapSpec != "0:a:0" || !a.Default {
		t.Errorf("rendition = (%q,%q,default=%v), want (a0,0:a:0,true)", a.Slug, a.MapSpec, a.Default)
	}
	if a.Action != actionReencode {
		t.Errorf("Vorbis devrait être réencodé, got %v", a.Action)
	}
	if a.Display != "Game" {
		t.Errorf("Display = %q, want Game (legacy: titre de piste)", a.Display)
	}
	if plan.FilterComplex != "" {
		t.Errorf("FilterComplex = %q, want vide (mono-piste)", plan.FilterComplex)
	}
}

func TestPlanHLS_MultiTrackUniformAAC(t *testing.T) {
	// Garde-fou : sur un clip MULTIPISTE, toutes les renditions sortent en AAC
	// (codec unique du groupe audio). C'est l'invariant qui fait marcher la
	// bascule de piste sur Firefox/Safari (sinon changement de codec MSE → la
	// piste ne change pas et l'utilisateur entend toujours la rendition par
	// défaut). La copy n'est donc tolérée que si la source est DÉJÀ en AAC.
	cases := []struct {
		name    string
		streams []AVStreamDetail
		want    [3]streamAction // game, voices, full
	}{
		{
			name: "2 pistes Opus → tout réencodé AAC",
			streams: []AVStreamDetail{
				{CodecType: "video", CodecName: "h264"},
				{CodecType: "audio", CodecName: "opus"},
				{CodecType: "audio", CodecName: "opus"},
			},
			want: [3]streamAction{actionReencode, actionReencode, actionReencode},
		},
		{
			name: "2 pistes AAC → game/voices copy (déjà AAC), full amix réencodé",
			streams: []AVStreamDetail{
				{CodecType: "video", CodecName: "h264"},
				{CodecType: "audio", CodecName: "aac"},
				{CodecType: "audio", CodecName: "aac"},
			},
			want: [3]streamAction{actionCopy, actionCopy, actionReencode},
		},
		{
			name: "3 pistes (AAC + Opus + Opus) → game copy (AAC), voices/full amix réencodé",
			streams: []AVStreamDetail{
				{CodecType: "video", CodecName: "h264"},
				{CodecType: "audio", CodecName: "aac"},
				{CodecType: "audio", CodecName: "opus"},
				{CodecType: "audio", CodecName: "opus"},
			},
			want: [3]streamAction{actionCopy, actionReencode, actionReencode},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := planHLS(tc.streams, audioLayout{})
			if err != nil {
				t.Fatalf("planHLS: %v", err)
			}
			if len(plan.Audios) != 3 {
				t.Fatalf("len(Audios) = %d, want 3", len(plan.Audios))
			}
			got := [3]streamAction{plan.Audios[0].Action, plan.Audios[1].Action, plan.Audios[2].Action}
			if got != tc.want {
				t.Errorf("Actions = %v, want %v (codec unique AAC sur le groupe)", got, tc.want)
			}
		})
	}
}

func TestRenditionSlugs(t *testing.T) {
	got := renditionSlugs([]audioRendition{{Slug: "game"}, {Slug: "voices"}, {Slug: "full"}})
	if len(got) != 3 || got[0] != "game" || got[1] != "voices" || got[2] != "full" {
		t.Errorf("renditionSlugs = %v, want [game voices full]", got)
	}
}

// argsContainPair retourne true si args contient a immédiatement suivi de b.
func argsContainPair(args []string, a, b string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == a && args[i+1] == b {
			return true
		}
	}
	return false
}

func TestBuildHLSArgs_HEVCTag(t *testing.T) {
	// Vidéo HEVC copiée → -tag:v hvc1 (Safari refuse le hev1 par défaut). H.264
	// copié ou HEVC réencodé (sortie h264) → pas de tag.
	base := []audioRendition{{Slug: "a0", MapSpec: "0:a:0", Action: actionCopy, Default: true}}
	cases := []struct {
		name    string
		plan    hlsPlan
		wantTag bool
	}{
		{"hevc copié", hlsPlan{VideoAction: actionCopy, VideoSrcCodec: "hevc", Audios: base}, true},
		{"h265 (alias) copié", hlsPlan{VideoAction: actionCopy, VideoSrcCodec: "h265", Audios: base}, true},
		{"h264 copié", hlsPlan{VideoAction: actionCopy, VideoSrcCodec: "h264", Audios: base}, false},
		{"hevc réencodé", hlsPlan{VideoAction: actionReencode, VideoCodec: "h264", VideoSrcCodec: "hevc", Audios: base}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := buildHLSArgs(tc.plan, "src.mkv", "out", 4)
			if got := argsContainPair(args, "-tag:v", "hvc1"); got != tc.wantTag {
				t.Errorf("tag hvc1 présent = %v, want %v\nargs=%v", got, tc.wantTag, args)
			}
		})
	}
}

func TestAmixFilter_LimiterOnOutputRenditions(t *testing.T) {
	// Un amix de SORTIE (rendition servie) porte le limiteur anti-écrêtage.
	out := amixFilter([]int{0, 1}, "full")
	if !strings.Contains(out, "alimiter=limit=0.98:level=false") {
		t.Errorf("amixFilter (sortie) sans limiteur: %q", out)
	}
}

func TestPlanHLS_Errors(t *testing.T) {
	if _, err := planHLS([]AVStreamDetail{{CodecType: "audio", CodecName: "opus"}}, audioLayout{}); err == nil {
		t.Error("planHLS sans vidéo: erreur attendue")
	}
	if _, err := planHLS([]AVStreamDetail{{CodecType: "video", CodecName: "h264"}}, audioLayout{}); err == nil {
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
	plan, err := planHLS(streams, audioLayout{})
	if err != nil {
		t.Fatalf("planHLS: %v", err)
	}
	if !strings.HasPrefix(plan.VarStreamMap, "v:0,agroup:aud ") {
		t.Errorf("VarStreamMap = %q, want une seule variante vidéo", plan.VarStreamMap)
	}
}

func TestBuildVarStreamMap(t *testing.T) {
	// L'index a:N suit l'ordinal de sortie (position dans Audios), pas un champ
	// source : game/voices/full → a:0/a:1/a:2 avec default:yes sur full.
	plan := hlsPlan{
		Audios: []audioRendition{
			{Slug: "game", Language: "fra"},
			{Slug: "voices"},
			{Slug: "full", Default: true},
		},
	}
	got := buildVarStreamMap(plan)
	want := "v:0,agroup:aud a:0,agroup:aud,name:game,language:fra a:1,agroup:aud,name:voices a:2,agroup:aud,name:full,default:yes"
	if got != want {
		t.Errorf("buildVarStreamMap =\n  %q\nwant\n  %q", got, want)
	}
}

func TestBuildVarStreamMap_NoLanguage(t *testing.T) {
	plan := hlsPlan{Audios: []audioRendition{{Slug: "a0", Default: true}}}
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
	// 2 pistes source → 3 renditions game/voices/full (pré-mixage pour les 2
	// interrupteurs indépendants du lecteur).
	if res.AudioTracks != 3 {
		t.Errorf("AudioTracks = %d, want 3", res.AudioTracks)
	}
	if res.Segments == 0 {
		t.Error("Segments = 0, want > 0")
	}
	if len(res.Renditions) != 3 || res.Renditions[0] != "game" || res.Renditions[2] != "full" {
		t.Errorf("Renditions = %v, want [game voices full] (observabilité log)", res.Renditions)
	}

	master, err := os.ReadFile(res.MasterPath)
	if err != nil {
		t.Fatalf("lecture master: %v", err)
	}
	ms := string(master)
	if n := strings.Count(ms, "TYPE=AUDIO"); n != 3 {
		t.Errorf("master: %d TYPE=AUDIO, want 3\n%s", n, ms)
	}
	if !strings.Contains(ms, `NAME="game"`) || !strings.Contains(ms, `NAME="voices"`) || !strings.Contains(ms, `NAME="full"`) {
		t.Errorf("master: NAME game/voices/full manquants\n%s", ms)
	}
	if n := strings.Count(ms, "DEFAULT=YES"); n != 1 {
		t.Errorf("master: %d DEFAULT=YES, want 1 (full)", n)
	}

	// Preuves codec sur les init segments fMP4 (autonomes : leur moov porte la
	// config codec). ffprobe traite les chemins comme des URL (séparateur '/') :
	// sur un chemin Windows en backslash il ne résout pas les segments relatifs
	// d'une playlist ; un fichier autonome n'a pas ce souci.
	//
	// CODEC UNIQUE AAC sur tout le groupe audio (game/voices/full) : invariant
	// requis pour que la bascule de piste fonctionne sur Firefox/Safari (pas de
	// SourceBuffer.changeType). Sources Opus → game/voices réencodées AAC.
	//   - game : piste 0 (Opus source) → réencodé aac
	//   - voices : piste 1 (Opus source, map direct) → réencodé aac
	//   - full : amix → aac
	assertSegmentCodec(t, filepath.Join(outDir, "init_game.mp4"), "audio", "aac")
	assertSegmentCodec(t, filepath.Join(outDir, "init_voices.mp4"), "audio", "aac")
	assertSegmentCodec(t, filepath.Join(outDir, "init_full.mp4"), "audio", "aac")
	assertSegmentCodec(t, filepath.Join(outDir, "init_0.mp4"), "video", "h264")
}

// TestVerifyHLSPlayable_Integration valide la garde anti-perte de données : un
// arbre HLS réel avec le bon nombre de renditions est accepté (nil) ; un compte
// de renditions inattendu, un master absent ou un arbre amputé sont rejetés.
// Documente aussi EMPIRIQUEMENT comment ffprobe expose les renditions d'un master
// (justifie le comptage par parse du master plutôt que par ffprobe). Gate ffmpeg.
func TestVerifyHLSPlayable_Integration(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent du PATH — test VerifyHLSPlayable ignoré")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe absent du PATH — test VerifyHLSPlayable ignoré")
	}

	dir := t.TempDir()
	src := generateTestMKV(t, dir)
	outDir := filepath.Join(dir, "hls")
	res, err := BuildHLS(context.Background(), src, outDir, HLSOptions{SegmentDuration: 1})
	if err != nil {
		t.Fatalf("BuildHLS: %v", err)
	}
	if res.AudioTracks != 3 {
		t.Fatalf("pré-condition: AudioTracks = %d, want 3", res.AudioTracks)
	}

	// Preuve empirique : le master DÉCLARE 3 EXT-X-MEDIA ; ce que ffprobe ÉNUMÈRE
	// peut différer (dépend de la version/du démuxeur). On loggue les deux.
	raw, _ := os.ReadFile(res.MasterPath)
	declared := strings.Count(string(raw), "TYPE=AUDIO")
	probed := 0
	if streams, perr := ProbeStreamsDetailed(context.Background(), res.MasterPath); perr == nil {
		for _, s := range streams {
			if s.CodecType == "audio" {
				probed++
			}
		}
	}
	t.Logf("renditions master: déclarées(EXT-X-MEDIA)=%d, énumérées(ffprobe show_streams)=%d", declared, probed)

	// Arbre complet + compte attendu → démultiplexable.
	if err := VerifyHLSPlayable(context.Background(), res.MasterPath, 3); err != nil {
		t.Errorf("VerifyHLSPlayable(arbre valide, 3) = %v, want nil", err)
	}

	// Compte attendu ERRONÉ → rejet (une rendition manquerait avant suppression source).
	if err := VerifyHLSPlayable(context.Background(), res.MasterPath, 2); err == nil {
		t.Error("VerifyHLSPlayable(arbre valide, attendu 2): erreur attendue (3 renditions)")
	}

	// Master inexistant → erreur (ffprobe échoue à ouvrir).
	if err := VerifyHLSPlayable(context.Background(), filepath.Join(dir, "absent.m3u8"), 3); err == nil {
		t.Error("VerifyHLSPlayable(master absent): erreur attendue")
	}

	// Master présent mais segments + sous-playlists supprimés → ffprobe ne remonte
	// aucun flux (donc pas de vidéo) → rejet AVANT la suppression du source.
	entries, _ := os.ReadDir(outDir)
	for _, e := range entries {
		if e.Name() != "master.m3u8" {
			_ = os.Remove(filepath.Join(outDir, e.Name()))
		}
	}
	if err := VerifyHLSPlayable(context.Background(), res.MasterPath, 3); err == nil {
		t.Error("VerifyHLSPlayable(arbre amputé): erreur attendue")
	}
}

// generateTestMKV produit un MKV synthétique H.264 + 2 pistes Opus (Game/Mic).
func generateTestMKV(t *testing.T, dir string) string {
	t.Helper()
	src := filepath.Join(dir, "source.mkv")
	cmd := exec.Command("ffmpeg", ffmpegQuietArgs("-y",
		"-f", "lavfi", "-i", "testsrc=size=320x240:rate=15:duration=2",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=2",
		"-f", "lavfi", "-i", "sine=frequency=880:duration=2",
		"-map", "0:v", "-map", "1:a", "-map", "2:a",
		"-c:v", "libx264", "-preset", "ultrafast", "-c:a", "libopus",
		"-metadata:s:a:0", "title=Game", "-metadata:s:a:1", "title=Mic",
		src)...)
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
