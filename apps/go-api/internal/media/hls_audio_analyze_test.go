package media

import (
	"context"
	"encoding/binary"
	"math"
	"os/exec"
	"path/filepath"
	"testing"
)

// appendF32 encode un float32 en little-endian (PCM f32le) pour les tests purs.
func appendF32(b []byte, v float32) []byte {
	var tmp [4]byte
	binary.LittleEndian.PutUint32(tmp[:], math.Float32bits(v))
	return append(b, tmp[:]...)
}

func TestPearson(t *testing.T) {
	cases := []struct {
		name   string
		xs, ys []float64
		want   float64
		tol    float64
	}{
		{"identiques", []float64{1, 2, 3, 4, 5, 6}, []float64{1, 2, 3, 4, 5, 6}, 1.0, 1e-9},
		{"anti-corrélées", []float64{1, 2, 3, 4, 5, 6}, []float64{6, 5, 4, 3, 2, 1}, -1.0, 1e-9},
		{"décalage constant", []float64{1, 2, 3, 4, 5, 6}, []float64{11, 12, 13, 14, 15, 16}, 1.0, 1e-9},
		{"x constant → 0", []float64{2, 2, 2, 2, 2, 2}, []float64{1, 2, 3, 4, 5, 6}, 0.0, 1e-9},
		{"trop court → 0", []float64{1, 2, 3}, []float64{1, 2, 3}, 0.0, 1e-9},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pearson(tc.xs, tc.ys); math.Abs(got-tc.want) > tc.tol {
				t.Errorf("pearson = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPearson_TruncatesToCommonLength(t *testing.T) {
	// Les enveloppes de deux pistes peuvent différer d'une trame ; pearson tronque.
	xs := []float64{1, 2, 3, 4, 5, 6, 7}
	ys := []float64{1, 2, 3, 4, 5, 6}
	if got := pearson(xs, ys); math.Abs(got-1.0) > 1e-9 {
		t.Errorf("pearson (longueurs ≠) = %v, want 1.0", got)
	}
}

func TestRMSFramesDB(t *testing.T) {
	// 800 échantillons à 0.5 (RMS=0.5 → −6,02 dB) puis 800 silencieux (→ plancher).
	var raw []byte
	for i := 0; i < 800; i++ {
		raw = appendF32(raw, 0.5)
	}
	for i := 0; i < 800; i++ {
		raw = appendF32(raw, 0.0)
	}
	got := rmsFramesDB(raw, 800)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 trames", len(got))
	}
	if math.Abs(got[0]-(-6.0206)) > 0.01 {
		t.Errorf("trame 0 = %v dB, want ~-6.02 (RMS 0.5)", got[0])
	}
	if got[1] != rmsFloorDB {
		t.Errorf("trame 1 (silence) = %v, want plancher %v", got[1], rmsFloorDB)
	}
}

func TestRMSFramesDB_PartialFrameIgnored(t *testing.T) {
	// 1000 échantillons → 1 trame pleine de 800, les 200 restants ignorés.
	var raw []byte
	for i := 0; i < 1000; i++ {
		raw = appendF32(raw, 0.5)
	}
	if got := rmsFramesDB(raw, 800); len(got) != 1 {
		t.Errorf("len = %d, want 1 (trame partielle ignorée)", len(got))
	}
}

func TestSilenceRatio(t *testing.T) {
	// Continu : toutes les trames au même niveau → 0 % de silence.
	cont := make([]float64, 100)
	for i := range cont {
		cont[i] = -20
	}
	if r := silenceRatio(cont); r != 0 {
		t.Errorf("continu : silenceRatio = %v, want 0", r)
	}
	// Intermittent : moitié forte (−10 dB), moitié silence (−91) → ~50 %.
	interm := make([]float64, 100)
	for i := range interm {
		if i%2 == 0 {
			interm[i] = -10
		} else {
			interm[i] = -91
		}
	}
	if r := silenceRatio(interm); r < 0.4 || r > 0.6 {
		t.Errorf("intermittent : silenceRatio = %v, want ~0.5", r)
	}
}

func TestRestMixFilter(t *testing.T) {
	if fc, m := restMixFilter(2); fc != "" || m != "0:a:1" {
		t.Errorf(`restMixFilter(2) = (%q,%q), want ("", "0:a:1")`, fc, m)
	}
	fc, m := restMixFilter(4)
	if fc != "[0:a:1][0:a:2][0:a:3]amix=inputs=3:normalize=0[mix]" || m != "[mix]" {
		t.Errorf("restMixFilter(4) = (%q,%q)", fc, m)
	}
}

func TestBuildEnvelopeArgs(t *testing.T) {
	// La borne mémoire `-t 600` doit précéder `-i` (option d'ENTRÉE → arrête le
	// décodage), et le map direct ne pas injecter de -filter_complex.
	args := buildEnvelopeArgs("clip.mkv", "", "0:a:0")
	ti, ii := indexOf(args, "-t"), indexOf(args, "-i")
	if ti < 0 || ii < 0 || ti+1 >= len(args) {
		t.Fatalf("args sans -t/-i: %v", args)
	}
	if args[ti+1] != "600" {
		t.Errorf("-t = %q, want 600", args[ti+1])
	}
	if ti > ii {
		t.Errorf("-t (%d) doit précéder -i (%d) pour borner le décodage: %v", ti, ii, args)
	}
	if indexOf(args, "-filter_complex") != -1 {
		t.Errorf("map direct: pas de -filter_complex attendu: %v", args)
	}
	// Avec filterComplex, le filtre est injecté.
	if a := buildEnvelopeArgs("clip.mkv", "[0:a:1][0:a:2]amix=inputs=2:normalize=0[mix]", "[mix]"); indexOf(a, "-filter_complex") == -1 {
		t.Errorf("filterComplex fourni mais -filter_complex absent: %v", a)
	}
}

// indexOf retourne l'index de la première occurrence de s dans args, ou -1.
func indexOf(args []string, s string) int {
	for i, a := range args {
		if a == s {
			return i
		}
	}
	return -1
}

func TestPlanAudioRenditions_FullMixFourTracks(t *testing.T) {
	// OBS : piste 0 = capture de sortie (mix complet), 1 = jeu, 2 = micro, 3 = Discord.
	src := []AVStreamDetail{
		{CodecType: "audio", CodecName: "aac"}, // 0 = mix complet
		{CodecType: "audio", CodecName: "aac"}, // 1 = jeu
		{CodecType: "audio", CodecName: "aac"}, // 2 = micro
		{CodecType: "audio", CodecName: "aac"}, // 3 = Discord
	}
	audios, fc := planAudioRenditions(src, audioLayout{Track0FullMix: true, GameComponent: 1})
	if len(audios) != 3 {
		t.Fatalf("len = %d, want 3", len(audios))
	}
	g, v, f := audios[0], audios[1], audios[2]
	// full lit la piste 0 DIRECTEMENT (pas d'amix → pas d'écho).
	if f.Slug != "full" || f.MapSpec != "0:a:0" || !f.Default {
		t.Errorf("full = (%q,%q,default=%v), want (full,0:a:0,true)", f.Slug, f.MapSpec, f.Default)
	}
	if g.Slug != "game" || g.MapSpec != "0:a:1" {
		t.Errorf("game = (%q,%q), want (game,0:a:1)", g.Slug, g.MapSpec)
	}
	if v.Slug != "voices" || v.MapSpec != "[voices]" {
		t.Errorf("voices = (%q,%q), want (voices,[voices])", v.Slug, v.MapSpec)
	}
	// voices = amix des pistes 2..3 (micro + Discord), suivi du limiteur anti-écrêtage.
	wantFC := "[0:a:2][0:a:3]amix=inputs=2:normalize=0:duration=longest,alimiter=limit=0.98:level=false[voices]"
	if fc != wantFC {
		t.Errorf("FilterComplex = %q, want %q", fc, wantFC)
	}
	// AAC partout → copy (groupe mono-codec, pas de SourceBuffer.changeType).
	if g.Action != actionCopy || f.Action != actionCopy {
		t.Errorf("game/full Action = %v/%v, want copy/copy (AAC)", g.Action, f.Action)
	}
}

func TestPlanAudioRenditions_FullMixThreeTracks(t *testing.T) {
	// 3 pistes : 0 = mix complet, 1 = jeu, 2 = voix → voices = map direct 0:a:2.
	src := []AVStreamDetail{
		{CodecType: "audio", CodecName: "aac"},
		{CodecType: "audio", CodecName: "aac"},
		{CodecType: "audio", CodecName: "aac"},
	}
	audios, fc := planAudioRenditions(src, audioLayout{Track0FullMix: true, GameComponent: 1})
	if len(audios) != 3 {
		t.Fatalf("len = %d, want 3", len(audios))
	}
	if audios[1].MapSpec != "0:a:2" {
		t.Errorf("voices MapSpec = %q, want 0:a:2 (map direct)", audios[1].MapSpec)
	}
	if fc != "" {
		t.Errorf("FilterComplex = %q, want vide (pas d'amix)", fc)
	}
	if audios[2].MapSpec != "0:a:0" || !audios[2].Default {
		t.Errorf("full = (%q,default=%v), want (0:a:0,true)", audios[2].MapSpec, audios[2].Default)
	}
}

func TestPlanAudioRenditions_FullMixTwoTracksCollapses(t *testing.T) {
	// Piste 0 = mix complet + 1 seule composante → rien à séparer → rendition unique.
	src := []AVStreamDetail{
		{CodecType: "audio", CodecName: "aac"},
		{CodecType: "audio", CodecName: "aac"},
	}
	audios, fc := planAudioRenditions(src, audioLayout{Track0FullMix: true, GameComponent: 1})
	if len(audios) != 1 {
		t.Fatalf("len = %d, want 1 (rendition unique)", len(audios))
	}
	if audios[0].Slug != "a0" || audios[0].MapSpec != "0:a:0" || !audios[0].Default {
		t.Errorf("rendition = (%q,%q,default=%v), want (a0,0:a:0,true)", audios[0].Slug, audios[0].MapSpec, audios[0].Default)
	}
	if fc != "" {
		t.Errorf("FilterComplex = %q, want vide", fc)
	}
}

func TestPlanAudioRenditions_NotFullMixUnchanged(t *testing.T) {
	// Track0FullMix=false → mapping historique (game = piste 0, full = amix).
	src := []AVStreamDetail{
		{CodecType: "audio", CodecName: "aac"},
		{CodecType: "audio", CodecName: "aac"},
	}
	audios, _ := planAudioRenditions(src, audioLayout{})
	if len(audios) != 3 || audios[0].MapSpec != "0:a:0" || audios[2].MapSpec != "[full]" {
		t.Errorf("mapping historique attendu : game=0:a:0, full=[full] ; got %+v", audios)
	}
}

func TestPlanAudioRenditions_FullMixGameNotFirst(t *testing.T) {
	// Le jeu N'EST PAS la 1ère composante (GameComponent=2) → game = 0:a:2,
	// voices = amix des autres composantes (0:a:1 + 0:a:3). Prouve l'indépendance
	// à l'ordre des pistes (classement acoustique, pas positionnel).
	src := []AVStreamDetail{
		{CodecType: "audio", CodecName: "aac"}, // 0 = mix complet
		{CodecType: "audio", CodecName: "aac"}, // 1 = voix
		{CodecType: "audio", CodecName: "aac"}, // 2 = jeu
		{CodecType: "audio", CodecName: "aac"}, // 3 = voix
	}
	audios, fc := planAudioRenditions(src, audioLayout{Track0FullMix: true, GameComponent: 2})
	if len(audios) != 3 {
		t.Fatalf("len = %d, want 3", len(audios))
	}
	if audios[0].Slug != "game" || audios[0].MapSpec != "0:a:2" {
		t.Errorf("game = (%q,%q), want (game,0:a:2)", audios[0].Slug, audios[0].MapSpec)
	}
	wantFC := "[0:a:1][0:a:3]amix=inputs=2:normalize=0:duration=longest,alimiter=limit=0.98:level=false[voices]"
	if fc != wantFC {
		t.Errorf("FilterComplex = %q, want %q (voix = composantes hors jeu)", fc, wantFC)
	}
}

// --- Intégration : analyse du layout audio (nécessite ffmpeg + ffprobe) ---

func TestAnalyzeAudioLayout_Integration(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent du PATH")
	}

	dir := t.TempDir()

	// POSITIF : piste 0 = amix(piste1, piste2) → la piste 0 EST le mix complet →
	// détecté. Les sources ont une forte dynamique d'enveloppe (AM lente) pour
	// passer la garde de variance, comme un vrai enregistrement de jeu.
	mix := generateAudioMKV(t, dir, "mix.mkv",
		"[0:a]asplit=2[t1a][t1b];[1:a]asplit=2[t2a][t2b];"+
			"[t1a][t2a]amix=inputs=2:normalize=0[mix]",
		[]string{"[mix]", "[t1b]", "[t2b]"})
	layout, corr, err := analyzeAudioLayout(context.Background(), mix, 3)
	if err != nil {
		t.Fatalf("analyzeAudioLayout (positif): %v", err)
	}
	if !layout.Track0FullMix {
		t.Errorf("piste 0 = amix(reste) : Track0FullMix = false (corr=%.3f), want true", corr)
	}

	// NÉGATIF : piste 0 = signal indépendant (AM de rythme distinct), pistes 1/2
	// autres → piste 0 ≠ mix des autres → non détecté.
	indep := generateAudioMKV(t, dir, "indep.mkv",
		"[0:a]anull[a0];[1:a]anull[a1];[2:a]anull[a2]",
		[]string{"[a0]", "[a1]", "[a2]"})
	layout, corr, err = analyzeAudioLayout(context.Background(), indep, 3)
	if err != nil {
		t.Fatalf("analyzeAudioLayout (négatif): %v", err)
	}
	if layout.Track0FullMix {
		t.Errorf("piste 0 indépendante : Track0FullMix = true (corr=%.3f), want false", corr)
	}
}

func TestAnalyzeAudioLayout_GameClassifiedNotByOrder(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent du PATH")
	}
	dir := t.TempDir()

	// Source : jeu CONTINU (fort) en 2ᵉ composante, voix INTERMITTENTE (faible) en
	// 1ʳᵉ. piste 0 = amix(jeu, voix). Le jeu domine le mix → enveloppe la plus
	// corrélée au mix → classé `game` MALGRÉ sa position (2). Prouve que le split
	// est acoustique, pas positionnel.
	src := generateGameVoiceMKV(t, dir)
	layout, _, err := analyzeAudioLayout(context.Background(), src, 3)
	if err != nil {
		t.Fatalf("analyzeAudioLayout: %v", err)
	}
	if !layout.Track0FullMix {
		t.Fatal("Track0FullMix = false, want true")
	}
	if layout.GameComponent != 2 {
		t.Errorf("GameComponent = %d, want 2 (le jeu est la 2ᵉ composante, classé par acoustique)", layout.GameComponent)
	}
}

// amExpr : porteuse sinusoïdale modulée en amplitude par un LFO lent (forte
// dynamique d'enveloppe, comme un vrai mix de jeu).
func amExpr(carrier, lfo string) string {
	return "0.6*sin(2*PI*" + carrier + "*t)*(0.5+0.49*sin(2*PI*" + lfo + "*t))"
}

// generateAudioMKV produit un MKV vidéo synthétique + 3 pistes audio (sources AM à
// porteuses/LFO distincts) recombinées via le filter_complex fourni, mappées dans
// l'ordre maps.
func generateAudioMKV(t *testing.T, dir, name, filter string, maps []string) string {
	t.Helper()
	out := filepath.Join(dir, name)
	src := func(c, l string) []string {
		return []string{"-f", "lavfi", "-i", "aevalsrc=" + amExpr(c, l) + ":d=3:s=8000"}
	}
	args := ffmpegQuietArgs("-y")
	args = append(args, src("440", "0.7")...)
	args = append(args, src("880", "1.1")...)
	args = append(args, src("660", "0.5")...)
	args = append(args, "-f", "lavfi", "-i", "testsrc=size=320x240:rate=15:duration=3",
		"-filter_complex", filter, "-map", "3:v")
	for _, m := range maps {
		args = append(args, "-map", m)
	}
	args = append(args, "-c:v", "libx264", "-preset", "ultrafast", "-c:a", "libopus", out)
	if o, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Fatalf("génération %s: %v\n%s", name, err, o)
	}
	return out
}

// generateGameVoiceMKV produit un MKV où la piste 0 = mix complet, la 1ʳᵉ composante
// (0:a:1) = voix INTERMITTENTE (enveloppe cubée → salves + silences, fort taux de
// silence) et la 2ᵉ (0:a:2) = jeu CONTINU (faible variation d'enveloppe, peu de
// silence). Le jeu, plus continu, doit être classé `game` malgré sa position (2).
func generateGameVoiceMKV(t *testing.T, dir string) string {
	t.Helper()
	out := filepath.Join(dir, "gamevoice.mkv")
	game := "0.4*sin(2*PI*200*t)*(0.75+0.2*sin(2*PI*1.3*t))" // continu (~5 dB de swing)
	// voix : enveloppe (0.5+0.5*sin) AU CUBE (pas de virgule/pow → compatible lavfi -i)
	// → reste près de 0 longtemps, salves brèves = intermittent.
	burst := "(0.5+0.5*sin(2*PI*0.4*t))"
	voice := "0.5*sin(2*PI*350*t)*" + burst + "*" + burst + "*" + burst
	args := ffmpegQuietArgs("-y",
		"-f", "lavfi", "-i", "aevalsrc="+game+":d=5:s=8000",
		"-f", "lavfi", "-i", "aevalsrc="+voice+":d=5:s=8000",
		"-f", "lavfi", "-i", "testsrc=size=320x240:rate=15:duration=5",
		// piste 0 = amix(jeu, voix) ; 0:a:1 = voix ; 0:a:2 = jeu (jeu non premier).
		"-filter_complex", "[0:a]asplit=2[ga][gb];[1:a]asplit=2[va][vb];"+
			"[ga][va]amix=inputs=2:normalize=0[mix]",
		"-map", "2:v", "-map", "[mix]", "-map", "[vb]", "-map", "[gb]",
		"-c:v", "libx264", "-preset", "ultrafast", "-c:a", "libopus", out)
	if o, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Fatalf("génération gamevoice.mkv: %v\n%s", err, o)
	}
	return out
}
