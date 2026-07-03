package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func loadBackupConfig(repoRoot, settingsPath string) BackupConfig {
	cfg := BackupConfig{
		BackupDir:   getEnvOrDefault("LEVELUP_BACKUP_DIR", filepath.Join(repoRoot, "data", "backups")),
		ResticRepo:  os.Getenv("RESTIC_REPOSITORY"),
		Interval:    6 * time.Hour,
		KeepDaily:   7,
		KeepWeekly:  4,
		KeepMonthly: 12,
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return cfg
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return cfg
	}
	if v, ok := m["backup_enabled"].(bool); ok {
		cfg.Enabled = v
	}
	if v, ok := m["backup_interval"].(string); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Interval = d
		}
	}
	if v, ok := m["backup_keep_daily"].(float64); ok {
		cfg.KeepDaily = int(v)
	}
	if v, ok := m["backup_keep_weekly"].(float64); ok {
		cfg.KeepWeekly = int(v)
	}
	if v, ok := m["backup_keep_monthly"].(float64); ok {
		cfg.KeepMonthly = int(v)
	}
	return cfg
}

// loadMultiTitleAPIEnabled lit multi_title_api_enabled depuis app_settings.json.
// La var d'env MULTI_TITLE_API_ENABLED prend la priorité (override d'urgence).
// Défaut : false.
//
// Nature : gate de déploiement progressif (rollout), PAS un kill-switch de
// rollback. Cycle de vie :
//   - défaut actuel : OFF (surface API multi-titre field-mappings/preview non
//     exposée par défaut) ;
//   - critère de bascule ON : surface multi-titre validée pour >= 2 titres
//     (activation Halo 5, chantier multi-titre phase 1b) ;
//   - date cible de retrait du flag : quand le multi-titre est le comportement
//     permanent — corriger la cause et livrer actif (CLAUDE.md n°11) plutôt que
//     de conserver un flag qui laisse la feature OFF « pour plus tard ».
func loadMultiTitleAPIEnabled(settingsPath string) bool {
	if v := os.Getenv("MULTI_TITLE_API_ENABLED"); v != "" {
		vl := strings.ToLower(strings.TrimSpace(v))
		return vl == "1" || vl == "true" || vl == "yes"
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return false
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return false
	}
	if b, ok := m["multi_title_api_enabled"].(bool); ok {
		return b
	}
	return false
}

// DiscordWebhookURLFromEnv retourne le webhook Discord depuis l'environnement SEUL
// (LEVELUP_DISCORD_WEBHOOK_URL prioritaire sur DISCORD_WEBHOOK_URL, nom legacy Python).
// Source UNIQUE de la précédence env du webhook (CR A6) : consommée par le loader config
// ci-dessous ET par les résolveurs notify/validation, qui conservent leur propre fallback
// settings (map résolue par titre côté notify). Chaîne vide si aucune n'est définie.
func DiscordWebhookURLFromEnv() string {
	if url := os.Getenv("LEVELUP_DISCORD_WEBHOOK_URL"); url != "" {
		return url
	}
	return os.Getenv("DISCORD_WEBHOOK_URL")
}

// loadDiscordWebhookURL lit le webhook Discord depuis l'environnement
// (DiscordWebhookURLFromEnv) ou le champ discord_webhook_url de app_settings.json.
func loadDiscordWebhookURL(settingsPath string) string {
	if url := DiscordWebhookURLFromEnv(); url != "" {
		return url
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	if s, ok := m["discord_webhook_url"].(string); ok {
		if strings.HasPrefix(s, "https://discord.com/api/webhooks/") {
			return s
		}
	}
	return ""
}

// BootstrapEnvLocal charge `.env.local` depuis la racine du repo (résolue via
// LEVELUP_REPO_ROOT ou autoDetectRepoRoot) dans os.Environ. Idempotent : les
// variables déjà définies dans l'env du process ne sont jamais écrasées.
//
// À appeler le plus tôt possible dans main() pour que toute lecture
// ultérieure de `os.Getenv` (notamment LEVELUP_LOG_LEVEL et autres
// LEVELUP_LOG_* lus AVANT config.Load) bénéficie des valeurs locales. Sans
// ça, le logger booterait toujours en INFO/compact même si .env.local force
// LEVELUP_LOG_LEVEL=warn.
