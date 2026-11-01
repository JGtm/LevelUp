package main

// cmd_film.go — sous-commandes `levelup film recompress` et `levelup film archive-move` : l'archive
// des films vue de l'exploitant (lot L14, plan Desktop ; ADR 0045 D-12, decision D-18).
//
// DEUX GESTES QUE L'UTILISATEUR LANCE LUI-MEME, JAMAIS LE PROGRAMME. Une recompression ou un
// deplacement touche des dizaines de Go de donnees irremplacables (un film expire cote serveur
// Halo et ne se retelecharge jamais) : aucun cron, aucune tache de fond, aucun demarrage
// n'appelle ces fonctions (garde-rail `archlint/no_auto_film_recompress_test.go`). Serveur arrete.
//
// Usage :
//
//	levelup film recompress --dry-run            # chiffre le gain, n'ecrit rien
//	levelup film recompress                      # recompresse en zstd-3 les chunks bruts et zlib
//	levelup film recompress --films 000d5950     # un film (ou plusieurs, separes par des virgules)
//	levelup film archive-move --to D:\Films      # range l'archive ailleurs, puis enregistre le dossier
//	levelup film archive-move --default          # la ramene sous le cache applicatif

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/JGtm/LevelUp/libs/film/filmcache"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	settings_platform "levelup/go-api/internal/platform/settings"
)

// Noms des sous-commandes de `levelup film`.
const (
	subcmdFilmRecompress  = "recompress"
	subcmdFilmArchiveMove = "archive-move"
)

const usageFilm = "usage: levelup film recompress [--dry-run] [--cache DIR] [--films ID,ID] | " +
	"levelup film archive-move (--to DIR | --default) [--resume]"

// runFilm route `film <sous-commande>`.
func runFilm(cfg *config.AppConfig, args []string) error {
	return runFilmTo(cfg, args, os.Stdout)
}

func runFilmTo(cfg *config.AppConfig, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(usageFilm)
	}
	switch args[0] {
	case subcmdFilmRecompress:
		return filmRecompress(cfg, args[1:], out)
	case subcmdFilmArchiveMove:
		return filmArchiveMove(cfg, args[1:], out)
	default:
		return fmt.Errorf("sous-commande film inconnue: %q (recompress | archive-move)", args[0])
	}
}

// --- recompress -------------------------------------------------------------------------------

type filmRecompressOptions struct {
	dryRun   bool
	cacheDir string
	films    string
}

func filmRecompress(cfg *config.AppConfig, args []string, out io.Writer) error {
	var o filmRecompressOptions
	fs := flag.NewFlagSet("film recompress", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.BoolVar(&o.dryRun, "dry-run", false, "lit les chunks et chiffre le gain en memoire, n'ecrit rien")
	fs.StringVar(&o.cacheDir, "cache", "", "racine de l'archive des films (defaut : l'archive du depot)")
	fs.StringVar(&o.films, "films", "", "courts de film (8 hex) a traiter, separes par des virgules ; defaut : toute l'archive")
	if err := fs.Parse(args); err != nil {
		return err
	}
	// ARRET DOUX : un Ctrl-C finit le chunk en cours puis s'arrete ; relancer reprend (les chunks
	// deja en zstd sont sautes).
	ctx, stop := contexteDArret()
	defer stop()
	root := resoudreCacheFilms(cfg, o.cacheDir)
	ids, err := filmsARecompresser(root, o.films)
	if err != nil {
		return err
	}
	s := &sortie{w: out}
	mode := "recompression"
	if o.dryRun {
		mode = "simulation (rien n'est ecrit)"
	}
	s.printf("film recompress : %d films, archive %s — %s\n", len(ids), root, mode)
	total, erreurs := recompresserLesFilms(ctx, root, ids, o.dryRun, s)
	s.printf("%d films traites : %d chunks convertis, %d deja en zstd, %d sans gain — %s -> %s (gain %s)\n",
		len(ids)-erreurs, total.Convertis, total.DejaZstd, total.SansGain,
		formaterOctets(total.OctetsAvant), formaterOctets(total.OctetsApres), formaterOctets(total.Gain()))
	if cause := causeDArret(ctx); cause != "" {
		s.printf("passe interrompue (%s) : relancer la meme commande reprend les films restants.\n", cause)
	}
	if s.err != nil {
		return s.err
	}
	if erreurs > 0 {
		return fmt.Errorf("%d films en erreur (voir le journal) : aucun chunk abime n'a ete modifie", erreurs)
	}
	return nil
}

// filmsARecompresser rend les films a traiter : ceux de `--films`, sinon toute l'archive.
func filmsARecompresser(root, liste string) ([]string, error) {
	if strings.TrimSpace(liste) != "" {
		var ids []string
		for _, id := range strings.Split(liste, ",") {
			if id = strings.TrimSpace(id); id != "" {
				ids = append(ids, id)
			}
		}
		return ids, nil
	}
	return filmcache.ListShortIDs(root)
}

// recompresserLesFilms traite les films UN A LA FOIS, journalise chacun, et rend le bilan cumule
// et le nombre de films en erreur. Un film en erreur n'arrete pas la passe.
func recompresserLesFilms(ctx context.Context, root string, ids []string, dryRun bool, s *sortie) (filmcache.RapportRecompression, int) {
	var total filmcache.RapportRecompression
	erreurs := 0
	debut := time.Now()
	for i, id := range ids {
		if ctx.Err() != nil {
			break
		}
		rap, err := filmcache.Recompresser(ctx, root, id, dryRun)
		total.Ajouter(rap)
		if err != nil {
			erreurs++
			slog.ErrorContext(ctx, "film recompress: film en erreur", "short_id", id, "err", err)
			continue
		}
		slog.InfoContext(ctx, "film recompress: film traite", "short_id", id, "dry_run", dryRun,
			"chunks", rap.Chunks, "convertis", rap.Convertis, "deja_zstd", rap.DejaZstd,
			"sans_gain", rap.SansGain, "absents", rap.Absents,
			"octets_avant", rap.OctetsAvant, "octets_apres", rap.OctetsApres, "gain", rap.Gain(),
			"manifeste_corrige", rap.ManifesteCorrige)
		if (i+1)%25 == 0 {
			s.printf("  [%d/%d] gain cumule %s — %s\n", i+1, len(ids),
				formaterOctets(total.Gain()), time.Since(debut).Round(time.Second))
		}
	}
	return total, erreurs
}

// formaterOctets ecrit une taille en unites binaires lisibles.
func formaterOctets(n int64) string {
	const unite = 1024
	if n < unite && n > -unite {
		return fmt.Sprintf("%d o", n)
	}
	val, suffixes := float64(n), []string{"Kio", "Mio", "Gio", "Tio"}
	i := -1
	for ; (val >= unite || val <= -unite) && i < len(suffixes)-1; i++ {
		val /= unite
	}
	return fmt.Sprintf("%.1f %s", val, suffixes[i])
}

// --- archive-move -----------------------------------------------------------------------------

type filmArchiveMoveOptions struct {
	to     string
	toDef  bool
	resume bool
}

func filmArchiveMove(cfg *config.AppConfig, args []string, out io.Writer) error {
	var o filmArchiveMoveOptions
	fs := flag.NewFlagSet("film archive-move", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.StringVar(&o.to, "to", "", "nouveau dossier de l'archive (chemin absolu, vide ou inexistant)")
	fs.BoolVar(&o.toDef, "default", false, "ramener l'archive sous le cache applicatif (emplacement par defaut)")
	fs.BoolVar(&o.resume, "resume", false, "reprendre un deplacement interrompu (la cible n'est alors pas vide)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	pr := cfg.PathResolver()
	cible := strings.TrimSpace(o.to)
	switch {
	case o.toDef && cible != "":
		return errors.New("film archive-move: --to et --default s'excluent")
	case o.toDef:
		cible = pr.Roots().DefaultFilmArchiveDir()
	case cible == "":
		return errors.New("film archive-move exige --to DIR ou --default")
	}
	source := pr.FilmArchiveDir()
	if titlePkg.SameRoot(source, cible) {
		return fmt.Errorf("film archive-move: l'archive est deja a %s", cible)
	}
	ctx, stop := contexteDArret()
	defer stop()
	s := &sortie{w: out}
	s.printf("film archive-move : %s -> %s (serveur arrete)\n", source, cible)
	debut := time.Now()
	n, err := filmcache.Deplacer(ctx, source, cible, o.resume)
	if err != nil {
		slog.ErrorContext(ctx, "film archive-move: deplacement interrompu", "source", source, "cible", cible, "deplaces", n, "err", err)
		return fmt.Errorf("deplacement interrompu apres %d films (relancer avec --resume reprend les films restants): %w", n, err)
	}
	// Le dossier n'est enregistre qu'une fois TOUS les films rangés : un reglage qui pointerait une
	// archive a moitie remplie ferait croire a des films perdus.
	if err := enregistrerDossierArchive(cfg, o, cible); err != nil {
		return fmt.Errorf("films deplaces mais reglage non enregistre (poser film_archive_dir=%q dans app_settings.json): %w", cible, err)
	}
	s.printf("%d films deplaces en %s ; film_archive_dir enregistre (pris en compte au prochain demarrage)\n",
		n, time.Since(debut).Round(time.Second))
	return s.err
}

// enregistrerDossierArchive ecrit film_archive_dir (vide pour l'emplacement par defaut).
func enregistrerDossierArchive(cfg *config.AppConfig, o filmArchiveMoveOptions, cible string) error {
	store := settings_platform.NewStore(cfg.AppSettingsPath)
	cur, err := store.Load()
	if err != nil {
		return err
	}
	cur.FilmArchiveDir = cible
	if o.toDef {
		cur.FilmArchiveDir = ""
	}
	return store.Save(cur)
}
