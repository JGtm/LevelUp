// sounds-encode convertit les sons WAV du rejeu en Opus 96 kbit/s (conteneur .ogg), même
// arborescence. ffmpeg est requis (résolu par internal/media). Idempotent : un .ogg au moins
// aussi récent que son .wav est sauté.
//
// Usage : go run ./cmd/sounds-encode [-src static/sounds] [-dst <src>] [-check] [-remove-sources]
//
//	-check            après encodage (et pour les sons déjà à jour), vérifie que le .ogg se
//	                  décode en exactement les échantillons du .wav
//	-remove-sources   supprime chaque .wav dont le .ogg est en place (et contrôlé si -check)
//
// Code de sortie 1 si un son a échoué (chaque échec est journalisé).
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"

	"levelup/go-api/internal/domain/title"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	opts, err := parseOptions(os.Args[1:])
	if err != nil {
		log.ErrorContext(ctx, "sounds-encode: options", "err", err)
		os.Exit(2)
	}
	rep, err := run(ctx, opts, mediaEncoder{}, log)
	log.InfoContext(ctx, "sounds-encode: bilan",
		"encodes", rep.Encoded, "deja_a_jour", rep.Skipped, "controles", rep.Checked,
		"sources_retirees", rep.Removed, "echecs", len(rep.Failures))
	if err != nil {
		log.ErrorContext(ctx, "sounds-encode: terminé en erreur", "err", err)
		os.Exit(1)
	}
}

// parseOptions lit la ligne de commande ; -src vaut static/sounds du dépôt par défaut, -dst vaut -src.
func parseOptions(args []string) (options, error) {
	fs := flag.NewFlagSet("sounds-encode", flag.ContinueOnError)
	src := fs.String("src", "", "racine des sons à parcourir (défaut : static/sounds du dépôt)")
	dst := fs.String("dst", "", "racine des fichiers livrés (défaut : -src)")
	check := fs.Bool("check", false, "vérifier que chaque .ogg se décode en exactement les échantillons de son .wav")
	remove := fs.Bool("remove-sources", false, "supprimer chaque .wav dont le .ogg est en place")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if *src == "" {
		root, err := title.FindRepoRoot()
		if err != nil {
			return options{}, err
		}
		*src = filepath.Join(root, "static", "sounds")
	}
	if *dst == "" {
		*dst = *src
	}
	return options{Src: *src, Dst: *dst, Check: *check, RemoveSources: *remove}, nil
}
