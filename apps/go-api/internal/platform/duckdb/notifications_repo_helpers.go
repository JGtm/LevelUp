package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/notifications"
)

func nowUTC() time.Time { return time.Now().UTC() }

// nullableString retourne nil pour une chaîne vide (insère NULL en base).
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullableJSON sérialise un []byte JSON pour la colonne VARCHAR :
// nil/vide → NULL, sinon string(b).
func nullableJSON(b json.RawMessage) any {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}

// dbExecutor est l'interface commune entre *DB et un sql.DB direct
// (utile quand on veut partager une vérification de pref entre Insert
// — sur une connexion RW déjà ouverte — et IsCategoryEnabled — sur la RO).
type dbExecutor interface {
	QueryRow(ctx context.Context, query string, args ...any) *sql.Row
}

// isCategoryEnabledOn lit la pref pour un xuid+catégorie. Default-on si pas d'entrée.
func isCategoryEnabledOn(ctx context.Context, db dbExecutor, xuid string, c notifications.Category) (bool, error) {
	var enabled sql.NullBool
	err := db.QueryRow(ctx,
		`SELECT enabled FROM notification_preferences_latest WHERE xuid = ? AND category = ?`,
		xuid, string(c),
	).Scan(&enabled)
	switch err {
	case nil:
		return enabled.Bool, nil
	case sql.ErrNoRows:
		return true, nil
	default:
		return false, err
	}
}

// buildListQuery construit la requête List scopée par xuid avec ses paramètres positionnels.
// Tri par created_at DESC, id DESC en tiebreaker (cursor stable).
func buildListQuery(xuid string, f notifications.ListFilter) (string, []any) {
	conds := []string{"xuid = ?"}
	args := []any{xuid}
	if f.UnreadOnly {
		conds = append(conds, "read_at IS NULL")
	}
	if f.Category != "" {
		conds = append(conds, "category = ?")
		args = append(args, string(f.Category))
	}
	if f.BeforeID > 0 {
		conds = append(conds, "id < ?")
		args = append(args, f.BeforeID)
	}
	where := "WHERE " + strings.Join(conds, " AND ")
	limit := f.Limit
	if limit <= 0 {
		limit = notifications.DefaultListLimit
	}
	q := fmt.Sprintf(`
		SELECT id, category, severity, title_key, body_key, params,
		       target_route, target_search, actor_xuid, actor_name,
		       source, created_at, read_at
		FROM player_notifications_latest
		%s
		ORDER BY created_at DESC, id DESC
		LIMIT %d
	`, where, limit)
	return q, args
}

// buildMarkReadQuery construit l'INSERT…SELECT carry-forward pour MarkRead
// (APPEND-ONLY : plus d'UPDATE in-place), scopé par xuid + clause IN dynamique.
func buildMarkReadQuery(xuid string, ids []int64) (string, []any) {
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+2)
	args = append(args, nowUTC(), xuid)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}
	q := fmt.Sprintf(`
		INSERT INTO player_notifications_history (
			xuid, id, category, severity, title_key, body_key, params,
			target_route, target_search, actor_xuid, actor_name, source,
			created_at, read_at, is_deleted, written_at
		)
		SELECT xuid, id, category, severity, title_key, body_key, params,
		       target_route, target_search, actor_xuid, actor_name, source,
		       created_at, ?, FALSE, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		FROM player_notifications_latest
		WHERE xuid = ? AND read_at IS NULL AND id IN (%s)
	`, strings.Join(placeholders, ","))
	return q, args
}

// scanNotifications scanne le résultat d'une requête List.
func scanNotifications(rows *sql.Rows) ([]notifications.Notification, error) {
	var out []notifications.Notification
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func scanNotification(rows *sql.Rows) (notifications.Notification, error) {
	var (
		n            notifications.Notification
		category     string
		severity     string
		bodyKey      sql.NullString
		params       sql.NullString
		targetRoute  sql.NullString
		targetSearch sql.NullString
		actorXUID    sql.NullString
		actorName    sql.NullString
		readAt       sql.NullTime
	)
	if err := rows.Scan(
		&n.ID, &category, &severity, &n.TitleKey,
		&bodyKey, &params,
		&targetRoute, &targetSearch,
		&actorXUID, &actorName,
		&n.Source, &n.CreatedAt, &readAt,
	); err != nil {
		return n, fmt.Errorf("notifications scan: %w", err)
	}
	n.Category = notifications.Category(category)
	n.Severity = notifications.Severity(severity)
	if bodyKey.Valid {
		n.BodyKey = bodyKey.String
	}
	if params.Valid {
		n.Params = json.RawMessage(params.String)
	}
	if targetRoute.Valid {
		n.TargetRoute = targetRoute.String
	}
	if targetSearch.Valid {
		n.TargetSearch = json.RawMessage(targetSearch.String)
	}
	if actorXUID.Valid || actorName.Valid {
		n.Actor = &notifications.Actor{XUID: actorXUID.String, Name: actorName.String}
	}
	if readAt.Valid {
		t := readAt.Time
		n.ReadAt = &t
	}
	return n, nil
}
