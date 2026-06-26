package media

// hls_audio_analyze.go — détection du rôle des pistes audio source à l'ingestion.
//
// Problème : certaines captures (OBS « capture de sortie » en piste 1) exportent
// une piste 0 qui est DÉJÀ le mix complet (jeu + micro + voix), suivie des pistes
// composantes (jeu, micro, Discord…). Le mapping HLS historique traite la piste 0
// comme « le jeu » puis synthétise `full = amix(toutes les pistes)` → il ré-ajoute
// le mix complet par-dessus lui-même = doublage audible (écho). Cf. fix audio HLS
// Firefox juin 2026.
//
// On détecte ce cas en comparant l'ENVELOPPE de loudness (RMS par trame ~100 ms) de
// la piste 0 à celle de `amix(pistes 1..N)` : si elles sont fortement corrélées, la
// piste 0 est le mix complet → `full` doit la lire directement (pas d'amix doublé).
// L'enveloppe est robuste au déphasage / dérive d'horloge (≠ corrélation au lag 0).
//
// L'audio est extrait en PCM brut mono 8 kHz sur stdout (déterministe, pas de
// dépendance au format de log ffmpeg ni de chemin échappé dans le filtergraph) ;
// l'enveloppe RMS par trame et la corrélation sont calculées en Go (pures, testées).

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os/exec"
	"sort"
	"strings"
)

// fullMixEnvelopeCorrThreshold : au-dessus, la piste 0 est considérée comme le mix
// complet des autres. Calibré sur 6 clips OBS réels (sessions solo, piste 1 = capture
// de sortie) : corrélation d'enveloppe piste0 vs amix(reste) mesurée 0,847 → 0,995.
// Un layout sans mix complet (piste 0 = composante disjointe, ex. jeu seul + voix
// seule) tombe bien en dessous → mapping historique conservé.
const fullMixEnvelopeCorrThreshold = 0.80

// rmsFloorDB plancher des trames silencieuses (RMS nul) pour garder une enveloppe
// numérique exploitable par la corrélation.
const rmsFloorDB = -91.0

// minEnvelopeStdDevDB : en dessous (enveloppe quasi stationnaire — ton pur, signal
// constant), la corrélation d'enveloppe n'est pas fiable → on ne classe PAS la piste
// 0 comme mix complet (repli mapping historique). Les vrais enregistrements de jeu
// ont une dynamique de plusieurs dB, bien au-dessus.
const minEnvelopeStdDevDB = 2.0

// envSampleRate / envFrameSamples : PCM mono ré-échantillonné à 8 kHz, trames de
// 800 échantillons = 100 ms (résolution d'enveloppe de loudness suffisante).
const (
	envSampleRate   = 8000
	envFrameSamples = 800
)

// silenceDropDB : une trame est « silencieuse » si son RMS est plus de 25 dB sous le
// niveau fort (90ᵉ centile) de SA piste. Capture la CONTINUITÉ indépendamment du
// volume absolu : le son de jeu est continu (peu de trames silencieuses), la voix
// (micro/Discord) est intermittente (beaucoup de silences entre les phrases).
// Calibré sur clips réels (jeu 9-13 % de silence vs voix 16-26 %).
const silenceDropDB = 25.0

// gameClassifyMarginRatio : écart minimal de taux de silence entre la composante la
// plus continue (jeu) et la suivante pour faire confiance au classement. En dessous
// (égalité — deux pistes aussi continues), repli sur la convention positionnelle.
const gameClassifyMarginRatio = 0.05

// audioLayout décrit le rôle des pistes audio source, déduit à l'ingestion (IO).
// Quand Track0FullMix, la piste 0 est le mix complet de sortie et GameComponent est
// l'index audio (0:a:N) de la composante « jeu » (classée acoustiquement, pas par
// position) ; les autres composantes sont la voix.
type audioLayout struct {
	Track0FullMix bool
	GameComponent int // index audio de la composante jeu (≥1) quand Track0FullMix
}

// pearson retourne le coefficient de corrélation de Pearson des deux séries,
// tronquées à leur longueur commune. Pur. Retourne 0 si trop peu d'échantillons ou
// si une série est constante (variance nulle → corrélation indéfinie).
func pearson(xs, ys []float64) float64 {
	n := len(xs)
	if len(ys) < n {
		n = len(ys)
	}
	if n < 5 {
		return 0
	}
	var sx, sy float64
	for i := 0; i < n; i++ {
		sx += xs[i]
		sy += ys[i]
	}
	mx, my := sx/float64(n), sy/float64(n)
	var sxy, sxx, syy float64
	for i := 0; i < n; i++ {
		dx, dy := xs[i]-mx, ys[i]-my
		sxy += dx * dy
		sxx += dx * dx
		syy += dy * dy
	}
	if sxx <= 0 || syy <= 0 {
		return 0
	}
	return sxy / math.Sqrt(sxx*syy)
}

// stdDev retourne l'écart-type d'une série. Pur. 0 si moins de 2 éléments.
func stdDev(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	var s float64
	for _, x := range xs {
		s += x
	}
	m := s / float64(len(xs))
	var v float64
	for _, x := range xs {
		d := x - m
		v += d * d
	}
	return math.Sqrt(v / float64(len(xs)))
}

// rmsFramesDB découpe un buffer PCM float32 little-endian (mono) en trames de
// frameSamples et retourne le niveau RMS de chaque trame en dB (planché à
// rmsFloorDB). Pur.
func rmsFramesDB(raw []byte, frameSamples int) []float64 {
	n := len(raw) / 4
	if frameSamples <= 0 || n < frameSamples {
		return nil
	}
	out := make([]float64, 0, n/frameSamples)
	for i := 0; i+frameSamples <= n; i += frameSamples {
		var sumSq float64
		for j := 0; j < frameSamples; j++ {
			bits := binary.LittleEndian.Uint32(raw[(i+j)*4:])
			v := float64(math.Float32frombits(bits))
			sumSq += v * v
		}
		db := rmsFloorDB
		if rms := math.Sqrt(sumSq / float64(frameSamples)); rms > 0 {
			if db = 20 * math.Log10(rms); db < rmsFloorDB {
				db = rmsFloorDB
			}
		}
		out = append(out, db)
	}
	return out
}

// audioEnvelope décode un flux audio en PCM mono 8 kHz via ffmpeg et retourne son
// enveloppe RMS par trame (dB). filterComplex/mapSpec sélectionnent le flux :
// filterComplex vide → map direct (ex. "0:a:0") ; sinon le filtre produit mapSpec
// (ex. "[mix]"). IO ffmpeg.
func audioEnvelope(ctx context.Context, src, filterComplex, mapSpec string) ([]float64, error) {
	args := []string{"-hide_banner", "-loglevel", "error", "-y", "-i", src}
	if filterComplex != "" {
		args = append(args, "-filter_complex", filterComplex)
	}
	args = append(args, "-map", mapSpec, "-ac", "1", "-ar",
		fmt.Sprintf("%d", envSampleRate), "-f", "f32le", "-")
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg pcm: %w (stderr: %s)", err, stderr.String())
	}
	return rmsFramesDB(stdout.Bytes(), envFrameSamples), nil
}

// restMixFilter retourne le couple (filterComplex, mapSpec) extrayant le mix des
// pistes audio 1..nAudio-1. Une seule piste restante → map direct ; sinon amix.
func restMixFilter(nAudio int) (string, string) {
	if nAudio-1 == 1 {
		return "", "0:a:1"
	}
	var b strings.Builder
	for i := 1; i < nAudio; i++ {
		fmt.Fprintf(&b, "[0:a:%d]", i)
	}
	fmt.Fprintf(&b, "amix=inputs=%d:normalize=0[mix]", nAudio-1)
	return b.String(), "[mix]"
}

// analyzeAudioLayout déduit le rôle des pistes audio source (IO ffmpeg) :
//  1. la piste 0 est-elle le mix complet des pistes 1..N (corrélation d'enveloppe
//     ≥ seuil, gardée contre les signaux stationnaires) ;
//  2. si oui, QUELLE composante est le jeu (classement acoustique, pas l'ordre).
//
// Retourne aussi la corrélation mix mesurée (log). Hypothèse : nAudio ≥ 2.
func analyzeAudioLayout(ctx context.Context, src string, nAudio int) (audioLayout, float64, error) {
	var layout audioLayout
	if nAudio < 2 {
		return layout, 0, nil
	}
	env0, err := audioEnvelope(ctx, src, "", "0:a:0")
	if err != nil {
		return layout, 0, err
	}
	fc, mapSpec := restMixFilter(nAudio)
	envRest, err := audioEnvelope(ctx, src, fc, mapSpec)
	if err != nil {
		return layout, 0, err
	}
	corr := pearson(env0, envRest)
	// Enveloppe trop stationnaire (ton pur, signal constant) → corrélation non
	// fiable → on ne classe pas en mix complet (repli mapping historique).
	if stdDev(env0) < minEnvelopeStdDevDB || stdDev(envRest) < minEnvelopeStdDevDB {
		return layout, corr, nil
	}
	if corr < fullMixEnvelopeCorrThreshold {
		return layout, corr, nil
	}
	layout.Track0FullMix = true
	layout.GameComponent = classifyGameComponent(ctx, src, nAudio)
	return layout, corr, nil
}

// silenceRatio retourne la fraction de trames dont le RMS est plus de silenceDropDB
// sous le niveau fort (90ᵉ centile) de l'enveloppe — mesure de discontinuité. Pur.
func silenceRatio(env []float64) float64 {
	if len(env) < 5 {
		return 0
	}
	sorted := append([]float64(nil), env...)
	sort.Float64s(sorted)
	thr := sorted[int(0.9*float64(len(sorted)))] - silenceDropDB
	c := 0
	for _, v := range env {
		if v < thr {
			c++
		}
	}
	return float64(c) / float64(len(env))
}

// classifyGameComponent identifie la composante « jeu » parmi les pistes 1..N-1 : le
// son de jeu est CONTINU (taux de silence le plus bas), la voix (micro/Discord)
// intermittente. Indépendant du volume ET de l'ordre des pistes. Repli sur la 1ère
// composante (index 1) si le décodage échoue ou si les deux plus continues sont à
// égalité (marge < gameClassifyMarginRatio). IO ffmpeg (best-effort, n'échoue pas).
func classifyGameComponent(ctx context.Context, src string, nAudio int) int {
	best, second := 1, -1
	bestSil, secondSil := 2.0, 2.0
	for i := 1; i < nAudio; i++ {
		env, err := audioEnvelope(ctx, src, "", fmt.Sprintf("0:a:%d", i))
		if err != nil {
			continue
		}
		sil := silenceRatio(env)
		switch {
		case sil < bestSil:
			second, secondSil = best, bestSil
			best, bestSil = i, sil
		case sil < secondSil:
			second, secondSil = i, sil
		}
	}
	if second != -1 && secondSil-bestSil < gameClassifyMarginRatio {
		return 1 // égalité → convention positionnelle
	}
	return best
}
