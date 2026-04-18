// Package service — compare_service.go : comparaison joueur vs joueur.
//
// Sprint 54 C : CompareService avec chargement parallèle via errgroup.
// Joueur A : données DuckDB (CompareRepository).
// Joueur B : Waypoint (PlayerStatsProvider) + fallback DuckDB si connu localement.
package service

import (
	"context"
	"fmt"
	"math"

	"golang.org/x/sync/errgroup"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// CompareService orchestre la comparaison joueur A vs joueur B.
type CompareService struct {
	repo      port.CompareRepository
	provider  port.PlayerStatsProvider
	xuidA     string
	titleSlug string
}

// NewCompareService crée un CompareService.
func NewCompareService(repo port.CompareRepository, provider port.PlayerStatsProvider, xuidA, titleSlug string) *CompareService {
	return &CompareService{repo: repo, provider: provider, xuidA: xuidA, titleSlug: titleSlug}
}

// GetPage construit la réponse de comparaison en chargeant les stats en parallèle.
func (s *CompareService) GetPage(ctx context.Context, req domain.CompareRequest) (domain.CompareResponse, error) {
	if err := req.Validate(); err != nil {
		return domain.CompareResponse{}, fmt.Errorf("CompareService: %w", err)
	}

	var statsA, statsB *domain.NormalizedPlayerStats
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		statsA, err = s.repo.GetLocalStats(gctx, s.xuidA, s.titleSlug)
		return err
	})

	g.Go(func() error {
		// Tenter d'abord un resolve local du gamertag B.
		xuidB, _ := s.repo.ResolveXUID(gctx, req.TargetGamertag)
		if xuidB != "" {
			// Joueur B connu localement → DuckDB.
			local, err := s.repo.GetLocalStats(gctx, xuidB, s.titleSlug)
			if err == nil && local != nil {
				statsB = local
				return nil
			}
		}
		// Fallback : Waypoint.
		remote, err := s.provider.FetchRemoteStats(gctx, req.TargetGamertag, s.titleSlug)
		if err != nil {
			return fmt.Errorf("CompareService.GetPage: stats joueur B introuvables: %w", err)
		}
		statsB = remote
		return nil
	})

	if err := g.Wait(); err != nil {
		return domain.CompareResponse{}, err
	}

	metrics := buildMetrics(*statsA, *statsB)
	return domain.CompareResponse{
		PlayerA:   *statsA,
		PlayerB:   *statsB,
		Metrics:   metrics,
		TitleSlug: s.titleSlug,
	}, nil
}

// buildMetrics construit les 12 CompareMetricRows à partir des deux stats.
func buildMetrics(a, b domain.NormalizedPlayerStats) []domain.CompareMetricRow {
	type metricDef struct {
		key   string
		label string
		va    float64
		vb    float64
	}
	defs := []metricDef{
		{"win_rate", "Taux de victoire", a.WinRate * 100, b.WinRate * 100},
		{"kda", "KDA", a.KDA, b.KDA},
		{"kdr", "K/D", a.KDR, b.KDR},
		{"kills_per_game", "Kills / partie", a.KillsPerGame, b.KillsPerGame},
		{"deaths_per_game", "Morts / partie", a.DeathsPerGame, b.DeathsPerGame},
		{"assists_per_game", "Assists / partie", a.AssistsPerGame, b.AssistsPerGame},
		{"accuracy", "Précision", a.Accuracy * 100, b.Accuracy * 100},
		{"damage_per_game", "Dégâts / partie", a.DamagePerGame, b.DamagePerGame},
		{"matches", "Matchs joués", float64(a.Matches), float64(b.Matches)},
		{"csr_current", "CSR actuel", float64(a.CSRCurrent), float64(b.CSRCurrent)},
		{"csr_best", "CSR meilleur", float64(a.CSRBest), float64(b.CSRBest)},
		{"career_rank", "Rang Carrière", float64(a.CareerRank), float64(b.CareerRank)},
	}

	rows := make([]domain.CompareMetricRow, 0, len(defs))
	for _, d := range defs {
		delta := d.vb - d.va
		winner := "tie"
		// Pour deaths_per_game : moins = mieux.
		if d.key == "deaths_per_game" {
			if d.va < d.vb {
				winner = "a"
			} else if d.vb < d.va {
				winner = "b"
			}
		} else {
			if math.Abs(delta) > 0.001 {
				if d.va > d.vb {
					winner = "a"
				} else {
					winner = "b"
				}
			}
		}
		rows = append(rows, domain.CompareMetricRow{
			Metric:  d.key,
			LabelFR: d.label,
			ValueA:  d.va,
			ValueB:  d.vb,
			Delta:   delta,
			Winner:  winner,
		})
	}
	return rows
}
