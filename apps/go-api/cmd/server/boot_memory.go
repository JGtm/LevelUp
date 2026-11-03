// boot_memory.go — le plafond mémoire des bases DuckDB au démarrage du serveur desktop (lot L22.d, ADR 0045 D-27).
package main

import (
	"log/slog"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/settings"
)

// applyBootMemoryLimit pose le plafond mémoire des bases (`memory_limit_mb`) AVANT l'ouverture de la première
// base, sous la capacité MemoryLimit : chaque connexion reçoit alors le plafond dès son ouverture. Zéro :
// la valeur par défaut du serveur reste celle de chaque connexion.
func applyBootMemoryLimit(cfg *config.AppConfig) {
	if !cfg.Capabilities.MemoryLimit {
		return
	}
	s, err := settings.NewStore(cfg.AppSettingsPath).Load()
	if err != nil {
		slog.Warn("duckdb: plafond mémoire non lu au démarrage (valeur par défaut du serveur)", "err", err)
		return
	}
	duckdb.SetMemoryLimitMB(s.MemoryLimitMB)
	slog.Info("duckdb: plafond mémoire des bases", "reglage_mo", s.MemoryLimitMB, "effectif", duckdb.EffectiveMemoryLimit())
}
