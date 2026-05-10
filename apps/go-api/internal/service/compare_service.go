// Package service — compare_service.go : comparaison joueur vs joueur.
//
// Sprint 54 C : CompareService avec chargement parallèle via errgroup.
// Joueur A : données DuckDB (CompareRepository) + CSR depuis Waypoint skill endpoint.
// Joueur B : Waypoint (PlayerStatsProvider) + fallback DuckDB si connu localement.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"

	"golang.org/x/sync/errgroup"
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
	var ath *domain.PlayerATH
	var csrCurrentA int
	var xuidBResolved string
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		statsA, err = s.repo.GetLocalStats(gctx, s.xuidA, s.titleSlug)
		if err != nil {
			return err
		}
		statsA.IsLocal = true
		// ATH depuis stats.duckdb — best-effort (non-fatal si absent).
		var athErr error
		ath, athErr = s.repo.GetPlayerATH(gctx)
		if athErr != nil {
			slog.DebugContext(gctx, "CompareService: ATH non disponible (best-effort)", "xuid", s.xuidA, "err", athErr)
		}
		// Arme favorite depuis shared.weapon_kills — best-effort.
		var wErr error
		statsA.FavoriteWeapon, wErr = s.repo.GetFavoriteWeapon(gctx, s.xuidA)
		if wErr != nil {
			slog.DebugContext(gctx, "CompareService: arme favorite non disponible (best-effort)", "xuid", s.xuidA, "err", wErr)
		}
		// CSR depuis match skill Waypoint — best-effort.
		if matchID, _ := s.repo.GetRecentRankedMatchID(gctx, s.xuidA); matchID != "" {
			var csrErr error
			csrCurrentA, csrErr = s.provider.FetchCSRFromMatch(gctx, matchID, s.xuidA)
			if csrErr != nil {
				slog.DebugContext(gctx, "CompareService: CSR joueur A indisponible (best-effort)", "xuid", s.xuidA, "match_id", matchID, "err", csrErr)
			} else {
				slog.DebugContext(gctx, "CompareService: CSR joueur A depuis match skill", "xuid", s.xuidA, "csr_current", csrCurrentA)
			}
		}
		return nil
	})

	g.Go(func() error {
		xuidB, _ := s.repo.ResolveXUID(gctx, req.TargetGamertag)
		xuidBResolved = xuidB
		if xuidB != "" {
			local, err := s.repo.GetLocalStats(gctx, xuidB, s.titleSlug)
			if err == nil && local != nil {
				local.IsLocal = true
				local.FavoriteWeapon, _ = s.repo.GetFavoriteWeapon(gctx, xuidB)

				// ATH depuis sa propre stats.duckdb — best-effort (pool lookup).
				if athB, athErr := s.repo.GetPlayerATHFor(gctx, local.Gamertag, s.titleSlug); athErr != nil {
					slog.DebugContext(gctx, "CompareService: ATH joueur B non disponible (best-effort)", "gamertag", local.Gamertag, "err", athErr)
				} else if athB != nil {
					local.CareerRank = athB.CareerRank
					local.PerfATH = athB.PerfATH
					local.LusrATH = athB.LusrATH
					slog.DebugContext(gctx, "CompareService: ATH joueur B depuis pool (hors CSR)", "gamertag", local.Gamertag)
				}

				// CSR joueur B depuis match skill Waypoint — best-effort.
				if matchID, _ := s.repo.GetRecentRankedMatchID(gctx, xuidB); matchID != "" {
					if csrCurrent, csrErr := s.provider.FetchCSRFromMatch(gctx, matchID, xuidB); csrErr != nil {
						slog.DebugContext(gctx, "CompareService: CSR joueur B indisponible (best-effort)", "gamertag", local.Gamertag, "err", csrErr)
					} else {
						local.CSRCurrent = csrCurrent
						slog.DebugContext(gctx, "CompareService: CSR joueur B depuis match skill", "gamertag", local.Gamertag, "csr_current", csrCurrent)
					}
				}

				statsB = local
				slog.DebugContext(gctx, "CompareService: joueur B résolu localement", "gamertag", req.TargetGamertag, "xuid", xuidB)
				return nil
			}
		}
		// Fallback : Waypoint.
		slog.DebugContext(gctx, "CompareService: joueur B non local, fallback Waypoint", "gamertag", req.TargetGamertag)
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

	// Merge ATH dans statsA (hors CSR — CSR depuis Waypoint uniquement).
	if ath != nil {
		statsA.CareerRank = ath.CareerRank
		statsA.PerfATH = ath.PerfATH
		statsA.LusrATH = ath.LusrATH
	}
	statsA.CSRCurrent = csrCurrentA
	slog.DebugContext(ctx, "CompareService: CSR final joueur A", "gamertag", statsA.Gamertag,
		"csr_current", statsA.CSRCurrent)

	metrics := buildMetrics(*statsA, *statsB)
	resp := domain.CompareResponse{
		PlayerA:   *statsA,
		PlayerB:   *statsB,
		Metrics:   metrics,
		TitleSlug: s.titleSlug,
	}

	// Badges de rencontres historiques — best-effort, ne bloque pas la réponse.
	if xuidBResolved != "" {
		if enc, err := s.repo.GetEncounterStats(ctx, s.xuidA, xuidBResolved); err == nil && enc != nil && enc.TotalEncounters > 0 {
			stats := narrative.EncounterStats{
				XUID:            xuidBResolved,
				Gamertag:        statsB.Gamertag,
				TotalEncounters: enc.TotalEncounters,
				AllyCount:       enc.AllyCount,
				EnemyCount:      enc.EnemyCount,
				WinrateAsAlly:   enc.WinrateAsAlly,
				WinrateVsEnemy:  enc.WinrateVsEnemy,
				KillsDealt:      enc.KillsDealt,
				DeathsSuffered:  enc.DeathsSuffered,
			}
			resp.EncounterBadges = convertNarrativeBadgesCompare(narrative.ComputeEncounterBadges(stats, enc.TotalEncounters))
			slog.DebugContext(ctx, "CompareService: encounter badges calculés",
				"gamertag_b", statsB.Gamertag, "total", enc.TotalEncounters, "badges", len(resp.EncounterBadges))
		}
	}

	return resp, nil
}

func convertNarrativeBadgesCompare(badges []narrative.EncounterBadge) []domain.MatchEncounterBadge {
	if len(badges) == 0 {
		return nil
	}
	result := make([]domain.MatchEncounterBadge, 0, len(badges))
	for _, b := range badges {
		result = append(result, domain.MatchEncounterBadge{
			Kind:       string(b.Kind),
			LabelKey:   b.LabelKey,
			ColorToken: b.ColorToken,
			Detail:     b.Detail,
		})
	}
	return result
}

// buildMetrics construit les CompareMetricRows à partir des deux stats normalisées.
func buildMetrics(a, b domain.NormalizedPlayerStats) []domain.CompareMetricRow {
	// Rendement = dégâts / frag / 225 — moins = plus efficace.
	rendementA, rendementB := computeRendement(a), computeRendement(b)
	// Résistance = dégâts subis / mort / 225 — plus = plus résistant.
	resistanceA, resistanceB := computeResistance(a), computeResistance(b)

	type metricDef struct {
		key          string
		label        string
		va           float64
		vb           float64
		lessIsBetter bool
	}
	defs := []metricDef{
		// win_rate et accuracy envoyés en fraction 0..1 — le frontend multiplie par 100 à l'affichage.
		{"win_rate", "Taux de victoire", a.WinRate, b.WinRate, false},
		{"kda", "KDA", a.KDA, b.KDA, false},
		{"kdr", "K/D", a.KDR, b.KDR, false},
		{"kills_per_game", "Frags / partie", a.KillsPerGame, b.KillsPerGame, false},
		{"deaths_per_game", "Morts / partie", a.DeathsPerGame, b.DeathsPerGame, true},
		{"assists_per_game", "Assistances / partie", a.AssistsPerGame, b.AssistsPerGame, false},
		{"accuracy", "Précision", a.Accuracy, b.Accuracy, false},
		{"damage_per_game", "Dégâts / partie", a.DamagePerGame, b.DamagePerGame, false},
		{"rendement", "Rendement", rendementA, rendementB, true},
		{"damage_taken_per_game", "Dégâts subis / partie", a.DamageTakenPerGame, b.DamageTakenPerGame, true},
		{"resistance", "Résistance", resistanceA, resistanceB, false},
		{"perfect_kills_per_game", "Tirs parfaits / partie", a.PerfectKillsPerGame, b.PerfectKillsPerGame, false},
		{"max_killing_spree", "Folie meurtrière max", float64(a.MaxKillingSpree), float64(b.MaxKillingSpree), false},
		{"avg_life_secs", "Survie moy. / partie", a.AvgLifeSecs, b.AvgLifeSecs, false},
		{"headshot_kills_per_game", "Headshots / partie", a.HeadshotKillsPerGame, b.HeadshotKillsPerGame, false},
		{"matches", "Parties", float64(a.Matches), float64(b.Matches), false},
		{"csr_current", "CSR actuel", float64(a.CSRCurrent), float64(b.CSRCurrent), false},
		{"csr_best", "CSR meilleur", float64(a.CSRBest), float64(b.CSRBest), false},
		{"career_rank", "Rang Carrière", float64(a.CareerRank), float64(b.CareerRank), false},
		{"perf_ath", "Perf. record", a.PerfATH, b.PerfATH, false},
		{"lusr_ath", "LUSR record", a.LusrATH, b.LusrATH, false},
	}

	// SampleSizeB non nul uniquement si B est un joueur local croisé.
	sampleSizeB := 0
	if b.IsLocal {
		sampleSizeB = b.Matches
	}

	rows := make([]domain.CompareMetricRow, 0, len(defs))
	for _, d := range defs {
		if d.va == 0 && d.vb == 0 {
			continue // pas de données pour les deux joueurs — masquer plutôt qu'afficher "0 vs 0"
		}
		delta := d.vb - d.va
		winner := computeWinner(d.va, d.vb, d.lessIsBetter)
		rows = append(rows, domain.CompareMetricRow{
			Metric:      d.key,
			LabelFR:     d.label,
			ValueA:      d.va,
			ValueB:      d.vb,
			Delta:       delta,
			Winner:      winner,
			SampleSizeB: sampleSizeB,
		})
	}
	return rows
}

func computeWinner(va, vb float64, lessIsBetter bool) string {
	const eps = 0.001
	if lessIsBetter {
		if va < vb-eps {
			return "a"
		}
		if vb < va-eps {
			return "b"
		}
		return "tie"
	}
	if math.Abs(va-vb) <= eps {
		return "tie"
	}
	if va > vb {
		return "a"
	}
	return "b"
}

func computeRendement(s domain.NormalizedPlayerStats) float64 {
	if s.KillsPerGame <= 0 {
		return 0
	}
	return s.DamagePerGame / s.KillsPerGame / 225.0
}

func computeResistance(s domain.NormalizedPlayerStats) float64 {
	if s.DeathsPerGame <= 0 {
		return 0
	}
	return s.DamageTakenPerGame / s.DeathsPerGame / 225.0
}
