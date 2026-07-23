package prestige

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// service_squad_challenges.go — méthodes Service pour les défis d'escouade
// (création, participation, abandon, liste enrichie, expiration). Extrait de
// service_arcs_squads.go pour respecter le seuil 500 L (CLAUDE.md règle 5).

// ---------- Squad challenges ----------

// Durées d'expiration d'un défi d'escouade par cadence de template (Lot 3).
// Bornes nommées (pas de magic number) ; le mois est approximé à 30 jours.
const (
	squadExpiryDaily   = 24 * time.Hour
	squadExpiryWeekly  = 7 * 24 * time.Hour
	squadExpiryMonthly = 30 * 24 * time.Hour
)

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
	// expires_at : explicite (req) sinon dérivé de la cadence du template
	// (Lot 3). Lookup best-effort — un template retiré/illisible → pas
	// d'expiration (nil), jamais d'échec de création. Loggé (règle 3).
	expiresAt := req.ExpiresAt
	if expiresAt == nil && req.TemplateID != "" {
		if tpl, err := s.deps.Templates.GetByID(ctx, req.TemplateID); err != nil {
			slog.DebugContext(ctx, "prestige: squad challenge expiry — template lookup failed (pas d'expiration)",
				"err", err, "template_id", req.TemplateID)
		} else {
			expiresAt = squadChallengeExpiry(now, tpl.Cadence)
		}
	}
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
		ExpiresAt:       expiresAt,
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

	// Garde d'appartenance objet-level (BOLA) : on ne rejoint QUE le défi d'une
	// escouade dont userID est membre-user. Les défis d'escouade vivent dans une DB
	// partagée (non isolés par player DB) → sans cette garde, un utilisateur
	// (actor-gardé sur son propre user_id via le handler) pourrait rejoindre le défi
	// de N'IMPORTE quelle escouade via un challenge_id arbitraire. Même contrôle que
	// les mutations squad (assertMemberUser).
	sc, err := s.deps.SquadChallenges.Get(ctx, challengeID)
	if err != nil {
		// Sans le squad_id du défi, la vérification d'appartenance est impossible.
		// On LOGGE la cause (règle 10 : jamais d'erreur avalée) puis on refuse —
		// challenge_id inexistant OU lecture KO. Le module `prestige` est auto-détecté
		// (logs/prestige.log).
		slog.WarnContext(ctx, "prestige: squad challenge lookup failed on join",
			"err", err, "squad_challenge_id", challengeID, "user_id", userID)
		return fmt.Errorf("%w: défi d'escouade introuvable", ErrInvalidInput)
	}
	if err := s.assertMemberUser(ctx, sc.SquadID, userID); err != nil {
		return err
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

// AbandonSquadChallenge archive un défi d'escouade (abandon volontaire) : il
// sort de la liste active (ListSquadChallenges filtre archived_at IS NULL).
//
// Garde d'appartenance objet-level (BOLA) identique à Join/Evaluate : les défis
// d'escouade vivent dans une DB partagée, donc requestedBy DOIT être membre-user
// de l'escouade du défi. Idempotent (ré-archiver = UPDATE sans effet).
func (s *service) AbandonSquadChallenge(ctx context.Context, challengeID, requestedBy string) error {
	if challengeID == "" {
		return fmt.Errorf("%w: challenge_id requis", ErrInvalidInput)
	}
	sc, err := s.deps.SquadChallenges.Get(ctx, challengeID)
	if err != nil {
		// Sans le squad_id du défi, la vérification d'appartenance est impossible.
		// On LOGGE (règle 10) puis on refuse — challenge_id inexistant OU lecture KO.
		slog.WarnContext(ctx, "prestige: squad challenge lookup failed on abandon",
			"err", err, "squad_challenge_id", challengeID, "requested_by", requestedBy)
		return fmt.Errorf("%w: défi d'escouade introuvable", ErrInvalidInput)
	}
	if err := s.assertMemberUser(ctx, sc.SquadID, requestedBy); err != nil {
		return err
	}
	if err := s.deps.SquadChallenges.Archive(ctx, challengeID); err != nil {
		return fmt.Errorf("archive squad challenge: %w", err)
	}
	slog.InfoContext(ctx, "prestige: squad challenge abandoned",
		"squad_challenge_id", challengeID, "squad_id", sc.SquadID, "by", requestedBy)
	return nil
}

// squadChallengeExpiry calcule la date d'expiration d'un défi d'escouade depuis
// la cadence de son template (daily → +1 j, weekly → +7 j, monthly → +30 j).
// Cadence libre/inconnue → nil (pas d'expiration). Bornes nommées, pas de magie.
func squadChallengeExpiry(now time.Time, cadence Cadence) *time.Time {
	var d time.Duration
	switch cadence {
	case CadenceDaily:
		d = squadExpiryDaily
	case CadenceWeekly:
		d = squadExpiryWeekly
	case CadenceMonthly:
		d = squadExpiryMonthly
	default:
		return nil
	}
	exp := now.Add(d)
	return &exp
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
func (s *service) ListSquadChallenges(ctx context.Context, squadID, requestedBy string) ([]SquadChallengeView, error) {
	if squadID == "" {
		return nil, fmt.Errorf("%w: squad_id requis", ErrInvalidInput)
	}
	if err := s.assertMemberUser(ctx, squadID, requestedBy); err != nil {
		return nil, err
	}
	challenges, err := s.deps.SquadChallenges.ListBySquad(ctx, squadID)
	if err != nil {
		return nil, err
	}
	views := make([]SquadChallengeView, 0, len(challenges))
	for _, c := range challenges {
		views = append(views, s.enrichSquadChallenge(ctx, c))
	}
	return views, nil
}

// enrichSquadChallenge hydrate un défi d'escouade avec ses libellés localisés
// (résolus depuis le template) et ses participants courants. Best-effort : une
// erreur de lecture (template retiré du catalogue, participants indisponibles)
// dégrade le champ concerné sans faire échouer la liste, et est LOGGÉE — jamais
// avalée en silence (CLAUDE.md règle 3). Le nombre de défis par escouade étant
// borné (poignée), le coût N+1 des lectures par défi est acceptable.
func (s *service) enrichSquadChallenge(ctx context.Context, c SquadChallenge) SquadChallengeView {
	view := SquadChallengeView{SquadChallenge: c, Participants: []SquadChallengeParticipant{}}
	// Expiré : comparé à l'horloge canonique UTC du service (cohérent avec le
	// calcul d'expires_at à la création) — jamais CURRENT_TIMESTAMP SQL (fuseau).
	view.Expired = c.ExpiresAt != nil && c.ExpiresAt.Before(s.deps.Now())
	if c.TemplateID != "" {
		if tpl, err := s.deps.Templates.GetByID(ctx, c.TemplateID); err != nil {
			slog.DebugContext(ctx, "prestige: squad challenge label lookup failed (libellés omis)",
				"err", err, "squad_challenge_id", c.ID, "template_id", c.TemplateID)
		} else {
			view.LabelFR, view.LabelEN = tpl.LabelFR, tpl.LabelEN
		}
	}
	if parts, err := s.deps.SquadChallenges.ListParticipants(ctx, c.ID); err != nil {
		slog.WarnContext(ctx, "prestige: squad challenge participants unavailable (liste partielle)",
			"err", err, "squad_challenge_id", c.ID)
	} else if parts != nil {
		view.Participants = parts
	}
	return view
}
