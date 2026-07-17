package external

import "levelup/go-api/internal/notifications"

// DefaultForwardedCategories retourne les catégories relayées par défaut vers le
// canal externe : UNIQUEMENT les catégories émises par le coach proactif
// (proposals de progression). Toute autre notification in-app (sync, média,
// version, ami…) reste locale.
//
// Source de vérité des catégories coach : internal/progression/coach/emitter.go
// (AlertType.NotificationCategory). Cette liste en est le miroir explicite ; un
// garde-rail (categories_guardrail_test.go) échoue si les deux divergent, pour
// éviter qu'un nouvel AlertType coach ne soit silencieusement non relayé (ou
// qu'une catégorie non-coach y entre par erreur).
func DefaultForwardedCategories() []notifications.Category {
	return []notifications.Category{
		notifications.CategoryPersonalRecord,
		notifications.CategoryRecordNearMiss,
		notifications.CategoryMilestoneUnlocked,
		notifications.CategoryMilestoneNearMiss,
		notifications.CategoryLUSRTierApproach,
		notifications.CategoryStreakMilestone,
		notifications.CategoryComebackWelcome,
		notifications.CategoryThresholdCrossed,
		notifications.CategoryTrendConsolidate,
		notifications.CategoryPatternStrength,
		notifications.CategoryPatternWeakness,
		notifications.CategoryPatternBehavior,
		notifications.CategoryPatternLever,
		notifications.CategoryCombatPattern,
	}
}
