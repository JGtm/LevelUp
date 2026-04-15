// Package service — SessionsService : calcul des sessions de jeu.
//
// Port Go de src/analysis/sessions.py + src/data/services/timeseries_service.py.
package service

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// SessionsService calcule et retourne les sessions de jeu d'un joueur.
type SessionsService struct {
	repo port.SessionsRepository
}

// NewSessionsService crée un SessionsService.
func NewSessionsService(repo port.SessionsRepository) *SessionsService {
	return &SessionsService{repo: repo}
}

// GetSessions charge les matchs et calcule les sessions selon les options fournies.
func (s *SessionsService) GetSessions(
	ctx context.Context,
	opts domain.SessionComputeOptions,
) (domain.SessionsResponse, error) {
	rows, err := s.repo.LoadSessionMatches(ctx)
	if err != nil {
		return domain.SessionsResponse{}, fmt.Errorf("SessionsService.GetSessions: %w", err)
	}
	if len(rows) == 0 {
		return domain.SessionsResponse{
			Sessions:    []domain.SessionGroup{},
			Assignments: []domain.SessionAssignment{},
		}, nil
	}

	// Calcul des sessions selon le mode.
	var assignments []domain.SessionAssignment
	switch opts.Mode {
	case domain.SessionModeGap:
		assignments = analysis.ComputeSessions(rows, opts.GapMinutes)
	default:
		assignments = analysis.ComputeSessionsWithContext(rows, opts)
	}

	// Construction des groupes + labels.
	groups := analysis.BuildSessionGroups(rows, assignments)
	assignments = analysis.MergeSessionLabels(assignments, groups)

	// Calcul du bucket info sur la plage totale.
	totalDays := computeTotalDays(rows)
	bucketInfo := analysis.GetBucketInfo(totalDays)

	return domain.SessionsResponse{
		Sessions:    groups,
		Assignments: assignments,
		BucketInfo:  bucketInfo,
		TotalDays:   totalDays,
	}, nil
}

// computeTotalDays calcule la plage en jours entre le premier et le dernier match.
func computeTotalDays(rows []domain.SessionMatchRow) float64 {
	if len(rows) == 0 {
		return 0
	}
	var first, last time.Time
	for i, r := range rows {
		if i == 0 {
			first = r.StartTime
			last = r.StartTime
			continue
		}
		if r.StartTime.Before(first) {
			first = r.StartTime
		}
		if r.StartTime.After(last) {
			last = r.StartTime
		}
	}
	return last.Sub(first).Hours() / 24.0
}
