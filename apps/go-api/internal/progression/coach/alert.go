package coach

import (
	"levelup/go-api/internal/notifications"
)

// alert.go — type Alert, intent d'alerte coach prêt pour émission.

// Alert est l'intent d'alerte produit par le générateur, prêt à être converti
// en notifications.EmitInput par le caller (orchestrateur).
//
// Champ `DedupKey` : clé stable identifiant le contenu de l'alerte (au-delà
// de la catégorie). Deux alertes de même Type avec même DedupKey dans la
// fenêtre `DedupWindow` ne doivent pas être ré-émises. Exemples :
//   - record_near_miss : "performance_score|30d"
//   - milestone_near_miss : "halo_infinite.wins.250"
//   - streak_milestone : "daily_play|7"  (palier 7j de streak)
//   - lusr_tier_approach : "diamond_iv"  (tier ciblé)
//   - comeback_welcome : "" (1 seule alerte type par fenêtre)
type Alert struct {
	Type     AlertType
	Severity notifications.Severity
	Params   map[string]any
	DedupKey string
}

// ToEmitInput convertit l'Alert en notifications.EmitInput prêt pour
// Emitter.Emit. Renvoie un input invalide (Category vide) si le type
// est inconnu — utiliser ToEmitInput().Validate() avant d'émettre.
func (a Alert) ToEmitInput() notifications.EmitInput {
	return notifications.EmitInput{
		Category: a.Type.NotificationCategory(),
		Severity: a.Severity,
		TitleKey: a.Type.TitleKey(),
		BodyKey:  a.Type.BodyKey(),
		Params:   a.Params,
		Source:   Source,
	}
}
