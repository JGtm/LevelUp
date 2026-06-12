//go:build cgo

// cmd/backfill-world-player-stats — backfill one-shot des stats joueur du
// classement mondial (Phase D, PLAN_WORLD_LEADERBOARD_ENRICHED.md).
//
// Pour chaque saison CSR scrapée, agrège les compteurs bruts (match/win/k/d/a/
// playtime) par (saison, playlist) de chaque joueur classé, via le pool de tokens
// (MULTI-TOKENS, round-robin) pour le fetch des matchs + un compte unique pour la
// résolution xuid PeopleHub. Persiste en append-only (InsertPlayerSeasonStats).
//
// REPRISE : un checkpoint JSON (un fichier) mémorise les gamertags déjà traités
// par saison (succès dans "done", échecs dans "failed"). Relancer la commande
// reprend où elle s'est arrêtée (skip des gamertags faits ET échoués, skip des
// saisons complètes). Un joueur en échec persistant (gros historique qui 429 en
// boucle, xuid non résolu...) ne rebloque donc PLUS la saison : il est compté
// comme tenté, la saison se complète, et -retry-failed le re-tente explicitement.
// Ctrl-C (SIGINT/SIGTERM) arrête proprement : le lot en cours est flushé + le
// checkpoint sauvegardé avant de sortir. Idempotent par construction (append-only
// + vue _latest).
//
// IMPORTANT : stopper le serveur API avant de lancer (la shared DB est ouverte en
// RW ; DuckDB interdit deux writers sur Windows). Lancer de préférence en heures
// creuses (le backfill profond fetch beaucoup de matchs).
//
// Usage :
//
//	# saison courante, tous les joueurs, reprise auto :
//	go run ./cmd/backfill-world-player-stats -token-gamertag JGtm
//
//	# une vieille saison (scan profond), 50 joueurs, off-peak :
//	go run ./cmd/backfill-world-player-stats -token-gamertag JGtm \
//	    -season csrseason10-1 -limit 50 -max-pages 240
//
//	# reprise après un Ctrl-C : relancer la MÊME commande (lit le checkpoint).
//	# repartir de zéro : ajouter -force (ignore le checkpoint).
//	# valider sans écrire : -dry-run.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/worldenrich"
)

type cliFlags struct {
	season        string
	tokenGamertag string
	sharedDB      string
	checkpoint    string
	limit         int
	concurrency   int
	maxPages      int
	rps           int
	flushEvery    int
	force         bool
	dryRun        bool
	allTokens     bool
	deep          bool
	retryFailed   bool
	topN          int
	xuidDelayMs   int
}

func main() {
	f := parseFlags()

	closeLogs := logging.InstallCLI(os.Getenv("LEVELUP_REPO_ROOT"))
	defer closeLogs()

	// Arrêt propre : SIGINT/SIGTERM annule le ctx → les workers s'arrêtent, le lot
	// en cours est flushé + checkpoint sauvegardé avant sortie.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		fatal("chargement config: %v", err)
	}
	if err := run(ctx, cfg, f); err != nil {
		fatal("%v", err)
	}
}

func parseFlags() cliFlags {
	var f cliFlags
	flag.StringVar(&f.season, "season", "all", "saison CSR ciblée (ex: csrseason13-2) ou 'all'")
	flag.StringVar(&f.tokenGamertag, "token-gamertag", "", "gamertag du compte dont le token sert à résoudre les xuid (PeopleHub) — requis")
	flag.StringVar(&f.sharedDB, "shared-db", "",
		"chemin shared_matches_v2.duckdb (RW — stopper le serveur) ; vide = dérivé de RepoRoot")
	flag.StringVar(&f.checkpoint, "checkpoint", "", "fichier de reprise (JSON) ; vide = <RepoRoot>/data/world_backfill_checkpoint.json")
	flag.IntVar(&f.limit, "limit", 0, "nb max de joueurs par saison (0 = tous)")
	flag.IntVar(&f.concurrency, "concurrency", 4, "nb de joueurs traités en parallèle (↓ si 429)")
	flag.IntVar(&f.maxPages, "max-pages", 80, "pages d'historique max/joueur (25 matchs/page ; ↑ pour vieilles saisons)")
	flag.IntVar(&f.rps, "rps", 3, "requêtes/seconde PAR token (Halo limite ~par IP : RPS effectif = rps × nb tokens ; ↓ si 429)")
	flag.IntVar(&f.flushEvery, "flush-every", 20, "persiste + checkpoint tous les N joueurs")
	flag.BoolVar(&f.force, "force", false, "ignore le checkpoint (repart de zéro)")
	flag.BoolVar(&f.dryRun, "dry-run", false, "agrège sans écrire en base")
	flag.BoolVar(&f.allTokens, "all-tokens", false, "fetch en round-robin sur TOUS les comptes db_profiles résolus (Halo limite ~par IP → gain borné, pas N×)")
	flag.BoolVar(&f.deep, "deep", false, "désactive l'arrêt-anticipé (scan profond) — pour backfiller une VIEILLE saison ; combiner avec -max-pages élevé")
	flag.BoolVar(&f.retryFailed, "retry-failed", false, "re-tente les joueurs précédemment en échec (par défaut ils sont sautés à la reprise pour ne pas rebloquer la saison)")
	flag.IntVar(&f.topN, "top-n", duckdb.WorldLeaderboardTopN, "n'enrichit que le top N PAR playlist (= profondeur affichée ; 0 = toutes les playlists/rangs)")
	flag.IntVar(&f.xuidDelayMs, "xuid-delay-ms", 1600, "délai entre résolutions xuid PeopleHub (limite ~10 req/15s/compte ; ↑ si 429, 0 = pas de throttle)")
	flag.Parse()
	if strings.TrimSpace(f.tokenGamertag) == "" {
		fatal("-token-gamertag est requis (compte dont le token résout les xuid PeopleHub)")
	}
	return f
}

func run(ctx context.Context, cfg *config.AppConfig, f cliFlags) error {
	// 1. Source de fetch Halo (résolution store-first + legacy, comme les backfills
	// existants — pas de pool reconstruit). Single-token par défaut ; -all-tokens
	// round-robine tous les comptes db_profiles résolus (gain borné par l'IP).
	var src service.WorldMatchSource
	if f.allTokens {
		s, gts, e := worldenrich.BuildMultiHaloSource(cfg, f.rps, true)
		if e != nil {
			return e
		}
		src = s
		fmt.Printf("tokens: %d comptes résolus (round-robin) — %v\n", len(gts), gts)
	} else {
		s, e := worldenrich.BuildHaloSource(cfg, f.tokenGamertag, f.rps, true)
		if e != nil {
			return e
		}
		src = s
		fmt.Printf("token: %s résolu — single-token (auto-refresh)\n", f.tokenGamertag)
	}

	// 2. Résolveur xuid PeopleHub (même compte, header RTA mémoïsé).
	resolver, err := worldenrich.BuildResolver(cfg, f.tokenGamertag)
	if err != nil {
		return err
	}

	// 3. Shared DB en RW (serveur stoppé) + migrations.
	sharedPath := f.sharedDB
	if strings.TrimSpace(sharedPath) == "" {
		sharedPath = titlepkg.NewPathResolver(cfg.RepoRoot).SharedDBPath(titlepkg.DefaultSlug)
	}
	f.sharedDB = sharedPath // propagé aux flush ultérieurs
	fmt.Printf("shared DB: %s\n", sharedPath)
	db, err := openSharedRW(sharedPath)
	if err != nil {
		return fmt.Errorf("open shared DB (serveur stoppé ?): %w", err)
	}
	defer db.Close()
	if !f.dryRun {
		if err := migration.RunForDB(db, migration.TargetShared); err != nil {
			return fmt.Errorf("migration shared: %w", err)
		}
	}

	// 4. Saisons cibles.
	seasons, err := resolveSeasons(ctx, db, f.season)
	if err != nil {
		return err
	}
	if len(seasons) == 0 {
		return fmt.Errorf("aucune saison à traiter (snapshots CSR absents ?)")
	}

	// 5. Checkpoint (reprise). Défaut dérivé de RepoRoot (dossier garanti existant
	// après MkdirAll au save), pas relatif au CWD.
	if strings.TrimSpace(f.checkpoint) == "" {
		f.checkpoint = filepath.Join(cfg.RepoRoot, "data", "world_backfill_checkpoint.json")
	}
	cp := loadCheckpoint(f.checkpoint, f.force)
	fmt.Printf("checkpoint: %s\n", f.checkpoint)
	fmt.Printf("saisons: %v%s\n", seasons, dryRunSuffix(f.dryRun))

	// 6. Backfill saison par saison.
	for _, season := range seasons {
		// -retry-failed rouvre une saison complétée pour re-tenter ses joueurs en échec.
		if cp.completed(season) && !f.retryFailed {
			fmt.Printf("[%s] déjà complète — skip\n", season)
			continue
		}
		if err := backfillSeason(ctx, db, src, resolver, season, f, cp); err != nil {
			if ctx.Err() != nil {
				fmt.Printf("\nArrêt demandé — checkpoint sauvegardé. Relancer la même commande pour reprendre.\n")
				return nil
			}
			fmt.Printf("[%s] erreur saison: %v (on continue)\n", season, err)
		}
	}
	fmt.Printf("\nBackfill terminé.\n")
	return nil
}

// playerOutcome porte le résultat d'un joueur (depuis un worker).
type playerOutcome struct {
	gamertag string
	stats    []domain.WorldPlayerSeasonStats
	err      error
}

// backfillSeason agrège les stats de tous les joueurs (non encore traités) d'une
// saison, en parallèle (multi-tokens), avec flush + checkpoint périodiques.
func backfillSeason(
	ctx context.Context, db *sql.DB, pooled service.WorldMatchSource,
	resolver service.WorldXUIDResolver, season string, f cliFlags, cp *checkpoint,
) error {
	gamertags, err := duckdb.WorldSeasonGamertags(ctx, db, season, f.topN)
	if err != nil {
		return err
	}
	pending := cp.remaining(season, gamertags, f.limit, f.retryFailed)
	already := cp.doneCount(season, gamertags)
	if len(pending) == 0 {
		markSeasonCompleteIfFull(season, gamertags, f, cp, cp.attemptedCount(season, gamertags))
		fmt.Printf("[%s] %d/%d joueurs déjà traités\n", season, already, len(gamertags))
		return nil
	}
	target := already + len(pending) // borne réaliste de ce run (respecte -limit)
	fmt.Printf("[%s] %d joueurs à traiter (%d déjà faits, %d au total)\n",
		season, len(pending), already, len(gamertags))

	// Arrêt-anticipé ACTIVÉ par défaut : l'historique est chronologique décroissant,
	// donc pour la saison courante on s'arrête dès qu'on enchaîne assez de matchs de
	// la saison précédente → fetch borné à ~(matchs de la saison + 2 pages), pas 80
	// pages. -deep le désactive pour les VIEILLES saisons (scan profond nécessaire
	// pour les atteindre), au prix d'un gros volume.
	stopAfter := 0 // 0 → défaut 50 (cf. withDefaults)
	if f.deep {
		stopAfter = -1
	}
	agg := service.NewWorldStatsAggregator(pooled, resolver, service.WorldStatsAggregatorConfig{
		TargetSeasons:      map[string]bool{analysis.NormalizeSeasonID(season): true},
		MaxPages:           f.maxPages,
		StopAfterNonTarget: stopAfter,
		RankedPlaylists:    service.RankedPlaylistSet(), // ranked-only (ignore le social)
		// Throttle PeopleHub : la résolution xuid est single-token (~10 req/15s) ;
		// sans délai, 200+ joueurs d'un coup → 429 qui les skippent en masse.
		XUIDResolveDelay: time.Duration(f.xuidDelayMs) * time.Millisecond,
	})

	// Résolution xuid EN AMONT : un seul GetMatchStats par match traite alors TOUS
	// les joueurs mondiaux présents (jusqu'à 8) au lieu de re-fetcher par joueur.
	if prepErrs := agg.PrepareWorldPlayers(ctx, pending); len(prepErrs) > 0 {
		fmt.Printf("[%s] %d xuid non résolus (joueurs skippés) — 1er: %v\n", season, len(prepErrs), prepErrs[0])
	}

	if err := runSeasonWorkers(ctx, agg, season, pending, already, target, f, cp); err != nil {
		return err
	}
	// Saison complète quand tous ses gamertags ont été TENTÉS (done ∪ failed) — un
	// échec accepté ne doit pas rebloquer indéfiniment (recompte après le run).
	markSeasonCompleteIfFull(season, gamertags, f, cp, cp.attemptedCount(season, gamertags))
	return nil
}

// markSeasonCompleteIfFull marque la saison complète seulement si TOUS ses
// gamertags ont été tentés (done ∪ failed ; jamais sur un sous-ensemble -limit) et
// hors dry-run.
func markSeasonCompleteIfFull(season string, gamertags []string, f cliFlags, cp *checkpoint, attemptedCount int) {
	if f.dryRun || attemptedCount < len(gamertags) {
		return
	}
	cp.markCompleted(season)
	_ = cp.save(f.checkpoint)
	fmt.Printf("[%s] saison complète (%d joueurs)\n", season, len(gamertags))
}

// runSeasonWorkers orchestre le pool de workers + le collecteur (flush/checkpoint/
// progression). Retourne ctx.Err() si arrêt demandé (après flush du lot en cours).
func runSeasonWorkers(
	ctx context.Context, agg *service.WorldStatsAggregator, season string,
	pending []string, already, total int, f cliFlags, cp *checkpoint,
) error {
	tasks := make(chan string)
	results := make(chan playerOutcome, f.concurrency)
	var wg sync.WaitGroup

	for i := 0; i < f.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for gt := range tasks {
				stats, err := agg.AggregatePlayer(ctx, gt)
				results <- playerOutcome{gamertag: gt, stats: stats, err: err}
			}
		}()
	}
	go func() { // producteur : stoppe d'alimenter dès que le ctx est annulé
		defer close(tasks)
		for _, gt := range pending {
			select {
			case <-ctx.Done():
				return
			case tasks <- gt:
			}
		}
	}()
	go func() { wg.Wait(); close(results) }()

	return collectSeason(ctx, season, results, already, total, f, cp)
}

// collectSeason draine les résultats, persiste + checkpoint par lots de flushEvery,
// affiche la progression, et flushe le reliquat à la fin.
func collectSeason(
	ctx context.Context, season string, results <-chan playerOutcome,
	already, total int, f cliFlags, cp *checkpoint,
) error {
	t0 := time.Now()
	var batch []domain.WorldPlayerSeasonStats
	var batchGTs []string
	var failedGTs []string
	done, failures, rows := already, 0, 0

	flush := func() error {
		n, err := flushBatch(ctx, f, batch)
		if err != nil {
			return err
		}
		rows += n
		// Dry-run = répétition à blanc : ne JAMAIS toucher le checkpoint (sinon un run
		// réel ultérieur saute des joueurs jamais insérés).
		if !f.dryRun {
			cp.markDone(season, batchGTs)
			// Les joueurs en échec sont AUSSI checkpointés (liste failed) : ils ne
			// rebloquent plus la saison à la reprise et comptent vers la complétude.
			cp.markFailed(season, failedGTs)
			if err := cp.save(f.checkpoint); err != nil {
				return err
			}
		}
		batch, batchGTs, failedGTs = nil, nil, nil
		return nil
	}

	for o := range results {
		done++
		if o.err != nil {
			failures++
			failedGTs = append(failedGTs, o.gamertag)
		} else {
			batch = append(batch, o.stats...)
			batchGTs = append(batchGTs, o.gamertag)
		}
		if len(batchGTs)+len(failedGTs) >= f.flushEvery {
			if err := flush(); err != nil {
				return err
			}
		}
		printProgress(season, done, total, failures, rows, time.Since(t0))
	}
	if err := flush(); err != nil { // reliquat (inclut le cas arrêt anticipé)
		return err
	}
	// La complétude de saison est décidée par le caller (backfillSeason), jamais ici :
	// un run -limit ne traite qu'un sous-ensemble et ne doit pas marquer la saison complète.
	if ctx.Err() == nil {
		suffix := ""
		if f.dryRun {
			suffix = " [dry-run, non persisté]"
		}
		fmt.Printf("\n[%s] terminé : %d joueurs traités, %d lignes%s, %d erreurs, %s\n",
			season, done-already, rows, suffix, failures, time.Since(t0).Round(time.Second))
	}
	return ctx.Err()
}

// flushBatch persiste un lot. En dry-run, ne touche pas la base mais retourne le
// nombre de lignes qui SERAIENT insérées (validation utile de l'agrégation).
func flushBatch(ctx context.Context, f cliFlags, batch []domain.WorldPlayerSeasonStats) (int, error) {
	if len(batch) == 0 {
		return 0, nil
	}
	if f.dryRun {
		return len(batch), nil
	}
	db, err := openSharedRW(f.sharedDB)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	return duckdb.InsertPlayerSeasonStats(ctx, db, batch)
}

// printProgress affiche une ligne de progression (réécrite en place via \r).
func printProgress(season string, done, total, failures, rows int, elapsed time.Duration) {
	fmt.Printf("\r[%s] %d/%d joueurs · %d lignes · %d err · %s    ",
		season, done, total, rows, failures, elapsed.Round(time.Second))
}

// resolveSeasons retourne la liste des saisons à traiter ("all" → toutes les
// saisons des snapshots CSR, plus récentes d'abord).
func resolveSeasons(ctx context.Context, db *sql.DB, season string) ([]string, error) {
	if s := strings.TrimSpace(season); s != "" && s != "all" {
		return []string{s}, nil
	}
	rows, err := db.QueryContext(ctx,
		`SELECT DISTINCT season_id FROM world_csr_leaderboard_latest
		 WHERE season_id <> ''`)
	if err != nil {
		return nil, fmt.Errorf("liste des saisons: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Récent d'abord. ATTENTION : un ORDER BY season_id en SQL est ALPHABÉTIQUE
	// (csrseason6-1 > csrseason13-2 car '6' > '1') → mettrait les vieilles saisons
	// en premier. On trie par NUMÉRO de saison.
	sort.SliceStable(out, func(i, j int) bool { return seasonRank(out[i]) > seasonRank(out[j]) })
	return out, nil
}

// seasonRank extrait un rang triable d'un id "csrseason{major}-{minor}"
// (major*100 + minor). Format inconnu → 0 (relégué en fin).
func seasonRank(id string) int {
	s := strings.TrimPrefix(id, "csrseason")
	major, minor := s, "0"
	if i := strings.IndexByte(s, '-'); i >= 0 {
		major, minor = s[:i], s[i+1:]
	}
	mj, _ := strconv.Atoi(major)
	mn, _ := strconv.Atoi(minor)
	return mj*100 + mn
}

// openSharedRW ouvre la shared DB en écriture (1 seule connexion).
func openSharedRW(path string) (*sql.DB, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func dryRunSuffix(dry bool) string {
	if dry {
		return " [dry-run]"
	}
	return ""
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\nFATAL: "+format+"\n", args...)
	os.Exit(1)
}
