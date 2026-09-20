package main

// security_warnings.go — niveau du journal de démarrage pour les réglages non
// sûrs (secret de session par défaut, AuthMode=none, CORS localhost, cookie sans
// Secure).
//
// POURQUOI UN NIVEAU VARIABLE (2026-09-20). Le WARN était émis à CHAQUE boot, y
// compris sur un poste de dev qui écoute sur 127.0.0.1 sans LEVELUP_ENV : la
// configuration y est délibérément permissive et rien n'est joignable de
// l'extérieur, donc l'avertissement n'avertissait de rien et noyait le journal de
// démarrage. Il reste un WARN dès que l'instance est réellement exposée (hôte
// d'écoute hors boucle locale, ou environnement déclaré) — c'est le seul cas où
// ces réglages ont une portée.
//
// Le garde-fou fail-fast (cfg.Validate, refus de démarrer en production) est
// INCHANGÉ : il ne dépend que de LEVELUP_ENV=production, jamais de ce niveau.

import (
	"log/slog"
	"strings"

	"levelup/go-api/internal/config"
)

// logSecurityWarnings journalise les réglages non sûrs au niveau que leur portée
// mérite. Mêmes champs dans les deux cas — seuls le niveau et le message changent.
func logSecurityWarnings(cfg *config.AppConfig) {
	warnings := cfg.SecurityWarnings()
	if len(warnings) == 0 {
		return
	}
	attrs := []any{
		"issues_count", len(warnings),
		"issues", strings.Join(warnings, " | "),
		"prod_guard", "LEVELUP_ENV=production refuserait de démarrer dans cet état",
		"listen_host", cfg.APIHost,
		"env", cfg.Environment,
	}
	if cfg.IsExposedDeployment() {
		slog.Warn("configuration non sûre pour un déploiement multi-user exposé", attrs...)
		return
	}
	slog.Info("réglages de sécurité permissifs — sans portée ici (écoute en boucle locale, aucun environnement déclaré)", attrs...)
}
