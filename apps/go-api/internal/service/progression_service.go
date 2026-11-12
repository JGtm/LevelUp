// Package service — progression_service.go : lecture de la progression d'un joueur (séries, records,
// jalons) pour la couche HTTP.
package service

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/port"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/progression/streaks"
)

// ProgressionService implémente port.ProgressionService au-dessus d'un dépôt résolu par joueur.
type ProgressionService struct {
	resolve port.ProgressionStoreResolver
}

var _ port.ProgressionService = (*ProgressionService)(nil)

// NewProgressionService construit le service ; `resolve` donne le dépôt d'un joueur.
func NewProgressionService(resolve port.ProgressionStoreResolver) *ProgressionService {
	return &ProgressionService{resolve: resolve}
}

// store résout le joueur ou rend un *port.ProgressionError de nature ProgErrPlayerNotFound.
func (s *ProgressionService) store(ctx context.Context, playerSlug string) (port.ProgressionStore, error) {
	st, err := s.resolve(ctx, playerSlug)
	if err != nil {
		return nil, &port.ProgressionError{Kind: port.ProgErrPlayerNotFound, Err: err}
	}
	return st, nil
}

// Streaks rend les séries du joueur pour le titre.
func (s *ProgressionService) Streaks(ctx context.Context, playerSlug, titleSlug string) ([]streaks.Streak, error) {
	st, err := s.store(ctx, playerSlug)
	if err != nil {
		return nil, err
	}
	list, err := st.ListStreaks(ctx, titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "progression: list streaks", "err", err)
		return nil, &port.ProgressionError{Kind: port.ProgErrListStreaks, Err: err}
	}
	return list, nil
}

// Records rend les records personnels courants et l'historique récent. Une métrique hors catalogue
// (ex. « best_kda » ancienne) n'est pas servie : elle n'a ni libellé ni bornes connues, et la
// dégradation est journalisée.
func (s *ProgressionService) Records(ctx context.Context, playerSlug, titleSlug string, historyLimit int) (port.PlayerRecords, error) {
	st, err := s.store(ctx, playerSlug)
	if err != nil {
		return port.PlayerRecords{}, err
	}
	pbList, err := st.ListPersonalBests(ctx)
	if err != nil {
		slog.WarnContext(ctx, "progression: list PBs", "err", err)
		return port.PlayerRecords{}, &port.ProgressionError{Kind: port.ProgErrListPersonalBests, Err: err}
	}
	histList, err := st.ListRecordHistory(ctx, titleSlug, historyLimit)
	if err != nil {
		slog.WarnContext(ctx, "progression: list record history", "err", err)
		return port.PlayerRecords{}, &port.ProgressionError{Kind: port.ProgErrListRecordHistory, Err: err}
	}
	out := port.PlayerRecords{
		PersonalBests: make([]records.PersonalRecord, 0, len(pbList)),
		History:       make([]records.RecordHistory, 0, len(histList)),
	}
	for _, pb := range pbList {
		if !records.IsKnownMetric(pb.Metric) {
			slog.WarnContext(ctx, "progression: PB métrique hors catalogue non servie",
				"metric", pb.Metric, "period", pb.Period)
			continue
		}
		out.PersonalBests = append(out.PersonalBests, pb)
	}
	for _, h := range histList {
		if !records.IsKnownMetric(h.Metric) {
			slog.WarnContext(ctx, "progression: record history métrique hors catalogue non servi",
				"metric", h.Metric, "period", h.Period)
			continue
		}
		out.History = append(out.History, h)
	}
	return out, nil
}

// Milestones joint le catalogue du titre aux jalons franchis par le joueur. Un jalon franchi dont la
// date est inconnue (earned_at NULL, rattrapage non dérivable) reste franchi, sans date : on ne sert pas
// de date fausse.
func (s *ProgressionService) Milestones(ctx context.Context, playerSlug, titleSlug string) ([]port.MilestoneStatus, error) {
	st, err := s.store(ctx, playerSlug)
	if err != nil {
		return nil, err
	}
	catalog, err := st.ListMilestoneCatalog(ctx, titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "progression: list catalog", "err", err)
		return nil, &port.ProgressionError{Kind: port.ProgErrListCatalog, Err: err}
	}
	earnedList, err := st.ListMilestonesEarned(ctx, titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "progression: list earned", "err", err)
		return nil, &port.ProgressionError{Kind: port.ProgErrListEarned, Err: err}
	}
	earnedByID := make(map[string]time.Time, len(earnedList))
	for _, e := range earnedList {
		earnedByID[e.MilestoneID] = e.EarnedAt
	}
	items := make([]port.MilestoneStatus, 0, len(catalog))
	for _, c := range catalog {
		item := port.MilestoneStatus{Entry: c}
		if at, ok := earnedByID[c.ID]; ok {
			item.Earned = true
			if !at.IsZero() {
				t := at
				item.EarnedAt = &t
			}
		}
		items = append(items, item)
	}
	return items, nil
}
