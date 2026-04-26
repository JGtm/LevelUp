// Package duckdb — NotificationsRepo : persistance des notifications in-app.
//
// Implémente notifications.Repository (port défini dans internal/notifications/port.go).
// Tables : player_notifications, notification_preferences (stats.duckdb, per-player).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/notifications"
)

// NotificationsRepo persiste les notifs et préférences d'un joueur dans stats.duckdb.
type NotificationsRepo struct {
	pdb *PlayerDB
}

// NewNotificationsRepo construit un NotificationsRepo depuis un PlayerDB.
func NewNotificationsRepo(pdb *PlayerDB) *NotificationsRepo {
	return &NotificationsRepo{pdb: pdb}
}

const (
	notifWriteTimeout = 10 * time.Second
	notifReadTimeout  = 15 * time.Second
)

// Insert persiste une notification déjà préparée par le service (ID, severity,
// created_at, params encodés). Vérifie la pref de catégorie : si OFF, retourne
// notifications.ErrCategoryDisabled (le service traduit en no-op).
func (r *NotificationsRepo) Insert(ctx context.Context, n *notifications.Notification) error {
	ctx, cancel := context.WithTimeout(ctx, notifWriteTimeout)
	defer cancel()

	rwDB, err := OpenReadWrite(r.pdb.Player.Path())
	if err != nil {
		return fmt.Errorf("NotificationsRepo.Insert: open rw: %w", err)
	}
	defer rwDB.Close()

	enabled, err := isCategoryEnabledOn(ctx, rwDB, n.Category)
	if err != nil {
		return fmt.Errorf("NotificationsRepo.Insert: check pref: %w", err)
	}
	if !enabled {
		return notifications.ErrCategoryDisabled
	}

	var actorXUID, actorName any
	if n.Actor != nil {
		actorXUID = n.Actor.XUID
		actorName = n.Actor.Name
	}

	_, err = rwDB.Exec(ctx, `
		INSERT INTO player_notifications
			(id, category, severity, title_key, body_key, params,
			 target_route, target_search, actor_xuid, actor_name,
			 source, created_at, read_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)
	`,
		n.ID, string(n.Category), string(n.Severity), n.TitleKey,
		nullableString(n.BodyKey),
		nullableJSON(n.Params),
		nullableString(n.TargetRoute),
		nullableJSON(n.TargetSearch),
		actorXUID, actorName,
		n.Source, n.CreatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("NotificationsRepo.Insert: exec: %w", err)
	}
	return nil
}

// List paginé. Tri DESC par created_at, cursor sur id (plus stable car snowflake).
func (r *NotificationsRepo) List(ctx context.Context, f notifications.ListFilter) (notifications.ListResult, error) {
	ctx, cancel := context.WithTimeout(ctx, notifReadTimeout)
	defer cancel()

	q, args := buildListQuery(f)
	rows, err := r.pdb.ReadDB().Query(ctx, q, args...)
	if err != nil {
		return notifications.ListResult{}, fmt.Errorf("NotificationsRepo.List: query: %w", err)
	}
	defer rows.Close()

	items, err := scanNotifications(rows)
	if err != nil {
		return notifications.ListResult{}, err
	}
	res := notifications.ListResult{Items: items}
	if len(items) == f.Limit && f.Limit > 0 {
		last := items[len(items)-1].ID
		res.NextCursor = &last
	}
	return res, nil
}

// UnreadCount retourne le total non-lu et la répartition par catégorie.
func (r *NotificationsRepo) UnreadCount(ctx context.Context) (notifications.UnreadCount, error) {
	ctx, cancel := context.WithTimeout(ctx, notifReadTimeout)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx, `
		SELECT category, COUNT(*) AS n
		FROM player_notifications
		WHERE read_at IS NULL
		GROUP BY category
	`)
	if err != nil {
		return notifications.UnreadCount{}, fmt.Errorf("NotificationsRepo.UnreadCount: %w", err)
	}
	defer rows.Close()

	out := notifications.UnreadCount{ByCategory: map[string]int{}}
	for rows.Next() {
		var cat string
		var n int
		if err := rows.Scan(&cat, &n); err != nil {
			return notifications.UnreadCount{}, err
		}
		out.ByCategory[cat] = n
		out.Count += n
	}
	return out, rows.Err()
}

// MarkRead positionne read_at = NOW() pour tous les IDs fournis (idempotent).
func (r *NotificationsRepo) MarkRead(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(ctx, notifWriteTimeout)
	defer cancel()

	rwDB, err := OpenReadWrite(r.pdb.Player.Path())
	if err != nil {
		return 0, fmt.Errorf("NotificationsRepo.MarkRead: open rw: %w", err)
	}
	defer rwDB.Close()

	q, args := buildMarkReadQuery(ids)
	res, err := rwDB.Exec(ctx, q, args...)
	if err != nil {
		return 0, fmt.Errorf("NotificationsRepo.MarkRead: exec: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// MarkUnread remet read_at = NULL.
func (r *NotificationsRepo) MarkUnread(ctx context.Context, id int64) error {
	ctx, cancel := context.WithTimeout(ctx, notifWriteTimeout)
	defer cancel()

	rwDB, err := OpenReadWrite(r.pdb.Player.Path())
	if err != nil {
		return fmt.Errorf("NotificationsRepo.MarkUnread: open rw: %w", err)
	}
	defer rwDB.Close()

	res, err := rwDB.Exec(ctx, `UPDATE player_notifications SET read_at = NULL WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("NotificationsRepo.MarkUnread: exec: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return notifications.ErrNotFound
	}
	return nil
}

// MarkAllRead applique read_at sur toutes les non-lues, filtré par category si non vide.
func (r *NotificationsRepo) MarkAllRead(ctx context.Context, category notifications.Category) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, notifWriteTimeout)
	defer cancel()

	rwDB, err := OpenReadWrite(r.pdb.Player.Path())
	if err != nil {
		return 0, fmt.Errorf("NotificationsRepo.MarkAllRead: open rw: %w", err)
	}
	defer rwDB.Close()

	now := time.Now().UTC()
	var res sql.Result
	if category == "" {
		res, err = rwDB.Exec(ctx,
			`UPDATE player_notifications SET read_at = ? WHERE read_at IS NULL`, now)
	} else {
		res, err = rwDB.Exec(ctx,
			`UPDATE player_notifications SET read_at = ? WHERE read_at IS NULL AND category = ?`,
			now, string(category))
	}
	if err != nil {
		return 0, fmt.Errorf("NotificationsRepo.MarkAllRead: exec: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// Delete supprime une notif.
func (r *NotificationsRepo) Delete(ctx context.Context, id int64) error {
	ctx, cancel := context.WithTimeout(ctx, notifWriteTimeout)
	defer cancel()

	rwDB, err := OpenReadWrite(r.pdb.Player.Path())
	if err != nil {
		return fmt.Errorf("NotificationsRepo.Delete: open rw: %w", err)
	}
	defer rwDB.Close()

	res, err := rwDB.Exec(ctx, `DELETE FROM player_notifications WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("NotificationsRepo.Delete: exec: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return notifications.ErrNotFound
	}
	return nil
}

// CapAndSweep purge les notifs au-delà du cap (best-effort, log si erreur).
func (r *NotificationsRepo) CapAndSweep(ctx context.Context, max int) error {
	if max <= 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, notifWriteTimeout)
	defer cancel()

	rwDB, err := OpenReadWrite(r.pdb.Player.Path())
	if err != nil {
		slog.Warn("NotificationsRepo.CapAndSweep: open rw", "err", err)
		return nil
	}
	defer rwDB.Close()

	_, err = rwDB.Exec(ctx, `
		DELETE FROM player_notifications
		WHERE id NOT IN (
			SELECT id FROM player_notifications
			ORDER BY created_at DESC
			LIMIT ?
		)
	`, max)
	if err != nil {
		slog.Warn("NotificationsRepo.CapAndSweep: exec", "err", err)
	}
	return nil
}

// GetPreferences retourne l'état complet de notification_preferences.
func (r *NotificationsRepo) GetPreferences(ctx context.Context) ([]notifications.Preference, error) {
	ctx, cancel := context.WithTimeout(ctx, notifReadTimeout)
	defer cancel()

	rows, err := r.pdb.ReadDB().Query(ctx,
		`SELECT category, enabled, delivery FROM notification_preferences ORDER BY category`)
	if err != nil {
		return nil, fmt.Errorf("NotificationsRepo.GetPreferences: %w", err)
	}
	defer rows.Close()

	var out []notifications.Preference
	for rows.Next() {
		var p notifications.Preference
		var cat, delivery string
		if err := rows.Scan(&cat, &p.Enabled, &delivery); err != nil {
			return nil, err
		}
		p.Category = notifications.Category(cat)
		p.Delivery = notifications.Delivery(delivery)
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpsertPreferences applique les changements (insert ou update par catégorie).
func (r *NotificationsRepo) UpsertPreferences(ctx context.Context, prefs []notifications.Preference) error {
	if len(prefs) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, notifWriteTimeout)
	defer cancel()

	rwDB, err := OpenReadWrite(r.pdb.Player.Path())
	if err != nil {
		return fmt.Errorf("NotificationsRepo.UpsertPreferences: open rw: %w", err)
	}
	defer rwDB.Close()

	now := time.Now().UTC()
	for _, p := range prefs {
		_, err := rwDB.Exec(ctx, `
			INSERT INTO notification_preferences (category, enabled, delivery, updated_at)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (category) DO UPDATE SET
				enabled    = EXCLUDED.enabled,
				delivery   = EXCLUDED.delivery,
				updated_at = EXCLUDED.updated_at
		`, string(p.Category), p.Enabled, string(p.Delivery), now)
		if err != nil {
			return fmt.Errorf("NotificationsRepo.UpsertPreferences (%s): %w", p.Category, err)
		}
	}
	return nil
}

// IsCategoryEnabled vérifie la pref. Si la catégorie n'a pas d'entrée, considère TRUE par défaut.
func (r *NotificationsRepo) IsCategoryEnabled(ctx context.Context, c notifications.Category) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, notifReadTimeout)
	defer cancel()
	return isCategoryEnabledOn(ctx, r.pdb.ReadDB(), c)
}

// Compile-time check.
var _ notifications.Repository = (*NotificationsRepo)(nil)
