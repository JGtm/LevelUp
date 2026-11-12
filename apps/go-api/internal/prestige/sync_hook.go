package prestige

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/observability"
)

// sync_hook.go — point d'extension Prestige post-sync.
//
// Référence : Annexe E du plan conceptuel + Phase 3 IMPL.
//
// Les chemins de sync appellent `RunPostSyncHook` (via PrestigeBundle.RunPostSync)
// après ingestion des matchs — cf. le détail du câblage sur RunPostSyncHook.
// La fonction est gardée par le réglage `prestige_enabled` d'app_settings.json :
// s'il est désactivé, aucune action — le module Prestige est totalement absent
// du flux de sync.

// RunPostSyncHook ré-évalue les défis actifs d'un joueur après une sync.
//
// Appelé par PrestigeBundle.RunPostSync une fois `match_participants` écrits.
// Câblage réel (VF-1, corrigé 2026-07-06) — le hook tourne sur TOUS les chemins
// de sync :
//   - V1 (auto-sync + HTTP delta + watcher) : SyncEngine.WithPrestigeHook, câblé
//     par scheduler.BuildEngine → fire à engine.run() (post-pipeline, lease tenu).
//   - HTTP sync initial : SyncHandler.newEngineFor câble le même hook engine.
//   - V2 (cycle orchestrator, ADR 0027) : CycleOrchestratorImpl invoque le hook
//     par joueur après RunPostSync (V2 ne passe pas par engine.run()).
//
// Le GATE Prestige est UNIQUE et vit chez l'appelant (PrestigeBundle.RunPostSync
// teste b.enabled = cfg.PrestigeEnabled avant d'appeler ce hook) — plus de
// re-lecture du flag ici (C7 : fin du double-gate env-only vs settings.json).
//
// Best-effort : log les erreurs mais ne casse pas le flux sync. Le sync
// engine ne doit pas dépendre du résultat de Prestige.
func RunPostSyncHook(ctx context.Context, svc Service, userID, titleSlug string) {
	// Heartbeat feature (A6/DC-5) : pose au PASSAGE REEL — le cas « hook cable
	// mais jamais invoque » devient visible dans le panneau Crons & features.
	observability.Heartbeat("prestige_hook")
	if svc == nil {
		slog.WarnContext(ctx, "prestige: sync hook called with nil service")
		return
	}
	outcomes, err := svc.EvaluateForUser(ctx, userID, titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "prestige: post-sync evaluation failed",
			"user_id", userID, "title_slug", titleSlug, "err", err)
		return
	}
	transitions := 0
	for _, o := range outcomes {
		if o.OldStatus != o.NewStatus {
			transitions++
		}
	}
	slog.InfoContext(ctx, "prestige: post-sync evaluation completed",
		"user_id", userID, "title_slug", titleSlug,
		"evaluated", len(outcomes), "transitions", transitions)
}
