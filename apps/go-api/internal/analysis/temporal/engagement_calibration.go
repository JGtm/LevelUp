package temporal

import "levelup/go-api/internal/games/canonical"

// engagement_calibration.go — coefficients d'engagement PAR TITRE (chantier F7).
//
// Les poids d'events (l'importance relative de chaque famille d'action dans le
// RYTHME de la courbe) sont le levier de calibration DÉPENDANT DU GAMEPLAY : un
// titre au rythme d'objectif différent, aux assists natives plus/moins fréquentes,
// peut vouloir d'autres poids. Ils sont donc externalisés par titre
// (config/titles/{slug}/constants.toml [engagement], chargé via
// games.EngagementWeightsFor(slug)) au lieu d'être des constantes Go partagées.
//
// Le moteur reste agnostic : il reçoit les poids en entrée (EngagementScoreInput.
// Weights). Un titre sans déclaration retombe sur DefaultEventWeights (byte-identique
// à l'historique Infinite). cf. plan F7 DE-4.

// EventWeights porte les poids d'engagement par famille d'event. Zéro-value = non
// renseigné (le compute retombe sur DefaultEventWeights).
type EventWeights struct {
	// Objective : event objectif ("mode", leadership — porter > fragger). Défaut 1.5.
	Objective float64
	// Assist : support (action menée, pas un double comptage du frag). Défaut 0.5.
	Assist float64
	// Death : mort subie. Défaut 0.0 (un affrontement compte une fois, côté acteur).
	Death float64
	// Default : kill, medal, first_kill/death, finisher, clutch… Défaut 1.0.
	Default float64
}

// DefaultEventWeights retourne les poids historiques validés (mode 1.5 / assist 0.5
// / death 0.0 / défaut 1.0, décisions user 2026-06-26 + 2026-07-07). Référence
// byte-identique quand un titre ne déclare pas ses poids.
func DefaultEventWeights() EventWeights {
	return EventWeights{Objective: 1.5, Assist: 0.5, Death: 0.0, Default: 1.0}
}

// IsZero indique des poids non renseignés (les 4 champs nuls). Sentinelle pour la
// résolution par défaut dans ComputeEngagementScore.
func (w EventWeights) IsZero() bool {
	return w.Objective == 0 && w.Assist == 0 && w.Death == 0 && w.Default == 0
}

// For retourne le poids d'un type d'event brut. Même mapping que l'historique
// engagementEventWeight, mais paramétré par titre.
func (w EventWeights) For(eventType string) float64 {
	switch eventType {
	case "mode": // objectif (Infinite event_type "mode" + impulses objectif H5)
		return w.Objective
	case string(canonical.EventAssist):
		return w.Assist
	case string(canonical.EventDeath):
		return w.Death
	default:
		return w.Default
	}
}
