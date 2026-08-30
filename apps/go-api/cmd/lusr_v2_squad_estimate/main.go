//go:build cgo

// lusr_v2_squad_estimate — LUSR v2 Sprint 1.C : estimation hors-ligne des offsets
// de synergie d'escouade (player_squad_offset).
//
// Pour chaque paire de coéquipiers ayant joué ensemble ≥ --min-matches dans les
// --weeks dernières semaines (par playlist_group), on calcule un offset de
// synergie = moyenne de (issue réelle − proba de victoire prédite par les
// ratings SOLO), convertie en unités μ via --gain et bornée à ±SquadOffsetCap.
//
// **MVP first-pass, à raffiner (cf. Sprint 3.A)** :
//   - SoloWinProb est estimé depuis les ratings SOLO COURANTS
//     (player_skill_state_v2_latest) comme proxy du rating pré-match. C'est une
//     approximation : les ratings courants incluent déjà ces matchs, ce qui
//     biaise SoloWinProb vers le haut → offset SOUS-estimé. Biais conservateur
//     (sous-correction) donc sûr ; ré-exécuter périodiquement raffine.
//   - L'offset n'est CONSOMMÉ au runtime que si LEVELUP_LUSR_V2_SQUAD_OFFSET=1.
//
// Usage :
//
//	go run -tags cgo ./apps/go-api/cmd/lusr_v2_squad_estimate [--dry-run]
//	  [--db PATH] [--weeks 4] [--min-matches 10] [--gain 6.0]
//
// Idempotent (UPSERT append-only ; la vue _latest dédoublonne).
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/games/halo_infinite/skillchain"
	"levelup/go-api/internal/platform/duckdb"
	lusync "levelup/go-api/internal/sync"

	_ "github.com/duckdb/duckdb-go/v2"
)

const sharedDBPath = "data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"

// matchRoster : un match 2 équipes avec ses rosters et l'issue par équipe.
type matchRoster struct {
	group     string
	teams     map[int][]string // team_id → xuids
	wonByTeam map[int]float64  // team_id → 1.0 win / 0.5 tie / 0.0 loss
}

// pairKey identifie une paire ordonnée (a<b) sur un groupe.
type pairKey struct {
	a, b, group string
}

func main() {
	lusync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)        // MT-15 (fail-loud)
	lusync.SetObjectiveFamilyClassifier(skillchain.IsObjectiveSubMode) // famille de la chaîne de perf classée

	dbPath := flag.String("db", sharedDBPath, "chemin vers shared_matches_v2.duckdb")
	weeks := flag.Int("weeks", 4, "fenêtre glissante (semaines) des matchs analysés")
	minMatches := flag.Int("min-matches", 10, "# minimum de matchs ensemble pour estimer une paire")
	gain := flag.Float64("gain", 6.0, "conversion résidu de win-rate → μ (muPerWinResidual)")
	dryRun := flag.Bool("dry-run", false, "n'écrit pas en DB, affiche le rapport")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	db, err := sql.Open("duckdb", *dbPath)
	if err != nil {
		slog.Error("open duckdb", "err", err)
		os.Exit(1)
	}
	defer db.Close() //nolint:errcheck

	ctx := context.Background()
	since := time.Now().AddDate(0, 0, -7*(*weeks))

	ratings, err := loadSoloRatings(ctx, db)
	if err != nil {
		slog.Error("loadSoloRatings", "err", err)
		os.Exit(1)
	}
	matches, err := loadWindowMatches(ctx, db, since)
	if err != nil {
		slog.Error("loadWindowMatches", "err", err)
		os.Exit(1)
	}
	offsets := estimateOffsets(matches, ratings, *minMatches, *gain)
	source := fmt.Sprintf("squad_batch_%s", time.Now().Format("2006_01_02"))
	printReport(offsets, since, *minMatches, *gain)

	if *dryRun {
		slog.Info("dry-run : aucune écriture DB", "pairs_éligibles", len(offsets))
		return
	}
	if err := writeOffsets(ctx, duckdb.NewSquadOffsetRepo(db), offsets, source); err != nil {
		slog.Error("writeOffsets", "err", err)
		os.Exit(1)
	}
	slog.Info("squad estimate terminé", "source", source, "pairs_écrites", 2*len(offsets))
}

// loadSoloRatings retourne les μ/σ courants par (group, xuid).
func loadSoloRatings(ctx context.Context, db *sql.DB) (map[string]map[string]skillv2.Gaussian, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT playlist_group, xuid, mu, sigma FROM player_skill_state_v2_latest`)
	if err != nil {
		return nil, fmt.Errorf("query ratings: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	out := make(map[string]map[string]skillv2.Gaussian)
	for rows.Next() {
		var group, xuid string
		var mu, sigma float64
		if err := rows.Scan(&group, &xuid, &mu, &sigma); err != nil {
			return nil, err
		}
		if out[group] == nil {
			out[group] = make(map[string]skillv2.Gaussian)
		}
		out[group][xuid] = skillv2.Gaussian{Mu: mu, Sigma: sigma}
	}
	return out, rows.Err()
}

// loadWindowMatches charge les matchs 2 équipes sociaux de la fenêtre, avec
// rosters et issue par équipe.
func loadWindowMatches(ctx context.Context, db *sql.DB, since time.Time) ([]matchRoster, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT mr.match_id, COALESCE(mr.pair_name, ''), mp.xuid, mp.team_id, COALESCE(mp.outcome, 0)
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
		WHERE COALESCE(mr.is_ranked, FALSE) = FALSE
		  AND COALESCE(mr.is_firefight, FALSE) = FALSE
		  AND mr.start_time IS NOT NULL
		  AND mr.start_time >= ?
		  AND (mr.duration_seconds IS NULL OR mr.duration_seconds >= 30)
		  AND mp.xuid IS NOT NULL AND mp.xuid != ''
		ORDER BY mr.match_id`, since)
	if err != nil {
		return nil, fmt.Errorf("query matches: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	byMatch := make(map[string]*matchRoster)
	for rows.Next() {
		var matchID, pairName, xuid string
		var teamID, outcome int
		if err := rows.Scan(&matchID, &pairName, &xuid, &teamID, &outcome); err != nil {
			return nil, err
		}
		group := lusync.GetLUSRChain(pairName)
		if group == "" {
			continue
		}
		mr, ok := byMatch[matchID]
		if !ok {
			mr = &matchRoster{group: group, teams: map[int][]string{}, wonByTeam: map[int]float64{}}
			byMatch[matchID] = mr
		}
		mr.teams[teamID] = append(mr.teams[teamID], xuid)
		mr.wonByTeam[teamID] = outcomeToWon(outcome)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]matchRoster, 0, len(byMatch))
	for _, mr := range byMatch {
		if len(mr.teams) == 2 { // strictement 2 équipes (cohérent shadow runner)
			out = append(out, *mr)
		}
	}
	return out, nil
}

// outcomeToWon convertit le code Halo (2=Win,1=Tie,3=Loss) en score [0,1].
func outcomeToWon(o int) float64 {
	switch o {
	case 2:
		return 1.0
	case 1:
		return 0.5
	default:
		return 0.0
	}
}
