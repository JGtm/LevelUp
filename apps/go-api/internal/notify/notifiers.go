// notifiers.go — Points d'entrée publics failsafe pour les notifications Discord.
//
// Toutes les fonctions sont failsafe : elles ne paniquent jamais et ne
// propagent jamais d'erreur à la couche appelante.
package notify

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
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
