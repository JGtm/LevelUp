package media

// hls_audio_analyze.go — détection du rôle des pistes audio source à l'ingestion (IO).
//
// Problème : certaines captures (OBS « capture de sortie ») exportent une piste 0 qui
// est DÉJÀ le mix complet (jeu + micro + voix), suivie des pistes composantes (jeu,
// micro, Discord…). Le mapping HLS historique traite la piste 0 comme « le jeu » puis
// synthétise `full = amix(toutes les pistes)` → il ré-ajoute le mix complet par-dessus
// lui-même = doublage audible (écho). Cf. fix audio HLS Firefox juin 2026.
//
// Détection : on décode l'enveloppe de loudness (RMS par trame ~100 ms) de la piste 0
// et de CHAQUE composante, puis (audio_fullmix_fit.go, pur) on ajuste la puissance de
// la piste 0 comme une combinaison à gains ≥ 0 des puissances des composantes. Une
// bonne qualité d'ajustement (R²) AVEC couverture de chaque composante active ⇒ piste 0
// = mix complet → `full` la lit directement (pas d'amix doublé). Cette régression en
// domaine puissance est robuste aux gains de mix OBS ≠ 1:1 et à la voix, là où une
// simple corrélation d'enveloppe dB échouait dès qu'il y avait de la voix significative.
//
// L'audio est extrait en PCM brut mono 8 kHz sur stdout (déterministe, pas de
// dépendance au format de log ffmpeg ni de chemin échappé dans le filtergraph) ;
// enveloppe RMS et régression sont calculées en Go (pures, testées).

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

// fullMixEnvelopeCorrThreshold : seuil de corrélation d'enveloppe game vs voices pour
// juger deux renditions HLS REDONDANTES (même son) au collapse en place
// (hls_audio_collapse.go) — clips legacy sans voix isolée, dont les renditions portent
// toutes le même mix. Au-dessus = redondant (collapse), en dessous = distinct (toggle
// conservé). La détection full-mix à l'ingestion utilise désormais la régression
// puissance (audio_fullmix_fit.go, robuste à la voix), pas cette corrélation.
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

// envMaxAnalysisSeconds borne la durée décodée par audioEnvelope. audioEnvelope
// bufferise TOUT le PCM décodé en RAM (~115 Mo/h/passe à 8 kHz f32, 2+N passes par
// clip) : sur le VPS 2 Go, un clip de plusieurs heures épuiserait la mémoire. La
// corrélation d'enveloppe et le taux de silence convergent bien avant 10 min —
// cette borne est une garde mémoire, pas un paramètre d'analyse.
const envMaxAnalysisSeconds = 600

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
// (ex. "[mix]"). Décode au plus envMaxAnalysisSeconds (garde mémoire). IO ffmpeg.
func audioEnvelope(ctx context.Context, src, filterComplex, mapSpec string) ([]float64, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg", buildEnvelopeArgs(src, filterComplex, mapSpec)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg pcm: %w (stderr: %s)", err, stderr.String())
	}
	return rmsFramesDB(stdout.Bytes(), envFrameSamples), nil
}

// buildEnvelopeArgs construit les arguments ffmpeg d'audioEnvelope. Extrait pour
// être testable sans lancer ffmpeg. `-t` est placé en option d'ENTRÉE (avant -i)
// pour ARRÊTER le décodage à envMaxAnalysisSeconds — borne la RAM à la source,
// que l'entrée soit un fichier direct ou une sous-playlist HLS (collapse).
func buildEnvelopeArgs(src, filterComplex, mapSpec string) []string {
	args := ffmpegQuietArgs("-y", "-t", fmt.Sprintf("%d", envMaxAnalysisSeconds), "-i", src)
	if filterComplex != "" {
		args = append(args, "-filter_complex", filterComplex)
	}
	return append(args, "-map", mapSpec, "-ac", "1", "-ar",
		fmt.Sprintf("%d", envSampleRate), "-f", "f32le", "-")
}

// decodeAudioEnvelopes décode l'enveloppe RMS (dB) de la piste 0 et de chaque
// composante 1..nAudio-1 (map direct par piste). IO ffmpeg. Hypothèse : nAudio ≥ 2.
func decodeAudioEnvelopes(ctx context.Context, src string, nAudio int) ([]float64, [][]float64, error) {
	env0, err := audioEnvelope(ctx, src, "", "0:a:0")
	if err != nil {
		return nil, nil, err
	}
	comps := make([][]float64, 0, nAudio-1)
	for i := 1; i < nAudio; i++ {
		env, err := audioEnvelope(ctx, src, "", fmt.Sprintf("0:a:%d", i))
		if err != nil {
			return nil, nil, err
		}
		comps = append(comps, env)
	}
	return env0, comps, nil
}

// analyzeAudioLayout déduit le rôle des pistes audio source (IO ffmpeg) :
//  1. la piste 0 est-elle le mix complet des composantes (régression puissance :
//     R² ≥ seuil, garde de stationnarité + couverture ; cf. decideFullMix) ;
//  2. si oui, QUELLE composante est le jeu (titre de piste, sinon acoustique).
//
// audioStreams = pistes AUDIO source en ordre (audioStreams[i] = 0:a:i). Retourne aussi
// le R² de l'ajustement (log). Hypothèse : len(audioStreams) ≥ 2.
func analyzeAudioLayout(ctx context.Context, src string, audioStreams []AVStreamDetail) (audioLayout, float64, error) {
	var layout audioLayout
	nAudio := len(audioStreams)
	if nAudio < 2 {
		return layout, 0, nil
	}
	env0, comps, err := decodeAudioEnvelopes(ctx, src, nAudio)
	if err != nil {
		return layout, 0, err
	}
	dec := decideFullMix(env0, comps)
	if !dec.IsFullMix {
		return layout, dec.R2, nil
	}
	layout.Track0FullMix = true
	layout.GameComponent = gameComponentIndex(ctx, src, audioStreams)
	return layout, dec.R2, nil
}

// gameComponentIndex choisit l'index audio (0:a:N, N ≥ 1) de la composante « jeu » d'un
// layout full-mix : le TITRE de piste (métadonnée OBS « game »/« jeu ») fait foi quand
// il désigne une composante ; sinon repli sur classifyGameComponent (continuité
// acoustique / taux de silence). Le titre est autoritaire là où le taux de silence se
// trompe (voix Discord plus continue que le jeu ; piste muette de silence nul). IO
// (le repli décode l'audio).
func gameComponentIndex(ctx context.Context, src string, audioStreams []AVStreamDetail) int {
	for i := 1; i < len(audioStreams); i++ {
		if titleIsGame(audioStreams[i].Title) {
			return i
		}
	}
	return classifyGameComponent(ctx, src, len(audioStreams))
}

// titleIsGame indique si un titre de piste OBS désigne la composante jeu.
func titleIsGame(title string) bool {
	switch strings.ToLower(strings.TrimSpace(title)) {
	case audioRenditionGameSlug, "jeu":
		return true
	}
	return false
}

// audioStreamsOnly filtre les pistes audio d'une liste de streams, en préservant
// l'ordre (résultat[i] = 0:a:i).
func audioStreamsOnly(streams []AVStreamDetail) []AVStreamDetail {
	out := make([]AVStreamDetail, 0, len(streams))
	for _, s := range streams {
		if s.CodecType == codecTypeAudio {
			out = append(out, s)
		}
	}
	return out
}

// AudioLayoutReport résume l'analyse de layout d'un fichier source (diag / --analyze).
// Gains/Shares/Active/SilenceRatios sont indexés par composante (0:a:1 .. 0:a:N-1).
type AudioLayoutReport struct {
	AudioTracks   int
	Track0FullMix bool
	GameComponent int // index audio 0:a:N de la composante jeu (si full-mix)
	R2            float64
	Env0StdDB     float64
	Gains         []float64
	Shares        []float64
	PowerCorr     []float64
	P90           []float64
	Active        []bool
	SilenceRatios []float64
}

// AnalyzeAudioLayoutReport sonde un fichier source et retourne l'analyse complète du
// layout audio (décision full-mix + métriques) sans rien écrire. Outil de diagnostic
// durable (cf. cmd/migrate-hls-audio --analyze). IO ffmpeg + ffprobe.
func AnalyzeAudioLayoutReport(ctx context.Context, src string) (AudioLayoutReport, error) {
	streams, err := ProbeStreamsDetailed(ctx, src)
	if err != nil {
		return AudioLayoutReport{}, err
	}
	audioStreams := audioStreamsOnly(streams)
	n := len(audioStreams)
	rep := AudioLayoutReport{AudioTracks: n}
	if n < 2 {
		return rep, nil
	}
	env0, comps, err := decodeAudioEnvelopes(ctx, src, n)
	if err != nil {
		return rep, err
	}
	dec := decideFullMix(env0, comps)
	rep.Track0FullMix, rep.R2, rep.Env0StdDB = dec.IsFullMix, dec.R2, dec.Env0StdDB
	rep.Gains, rep.Shares, rep.Active = dec.Gains, dec.Shares, dec.Active
	rep.PowerCorr, rep.P90 = dec.PowerCorr, dec.P90
	rep.SilenceRatios = make([]float64, len(comps))
	for i, c := range comps {
		rep.SilenceRatios[i] = silenceRatio(c)
	}
	if dec.IsFullMix {
		rep.GameComponent = gameComponentIndex(ctx, src, audioStreams)
	}
	return rep, nil
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
