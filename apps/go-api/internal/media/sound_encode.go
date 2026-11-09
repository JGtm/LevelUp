package media

// sound_encode.go — l'encodage des sons livrés du rejeu : WAV PCM -> Opus en conteneur Ogg.
//
// Source unique de la recette (cmd/sounds-encode et la livraison des armes de cmd/weapon-sounds
// l'appellent toutes deux) : Opus 96 kbit/s, conteneur .ogg, nombre de canaux de la source
// conservé. Le pré-décalage (pre-skip) du codec est absorbé par le décodeur et la fin est
// rognée par la position granulaire du conteneur : un son décodé a EXACTEMENT les échantillons
// de sa source, ce qui garde les boucles de moteur sans raccord. CheckSameFrames le vérifie sur
// pièces. Le décodage de contrôle passe par ffmpeg (le moteur du navigateur n'est pas mesuré).

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Extensions des sons : la source déposée et le fichier livré.
const (
	SoundSourceExt = ".wav"
	SoundExt       = ".ogg"
)

// soundOpusBitrate : le débit cible de l'encodage (décision D-6 du chantier Desktop).
const soundOpusBitrate = "96k"

// EncodeSound encode le WAV src en Opus/Ogg dans dst. L'écriture est atomique : ffmpeg écrit
// dst+".part", renommé à la fin, jamais un .ogg tronqué sous son nom définitif. L'encodage est
// reproductible (métadonnées retirées, drapeaux bitexact : le numéro de série du flux Ogg est
// fixe).
func EncodeSound(ctx context.Context, src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("encodage %s : %w", src, err)
	}
	part := dst + ".part"
	stderr := &bytes.Buffer{}
	err := RunFFmpeg(ctx, FFmpegJob{Args: soundEncodeArgs(src, part), Stderr: stderr})
	if err != nil {
		_ = os.Remove(part) // reste d'un encodage avorté ; la cause est rendue ci-dessous
		return fmt.Errorf("encodage %s : %w : %s", src, err, strings.TrimSpace(stderr.String()))
	}
	if err := os.Rename(part, dst); err != nil {
		_ = os.Remove(part)
		return fmt.Errorf("encodage %s : %w", src, err)
	}
	return nil
}

// soundEncodeArgs : les arguments ffmpeg de l'encodage d'un son.
func soundEncodeArgs(src, out string) []string {
	return ffmpegQuietArgs("-y",
		"-i", src,
		"-vn", "-map_metadata", "-1",
		"-c:a", "libopus", "-b:a", soundOpusBitrate,
		"-fflags", "+bitexact", "-flags:a", "+bitexact",
		"-f", "ogg", out)
}

// SoundFrames décrit un son décodé : cadence, canaux et nombre d'échantillons par canal.
type SoundFrames struct {
	SampleRate int
	Channels   int
	Frames     int64
}

// DecodedSoundFrames décode le fichier (WAV ou Ogg) en PCM 16 bits à sa cadence et ses canaux
// natifs, et compte les échantillons obtenus par canal.
func DecodedSoundFrames(ctx context.Context, path string) (SoundFrames, error) {
	rate, channels, err := probeSoundFormat(ctx, path)
	if err != nil {
		return SoundFrames{}, err
	}
	counter := &byteCounter{}
	stderr := &bytes.Buffer{}
	job := FFmpegJob{
		Args:   ffmpegQuietArgs("-i", path, "-vn", "-f", "s16le", "pipe:1"),
		Stdout: counter,
		Stderr: stderr,
	}
	if err := RunFFmpeg(ctx, job); err != nil {
		return SoundFrames{}, fmt.Errorf("décodage %s : %w : %s", path, err, strings.TrimSpace(stderr.String()))
	}
	return SoundFrames{SampleRate: rate, Channels: channels, Frames: counter.n / int64(2*channels)}, nil
}

// probeSoundFormat lit la cadence et le nombre de canaux du premier flux audio.
func probeSoundFormat(ctx context.Context, path string) (rate, channels int, err error) {
	out, err := RunFFprobe(ctx, "-v", "error", "-select_streams", "a:0",
		"-show_entries", "stream=sample_rate,channels", "-of", "csv=p=0", path)
	if err != nil {
		return 0, 0, fmt.Errorf("sonde %s : %w", path, err)
	}
	fields := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("sonde %s : réponse inattendue %q", path, strings.TrimSpace(string(out)))
	}
	if rate, err = strconv.Atoi(fields[0]); err != nil {
		return 0, 0, fmt.Errorf("sonde %s : cadence %q : %w", path, fields[0], err)
	}
	if channels, err = strconv.Atoi(fields[1]); err != nil || channels <= 0 {
		return 0, 0, fmt.Errorf("sonde %s : canaux %q illisibles", path, fields[1])
	}
	return rate, channels, nil
}

// CheckSameFrames échoue si le son encodé ne se décode pas en exactement les échantillons de
// sa source (même cadence, mêmes canaux, même nombre d'échantillons par canal).
func CheckSameFrames(ctx context.Context, src, encoded string) error {
	want, err := DecodedSoundFrames(ctx, src)
	if err != nil {
		return err
	}
	got, err := DecodedSoundFrames(ctx, encoded)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("%s : décodé %+v, source %s : %+v", encoded, got, src, want)
	}
	return nil
}

// byteCounter compte les octets écrits sans les garder.
type byteCounter struct{ n int64 }

var _ io.Writer = (*byteCounter)(nil)

func (b *byteCounter) Write(p []byte) (int, error) {
	b.n += int64(len(p))
	return len(p), nil
}
