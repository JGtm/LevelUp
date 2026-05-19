package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Config résume la configuration du système de logging multi-module.
// Chargée depuis les variables d'environnement au boot du serveur.
type Config struct {
	// LogsDir : répertoire où écrire les fichiers `{module}.log`.
	// Vide → file logging désactivé (console uniquement).
	// Par défaut : `logs/` en non-prod, vide en prod (LEVELUP_LOG_JSON=true).
	// Override via LEVELUP_LOGS_DIR.
	LogsDir string

	// FileLevel : niveau minimal des logs écrits dans les fichiers. Permet
	// d'avoir des fichiers en DEBUG (debug post-mortem complet) pendant que
	// la console reste en INFO (volume gérable). Par défaut INFO.
	// Override via LEVELUP_LOGS_FILE_LEVEL=debug|info|warn|error.
	FileLevel slog.Level

	// Enabled : kill-switch global. Si false, MultiModuleHandler n'est pas
	// instancié et le handler console pré-existant est utilisé tel quel.
	// Par défaut true. Override via LEVELUP_LOGS_ENABLED=false.
	Enabled bool
}

// LoadConfig lit la config depuis l'env. Tous les fallbacks sont sains :
// jamais d'erreur, valeurs par défaut conservatives (file logging à l'emplacement
// `logs/` du repoRoot, niveau INFO).
//
// repoRoot : utilisé pour résoudre le LogsDir par défaut (logs/ sous repoRoot).
// Peut être vide → LogsDir = "logs" (relatif au cwd).
func LoadConfig(repoRoot string) Config {
	cfg := Config{
		Enabled:   parseBoolEnv("LEVELUP_LOGS_ENABLED", true),
		FileLevel: parseLevelEnv("LEVELUP_LOGS_FILE_LEVEL", slog.LevelInfo),
	}
	cfg.LogsDir = os.Getenv("LEVELUP_LOGS_DIR")
	if cfg.LogsDir == "" && cfg.Enabled {
		if repoRoot != "" {
			cfg.LogsDir = repoRoot + string(os.PathSeparator) + "logs"
		} else {
			cfg.LogsDir = "logs"
		}
	}
	// Si désactivé, vider le path pour signal clair "no file logging".
	if !cfg.Enabled {
		cfg.LogsDir = ""
	}
	return cfg
}

// parseBoolEnv retourne la valeur env si "true"/"false", sinon le défaut.
func parseBoolEnv(key string, def bool) bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch raw {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	}
	return def
}

// parseLevelEnv retourne le slog.Level depuis env si reconnu, sinon le défaut.
func parseLevelEnv(key string, def slog.Level) slog.Level {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	}
	return def
}
