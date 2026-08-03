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

// Le gate de rollout MULTI_TITLE_API_ENABLED (loadMultiTitleAPIEnabled) a été
// RETIRÉ le 2026-08-02 (v7.3 lot 2, item 3.3). Son critère de bascule documenté
// — « surface multi-titre validée pour >= 2 titres » — était atteint et le flag
// valait true dans tous les environnements (prod via app_settings.json, démo via
// le compose, CI via les deux jobs go test). Les routes /titles/{slug}/* sont
// désormais montées inconditionnellement (server_apiv1.go) et leurs libellés sont
// la source unique du front. Ne pas réintroduire de flag ici : ce serait rendre
// éteignable un affichage qui n'a plus de repli.

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
