package coach

import (
	"encoding/json"
	"time"

	"levelup/go-api/internal/notifications"
)

// dedup.go — filtrage des alertes déjà émises récemment.
//
// Sans table dédiée coach_alert (cf. §2bis du plan), la dédup s'appuie sur
// l'historique de `player_notifications`. L'orchestrateur fournit la liste
// des notifs récentes (typ. 24h) ; ce module filtre les alertes dont la
// catégorie + dedup_key est déjà présente.

// FilterRecent retire les alertes qui ont déjà été émises dans `window`
// (typ. DedupWindow = 24h). Compare la catégorie + le dedup_key encodé
// dans les params de la notification existante (champ `dedup_key`).
//
// Comportement :
//   - Si une notif récente partage (Category, dedup_key) avec l'alerte
//     ET sa date d'émission est dans la fenêtre → alerte filtrée.
//   - dedup_key vide → 1 seule alerte de ce type par fenêtre
//     (utilisé pour comeback_welcome).
func FilterRecent(alerts []Alert, recent []notifications.Notification, now time.Time, window time.Duration) []Alert {
	cutoff := now.Add(-window)
	// Index : (category, dedup_key) → existence dans la fenêtre.
	seen := make(map[string]bool, len(recent))
	for _, n := range recent {
		if n.CreatedAt.Before(cutoff) {
			continue
		}
		dedupKey := extractDedupKey(n.Params)
		seen[notifKey(n.Category, dedupKey)] = true
	}
	out := alerts[:0]
	for _, a := range alerts {
		k := notifKey(a.Type.NotificationCategory(), a.DedupKey)
		if seen[k] {
			continue
		}
		out = append(out, a)
	}
	return out
}

// AnnotateDedupKey injecte le champ `dedup_key` dans Alert.Params pour qu'il
// soit persisté dans la notif et puisse être lu par FilterRecent à la passe
// suivante. À appeler juste avant émission.
//
// Mutation in-place : Params est créé si nil.
func AnnotateDedupKey(a *Alert) {
	if a.DedupKey == "" {
		return
	}
	if a.Params == nil {
		a.Params = map[string]any{}
	}
	a.Params["dedup_key"] = a.DedupKey
}

// extractDedupKey lit le champ `dedup_key` d'un payload JSON de notification.
// Retourne "" si absent ou parsing échoue (cas dégradé : pas de dédup).
func extractDedupKey(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	if v, ok := m["dedup_key"].(string); ok {
		return v
	}
	return ""
}

// notifKey construit la clé d'index unique pour la dédup.
func notifKey(cat notifications.Category, dedupKey string) string {
	return string(cat) + "|" + dedupKey
}
