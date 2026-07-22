// Package service - teammates_squad_charts_weapons_accuracy.go : builder de la
// comparaison « Précision par rôle » multi-joueurs de la page Escouade (précision
// NATIVE Halo 5, ~30 armes agrégées par rôle pour la lisibilité). MIROIR de
// buildSquadWeaponKills (teammates_squad_charts_weapons_perf.go) pour la métrique
// précision — réutilise le MÊME périmètre via resolveSquadScope (zéro duplication de
// scope). Best-effort : omise sur Halo Infinite (capability absente).
package teammates

import (
	"context"
	"errors"
	"log/slog"
	"sort"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// buildSquadWeaponAccuracy construit la comparaison « Précision par rôle » multi-joueurs,
// MIROIR de buildSquadWeaponKills. Réutilise le périmètre partagé (resolveSquadScope :
// matchs partagés + xuids/gamertags ordonnés) puis charge la précision agrégée de TOUS les
// xuids en 1 appel (table weapon_accuracy SHARED par titre → un repo lié à la player DB du
// main suffit, filtre MatchIDs + XUIDs). Best-effort (jamais fatal) : repo nil / capability
// absente (Infinite, ErrCapabilityNotSupported → Debug) / erreur (Warn) / aucune donnée →
// nil (le front n'affiche pas les blocs précision). Exclut les classes sans « tir au but »
// (grenade/mêlée/capacités, via domain.WeaponClassHasAccuracy) ; regroupe les armes à tir
// PAR RÔLE (precision/automatic/sniper/…). Précision par joueur = ΣShotsLanded/ΣShotsFired
// (0..1) du rôle, sommée sur toutes les armes du rôle (RAW).
func (s *TeammatesService) buildSquadWeaponAccuracy(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag, mainXUID string,
	teammates []domain.TeammateRow,
) *domain.SquadWeaponAccuracy {
	if s.weaponAccuracyRepo == nil || len(allSquadRows) == 0 || len(teammates) == 0 {
		return nil
	}
	sc := resolveSquadScope(allSquadRows, mainGamertag, mainXUID, teammates)
	if len(sc.sharedMatches) == 0 || len(sc.playersOrdered) == 0 {
		return nil
	}

	// 1 seul appel pour tous les xuids (table partagée, filtre MatchIDs + XUIDs).
	rows, err := s.weaponAccuracyRepo.LoadWeaponAccuracyAggregated(ctx, s.titleSlug,
		port.WeaponAccuracyFilters{MatchIDs: sc.sharedMatches, XUIDs: sc.xuids})
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			slog.DebugContext(ctx, "teammates_weapon_accuracy_capability_absente",
				"title", s.titleSlug, "matches", len(sc.sharedMatches))
		} else {
			slog.WarnContext(ctx, "teammates_weapon_accuracy_load_failed",
				"err", err, "matches", len(sc.sharedMatches), "xuids", len(sc.xuids))
		}
		return nil
	}
	if len(rows) == 0 {
		slog.DebugContext(ctx, "teammates_weapon_accuracy_empty_rows",
			"matches", len(sc.sharedMatches), "xuids", len(sc.xuids))
		return nil
	}

	bars := aggregateSquadWeaponAccuracy(ctx, rows, sc.gtByXUID)
	if len(bars) == 0 {
		return nil
	}
	slog.DebugContext(ctx, "teammates_weapon_accuracy_built",
		"title", s.titleSlug, "roles", len(bars), "players", len(sc.playersOrdered))
	return &domain.SquadWeaponAccuracy{
		Players: sc.playersOrdered,
		Bars:    bars,
	}
}

// squadWeaponAccuracyAgg agrège un RÔLE d'arme (précision + tirs par gamertag) avant
// projection vers le DTO. La table weapon_accuracy fournit 1 row par (xuid, weapon_id) ;
// on SOMME shots_landed / shots_fired par (rôle, gamertag) puis la précision par joueur =
// ΣShotsLanded/ΣShotsFired (RAW, jamais re-dérivée d'un ratio arrondi).
type squadWeaponAccuracyAgg struct {
	role       string
	landed     map[string]int // gamertag → Σ shots_landed du rôle
	fired      map[string]int // gamertag → Σ shots_fired du rôle
	totalShots int
}

// aggregateSquadWeaponAccuracy groupe les rows par RÔLE d'arme et projette vers les barres du
// DTO, triées ASC par TotalShotsSquad (parité tri de SquadWeaponKills : rôles peu tirés en
// haut), tie-break par clé de rôle. Écarte les rows de classe sans précision pertinente
// (domain.WeaponClassHasAccuracy) ; une arme à tir sans rôle résolu est comptée puis écartée
// (log Debug — pas de perte silencieuse). Précision par joueur = ΣLanded/ΣFired RAW.
func aggregateSquadWeaponAccuracy(
	ctx context.Context,
	rows []port.WeaponAccuracyRow,
	gtByXUID map[string]string,
) []domain.SquadWeaponAccuracyBar {
	agg := make(map[string]*squadWeaponAccuracyAgg)
	weaponsWithoutRole := make(map[int64]struct{})
	for _, r := range rows {
		if !domain.WeaponClassHasAccuracy(r.Class) {
			continue
		}
		if r.Role == "" {
			weaponsWithoutRole[r.WeaponID] = struct{}{}
			continue
		}
		if r.ShotsFired <= 0 {
			continue
		}
		gt, ok := gtByXUID[r.XUID]
		if !ok {
			continue
		}
		b, exists := agg[r.Role]
		if !exists {
			b = &squadWeaponAccuracyAgg{
				role:   r.Role,
				landed: make(map[string]int),
				fired:  make(map[string]int),
			}
			agg[r.Role] = b
		}
		b.landed[gt] += r.ShotsLanded
		b.fired[gt] += r.ShotsFired
		b.totalShots += r.ShotsFired
	}
	if len(weaponsWithoutRole) > 0 {
		slog.DebugContext(ctx, "teammates_weapon_accuracy_weapons_without_role",
			"count", len(weaponsWithoutRole))
	}
	if len(agg) == 0 {
		return nil
	}
	out := make([]domain.SquadWeaponAccuracyBar, 0, len(agg))
	for _, b := range agg {
		accuracy := make(map[string]float64, len(b.fired))
		for gt, fired := range b.fired {
			accuracy[gt] = float64(b.landed[gt]) / float64(fired)
		}
		out = append(out, domain.SquadWeaponAccuracyBar{
			Role:               b.role,
			AccuracyByPlayer:   accuracy,
			ShotsFiredByPlayer: b.fired,
			TotalShotsSquad:    b.totalShots,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TotalShotsSquad != out[j].TotalShotsSquad {
			return out[i].TotalShotsSquad < out[j].TotalShotsSquad
		}
		return out[i].Role < out[j].Role
	})
	return out
}
