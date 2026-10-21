package main

// cmd_backfill_match_summaries.go — sous-commande `levelup backfill-match-summaries`.
//
// ELLE ÉCRIT LES RÉSUMÉS PAR MATCH DU JOURNAL DES MORTS (`match_kill_heads`, `match_kill_pairs`,
// `match_assist_pairs`, ADR 0039) POUR LES PASSES QUI N'EN ONT PAS : les passes écrites avant
// les résumés (migrations historiques, écrivains hors ligne) et celles dont la tête est à une
// ancienne révision de la règle (`killsummary.Rev`). Le fil de l'eau écrit les résumés AVEC la
// passe ; cette commande rattrape le parc existant et, après une montée de révision, le parc
// entier (séquence de release : rattrapage sur les bases locales, puis copie vers le serveur).
//
// # ELLE NE DÉCODE AUCUN FILM
//
// Elle relit la passe COURANTE du match par la vue `match_kill_events_latest` (jamais la table,
// ADR 0026) et appelle `persist.PersistLatestKillPassSummaries` : même fonction pure, même
// insertion que le fil de l'eau. Une transaction par match ; un échec se compte, se journalise
// et n'arrête pas la passe.
//
// # ELLE EST REPRENABLE
//
// La liste des matchs à rattraper est une anti-jointure (passe courante sans tête à la révision
// courante, `persist.PendingKillSummaryMatches`, le même texte que la jauge du contrôle de
// santé) : interrompre puis relancer reprend où l'on en était, et un second passage ne trouve
// rien à faire. Elle se calcule UNE fois (la fenêtre de la vue du journal porte sur toute la
// table) puis se parcourt par lots, avec un journal de progression par lot.
//
// Usage (SERVEUR ARRÊTÉ — l'ouverture en écriture échoue si la base est tenue par un autre
// processus, y compris pour --dry-run, qui joue les migrations) :
//
//	levelup backfill-match-summaries --title halo_infinite --dry-run   # compte, n'écrit rien
//	levelup backfill-match-summaries --title halo_infinite             # rattrape
//	levelup backfill-match-summaries --title halo_5 --batch 500

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/duckdb"
)

// defaultSummariesBatch : nombre de matchs entre deux lignes de progression.
const defaultSummariesBatch = 250

// summariesDryRunSample : nombre d'identifiants imprimés sous --dry-run.
const summariesDryRunSample = 10

// matchSummariesOptions : les réglages de la passe.
type matchSummariesOptions struct {
	titleSlug string
	dryRun    bool
	batch     int
}

// bilanMatchSummaries : ce que la passe a fait.
type bilanMatchSummaries struct {
	// aRattraper : matchs sans tête à jour au départ.
	aRattraper int
	// ecrits : matchs dont une génération de résumés a été écrite.
	ecrits int
	// disparus : matchs à rattraper au départ mais sans ligne au journal à la relecture.
	disparus int
	// echecs : matchs dont l'écriture a échoué (journalisés, la passe continue).
	echecs int
	// restants : matchs encore à rattraper après la passe (jauge de l'ADR 0039 ; -1 = non mesuré).
	restants int
}

func runBackfillMatchSummaries(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill-match-summaries", flag.ExitOnError)
	o := matchSummariesOptions{}
	fs.StringVar(&o.titleSlug, "title", titlePkg.DefaultSlug, "slug du titre")
	fs.BoolVar(&o.dryRun, "dry-run", false, "compter les matchs à rattraper sans rien écrire")
	fs.IntVar(&o.batch, "batch", defaultSummariesBatch, "matchs entre deux lignes de progression")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if o.batch <= 0 {
		return fmt.Errorf("--batch doit être > 0 (reçu %d)", o.batch)
	}

	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath := pr.SharedDBPath(o.titleSlug)
	if _, err := os.Stat(sharedPath); err != nil {
		return fmt.Errorf("shared_matches introuvable (%s): %w", sharedPath, err)
	}
	handle, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		return erreurOuvertureEcriturePartagee(sharedPath, err)
	}
	defer handle.Close()
	db := handle.SQLDb()

	// Les tables et leurs vues doivent exister AVANT de lire ou d'écrire : cette commande tourne
	// SERVEUR ARRÊTÉ, donc rien n'a joué les migrations pour elle.
	if err := migrerSchemaPartage(db, o.titleSlug); err != nil {
		return err
	}
	debut := time.Now()
	b, err := rattraperResumes(context.Background(), db, o)
	fmt.Printf("résumés par match (%s) : %d à rattraper, %d écrits, %d disparus, %d échecs, %s restant(s) — %s\n",
		o.titleSlug, b.aRattraper, b.ecrits, b.disparus, b.echecs, restantsTexte(b.restants),
		time.Since(debut).Round(time.Second))
	return err
}

// restantsTexte : le nombre de matchs encore à rattraper, ou « non mesuré ».
func restantsTexte(n int) string {
	if n < 0 {
		return "non mesuré"
	}
	return fmt.Sprint(n)
}

// erreurOuvertureEcriturePartagee dit clairement qu'une base tenue par un autre processus
// (le serveur local) empêche la passe : DuckDB n'admet qu'un processus par fichier.
func erreurOuvertureEcriturePartagee(path string, err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "lock") {
		return fmt.Errorf("la base %s est tenue par un autre processus (le serveur tourne-t-il ?) : "+
			"arrêter le serveur puis relancer la commande : %w", path, err)
	}
	return fmt.Errorf("ouverture en écriture de %s impossible : %w", path, err)
}

// rattraperResumes calcule la liste des matchs à rattraper, puis écrit leurs résumés par lots
// (sauf sous --dry-run). Rend une erreur quand au moins un match a échoué.
func rattraperResumes(ctx context.Context, db *sql.DB, o matchSummariesOptions) (bilanMatchSummaries, error) {
	b := bilanMatchSummaries{restants: -1}
	pending, err := persist.PendingKillSummaryMatches(ctx, db)
	if err != nil {
		return b, err
	}
	b.aRattraper = len(pending)
	slog.InfoContext(ctx, "backfill résumés : matchs à rattraper", "title", o.titleSlug,
		"matchs", b.aRattraper, "dry_run", o.dryRun)
	if o.dryRun {
		for i := 0; i < len(pending) && i < summariesDryRunSample; i++ {
			fmt.Printf("  à rattraper : %s\n", pending[i])
		}
		b.restants = b.aRattraper
		return b, nil
	}
	for debut := 0; debut < len(pending); debut += o.batch {
		fin := min(debut+o.batch, len(pending))
		ecrireLotResumes(ctx, db, pending[debut:fin], &b)
		slog.InfoContext(ctx, "backfill résumés : progression", "title", o.titleSlug,
			"traites", fin, "total", len(pending), "ecrits", b.ecrits, "echecs", b.echecs)
	}
	if n, err := persist.CountPendingKillSummaries(ctx, db); err != nil {
		slog.ErrorContext(ctx, "backfill résumés : jauge finale illisible", "title", o.titleSlug, "err", err)
	} else {
		b.restants = n
	}
	if b.echecs > 0 {
		return b, fmt.Errorf("%d match(s) en échec sur %d (voir le journal)", b.echecs, b.aRattraper)
	}
	return b, nil
}

// ecrireLotResumes écrit les résumés d'un lot de matchs, une transaction par match.
func ecrireLotResumes(ctx context.Context, db *sql.DB, lot []string, b *bilanMatchSummaries) {
	for _, id := range lot {
		ecrit, err := persist.PersistLatestKillPassSummaries(ctx, db, id)
		switch {
		case err != nil:
			// Jamais avalé : l'échec d'UN match n'arrête pas la passe, mais il se voit et se compte.
			slog.ErrorContext(ctx, "backfill résumés : écriture en échec", "match_id", id, "err", err)
			b.echecs++
		case !ecrit:
			b.disparus++
		default:
			b.ecrits++
		}
	}
}
