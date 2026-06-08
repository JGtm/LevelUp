package logging

import (
	"log/slog"
	"os"
)

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
	cfg := LoadConfig(repoRoot)

	var handler slog.Handler
	switch cfg.ConsoleFormat {
	case ConsoleFormatJSON:
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	case ConsoleFormatText:
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	default:
		handler = NewConsoleHandler(os.Stderr, ConsoleHandlerOptions{
			Level:    slog.LevelInfo,
			MaxWidth: cfg.MaxLineWidth,
			Color:    cfg.ConsoleColor,
		})
	}

	closer := func() {}
	if cfg.Enabled && cfg.LogsDir != "" {
		if mh, err := NewMultiModuleHandler(handler, cfg.LogsDir, cfg.FileLevel); err == nil {
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
