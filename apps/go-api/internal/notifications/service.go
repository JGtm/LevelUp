package notifications

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// DefaultListLimit est la pagination par défaut de Service.List.
const DefaultListLimit = 50

// MaxListLimit plafonne le param `limit` côté handler (défense contre dump).
const MaxListLimit = 100

// DefaultRetentionCap est le nombre max de notifs conservées par joueur.
const DefaultRetentionCap = 500

// ErrCategoryDisabled est retourné par Repository.Insert quand la pref est OFF.
// Service le traite comme un Emit silencieux (retourne nil au caller).
var ErrCategoryDisabled = errors.New("notifications: category disabled")

// ErrNotFound est retourné par les opérations sur ID inexistant
// (MarkUnread, Delete) et propagé tel quel par le handler en 404.
var ErrNotFound = errors.New("notifications: not found")

// Service implémente la logique métier de gestion des notifications.
// Stateless (sauf l'IDGenerator) — peut être instancié à la demande
// ou cacheé par xuid via la ServiceRegistry.
type Service struct {
	repo Repository
	idgen *IDGenerator
}

// NewService crée un Service à partir d'un Repository.
func NewService(repo Repository) *Service {
	return &Service{
		repo:  repo,
		idgen: NewIDGenerator(),
	}
}

// Emit construit une Notification depuis un EmitInput, applique les défauts
// (ID, severity, created_at), vérifie la pref puis insère.
//
// Retourne nil silencieusement si la catégorie est désactivée — c'est le
// comportement attendu (cf. plan §1.4 : drop opportuniste). Toute autre
// erreur (validation, encode, DB) est propagée pour que les hooks puissent logger.
func (s *Service) Emit(ctx context.Context, in EmitInput) error {
	if err := in.Validate(); err != nil {
		return err
	}
	enabled, err := s.repo.IsCategoryEnabled(ctx, in.Category)
	if err != nil {
		return fmt.Errorf("notifications: check pref: %w", err)
	}
	if !enabled {
		return nil // drop silencieux
	}

	params, err := in.EncodedParams()
	if err != nil {
		return err
	}
	target, err := in.EncodedTargetSearch()
	if err != nil {
		return err
	}

	severity := in.Severity
	if severity == "" {
		severity = SeverityInfo
	}

	n := &Notification{
		ID:           s.idgen.Next(),
		Category:     in.Category,
		Severity:     severity,
		TitleKey:     in.TitleKey,
		BodyKey:      in.BodyKey,
		Params:       params,
		TargetRoute:  in.TargetRoute,
		TargetSearch: target,
		Actor:        in.Actor,
		Source:       in.Source,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.repo.Insert(ctx, n); err != nil {
		if errors.Is(err, ErrCategoryDisabled) {
			return nil
		}
		return fmt.Errorf("notifications: insert: %w", err)
	}
	// Cap rétention best-effort : erreur loguée par le repo, jamais propagée.
	_ = s.repo.CapAndSweep(ctx, DefaultRetentionCap)
	return nil
}

// List retourne une page de notifications selon le filtre fourni.
func (s *Service) List(ctx context.Context, f ListFilter) (ListResult, error) {
	if f.Limit <= 0 || f.Limit > MaxListLimit {
		f.Limit = DefaultListLimit
	}
	return s.repo.List(ctx, f)
}

// UnreadCount renvoie le total et la répartition par catégorie.
func (s *Service) UnreadCount(ctx context.Context) (UnreadCount, error) {
	return s.repo.UnreadCount(ctx)
}

// MarkRead applique read_at sur les IDs fournis (idempotent).
func (s *Service) MarkRead(ctx context.Context, ids []int64) (MarkResult, error) {
	if len(ids) == 0 {
		return MarkResult{Updated: 0}, nil
	}
	n, err := s.repo.MarkRead(ctx, ids)
	return MarkResult{Updated: n}, err
}

// MarkUnread remet une notif en non-lu.
func (s *Service) MarkUnread(ctx context.Context, id int64) error {
	return s.repo.MarkUnread(ctx, id)
}

// MarkAllRead applique read_at sur toutes les non-lues (filtré par catégorie si non vide).
func (s *Service) MarkAllRead(ctx context.Context, category Category) (MarkResult, error) {
	n, err := s.repo.MarkAllRead(ctx, category)
	return MarkResult{Updated: n}, err
}

// Delete supprime une notification (action "ignorer").
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// GetPreferences charge l'état des préférences pour le joueur courant.
func (s *Service) GetPreferences(ctx context.Context) ([]Preference, error) {
	return s.repo.GetPreferences(ctx)
}

// UpdatePreferences upsert les préférences fournies. Les autres ne sont pas touchées.
func (s *Service) UpdatePreferences(ctx context.Context, prefs []Preference) ([]Preference, error) {
	if err := s.repo.UpsertPreferences(ctx, prefs); err != nil {
		return nil, err
	}
	return s.repo.GetPreferences(ctx)
}

// Compile-time check : Service satisfait Emitter.
var _ Emitter = (*Service)(nil)
