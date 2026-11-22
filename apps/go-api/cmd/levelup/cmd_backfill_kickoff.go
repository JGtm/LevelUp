package main

// cmd_backfill_kickoff.go — sous-commande `levelup backfill-kickoff` : le rattrapage du coup
// d'envoi, de sa source et du temps de jeu de tous les matchs d'une base partagée (ADR 0050).
//
// C'EST LA MÊME FONCTION QUE LE DÉMARRAGE DU SERVEUR (`kickoff.Backfill`) : la commande sert à
// simuler avant une mise à jour, ou à rattraper une base serveur arrêté. Elle ne décode aucun film :
// le coup d'envoi de source `film` est lu dans les artefacts de rejeu déjà rangés, tel qu'ils le
// publient. Sans `--dry-run`, elle range aussi la révision `timeline.KickoffRev` dans `sync_meta`
// après un passage complet sans échec : le démarrage suivant n'a alors plus rien à faire.
//
// Usage :
//
//	levelup backfill-kickoff --dry-run                      # simulation : les écarts, rien d'écrit
//	levelup backfill-kickoff                                # application (SERVEUR ARRÊTÉ)
//	levelup backfill-kickoff --dry-run --db COPIE.duckdb    # simulation sur une copie de la base
//
// `--dry-run` ouvre la base en LECTURE SEULE, sans migration (une base sans la colonne de source
// est lue comme ne portant aucune source) ; il cohabite avec un serveur en marche. L'application
// ouvre la base en écriture après les migrations : un seul processus écrivain par base.

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/kickoff"
)

// kickoffOptions : les réglages de la commande.
type kickoffOptions struct {
	titleSlug  string
	dryRun     bool
	dbPath     string
	replaysDir string
	lot        int
}

func runBackfillKickoff(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill-kickoff", flag.ExitOnError)
	o := kickoffOptions{}
	fs.StringVar(&o.titleSlug, "title", titlePkg.DefaultSlug, "slug du titre")
	fs.BoolVar(&o.dryRun, "dry-run", false,
		"simulation : tout est lu et resolu, rien n'est ecrit (base en lecture seule, aucune migration)")
	fs.StringVar(&o.dbPath, "db", "", "base partagee a traiter (defaut : celle du titre)")
	fs.StringVar(&o.replaysDir, "replays-dir", "", "dossier des artefacts de rejeu (defaut : celui du titre)")
	fs.IntVar(&o.lot, "lot", kickoff.DefaultLotSize, "matchs par burst d'ecriture")
	if err := fs.Parse(args); err != nil {
		return err
	}
	pr := cfg.PathResolver()
	if o.dbPath == "" {
		o.dbPath = pr.SharedDBPath(o.titleSlug)
	}
	if _, err := os.Stat(o.dbPath); err != nil {
		return fmt.Errorf("base partagee introuvable (%s): %w", o.dbPath, err)
	}
	opts, err := kickoffBackfillOptions(cfg, pr, o)
	if err != nil {
		return err
	}
	db, fermer, err := ouvrirSharedCoupDEnvoi(o.dbPath, o.titleSlug, o.dryRun)
	if err != nil {
		return err
	}
	defer fermer()
	tenu := func(context.Context) (*sql.DB, func(), error) { return db, func() {}, nil }
	acc := kickoff.Access{Read: tenu, Write: tenu}

	ctx := context.Background()
	debut := time.Now()
	rep, err := kickoff.Backfill(ctx, acc, opts)
	fmt.Print(syntheseCoupDEnvoi(rep, o, time.Since(debut).Round(time.Millisecond)))
	if err != nil {
		return err
	}
	if o.dryRun {
		return nil
	}
	if err := kickoff.StoreRevision(ctx, acc); err != nil {
		return err
	}
	fmt.Printf("revision rangee : %s\n", timeline.KickoffRev)
	return nil
}

// kickoffBackfillOptions : le chemin des artefacts n'existe que pour un titre qui déclare le rejeu.
func kickoffBackfillOptions(cfg *config.AppConfig, pr *titlePkg.PathResolver, o kickoffOptions) (kickoff.Options, error) {
	opts := kickoff.Options{TitleSlug: o.titleSlug, DryRun: o.dryRun, LotSize: o.lot}
	caps, err := capabilitesDuTitre(cfg, o.titleSlug)
	if err != nil {
		return opts, err
	}
	if !caps.Has(games.CapFilmReplayArtifact) {
		return opts, nil
	}
	dir := o.replaysDir
	if dir == "" {
		dir = pr.ReplayArtifactsDir(o.titleSlug)
	}
	opts.ArtifactPath = func(matchID string) string {
		return filepath.Join(dir, titlePkg.ReplayArtifactFileName(matchID))
	}
	return opts, nil
}

// ouvrirSharedCoupDEnvoi ouvre la base : en LECTURE pour --dry-run (cohabite avec un serveur), en
// écriture sinon, après les migrations (la commande tourne serveur arrêté, rien ne les a jouées).
func ouvrirSharedCoupDEnvoi(path, slug string, dryRun bool) (*sql.DB, func(), error) {
	if dryRun {
		return duckdb.OpenReadForQuery(path)
	}
	h, err := duckdb.OpenReadWrite(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open shared RW (%s): %w (serveur arrete ?)", path, err)
	}
	if err := migrerSchemaPartage(h.SQLDb(), slug); err != nil {
		h.Close()
		return nil, nil, err
	}
	return h.SQLDb(), func() { h.Close() }, nil
}

// syntheseCoupDEnvoi met le bilan en texte (sortie de commande).
func syntheseCoupDEnvoi(r kickoff.Report, o kickoffOptions, duree time.Duration) string {
	mode := "APPLICATION"
	if o.dryRun {
		mode = "SIMULATION (rien d'ecrit)"
	}
	s := fmt.Sprintf("=== Coup d'envoi — %s — %s — %s ===\n", o.titleSlug, mode, o.dbPath)
	s += fmt.Sprintf("matchs resolus : %d   (rejeux presents : %d, illisibles : %d)\n",
		r.Matches, r.Artifacts, r.UnreadableArtifacts)
	s += fmt.Sprintf("  source film : %d   api : %d   inconnue : %d\n", r.BySource[timeline.KickoffSourceFilm],
		r.BySource[timeline.KickoffSourceAPI], r.BySource[timeline.KickoffSourceUnknown])
	s += "  estimation jouee : " + issues(r.Estimates) + "\n"
	s += fmt.Sprintf("coup d'envoi : %d match(s) changent de valeur (dont %d deviennent inconnus), %d de source\n",
		r.ValueChanged, r.BecameUnknown, r.SourceChanged)
	s += "  ecart nouveau - ancien (ms) : " + distribution(kickoff.Summarize(r.KickoffShiftsMs, 1000), "1 s") + "\n"
	s += fmt.Sprintf("temps de jeu : %d joueur(s) changent, sur %d match(s) (%d sans valeur avant)\n",
		r.PlayersChanged, r.MatchesWithPlayerChanges, r.PlayersFromNothing)
	s += "  ecart nouveau - ancien (s) : " + distribution(kickoff.Summarize(r.PlayerGapsSeconds, 2), "2 s") + "\n"
	s += fmt.Sprintf("film garde (rejeu absent ou illisible) : %d   rejeu hors de la fenetre du match : %d   "+
		"absents du registre : %d\n", r.KeptFilm, r.FilmOutOfBounds, r.Missing)
	s += fmt.Sprintf("ecrits : %d match(s)   echecs : %d%s   duree : %s\n", r.Written, r.Failures, exempleDEchec(r), duree)
	return s
}

func exempleDEchec(r kickoff.Report) string {
	if r.FailedExample == "" {
		return ""
	}
	return " (par exemple " + r.FailedExample + ")"
}

func distribution(d kickoff.Distribution, tol string) string {
	if d.N == 0 {
		return "aucun"
	}
	return fmt.Sprintf("n=%d min=%d p5=%d mediane=%d p95=%d max=%d, a %s pres : %d",
		d.N, d.Min, d.P5, d.P50, d.P95, d.Max, tol, d.Within)
}

func issues(m map[timeline.T0Quality]int) string {
	if len(m) == 0 {
		return "aucune"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)
	s := ""
	for i, k := range keys {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprintf("%s %d", k, m[timeline.T0Quality(k)])
	}
	return s
}
