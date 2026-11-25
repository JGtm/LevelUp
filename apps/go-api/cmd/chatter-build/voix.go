package main

// voix.go — DU MEDIA WWISE AU WAV STEREO, AVANT `sounds-encode`.
//
// Les voix sont des médias Wwise (Vorbis). Le dépôt n'a aucun décodeur Wwise : `vgmstream-cli` (celui des sons
// d'armes, chemin passé en option) les décode en WAV, mono pour une voix. Les sons livrés du rejeu sont tous
// en stéréo 48 kHz : le WAV est donc mis en stéréo ici (le canal est dupliqué, ce que l'encodeur Opus paie
// moins cher qu'un mono au même débit), puis `go run ./cmd/sounds-encode -src <wav> -dst <ogg>` fait l'Opus,
// avec la recette commune de `internal/media`.

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/media"
)

// decodeurWem décode un média Wwise en WAV ; les tests le remplacent.
type decodeurWem interface {
	Decoder(ctx context.Context, wem, wav string) error
}

// vgmstream est le décodeur réel.
type vgmstream struct{ Exe string }

func (v vgmstream) Decoder(ctx context.Context, wem, wav string) error {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, v.Exe, "-o", wav, wem)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vgmstream %s : %w : %s", filepath.Base(wem), err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// mettreEnStereo écrit dans dst le WAV src sur deux canaux.
func mettreEnStereo(ctx context.Context, src, dst string) error {
	var stderr bytes.Buffer
	args := []string{"-hide_banner", "-loglevel", "error", "-y", "-i", src, "-vn", "-map_metadata", "-1",
		"-ac", "2", "-c:a", "pcm_s16le", dst}
	if err := media.RunFFmpeg(ctx, media.FFmpegJob{Args: args, Stderr: &stderr}); err != nil {
		return fmt.Errorf("stéréo %s : %w : %s", filepath.Base(src), err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// convertirVoix décode tous les `.wem` de `src` en WAV stéréo dans `dst` (même nom, extension `.wav`). Un média
// qui ne se décode pas est journalisé et compté ; la fonction rend une erreur à la fin s'il y en a.
func convertirVoix(ctx context.Context, dec decodeurWem, stereo func(context.Context, string, string) error, src, dst string, log *slog.Logger) (int, error) {
	ents, err := os.ReadDir(src)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return 0, err
	}
	var noms []string
	for _, e := range ents {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".wem") {
			noms = append(noms, e.Name())
		}
	}
	sort.Strings(noms)
	tmp, err := os.MkdirTemp("", "chatter-build-")
	if err != nil {
		return 0, err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	faits, echecs := 0, 0
	for _, n := range noms {
		base := strings.TrimSuffix(n, ".wem")
		mono := filepath.Join(tmp, base+".wav")
		if err := dec.Decoder(ctx, filepath.Join(src, n), mono); err != nil {
			log.Error("chatter-build: décodage en échec", "media", base, "err", err)
			echecs++
			continue
		}
		if err := stereo(ctx, mono, filepath.Join(dst, base+".wav")); err != nil {
			log.Error("chatter-build: mise en stéréo en échec", "media", base, "err", err)
			echecs++
			continue
		}
		_ = os.Remove(mono)
		faits++
	}
	if echecs > 0 {
		return faits, fmt.Errorf("%d média(s) en échec sur %d", echecs, len(noms))
	}
	return faits, nil
}
