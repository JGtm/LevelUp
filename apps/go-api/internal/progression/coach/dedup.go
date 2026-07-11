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

// StateNudgeDedupWindow (DP13) est la fenêtre de dédup des nudges d'ÉTAT — une
// condition qui persiste (μ proche d'un palier, milestone à 98 %, tendance
// positive/négative) ne doit pas re-notifier chaque jour. Max 1 nudge par cible
// par mois, contre 24 h pour les catégories ÉVÉNEMENT (records battus, streaks).
const StateNudgeDedupWindow = 30 * 24 * time.Hour

// stateNudgeCategories est l'ensemble des catégories « ÉTAT » soumises à la
// fenêtre de 30 jours (DP13). Les autres restent à DedupWindow (24 h).
var stateNudgeCategories = map[notifications.Category]bool{
	notifications.CategoryRecordNearMiss:    true,
	notifications.CategoryMilestoneNearMiss: true,
	notifications.CategoryLUSRTierApproach:  true,
	notifications.CategoryThresholdCrossed:  true, // source coach/LOWESS
	notifications.CategoryTrendConsolidate:  true,
}

// DedupWindowFor résout la fenêtre de dédup applicable à une catégorie (DP13) :
// 30 jours pour les nudges d'état, 24 h sinon.
func DedupWindowFor(cat notifications.Category) time.Duration {
	if stateNudgeCategories[cat] {
		return StateNudgeDedupWindow
	}
	return DedupWindow
}

// FilterRecent retire les alertes déjà émises dans leur fenêtre de dédup, résolue
// PAR CATÉGORIE via windowFor (DP13). Compare la catégorie + le dedup_key encodé
// dans les params de la notification existante (champ `dedup_key`).
//
// Comportement :
//   - Si une notif récente partage (Category, dedup_key) avec l'alerte ET sa date
//     d'émission est dans la fenêtre de CETTE catégorie → alerte filtrée.
//   - dedup_key vide → 1 seule alerte de ce type par fenêtre
//     (utilisé pour comeback_welcome).
//
// `windowFor` peut être nil → DedupWindowFor par défaut.
func FilterRecent(alerts []Alert, recent []notifications.Notification, now time.Time, windowFor func(notifications.Category) time.Duration) []Alert {
	if windowFor == nil {
		windowFor = DedupWindowFor
	}
	// Index : (category, dedup_key) → date d'émission la plus récente.
	latest := make(map[string]time.Time, len(recent))
	for _, n := range recent {
		dedupKey := extractDedupKey(n.Params)
		k := notifKey(n.Category, dedupKey)
		if t, ok := latest[k]; !ok || n.CreatedAt.After(t) {
			latest[k] = n.CreatedAt
		}
	}
	out := alerts[:0]
	for _, a := range alerts {
		cat := a.Type.NotificationCategory()
		if t, ok := latest[notifKey(cat, a.DedupKey)]; ok {
			if !t.Before(now.Add(-windowFor(cat))) {
				continue // dans la fenêtre → filtrée
			}
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
