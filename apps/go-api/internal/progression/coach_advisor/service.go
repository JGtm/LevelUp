package coach_advisor

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/prestige"
)

// Service est la façade applicative du coach_advisor.
//
// Cinq familles d'opérations :
//
//  1. Génération : GenerateProposals — pipeline matcher → synthesizer → composer
//     → supersession → persistence. Cf. ADR 0020.
//  2. Action joueur : ListProposals, AcceptProposal, DismissProposal.
//  3. Maintenance : ObsoletePendingForAxis — appelé quand un challenge accepté
//     coach se complète (cf. ADR 0020 §3 obsolescence).
//
// Toutes les méthodes sont scopées par (userID, titleSlug) explicites — le
// Service n'a pas d'état lié à un joueur (stateless).
type Service interface {
	// GenerateProposals exécute le pipeline complet : catalogue matching →
	// template synthesis → arc composition → supersession → persistence.
	// Retourne uniquement les proposals nouvellement créées (les superseded
	// sont mises à jour en DB mais pas retournées).
	//
	// Si input.ProactiveEnabled == false → short-circuit, retourne (nil, nil).
	GenerateProposals(ctx context.Context, input GenerateInput) ([]Proposal, error)

	// ListProposals retourne les proposals d'un joueur, optionnellement filtrées
	// par status (vide = toutes). Tri par created_at DESC.
	ListProposals(ctx context.Context, userID, titleSlug string, status ProposalStatus) ([]Proposal, error)

	// AcceptProposal matérialise la proposal via prestige.CreateChallenge ou
	// CreateArc + N CreateChallenge selon Kind. Marque la proposal accepted
	// avec resolved_ref = challenge_id ou arc_id.
	//
	// Erreurs : ErrProposalNotFound si id inconnu ; ErrProposalNotAcceptable
	// si la proposal n'est plus pending.
	AcceptProposal(ctx context.Context, id string) (AcceptResult, error)

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

// ErrProposalNotAcceptable est retournée par AcceptProposal si la proposal
// n'est plus dans l'état pending (déjà accepted, dismissed, etc.).
var ErrProposalNotAcceptable = errors.New("coach_advisor: proposal is not pending")

// GenerateInput regroupe les paramètres d'invocation de GenerateProposals.
type GenerateInput struct {
	UserID    string
	TitleSlug string
	Now       time.Time

	// ProactiveEnabled est lu depuis settings.AppSettings.CoachProactiveMode.
	// Si false, GenerateProposals retourne immédiatement (nil, nil).
	ProactiveEnabled bool

	// Signals construits par SignalsFromAlerts() en amont (Phase 4).
	Signals []Signal
}

// AcceptResult est retourné par AcceptProposal.
type AcceptResult struct {
	// ChallengeID est positionné si Kind=challenge.
	ChallengeID string
	// ArcID + ChallengeIDs sont positionnés si Kind=arc (un arc + N challenges).
	ArcID        string
	ChallengeIDs []string
}

// Tuning regroupe les seuils numériques du Service (cf. ADR 0020).
type Tuning struct {
	// MinCatalogMatchScore : score minimum pour qu'un template catalogue
	// soit retenu (sinon fallback synthèse). Défaut 0.4.
	MinCatalogMatchScore float64
	// MaxProposalsPerSync : cap dur sur le nombre de proposals créées en
	// une seule invocation de GenerateProposals. Défaut 3.
	MaxProposalsPerSync int
	// SupersessionStrengthUplift : ratio minimum (P_new.strength /
	// P_old.strength) pour qu'une nouvelle proposal supersède une ancienne.
	// Défaut 1.10 (10 % plus fort).
	SupersessionStrengthUplift float64
}

// DefaultTuning retourne les seuils canoniques de l'ADR 0020.
func DefaultTuning() Tuning {
	return Tuning{
		MinCatalogMatchScore:       0.4,
		MaxProposalsPerSync:        3,
		SupersessionStrengthUplift: 1.10,
	}
}

// PrestigeWriter est le sous-ensemble minimal de prestige.Service utilisé
// par coach_advisor — limite la surface des mocks dans les tests.
type PrestigeWriter interface {
	CreateChallenge(ctx context.Context, req prestige.CreateChallengeRequest) (prestige.Challenge, error)
	CreateArc(ctx context.Context, req prestige.CreateArcRequest) (prestige.Arc, error)
}

// ServiceDeps regroupe les dépendances du Service. Les deps optionnelles
// peuvent être nil : les méthodes correspondantes retourneront une erreur
// claire si appelées dans cet état.
//
// Required pour List/Dismiss/Obsolete : Repo uniquement.
// Required pour GenerateProposals    : Repo, Templates, Synthesizer (au moins
//
//	un de Templates ou Synthesizer suffit
//	en pratique selon les signaux reçus).
//
// Required pour AcceptProposal       : Repo, Templates, Prestige.
type ServiceDeps struct {
	Repo           Repo
	Templates      prestige.TemplateRepo
	Synthesizer    *Synthesizer
	Prestige       PrestigeWriter
	ComposerConfig ArcComposerConfig
	MatcherWeights MatcherWeights
	Tuning         Tuning
	Now            func() time.Time // injectable pour tests
	IDGen          func() string    // injectable pour tests
}

// service est l'implémentation par défaut. Constructeur via NewService.
type service struct {
	deps ServiceDeps
}

// NewService construit un Service avec les dépendances fournies.
//
// Defaults appliqués si vides : ComposerConfig=DefaultArcComposerConfig,
// MatcherWeights=DefaultMatcherWeights, Tuning=DefaultTuning, Now=time.Now,
// IDGen=newProposalID.
func NewService(deps ServiceDeps) Service {
	if deps.Now == nil {
		deps.Now = func() time.Time { return time.Now().UTC() }
	}
	if deps.IDGen == nil {
		deps.IDGen = func() string { return newProposalID() }
	}
	if (deps.ComposerConfig == ArcComposerConfig{}) {
		deps.ComposerConfig = DefaultArcComposerConfig()
	}
	if (deps.MatcherWeights == MatcherWeights{}) {
		deps.MatcherWeights = DefaultMatcherWeights()
	}
	if (deps.Tuning == Tuning{}) {
		deps.Tuning = DefaultTuning()
	}
	return &service{deps: deps}
}

// newProposalID génère un id court préfixé "prop_" + 16 chars hex.
func newProposalID() string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	return "prop_" + hex.EncodeToString(buf[:])
}

// ListProposals délègue au Repo.
func (s *service) ListProposals(ctx context.Context, userID, titleSlug string, status ProposalStatus) ([]Proposal, error) {
	if userID == "" || titleSlug == "" {
		return nil, fmt.Errorf("coach_advisor.ListProposals: userID and titleSlug required")
	}
	props, err := s.deps.Repo.ListByUser(ctx, userID, titleSlug, status)
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
	if err := s.deps.Repo.MarkDismissed(ctx, id, s.deps.Now()); err != nil {
		return fmt.Errorf("coach_advisor.DismissProposal: %w", err)
	}
	return nil
}

// ObsoletePendingForAxis : best-effort batch obsolescence.
func (s *service) ObsoletePendingForAxis(ctx context.Context, userID, titleSlug, axis string) (int, error) {
	if userID == "" || titleSlug == "" || axis == "" {
		return 0, fmt.Errorf("coach_advisor.ObsoletePendingForAxis: userID, titleSlug, axis required")
	}
	pending, err := s.deps.Repo.ListPendingByAxis(ctx, userID, titleSlug, axis)
	if err != nil {
		return 0, fmt.Errorf("coach_advisor.ObsoletePendingForAxis: list: %w", err)
	}
	now := s.deps.Now()
	obsoleted := 0
	for _, p := range pending {
		if err := s.deps.Repo.MarkObsoleted(ctx, p.ID, now); err != nil {
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
