// Outil d'analyse : produit des RECOMMANDATIONS de tuning de la grammaire de
// synthèse du coach_advisor (config/coach_advisor/synthesis_grammar.toml) à partir
// de la télémétrie Prestige (ADR 0020/0028).
//
// L'application reste MANUELLE : un humain lit le rapport et édite le TOML. Aucun
// mécanisme de PR automatique, aucun override runtime (cadrage superviseur).
//
// Règle de référence (paramétrable) : par métrique de grammaire, si le taux de
// complétion est < --min-completion sur un échantillon d'au moins --min-sample
// défis coach acceptés, recommander de retirer la métrique ou de réduire ses
// fenêtres. En dessous de l'échantillon : "données insuffisantes".
//
// Lecture SEULE (duckdb.OpenReadForQuery) — jamais d'ouverture RW.
//
// Usage :
//
//	go run ./cmd/prestige-tuning-analyze [--format text|json] [--player <slug>]
//	    [--title halo_infinite] [--min-completion 0.30] [--min-sample 50]
//	    [--source coach] [--grammar <path>]
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis/prestigetuning"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/progression/coach_advisor"
)

// tuningLog route les logs de la CLI vers logs/prestige.log (tag explicite : le
// package main n'a pas de module dédié → general.log). Les skips de collecte
// best-effort (DB absente, verrouillée, agrégation échouée) restent diagnosticables
// avec le reste du sous-système Prestige.
var tuningLog = slog.With("module", logging.ModulePrestige)

type options struct {
	format        string
	player        string
	title         string
	minCompletion float64
	minSample     int
	source        string
	grammarPath   string
}

func main() {
	if err := run(context.Background()); err != nil {
		tuningLog.Error("prestige-tuning-analyze failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	opt := parseFlags()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config.Load: %w", err)
	}
	titleSlug := opt.title
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}

	grammar, err := loadGrammar(cfg.RepoRoot, opt.grammarPath)
	if err != nil {
		return err
	}

	players, err := cfg.LoadPlayers(titleSlug)
	if err != nil {
		return fmt.Errorf("LoadPlayers: %w", err)
	}
	players = filterPlayers(players, opt.player)

	counts, accept, scanned := collectAll(ctx, cfg, titleSlug, players)

	thr := prestigetuning.Thresholds{
		MinCompletionRate: opt.minCompletion,
		MinSample:         opt.minSample,
		Source:            opt.source,
	}
	rep := prestigetuning.Analyze(
		prestigetuning.MergeCounts(counts),
		prestigetuning.MergeAcceptance(accept),
		grammar, thr, time.Now().UTC(),
	)
	rep.TitleSlug = titleSlug
	rep.PlayersScanned = scanned
	rep.PlayerScope = playerScope(opt.player)

	return emit(rep, opt.format)
}

func parseFlags() options {
	opt := options{}
	def := prestigetuning.DefaultThresholds()
	flag.StringVar(&opt.format, "format", "text", "format de sortie : text | json")
	flag.StringVar(&opt.player, "player", "", "restreindre à un joueur (player_slug ou gamertag) ; vide = tous")
	flag.StringVar(&opt.title, "title", "", "slug du titre (défaut : halo_infinite)")
	flag.Float64Var(&opt.minCompletion, "min-completion", def.MinCompletionRate, "seuil de complétion (0..1) sous lequel recommander un ajustement")
	flag.IntVar(&opt.minSample, "min-sample", def.MinSample, "échantillon minimal de défis acceptés pour statuer")
	flag.StringVar(&opt.source, "source", def.Source, "origine des défis analysée pour les recommandations")
	flag.StringVar(&opt.grammarPath, "grammar", "", "chemin de synthesis_grammar.toml (défaut : config/coach_advisor/)")
	flag.Parse()
	return opt
}

// loadGrammar charge la grammaire et la projette en GrammarView pure.
func loadGrammar(repoRoot, override string) (prestigetuning.GrammarView, error) {
	path := override
	if path == "" {
		path = filepath.Join(repoRoot, "config", "coach_advisor", "synthesis_grammar.toml")
	}
	g, err := coach_advisor.LoadSynthesisGrammar(path)
	if err != nil {
		return prestigetuning.GrammarView{}, fmt.Errorf("chargement grammaire: %w", err)
	}
	mw := map[string][]string{}
	for _, m := range g.Metrics() {
		mw[m] = g.WindowSpecs(m)
	}
	return prestigetuning.NewGrammarView(mw), nil
}

// filterPlayers restreint à un joueur si demandé (par player_slug OU gamertag).
func filterPlayers(players []domain.PlayerSummary, want string) []domain.PlayerSummary {
	if want == "" {
		return players
	}
	var out []domain.PlayerSummary
	for _, p := range players {
		if p.PlayerSlug == want || p.Gamertag == want {
			out = append(out, p)
		}
	}
	return out
}

// collectAll agrège la télémétrie de chaque player DB en lecture seule.
// Best-effort : une DB illisible (verrouillée, absente, legacy sans tables) est
// loguée puis ignorée — l'analyse continue sur les DB disponibles.
func collectAll(ctx context.Context, cfg *config.AppConfig, titleSlug string, players []domain.PlayerSummary) (
	[]prestigetuning.MetricWindowCount, []prestigetuning.SourceAcceptance, int) {

	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	var counts []prestigetuning.MetricWindowCount
	var accept []prestigetuning.SourceAcceptance
	scanned := 0

	for _, p := range players {
		dbPath := pr.PlayerDBPath(titleSlug, p.Gamertag)
		if _, err := os.Stat(dbPath); err != nil {
			tuningLog.WarnContext(ctx, "player DB absente, ignorée", "player", p.Gamertag, "path", dbPath)
			continue
		}
		db, release, err := duckdb.OpenReadForQuery(dbPath)
		if err != nil {
			tuningLog.WarnContext(ctx, "ouverture lecture échouée (verrou ?), joueur ignoré",
				"player", p.Gamertag, "path", dbPath, "err", err)
			continue
		}
		c, a, err := prestigetuning.CollectFromDB(ctx, db)
		release()
		if err != nil {
			tuningLog.WarnContext(ctx, "agrégation télémétrie échouée, joueur ignoré",
				"player", p.Gamertag, "err", err)
			continue
		}
		counts = append(counts, c...)
		accept = append(accept, a...)
		scanned++
	}
	return counts, accept, scanned
}

func emit(rep prestigetuning.Report, format string) error {
	if format == "json" {
		out, err := prestigetuning.RenderJSON(rep)
		if err != nil {
			return fmt.Errorf("render JSON: %w", err)
		}
		fmt.Println(string(out))
		return nil
	}
	fmt.Print(prestigetuning.RenderText(rep))
	return nil
}

func playerScope(player string) string {
	if player == "" {
		return "all"
	}
	return player
}
