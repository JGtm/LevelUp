package notifications

import (
	"context"
	"errors"
	"fmt"
	"levelup/go-api/internal/observability"
	"log/slog"
	"time"

	"levelup/go-api/internal/platform/dblease"
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
	repo          Repository
	idgen         *IDGenerator
	acquireWriter func() (*dblease.LeasedWriter, error) // optionnel, cf. WithWriterAcquirer
}

// Option configure un Service à la construction.
type Option func(*Service)

// WithWriterAcquirer configure l'acquisition d'un *LeasedWriter avant chaque
// méthode write. Utilisé par la registry HTTP pour sérialiser les écritures
// shared_social.duckdb avec le sync engine et les autres handlers.
//
// Si nil ou non fourni, le Service écrit directement sans acquisition — pour
// les tests ou les contextes (sync hook futur) où le caller tient déjà le
// lease.
//
// Comportement par méthode quand le lease est bloqué :
//   - Emit, CapAndSweep : best-effort — log warn et continue (return nil),
//     ne casse pas le pipeline d'émission.
//   - MarkRead, MarkUnread, MarkAllRead, Delete, UpdatePreferences :
//     propage ErrDBLocked au caller (handler HTTP mappe en 503).
func WithWriterAcquirer(f func() (*dblease.LeasedWriter, error)) Option {
	return func(s *Service) { s.acquireWriter = f }
}

// NewService crée un Service à partir d'un Repository et d'options optionnelles.
func NewService(repo Repository, opts ...Option) *Service {
	s := &Service{
		repo:  repo,
		idgen: NewIDGenerator(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// withWriter exécute fn sous un *LeasedWriter acquis (si configuré).
// Retourne l'erreur du fn ou ErrDBLocked si le lease n'a pas pu être acquis.
//
// Si acquireWriter est nil, exécute directement fn — comportement legacy.
func (s *Service) withWriter(fn func() error) error {
	if s.acquireWriter == nil {
		return fn()
	}
	w, err := s.acquireWriter()
	if err != nil {
		return err
	}
	defer w.Release()
	return fn()
}

// withWriterBestEffort est une variante "best-effort" de withWriter : si le
// lease est bloqué (ErrDBLocked), log un warn et retourne nil au lieu de
// propager. Pour Emit / CapAndSweep où le caller (sync hook, boot) ne doit
// pas voir son pipeline cassé par une saturation temporaire de la DB.
func (s *Service) withWriterBestEffort(ctx context.Context, op string, fn func() error) error {
	if s.acquireWriter == nil {
		return fn()
	}
	w, err := s.acquireWriter()
	if err != nil {
		if errors.Is(err, dblease.ErrDBLocked) {
			slog.WarnContext(ctx, "notifications: write dropped (lease busy)",
				"op", op, "err", err)
			return nil
		}
		return err
	}
	defer w.Release()
	return fn()
}

// Emit construit une Notification depuis un EmitInput, applique les défauts
// (ID, severity, created_at), vérifie la pref puis insère.
//
// Retourne nil silencieusement si la catégorie est désactivée — c'est le
// comportement attendu (cf. plan §1.4 : drop opportuniste). Toute autre
// erreur (validation, encode, DB) est propagée pour que les hooks puissent logger.
//
// Si un WriterAcquirer est configuré et que le lease est saturé, l'émission
// est silencieusement droppée (best-effort) — le sync engine et le boot ne
// doivent jamais échouer à cause d'une notif qu'on n'a pas pu écrire.
func (s *Service) Emit(ctx context.Context, in EmitInput) error {
	// Heartbeat feature (A6/DC-5) : emission de notification vue vivante.
	observability.Heartbeat("notifications_push")
	if err := in.Validate(); err != nil {
		return err
	}
	return s.withWriterBestEffort(ctx, "Emit", func() error {
		return s.emitInner(ctx, in)
	})
}

// emitInner contient la logique d'Emit, exécutée sous le writer (si acquis).
// Extraite pour rester sous la barre 80 L (CLAUDE.md règle 13).
func (s *Service) emitInner(ctx context.Context, in EmitInput) error {
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
		slog.WarnContext(ctx, "notifications: insert échoué",
			"category", in.Category, "title_key", in.TitleKey, "err", err)
		return fmt.Errorf("notifications: insert: %w", err)
	}
	// Sprint B1 commit 18 : log InfoContext sur émission réussie pour tracer
	// les notifs post-sync (match_synced, sync_error) cross-module via event_id.
	slog.InfoContext(ctx, "notifications: émise",
		"category", in.Category, "title_key", in.TitleKey, "severity", severity)
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
//
// Si un WriterAcquirer est configuré, retourne ErrDBLocked au caller en cas
// de saturation du lease — le handler HTTP mappe en 503.
func (s *Service) MarkRead(ctx context.Context, ids []int64) (MarkResult, error) {
	if len(ids) == 0 {
		return MarkResult{Updated: 0}, nil
	}
	var n int
	err := s.withWriter(func() error {
		var inner error
		n, inner = s.repo.MarkRead(ctx, ids)
		return inner
	})
	return MarkResult{Updated: n}, err
}

// MarkUnread remet une notif en non-lu.
func (s *Service) MarkUnread(ctx context.Context, id int64) error {
	return s.withWriter(func() error { return s.repo.MarkUnread(ctx, id) })
}

// MarkAllRead applique read_at sur toutes les non-lues (filtré par catégorie si non vide).
func (s *Service) MarkAllRead(ctx context.Context, category Category) (MarkResult, error) {
	var n int
	err := s.withWriter(func() error {
		var inner error
		n, inner = s.repo.MarkAllRead(ctx, category)
		return inner
	})
	return MarkResult{Updated: n}, err
}

// Delete supprime une notification (action "ignorer").
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.withWriter(func() error { return s.repo.Delete(ctx, id) })
}

// GetPreferences charge l'état des préférences pour le joueur courant.
func (s *Service) GetPreferences(ctx context.Context) ([]Preference, error) {
	return s.repo.GetPreferences(ctx)
}

// UpdatePreferences upsert les préférences fournies. Les autres ne sont pas touchées.
func (s *Service) UpdatePreferences(ctx context.Context, prefs []Preference) ([]Preference, error) {
	if err := s.withWriter(func() error { return s.repo.UpsertPreferences(ctx, prefs) }); err != nil {
		return nil, err
	}
	return s.repo.GetPreferences(ctx)
}

// Compile-time check : Service satisfait Emitter.
var _ Emitter = (*Service)(nil)
