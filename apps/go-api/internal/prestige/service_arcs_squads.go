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

// arcCooldownExemptionWindow : un arc supprimé moins d'une heure après sa
// création est considéré « à peine entamé » → ses objectifs sont supprimés
// physiquement (zéro cooldown). Au-delà, ils sont abandonnés (cooldown appliqué).
const arcCooldownExemptionWindow = time.Hour

// DeleteArcOptions paramètre la suppression d'un arc.
type DeleteArcOptions struct {
	// CascadeObjectives : true = supprime aussi les objectifs (abandon, ou
	// hard delete si l'arc est dans la fenêtre d'exemption) ; false = détache
	// les objectifs (arc_id = NULL), ils redeviennent libres.
	CascadeObjectives bool
}

// DeleteArc supprime un arc appartenant à userID.
//
//   - CascadeObjectives=false : les objectifs sont détachés (gardés, libres).
//   - CascadeObjectives=true & arc créé < 1h : objectifs supprimés (zéro cooldown).
//   - CascadeObjectives=true & arc créé ≥ 1h : objectifs actifs abandonnés
//     (cooldown 24h appliqué par métrique), puis l'arc est supprimé.
func (s *service) DeleteArc(ctx context.Context, userID, id string, opts DeleteArcOptions) error {
	arc, err := s.deps.Arcs.Get(ctx, id)
	if err != nil {
		if errors.Is(err, ErrArcNotFound) {
			return ErrArcNotFound
		}
		return fmt.Errorf("get arc: %w", err)
	}
	if arc.UserID != userID {
		return ErrForbidden
	}

	if opts.CascadeObjectives {
		if err := s.cascadeDeleteObjectives(ctx, arc); err != nil {
			return err
		}
	} else if err := s.deps.Challenges.DetachFromArc(ctx, id); err != nil {
		return fmt.Errorf("detach objectives: %w", err)
	}

	if err := s.deps.Arcs.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete arc: %w", err)
	}
	slog.InfoContext(ctx, "prestige: arc deleted",
		"arc_id", id, "user_id", userID, "cascade", opts.CascadeObjectives)
	return nil
}

// cascadeDeleteObjectives traite les objectifs d'un arc supprimé en cascade :
// hard delete si l'arc est dans la fenêtre d'exemption (zéro cooldown), sinon
// abandon des objectifs encore actifs (cooldown appliqué, télémétrie conservée).
func (s *service) cascadeDeleteObjectives(ctx context.Context, arc Arc) error {
	if s.deps.Now().Sub(arc.CreatedAt) < arcCooldownExemptionWindow {
		if err := s.deps.Challenges.DeleteByArc(ctx, arc.ID); err != nil {
			return fmt.Errorf("delete objectives: %w", err)
		}
		return nil
	}
	objs, err := s.deps.Challenges.List(ctx, ChallengeFilter{ArcID: &arc.ID})
	if err != nil {
		return fmt.Errorf("list objectives: %w", err)
	}
	for _, c := range objs {
		if !CanAbandon(c) {
			continue
		}
		if err := s.AbandonChallenge(ctx, c.ID); err != nil {
			return fmt.Errorf("abandon objective %s: %w", c.ID, err)
		}
	}
	return nil
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

// ListArcPresets retourne le catalogue d'arcs preset du titre, chaque preset
// hydraté avec ses étapes. Best-effort sur l'hydratation des étapes (un preset
// dont les étapes échouent est tout de même retourné, sans steps).
func (s *service) ListArcPresets(ctx context.Context, _ /*userID*/, titleSlug string) ([]PresetArc, error) {
	if titleSlug == "" {
		return nil, fmt.Errorf("%w: title_slug requis", ErrInvalidInput)
	}
	presets, err := s.deps.PresetArcs.ListByTitle(ctx, titleSlug)
	if err != nil {
		return nil, fmt.Errorf("list preset arcs: %w", err)
	}
	for i := range presets {
		steps, err := s.deps.PresetArcs.GetSteps(ctx, presets[i].ID)
		if err != nil {
			slog.WarnContext(ctx, "prestige: preset steps load failed",
				"preset_id", presets[i].ID, "err", err)
			continue
		}
		presets[i].Steps = steps
	}
	return presets, nil
}

// AdoptPresetArc matérialise un arc preset pour le joueur : crée l'arc
// (IsPreset=true, PresetID) puis un objectif libre par étape (template + palier
// cible). Best-effort par étape : un objectif refusé (cible trop facile, etc.)
// n'annule pas l'adoption — on log et on poursuit.
func (s *service) AdoptPresetArc(ctx context.Context, userID, titleSlug, presetID string) (Arc, error) {
	if userID == "" || titleSlug == "" || presetID == "" {
		return Arc{}, fmt.Errorf("%w: user_id/title_slug/preset_id requis", ErrInvalidInput)
	}
	preset, err := s.deps.PresetArcs.GetByID(ctx, presetID)
	if err != nil {
		return Arc{}, ErrArcNotFound
	}
	if preset.TitleSlug != titleSlug {
		return Arc{}, fmt.Errorf("%w: preset hors du titre demandé", ErrInvalidInput)
	}
	steps, err := s.deps.PresetArcs.GetSteps(ctx, presetID)
	if err != nil {
		return Arc{}, fmt.Errorf("get preset steps: %w", err)
	}

	arc := Arc{
		ID:          newID("arc"),
		UserID:      userID,
		TitleSlug:   titleSlug,
		Title:       presetTitle(preset),
		Description: presetDescription(preset),
		IsPreset:    true,
		PresetID:    presetID,
		CreatedAt:   s.deps.Now(),
	}
	if err := s.deps.Arcs.Create(ctx, arc); err != nil {
		return Arc{}, fmt.Errorf("create preset arc: %w", err)
	}

	templates, err := s.deps.Templates.ListByTitle(ctx, titleSlug)
	if err != nil {
		return Arc{}, fmt.Errorf("list templates: %w", err)
	}
	byID := make(map[string]Template, len(templates))
	for _, t := range templates {
		byID[t.ID] = t
	}

	created := 0
	for _, st := range steps {
		tmpl, ok := byID[st.TemplateID]
		if !ok {
			slog.WarnContext(ctx, "prestige: preset step template missing",
				"template_id", st.TemplateID, "preset_id", presetID)
			continue
		}
		if _, err := s.CreateChallenge(ctx, CreateChallengeRequest{
			UserID:      userID,
			TitleSlug:   titleSlug,
			ArcID:       arc.ID,
			Position:    st.Position,
			TemplateID:  tmpl.ID,
			Metric:      tmpl.Metric,
			Target:      targetForTier(tmpl, st.TargetTier),
			WindowType:  tmpl.WindowType,
			WindowValue: tmpl.WindowValue,
			Cadence:     tmpl.Cadence,
			EvalType:    tmpl.EvalType,
			Mode:        ModeLibre,
			Label:       tmpl.LabelFR,
			// Adoption d'un preset arc = action initiée par le joueur → origine user
			// (le coach a sa propre voie via coach_advisor). Calage coach, ADR 0020.
			Source: ChallengeSourceUser,
		}); err != nil {
			slog.WarnContext(ctx, "prestige: preset step skipped",
				"template_id", tmpl.ID, "preset_id", presetID, "err", err)
			continue
		}
		created++
	}
	if created == 0 && len(steps) > 0 {
		slog.WarnContext(ctx, "prestige: preset arc adopted but all steps were skipped",
			"arc_id", arc.ID, "preset_id", presetID, "user_id", userID, "steps_total", len(steps))
	}
	slog.InfoContext(ctx, "prestige: preset arc adopted",
		"arc_id", arc.ID, "preset_id", presetID, "user_id", userID,
		"steps_created", created, "steps_total", len(steps))
	return arc, nil
}

// targetForTier sélectionne la cible du template correspondant au palier.
func targetForTier(t Template, tier Tier) float64 {
	switch tier {
	case TierHeroic:
		return t.HeroicTarget
	case TierLegendary:
		return t.LegendaryTarget
	case TierMythic:
		return t.MythicTarget
	default:
		return t.NormalTarget
	}
}

// presetTitle / presetDescription : libellé FR prioritaire, repli EN.
func presetTitle(p PresetArc) string {
	if p.TitleFR != "" {
		return p.TitleFR
	}
	return p.TitleEN
}

func presetDescription(p PresetArc) string {
	if p.DescriptionFR != "" {
		return p.DescriptionFR
	}
	return p.DescriptionEN
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

// ListSquadChallenges retourne tous les défis d'une escouade. requestedBy
// (player_slug) doit être membre-user de l'escouade.
//
// Garde d'appartenance objet-level (BOLA) : contrairement aux défis/arcs perso
// (isolés par player DB), les défis d'escouade vivent dans une DB sociale
// partagée (tous joueurs). Sans cette garde, un utilisateur possédant son propre
// slug (ownershipMW OK) pourrait lister les défis de N'IMPORTE quelle escouade
// via un squad_id arbitraire. Même contrôle que les mutations squad
// (assertMemberUser), appliqué en lecture.
func (s *service) ListSquadChallenges(ctx context.Context, squadID, requestedBy string) ([]SquadChallenge, error) {
	if squadID == "" {
		return nil, fmt.Errorf("%w: squad_id requis", ErrInvalidInput)
	}
	if err := s.assertMemberUser(ctx, squadID, requestedBy); err != nil {
		return nil, err
	}
	return s.deps.SquadChallenges.ListBySquad(ctx, squadID)
}

// Ensure time is imported (used pour expires_at).
var _ = time.Time{}
