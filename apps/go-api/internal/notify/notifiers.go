// notifiers.go — Points d'entrée publics failsafe pour les notifications Discord.
//
// Toutes les fonctions sont failsafe : elles ne paniquent jamais et ne
// propagent jamais d'erreur à la couche appelante.
package notify

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ─────────────────────────────────────────────────────────────────────────────
// NotifySync — fin de sync ou backfill
// ─────────────────────────────────────────────────────────────────────────────

// NotifySync envoie la notification Discord de fin d'opération sync ou backfill.
//
// op accepte : "sync_delta", "sync_full", "backfill" (ou tout préfixe).
// skipIdle=true envoie un embed allégé si aucun joueur n'a de nouveaux matchs.
func NotifySync(
	cfg NotifyConfig,
	op string,
	startedAt, finishedAt time.Time,
	players []PlayerSyncResult,
	success bool,
	skipIdle bool,
) {
	ctx := context.Background()
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "discord_sync_panic", "op", op, "recover", fmt.Sprintf("%v", r))
		}
	}()

	if cfg.WebhookURL == "" {
		return
	}
	if strings.HasPrefix(op, "backfill") && !cfg.NotifyBackfill {
		slog.DebugContext(ctx, "discord_sync_skip_disabled", "op", op, "reason", "NotifyBackfill=false")
		return
	}
	if !strings.HasPrefix(op, "backfill") && !cfg.NotifySync {
		slog.DebugContext(ctx, "discord_sync_skip_disabled", "op", op, "reason", "NotifySync=false")
		return
	}
	if len(players) == 0 {
		slog.DebugContext(ctx, "discord_sync_skip_no_players", "op", op)
		return
	}

	labels := labelsOrDefault(cfg.Labels)

	// Mode skipIdle : embed allégé si tous les joueurs sont à jour
	if skipIdle && allIdle(players) {
		slog.DebugContext(ctx, "discord_sync_idle", "op", op, "players", len(players))
		embed := BuildSyncEmbedWithLabels(op, startedAt, finishedAt, players, success, cfg.Lang, labels)
		if SendWebhook(cfg.WebhookURL, WebhookPayload{Embeds: []Embed{embed}}) {
			slog.InfoContext(ctx, "discord_sync_idle_sent", "op", op)
		}
		return
	}

	embed := BuildSyncEmbedWithLabels(op, startedAt, finishedAt, players, success, cfg.Lang, labels)
	if SendWebhook(cfg.WebhookURL, WebhookPayload{Embeds: []Embed{embed}}) {
		slog.InfoContext(ctx, "discord_sync_sent", "op", op, "players", len(players))
	} else {
		slog.WarnContext(ctx, "discord_sync_not_received", "op", op)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// NotifyNewMedia — nouveaux médias avec anti-spam DuckDB
// ─────────────────────────────────────────────────────────────────────────────

// NotifyNewMedia envoie une notification Discord pour les médias indexés
// mais pas encore notifiés (discord_notified_at IS NULL dans media_files).
//
// Anti-spam DuckDB : seuls les médias avec discord_notified_at IS NULL sont
// notifiés. Après envoi réussi, discord_notified_at est mis à jour dans la DB.
// dbPath est le chemin vers shared_social.duckdb (fallback : stats.duckdb du joueur).
func NotifyNewMedia(cfg NotifyConfig, dbPath, gamertag string) {
	ctx := context.Background()
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "discord_media_panic", "op", "media", "gamertag", gamertag, "recover", fmt.Sprintf("%v", r))
		}
	}()

	if cfg.WebhookURL == "" || !cfg.NotifyNewMedia {
		return
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		slog.WarnContext(ctx, "discord_media_db_open_failed", "op", "media", "gamertag", gamertag, "err", err)
		return
	}
	defer db.Close()

	rows, err := queryUnnotifiedMedia(ctx, db)
	if err != nil || len(rows) == 0 {
		return
	}

	slog.InfoContext(ctx, "discord_media_new", "op", "media", "gamertag", gamertag, "count", len(rows))

	embed := buildMediaEmbed(rows, gamertag, cfg.Lang, cfg.Labels)
	if !SendWebhook(cfg.WebhookURL, WebhookPayload{Embeds: []Embed{embed}}) {
		slog.WarnContext(ctx, "discord_media_send_failed", "op", "media", "gamertag", gamertag)
		return
	}

	paths := make([]string, len(rows))
	for i, r := range rows {
		paths[i] = r.FilePath
	}
	if err := markMediaNotified(ctx, db, paths); err != nil {
		slog.ErrorContext(ctx, "discord_media_mark_failed", "op", "media", "gamertag", gamertag, "err", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Types + requêtes médias
// ─────────────────────────────────────────────────────────────────────────────

// mediaKindVideo est la valeur du champ Kind pour un média vidéo (mp4, clip).
// Le pendant mediaKindImage est implicite (toute autre valeur).
const mediaKindVideo = "video"

type mediaRow struct {
	FilePath string
	FileName string
	Kind     string // "video" | "image"
	MatchID  string
}

func queryUnnotifiedMedia(ctx context.Context, db *sql.DB) ([]mediaRow, error) {
	q := `
		SELECT
			mf.file_path,
			mf.file_name,
			mf.kind,
			COALESCE(mma.match_id, '') AS match_id
		FROM media_files mf
		LEFT JOIN media_match_associations_latest mma
			ON mma.media_file_id = mf.id
		WHERE mf.discord_notified_at IS NULL
		ORDER BY mf.indexed_at DESC
		LIMIT 10
	`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		// Table absente ou vide : pas une erreur critique
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	var result []mediaRow
	for rows.Next() {
		var r mediaRow
		if err := rows.Scan(&r.FilePath, &r.FileName, &r.Kind, &r.MatchID); err != nil {
			continue
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func markMediaNotified(ctx context.Context, db *sql.DB, filePaths []string) error {
	if len(filePaths) == 0 {
		return nil
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	placeholders := make([]string, len(filePaths))
	args := make([]any, len(filePaths)+1)
	args[0] = now
	for i, p := range filePaths {
		placeholders[i] = "?"
		args[i+1] = p
	}
	q := fmt.Sprintf(
		"UPDATE media_files SET discord_notified_at = ? WHERE file_path IN (%s)",
		strings.Join(placeholders, ", "),
	)
	_, err := db.ExecContext(ctx, q, args...)
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// BuildMediaEmbed — embed pour les médias
// ─────────────────────────────────────────────────────────────────────────────

func buildMediaEmbed(rows []mediaRow, gamertag, lang string, labels NotifyLabels) Embed {
	nVideos, nImages := 0, 0
	for _, r := range rows {
		if r.Kind == mediaKindVideo {
			nVideos++
		} else {
			nImages++
		}
	}

	title := T("discord_media_title_fr", lang, "gamertag", gamertag)

	var desc string
	switch {
	case nVideos > 0 && nImages > 0:
		desc = T("discord_media_desc_both", lang, "nv", nVideos, "ni", nImages)
	case nVideos > 0:
		desc = T("discord_media_desc_video", lang, "n", nVideos)
	default:
		desc = T("discord_media_desc_image", lang, "n", nImages)
	}

	embed := Embed{
		Title:       title,
		Description: desc,
		Color:       colorBlurple,
		Footer:      &EmbedFooter{Text: discordFooterText(labels)},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	// Max 6 fields + overflow
	maxFields := 6
	for i, r := range rows {
		if i >= maxFields {
			remaining := len(rows) - maxFields
			embed.Fields = append(embed.Fields, EmbedField{
				Name:  "…",
				Value: fmt.Sprintf("%d autre(s)", remaining),
			})
			break
		}
		icon := "📸"
		if r.Kind == mediaKindVideo {
			icon = "🎬"
		}
		value := r.FileName
		if r.MatchID != "" {
			value += fmt.Sprintf("\n🎮 `%s`", r.MatchID[:min(len(r.MatchID), 16)])
		}
		embed.Fields = append(embed.Fields, EmbedField{
			Name:   icon + " " + truncate(r.FileName, 50),
			Value:  truncate(value, 256),
			Inline: true,
		})
	}

	return embed
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─────────────────────────────────────────────────────────────────────────────
// §6.B — Flow ami (Squad/Sessions overhaul)
// ─────────────────────────────────────────────────────────────────────────────

// NotifyFriendAdded envoie une notification Discord quand un gamertag est
// ajouté à friend_gamertags via PATCH /settings.
//
// Failsafe : panic récupéré, webhook vide / NotifyFriends off → no-op silencieux.
func NotifyFriendAdded(cfg NotifyConfig, gamertag string) {
	ctx := context.Background()
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "discord_friend_added_panic", "op", "friend_added", "gamertag", gamertag, "recover", fmt.Sprintf("%v", r))
		}
	}()

	if cfg.WebhookURL == "" || !cfg.NotifyFriends {
		return
	}

	embed := Embed{
		Title:       T("discord_friend_added_title", cfg.Lang),
		Description: T("discord_friend_added_desc", cfg.Lang, "gamertag", gamertag),
		Color:       colorBlurple,
		Footer:      &EmbedFooter{Text: discordFooterText(cfg.Labels)},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
	if SendWebhook(cfg.WebhookURL, WebhookPayload{Embeds: []Embed{embed}}) {
		slog.InfoContext(ctx, "discord_friend_added_sent", "op", "friend_added", "gamertag", gamertag)
	}
}

// NotifyFriendSyncCompleted envoie une notification Discord quand un recompute
// is_with_friends a promu au moins 1 match pour le slug joueur indiqué.
//
// `promoted` est passé tel quel dans les params i18n (ICU plural-aware côté
// front, mais le template Discord switch FR/EN utilise simple/many).
//
// Failsafe : panic récupéré, webhook vide / NotifyFriends off / promoted ≤ 0 → no-op.
func NotifyFriendSyncCompleted(cfg NotifyConfig, slug string, promoted int64) {
	ctx := context.Background()
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "discord_friend_sync_panic", "op", "friend_sync", "slug", slug, "recover", fmt.Sprintf("%v", r))
		}
	}()

	if cfg.WebhookURL == "" || !cfg.NotifyFriends || promoted <= 0 {
		return
	}

	descKey := "discord_friend_sync_desc_many"
	if promoted == 1 {
		descKey = "discord_friend_sync_desc_one"
	}
	embed := Embed{
		Title:       T("discord_friend_sync_title", cfg.Lang),
		Description: T(descKey, cfg.Lang, "promoted", promoted, "slug", slug),
		Color:       colorSuccess,
		Footer:      &EmbedFooter{Text: discordFooterText(cfg.Labels)},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
	if SendWebhook(cfg.WebhookURL, WebhookPayload{Embeds: []Embed{embed}}) {
		slog.InfoContext(ctx, "discord_friend_sync_sent", "op", "friend_sync", "slug", slug, "promoted", promoted)
	}
}
