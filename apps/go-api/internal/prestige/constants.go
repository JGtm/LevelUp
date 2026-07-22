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

// Origines d'un défi (champ `source` de challenge / prestige_telemetry — ADR 0020).
// Renseignées par CreateChallengeRequest.Source et propagées à la télémétrie pour
// mesurer l'efficacité du coach proactif (taux d'acceptation/complétion par origine).
// ChallengeSourceUnknown n'est jamais écrit : c'est le bucket d'agrégation des lignes
// historiques (source NULL, créées avant le plumbing) côté endpoint diag.
const (
	ChallengeSourceUser      = "user"
	ChallengeSourcePilotMode = "pilot_mode"
	ChallengeSourceCoach     = "coach"
	ChallengeSourceUnknown   = "unknown"
)

// IsValidChallengeSource indique si s est une origine de défi reconnue
// (hors "unknown", qui est un bucket d'agrégation, jamais une valeur écrite).
// Sert à valider une origine fournie par un client HTTP avant de la persister.
func IsValidChallengeSource(s string) bool {
	switch s {
	case ChallengeSourceUser, ChallengeSourcePilotMode, ChallengeSourceCoach:
		return true
	default:
		return false
	}
}

// Types d'événements de télémétrie (champ event_type de prestige_telemetry).
const (
	TelemetryCreated   = "created"
	TelemetryRejected  = "rejected"
	TelemetryCommitted = "committed"
	TelemetryCompleted = "completed"
	TelemetryExpired   = "expired"
	TelemetryAbandoned = "abandoned"
	// TelemetryArchived : retrait SYSTÈME d'un défi (ex. désactivation du mode
	// pilote) — distinct de `abandoned` (abandon volontaire du joueur) pour
	// mesurer le churn du mode pilote sans le confondre avec les abandons.
	TelemetryArchived         = "archived"
	TelemetryPalierRecomputed = "palier_recomputed"
)

// Noms de métriques canoniques utilisés par les défis Prestige.
//
// Coexistent deux formats : noms PascalCase (legacy, dérivés des champs
// canonical.PlayerMatchRow.Field*) et snake_case (API publique / TOML tuning).
// Tout évaluateur doit accepter les deux orthographes pour rester
// rétrocompatible. Centralisés ici pour éviter la duplication littérale.
const (
	MetricWinRatePascal = "FieldWinRate"
	MetricWinRateSnake  = "win_rate"
	MetricKDAPascal     = "FieldKDA"
)
