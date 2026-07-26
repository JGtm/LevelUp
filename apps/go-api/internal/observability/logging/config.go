package logging

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Formats de sortie console reconnus (LEVELUP_LOG_FORMAT).
const (
	ConsoleFormatCompact = "compact"
	ConsoleFormatText    = "text"
	ConsoleFormatJSON    = "json"
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

	// Rotation : bornes de taille appliquées à CHAQUE fichier `{module}.log`
	// (cf. rotation.go). Défaut 100 Mo × 3 archives → ~400 Mo par catégorie au
	// pire. Override via LEVELUP_LOGS_MAX_SIZE_MB / LEVELUP_LOGS_MAX_BACKUPS
	// (0 Mo = rotation désactivée, croissance illimitée — l'état d'avant
	// 2026-07-26 qui avait produit 2,1 Go en prod).
	Rotation RotationPolicy

	// ConsoleFormat : format de sortie console. Valeurs reconnues :
	//   - "compact" (défaut) : `HH:MM:SS [LEVEL] msg key=val ...` (ConsoleHandler)
	//   - "text"             : slog.NewTextHandler (verbose, pre-sprint)
	//   - "json"             : slog.NewJSONHandler (prod)
	// Override via LEVELUP_LOG_FORMAT=compact|text|json. La var legacy
	// LEVELUP_LOG_JSON=true force "json" si LEVELUP_LOG_FORMAT n'est pas
	// défini (rétro-compat).
	ConsoleFormat string

	// MaxLineWidth : tronquage des lignes console au-delà de N caractères.
	// 0 = pas de tronquage. Défaut 200. Override via LEVELUP_LOG_MAX_LINE.
	// Appliqué uniquement au format "compact" — les formats "text"/"json"
	// gardent leur sortie native pour préserver le parsing machine.
	MaxLineWidth int

	// ConsoleColor : active les codes ANSI couleur sur la console (rouge
	// pour ERROR, jaune WARN, bleu INFO, gris DEBUG). Off par défaut pour
	// rester portable Windows cmd.exe (pas d'auto-détection TTY). Override
	// via LEVELUP_LOG_COLOR=on|off.
	ConsoleColor bool
}

// LoadConfig lit la config depuis l'env. Tous les fallbacks sont sains :
// jamais d'erreur, valeurs par défaut conservatives (file logging à l'emplacement
// `logs/` du repoRoot, niveau INFO).
//
// repoRoot : utilisé pour résoudre le LogsDir par défaut (logs/ sous repoRoot).
// Peut être vide → LogsDir = "logs" (relatif au cwd).
func LoadConfig(repoRoot string) Config {
	cfg := Config{
		Enabled:       parseBoolEnv("LEVELUP_LOGS_ENABLED", true),
		FileLevel:     parseLevelEnv("LEVELUP_LOGS_FILE_LEVEL", slog.LevelInfo),
		ConsoleFormat: parseConsoleFormat(),
		MaxLineWidth:  parseIntEnv("LEVELUP_LOG_MAX_LINE", 200),
		ConsoleColor:  parseBoolEnv("LEVELUP_LOG_COLOR", false),
		Rotation:      loadRotationPolicy(),
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

// WithTitleNamespace retourne une COPIE de la Config dont le LogsDir est
// namespacé par titre (`<LogsDir>/<titleSlug>/{module}.log`) — MT-05 (PMT-10
// PR-4). À n'appliquer QUE quand plusieurs titres sont réellement servis
// (len(registry) > 1) : en mono-titre le LogsDir reste nu → byte-identique.
// No-op si LogsDir ou titleSlug est vide. Le receiver valeur garantit que la
// Config d'origine n'est pas mutée.
func (cfg Config) WithTitleNamespace(titleSlug string) Config {
	if cfg.LogsDir == "" || titleSlug == "" {
		return cfg
	}
	cfg.LogsDir = cfg.LogsDir + string(os.PathSeparator) + titleSlug
	return cfg
}

// parseConsoleFormat résout le format console selon (priorité) :
//  1. LEVELUP_LOG_FORMAT explicite (compact|text|json)
//  2. LEVELUP_LOG_JSON=true → json (rétro-compat)
//  3. Défaut → compact
func parseConsoleFormat() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LEVELUP_LOG_FORMAT"))) {
	case ConsoleFormatCompact:
		return ConsoleFormatCompact
	case ConsoleFormatText:
		return ConsoleFormatText
	case ConsoleFormatJSON:
		return ConsoleFormatJSON
	}
	if parseBoolEnv("LEVELUP_LOG_JSON", false) {
		return ConsoleFormatJSON
	}
	return ConsoleFormatCompact
}

// loadRotationPolicy lit les bornes de rotation depuis l'env. Défauts :
// 100 Mo par fichier, 3 archives. Une valeur négative est ramenée à 0 par
// parseIntEnv → 0 Mo désactive la rotation, 0 archive supprime l'historique.
func loadRotationPolicy() RotationPolicy {
	sizeMB := parseIntEnv("LEVELUP_LOGS_MAX_SIZE_MB", DefaultRotationMaxSizeMB)
	backups := parseIntEnv("LEVELUP_LOGS_MAX_BACKUPS", DefaultRotationMaxBackups)
	return RotationPolicy{
		MaxSizeBytes: int64(sizeMB) * bytesPerMB,
		MaxBackups:   backups,
	}
}

// parseIntEnv retourne l'entier depuis env, sinon le défaut. Valeurs négatives
// remplacées par 0 (interprétées comme "désactivé" : pas de tronquage console,
// pas de rotation).
func parseIntEnv(key string, def int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	if n < 0 {
		return 0
	}
	return n
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
