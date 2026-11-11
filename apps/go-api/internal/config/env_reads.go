package config

// env_reads.go — lectures d'environnement qui ne vivent pas dans AppConfig parce que leurs
// appelants tournent avant, ou hors, du chargement de la configuration (journal du serveur, outils
// en ligne de commande, moteur de sync, notification de version). Chaque variable se lit ICI, une
// seule fois ; le garde-rail `env_centralization_test.go` interdit de la relire ailleurs.

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Noms des variables lues par ce fichier.
const (
	envLogLevel            = "LEVELUP_LOG_LEVEL"
	envOAuthClientID       = "LEVELUP_OAUTH_CLIENT_ID"
	envLegacyFilmCacheDir  = "LEVELUP_LEGACY_FILM_CACHE_DIR"
	envEventsConvChunk     = "LEVELUP_EVENTS_CONVERGENCE_CHUNK"
	envEventsConvYieldMS   = "LEVELUP_EVENTS_CONVERGENCE_YIELD_MS"
	envNotifyVersions      = "LEVELUP_NOTIFY_VERSIONS"
	notifyVersionsOptInVal = "1"
)

// ConsoleLogLevel rend le niveau de la console demandé par LEVELUP_LOG_LEVEL
// (debug | info | warn | error ; absent ou inconnu : info). Le serveur et les outils en ligne de
// commande qui installent leur journal (`logging.InstallCLILevel`) lisent le niveau ici : la
// même table pour tous. À appeler après `BootstrapEnvLocal` pour que `.env.local` compte.
func ConsoleLogLevel() slog.Level {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envLogLevel))) {
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

// OAuthClientIDFromEnv rend l'identifiant du client Azure demandé par LEVELUP_OAUTH_CLIENT_ID, ou
// "" quand la variable est absente (l'appelant retombe alors sur l'application canonique). Le
// serveur (refresh OAuth) et l'outil de capture de jetons lisent la MÊME valeur : un jeton de
// rafraîchissement est lié au client qui l'a émis.
func OAuthClientIDFromEnv() string {
	return os.Getenv(envOAuthClientID)
}

// LegacyFilmCacheDirFromEnv rend la racine du cache de films hérité demandée par
// LEVELUP_LEGACY_FILM_CACHE_DIR, sans espaces de bord ; "" quand elle est absente.
func LegacyFilmCacheDirFromEnv() string {
	return strings.TrimSpace(os.Getenv(envLegacyFilmCacheDir))
}

// EventsConvergenceTuning rend les réglages fins de la passe de convergence des événements :
// taille de lot (LEVELUP_EVENTS_CONVERGENCE_CHUNK) et pause entre deux lots
// (LEVELUP_EVENTS_CONVERGENCE_YIELD_MS, en millisecondes). Une valeur absente ou non strictement
// positive rend zéro : l'orchestrateur applique alors ses défauts.
func EventsConvergenceTuning() (chunk int, yield time.Duration) {
	return positiveIntFromEnv(envEventsConvChunk),
		time.Duration(positiveIntFromEnv(envEventsConvYieldMS)) * time.Millisecond
}

// NotifyVersionsEnabled dit si l'opt-in des notifications de version est posé
// (LEVELUP_NOTIFY_VERSIONS=1 en production ; absent en développement et en démo).
func NotifyVersionsEnabled() bool {
	return os.Getenv(envNotifyVersions) == notifyVersionsOptInVal
}

// positiveIntFromEnv lit un entier strictement positif ; 0 si la variable est absente ou invalide.
func positiveIntFromEnv(key string) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 0
}
