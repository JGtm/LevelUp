package prestige

import (
	"context"
	"fmt"
	"log/slog"
)

// service_squads.go — CRUD du roster d'escouade (entité Squad / SquadMember).
//
// Le roster est clé xuid (cf. SquadMember, PLAN_COACH_V3_GENERATION § Identité
// d'escouade). Les writes vont dans shared_social via SquadRepo. Règle d'accès
// « membre-user, sans consentement » : toute mutation exige que requestedBy
// (player_slug) soit déjà membre-user de l'escouade — sauf CreateSquad, où le
// créateur fonde l'escouade.

// CreateSquad crée une escouade et y inscrit ses membres initiaux.
//
// req.Members doit inclure le créateur (avec son xuid + son user_id slug) : la
// résolution xuid/slug est la responsabilité de l'appelant (handler), car le
// package prestige ne connaît pas db_profiles.
func (s *service) CreateSquad(ctx context.Context, req CreateSquadRequest) (Squad, error) {
	if req.Name == "" || req.CreatedBy == "" {
		return Squad{}, fmt.Errorf("%w: name/created_by requis", ErrInvalidInput)
	}
	now := s.deps.Now()
	sq := Squad{
		ID:        newID("sq"),
		Name:      req.Name,
		CreatedBy: req.CreatedBy,
		CreatedAt: now,
	}
	if err := s.deps.Squads.Create(ctx, sq); err != nil {
		return Squad{}, fmt.Errorf("create squad: %w", err)
	}
	for _, m := range req.Members {
		if m.Xuid == "" {
			continue // membre sans xuid ignoré (clé invalide)
		}
		m.SquadID = sq.ID
		if m.JoinedAt.IsZero() {
			m.JoinedAt = now
		}
		if err := s.deps.Squads.AddMember(ctx, m); err != nil {
			slog.WarnContext(ctx, "prestige: add squad member failed",
				"squad_id", sq.ID, "xuid", m.Xuid, "err", err)
		}
	}
	return sq, nil
}

// ListSquadsForUser retourne les escouades dont userID (player_slug) est
// membre-user.
func (s *service) ListSquadsForUser(ctx context.Context, userID string) ([]Squad, error) {
	if userID == "" {
		return nil, fmt.Errorf("%w: user_id requis", ErrInvalidInput)
	}
	return s.deps.Squads.ListSquadsForUser(ctx, userID)
}

// GetSquad retourne une escouade par ID.
func (s *service) GetSquad(ctx context.Context, id string) (Squad, error) {
	if id == "" {
		return Squad{}, fmt.Errorf("%w: id requis", ErrInvalidInput)
	}
	sq, err := s.deps.Squads.Get(ctx, id)
	if err != nil {
		return Squad{}, fmt.Errorf("get squad: %w", err)
	}
	return sq, nil
}

// ListSquadMembers retourne le roster d'une escouade.
func (s *service) ListSquadMembers(ctx context.Context, squadID string) ([]SquadMember, error) {
	if squadID == "" {
		return nil, fmt.Errorf("%w: squad_id requis", ErrInvalidInput)
	}
	return s.deps.Squads.ListMembers(ctx, squadID)
}

// AddSquadMember ajoute un membre. requestedBy (player_slug) doit déjà être
// membre-user de l'escouade (règle « membre-user, sans consentement »).
func (s *service) AddSquadMember(ctx context.Context, squadID string, member SquadMember, requestedBy string) error {
	if squadID == "" || member.Xuid == "" {
		return fmt.Errorf("%w: squad_id/xuid requis", ErrInvalidInput)
	}
	if err := s.assertMemberUser(ctx, squadID, requestedBy); err != nil {
		return err
	}
	member.SquadID = squadID
	if member.JoinedAt.IsZero() {
		member.JoinedAt = s.deps.Now()
	}
	return s.deps.Squads.AddMember(ctx, member)
}

// RemoveSquadMember retire un membre (par xuid). requestedBy doit être
// membre-user de l'escouade.
func (s *service) RemoveSquadMember(ctx context.Context, squadID, xuid, requestedBy string) error {
	if squadID == "" || xuid == "" {
		return fmt.Errorf("%w: squad_id/xuid requis", ErrInvalidInput)
	}
	if err := s.assertMemberUser(ctx, squadID, requestedBy); err != nil {
		return err
	}
	return s.deps.Squads.RemoveMember(ctx, squadID, xuid)
}

// assertMemberUser vérifie que userID (player_slug) est membre-user de
// l'escouade (squad_member.user_id renseigné = utilisateur de l'app).
func (s *service) assertMemberUser(ctx context.Context, squadID, userID string) error {
	if userID == "" {
		return fmt.Errorf("%w: requested_by requis", ErrInvalidInput)
	}
	members, err := s.deps.Squads.ListMembers(ctx, squadID)
	if err != nil {
		return fmt.Errorf("list members: %w", err)
	}
	if !isMember(members, userID) {
		return fmt.Errorf("%w: not a member of squad", ErrInvalidInput)
	}
	return nil
}
