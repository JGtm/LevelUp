package media

import (
	"context"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDBToPower(t *testing.T) {
	cases := []struct{ db, want float64 }{
		{0, 1}, {-10, 0.1}, {-20, 0.01}, {10, 10},
	}
	for _, c := range cases {
		if got := dbToPower([]float64{c.db})[0]; math.Abs(got-c.want) > 1e-9 {
			t.Errorf("dbToPower(%v) = %v, want %v", c.db, got, c.want)
		}
	}
}

func TestNNLS_RecoversGains(t *testing.T) {
	// A = [[1,0],[0,1],[1,1]], x = [1,0.5] → b = [1,0.5,1.5].
	// AtA = [[2,1],[1,2]], Atb = [2.5,2.0] → g = [1,0.5].
	g := nnls([][]float64{{2, 1}, {1, 2}}, []float64{2.5, 2.0})
	if len(g) != 2 || math.Abs(g[0]-1) > 1e-9 || math.Abs(g[1]-0.5) > 1e-9 {
		t.Errorf("nnls = %v, want [1 0.5]", g)
	}
}

func TestNNLS_ClampsNegative(t *testing.T) {
	// Solution non contrainte = [1,-1] ; sous g ≥ 0, la 2ᵉ variable est clampée à 0.
	g := nnls([][]float64{{1, 0}, {0, 1}}, []float64{1, -1})
	if len(g) != 2 || math.Abs(g[0]-1) > 1e-9 || g[1] != 0 {
		t.Errorf("nnls = %v, want [1 0]", g)
	}
}

// synthEnv construit l'enveloppe dB d'une puissance base+amp·(0.5+0.5·sin(2π·freq·t/n)).
func synthEnv(base, amp, freq float64, n int) []float64 {
	out := make([]float64, n)
	for t := 0; t < n; t++ {
		p := base + amp*(0.5+0.5*math.Sin(2*math.Pi*freq*float64(t)/float64(n)))
		out[t] = 10 * math.Log10(p)
	}
	return out
}

// mixPower recombine des enveloppes dB de composantes en une enveloppe dB de mix
// pondéré (domaine puissance) : P0 = Σ wᵢ·Pᵢ.
func mixPower(weights []float64, compsDB [][]float64) []float64 {
	n := len(compsDB[0])
	out := make([]float64, n)
	for t := 0; t < n; t++ {
		var p float64
		for i, c := range compsDB {
			p += weights[i] * math.Pow(10, c[t]/10)
		}
		out[t] = 10 * math.Log10(p)
	}
	return out
}

func TestDecideFullMix_UnequalGains(t *testing.T) {
	// 3 composantes distinctes (fréquences d'enveloppe séparées), forte dynamique
	// dB (comme un vrai enregistrement), piste 0 = mix pondéré 1.0/0.5/0.3 →
	// ajustement quasi parfait, toutes actives → full-mix.
	comps := [][]float64{
		synthEnv(0.05, 1.2, 3, 120),
		synthEnv(0.05, 0.9, 7, 120),
		synthEnv(0.05, 0.7, 11, 120),
	}
	env0 := mixPower([]float64{1.0, 0.5, 0.3}, comps)
	dec := decideFullMix(env0, comps)
	if !dec.IsFullMix {
		t.Fatalf("IsFullMix = false (R²=%.4f, gains=%v, shares=%v), want true", dec.R2, dec.Gains, dec.Shares)
	}
	if dec.R2 < 0.99 {
		t.Errorf("R² = %.4f, want ≥ 0.99 (mix exact)", dec.R2)
	}
	if math.Abs(dec.Gains[0]-1.0) > 0.05 || math.Abs(dec.Gains[1]-0.5) > 0.05 || math.Abs(dec.Gains[2]-0.3) > 0.05 {
		t.Errorf("gains = %v, want ≈ [1 0.5 0.3]", dec.Gains)
	}
}

func TestDecideFullMix_DisjointRejectedByCoverage(t *testing.T) {
	// PIÈGE : piste 0 = copie EXACTE de la composante 0 (pas un mix). L'ajustement à
	// gains libres donne R²≈1 (g0≈1, g1≈g2≈0), mais les composantes 1/2 sont actives
	// et ne contribuent PAS → la couverture rejette le faux positif full-mix.
	comps := [][]float64{
		synthEnv(0.05, 1.2, 3, 120),
		synthEnv(0.05, 0.9, 7, 120),
		synthEnv(0.05, 0.7, 11, 120),
	}
	env0 := append([]float64(nil), comps[0]...) // piste 0 ≡ composante 0
	dec := decideFullMix(env0, comps)
	if dec.R2 < 0.99 {
		t.Errorf("R² = %.4f, want ≥ 0.99 (piste0 = composante0)", dec.R2)
	}
	if dec.IsFullMix {
		t.Fatalf("IsFullMix = true (gains=%v, shares=%v, active=%v), want false (couverture)", dec.Gains, dec.Shares, dec.Active)
	}
	if !dec.Active[1] || !dec.Active[2] {
		t.Errorf("composantes 1/2 devraient être actives : %v", dec.Active)
	}
}

func TestDecideFullMix_SilentComponentIgnored(t *testing.T) {
	// Session solo : micro coupé (composante 1 au plancher −91 dB). Piste 0 = jeu seul.
	// La composante muette n'impose aucune couverture → full-mix accepté (full lit
	// la piste 0, game = composante active).
	game := synthEnv(0.05, 1.2, 3, 120)
	silent := make([]float64, 120)
	for i := range silent {
		silent[i] = rmsFloorDB
	}
	env0 := append([]float64(nil), game...)
	dec := decideFullMix(env0, [][]float64{game, silent})
	if !dec.IsFullMix {
		t.Fatalf("IsFullMix = false (R²=%.4f, active=%v), want true", dec.R2, dec.Active)
	}
	if dec.Active[0] != true || dec.Active[1] != false {
		t.Errorf("active = %v, want [true false] (composante 1 muette)", dec.Active)
	}
}

func TestDecideFullMix_StationaryRejected(t *testing.T) {
	// Piste 0 quasi constante (ton pur) → variance d'enveloppe insuffisante → pas de
	// classement full-mix (repli mapping historique), peu importe l'ajustement.
	n := 120
	env0 := make([]float64, n)
	comp := make([]float64, n)
	for i := 0; i < n; i++ {
		env0[i] = -12.0 // constant
		comp[i] = -12.0
	}
	dec := decideFullMix(env0, [][]float64{comp})
	if dec.IsFullMix {
		t.Errorf("IsFullMix = true (stdDB=%.3f), want false (stationnaire)", dec.Env0StdDB)
	}
}

// --- Intégration ffmpeg : synthétiques + vrais clips OBS ---

func TestDecideFullMix_SyntheticWeightedMix_Integration(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent du PATH")
	}
	dir := t.TempDir()

	// FULL-MIX à gains INÉGAUX : piste 0 = 1.0·A + 0.35·B, piste 1 = A, piste 2 = B.
	// A/B = AM à porteuses/LFO distincts (voix plus faible). Doit être détecté full-mix.
	mix := weightedMixMKV(t, dir)
	layout, r2, err := analyzeAudioLayout(context.Background(), mix, audioStreamsNoTitle(3))
	if err != nil {
		t.Fatalf("analyzeAudioLayout (mix pondéré): %v", err)
	}
	if !layout.Track0FullMix {
		t.Errorf("mix pondéré 1.0/0.35 : Track0FullMix = false (R²=%.4f), want true", r2)
	}

	// DISJOINT 2 pistes : piste 0 = ton A, piste 1 = ton B distinct → pas un mix.
	disj := twinTrackMKV(t, dir, "disjoint.mkv", false)
	layout2, r22, err := analyzeAudioLayout(context.Background(), disj, audioStreamsNoTitle(2))
	if err != nil {
		t.Fatalf("analyzeAudioLayout (disjoint): %v", err)
	}
	if layout2.Track0FullMix {
		t.Errorf("disjoint A/B : Track0FullMix = true (R²=%.4f), want false", r22)
	}
}

// weightedMixMKV : MKV vidéo + 3 pistes audio où piste 0 = amix(A·1.0, B·0.35),
// piste 1 = A (jeu, continu), piste 2 = B (voix, plus faible). Gains de mix inégaux.
func weightedMixMKV(t *testing.T, dir string) string {
	t.Helper()
	out := filepath.Join(dir, "weightedmix.mkv")
	a := amExpr("200", "1.3") // jeu continu
	b := amExpr("350", "0.6") // voix, rythme distinct
	args := ffmpegQuietArgs("-y",
		"-f", "lavfi", "-i", "aevalsrc="+a+":d=4:s=8000",
		"-f", "lavfi", "-i", "aevalsrc="+b+":d=4:s=8000",
		"-f", "lavfi", "-i", "testsrc=size=320x240:rate=15:duration=4",
		"-filter_complex", "[0:a]asplit=2[a0][a1];[1:a]asplit=2[b0][b1];"+
			"[b0]volume=0.35[bs];[a0][bs]amix=inputs=2:normalize=0[mix]",
		"-map", "2:v", "-map", "[mix]", "-map", "[a1]", "-map", "[b1]",
		"-c:v", "libx264", "-preset", "ultrafast", "-c:a", "libopus", out)
	if o, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Fatalf("génération weightedmix.mkv: %v\n%s", err, o)
	}
	return out
}

// TestAnalyzeAudioLayout_RealClips valide le détecteur sur les VRAIS enregistrements
// OBS (lecture seule). Env-gated : LEVELUP_MEDIA_TEST_DIR doit pointer le dossier des
// captures ; skip propre sinon (PAS de chemin utilisateur en dur dans le code versionné).
func TestAnalyzeAudioLayout_RealClips(t *testing.T) {
	baseDir := os.Getenv("LEVELUP_MEDIA_TEST_DIR")
	if baseDir == "" {
		t.Skip("LEVELUP_MEDIA_TEST_DIR non défini (validation sur vrais clips)")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent du PATH")
	}
	cases := []struct {
		file     string
		wantMix  bool
		wantGame int // composante jeu attendue (0:a:N ; 0 = ne pas vérifier)
	}{
		{"Replay 2026-07-03 21-38-44.mkv", true, 1}, // 4 pistes génériques, voix → jeu classé par continuité (0:a:1)
		{"Replay 2026-07-07 23-03-38.mkv", true, 1}, // pistes titrées full/game/voices → jeu = 0:a:1 (titre)
		{"Replay 2026-07-16 16-07-19.mkv", true, 1}, // titrées, sans voix (non-régression) → jeu = 0:a:1 (titre)
	}
	for _, c := range cases {
		path := filepath.Join(baseDir, c.file)
		if _, err := os.Stat(path); err != nil {
			t.Logf("SKIP %s (absent)", c.file)
			continue
		}
		rep, err := AnalyzeAudioLayoutReport(context.Background(), path)
		if err != nil {
			t.Fatalf("%s: AnalyzeAudioLayoutReport: %v", c.file, err)
		}
		t.Logf("%s : pistes=%d full_mix=%v game=%d R²=%.4f env0std=%.2f gains=%v shares=%v pcorr=%v p90=%v active=%v silence=%v",
			c.file, rep.AudioTracks, rep.Track0FullMix, rep.GameComponent, rep.R2,
			rep.Env0StdDB, fmtF(rep.Gains), fmtF(rep.Shares), fmtF(rep.PowerCorr),
			fmtF(rep.P90), rep.Active, fmtF(rep.SilenceRatios))
		if rep.Track0FullMix != c.wantMix {
			t.Errorf("%s : Track0FullMix = %v, want %v (R²=%.4f)", c.file, rep.Track0FullMix, c.wantMix, rep.R2)
		}
		if c.wantGame != 0 && rep.GameComponent != c.wantGame {
			t.Errorf("%s : GameComponent = %d, want %d", c.file, rep.GameComponent, c.wantGame)
		}
	}
}

// fmtF arrondit une série pour un log lisible (2 décimales).
func fmtF(xs []float64) []float64 {
	out := make([]float64, len(xs))
	for i, x := range xs {
		out[i] = math.Round(x*100) / 100
	}
	return out
}
