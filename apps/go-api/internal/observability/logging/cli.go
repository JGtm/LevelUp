package logging

import (
	"log/slog"
	"os"
	"strings"
)

// ConsoleLevelFromEnv lit le niveau de CONSOLE dans LEVELUP_LOG_LEVEL (defaut INFO) — la meme
// convention que le serveur (`cmd/server/main.go`). C'est `debug` qui fait apparaitre les lignes
// de mesure (durees de balayage de la cuisson, cf. `analysis/replay/observe.go`).
//
// ELLE VIT ICI, ET PAS DANS CHAQUE CLI : la table etait deja recopiee dans `cmd/replay-build` et
// s'appretait a l'etre dans `cmd/replay-equiv` — a la troisieme copie la regle du depot impose
// de centraliser. Un CLI qui installe son journal appelle
// `InstallCLILevel(root, ConsoleLevelFromEnv())`.
func ConsoleLevelFromEnv() slog.Level {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LEVELUP_LOG_LEVEL"))) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// InstallCLI configure le logging pour un binaire CLI (cmd/*) : console compacte
// sur stderr + fichiers `logs/{module}.log` (comme le serveur). Permet aux jobs
// one-shot (snapshot leaderboard, backfills) de tracer dans le dossier logs/
// dédié, et pas seulement sur stderr.
//
// repoRoot est passé à LoadConfig (peut être vide → défauts sains). Retourne une
// fonction de fermeture à `defer` (flush + close des fichiers). No-op si le file
// logging est désactivé (LEVELUP_LOGS_ENABLED=false ou LogsDir vide).
//
// Installe le handler comme slog.Default() — les logs taggés
// `slog.With("module", logging.ModuleLeaderboard)` sont routés vers
// logs/leaderboard.log.
func InstallCLI(repoRoot string) func() {
	return InstallCLILevel(repoRoot, slog.LevelInfo)
}

// InstallCLILevel est comme InstallCLI mais fixe le niveau de la CONSOLE (stderr).
// Les fichiers logs/*.log gardent leur propre niveau (cfg.FileLevel) : monter le
// niveau console (ex. slog.LevelError pour un mode -quiet) coupe le bruit INFO/WARN
// à l'écran SANS perdre le détail dans les fichiers. Utile aux jobs bruyants (backfill :
// les retries 429/réseau sont des WARN qui polluent la progression).
func InstallCLILevel(repoRoot string, consoleLevel slog.Level) func() {
	cfg := LoadConfig(repoRoot)

	var handler slog.Handler
	switch cfg.ConsoleFormat {
	case ConsoleFormatJSON:
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: consoleLevel})
	case ConsoleFormatText:
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: consoleLevel})
	default:
		handler = NewConsoleHandler(os.Stderr, ConsoleHandlerOptions{
			Level:    consoleLevel,
			MaxWidth: cfg.MaxLineWidth,
			Color:    cfg.ConsoleColor,
		})
	}

	closer := func() {}
	if cfg.Enabled && cfg.LogsDir != "" {
		if mh, err := NewMultiModuleHandler(handler, cfg.LogsDir, cfg.FileLevel, cfg.Rotation); err == nil {
			handler = mh
			closer = func() { _ = mh.Close() }
		} else {
			slog.New(handler).Warn("logging: multi-module indisponible (console only)",
				"err", err, "logs_dir", cfg.LogsDir)
		}
	}
	slog.SetDefault(slog.New(handler))
	return closer
}
