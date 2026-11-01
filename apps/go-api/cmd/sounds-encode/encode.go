package main

// encode.go — le parcours de l'arbre des sons : trouver les WAV, encoder ceux dont l'Opus est
// absent ou périmé, contrôler, retirer les sources. L'encodage lui-même (recette ffmpeg) vit
// dans internal/media ; ici on ne décide que de QUOI encoder et de ce qu'on fait ensuite.

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/media"
)

// encoder est l'encodeur de sons ; les tests le remplacent.
type encoder interface {
	Encode(ctx context.Context, src, dst string) error
	// Check échoue si dst ne se décode pas en exactement les échantillons de src.
	Check(ctx context.Context, src, dst string) error
}

// mediaEncoder est l'encodeur réel (ffmpeg, via internal/media).
type mediaEncoder struct{}

func (mediaEncoder) Encode(ctx context.Context, src, dst string) error {
	return media.EncodeSound(ctx, src, dst)
}

func (mediaEncoder) Check(ctx context.Context, src, dst string) error {
	return media.CheckSameFrames(ctx, src, dst)
}

// options : ce que la ligne de commande demande.
type options struct {
	Src           string // racine des sons à parcourir
	Dst           string // racine des fichiers livrés (même arborescence)
	Check         bool   // vérifier chaque son, y compris ceux déjà à jour
	RemoveSources bool   // retirer chaque WAV dont l'Opus est en place (et contrôlé si Check)
}

// report : le bilan d'un passage.
type report struct {
	Encoded, Skipped, Checked, Removed int
	Failures                           []string
}

// run parcourt opts.Src, traite chaque WAV et rend le bilan. Une erreur de parcours est
// rendue telle quelle ; les échecs par fichier sont journalisés, comptés dans le bilan, et
// rendus ensemble à la fin (un fichier en échec n'arrête pas les autres).
func run(ctx context.Context, opts options, enc encoder, log *slog.Logger) (report, error) {
	var rep report
	err := filepath.WalkDir(opts.Src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(path), media.SoundSourceExt) {
			return nil
		}
		rel, err := filepath.Rel(opts.Src, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(opts.Dst, strings.TrimSuffix(rel, filepath.Ext(rel))+media.SoundExt)
		if err := processOne(ctx, opts, enc, log, path, dst, &rep); err != nil {
			log.ErrorContext(ctx, "sounds-encode: échec", "src", path, "err", err)
			rep.Failures = append(rep.Failures, path)
		}
		return nil
	})
	if err != nil {
		return rep, fmt.Errorf("parcours de %s : %w", opts.Src, err)
	}
	if len(rep.Failures) > 0 {
		return rep, fmt.Errorf("%d son(s) en échec", len(rep.Failures))
	}
	return rep, nil
}

// processOne encode (si nécessaire), contrôle (si demandé) puis retire (si demandé) un WAV.
func processOne(ctx context.Context, opts options, enc encoder, log *slog.Logger, src, dst string, rep *report) error {
	fresh, err := upToDate(src, dst)
	if err != nil {
		return err
	}
	if fresh {
		rep.Skipped++
	} else {
		if err := enc.Encode(ctx, src, dst); err != nil {
			return err
		}
		rep.Encoded++
		log.InfoContext(ctx, "sounds-encode: encodé", "src", src, "dst", dst)
	}
	if opts.Check {
		if err := enc.Check(ctx, src, dst); err != nil {
			return err
		}
		rep.Checked++
	}
	if opts.RemoveSources {
		if err := os.Remove(src); err != nil {
			return err
		}
		rep.Removed++
	}
	return nil
}

// upToDate dit si dst existe et n'est pas plus ancien que src. Un dst absent n'est pas une
// erreur ; toute autre erreur de lecture l'est.
func upToDate(src, dst string) (bool, error) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return false, err
	}
	dstInfo, err := os.Stat(dst)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !dstInfo.ModTime().Before(srcInfo.ModTime()), nil
}
