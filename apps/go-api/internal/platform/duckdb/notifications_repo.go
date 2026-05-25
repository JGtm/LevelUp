// Package duckdb — NotificationsRepo : persistance des notifications in-app.
//
// Implémente notifications.Repository (port défini dans internal/notifications/port.go).
// Tables : player_notifications, notification_preferences, player_records.
//
// Stockage : `shared_social.duckdb` (multi-joueur), avec colonne `xuid` en
// première position de la PK pour scoper toutes les opérations au joueur courant.
// Permet le fan-out cross-joueur (ex: media_added qui notifie plusieurs joueurs)
// sans avoir à gérer N connexions DB.
package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/notifications"
)

// NotificationsRepo persiste les notifs et préférences d'un joueur.
// Toutes les opérations sont scopées par xuid (lu depuis le PlayerDB).
type NotificationsRepo struct {
	pdb  *PlayerDB
	xuid string
}

// NewNotificationsRepo construit un NotificationsRepo depuis un PlayerDB.
// Le xuid est extrait du PlayerDB (résolu par le pool au moment de l'ouverture).
func NewNotificationsRepo(pdb *PlayerDB) *NotificationsRepo {
	xuid := ""
	if pdb != nil {
		xuid = pdb.XUID
	}
	return &NotificationsRepo{pdb: pdb, xuid: xuid}
}

const (
	notifWriteTimeout = 10 * time.Second
	notifReadTimeout  = 15 * time.Second
)

// sharedSocialPath retourne le chemin de la base shared_social pour ce repo.
// Renvoie "" si la base n'est pas attachée (cas dégradé : aucune opération possible).
func (r *NotificationsRepo) sharedSocialPath() string {
	if r.pdb == nil || r.pdb.SharedSocial == nil {
		return ""
	}
	return r.pdb.SharedSocial.Path()
}

// readDB retourne la connexion read-only sur shared_social pour ce repo.
// Renvoie nil si la base n'est pas attachée (les méthodes qui en dépendent
// court-circuitent gracieusement).
func (r *NotificationsRepo) readDB() *DB {
	if r.pdb == nil {
		return nil
	}
	return r.pdb.SharedSocial
}

// errNoSocial est l'erreur retournée si la base shared_social n'est pas attachée.
var errNoSocial = fmt.Errorf("notifications: shared_social DB not attached")

// Insert persiste une notification. Vérifie la pref via xuid+catégorie.
// Si OFF, retourne notifications.ErrCategoryDisabled (no-op pour le service).
//
// L'écriture est protégée par WithReopenOnInvalidated : si la connexion
// shared_social a été invalidée par un bug DuckDB transitoire, on tente un
// reopen automatique avant de remonter l'erreur au caller.
func (r *NotificationsRepo) Insert(ctx context.Context, n *notifications.Notification) error {
	if r.sharedSocialPath() == "" {
		return errNoSocial
	}
	ctx, cancel := context.WithTimeout(ctx, notifWriteTimeout)
	defer cancel()

	// Vérification préférence catégorie (lecture, pas d'écriture).
	if r.readDB() == nil {
		return errNoSocial
	}
	enabled, err := isCategoryEnabledOn(ctx, r.readDB(), r.xuid, n.Category)
	if err != nil {
		return fmt.Errorf("NotificationsRepo.Insert: check pref: %w", err)
	}
	if !enabled {
		return notifications.ErrCategoryDisabled
	}

	// ADR 0022 : route via Persister (CHECKPOINT garanti + TX atomique +
	// pas de réouverture concurrente OpenReadWrite qui crée un handle
	// supplémentaire).
	if r.pdb.SocialPersister != nil {
		var actorXUID, actorName *string
		if n.Actor != nil {
			ax := n.Actor.XUID
			actorXUID = &ax
			an := n.Actor.Name
			actorName = &an
		}
		bodyKey := optionalStr(n.BodyKey)
		params := optionalJSONRaw(n.Params)
		targetRoute := optionalStr(n.TargetRoute)
		targetSearch := optionalJSONRaw(n.TargetSearch)
		return r.pdb.SocialPersister.CreateNotification(ctx, NotificationData{
			XUID:         r.xuid,
			ID:           n.ID,
			Category:     string(n.Category),
			Severity:     string(n.Severity),
			TitleKey:     n.TitleKey,
			BodyKey:      bodyKey,
			Params:       params,
			TargetRoute:  targetRoute,
			TargetSearch: targetSearch,
			ActorXUID:    actorXUID,
			ActorName:    actorName,
			Source:       n.Source,
			CreatedAt:    n.CreatedAt.UTC(),
		})
	}

	// Fallback legacy (tests sans wiring).
	rwDB, err := OpenReadWrite(r.sharedSocialPath())
	if err != nil {
		return fmt.Errorf("NotificationsRepo.Insert: open rw: %w", err)
	}
	defer rwDB.Close()

	var actorXUID, actorName any
	if n.Actor != nil {
		actorXUID = n.Actor.XUID
		actorName = n.Actor.Name
	}

	err = rwDB.WithReopenOnInvalidated(func() error {
		_, execErr := rwDB.Exec(ctx, `
			INSERT INTO player_notifications
				(xuid, id, category, severity, title_key, body_key, params,
				 target_route, target_search, actor_xuid, actor_name,
				 source, created_at, read_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)
		`,
			r.xuid, n.ID, string(n.Category), string(n.Severity), n.TitleKey,
			nullableString(n.BodyKey),
			nullableJSON(n.Params),
			nullableString(n.TargetRoute),
			nullableJSON(n.TargetSearch),
			actorXUID, actorName,
			n.Source, n.CreatedAt.UTC(),
		)
		return execErr
	})
	if err != nil {
		return fmt.Errorf("NotificationsRepo.Insert: exec: %w", err)
	}
	return nil
}

// optionalStr retourne nil si la string est vide, sinon un pointeur.
func optionalStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// optionalJSONRaw retourne nil si le json.RawMessage est vide, sinon un
// pointeur string contenant la version sérialisée.
func optionalJSONRaw(v json.RawMessage) *string {
	if len(v) == 0 {
		return nil
	}
	s := string(v)
	return &s
}

// List paginé scopé par xuid.
//
// Lecture sensible à l'invalidation : si la connexion partagée est dans
// l'état fatal observé en prod (log 2026-05-14), on tente un reopen
// automatique avant d'échouer.
func (r *NotificationsRepo) List(ctx context.Context, f notifications.ListFilter) (notifications.ListResult, error) {
	if r.readDB() == nil {
		return notifications.ListResult{Items: []notifications.Notification{}}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, notifReadTimeout)
	defer cancel()

	q, args := buildListQuery(r.xuid, f)
	var rows *sql.Rows
	err := r.readDB().WithReopenOnInvalidated(func() error {
		var qerr error
		rows, qerr = r.readDB().Query(ctx, q, args...)
		return qerr
	})
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

// UnreadCount retourne le total non-lu et la répartition par catégorie pour ce joueur.
func (r *NotificationsRepo) UnreadCount(ctx context.Context) (notifications.UnreadCount, error) {
	if r.readDB() == nil {
		return notifications.UnreadCount{ByCategory: map[string]int{}}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, notifReadTimeout)
	defer cancel()

	var rows *sql.Rows
	err := r.readDB().WithReopenOnInvalidated(func() error {
		var qerr error
		rows, qerr = r.readDB().Query(ctx, `
			SELECT category, COUNT(*) AS n
			FROM player_notifications
			WHERE xuid = ? AND read_at IS NULL
			GROUP BY category
		`, r.xuid)
		return qerr
	})
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

// MarkRead positionne read_at = NOW() pour tous les IDs fournis (idempotent), scope xuid.
func (r *NotificationsRepo) MarkRead(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 || r.sharedSocialPath() == "" {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(ctx, notifWriteTimeout)
	defer cancel()

	rwDB, err := OpenReadWrite(r.sharedSocialPath())
	if err != nil {
		return 0, fmt.Errorf("NotificationsRepo.MarkRead: open rw: %w", err)
	}
	defer rwDB.Close()

	q, args := buildMarkReadQuery(r.xuid, ids)
	var res sql.Result
	err = rwDB.WithReopenOnInvalidated(func() error {
		var execErr error
		res, execErr = rwDB.Exec(ctx, q, args...)
		return execErr
	})
	if err != nil {
		return 0, fmt.Errorf("NotificationsRepo.MarkRead: exec: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// MarkUnread remet read_at = NULL (scope xuid).
func (r *NotificationsRepo) MarkUnread(ctx context.Context, id int64) error {
	if r.sharedSocialPath() == "" {
		return errNoSocial
	}
	ctx, cancel := context.WithTimeout(ctx, notifWriteTimeout)
	defer cancel()

	rwDB, err := OpenReadWrite(r.sharedSocialPath())
	if err != nil {
		return fmt.Errorf("NotificationsRepo.MarkUnread: open rw: %w", err)
	}
	defer rwDB.Close()

	var res sql.Result
	err = rwDB.WithReopenOnInvalidated(func() error {
		var execErr error
		res, execErr = rwDB.Exec(ctx,
			`UPDATE player_notifications SET read_at = NULL WHERE xuid = ? AND id = ?`,
			r.xuid, id,
		)
		return execErr
	})
	if err != nil {
		return fmt.Errorf("NotificationsRepo.MarkUnread: exec: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return notifications.ErrNotFound
	}
	return nil
}

// MarkAllRead applique read_at sur toutes les non-lues du joueur, filtré par category si non vide.
func (r *NotificationsRepo) MarkAllRead(ctx context.Context, category notifications.Category) (int, error) {
	if r.sharedSocialPath() == "" {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(ctx, notifWriteTimeout)
	defer cancel()

	rwDB, err := OpenReadWrite(r.sharedSocialPath())
	if err != nil {
		return 0, fmt.Errorf("NotificationsRepo.MarkAllRead: open rw: %w", err)
	}
	defer rwDB.Close()

	now := time.Now().UTC()
	var res sql.Result
	err = rwDB.WithReopenOnInvalidated(func() error {
		var execErr error
		if category == "" {
			res, execErr = rwDB.Exec(ctx,
				`UPDATE player_notifications SET read_at = ? WHERE xuid = ? AND read_at IS NULL`,
				now, r.xuid)
		} else {
			res, execErr = rwDB.Exec(ctx,
				`UPDATE player_notifications SET read_at = ? WHERE xuid = ? AND read_at IS NULL AND category = ?`,
				now, r.xuid, string(category))
		}
		return execErr
	})
	if err != nil {
		return 0, fmt.Errorf("NotificationsRepo.MarkAllRead: exec: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// Delete supprime une notif (scope xuid).
func (r *NotificationsRepo) Delete(ctx context.Context, id int64) error {
	if r.sharedSocialPath() == "" {
		return errNoSocial
	}
	ctx, cancel := context.WithTimeout(ctx, notifWriteTimeout)
	defer cancel()

	rwDB, err := OpenReadWrite(r.sharedSocialPath())
	if err != nil {
		return fmt.Errorf("NotificationsRepo.Delete: open rw: %w", err)
	}
	defer rwDB.Close()

	var res sql.Result
	err = rwDB.WithReopenOnInvalidated(func() error {
		var execErr error
		res, execErr = rwDB.Exec(ctx,
			`DELETE FROM player_notifications WHERE xuid = ? AND id = ?`,
			r.xuid, id,
		)
		return execErr
	})
	if err != nil {
		return fmt.Errorf("NotificationsRepo.Delete: exec: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return notifications.ErrNotFound
	}
	return nil
}

// CapAndSweep purge les notifs au-delà du cap (best-effort, log si erreur), scope xuid.
func (r *NotificationsRepo) CapAndSweep(ctx context.Context, max int) error {
	if max <= 0 || r.sharedSocialPath() == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, notifWriteTimeout)
	defer cancel()

	rwDB, err := OpenReadWrite(r.sharedSocialPath())
	if err != nil {
		slog.Warn("NotificationsRepo.CapAndSweep: open rw", "err", err)
		return nil
	}
	defer rwDB.Close()

	err = rwDB.WithReopenOnInvalidated(func() error {
		_, execErr := rwDB.Exec(ctx, `
			DELETE FROM player_notifications
			WHERE xuid = ? AND id NOT IN (
				SELECT id FROM player_notifications
				WHERE xuid = ?
				ORDER BY created_at DESC
				LIMIT ?
			)
		`, r.xuid, r.xuid, max)
		return execErr
	})
	if err != nil {
		slog.Warn("NotificationsRepo.CapAndSweep: exec", "err", err)
	}
	return nil
}

// GetPreferences retourne l'état complet de notification_preferences pour ce joueur.
func (r *NotificationsRepo) GetPreferences(ctx context.Context) ([]notifications.Preference, error) {
	if r.readDB() == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, notifReadTimeout)
	defer cancel()

	var rows *sql.Rows
	err := r.readDB().WithReopenOnInvalidated(func() error {
		var qerr error
		rows, qerr = r.readDB().Query(ctx,
			`SELECT category, enabled, delivery FROM notification_preferences WHERE xuid = ? ORDER BY category`,
			r.xuid,
		)
		return qerr
	})
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

// UpsertPreferences applique les changements (insert ou update par xuid+catégorie).
func (r *NotificationsRepo) UpsertPreferences(ctx context.Context, prefs []notifications.Preference) error {
	if len(prefs) == 0 {
		return nil
	}
	if r.sharedSocialPath() == "" {
		return errNoSocial
	}
	ctx, cancel := context.WithTimeout(ctx, notifWriteTimeout)
	defer cancel()

	rwDB, err := OpenReadWrite(r.sharedSocialPath())
	if err != nil {
		return fmt.Errorf("NotificationsRepo.UpsertPreferences: open rw: %w", err)
	}
	defer rwDB.Close()

	now := time.Now().UTC()
	for _, p := range prefs {
		pref := p // capture pour la closure
		err := rwDB.WithReopenOnInvalidated(func() error {
			_, execErr := rwDB.Exec(ctx, `
				INSERT INTO notification_preferences (xuid, category, enabled, delivery, updated_at)
				VALUES (?, ?, ?, ?, ?)
				ON CONFLICT (xuid, category) DO UPDATE SET
					enabled    = EXCLUDED.enabled,
					delivery   = EXCLUDED.delivery,
					updated_at = EXCLUDED.updated_at
			`, r.xuid, string(pref.Category), pref.Enabled, string(pref.Delivery), now)
			return execErr
		})
		if err != nil {
			return fmt.Errorf("NotificationsRepo.UpsertPreferences (%s): %w", pref.Category, err)
		}
	}
	return nil
}

// IsCategoryEnabled vérifie la pref. Default-on si pas d'entrée.
func (r *NotificationsRepo) IsCategoryEnabled(ctx context.Context, c notifications.Category) (bool, error) {
	if r.readDB() == nil {
		return true, nil
	}
	ctx, cancel := context.WithTimeout(ctx, notifReadTimeout)
	defer cancel()
	return isCategoryEnabledOn(ctx, r.readDB(), r.xuid, c)
}

// Compile-time check.
var _ notifications.Repository = (*NotificationsRepo)(nil)
