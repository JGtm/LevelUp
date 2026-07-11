package observability

import (
	"sync"
	"sync/atomic"
	"time"
)

// NetworkFloodWindow est la fenêtre par défaut d'anti-flood des logs d'échec
// réseau répétés (B6.4 plan monitoring). Choisie assez large pour qu'un incident
// infra (panne DNS, Xbox down) qui frappe tous les pollers en boucle ne produise
// qu'une poignée de lignes par minute au lieu de dizaines de milliers — leçon de
// la panne DNS du 2026-07 (67 165 WARN rest_poller en 7 jours).
const NetworkFloodWindow = 30 * time.Second

// throttleNow est la source d'horloge, surchargeable en test.
var throttleNow = time.Now

// throttleState suit l'état d'anti-flood d'une clé : instant de la dernière
// émission autorisée (UnixNano) + nombre d'occurrences étouffées depuis.
type throttleState struct {
	lastEmit   atomic.Int64
	suppressed atomic.Int64
}

var throttleStates sync.Map // key(string) -> *throttleState

// AllowThrottledLog décide si un log répétitif identifié par `key` doit être émis
// maintenant, et rapporte combien d'occurrences ont été étouffées depuis la
// dernière émission autorisée.
//
// Contrat (DC-B2 / règle 3 CLAUDE.md — jamais d'erreur avalée en silence) :
//   - la PREMIÈRE occurrence d'une clé est TOUJOURS autorisée (log de première
//     occurrence) ;
//   - les suivantes dans `window` sont étouffées mais COMPTÉES exactement dans un
//     compteur expvar cumulatif "<key>_total" (visible sur /debug/vars) ;
//   - la première émission APRÈS la fenêtre rapporte `throttledSinceLast` (nombre
//     d'occurrences étouffées depuis la précédente émission), à joindre au log.
//
// Anti-flood transverse : la clé DOIT être globale par cause (ex.
// "log_throttle_rest_poller_transient"), jamais par joueur — sinon un incident
// infra simultané sur N pollers floode toujours (N clés distinctes). Le détail
// par joueur est volontairement perdu pendant une rafale : le premier log porte
// le détail complet, la magnitude reste lisible via le compteur expvar et le
// champ `throttled_since_last`.
func AllowThrottledLog(key string, window time.Duration) (allow bool, throttledSinceLast int64) {
	// Compteur exact de TOUTES les occurrences (étouffées ou non) — jamais avalé.
	IncCounter(key + "_total")

	st := getOrCreateThrottleState(key)
	now := throttleNow().UnixNano()
	last := st.lastEmit.Load()
	if now-last >= int64(window) {
		// Fenêtre écoulée : on tente de gagner le droit d'émettre (CAS pour éviter
		// que deux goroutines émettent en même temps sous la même clé).
		if st.lastEmit.CompareAndSwap(last, now) {
			return true, st.suppressed.Swap(0)
		}
	}
	// Fenêtre encore active, ou course CAS perdue : on étouffe et on compte.
	st.suppressed.Add(1)
	return false, 0
}

func getOrCreateThrottleState(key string) *throttleState {
	if v, ok := throttleStates.Load(key); ok {
		return v.(*throttleState)
	}
	st := &throttleState{}
	actual, _ := throttleStates.LoadOrStore(key, st)
	return actual.(*throttleState)
}

// ResetThrottle réinitialise l'état d'anti-flood (réservé aux tests).
func ResetThrottle() {
	throttleStates.Range(func(k, _ any) bool {
		throttleStates.Delete(k)
		return true
	})
}
