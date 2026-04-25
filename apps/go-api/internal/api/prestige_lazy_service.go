// Package api — adaptateur "lazy" qui résout un prestige.Service
// par requête en s'appuyant sur le PrestigeBundle.
//
// Ce wrapper permet de passer une instance unique au PrestigeHandler tout en
// résolvant le PlayerDB à la demande — chaque méthode reçoit le user_id en
// paramètre, qui sert de clé pour ouvrir/réutiliser la connexion player.

package api

import (
	"context"
	"errors"
	"net/http"

	"levelup/go-api/internal/prestige"
)

// playerSlugCtxKey extrait le player_slug d'un context HTTP.
//
// Phase 4 : on s'appuie sur le user_id passé en query/body. À terme,
// un middleware Auth pourra le mettre dans le context (clé identique à
// celle utilisée par TitleExtractor pour title_slug).
type playerSlugCtxKey struct{}

// WithPlayerSlug attache le player_slug au context (utilisable par middleware).
func WithPlayerSlug(ctx context.Context, slug string) context.Context {
	return context.WithValue(ctx, playerSlugCtxKey{}, slug)
}

// PlayerSlugFromContext extrait le player_slug, retourne "" si absent.
func PlayerSlugFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(playerSlugCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// LazyPrestigeService implémente prestige.Service en résolvant le Service
// à chaque appel via le bundle.
//
// Le player_slug est extrait depuis :
//  1. la fonction extractFn fournie (lit body/query/context)
//  2. en fallback, retourne ErrPlayerNotResolved
type LazyPrestigeService struct {
	bundle    *PrestigeBundle
	extractFn func(ctx context.Context) string
}

// NewLazyPrestigeService construit un wrapper.
//
// extractFn est typiquement une closure qui lit le request en cours via
// un middleware (chi context value). Si elle retourne "", les méthodes
// retournent ErrPlayerNotResolved.
func NewLazyPrestigeService(bundle *PrestigeBundle, extractFn func(ctx context.Context) string) *LazyPrestigeService {
	if extractFn == nil {
		extractFn = PlayerSlugFromContext
	}
	return &LazyPrestigeService{bundle: bundle, extractFn: extractFn}
}

// ErrPlayerNotResolved est retournée quand le player_slug ne peut pas être
// extrait du context — l'appelant doit le fournir via middleware ou body.
var ErrPlayerNotResolved = errors.New("prestige: player_slug missing from context")

// resolve charge le Service pour le player courant ; retourne err si introuvable.
func (l *LazyPrestigeService) resolve(ctx context.Context) (prestige.Service, error) {
	slug := l.extractFn(ctx)
	if slug == "" {
		return nil, ErrPlayerNotResolved
	}
	return l.bundle.ServiceForPlayer(ctx, slug)
}

// resolveByUserID est un fallback : si le contexte ne contient pas de slug
// mais qu'un userID est fourni explicitement (ex. depuis le body), on l'utilise.
func (l *LazyPrestigeService) resolveByUserID(ctx context.Context, userID string) (prestige.Service, error) {
	slug := l.extractFn(ctx)
	if slug == "" {
		slug = userID
	}
	if slug == "" {
		return nil, ErrPlayerNotResolved
	}
	return l.bundle.ServiceForPlayer(ctx, slug)
}

// ─── Compile-time assertion ───
var _ prestige.Service = (*LazyPrestigeService)(nil)

// ─── Méthodes Service (délégation) ───

func (l *LazyPrestigeService) CreateChallenge(ctx context.Context, req prestige.CreateChallengeRequest) (prestige.Challenge, error) {
	svc, err := l.resolveByUserID(ctx, req.UserID)
	if err != nil {
		return prestige.Challenge{}, err
	}
	return svc.CreateChallenge(ctx, req)
}

func (l *LazyPrestigeService) UpdateChallenge(ctx context.Context, id string, patch prestige.UpdateChallengePatch) (prestige.Challenge, error) {
	svc, err := l.resolve(ctx)
	if err != nil {
		return prestige.Challenge{}, err
	}
	return svc.UpdateChallenge(ctx, id, patch)
}

func (l *LazyPrestigeService) AbandonChallenge(ctx context.Context, id string) error {
	svc, err := l.resolve(ctx)
	if err != nil {
		return err
	}
	return svc.AbandonChallenge(ctx, id)
}

func (l *LazyPrestigeService) GetChallenge(ctx context.Context, id string) (prestige.Challenge, error) {
	svc, err := l.resolve(ctx)
	if err != nil {
		return prestige.Challenge{}, err
	}
	return svc.GetChallenge(ctx, id)
}

func (l *LazyPrestigeService) ListActiveChallenges(ctx context.Context, userID, titleSlug string) ([]prestige.Challenge, error) {
	svc, err := l.resolveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return svc.ListActiveChallenges(ctx, userID, titleSlug)
}

func (l *LazyPrestigeService) EvaluateForUser(ctx context.Context, userID, titleSlug string) ([]prestige.EvaluationOutcome, error) {
	svc, err := l.resolveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return svc.EvaluateForUser(ctx, userID, titleSlug)
}

func (l *LazyPrestigeService) GetUserPrestige(ctx context.Context, userID, titleSlug string) (prestige.UserPrestige, error) {
	svc, err := l.resolveByUserID(ctx, userID)
	if err != nil {
		return prestige.UserPrestige{}, err
	}
	return svc.GetUserPrestige(ctx, userID, titleSlug)
}

func (l *LazyPrestigeService) SuggestTemplates(ctx context.Context, userID, titleSlug string, count int) ([]prestige.Template, error) {
	svc, err := l.resolveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return svc.SuggestTemplates(ctx, userID, titleSlug, count)
}

func (l *LazyPrestigeService) SuggestNext(ctx context.Context, completedID string) ([]prestige.Template, error) {
	svc, err := l.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return svc.SuggestNext(ctx, completedID)
}

func (l *LazyPrestigeService) CreateArc(ctx context.Context, req prestige.CreateArcRequest) (prestige.Arc, error) {
	svc, err := l.resolveByUserID(ctx, req.UserID)
	if err != nil {
		return prestige.Arc{}, err
	}
	return svc.CreateArc(ctx, req)
}

func (l *LazyPrestigeService) ListArcs(ctx context.Context, userID, titleSlug string) ([]prestige.Arc, error) {
	svc, err := l.resolveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return svc.ListArcs(ctx, userID, titleSlug)
}

func (l *LazyPrestigeService) GetArc(ctx context.Context, id string) (prestige.Arc, error) {
	svc, err := l.resolve(ctx)
	if err != nil {
		return prestige.Arc{}, err
	}
	return svc.GetArc(ctx, id)
}

func (l *LazyPrestigeService) CreateSquadChallenge(ctx context.Context, req prestige.CreateSquadChallengeRequest) (prestige.SquadChallenge, error) {
	svc, err := l.resolveByUserID(ctx, req.CreatedBy)
	if err != nil {
		return prestige.SquadChallenge{}, err
	}
	return svc.CreateSquadChallenge(ctx, req)
}

func (l *LazyPrestigeService) JoinSquadChallenge(ctx context.Context, challengeID, userID string, chosenTier prestige.Tier, isPrivate bool) error {
	svc, err := l.resolveByUserID(ctx, userID)
	if err != nil {
		return err
	}
	return svc.JoinSquadChallenge(ctx, challengeID, userID, chosenTier, isPrivate)
}

func (l *LazyPrestigeService) GetSquadChallenge(ctx context.Context, id string) (prestige.SquadChallenge, error) {
	svc, err := l.resolve(ctx)
	if err != nil {
		return prestige.SquadChallenge{}, err
	}
	return svc.GetSquadChallenge(ctx, id)
}

func (l *LazyPrestigeService) ListSquadChallenges(ctx context.Context, squadID string) ([]prestige.SquadChallenge, error) {
	svc, err := l.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return svc.ListSquadChallenges(ctx, squadID)
}

// ─── Helper HTTP middleware ───

// PlayerSlugFromQueryOrBody extrait player_slug d'une request HTTP via :
//  1. query param "user_id"
//  2. en fallback, le slug "halo_infinite" du DefaultSlug si ?title_slug indique
//
// Pour le body, l'extraction se fait ailleurs (avant de wrapper le ctx).
// Phase 4 : implémentation pragmatique, à étendre avec un middleware Auth
// quand l'identification utilisateur sera robuste.
func PlayerSlugFromQueryOrBody(r *http.Request) string {
	if v := r.URL.Query().Get("user_id"); v != "" {
		return v
	}
	return ""
}
