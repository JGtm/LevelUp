//go:build cgo

package main

// ttt_smooth.go — Sprint 3.A wiring complet : TrueSkill Through Time sur les
// séries μ des joueurs trackés.
//
// Pour chaque (xuid, playlist_group) disposant d'au moins 2 snapshots dans
// player_skill_state_v2, on applique le lisseur EM (kalmanForward + rtsBackward)
// pour ré-estimer q = τ² et r = bruit d'observation. Par groupe de modes :
//
//	τ_groupe = √(mean(q_joueurs_du_groupe))
//
// est écrit dans lusr_hyperparams_v2 comme "ttt_tau_empirical", et sera lu au
// runtime par LoadPriorsFromHyperparams → Priors.Tau (Sprint 1.B).
//
// Flag --write-smoothed : écrit le μ lissé terminal de chaque joueur comme
// nouveau snapshot dans player_skill_state_v2 (written_at = NOW()). La vue
// _latest sélectionnera ce snapshot comme état courant. Le lisseur exploitant
// passé ET futur, cette valeur est une meilleure estimation du skill réel que
// le filtre forward seul.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// tttDefaultConfig retourne un TTTConfig calibré pour les séries μ LUSR.
// μ ∈ [0, 50], σ₀ ≈ 8.33 → q₀ = τ₀² ≈ 0.007, r₀ = β₀² ≈ 17.4.
func tttDefaultConfig(mu0 float64) skillv2.TTTConfig {
	sigma0 := mu0 / 3.0
	return skillv2.TTTConfig{
		InitialMean:    mu0,
		InitialVar:     sigma0 * sigma0,
		InitProcessVar: (sigma0 / 100.0) * (sigma0 / 100.0),
		InitObsVar:     (sigma0 / 2.0) * (sigma0 / 2.0),
		MaxIter:        30,
		Tol:            1e-3,
	}
}

// playerGroupHistory regroupe l'historique chronologique d'un joueur sur un groupe.
type playerGroupHistory struct {
	XUID   string
	Group  string
	States []domain.SkillV2State // written_at ASC
}

// tttGroupResult agrège les τ estimés par EM pour un groupe de modes.
type tttGroupResult struct {
	Group       string
	PlayerCount int
	AvgQ        float64 // mean(ProcessVar) sur les joueurs du groupe
	AvgR        float64 // mean(ObsVar)
	Tau         float64 // sqrt(AvgQ) — hyperparam ttt_tau_empirical
}

// runTTTSmoothing charge les historiques μ, appelle EstimateTTT par joueur/groupe,
// agrège par groupe, et écrit ttt_tau_empirical dans lusr_hyperparams_v2.
// Si writeSmoothed, écrit en plus le μ lissé terminal comme nouveau snapshot.
func runTTTSmoothing(ctx context.Context, db *sql.DB, dryRun, writeSmoothed bool) error {
	histories, err := loadAllHistories(ctx, db)
	if err != nil {
		return fmt.Errorf("loadAllHistories: %w", err)
	}
	if len(histories) == 0 {
		slog.InfoContext(ctx, "TTT smooth : aucun historique ≥ 2 points, rien à faire")
		return nil
	}

	cfg := tttDefaultConfig(25.0)

	groupQSum := make(map[string]float64)
	groupRSum := make(map[string]float64)
	groupCount := make(map[string]int)

	type smoothUpdate struct {
		state      domain.SkillV2State
		smoothedMu float64
	}
	var updates []smoothUpdate

	for _, h := range histories {
		z := make([]float64, len(h.States))
		for i, s := range h.States {
			z[i] = s.Mu
		}
		res := skillv2.EstimateTTT(z, cfg)
		if !res.Converged {
			slog.WarnContext(ctx, "TTT non convergé",
				"xuid", h.XUID, "group", h.Group, "iters", res.Iterations)
		}
		groupQSum[h.Group] += res.ProcessVar
		groupRSum[h.Group] += res.ObsVar
		groupCount[h.Group]++

		if writeSmoothed && len(res.SmoothedMean) > 0 {
			latest := h.States[len(h.States)-1]
			updates = append(updates, smoothUpdate{
				state:      latest,
				smoothedMu: res.SmoothedMean[len(res.SmoothedMean)-1],
			})
		}
	}

	groupResults := buildGroupResults(groupQSum, groupRSum, groupCount)
	printTTTSmoothReport(groupResults, writeSmoothed, len(updates))

	if dryRun {
		slog.InfoContext(ctx, "TTT smooth dry-run : aucune écriture DB")
		return nil
	}

	repo := duckdb.NewSkillV2Repo(db)
	source := fmt.Sprintf("ttt_%s", time.Now().Format("2006_01_02"))

	for _, gr := range groupResults {
		if err := repo.UpsertHyperparam(ctx, domain.SkillV2Hyperparam{
			PlaylistGroup: gr.Group,
			Name:          "ttt_tau_empirical",
			Value:         gr.Tau,
			Source:        source,
		}); err != nil {
			return fmt.Errorf("upsert ttt_tau %s: %w", gr.Group, err)
		}
	}

	if writeSmoothed {
		for _, u := range updates {
			s := u.state
			s.Mu = u.smoothedMu
			if err := repo.UpsertState(ctx, s); err != nil {
				return fmt.Errorf("write smoothed state %s/%s: %w", s.XUID, s.PlaylistGroup, err)
			}
		}
		slog.InfoContext(ctx, "TTT μ lissés écrits", "count", len(updates))
	}

	slog.InfoContext(ctx, "TTT smoothing terminé",
		"groups", len(groupResults), "players", len(histories), "source", source)
	return nil
}

func buildGroupResults(qSum, rSum map[string]float64, count map[string]int) []tttGroupResult {
	results := make([]tttGroupResult, 0, len(count))
	for group, n := range count {
		avgQ := qSum[group] / float64(n)
		avgR := rSum[group] / float64(n)
		results = append(results, tttGroupResult{
			Group:       group,
			PlayerCount: n,
			AvgQ:        avgQ,
			AvgR:        avgR,
			Tau:         math.Sqrt(avgQ),
		})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Group < results[j].Group })
	return results
}

// loadAllHistories charge tous les snapshots de player_skill_state_v2 en un seul
// round-trip, groupés par (xuid, playlist_group), dans l'ordre chronologique.
// Filtre les paires avec < 2 points (EstimateTTT exige ≥ 2 observations).
func loadAllHistories(ctx context.Context, db *sql.DB) ([]playerGroupHistory, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT xuid, playlist_group, mu, sigma, experience,
		       last_match_id, last_match_at, written_at
		FROM player_skill_state_v2
		ORDER BY xuid, playlist_group, written_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("query histories: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	histMap := make(map[string]*playerGroupHistory)
	for rows.Next() {
		var s domain.SkillV2State
		var lastMatchID sql.NullString
		var lastMatchAt sql.NullTime
		if err := rows.Scan(
			&s.XUID, &s.PlaylistGroup, &s.Mu, &s.Sigma, &s.Experience,
			&lastMatchID, &lastMatchAt, &s.WrittenAt,
		); err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		if lastMatchID.Valid {
			s.LastMatchID = &lastMatchID.String
		}
		if lastMatchAt.Valid {
			s.LastMatchAt = &lastMatchAt.Time
		}
		key := s.XUID + "|" + s.PlaylistGroup
		h, ok := histMap[key]
		if !ok {
			h = &playerGroupHistory{XUID: s.XUID, Group: s.PlaylistGroup}
			histMap[key] = h
		}
		h.States = append(h.States, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]playerGroupHistory, 0, len(histMap))
	for _, h := range histMap {
		if len(h.States) >= 2 {
			out = append(out, *h)
		}
	}
	return out, nil
}

func printTTTSmoothReport(results []tttGroupResult, writeSmoothed bool, smoothCount int) {
	var sb strings.Builder
	sb.WriteString("\n=== LUSR v2 TTT Smoothing — Rapport ===\n")
	if writeSmoothed {
		sb.WriteString(fmt.Sprintf("Mode: écriture μ lissés activée (%d joueurs/groupes)\n\n", smoothCount))
	} else {
		sb.WriteString("Mode: hyperparams seuls (--write-smoothed absent)\n\n")
	}
	for _, gr := range results {
		sb.WriteString(fmt.Sprintf("Groupe: %s (n=%d joueurs)\n", gr.Group, gr.PlayerCount))
		sb.WriteString(fmt.Sprintf("  q moyen (τ²) : %.6f\n", gr.AvgQ))
		sb.WriteString(fmt.Sprintf("  r moyen (β²) : %.4f\n", gr.AvgR))
		sb.WriteString(fmt.Sprintf("  τ estimé     : %.6f → ttt_tau_empirical\n", gr.Tau))
		sb.WriteString("\n")
	}
	fmt.Println(sb.String())
}
