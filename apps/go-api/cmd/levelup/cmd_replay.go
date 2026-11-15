package main

// cmd_replay.go — sous-commande `levelup replay compress` : convertit en place un parc d'artefacts de
// rejeu bruts (`{short8}.json`) en artefacts compresses (`{short8}.json.gz`).
//
// UN GESTE QUE L'UTILISATEUR LANCE LUI-MEME, JAMAIS LE PROGRAMME : la conversion reecrit tout le dossier
// d'artefacts. Aucun cron, aucune tache de fond, aucun demarrage n'appelle `replaybuild.CompressDir`
// (garde-rail `archlint/no_auto_replay_compress_test.go`). Elle supporte un serveur actif : la pose d'un
// `.json.gz` est exclusive et atomique (un `.json.gz` existant, par exemple pose par le serveur pendant la
// conversion, n'est JAMAIS remplace) et un lecteur ne voit jamais un fichier a moitie ecrit.
//
// Chaque artefact est relu et verifie (contenu decompresse identique, octet pour octet) AVANT que le `.json`
// soit supprime ; la commande est idempotente (un artefact deja compresse n'est pas touche) et chiffre ce
// qu'elle a fait. Un match aux deux formes se juge sur le contenu : identique, le `.json` est supprime ;
// different, rien n'est supprime. Un `.json` qui ne se supprime pas est « a relancer » (code de sortie non
// nul) : la relance termine. Le contenu des artefacts ne change pas d'un octet.
//
// Usage :
//
//	levelup replay compress --dry-run            # chiffre le gain, n'ecrit rien
//	levelup replay compress                      # convertit le parc du titre par defaut
//	levelup replay compress --title halo_5       # un autre titre
//	levelup replay compress --dir D:\copie\parc  # un dossier d'artefacts explicite

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replaybuild"
)

const (
	subcmdReplayCompress = "compress"
	usageReplay          = "usage: levelup replay compress [--dry-run] [--title SLUG] [--dir DIR]"
)

// runReplay route `replay <sous-commande>`.
func runReplay(cfg *config.AppConfig, args []string) error {
	return runReplayTo(cfg, args, os.Stdout)
}

func runReplayTo(cfg *config.AppConfig, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(usageReplay)
	}
	switch args[0] {
	case subcmdReplayCompress:
		return replayCompress(cfg, args[1:], out)
	default:
		return fmt.Errorf("sous-commande replay inconnue: %q (%s)", args[0], subcmdReplayCompress)
	}
}

type replayCompressOptions struct {
	dryRun    bool
	titleSlug string
	dir       string
}

func replayCompress(cfg *config.AppConfig, args []string, out io.Writer) error {
	var o replayCompressOptions
	fs := flag.NewFlagSet("replay compress", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.BoolVar(&o.dryRun, "dry-run", false, "lit les artefacts et chiffre le gain en memoire, n'ecrit rien")
	fs.StringVar(&o.titleSlug, "title", titlePkg.DefaultSlug, "slug du titre")
	fs.StringVar(&o.dir, "dir", "", "dossier d'artefacts explicite (defaut : celui du titre, par PathResolver)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir := o.dir
	if dir == "" {
		dir = cfg.PathResolver().ReplayArtifactsDir(o.titleSlug)
	}
	// ARRET DOUX : un Ctrl-C finit l'artefact en cours puis s'arrete ; relancer reprend (les artefacts deja
	// compresses sont sautes).
	ctx, stop := contexteDArret()
	defer stop()

	s := &sortie{w: out}
	mode := "conversion"
	if o.dryRun {
		mode = "simulation (rien n'est ecrit)"
	}
	s.printf("replay compress : dossier %s — %s\n", dir, mode)
	rep, err := replaybuild.CompressDir(ctx, dir, o.dryRun)
	if err != nil && ctx.Err() == nil {
		return err
	}
	verbe := "convertis"
	if o.dryRun {
		verbe = "a convertir"
	}
	s.printf("%d %s, %d deja compresses, %d doublons identiques resorbes, %d doublons de contenus differents "+
		"(rien n'est supprime, a trancher a la main), %d a relancer (.json non supprime), %d en echec — %s -> %s (gain %s)\n",
		rep.Converted, verbe, rep.AlreadyCompressed, rep.Resolved, rep.Different, rep.ToRetry, rep.Failed,
		formaterOctets(rep.BytesBefore), formaterOctets(rep.BytesAfter), formaterOctets(rep.BytesBefore-rep.BytesAfter))
	if cause := causeDArret(ctx); cause != "" {
		s.printf("passe interrompue (%s) : relancer la meme commande reprend les artefacts restants.\n", cause)
	}
	if s.err != nil {
		return s.err
	}
	if rep.Failed > 0 {
		return fmt.Errorf("%d artefacts en echec (voir le journal) : aucun .json en echec n'a ete supprime", rep.Failed)
	}
	if rep.ToRetry > 0 {
		return fmt.Errorf("%d .json non supprimes (le .json.gz est pose et verifie) : relancer la commande", rep.ToRetry)
	}
	return nil
}
