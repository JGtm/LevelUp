package prestige

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// service_arcs_squads.go — méthodes Service pour arcs libres et défis d'escouade.
//
// Séparé de service.go pour respecter le seuil 500 L de CLAUDE.md.

// ---------- Arcs ----------

// CreateArc crée un arc libre (non preset) appartenant au joueur.
func (s *service) CreateArc(ctx context.Context, req CreateArcRequest) (Arc, error) {
	if req.UserID == "" || req.TitleSlug == "" || strings.TrimSpace(req.Title) == "" {
		return Arc{}, fmt.Errorf("%w: user_id/title_slug/title requis", ErrInvalidInput)
	}
	now := s.deps.Now()
	a := Arc{
		ID:          newID("arc"),
		UserID:      req.UserID,
		TitleSlug:   req.TitleSlug,
		Title:       req.Title,
		Description: req.Description,
		IsPreset:    false,
		CreatedAt:   now,
	}
	if err := s.deps.Arcs.Create(ctx, a); err != nil {
		return Arc{}, fmt.Errorf("create arc: %w", err)
	}
	slog.InfoContext(ctx, "prestige: arc created",
		"arc_id", a.ID, "user_id", a.UserID, "title", a.Title)
	return a, nil
}

// ListArcs liste les arcs d'un joueur sur un titre donné (preset + libres),
// enrichis avec la récompense PP (objectifs cumulés + bonus de complétion).
func (s *service) ListArcs(ctx context.Context, userID, titleSlug string) ([]Arc, error) {
	if userID == "" || titleSlug == "" {
		return nil, fmt.Errorf("%w: user_id/title_slug requis", ErrInvalidInput)
	}
	arcs, err := s.deps.Arcs.ListByUser(ctx, userID, titleSlug)
	if err != nil {
		return nil, err
	}
	for i := range arcs {
		arcs[i] = s.enrichArcReward(ctx, arcs[i])
	}
	return arcs, nil
}

// GetArc retourne un arc par ID, enrichi avec sa récompense PP.
func (s *service) GetArc(ctx context.Context, id string) (Arc, error) {
	a, err := s.deps.Arcs.Get(ctx, id)
	if err != nil {
		if errors.Is(err, ErrArcNotFound) {
			return Arc{}, ErrArcNotFound
		}
		return Arc{}, fmt.Errorf("get arc: %w", err)
	}
	return s.enrichArcReward(ctx, a), nil
}

// enrichArcReward calcule ObjectivesPP (somme des PP des objectifs) et
// CompletionBonusPP (bonus de complétion) pour l'affichage. Best-effort :
// si la liste des objectifs échoue, l'arc est retourné non enrichi.
func (s *service) enrichArcReward(ctx context.Context, a Arc) Arc {
	objectives, err := s.deps.Challenges.List(ctx, ChallengeFilter{
		UserID: a.UserID, TitleSlug: a.TitleSlug, ArcID: &a.ID,
	})
	if err != nil {
		slog.WarnContext(ctx, "prestige: enrich arc reward failed", "arc_id", a.ID, "err", err)
		return a
	}
	objectivesPP := 0
	for _, o := range objectives {
		objectivesPP += PPForCompletion(s.deps.Tuning, o.Tier, false, o.DataTier)
	}
	a.ObjectivesPP = objectivesPP
	a.CompletionBonusPP = PPForArcCompletion(s.deps.Tuning, objectivesPP)
	return a
}

// ---------- Squad challenges ----------

// CreateSquadChallenge crée un défi d'escouade (collectif ou compétitif).
//
// L'invariant target_per_member est conservé : la cible affichée au total
// est calculée à la volée selon le nombre de participants actifs.
func (s *service) CreateSquadChallenge(ctx context.Context, req CreateSquadChallengeRequest) (SquadChallenge, error) {
	if req.SquadID == "" || req.TitleSlug == "" || req.CreatedBy == "" {
		return SquadChallenge{}, fmt.Errorf("%w: squad_id/title_slug/created_by requis", ErrInvalidInput)
	}
	if !req.Mode.Valid() || !req.EvalType.Valid() || !req.WindowType.Valid() {
		return SquadChallenge{}, fmt.Errorf("%w: enum invalide", ErrInvalidInput)
	}
	if req.TargetPerMember <= 0 && req.Mode == SquadCollective {
		return SquadChallenge{}, fmt.Errorf("%w: target_per_member requis en mode collectif", ErrInvalidInput)
	}

	now := s.deps.Now()
	sc := SquadChallenge{
		ID:              newID("sc"),
		SquadID:         req.SquadID,
		TemplateID:      req.TemplateID,
		TitleSlug:       req.TitleSlug,
		Mode:            req.Mode,
		EvalType:        req.EvalType,
		WindowType:      req.WindowType,
		WindowValue:     req.WindowValue,
		TargetPerMember: req.TargetPerMember,
		ExpiresAt:       req.ExpiresAt,
		CreatedBy:       req.CreatedBy,
		CreatedAt:       now,
	}
	if err := s.deps.SquadChallenges.Create(ctx, sc); err != nil {
		return SquadChallenge{}, fmt.Errorf("create squad challenge: %w", err)
	}

	// Le créateur rejoint automatiquement avec un palier par défaut Heroic.
	// C'est cohérent avec l'intuition "celui qui propose s'engage".
	creatorParticipant := SquadChallengeParticipant{
		SquadChallengeID: sc.ID,
		UserID:           req.CreatedBy,
		ChosenTier:       TierHeroic,
		DataTier:         DataFull, // Phase 4 affinera selon la baseline réelle
		JoinedAt:         now,
	}
	if err := s.deps.SquadChallenges.AddParticipant(ctx, creatorParticipant); err != nil {
		slog.WarnContext(ctx, "prestige: creator auto-join failed", "err", err)
	}

	slog.InfoContext(ctx, "prestige: squad challenge created",
		"squad_challenge_id", sc.ID, "squad_id", sc.SquadID, "mode", sc.Mode)
	return sc, nil
}

// JoinSquadChallenge fait rejoindre un membre à un défi d'escouade existant.
//
// Le palier choisi (chosenTier) est l'engagement personnel du membre — il
// n'est pas recalculé via CalculatePalier, c'est le joueur qui s'auto-évalue
// dans le pool collectif.
func (s *service) JoinSquadChallenge(ctx context.Context, challengeID, userID string, chosenTier Tier, isPrivate bool) error {
	if challengeID == "" || userID == "" {
		return fmt.Errorf("%w: challenge_id/user_id requis", ErrInvalidInput)
	}
	if chosenTier != "" && !chosenTier.Valid() {
		return fmt.Errorf("%w: chosen_tier invalide", ErrInvalidInput)
	}

	now := s.deps.Now()
	p := SquadChallengeParticipant{
		SquadChallengeID: challengeID,
		UserID:           userID,
		ChosenTier:       chosenTier,
		DataTier:         DataFull,
		IsPrivate:        isPrivate,
		JoinedAt:         now,
	}
	if err := s.deps.SquadChallenges.AddParticipant(ctx, p); err != nil {
		return fmt.Errorf("add participant: %w", err)
	}
	slog.InfoContext(ctx, "prestige: squad participant joined",
		"squad_challenge_id", challengeID, "user_id", userID, "tier", chosenTier)
	return nil
}

// GetSquadChallenge retourne un défi d'escouade par ID.
func (s *service) GetSquadChallenge(ctx context.Context, id string) (SquadChallenge, error) {
	sc, err := s.deps.SquadChallenges.Get(ctx, id)
	if err != nil {
		return SquadChallenge{}, fmt.Errorf("get squad challenge: %w", err)
	}
	return sc, nil
}

// ListSquadChallenges retourne tous les défis d'une escouade.
func (s *service) ListSquadChallenges(ctx context.Context, squadID string) ([]SquadChallenge, error) {
	if squadID == "" {
		return nil, fmt.Errorf("%w: squad_id requis", ErrInvalidInput)
	}
	return s.deps.SquadChallenges.ListBySquad(ctx, squadID)
}

// Ensure time is imported (used pour expires_at).
var _ = time.Time{}
