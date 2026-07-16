package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
)

// ErrRemuxProbeFailed et ErrRemuxIncompatibleCodec distinguent les deux modes
// d'échec du pré-flight (PlanRemuxWebM) pour que l'appelant HTTP choisisse son
// status AVANT d'écrire le moindre octet : probe cassé (source illisible / ffprobe
// indisponible) vs codecs source incompatibles WebM.
var (
	ErrRemuxProbeFailed       = errors.New("remux: pré-flight ffprobe échoué")
	ErrRemuxIncompatibleCodec = errors.New("remux: codecs source incompatibles WebM")
)

// RemuxPlan porte la décision de remux validée par le pré-flight (codecs
// compatibles WebM confirmés) : le seul paramètre variable est le -map audio.
// Construit par PlanRemuxWebM, consommé par StreamRemuxWebMPlan.
type RemuxPlan struct {
	AudioMap string // argument ffmpeg -map audio (0:a, 0:a:0 ou 0:a?)
}

// PlanRemuxWebM exécute le PRÉ-FLIGHT du remux WebM (ffprobe + validation codecs)
// SANS produire d'octet : c'est le point où l'appelant HTTP peut encore répondre
// un status d'erreur. Retourne une erreur enveloppant ErrRemuxProbeFailed (probe
// impossible) ou ErrRemuxIncompatibleCodec (vidéo non av1/vp8/vp9, ou aucune piste
// audio compatible). ffmpeg + ffprobe doivent être dans le PATH.
//
// Stratégie pistes audio :
//   - Toutes les pistes Opus/Vorbis → -map 0:a (toutes copiées)
//   - Première piste Opus/Vorbis mais pas toutes → -map 0:a:0 (première seule)
//   - Aucune piste compatible → ErrRemuxIncompatibleCodec
func PlanRemuxWebM(ctx context.Context, absPath string) (RemuxPlan, error) {
	info, err := probeAVStreams(ctx, absPath)
	if err != nil {
		return RemuxPlan{}, fmt.Errorf("%w: %v", ErrRemuxProbeFailed, err)
	}
	if !isWebMCompatibleVideoCodec(info.VideoCodec) {
		return RemuxPlan{}, fmt.Errorf("%w: vidéo %q (attendu av1/vp8/vp9)", ErrRemuxIncompatibleCodec, info.VideoCodec)
	}
	audioMap, err := chooseAudioMap(info.AudioCodecs)
	if err != nil {
		return RemuxPlan{}, fmt.Errorf("%w: %v", ErrRemuxIncompatibleCodec, err)
	}
	return RemuxPlan{AudioMap: audioMap}, nil
}

// StreamRemuxWebMPlan lance ffmpeg (remux -c copy vers WebM) selon un RemuxPlan
// DÉJÀ validé par PlanRemuxWebM et streame le résultat dans w. Ne re-probe PAS :
// l'appelant a déjà tranché codecs et status. Un échec ici survient en cours de
// flux (status HTTP déjà envoyé) → l'appelant ne peut plus que logger. Le caller
// est responsable du Content-Type (video/webm) ; les Range requests ne sont pas
// supportés sur le flux remuxé.
func StreamRemuxWebMPlan(ctx context.Context, absPath string, plan RemuxPlan, w io.Writer) error {
	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-i", absPath,
		"-map", "0:v:0",
		"-map", plan.AudioMap,
		"-c", "copy",
		"-f", "webm",
		"pipe:1",
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stdout = w
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		slog.ErrorContext(ctx, "remux ffmpeg failed",
			"path", absPath,
			"audio_map", plan.AudioMap,
			"stderr", stderr.String(),
			"err", err)
		return fmt.Errorf("ffmpeg remux: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}

// avStreamsInfo synthétise les streams retournés par ffprobe.
type avStreamsInfo struct {
	VideoCodec  string
	AudioCodecs []string
}

func probeAVStreams(ctx context.Context, absPath string) (avStreamsInfo, error) {
	out, err := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		absPath,
	).Output()
	if err != nil {
		return avStreamsInfo{}, err
	}
	var parsed struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return avStreamsInfo{}, fmt.Errorf("parse ffprobe json: %w", err)
	}
	info := avStreamsInfo{}
	for _, s := range parsed.Streams {
		switch s.CodecType {
		case "video":
			if info.VideoCodec == "" {
				info.VideoCodec = s.CodecName
			}
		case "audio":
			info.AudioCodecs = append(info.AudioCodecs, s.CodecName)
		}
	}
	return info, nil
}

func isWebMCompatibleVideoCodec(codec string) bool {
	switch codec {
	case "av1", "vp8", "vp9":
		return true
	}
	return false
}

func isWebMCompatibleAudioCodec(codec string) bool {
	switch codec {
	case "opus", "vorbis":
		return true
	}
	return false
}

func chooseAudioMap(audioCodecs []string) (string, error) {
	if len(audioCodecs) == 0 {
		return "0:a?", nil
	}

	allCompatible := true
	for _, c := range audioCodecs {
		if !isWebMCompatibleAudioCodec(c) {
			allCompatible = false
			break
		}
	}
	if allCompatible {
		return "0:a", nil
	}

	if isWebMCompatibleAudioCodec(audioCodecs[0]) {
		return "0:a:0", nil
	}

	return "", fmt.Errorf("aucune piste audio compatible WebM (codecs : %v)", audioCodecs)
}
