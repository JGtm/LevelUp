// config_checkpoint.go — intervalle du CHECKPOINT périodique sous la capacité FrequentCheckpoint
// (profil desktop). Dans un fichier dédié : config.go est gelé en taille (config_size_ratchet_test.go).
package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"
)

const (
	// DefaultDesktopCheckpointInterval : intervalle du CHECKPOINT périodique des bases tenues en
	// RW, quand `desktop_checkpoint_interval` est absent d'app_settings.json.
	DefaultDesktopCheckpointInterval = 5 * time.Minute
	// MinDesktopCheckpointInterval : plancher (un CHECKPOINT par seconde ruinerait les écritures).
	MinDesktopCheckpointInterval = 30 * time.Second
	// MaxDesktopCheckpointInterval : plafond (au-delà, le réglage n'a plus de sens).
	MaxDesktopCheckpointInterval = time.Hour
)

// adoptCheckpointInterval pose l'intervalle du CHECKPOINT périodique sous la capacité
// FrequentCheckpoint ; hors capacité il reste nul et la tâche n'est pas lancée.
func adoptCheckpointInterval(cfg *AppConfig) {
	if !cfg.Capabilities.FrequentCheckpoint {
		return
	}
	cfg.CheckpointInterval = readCheckpointInterval(cfg.AppSettingsPath)
}

// readCheckpointInterval lit `desktop_checkpoint_interval` (durée Go : "5m", "90s") dans
// app_settings.json. Absent : DefaultDesktopCheckpointInterval. Illisible ou hors des bornes :
// la valeur par défaut ou la borne, avec un avertissement.
func readCheckpointInterval(settingsPath string) time.Duration {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return DefaultDesktopCheckpointInterval
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return DefaultDesktopCheckpointInterval
	}
	raw, ok := m["desktop_checkpoint_interval"].(string)
	if !ok || raw == "" {
		return DefaultDesktopCheckpointInterval
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		slog.Warn("config: desktop_checkpoint_interval illisible, valeur par défaut", "valeur", raw, "err", err,
			"defaut", DefaultDesktopCheckpointInterval)
		return DefaultDesktopCheckpointInterval
	}
	switch {
	case d < MinDesktopCheckpointInterval:
		slog.Warn("config: desktop_checkpoint_interval sous le plancher, plancher appliqué", "valeur", raw,
			"plancher", MinDesktopCheckpointInterval)
		return MinDesktopCheckpointInterval
	case d > MaxDesktopCheckpointInterval:
		slog.Warn("config: desktop_checkpoint_interval au-dessus du plafond, plafond appliqué", "valeur", raw,
			"plafond", MaxDesktopCheckpointInterval)
		return MaxDesktopCheckpointInterval
	}
	return d
}
