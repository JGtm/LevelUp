package coach

import (
	"levelup/go-api/internal/notifications"
)

// emitter.go — mapping AlertType → catégorie notifications + clés i18n.
//
// Le coach ne stocke pas ses alertes dans une table dédiée (cf. §2bis du plan
// PROGRESSION_TRACKING) : il utilise le système `player_notifications`
// existant. Ce fichier centralise la conversion entre le domaine coach
// (AlertType, params métier) et le format `notifications.EmitInput`.

// NotificationCategory retourne la catégorie notification associée à un
// AlertType. Les types qui réutilisent une catégorie existante :
//   - AlertTypeRecordBroken     → CategoryPersonalRecord (déjà existant)
//   - AlertTypeLOWESSPositive   → CategoryThresholdCrossed
//   - AlertTypeCampaignProgress → CategoryThresholdCrossed (avec param campaign_id)
//   - AlertTypeCampaignCloseAuto → CategoryThresholdCrossed (param close_suggestion=true)
//
// Les autres types ont leur propre catégorie ajoutée dans
// internal/notifications/types.go (§2bis plan).
func (a AlertType) NotificationCategory() notifications.Category {
	switch a {
	case AlertTypeRecordBroken:
		return notifications.CategoryPersonalRecord
	case AlertTypeRecordNearMiss:
		return notifications.CategoryRecordNearMiss
	case AlertTypeMilestoneUnlocked:
		return notifications.CategoryMilestoneUnlocked
	case AlertTypeMilestoneNearMiss:
		return notifications.CategoryMilestoneNearMiss
	case AlertTypeLUSRTierApproach:
		return notifications.CategoryLUSRTierApproach
	case AlertTypeStreakMilestone:
		return notifications.CategoryStreakMilestone
	case AlertTypeComebackWelcome:
		return notifications.CategoryComebackWelcome
	case AlertTypeLOWESSPositive, AlertTypeCampaignProgress, AlertTypeCampaignCloseAuto:
		return notifications.CategoryThresholdCrossed
	case AlertPatternStrength:
		return notifications.CategoryPatternStrength
	case AlertPatternWeakness:
		return notifications.CategoryPatternWeakness
	case AlertPatternBehavior:
		return notifications.CategoryPatternBehavior
	case AlertPatternLever:
		return notifications.CategoryPatternLever
	case AlertTypeCombatPatternActif, AlertTypeCombatPatternDiscret, AlertTypeCombatPatternFragile:
		return notifications.CategoryCombatPattern
	default:
		return ""
	}
}

// TitleKey retourne la clé i18n du titre de la notif pour un AlertType.
// Convention : `notif.<category>.title`. Le manifest i18n correspondant
// est livré en commit 11.
func (a AlertType) TitleKey() string {
	cat := a.NotificationCategory()
	if cat == "" {
		return ""
	}
	return "notif." + string(cat) + ".title"
}

// BodyKey retourne la clé i18n du body pour un AlertType. Convention :
// `notif.<category>.body`. Les params (metric, value, etc.) sont passés
// au moment du rendu pour interpolation.
func (a AlertType) BodyKey() string {
	cat := a.NotificationCategory()
	if cat == "" {
		return ""
	}
	return "notif." + string(cat) + ".body"
}

// Source est l'identifiant `Source` requis par notifications.EmitInput.
// Tous les alerts du coach partagent la même source pour faciliter le
// filtrage et la dédup.
const Source = "progression.coach"
