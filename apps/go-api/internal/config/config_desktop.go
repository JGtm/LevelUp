// config_desktop.go — réglages du serveur lancé par un processus parent (coquille desktop,
// pré-commande d'air en développement). Dans un fichier dédié : config.go est gelé en taille
// (config_size_ratchet_test.go).
package config

import "strings"

// DesktopConfig est embarquée dans AppConfig : ses champs s'y lisent directement.
type DesktopConfig struct {
	// DesktopSecret : secret partagé avec le processus qui lance le serveur. Lit
	// LEVELUP_DESKTOP_SECRET. VIDE PAR DÉFAUT : la route POST /api/v1/internal/shutdown n'est
	// alors pas montée (un serveur hébergé n'a pas cette route). Jamais journalisé ni exposé
	// (ni /bootstrap, ni les journaux de configuration).
	DesktopSecret string
	// ShutdownRequester demande l'arrêt propre du serveur (reason : cause, journalisée).
	// Posé par main (bootServer.requestShutdown) après la construction de la config, comme
	// TitleReadyNotifier, avant celle du routeur ; nil en CLI/tests, où la route d'arrêt
	// répond 404.
	ShutdownRequester func(reason string)
}

// withDesktopConfig lit les réglages desktop de l'environnement dans cfg et le rend.
func withDesktopConfig(cfg *AppConfig) *AppConfig {
	cfg.DesktopSecret = strings.TrimSpace(getEnvOrDefault("LEVELUP_DESKTOP_SECRET", ""))
	return cfg
}
