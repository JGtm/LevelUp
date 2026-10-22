// Package service — SessionsService : calcul des sessions de jeu.
//
// Port Go de src/analysis/sessions.py + src/data/services/timeseries_service.py.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
)

// SessionsService calcule et retourne les sessions de jeu d'un joueur.
type SessionsService struct {
	repo port.SessionsRepository
	// rule rend la règle de découpage du titre (settings.SessionOptionsLoader).
	rule func(context.Context) domain.SessionComputeOptions
	// friendGamertags rend les amis du joueur (nil : aucun ami).
	friendGamertags func(context.Context) []string
}

// NewSessionsService crée un SessionsService.
func NewSessionsService(repo port.SessionsRepository) *SessionsService {
	return &SessionsService{repo: repo}
}

// WithSessionRule branche la règle de découpage du titre et les amis du joueur : ceux
// qu'appliquent les écrivains de session_id. Sans règle, SessionOptions rend la règle par
// défaut (journalisée par settings.SessionOptions).
func (s *SessionsService) WithSessionRule(
	rule func(context.Context) domain.SessionComputeOptions,
	friendGamertags func(context.Context) []string,
) *SessionsService {
	s.rule = rule
	s.friendGamertags = friendGamertags
	return s
}

// SessionOptions rend la règle de découpage du joueur, amis résolus en xuids. Un échec de
// résolution des amis est journalisé ; la règle s'applique alors sans amis.
func (s *SessionsService) SessionOptions(ctx context.Context) domain.SessionComputeOptions {
	var opts domain.SessionComputeOptions
	if s.rule != nil {
		opts = s.rule(ctx)
	} else {
		opts = settings_platform.SessionOptions(ctx, nil, "")
	}
	if s.friendGamertags == nil {
		return opts
	}
	gamertags := s.friendGamertags(ctx)
	if len(gamertags) == 0 {
		return opts
	}
	xuids, err := s.repo.LookupFriendXUIDs(ctx, gamertags)
	if err != nil {
		slog.WarnContext(ctx, "sessions: amis non résolus, découpage sans amis", "err", err)
		return opts
	}
	opts.FriendsXUIDs = xuids
	return opts
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
