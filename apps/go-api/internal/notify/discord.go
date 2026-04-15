// Package notify — Client Discord webhook + types de base + i18n inline.
//
// Tout le package est failsafe : aucune fonction exportée ne panic ni ne lève
// d'erreur vers la couche appelante (sauf NotifyNewVersion qui retourne bool).
package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Types Discord webhook
// ─────────────────────────────────────────────────────────────────────────────

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
	// NotifyNewMedia active les notifications de nouveaux médias.
	NotifyNewMedia bool
	// NotifyVersion active les notifications de nouvelle version.
	// Opt-in explicite : requiert aussi l'env var LEVELUP_NOTIFY_VERSIONS=1.
	NotifyVersion bool
	// SettingsPath est le chemin vers app_settings.json pour l'anti-spam de version.
	SettingsPath string
}

// LoadNotifyConfig charge la configuration Discord depuis app_settings.json.
// La variable d'environnement DISCORD_WEBHOOK_URL prévaut sur le champ JSON.
func LoadNotifyConfig(settingsPath string) NotifyConfig {
	cfg := NotifyConfig{Lang: "fr", SettingsPath: settingsPath}

	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		return cfg
	}
	var s map[string]any
	if err := json.Unmarshal(raw, &s); err != nil {
		return cfg
	}

	if !boolVal(s, "discord_notifications_enabled") {
		return cfg
	}

	// Résolution webhook : env var > app_settings.json
	url := strings.TrimSpace(os.Getenv("DISCORD_WEBHOOK_URL"))
	if url == "" {
		url = strings.TrimSpace(strVal(s, "discord_webhook_url"))
	}
	if !strings.HasPrefix(url, "https://discord.com/api/webhooks/") {
		if url != "" {
			log.Printf("[Discord] URL webhook invalide (ignorée)")
		}
		url = ""
	}
	cfg.WebhookURL = url
	cfg.Lang = strValDefault(s, "discord_lang", "fr")
	cfg.NotifySync = boolValDefault(s, "discord_notify_sync", true)
	cfg.NotifyBackfill = boolValDefault(s, "discord_notify_backfill", true)
	cfg.NotifyNewMedia = boolValDefault(s, "discord_notify_new_media", true)
	cfg.NotifyVersion = boolValDefault(s, "discord_notify_new_version", true)
	return cfg
}

// ─────────────────────────────────────────────────────────────────────────────
// Envoi HTTP
// ─────────────────────────────────────────────────────────────────────────────

var httpClient = &http.Client{Timeout: 10 * time.Second}

// SendWebhook envoie un payload JSON au webhook Discord.
// Retourne true si Discord répond 200 ou 204.
func SendWebhook(webhookURL string, payload WebhookPayload) bool {
	if webhookURL == "" {
		return false
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Discord] marshal payload: %v", err)
		return false
	}
	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(data))
	if err != nil {
		log.Printf("[Discord] build request: %v", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", "LevelUp-HaloBot/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("[Discord] envoi échoué: %v", err)
		return false
	}
	defer resp.Body.Close()
	ok := resp.StatusCode == 200 || resp.StatusCode == 204
	if !ok {
		log.Printf("[Discord] réponse inattendue: HTTP %d", resp.StatusCode)
	}
	return ok
}

// ─────────────────────────────────────────────────────────────────────────────
// I18n inline
// ─────────────────────────────────────────────────────────────────────────────

// discordStrings contient toutes les chaînes bilingues FR/EN.
// Structure : map[clé]map[lang]template
var discordStrings = map[string]map[string]string{
	"discord_outcome_draw":  {"fr": "Égalité", "en": "Draw"},
	"discord_outcome_win":   {"fr": "Victoire", "en": "Win"},
	"discord_outcome_loss":  {"fr": "Défaite", "en": "Loss"},
	"discord_outcome_quit":  {"fr": "Abandon", "en": "Quit"},

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

	"discord_bf_lusr":              {"fr": "🏅  {count} LUSR calculé(s)", "en": "🏅  {count} LUSR computed"},
	"discord_bf_medals":            {"fr": "🥇  {count} médaille(s)", "en": "🥇  {count} medal(s)"},
	"discord_bf_events":            {"fr": "🎬  {count} event(s) highlight", "en": "🎬  {count} highlight event(s)"},
	"discord_bf_csr":               {"fr": "📈  {count} CSR récupéré(s)", "en": "📈  {count} CSR fetched"},
	"discord_bf_sessions":          {"fr": "📅  {count} session(s) recalculée(s)", "en": "📅  {count} session(s) updated"},
	"discord_bf_citations":         {"fr": "💬  {count} citation(s)", "en": "💬  {count} citation(s)"},
	"discord_bf_kvp":               {"fr": "⚔️  {count} paire(s) killer-victim", "en": "⚔️  {count} killer-victim pair(s)"},
	"discord_bf_personal_scores":   {"fr": "🎯  {count} personal score(s)", "en": "🎯  {count} personal score(s)"},
	"discord_bf_perf_scores":       {"fr": "⚡  {count} perf score(s)", "en": "⚡  {count} perf score(s)"},
	"discord_bf_aliases":           {"fr": "👤  {count} alias(es)", "en": "👤  {count} alias(es)"},
	"discord_bf_pve":               {"fr": "🤖  {count} stat(s) PvE", "en": "🤖  {count} PvE stat(s)"},

	"discord_last_match":   {"fr": "Dernier match", "en": "Last match"},
	"discord_ranked_tag":   {"fr": "Classé", "en": "Ranked"},
	"discord_footer":       {"fr": "LevelUp · Halo Infinite Stats", "en": "LevelUp · Halo Infinite Stats"},
	"discord_title":        {"fr": "🎮  LevelUp — {op}", "en": "🎮  LevelUp — {op}"},
	"discord_time_range":   {"fr": "🕐  `{t_start}`  →  `{t_end}`", "en": "🕐  `{t_start}`  →  `{t_end}`"},
	"discord_kda":          {"fr": "{k}F / {d}D / {a}A", "en": "{k}K / {d}D / {a}A"},
	"discord_squad_match":  {"fr": "🎮 Match en escouade", "en": "🎮 Squad match"},
	"discord_squad_friends":{"fr": "👥 Amis : {friends}", "en": "👥 Friends: {friends}"},

	"discord_up_to_date_sync":           {"fr": "Déjà à jour", "en": "Already up to date"},
	"discord_no_new_matches":            {"fr": "Aucun nouveau match", "en": "No new matches"},
	"discord_no_matches_to_reprocess":   {"fr": "Aucun match à retraiter", "en": "Nothing to reprocess"},
	"discord_all_up_to_date":            {"fr": "Tout déjà à jour", "en": "All up to date"},
	"discord_player_count":              {"fr": "👥  {count} joueur(s)", "en": "👥  {count} player(s)"},

	// Version
	"discord_version_title": {
		"fr": "🚀 LevelUp v{version} — Nouvelle version déployée",
		"en": "🚀 LevelUp v{version} — New version deployed",
	},
	"discord_version_footer": {
		"fr": "LevelUp · Mise à jour automatique",
		"en": "LevelUp · Auto-update",
	},

	// Médias
	"discord_media_title_fr": {"fr": "📸 Nouveaux médias — {gamertag}", "en": "📸 New media — {gamertag}"},
	"discord_media_desc_video": {
		"fr": "Nouvellement indexé : {n} vidéo(s)",
		"en": "Newly indexed: {n} video(s)",
	},
	"discord_media_desc_image": {
		"fr": "Nouvellement indexé : {n} capture(s)",
		"en": "Newly indexed: {n} screenshot(s)",
	},
	"discord_media_desc_both": {
		"fr": "Nouvellement indexés : {nv} vidéo(s) · {ni} capture(s)",
		"en": "Newly indexed: {nv} video(s) · {ni} screenshot(s)",
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

func boolValDefault(m map[string]any, key string, def bool) bool {
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
