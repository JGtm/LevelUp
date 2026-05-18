// Package coach — types pour le générateur d'alertes proactif (couche 3).
//
// Le coach observe en continu et agit sur signaux POSITIFS uniquement :
// améliorations soutenues, approches de paliers/records, streaks notables,
// reprises après pause. Il ne pointe jamais une régression.
//
// Décision §2bis du PLAN_PROGRESSION_TRACKING_ASCENSION.md : pas de table
// `coach_alert` dédiée. Les alertes sont émises via le package
// `internal/notifications` et stockées dans `player_notifications`
// (shared_social.duckdb). Ce package se limite donc aux types métier
// (AlertType, déduplication keys, params payload).
package coach

import "time"

// AlertType identifie la nature d'un signal détecté par le coach.
//
// Mappé sur notifications.Category au moment de l'émission via
// NotificationCategory() — défini dans emitter.go pour éviter l'import
// cyclique potentiel sur internal/notifications.
type AlertType string

const (
	// AlertTypeRecordBroken : nouveau PB sur une métrique × fenêtre.
	AlertTypeRecordBroken AlertType = "record_broken"
	// AlertTypeRecordNearMiss : valeur courante à moins de 5% du PB.
	AlertTypeRecordNearMiss AlertType = "record_near_miss"
	// AlertTypeMilestoneUnlocked : milestone débloqué.
	AlertTypeMilestoneUnlocked AlertType = "milestone_unlocked"
	// AlertTypeMilestoneNearMiss : valeur courante à moins de 10% du seuil.
	AlertTypeMilestoneNearMiss AlertType = "milestone_near_miss"
	// AlertTypeLUSRTierApproach : μ à moins de 10 points du sub-tier suivant.
	AlertTypeLUSRTierApproach AlertType = "lusr_tier_approach"
	// AlertTypeStreakMilestone : palier de streak atteint (7j, 14j, 30j).
	AlertTypeStreakMilestone AlertType = "streak_milestone"
	// AlertTypeComebackWelcome : reprise après pause > 5j.
	AlertTypeComebackWelcome AlertType = "comeback_welcome"
	// AlertTypeLOWESSPositive : composante LUSR en tendance positive sur 14j.
	AlertTypeLOWESSPositive AlertType = "lowess_positive"
	// AlertTypeCampaignProgress : progression Mann-Whitney confirmée (V1).
	AlertTypeCampaignProgress AlertType = "campaign_progress"
	// AlertTypeCampaignCloseAuto : axe ciblé sorti du bottom 3 (V1).
	AlertTypeCampaignCloseAuto AlertType = "campaign_close_auto"
)

// AllAlertTypes liste tous les types supportés.
func AllAlertTypes() []AlertType {
	return []AlertType{
		AlertTypeRecordBroken, AlertTypeRecordNearMiss,
		AlertTypeMilestoneUnlocked, AlertTypeMilestoneNearMiss,
		AlertTypeLUSRTierApproach, AlertTypeStreakMilestone,
		AlertTypeComebackWelcome, AlertTypeLOWESSPositive,
		AlertTypeCampaignProgress, AlertTypeCampaignCloseAuto,
	}
}

// DedupWindow est la fenêtre de déduplication par type d'alerte.
// Une alerte du même type pour le même joueur ne peut pas être ré-émise
// dans cette fenêtre. Cf. PLAN §10.1 (risque "notifications trop spammy").
const DedupWindow = 24 * time.Hour

// MaxConcurrentUnread est le plafond d'alertes coach non lues simultanément
// par joueur. Au-delà, les anciennes sont auto-dismissed.
const MaxConcurrentUnread = 3

// AutoDismissAfter est le délai après lequel une alerte non lue est
// automatiquement marquée comme dismissed (purge passive).
const AutoDismissAfter = 7 * 24 * time.Hour

// ComebackPauseThreshold est la durée d'inactivité au-delà de laquelle on
// émet une AlertTypeComebackWelcome au retour.
const ComebackPauseThreshold = 5 * 24 * time.Hour

// LUSRTierApproachDelta est le seuil de proximité (en pts μ) en-deçà duquel
// on déclenche AlertTypeLUSRTierApproach.
const LUSRTierApproachDelta = 10.0

// LOWESSObservationWindow est la fenêtre minimale en jours pour confirmer
// une tendance LOWESS positive.
const LOWESSObservationWindow = 14
