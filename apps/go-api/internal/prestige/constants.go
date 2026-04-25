package prestige

// constants.go — valeurs de référence du système Prestige.
//
// Ces constantes sont *non-tunables* (référence visuelle/sémantique).
// Les seuils et plafonds *tunables* vivent dans config/prestige/tuning.toml
// pour être modifiables sans redéploiement.

// PalierColor retourne le code hex de la bordure/badge pour un palier.
//
// Référence Annexe B du plan conceptuel — couleurs de rareté Halo Infinite :
// Common gris, Rare bleu, Epic violet, Legendary or.
func PalierColor(t Tier) string {
	switch t {
	case TierNormal:
		return "#9CA3AF"
	case TierHeroic:
		return "#3B82F6"
	case TierLegendary:
		return "#8B5CF6"
	case TierMythic:
		return "#F59E0B"
	}
	return ""
}

// Sources d'événements PP (champ source_type de prestige_events).
const (
	SourceMatch     = "match"
	SourceChallenge = "challenge"
	SourceArc       = "arc"
	SourceStreak    = "streak"
	SourceMedal     = "medal"
)

// Types d'événements de télémétrie (champ event_type de prestige_telemetry).
const (
	TelemetryCreated          = "created"
	TelemetryRejected         = "rejected"
	TelemetryCommitted        = "committed"
	TelemetryCompleted        = "completed"
	TelemetryExpired          = "expired"
	TelemetryAbandoned        = "abandoned"
	TelemetryPalierRecomputed = "palier_recomputed"
)
