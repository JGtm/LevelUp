// Package api — notifications_boot.go : émission app_release au démarrage.
//
// Au boot du serveur, on compare cfg.AppVersion avec sync_meta.last_seen_app_version
// pour chaque joueur connu et émet une notification app_release sur changement.
// La clé sync_meta est mise à jour seulement si l'émission réussit. Le même changement
// de version déclenche aussi, une seule fois, la notification Discord « nouvelle
// version » (webhook unique, auto-gardée : cf. NotifyNewVersion).
package wire

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/notify"
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

	// Notif Discord « nouvelle version » — UNE seule fois par boot (webhook unique,
	// pas per-player, contrairement à app_release in-app ci-dessous). Entièrement
	// auto-gardée en interne : toggle discord_notify_new_version, env
	// LEVELUP_NOTIFY_VERSIONS=1 (posé en prod, absent en dev/demo), présence webhook,
	// et anti-spam last_notified_version (patch seul / version déjà notifiée). Best-effort
	// (jamais de panic ni d'erreur remontée), n'entrave pas l'émission in-app qui suit.
	notify.NotifyNewVersion(notify.LoadNotifyConfig(cfg.AppSettingsPath), currentVersion)

	players, err := cfg.LoadPlayers()
	if err != nil {
		slog.WarnContext(ctx, "app_release: load players", "err", err)
		return
	}
	targets, skipped := appReleaseTargets(players)
	if len(skipped) > 0 {
		// Trace de niveau DEBUG (et non WARN) : ne PAS avoir de player DB est l'état
		// NORMAL et attendu d'un profil auth_only, pas une anomalie.
		slog.DebugContext(ctx, "app_release: profils auth_only ignorés (aucune player DB)",
			"slugs", skipped, "count", len(skipped))
	}
	for _, slug := range targets {
		if err := emitAppReleaseForPlayer(ctx, reg, slug, currentVersion); err != nil {
			slog.WarnContext(ctx, "app_release: emit", "slug", slug, "err", err)
		}
	}
}

// appReleaseTargets sépare les profils db_profiles.json en (cibles, ignorés) pour
// l'émission app_release. Fonction PURE (testable sans registre ni DuckDB).
//
// Deux règles :
//  1. Un profil `auth_only` n'a AUCUNE player DB (`db_path` vide dans
//     db_profiles.json) : il n'existe que pour le pool de tokens. Tenter d'y émettre
//     échouait à la résolution → un WARN par compte à CHAQUE redémarrage
//     post-release (5 comptes en production). Il est écarté en amont.
//  2. Déduplication par slug : la notification est per-JOUEUR (« la version X est
//     disponible »), pas per-titre, alors que LoadPlayers() sans filtre retourne une
//     entrée par (titre, joueur) — un même joueur déclaré sur 2 titres était traité
//     deux fois.
//
// Les deux listes sont triées : ordre d'émission et de log stables.
func appReleaseTargets(players []domain.PlayerSummary) (targets, skipped []string) {
	seen := make(map[string]bool, len(players))
	skippedSeen := make(map[string]bool)
	for _, p := range players {
		if p.PlayerSlug == "" {
			continue
		}
		if p.AuthOnly {
			if !skippedSeen[p.PlayerSlug] {
				skippedSeen[p.PlayerSlug] = true
				skipped = append(skipped, p.PlayerSlug)
			}
			continue
		}
		if seen[p.PlayerSlug] {
			continue
		}
		seen[p.PlayerSlug] = true
		targets = append(targets, p.PlayerSlug)
	}
	sort.Strings(targets)
	sort.Strings(skipped)
	return targets, skipped
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
