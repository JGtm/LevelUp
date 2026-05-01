package prestige

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// sync_hook.go — point d'extension Prestige post-sync.
//
// Référence : Annexe E du plan conceptuel + Phase 3 IMPL.
//
// Le sync engine appelle `RunPostSyncHook` après ingestion des matchs.
// La fonction est gardée par le feature flag `PRESTIGE_ENABLED` :
// si le flag est désactivé, aucune action — le module Prestige est
// totalement absent du flux de sync.

// FeatureFlagEnv est le nom de la variable d'environnement contrôlant
// l'activation du module Prestige.
//
// Activé par défaut. Désactivé seulement si PRESTIGE_ENABLED vaut explicitement
// "0", "false", "no" ou "off" (insensible à la casse).
const FeatureFlagEnv = "PRESTIGE_ENABLED"

// IsEnabled retourne true si le feature flag Prestige est activé.
//
// Lit PRESTIGE_ENABLED à chaque appel — le boot ne cache pas l'état pour
// permettre un toggle sans redémarrage en dev.
//
// Défaut : activé. Pour désactiver, exporter PRESTIGE_ENABLED=false.
func IsEnabled() bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(FeatureFlagEnv)))
	switch raw {
	case "0", "false", "no", "off":
		return false
	}
	return true
}

// RunPostSyncHook ré-évalue les défis actifs d'un joueur après une sync.
//
// Appelé par le sync engine une fois `match_participants` écrits. Si le
// feature flag est désactivé, retourne immédiatement (aucun effet de bord).
//
// Best-effort : log les erreurs mais ne casse pas le flux sync. Le sync
// engine ne doit pas dépendre du résultat de Prestige.
func RunPostSyncHook(ctx context.Context, svc Service, userID, titleSlug string) {
	if !IsEnabled() {
		slog.DebugContext(ctx, "prestige: sync hook skipped (feature flag off)",
			"user_id", userID, "title_slug", titleSlug)
		return
	}
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
