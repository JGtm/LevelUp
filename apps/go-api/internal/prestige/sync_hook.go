package prestige

import (
	"context"
	"encoding/json"
	"levelup/go-api/internal/observability"
	"log/slog"
	"os"
	"strings"
)

// sync_hook.go — point d'extension Prestige post-sync.
//
// Référence : Annexe E du plan conceptuel + Phase 3 IMPL.
//
// Les chemins de sync appellent `RunPostSyncHook` (via PrestigeBundle.RunPostSync)
// après ingestion des matchs — cf. le détail du câblage sur RunPostSyncHook.
// La fonction est gardée par le feature flag `PRESTIGE_ENABLED` :
// si le flag est désactivé, aucune action — le module Prestige est
// totalement absent du flux de sync.

// FeatureFlagEnv est le nom de la variable d'environnement d'override d'urgence
// de la source canonique (app_settings.json).
const FeatureFlagEnv = "PRESTIGE_ENABLED"

// IsEnabled est la SOURCE UNIQUE de l'activation Prestige (C7 / DEC-4, ADR 0005).
// Lit `prestige_enabled` dans app_settings.json (settingsPath), avec la var d'env
// PRESTIGE_ENABLED en override d'urgence (falsy "0"/"false"/"no"/"off" → OFF).
// Défaut : ACTIVÉ (fichier/clé absents ou JSON malformé → true). Consommée par
// TOUTES les surfaces via cfg.PrestigeEnabled (config.go) → un seul gate.
func IsEnabled(settingsPath string) bool {
	if v := os.Getenv(FeatureFlagEnv); v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "0", "false", "no", "off":
			return false
		}
		return true
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return true
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return true
	}
	if b, ok := m["prestige_enabled"].(bool); ok {
		return b
	}
	return true
}

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
