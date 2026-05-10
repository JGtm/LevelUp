// Package service — AchievementsService : page Achievements Xbox.
//
// Le service merge deux sources DuckDB :
//   - port.AchievementsRepository    → player_achievements (stats.duckdb par joueur)
//   - port.MetadataAchievementsRepo  → xbox_achievement_definitions (metadata.duckdb)
//
// Les deux sources sont jointes côté Go par achievement_id (Volume très limité,
// quelques centaines au max — pas de besoin d'ATTACH SQL cross-DB).
package service

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

// AchievementsService construit la page Achievements pour un joueur.
type AchievementsService struct {
	repo      port.AchievementsRepository
	metaRepo  port.MetadataAchievementsRepository
	titleSlug string
}

// NewAchievementsService crée un AchievementsService.
//
//   - repo      : lit player_achievements (stats.duckdb du joueur)
//   - metaRepo  : lit xbox_achievement_definitions (metadata.duckdb)
func NewAchievementsService(
	repo port.AchievementsRepository,
	metaRepo port.MetadataAchievementsRepository,
) *AchievementsService {
	return &AchievementsService{repo: repo, metaRepo: metaRepo}
}

// WithTitleSlug configure le slug du titre courant (logging/observabilité).
func (s *AchievementsService) WithTitleSlug(slug string) *AchievementsService {
	s.titleSlug = slug
	return s
}

// GetAchievementsPage retourne la page complète : summary + liste fusionnée.
//
// Comportement aux limites :
//   - Définitions vides (backfill jamais lancé) → réponse {summary zeros, achievements: []}
//   - Player rows vides (joueur neuf) → toutes les définitions retournées en locked
//   - Player row sans définition correspondante (orphelin) → ignorée silencieusement
//
// Tri : unlocked d'abord (UnlockedAt DESC), puis locked par gamerscore DESC,
// puis achievement_id ASC pour stabilité.
func (s *AchievementsService) GetAchievementsPage(ctx context.Context) (domain.AchievementsPageResponse, error) {
	defer func(start time.Time) {
		observability.RecordDurationMS("achievements_get_page", time.Since(start).Milliseconds())
	}(time.Now())

	defs, err := s.metaRepo.GetAchievementDefinitions(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "achievements service: load definitions failed",
			"err", err, "titleSlug", s.titleSlug)
		return domain.AchievementsPageResponse{}, err
	}

	playerRows, err := s.repo.GetPlayerAchievements(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "achievements service: load player rows failed",
			"err", err, "titleSlug", s.titleSlug)
		return domain.AchievementsPageResponse{}, err
	}

	playerByID := make(map[string]domain.PlayerAchievementRow, len(playerRows))
	for _, r := range playerRows {
		playerByID[r.AchievementID] = r
	}

	entries := make([]domain.AchievementEntry, 0, len(defs))
	summary := domain.AchievementsSummary{}
	for _, d := range defs {
		entry := buildAchievementEntry(d, playerByID[d.AchievementID])
		entries = append(entries, entry)
		summary.TotalCount++
		summary.TotalGamerscore += d.Gamerscore
		if entry.Unlocked {
			summary.UnlockedCount++
			summary.EarnedGamerscore += d.Gamerscore
		}
	}
	summary.CompletionPct = computeCompletionPct(summary.UnlockedCount, summary.TotalCount)

	sortAchievementEntries(entries)

	slog.DebugContext(ctx, "achievements service: page built",
		"titleSlug", s.titleSlug,
		"definitions", len(defs),
		"player_rows", len(playerRows),
		"unlocked", summary.UnlockedCount,
		"completion_pct", summary.CompletionPct,
	)

	return domain.AchievementsPageResponse{
		Summary:      summary,
		Achievements: entries,
	}, nil
}

// buildAchievementEntry fusionne une définition et la ligne player (peut être zero-value
// si le joueur n'a pas encore syncé cet achievement).
func buildAchievementEntry(d domain.AchievementDefinitionRow, p domain.PlayerAchievementRow) domain.AchievementEntry {
	return domain.AchievementEntry{
		AchievementID:   d.AchievementID,
		NameEN:          d.NameEN,
		NameFR:          d.NameFR,
		DescriptionEN:   d.DescriptionEN,
		DescriptionFR:   d.DescriptionFR,
		LockedDescEN:    d.LockedDescEN,
		LockedDescFR:    d.LockedDescFR,
		Gamerscore:      d.Gamerscore,
		ImageURL:        d.ImageURL,
		IsSecret:        d.IsSecret,
		RarityCategory:  d.RarityCategory,
		RarityPercent:   d.RarityPercent,
		Unlocked:        p.Unlocked,
		UnlockedAt:      p.UnlockedAt,
		CurrentProgress: p.CurrentProgress,
		TargetProgress:  p.TargetProgress,
		XboxTitleID:     d.XboxTitleID,
	}
}

// computeCompletionPct calcule le pourcentage de complétion (0..100, arrondi à 0.1).
func computeCompletionPct(unlocked, total int) float64 {
	if total <= 0 {
		return 0
	}
	pct := float64(unlocked) / float64(total) * 100
	return float64(int(pct*10+0.5)) / 10
}

// sortAchievementEntries applique le tri métier : locked en premier (par
// gamerscore DESC, priorité aux plus rémunérateurs), puis unlocked (par
// UnlockedAt DESC, les plus récents en bas). Tri stable par achievement_id ASC.
func sortAchievementEntries(entries []domain.AchievementEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.Unlocked != b.Unlocked {
			return !a.Unlocked // locked avant unlocked
		}
		if !a.Unlocked {
			if a.Gamerscore != b.Gamerscore {
				return a.Gamerscore > b.Gamerscore
			}
			return a.AchievementID < b.AchievementID
		}
		// les deux sont unlocked : plus récent en dernier (date ASC)
		ai := a.UnlockedAt
		bi := b.UnlockedAt
		switch {
		case ai != nil && bi != nil && !ai.Equal(*bi):
			return ai.Before(*bi)
		case ai == nil && bi != nil:
			return true
		case ai != nil && bi == nil:
			return false
		}
		return a.AchievementID < b.AchievementID
	})
}
