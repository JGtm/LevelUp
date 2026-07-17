// Package notify — Client Discord webhook + types de base + i18n inline.
//
// Tout le package est failsafe : aucune fonction exportée ne panic ni ne lève
// d'erreur vers la couche appelante (sauf NotifyNewVersion qui retourne bool).
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/config"
)

// ─────────────────────────────────────────────────────────────────────────────
// Types Discord webhook
// ─────────────────────────────────────────────────────────────────────────────

// keyDiscordVersionFooter est la clé i18n du footer de notification "nouvelle
// version". Centralisée pour réduire la duplication (cf. lint goconst).
const keyDiscordVersionFooter = "discord_version_footer"

// EmbedField est un champ inline ou non d'un embed Discord.
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// EmbedFooter est le footer d'un embed Discord.
type EmbedFooter struct {
	Text string `json:"text"`
}

// EmbedImage est l'image d'un embed Discord (local ou attachment://).
type EmbedImage struct {
	URL string `json:"url"`
}

// Embed représente un Rich Embed Discord.
type Embed struct {
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	Color       int          `json:"color,omitempty"`
	Fields      []EmbedField `json:"fields,omitempty"`
	Footer      *EmbedFooter `json:"footer,omitempty"`
	Image       *EmbedImage  `json:"image,omitempty"`
	Timestamp   string       `json:"timestamp,omitempty"` // RFC3339
}

// WebhookPayload est le payload envoyé au webhook Discord.
type WebhookPayload struct {
	Embeds []Embed `json:"embeds"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Configuration
// ─────────────────────────────────────────────────────────────────────────────

// NotifyConfig centralise la configuration des notifications Discord.
type NotifyConfig struct {
	// WebhookURL est l'URL du webhook Discord (vide = notifications désactivées).
	WebhookURL string
	// Lang est la langue des embeds ("fr" ou "en").
	Lang string
	// NotifySync active les notifications de fin de sync.
	NotifySync bool
	// NotifyBackfill active les notifications de fin de backfill.
	NotifyBackfill bool
	// NotifyFriends active les notifications du flow ami (§6.B Squad/Sessions
	// overhaul) : friend_added + friend_sync_completed.
	NotifyFriends bool
	// NotifyVersion active les notifications de nouvelle version.
	// Opt-in explicite : requiert aussi l'env var LEVELUP_NOTIFY_VERSIONS=1.
	NotifyVersion bool
	// NotifyReauth active la notification « reconnexion Xbox requise » quand le
	// refresh_token d'un joueur meurt (PR-B). Défaut : true.
	NotifyReauth bool
	// NotifyDisk active les alertes disque (warn > 80 % / critical > 90 %,
	// seuils ops A5.3 — lot ops 2026-07-13, suite incident disque-plein VPS).
	// Défaut : true.
	NotifyDisk bool
	// SettingsPath est le chemin vers app_settings.json pour l'anti-spam de version.
	SettingsPath string
	// Labels fournit les libellés title-aware des embeds (PMT-11). nil → libellés
	// Halo (failsafe, byte-identique). Un caller multi-titre pose
	// notify.LabelsFor(titleSemanticAdapter) pour rendre les outcomes du titre.
	Labels NotifyLabels
}

// LoadNotifyConfig charge la configuration Discord depuis app_settings.json.
// La variable d'environnement DISCORD_WEBHOOK_URL prévaut sur le champ JSON.
func LoadNotifyConfig(settingsPath string) NotifyConfig {
	return notifyConfigFromMap(settingsPath, readSettingsMap(settingsPath))
}

// LoadNotifyConfigForTitle charge la config Discord du titre (PMT-4) : settings
// globaux + overlay du titre (champ-présent-only) lu depuis overlayPath. Le caller
// fournit les 2 chemins (`PathResolver.AppSettingsPath()` + `TitleSettingsPath(slug)`)
// → notify ne dépend pas du registre titres. overlayPath vide / absent / vide ⇒
// identique à LoadNotifyConfig (Halo byte-identique). Le gate
// `discord_notifications_enabled` est évalué sur la map RÉSOLUE (overlay > global).
func LoadNotifyConfigForTitle(settingsPath, overlayPath string) NotifyConfig {
	m := readSettingsMap(settingsPath)
	if overlayPath != "" {
		if overlay := readSettingsMap(overlayPath); len(overlay) > 0 {
			merged := make(map[string]any, len(m)+len(overlay))
			for k, v := range m {
				merged[k] = v
			}
			for k, v := range overlay {
				merged[k] = v
			}
			m = merged
		}
	}
	return notifyConfigFromMap(settingsPath, m)
}

// readSettingsMap lit un fichier JSON de settings en map. nil si absent/illisible.
func readSettingsMap(path string) map[string]any {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var s map[string]any
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil
	}
	return s
}

// notifyConfigFromMap construit le NotifyConfig à partir d'une map de settings
// (globale ou globale+overlay résolue). Logique partagée par LoadNotifyConfig et
// LoadNotifyConfigForTitle (une seule source de vérité).
func notifyConfigFromMap(settingsPath string, s map[string]any) NotifyConfig {
	cfg := NotifyConfig{Lang: "fr", SettingsPath: settingsPath}
	if s == nil || !boolVal(s, "discord_notifications_enabled") {
		return cfg
	}

	// Résolution webhook : env var (LEVELUP_DISCORD_WEBHOOK_URL > DISCORD_WEBHOOK_URL,
	// via config — source unique de précédence) > settings résolus (map par titre).
	url := strings.TrimSpace(config.DiscordWebhookURLFromEnv())
	if url == "" {
		url = strings.TrimSpace(strVal(s, "discord_webhook_url"))
	}
	if !strings.HasPrefix(url, "https://discord.com/api/webhooks/") {
		if url != "" {
			slog.WarnContext(context.Background(), "discord_webhook_url_invalid", "op", "config_load")
		}
		url = ""
	}
	cfg.WebhookURL = url
	cfg.Lang = strValDefault(s, "discord_lang", "fr")
	cfg.NotifySync = boolValDefault(s, "discord_notify_sync", true)
	cfg.NotifyBackfill = boolValDefault(s, "discord_notify_backfill", true)
	cfg.NotifyFriends = boolValDefault(s, "discord_notify_friends", true)
	cfg.NotifyVersion = boolValDefault(s, "discord_notify_new_version", true)
	cfg.NotifyReauth = boolValDefault(s, "discord_notify_reauth", true)
	cfg.NotifyDisk = boolValDefault(s, "discord_notify_disk", true)
	return cfg
}

// NotifyReauthRequired envoie un embed « reconnexion Xbox requise » pour un joueur
// dont le refresh_token est mort. Failsafe : no-op si webhook absent ou toggle off.
func NotifyReauthRequired(cfg NotifyConfig, gamertag string) {
	if cfg.WebhookURL == "" || !cfg.NotifyReauth {
		return
	}
	_ = SendWebhook(cfg.WebhookURL, WebhookPayload{Embeds: []Embed{{
		Title:       T("discord_reauth_title", cfg.Lang),
		Description: T("discord_reauth_desc", cfg.Lang, "gamertag", gamertag),
		Color:       0xE0A800, // ambre — avertissement
		Footer:      &EmbedFooter{Text: discordFooterText(cfg.Labels)},
	}}})
}

// ─────────────────────────────────────────────────────────────────────────────
// Envoi HTTP
// ─────────────────────────────────────────────────────────────────────────────

var httpClient = &http.Client{Timeout: 10 * time.Second}

// sanitizeSendError expurge l'URL du webhook (secret : le token d'écriture du canal
// est dans le path) d'une erreur d'envoi avant tout log. http.Client.Do et
// http.NewRequest retournent un *url.Error dont Error() concatène l'URL complète ;
// on ne conserve que l'opération + l'erreur interne (transport / parse), qui ne
// portent jamais le token. Toute autre erreur est renvoyée telle quelle.
func sanitizeSendError(err error) string {
	if err == nil {
		return ""
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		inner := "erreur inconnue"
		if urlErr.Err != nil {
			inner = urlErr.Err.Error()
		}
		return fmt.Sprintf("%s (webhook expurgé): %s", urlErr.Op, inner)
	}
	return err.Error()
}

// SendWebhook envoie un payload JSON au webhook Discord.
// Retourne true si Discord répond 200 ou 204.
func SendWebhook(webhookURL string, payload WebhookPayload) bool {
	ctx := context.Background()
	if webhookURL == "" {
		return false
	}
	data, err := json.Marshal(payload)
	if err != nil {
		slog.WarnContext(ctx, "discord_marshal_failed", "op", "send", "err", err)
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(data))
	if err != nil {
		slog.WarnContext(ctx, "discord_build_request_failed", "op", "send", "err", sanitizeSendError(err))
		return false
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", "LevelUp-HaloBot/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		// httpClient.Do retourne un *url.Error dont Error() concatène l'URL COMPLÈTE
		// (token du webhook dans le path = secret d'écriture du canal). On expurge
		// avant tout log (logs persistés sur le VPS, lisibles par backup/diagnostic).
		slog.WarnContext(ctx, "discord_send_failed", "op", "send", "err", sanitizeSendError(err))
		return false
	}
	defer resp.Body.Close()
	ok := resp.StatusCode == 200 || resp.StatusCode == 204
	if !ok {
		slog.WarnContext(ctx, "discord_unexpected_status", "op", "send", "status", resp.StatusCode)
	}
	return ok
}

// ─────────────────────────────────────────────────────────────────────────────
// I18n inline
// ─────────────────────────────────────────────────────────────────────────────

// discordStrings contient toutes les chaînes bilingues FR/EN.
// Structure : map[clé]map[lang]template
//
// EXEMPTION EMOJIS (D2, revue 2026-07-17) : les emojis ci-dessous sont le CONTENU des
// messages Discord (payload produit envoyé au webhook), PAS de la décoration de code
// source. Ils sont donc exemptés de la règle « pas d'emojis dans les fichiers versionnés »
// (CLAUDE.md) — au même titre qu'un libellé UI. Convention locale préexistante du fichier ;
// toute décoration de code (logs, commentaires, sortie CLI) reste, elle, sans emoji.
var discordStrings = map[string]map[string]string{
	"discord_outcome_draw": {"fr": "Égalité", "en": "Draw"},
	"discord_outcome_win":  {"fr": "Victoire", "en": "Win"},
	"discord_outcome_loss": {"fr": "Défaite", "en": "Loss"},
	"discord_outcome_quit": {"fr": "Abandon", "en": "Quit"},

	"discord_op_sync_delta": {"fr": "Sync delta", "en": "Delta sync"},
	"discord_op_sync_full":  {"fr": "Sync complète", "en": "Full sync"},
	"discord_op_backfill":   {"fr": "Backfill", "en": "Backfill"},

	"discord_completed_in": {
		"fr": "**{status}  {op}** terminée en **{duration}**",
		"en": "**{status}  {op}** completed in **{duration}**",
	},
	"discord_players_matches": {
		"fr": "👥  {players} joueur(s)  ·  {matches} match(s) traité(s)",
		"en": "👥  {players} player(s)  ·  {matches} match(es) processed",
	},
	"discord_matches_synced": {
		"fr": "**+{count}** match(s) synchronisé(s)",
		"en": "**+{count}** match(es) synced",
	},
	"discord_matches_processed": {
		"fr": "**{count}** match(s) retraité(s)",
		"en": "**{count}** match(es) reprocessed",
	},
	"discord_data_complete":   {"fr": "✅  Données complètes", "en": "✅  Data complete"},
	"discord_data_incomplete": {"fr": "⚠️   **{count}** match(s) avec données incomplètes", "en": "⚠️   **{count}** match(es) with incomplete data"},
	"discord_error_field":     {"fr": "⛔  Erreur : {error}", "en": "⛔  Error: {error}"},

	"discord_bf_lusr":            {"fr": "🏅  {count} LUSR calculé(s)", "en": "🏅  {count} LUSR computed"},
	"discord_bf_medals":          {"fr": "🥇  {count} médaille(s)", "en": "🥇  {count} medal(s)"},
	"discord_bf_events":          {"fr": "🎬  {count} event(s) highlight", "en": "🎬  {count} highlight event(s)"},
	"discord_bf_csr":             {"fr": "📈  {count} CSR récupéré(s)", "en": "📈  {count} CSR fetched"},
	"discord_bf_sessions":        {"fr": "📅  {count} session(s) recalculée(s)", "en": "📅  {count} session(s) updated"},
	"discord_bf_citations":       {"fr": "💬  {count} citation(s)", "en": "💬  {count} citation(s)"},
	"discord_bf_kvp":             {"fr": "⚔️  {count} paire(s) killer-victim", "en": "⚔️  {count} killer-victim pair(s)"},
	"discord_bf_personal_scores": {"fr": "🎯  {count} personal score(s)", "en": "🎯  {count} personal score(s)"},
	"discord_bf_perf_scores":     {"fr": "⚡  {count} perf score(s)", "en": "⚡  {count} perf score(s)"},
	"discord_bf_aliases":         {"fr": "👤  {count} alias(es)", "en": "👤  {count} alias(es)"},
	"discord_bf_pve":             {"fr": "🤖  {count} stat(s) PvE", "en": "🤖  {count} PvE stat(s)"},

	"discord_disk_warn_title":     {"fr": "💾  Disque serveur : espace faible", "en": "💾  Server disk: low space"},
	"discord_disk_critical_title": {"fr": "🚨  Disque serveur : espace CRITIQUE", "en": "🚨  Server disk: CRITICAL space"},
	"discord_disk_ok_title":       {"fr": "✅  Disque serveur : espace rétabli", "en": "✅  Server disk: space recovered"},
	"discord_disk_alert_desc": {
		"fr": "Le volume de données est rempli à **{used_pct} %** — **{free}** libres sur {total} (`{path}`). Libérer de l'espace avant saturation (incident du 2026-07-13 : prod down disque plein).",
		"en": "Data volume is **{used_pct}%** full — **{free}** free of {total} (`{path}`). Free up space before saturation.",
	},
	"discord_disk_ok_desc": {
		"fr": "Le volume de données est revenu sous les seuils d'alerte : **{free}** libres sur {total} ({used_pct} % utilisés).",
		"en": "Data volume is back under alert thresholds: **{free}** free of {total} ({used_pct}% used).",
	},

	"discord_reauth_title": {"fr": "🔑  Reconnexion Xbox requise", "en": "🔑  Xbox reconnection required"},
	"discord_reauth_desc": {
		"fr": "Le jeton de **{gamertag}** a expiré — la synchronisation est en pause. Reconnecte ton compte Xbox dans LevelUp.",
		"en": "Token for **{gamertag}** expired — sync is paused. Reconnect your Xbox account in LevelUp.",
	},

	"discord_last_match": {"fr": "Dernier match", "en": "Last match"},
	"discord_ranked_tag": {"fr": "Classé", "en": "Ranked"},
	// discord_footer retiré : le footer est dérivé du nom du titre (descripteur),
	// cf. discordFooterText() dans labels.go (source unique, évite la 2e copie).
	"discord_title":         {"fr": "🎮  LevelUp — {op}", "en": "🎮  LevelUp — {op}"},
	"discord_time_range":    {"fr": "🕐  `{t_start}`  →  `{t_end}`", "en": "🕐  `{t_start}`  →  `{t_end}`"},
	"discord_kda":           {"fr": "{k}F / {d}D / {a}A", "en": "{k}K / {d}D / {a}A"},
	"discord_squad_match":   {"fr": "🎮 Match en escouade", "en": "🎮 Squad match"},
	"discord_squad_friends": {"fr": "👥 Amis : {friends}", "en": "👥 Friends: {friends}"},

	"discord_up_to_date_sync":         {"fr": "Déjà à jour", "en": "Already up to date"},
	"discord_no_new_matches":          {"fr": "Aucun nouveau match", "en": "No new matches"},
	"discord_no_matches_to_reprocess": {"fr": "Aucun match à retraiter", "en": "Nothing to reprocess"},
	"discord_all_up_to_date":          {"fr": "Tout déjà à jour", "en": "All up to date"},
	"discord_player_count":            {"fr": "👥  {count} joueur(s)", "en": "👥  {count} player(s)"},

	// Version
	"discord_version_title": {
		"fr": "🚀 LevelUp v{version} — Nouvelle version déployée",
		"en": "🚀 LevelUp v{version} — New version deployed",
	},
	keyDiscordVersionFooter: {
		"fr": "LevelUp · Mise à jour automatique",
		"en": "LevelUp · Auto-update",
	},

	// §6.B — Flow ami (Squad/Sessions overhaul)
	"discord_friend_added_title": {
		"fr": "👤 Nouvel ami ajouté",
		"en": "👤 New friend added",
	},
	"discord_friend_added_desc": {
		"fr": "{gamertag} a été ajouté à ta liste d'amis. Les sessions de jeu communes seront automatiquement reclassées en escouade.",
		"en": "{gamertag} has been added to your friends list. Shared sessions will be automatically reclassified as squad.",
	},
	"discord_friend_sync_title": {
		"fr": "🔄 Sessions amis mises à jour",
		"en": "🔄 Friend sessions updated",
	},
	"discord_friend_sync_desc_one": {
		"fr": "{promoted} match a été reclassé en escouade-amis pour {slug}.",
		"en": "{promoted} match reclassified as squad-friends for {slug}.",
	},
	"discord_friend_sync_desc_many": {
		"fr": "{promoted} matchs ont été reclassés en escouade-amis pour {slug}.",
		"en": "{promoted} matches reclassified as squad-friends for {slug}.",
	},
}

// T traduit une clé Discord selon la langue donnée.
// Les tokens {key} sont remplacés par les valeurs de args.
func T(key, lang string, args ...any) string {
	if lang == "" {
		lang = "fr"
	}
	entry, ok := discordStrings[key]
	if !ok {
		return key
	}
	tmpl, ok := entry[lang]
	if !ok {
		tmpl = entry["fr"] // fallback français
	}
	if len(args) == 0 || len(args)%2 != 0 {
		return tmpl
	}
	// Substitution simple : args = [key1, val1, key2, val2, ...]
	for i := 0; i+1 < len(args); i += 2 {
		k := fmt.Sprintf("{%v}", args[i])
		v := fmt.Sprintf("%v", args[i+1])
		tmpl = strings.ReplaceAll(tmpl, k, v)
	}
	return tmpl
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers lecture app_settings.json
// ─────────────────────────────────────────────────────────────────────────────

func boolVal(m map[string]any, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func boolValDefault(m map[string]any, key string, def bool) bool { //nolint:unparam // def=true actuellement, conserver pour flexibilité
	v, ok := m[key]
	if !ok {
		return def
	}
	b, ok := v.(bool)
	if !ok {
		return def
	}
	return b
}

func strVal(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func strValDefault(m map[string]any, key string, def string) string {
	s := strVal(m, key)
	if s == "" {
		return def
	}
	return s
}
