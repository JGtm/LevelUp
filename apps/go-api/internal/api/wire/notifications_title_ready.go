// Package api — notifications_title_ready.go : notification « titre prêt » (MT-19,
// axe E).
//
// Émise UNE fois lorsqu'un titre live (Halo 5+, servi par un Runner dédié) a des
// matchs synchronisés, dans le flux de notifications du titre PAR DÉFAUT (Halo
// Infinite) — là où l'utilisateur, invité à « retourner sur Halo Infinite » le
// temps du backfill, la verra (les notifications sont scopées par titre :
// shared_social est per-title, cf. PathResolver.SharedSocialDBPath).
//
// N'emprunte JAMAIS la pipeline progression/prestige (hardcodée Infinite,
// post_sync_deltas.go) : c'est un simple Emit + watermark sync_meta, calqué sur
// app_release (notifications_boot.go). Best-effort, title-agnostic (le nom du
// titre est résolu via le registre, jamais de slug brut affiché).
package wire

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
)

// titleReadyWatermarkKey construit la clé sync_meta d'idempotence par titre annoncé
// (per-joueur, stockée dans la player DB du titre par défaut).
func titleReadyWatermarkKey(titleSlug string) string {
	return "title_ready_announced_" + titleSlug
}

// BuildTitleReadyNotifier retourne la closure injectée dans config.AppConfig
// (TitleReadyNotifier), appelée par le Runner live d'un titre à la fin d'un cycle
// de sync qui a produit des matchs. Best-effort (jamais d'erreur propagée au sync),
// idempotente (watermark sync_meta), title-agnostic.
func BuildTitleReadyNotifier(reg *ServiceRegistry, cfg *config.AppConfig) func(ctx context.Context, titleSlug, gamertag, xuid string, inserted int) {
	return func(ctx context.Context, titleSlug, gamertag, xuid string, inserted int) {
		if err := emitTitleReadyForPlayer(ctx, reg, cfg, titleSlug, gamertag, xuid, inserted); err != nil {
			slog.WarnContext(ctx, "title_ready: emit",
				"titleSlug", titleSlug, "gamertag", gamertag, "err", err)
		}
	}
}

// emitTitleReadyForPlayer résout le joueur dans le titre PAR DÉFAUT, vérifie le
// watermark (déjà annoncé → no-op), émet la notif, puis n'avance le watermark
// QUE sur succès (un Emit best-effort drop par lease saturé sera ré-essayé au
// prochain cycle — pas de perte définitive).
func emitTitleReadyForPlayer(
	ctx context.Context,
	reg *ServiceRegistry,
	cfg *config.AppConfig,
	titleSlug, gamertag, xuid string,
	inserted int,
) error {
	// La notif vit dans le flux du titre PAR DÉFAUT (Infinite) : on force le slug
	// défaut dans le ctx de résolution, indépendamment du ctx du sync du titre live.
	resCtx := ctxkeys.WithTitleSlug(ctx, titlePkg.DefaultSlug)

	playerSlug := playerSlugForXUID(cfg, xuid)
	if playerSlug == "" {
		// Joueur non déclaré dans le titre par défaut (db_profiles) → pas de flux où
		// poser la notif. Dégradation silencieuse (titre-only sans profil Infinite).
		return nil
	}

	pdb, err := reg.resolve(resCtx, playerSlug)
	if err != nil {
		return fmt.Errorf("resolve: %w", err)
	}

	key := titleReadyWatermarkKey(titleSlug)
	already, err := duckdb.ReadSyncMeta(ctx, pdb, key)
	if err != nil {
		return fmt.Errorf("read sync_meta: %w", err)
	}
	if already == "1" {
		return nil // déjà annoncé
	}

	name := titleSlug
	if d := titlePkg.DefaultRegistry().Get(titleSlug); d != nil && d.Name != "" {
		name = d.Name
	}

	emitter, err := reg.NotificationsEmitter(resCtx, playerSlug)
	if err != nil {
		return fmt.Errorf("emitter factory: %w", err)
	}
	if err := emitter.Emit(resCtx, notifications.EmitInput{
		Category:    notifications.CategoryTitleReady,
		Severity:    notifications.SeveritySuccess,
		TitleKey:    "notif.title_ready.title",
		BodyKey:     "notif.title_ready.body",
		Params:      map[string]any{"title_slug": titleSlug, "title_name": name, "count": inserted},
		TargetRoute: "/players/" + playerSlug + "/home",
		Source:      "livesync",
	}); err != nil {
		return fmt.Errorf("emit: %w", err)
	}
	return duckdb.WriteSyncMeta(ctx, pdb, key, "1")
}

// playerSlugForXUID résout le player_slug (db_profiles) d'un xuid Xbox dans le
// titre par défaut. "" si le joueur n'y est pas déclaré.
func playerSlugForXUID(cfg *config.AppConfig, xuid string) string {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return ""
	}
	for i := range players {
		if players[i].XUID == xuid {
			return players[i].PlayerSlug
		}
	}
	return ""
}

// Lecture/écriture sync_meta : centralisées dans duckdb.{Read,Write}SyncMeta
// (dédup #6, K1c) — cf. platform/duckdb/sync_meta_repo.go.
