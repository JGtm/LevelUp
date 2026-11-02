package media

// h264_encoder.go : choix de l'encodeur vidéo H.264 du transcodage HLS.
//
// Un build ffmpeg sous licence LGPL (celui que livre l'application de bureau) n'embarque pas
// libx264 (GPL). L'encodeur est donc choisi parmi ceux que le binaire résolu sait produire, par
// ordre de préférence, puis ESSAYÉ sur une image : un encodeur listé mais inutilisable sur le
// poste (Media Foundation absent d'une édition N de Windows, par exemple) est écarté et le
// suivant est tenté. Aucun encodeur utilisable = ErrNoH264Encoder, et le transcodage est refusé.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
)

// H264Encoder est le nom ffmpeg d'un encodeur H.264.
type H264Encoder string

const (
	// EncoderLibx264 : encodeur de référence (builds GPL). Rendu à qualité constante (CRF).
	EncoderLibx264 H264Encoder = "libx264"
	// EncoderMediaFoundation : encodeur Media Foundation de Windows, présent dans le build LGPL
	// livré. Pas de preset ni de CRF : débit cible.
	EncoderMediaFoundation H264Encoder = "h264_mf"
	// EncoderOpenH264 : OpenH264 de Cisco, présent dans le build LGPL livré. Débit cible.
	EncoderOpenH264 H264Encoder = "libopenh264"
)

// h264Preference est l'ordre d'essai : la qualité par CRF d'abord, puis l'encodeur du système,
// puis OpenH264.
var h264Preference = []H264Encoder{EncoderLibx264, EncoderMediaFoundation, EncoderOpenH264}

// ErrNoH264Encoder : le ffmpeg résolu n'a aucun encodeur H.264 utilisable parmi h264Preference.
var ErrNoH264Encoder = errors.New("aucun encodeur H.264 utilisable (libx264, h264_mf, libopenh264)")

const (
	// h264FallbackBitrate est le débit cible des encodeurs sans CRF : un clip de jeu 1080p60 y
	// reste net sans produire des segments démesurés.
	h264FallbackBitrate = "12M"
	// h264FallbackGOP est la longueur de groupe d'images des encodeurs sans CRF (images) : la
	// valeur par défaut de ffmpeg (12) multiplierait les images-clés et alourdirait le flux.
	h264FallbackGOP = 240
	// h264ProbeInput : source synthétique d'une image pour essayer un encodeur.
	h264ProbeInput = "color=c=black:s=640x360:r=30:d=0.2"
)

// H264Candidates rend, dans l'ordre de préférence, les encodeurs H.264 présents dans l'ensemble
// des encodeurs listés par ffmpeg. Pur.
func H264Candidates(listed map[string]bool) []H264Encoder {
	var out []H264Encoder
	for _, enc := range h264Preference {
		if listed[string(enc)] {
			out = append(out, enc)
		}
	}
	return out
}

// h264VideoArgs rend les arguments de l'encodage vidéo pour l'encodeur donné ; segDur est la
// durée visée d'un segment HLS en secondes. Pur.
func h264VideoArgs(enc H264Encoder, segDur int) []string {
	switch enc {
	case EncoderMediaFoundation:
		return append([]string{"-c:v", string(enc)}, fallbackRateArgs(segDur)...)
	case EncoderOpenH264:
		return append([]string{"-c:v", string(enc), "-rc_mode", "bitrate"}, fallbackRateArgs(segDur)...)
	default: // EncoderLibx264
		return []string{"-c:v", string(EncoderLibx264), "-preset", "veryfast", "-crf", "20", "-pix_fmt", "yuv420p"}
	}
}

// fallbackRateArgs : débit cible, groupe d'images long et images-clés forcées à la durée d'un
// segment (une coupure de segment HLS ne tombe que sur une image-clé).
func fallbackRateArgs(segDur int) []string {
	return []string{
		"-b:v", h264FallbackBitrate,
		"-g", strconv.Itoa(h264FallbackGOP),
		"-force_key_frames", "expr:gte(t,n_forced*" + strconv.Itoa(segDur) + ")",
		"-pix_fmt", "yuv420p",
	}
}

// h264ProbeArgs : encode une image synthétique vers la sortie nulle avec l'encodeur donné.
func h264ProbeArgs(enc H264Encoder) []string {
	args := ffmpegQuietArgs("-f", "lavfi", "-i", h264ProbeInput, "-frames:v", "1")
	args = append(args, h264VideoArgs(enc, defaultSegmentDuration)...)
	return append(args, "-f", "null", "-")
}

// chooseH264 rend le premier candidat que probe accepte ; les refus sont journalisés. Sans
// candidat accepté, ErrNoH264Encoder. Logique pure à la sonde près (injectée par les tests).
func chooseH264(ctx context.Context, candidates []H264Encoder, probe func(context.Context, H264Encoder) error) (H264Encoder, error) {
	for _, enc := range candidates {
		err := probe(ctx, enc)
		if err == nil {
			return enc, nil
		}
		slog.WarnContext(ctx, "media: encodeur H.264 listé mais inutilisable, suivant essayé",
			"encoder", string(enc), "err", err)
	}
	return "", ErrNoH264Encoder
}

// probeH264 essaie réellement l'encodeur sur une image.
func probeH264(ctx context.Context, enc H264Encoder) error {
	if err := RunFFmpeg(ctx, FFmpegJob{Args: h264ProbeArgs(enc)}); err != nil {
		return fmt.Errorf("essai d'encodage: %w", err)
	}
	return nil
}

// SelectedH264Encoder rend l'encodeur H.264 retenu pour le ffmpeg résolu. Le choix (liste des
// encodeurs, essai sur une image, journal « encodeur retenu ») est fait une fois par chemin de
// binaire puis gardé ; un échec n'est pas gardé. ErrNoH264Encoder (enveloppée ou non) quand aucun
// encodeur n'est utilisable.
func SelectedH264Encoder(ctx context.Context) (H264Encoder, error) {
	path, err := ResolveFFmpeg()
	if err != nil {
		return "", err
	}
	capabilities.mu.Lock()
	if capabilities.h264 != "" && capabilities.h264Path == path {
		enc := capabilities.h264
		capabilities.mu.Unlock()
		return enc, nil
	}
	capabilities.mu.Unlock()

	listed, err := AvailableEncoders(ctx)
	if err != nil {
		return "", err
	}
	enc, err := chooseH264(ctx, H264Candidates(listed), probeH264)
	if err != nil {
		return "", err
	}
	slog.InfoContext(ctx, "media: encodeur H.264 retenu", "encoder", string(enc), "ffmpeg_path", path)
	capabilities.mu.Lock()
	capabilities.h264, capabilities.h264Path = enc, path
	capabilities.mu.Unlock()
	return enc, nil
}
