// Package service — home_service_battlepass.go : concern Battle Pass + Challenges
// (live-d'abord / fallback cache DB). Extrait de home_service.go (refactor god-file).
package service

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
)

// battlePassCacheTTLFallback : fallback si live indisponible (accepte des données anciennes).
const battlePassCacheTTLFallback = 24 * time.Hour

func (s *HomeService) GetBattlePass(ctx context.Context) domain.BattlePassResponse {
	resp, raw := s.provider.GetBattlePassWithRaw(ctx)
	if resp.Available && resp.RewardTrack != nil {
		slog.DebugContext(ctx, "home: BattlePass obtenu depuis API live")
		if s.sink != nil {
			if err := s.sink.PersistBattlePassSync(ctx, *resp.RewardTrack, raw); err != nil {
				slog.WarnContext(ctx, "home: BattlePass persist failed", "err", err)
			}
		}
		if resp.SnapshotAt == nil {
			now := time.Now().UTC().Format(time.RFC3339)
			resp.SnapshotAt = &now
		}
		return resp
	}
	// Live indisponible (pas de tokens, erreur rÃ©seau) â†’ fallback cache DB.
	if s.cacheRepo != nil {
		if cached, hit, err := s.cacheRepo.LoadCachedBattlePass(ctx, battlePassCacheTTLFallback); err == nil && hit {
			slog.DebugContext(ctx, "home: BattlePass live indisponible - fallback cache DB",
				"snapshot_at", snapshotAtValue(cached.SnapshotAt),
				"age_hours", snapshotAgeHours(cached.SnapshotAt),
			)
			return *cached
		}
	}
	slog.DebugContext(ctx, "home: BattlePass live indisponible, aucun cache disponible")
	return resp
}

// snapshotAtValue retourne la valeur du pointeur ou "" si nil.
// Utilisé pour les logs structurés sans déréférencement risqué.
func snapshotAtValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// snapshotAgeHours retourne l'âge en heures (arrondi) d'un snapshot RFC3339.
// Retourne -1 si le pointeur est nil ou si le parsing échoue — signe que le
// snapshot_at est manquant côté DB plutôt que d'inventer une valeur.
func snapshotAgeHours(s *string) int {
	if s == nil {
		return -1
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return -1
	}
	return int(time.Since(t).Hours())
}

// RefreshTrack hydrate (synchroneement) une définition de reward track et toutes
// ses définitions d'items via le resolver assets (KindRewardTrackDefinition →
// KindBPItemDefinition). Persiste dans battlepass_track_definitions et
// battlepass_item_definitions. Best-effort : silencieux en cas d'erreur.
//
// Utilisé par SeasonPassService pour hydrater on-demand les passes non-actives
// dont les items n'ont jamais été résolus en DB.
func (s *HomeService) RefreshTrack(ctx context.Context, trackPath string) {
	if s.provider == nil {
		return
	}
	s.provider.FetchAndWarmTrack(ctx, trackPath)
}

// GetChallenges retourne les dÃ©fis actifs (live d'abord, cache DB en fallback).
// Appel live systÃ©matique pour garantir des donnÃ©es fraÃ®ches au rechargement de page.
// Si le live Ã©choue (tokens absents, API indisponible), le cache DB est retournÃ©.
func (s *HomeService) GetChallenges(ctx context.Context) domain.ChallengesResponse {
	resp, raw := s.provider.GetChallengesWithRaw(ctx)
	if resp.Available {
		slog.DebugContext(ctx, "home: Challenges obtenus depuis API live")
		if s.sink != nil {
			// W6 (lifecycle) : écriture SYNCHRONE sur le ctx HTTP — comme
			// GetBattlePass — pour qu'elle se termine avant que srv.Shutdown ne
			// rende la main (donc avant duckdb.CloseAll()). Plus de goroutine
			// détachée en context.Background() (handle orphelin / WAL au shutdown).
			if err := s.sink.PersistChallengesSync(ctx, raw); err != nil {
				slog.WarnContext(ctx, "home: Challenges persist failed", "err", err)
			}
		}
		if resp.SnapshotAt == nil {
			now := time.Now().UTC().Format(time.RFC3339)
			resp.SnapshotAt = &now
		}
		return resp
	}
	// Live indisponible â†’ fallback cache DB.
	if s.cacheRepo != nil {
		if cached, hit, err := s.cacheRepo.LoadCachedChallenges(ctx, battlePassCacheTTLFallback); err == nil && hit {
			if cacheChallengesAreRenderable(cached) {
				slog.DebugContext(ctx, "home: Challenges live indisponibles - fallback cache DB",
					"snapshot_at", snapshotAtValue(cached.SnapshotAt),
					"age_hours", snapshotAgeHours(cached.SnapshotAt),
				)
				return *cached
			}
		}
	}
	slog.DebugContext(ctx, "home: Challenges live indisponibles, aucun cache disponible")
	return resp
}

func cacheChallengesAreRenderable(resp *domain.ChallengesResponse) bool {
	if resp == nil {
		return false
	}
	if len(resp.Items) > 0 {
		return true
	}
	if resp.Total != nil && resp.Completed != nil && *resp.Total > *resp.Completed {
		return false
	}
	return true
}

// =============================================================================
// P4.3b (ADR 0011) : les converters canonical â†’ home types ont Ã©tÃ© dÃ©placÃ©s
// dans `analysis/home_canonical.go` (encapsulÃ©s derriÃ¨re les wrappers
// `analysis.*FromCanonical`). Le service ne porte plus de logique de
// conversion : il consomme les wrappers directement.
// =============================================================================
