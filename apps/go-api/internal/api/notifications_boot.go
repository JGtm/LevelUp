// Package api — notifications_boot.go : émission app_release au démarrage.
//
// Au boot du serveur, on compare cfg.AppVersion avec sync_meta.last_seen_app_version
// pour chaque joueur connu et émet une notification app_release sur changement.
// La clé sync_meta est mise à jour seulement si l'émission réussit.
package api

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
)

const lastSeenAppVersionKey = "last_seen_app_version"

// EmitAppReleaseForAllPlayers détecte les joueurs dont la version vue diffère de
// currentVersion et émet une notification app_release per-player. Best-effort :
// les erreurs sont loguées mais ne bloquent pas le boot.
//
// Appelé une seule fois au démarrage du serveur après initialisation du
// ServiceRegistry.
func EmitAppReleaseForAllPlayers(
	ctx context.Context,
	cfg *config.AppConfig,
	reg *ServiceRegistry,
	currentVersion string,
) {
	if currentVersion == "" || currentVersion == "dev" {
		// Pas de notification en mode dev (évite le spam au reload).
		return
	}
	players, err := cfg.LoadPlayers()
	if err != nil {
		slog.WarnContext(ctx, "app_release: load players", "err", err)
		return
	}
	for _, p := range players {
		if p.PlayerSlug == "" {
			continue
		}
		if err := emitAppReleaseForPlayer(ctx, reg, p.PlayerSlug, currentVersion); err != nil {
			slog.WarnContext(ctx, "app_release: emit",
				"slug", p.PlayerSlug, "err", err)
		}
	}
}

func emitAppReleaseForPlayer(
	ctx context.Context,
	reg *ServiceRegistry,
	slug, currentVersion string,
) error {
	pdb, err := reg.resolve(ctx, slug)
	if err != nil {
		return fmt.Errorf("resolve: %w", err)
	}
	last, err := readLastSeenAppVersion(ctx, pdb)
	if err != nil {
		return fmt.Errorf("read sync_meta: %w", err)
	}
	if last == currentVersion {
		return nil // déjà notifié
	}
	emitter, err := reg.NotificationsEmitter(ctx, slug)
	if err != nil {
		return fmt.Errorf("emitter factory: %w", err)
	}
	if last == "" {
		// Premier démarrage observé : initialiser sans notifier (évite le bruit
		// d'une notif "v1.0.0 disponible" au tout premier boot du joueur).
		return writeLastSeenAppVersion(ctx, pdb, currentVersion)
	}
	if err := emitter.Emit(ctx, notifications.EmitInput{
		Category:    notifications.CategoryAppRelease,
		Severity:    notifications.SeverityInfo,
		TitleKey:    "notif.app_release.title",
		BodyKey:     "notif.app_release.body",
		Params:      map[string]any{"version": currentVersion, "previous": last},
		TargetRoute: "/help/changelog",
		Source:      "app_boot",
	}); err != nil {
		return fmt.Errorf("emit: %w", err)
	}
	return writeLastSeenAppVersion(ctx, pdb, currentVersion)
}

// readLastSeenAppVersion retourne la valeur sync_meta.last_seen_app_version, ou ""
// si la clé n'existe pas. La table sync_meta est garantie présente après migration
// "create_base_player_schema".
func readLastSeenAppVersion(ctx context.Context, pdb *duckdb.PlayerDB) (string, error) {
	return duckdb.ReadSyncMeta(ctx, pdb, lastSeenAppVersionKey)
}

// writeLastSeenAppVersion upsert la clé sync_meta.last_seen_app_version.
func writeLastSeenAppVersion(ctx context.Context, pdb *duckdb.PlayerDB, version string) error {
	return duckdb.WriteSyncMeta(ctx, pdb, lastSeenAppVersionKey, version)
}
