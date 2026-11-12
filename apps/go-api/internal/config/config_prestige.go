package config

import (
	"encoding/json"
	"os"
)

// loadPrestigeEnabled lit `prestige_enabled` dans app_settings.json (settingsPath). Défaut : ACTIVÉ (fichier ou
// clé absents, JSON malformé -> true). Lu une fois par Load ; toutes les surfaces passent par
// cfg.PrestigeEnabled, un seul gate (C7/DEC-4).
func loadPrestigeEnabled(settingsPath string) bool {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return true
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return true
	}
	if b, ok := m["prestige_enabled"].(bool); ok {
		return b
	}
	return true
}
