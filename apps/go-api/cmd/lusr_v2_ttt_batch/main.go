//go:build cgo

// lusr_v2_ttt_batch — Phase 5 du chantier LUSR v2 : ré-estimation batch des
// hyperparamètres globaux à partir de l'historique des matchs.
//
// **MVP volontairement léger** : pas de full Through-Time TS2 §10 (forward +
// backward smoothing + EM sur σ_skill/β/τ). On calcule des statistiques
// empiriques par playlist_group et on les pousse dans lusr_hyperparams_v2
// avec une source datée "batch_YYYY_MM_DD".
//
// Statistiques produites (par playlist_group) :
//   - draw_probability       = #matchs draw / #matchs
//   - kill_mean              = moyenne kills / participant (pour bias kills)
//   - kill_std               = écart-type kills / participant
//   - death_mean             = moyenne deaths
//   - death_std              = écart-type deaths
//   - match_count            = #matchs analysés
//
// Les hyperparams empiriques sont relus au runtime par le shadow runner depuis
// lusr_hyperparams_v2_latest (Sprint 1.B : resolveGroupParams →
// LoadPriorsFromHyperparams / LoadCountHyperparamsFromDB). En plus des stats de
// base, ce batch calcule la matrice de couplage cross-mode (Sprint 2.B).
//
// **Limites** : pas de TTT smoothing forward+backward complet ici — le prototype
// de lisseur EM vit dans internal/analysis/skill_v2/ttt.go (Sprint 3.A), pas
// encore branché sur ce batch (couplage inter-joueurs = follow-up).
//
// Usage :
//
//	go run -tags cgo ./apps/go-api/cmd/lusr_v2_ttt_batch [--dry-run]
//	--dry-run : affiche le rapport sans écrire en DB
//
// Idempotent : ré-exécuter le même jour réécrit la même source ; chaque rerun
// crée une nouvelle row append-only mais la vue _latest dédoublonne.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/skillchain"
	"levelup/go-api/internal/platform/duckdb"
	lusync "levelup/go-api/internal/sync"

	_ "github.com/duckdb/duckdb-go/v2"
)

const sharedDBPath = "data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"

type groupStats struct {
	matchCount int
	drawCount  int
	killSum    float64
	killSqSum  float64
	killN      int
	deathSum   float64
	deathSqSum float64
	deathN     int
}

func (g *groupStats) drawProb() float64 {
	if g.matchCount == 0 {
		return 0
	}
	return float64(g.drawCount) / float64(g.matchCount)
}

func (g *groupStats) killStats() (mean, std float64) {
	if g.killN == 0 {
		return 0, 0
	}
	mean = g.killSum / float64(g.killN)
	variance := (g.killSqSum / float64(g.killN)) - mean*mean
	if variance < 0 {
		variance = 0
	}
	return mean, math.Sqrt(variance)
}

func (g *groupStats) deathStats() (mean, std float64) {
	if g.deathN == 0 {
		return 0, 0
	}
	mean = g.deathSum / float64(g.deathN)
	variance := (g.deathSqSum / float64(g.deathN)) - mean*mean
	if variance < 0 {
		variance = 0
	}
	return mean, math.Sqrt(variance)
}

func main() {
	lusync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)        // MT-15 (fail-loud)
	lusync.SetObjectiveFamilyClassifier(skillchain.IsObjectiveSubMode) // famille de la chaîne de perf classée

	dbPath := flag.String("db", sharedDBPath, "chemin vers shared_matches_v2.duckdb")
	dryRun := flag.Bool("dry-run", false, "n'écrit pas en DB, affiche le rapport seulement")
	smooth := flag.Bool("smooth", false, "TTT smoothing 3.A : estime τ par joueur/groupe et écrit ttt_tau_empirical")
	writeSmoothed := flag.Bool("write-smoothed", false, "avec --smooth : écrit le μ lissé terminal dans player_skill_state_v2")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	db, err := sql.Open("duckdb", *dbPath)
	if err != nil {
		slog.Error("open duckdb", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()
	stats, err := computeStats(ctx, db)
	if err != nil {
		slog.Error("computeStats", "err", err)
		os.Exit(1)
	}
	// Sprint 2.B : matrice de couplage cross-mode (corrélation des μ entre modes).
	states, err := loadPlayerStatesByXUID(ctx, db)
	if err != nil {
		slog.Error("loadPlayerStatesByXUID", "err", err)
		os.Exit(1)
	}
	matrix := skillv2.EstimateCouplingMatrix(states)

	source := fmt.Sprintf("batch_%s", time.Now().Format("2006_01_02"))
	printReport(stats, source)
	printModeCouplingReport(matrix)

	if *smooth {
		if err := runTTTSmoothing(ctx, db, *dryRun, *writeSmoothed); err != nil {
			slog.Error("runTTTSmoothing", "err", err)
			os.Exit(1)
		}
	}

	if *dryRun {
		slog.Info("dry-run : aucune écriture DB")
		return
	}
	repo := duckdb.NewSkillV2Repo(db)
	if err := writeHyperparams(ctx, repo, stats, source); err != nil {
		slog.Error("writeHyperparams", "err", err)
		os.Exit(1)
	}
	if err := writeModeCoupling(ctx, repo, matrix, source); err != nil {
		slog.Error("writeModeCoupling", "err", err)
		os.Exit(1)
	}
	slog.Info("Phase 5 TTT batch terminé", "source", source, "groups", len(stats), "coupling_pairs", len(matrix))
}

// computeStats scanne match_registry × match_participants et agrège les
// statistiques par playlist_group LUSR. Skip les ranked et firefight (mêmes
// règles que le shadow runner).
func computeStats(ctx context.Context, db *sql.DB) (map[string]*groupStats, error) {
	stats := make(map[string]*groupStats)

	// Pass 1 : matchs (pour draws + match_count).
	rows, err := db.QueryContext(ctx, `
		SELECT mr.match_id,
		       COALESCE(mr.pair_name, ''),
		       (SELECT MAX(outcome) FROM match_participants WHERE match_id = mr.match_id) AS max_outcome,
		       (SELECT MIN(outcome) FROM match_participants WHERE match_id = mr.match_id) AS min_outcome
		FROM match_registry mr
		WHERE COALESCE(mr.is_ranked, FALSE) = FALSE
		  AND COALESCE(mr.is_firefight, FALSE) = FALSE
		  AND mr.start_time IS NOT NULL
		  AND (mr.duration_seconds IS NULL OR mr.duration_seconds >= 30)`)
	if err != nil {
		return nil, fmt.Errorf("query matches: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var matchID, pairName string
		var maxOut, minOut sql.NullInt64
		if err := rows.Scan(&matchID, &pairName, &maxOut, &minOut); err != nil {
			return nil, err
		}
		group := lusync.GetLUSRChain(pairName)
		if group == "" {
			continue
		}
		g, ok := stats[group]
		if !ok {
			g = &groupStats{}
			stats[group] = g
		}
		g.matchCount++
		// Draw heuristique : tous les participants ont outcome = 1 (Tie).
		if maxOut.Valid && minOut.Valid && maxOut.Int64 == 1 && minOut.Int64 == 1 {
			g.drawCount++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Pass 2 : participants pour kill/death stats.
	rows2, err := db.QueryContext(ctx, `
		SELECT COALESCE(mr.pair_name, ''),
		       mp.kills, mp.deaths
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
		WHERE COALESCE(mr.is_ranked, FALSE) = FALSE
		  AND COALESCE(mr.is_firefight, FALSE) = FALSE
		  AND mr.start_time IS NOT NULL
		  AND (mr.duration_seconds IS NULL OR mr.duration_seconds >= 30)
		  AND mp.xuid IS NOT NULL AND mp.xuid != ''`)
	if err != nil {
		return nil, fmt.Errorf("query participants: %w", err)
	}
	defer rows2.Close() //nolint:errcheck

	for rows2.Next() {
		var pairName string
		var kills, deaths sql.NullFloat64
		if err := rows2.Scan(&pairName, &kills, &deaths); err != nil {
			return nil, err
		}
		group := lusync.GetLUSRChain(pairName)
		if group == "" {
			continue
		}
		g, ok := stats[group]
		if !ok {
			continue
		}
		if kills.Valid {
			g.killSum += kills.Float64
			g.killSqSum += kills.Float64 * kills.Float64
			g.killN++
		}
		if deaths.Valid {
			g.deathSum += deaths.Float64
			g.deathSqSum += deaths.Float64 * deaths.Float64
			g.deathN++
		}
	}
	return stats, rows2.Err()
}

func printReport(stats map[string]*groupStats, source string) {
	groups := make([]string, 0, len(stats))
	for k := range stats {
		groups = append(groups, k)
	}
	sort.Strings(groups)

	var sb strings.Builder
	sb.WriteString("\n=== LUSR v2 Phase 5 TTT batch — Rapport ===\n")
	sb.WriteString(fmt.Sprintf("Source: %s\n\n", source))
	for _, g := range groups {
		s := stats[g]
		killMean, killStd := s.killStats()
		deathMean, deathStd := s.deathStats()
		sb.WriteString(fmt.Sprintf("Group: %s\n", g))
		sb.WriteString(fmt.Sprintf("  matchs analyses       : %d\n", s.matchCount))
		sb.WriteString(fmt.Sprintf("  draw probability      : %.4f (%d draws)\n", s.drawProb(), s.drawCount))
		sb.WriteString(fmt.Sprintf("  kills mean/std        : %.2f / %.2f (n=%d)\n", killMean, killStd, s.killN))
		sb.WriteString(fmt.Sprintf("  deaths mean/std       : %.2f / %.2f (n=%d)\n", deathMean, deathStd, s.deathN))
		sb.WriteString("\n")
	}
	fmt.Println(sb.String())
}

func writeHyperparams(ctx context.Context, repo *duckdb.SkillV2Repo,
	stats map[string]*groupStats, source string) error {
	for group, s := range stats {
		killMean, killStd := s.killStats()
		deathMean, deathStd := s.deathStats()
		params := []domain.SkillV2Hyperparam{
			{PlaylistGroup: group, Name: "draw_probability_empirical", Value: s.drawProb(), Source: source},
			{PlaylistGroup: group, Name: "kill_mean_empirical", Value: killMean, Source: source},
			{PlaylistGroup: group, Name: "kill_std_empirical", Value: killStd, Source: source},
			{PlaylistGroup: group, Name: "death_mean_empirical", Value: deathMean, Source: source},
			{PlaylistGroup: group, Name: "death_std_empirical", Value: deathStd, Source: source},
			{PlaylistGroup: group, Name: "match_count_analyzed", Value: float64(s.matchCount), Source: source},
		}
		for _, p := range params {
			if err := repo.UpsertHyperparam(ctx, p); err != nil {
				return fmt.Errorf("upsert %s/%s: %w", group, p.Name, err)
			}
		}
	}
	return nil
}
