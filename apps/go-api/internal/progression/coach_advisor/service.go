package coach_advisor

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Service est la façade applicative du coach_advisor.
//
// Trois familles d'opérations :
//
//  1. Génération : GenerateProposals — pipeline matcher → synthesizer → composer
//     (implémentation différée à la Phase 7 quand toutes les briques sont en place).
//  2. Action joueur : ListProposals, AcceptProposal (Phase 7), DismissProposal.
//  3. Maintenance : ObsoletePendingForAxis — appelé quand un challenge accepté
//     coach se complète (cf. ADR 0020 §3 obsolescence).
//
// Toutes les méthodes sont scopées par (userID, titleSlug) explicites — le
// Service n'a pas d'état lié à un joueur (stateless).
type Service interface {
	// ListProposals retourne les proposals d'un joueur, optionnellement filtrées
	// par status (vide = toutes). Tri par created_at DESC.
	ListProposals(ctx context.Context, userID, titleSlug string, status ProposalStatus) ([]Proposal, error)

	// DismissProposal marque une proposal comme dismissed (action explicite
	// du joueur). Idempotent si déjà résolue (no-op).
	DismissProposal(ctx context.Context, id string) error

	// ObsoletePendingForAxis marque comme obsoleted toutes les proposals
	// pending qui ciblent ce radar_axis. Appelé quand un challenge issu d'une
	// proposal acceptée se complète sur le même axis — le coach se tait sur
	// cet axe le temps que la baseline du joueur bouge (cf. ADR 0020 §3).
	//
	// Retourne le nombre de proposals obsoletées (best-effort : les erreurs
	// individuelles sont loggées sans interrompre le batch).
	ObsoletePendingForAxis(ctx context.Context, userID, titleSlug, axis string) (int, error)
}

// service est l'implémentation par défaut. Constructeur via NewService.
type service struct {
	repo Repo
	now  func() time.Time // injectable pour tests
}

// NewService construit un Service avec le Repo donné. Utilise time.Now par défaut.
func NewService(repo Repo) Service {
	return &service{repo: repo, now: func() time.Time { return time.Now().UTC() }}
}

// ListProposals délègue au Repo.
func (s *service) ListProposals(ctx context.Context, userID, titleSlug string, status ProposalStatus) ([]Proposal, error) {
	if userID == "" || titleSlug == "" {
		return nil, fmt.Errorf("coach_advisor.ListProposals: userID and titleSlug required")
	}
	props, err := s.repo.ListByUser(ctx, userID, titleSlug, status)
	if err != nil {
		return nil, fmt.Errorf("coach_advisor.ListProposals: %w", err)
	}
	return props, nil
}

// DismissProposal marque comme dismissed. Si la proposal n'est plus pending,
// l'opération est silencieuse (no-op) car DuckDB UPDATE sans WHERE status
// n'aurait pas de garde — le repo lui-même ne filtre pas pour Dismiss
// (action explicite joueur prime sur supersession).
func (s *service) DismissProposal(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("coach_advisor.DismissProposal: id required")
	}
	if err := s.repo.MarkDismissed(ctx, id, s.now()); err != nil {
		return fmt.Errorf("coach_advisor.DismissProposal: %w", err)
	}
	return nil
}

// ObsoletePendingForAxis : best-effort batch obsolescence.
func (s *service) ObsoletePendingForAxis(ctx context.Context, userID, titleSlug, axis string) (int, error) {
	if userID == "" || titleSlug == "" || axis == "" {
		return 0, fmt.Errorf("coach_advisor.ObsoletePendingForAxis: userID, titleSlug, axis required")
	}
	pending, err := s.repo.ListPendingByAxis(ctx, userID, titleSlug, axis)
	if err != nil {
		return 0, fmt.Errorf("coach_advisor.ObsoletePendingForAxis: list: %w", err)
	}
	now := s.now()
	obsoleted := 0
	for _, p := range pending {
		if err := s.repo.MarkObsoleted(ctx, p.ID, now); err != nil {
			slog.WarnContext(ctx, "coach_advisor: obsolete proposal failed",
				"err", err, "id", p.ID, "user", userID, "titleSlug", titleSlug, "axis", axis)
			continue
		}
		obsoleted++
	}
	slog.InfoContext(ctx, "coach_advisor: pending obsoleted by axis completion",
		"count", obsoleted, "user", userID, "titleSlug", titleSlug, "axis", axis)
	return obsoleted, nil
}
