// Package service — ExplorerService : matchs communs, recherche croisée.
//
// Port Go de apps/api/app/routers/explorer.py.
package service

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// outcomeWin est le code de victoire (Halo Infinite outcome = 2).
const outcomeWin = 2

// ExplorerService orchestre les requêtes de l'Explorer.
type ExplorerService struct {
	repo port.ExplorerRepository
	xuid string
	// dataAdapter (optionnel, Phase 2 plan finition multi-titres) :
	// quand fourni, le service mesure la capability match.history pour
	// loguer une éventuelle dégradation. La bascule fonctionnelle reste
	// future car canonical.MatchSummary ne couvre pas encore le filtrage
	// Explorer (joueur commun + plage temporelle).
	dataAdapter games.TitleDataAdapter
}

// NewExplorerService crée un ExplorerService.
func NewExplorerService(repo port.ExplorerRepository, xuid string) *ExplorerService {
	return &ExplorerService{repo: repo, xuid: xuid}
}

// WithDataAdapter injecte un games.TitleDataAdapter optionnel (Phase 2 plan
// finition multi-titres). Permet de logger le statut des capabilities et
// d'amorcer la bascule vers la couche canonique.
func (s *ExplorerService) WithDataAdapter(a games.TitleDataAdapter) *ExplorerService {
	s.dataAdapter = a
	return s
}

// logCapabilityIfMissing log un warning si la capability est absente du
// DataAdapter injecté. No-op si pas de DataAdapter.
func (s *ExplorerService) logCapabilityIfMissing(ctx context.Context, cap games.CapabilityKey, caller string) {
	if s.dataAdapter == nil {
		return
	}
	if !s.dataAdapter.Capabilities().Has(cap) {
		slog.WarnContext(ctx, "capability_not_supported",
			"title_slug", s.dataAdapter.TitleSlug(),
			"capability", string(cap),
			"caller", caller,
		)
	}
}

// GetCommonMatches retourne l'historique paginé de matchs communs avec un autre
// joueur, enrichi des badges encounter (ally_plus, tough_enemy, ordinal).
// page est 1-indexé ; PageSizeCommonMatches = 20 éléments par page.
func (s *ExplorerService) GetCommonMatches(
	ctx context.Context,
	otherGamertag string,
	page int,
) (domain.ExplorerPlayerQueryResponse, error) {
	s.logCapabilityIfMissing(ctx, games.CapMatchHistory, "explorer_service.GetCommonMatches")

	otherXUID, err := s.repo.ResolveXUIDByGamertag(ctx, otherGamertag)
	if err != nil {
		return domain.ExplorerPlayerQueryResponse{},
			fmt.Errorf("ExplorerService: résolution gamertag %q: %w", otherGamertag, err)
	}

	rawMatches, err := s.repo.GetCommonMatches(ctx, s.xuid, otherXUID)
	if err != nil {
		return domain.ExplorerPlayerQueryResponse{},
			fmt.Errorf("ExplorerService: matchs communs: %w", err)
	}

	kv, err := s.repo.GetKillerVictimBetween(ctx, s.xuid, otherXUID)
	if err != nil {
		// Dégradation gracieuse : les badges tough_enemy ne seront pas calculés,
		// mais le reste de la réponse reste valide.
		slog.WarnContext(ctx, "explorer_kv_between_failed",
			"xuid1", s.xuid, "xuid2", otherXUID, "err", err)
		kv = domain.KillerVictimAggregate{}
	}

	totalCount := len(rawMatches)
	if page < 1 {
		page = 1
	}
	pageSize := domain.PageSizeCommonMatches
	offset := (page - 1) * pageSize
	end := offset + pageSize
	if end > totalCount {
		end = totalCount
	}

	var pageMatches []domain.CommonMatchRaw
	if offset < totalCount {
		pageMatches = rawMatches[offset:end]
	}

	rows := convertCommonMatches(pageMatches)

	stats := buildEncounterStats(otherXUID, otherGamertag, rawMatches, kv)
	badges := narrative.ComputeEncounterBadges(stats, totalCount)
	wins, losses := countWinsLosses(rawMatches)
	encounterStats := convertEncounterStatsToExplorer(stats, totalCount)
	activityHeatmap := analysis.ComputeActivityHeatmapFromCommonMatches(rawMatches)

	slog.DebugContext(ctx, "explorer_common_matches",
		"xuid", s.xuid, "other_xuid", otherXUID,
		"total", totalCount, "page", page, "badges", len(badges),
		"heatmap_cells", len(activityHeatmap))

	return domain.ExplorerPlayerQueryResponse{
		TargetGamertag:  otherGamertag,
		TargetXUID:      otherXUID,
		CommonMatches:   rows,
		Badges:          convertEncounterBadges(badges),
		EncounterStats:  encounterStats,
		Total:           len(rows),
		TotalCount:      totalCount,
		WinsTogether:    wins,
		LossesTogether:  losses,
		Page:            page,
		PageSize:        pageSize,
		ActivityHeatmap: activityHeatmap,
	}, nil
}

// convertEncounterStatsToExplorer projette narrative.EncounterStats → domain.ExplorerEncounterStats.
// Les compteurs ally/enemy et K/D croisés sont retournés en pointeurs pour
// distinguer "absent" de "zéro" (cohérent avec MatchEncounterRow JSON).
func convertEncounterStatsToExplorer(s narrative.EncounterStats, totalCount int) *domain.ExplorerEncounterStats {
	if totalCount == 0 {
		return nil
	}
	ally, enemy := s.AllyCount, s.EnemyCount
	kills, deaths := s.KillsDealt, s.DeathsSuffered
	out := &domain.ExplorerEncounterStats{
		CountTogether:  totalCount,
		AllyCount:      &ally,
		EnemyCount:     &enemy,
		KillsDealt:     &kills,
		DeathsSuffered: &deaths,
		WinrateAsAlly:  s.WinrateAsAlly,
		WinrateVsEnemy: s.WinrateVsEnemy,
		LastSeenAt:     s.LastSeen,
	}
	return out
}

// convertCommonMatches convertit les lignes brutes en CommonMatchRow avec
// were_teammates et outcome_label résolus.
func convertCommonMatches(raw []domain.CommonMatchRaw) []domain.CommonMatchRow {
	if len(raw) == 0 {
		return []domain.CommonMatchRow{}
	}
	result := make([]domain.CommonMatchRow, 0, len(raw))
	for i := range raw {
		r := &raw[i]
		wereTeammates := r.Player1TeamID != nil &&
			r.Player2TeamID != nil &&
			*r.Player1TeamID == *r.Player2TeamID
		result = append(result, domain.CommonMatchRow{
			MatchID:       r.MatchID,
			StartTime:     r.StartTime,
			MapUI:         r.MapUI,
			ModeUI:        r.ModeUI,
			WereTeammates: wereTeammates,
			PlayerOutcome: r.Player1Outcome,
			OutcomeLabel:  outcomeLabel(r.Player1Outcome),
			Kills:         r.Player1Kills,
			Deaths:        r.Player1Deaths,
			KDA:           r.Player1KDA,
		})
	}
	return result
}

// buildEncounterStats construit un narrative.EncounterStats depuis les données
// brutes de matchs communs et les kills croisés.
func buildEncounterStats(xuid, gamertag string, raw []domain.CommonMatchRaw, kv domain.KillerVictimAggregate) narrative.EncounterStats {
	stats := narrative.EncounterStats{
		XUID:            xuid,
		Gamertag:        gamertag,
		TotalEncounters: len(raw),
		KillsDealt:      kv.KillsDealt,
		DeathsSuffered:  kv.DeathsSuffered,
	}

	var allyWins, allyTotal, enemyWins, enemyTotal int
	for i := range raw {
		r := &raw[i]
		wereTeammates := r.Player1TeamID != nil &&
			r.Player2TeamID != nil &&
			*r.Player1TeamID == *r.Player2TeamID
		if wereTeammates {
			allyTotal++
			if r.Player1Outcome == outcomeWin {
				allyWins++
			}
		} else {
			enemyTotal++
			if r.Player1Outcome == outcomeWin {
				enemyWins++
			}
		}
		if stats.LastSeen == nil || r.StartTime.After(*stats.LastSeen) {
			t := r.StartTime
			stats.LastSeen = &t
		}
	}

	stats.AllyCount = allyTotal
	stats.EnemyCount = enemyTotal
	if allyTotal > 0 {
		wr := float64(allyWins) / float64(allyTotal)
		stats.WinrateAsAlly = &wr
	}
	if enemyTotal > 0 {
		wr := float64(enemyWins) / float64(enemyTotal)
		stats.WinrateVsEnemy = &wr
	}
	return stats
}

// countWinsLosses compte les victoires et défaites sur l'ensemble des matchs.
func countWinsLosses(raw []domain.CommonMatchRaw) (wins, losses int) {
	for i := range raw {
		switch raw[i].Player1Outcome {
		case outcomeWin:
			wins++
		default:
			losses++
		}
	}
	return
}

// convertEncounterBadges convertit les badges narrative en types domain.
func convertEncounterBadges(badges []narrative.EncounterBadge) []domain.MatchEncounterBadge {
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
