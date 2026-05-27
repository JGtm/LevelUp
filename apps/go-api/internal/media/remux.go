package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
)

// StreamRemuxAsWebM lance ffmpeg en sous-processus pour remuxer un fichier
// vidéo non-web-natif (MKV, AVI) vers WebM sans réencodage (-c copy) et
// streame le résultat dans w.
//
// Préconditions :
//   - Le codec vidéo doit être AV1, VP8 ou VP9 (compatible WebM)
//   - Au moins une piste audio doit être Opus ou Vorbis
//   - ffmpeg + ffprobe doivent être dans le PATH
//
// Stratégie pistes audio (décidée via ffprobe pre-flight) :
//   - Toutes les pistes Opus/Vorbis → -map 0:a (toutes copiées)
//   - Première piste Opus/Vorbis mais pas toutes → -map 0:a:0 (première seule)
//   - Aucune piste compatible → erreur
//
// Le caller est responsable du Content-Type (video/webm) et de la gestion du
// status code HTTP. ServeFile-style Range requests ne sont pas supportés sur
// le flux remuxé.
func StreamRemuxAsWebM(ctx context.Context, absPath string, w io.Writer) error {
	info, err := probeAVStreams(ctx, absPath)
	if err != nil {
		return fmt.Errorf("ffprobe: %w", err)
	}

	if !isWebMCompatibleVideoCodec(info.VideoCodec) {
		return fmt.Errorf("video codec %q non compatible WebM (attendu : av1, vp8, vp9)", info.VideoCodec)
	}

	audioMap, err := chooseAudioMap(info.AudioCodecs)
	if err != nil {
		return err
	}

	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-i", absPath,
		"-map", "0:v:0",
		"-map", audioMap,
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
			"audio_map", audioMap,
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
