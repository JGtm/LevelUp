package games

import (
	"errors"
	"fmt"

	"levelup/go-api/internal/games/mappings"
)

// AllCapabilityKeys retourne les CapabilityKey canoniques connues du package
// games. Source de vérité du vocabulaire (le package mappings, title-agnostic,
// ne valide pas les clés — il délègue ici).
func AllCapabilityKeys() []CapabilityKey {
	return []CapabilityKey{
		CapMatchHistory,
		CapMatchDetailCore,
		CapMatchSkillSnapshot,
		CapCareerProgression,
		CapCareerRankCatalog,
		CapPveFirefight,
		CapTimeseries,
		CapAnalyticsCareerXPEstimate,
		CapScoreboardExtra,
		CapCitationsEngine,
		CapEngagement,
		CapBattlePass,
		CapChallenges,
		CapMatchEventsTimeline,
		CapMatchKillfeedPerKill,
		CapMatchEventsSpatial,
		CapCommendationsNative,
		CapWeaponAccuracy,
		CapPlaylistCategoryStrip,
		CapMatchObjectiveStats,
		CapFilmKillSource,
		CapFilmWeaponShots,
	}
}

// IsKnownCapabilityKey indique si une clé fait partie du vocabulaire canonique.
func IsKnownCapabilityKey(k CapabilityKey) bool {
	for _, known := range AllCapabilityKeys() {
		if known == k {
			return true
		}
	}
	return false
}

// CapabilityMapFromMappings convertit un mappings.CapabilityMappingSet (chargé
// depuis capabilities.toml) en games.CapabilityMap (Phase 1.7a). Valide que
// chaque clé est une CapabilityKey connue ET que chaque statut est admis ;
// retourne une erreur agrégée sinon (un TOML title-owned ne doit pas déclarer
// une capability hors vocabulaire produit).
func CapabilityMapFromMappings(set *mappings.CapabilityMappingSet) (CapabilityMap, error) {
	if set == nil {
		return nil, errors.New("games: CapabilityMappingSet nil")
	}
	raw := set.All()
	out := make(CapabilityMap, len(raw))
	var errs []error
	for key, status := range raw {
		ck := CapabilityKey(key)
		if !IsKnownCapabilityKey(ck) {
			errs = append(errs, fmt.Errorf("clé de capability inconnue %q", key))
			continue
		}
		switch CapabilityStatus(status) {
		case CapSupported, CapDegraded, CapNotExposed:
			out[ck] = CapabilityStatus(status)
		default:
			errs = append(errs, fmt.Errorf("statut invalide %q pour %q", status, key))
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return out, nil
}
