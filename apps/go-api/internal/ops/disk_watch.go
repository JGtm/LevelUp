// Package ops — disk_watch.go : politique PURE de notification d'alerte disque
// (lot ops 2026-07-13, suite de l'incident disque-plein VPS du 2026-07-13 :
// l'alerte hôte existante écrivait dans journald que personne ne lit).
//
// La mesure et l'envoi vivent chez le caller (wire.RunDiskWatchLoop : diskfree
// sur le volume data + notify Discord) ; ici uniquement la décision d'alerter,
// testable sans horloge ni réseau.
package ops

import (
	"time"

	"levelup/go-api/internal/domain"
)

// DiskRenotifyInterval : tant que le disque reste en warn/critical, on renvoie
// un rappel à cette fréquence (une alerte unique se perd ; un rappel quotidien
// reste discret).
const DiskRenotifyInterval = 24 * time.Hour

// DiskWatchState est l'état mémoire de la boucle de surveillance (process-local,
// repart de zéro au boot — au pire une notification de plus après restart).
type DiskWatchState struct {
	// LastStatus est le dernier statut OBSERVÉ (ok/warn/critical). Vide au boot.
	LastStatus string
	// LastNotifiedAt est l'horodatage de la dernière notification envoyée.
	LastNotifiedAt time.Time
}

// ShouldNotifyDisk décide si le statut courant justifie une notification :
//   - entrée ou aggravation/amélioration ENTRE warn et critical (transition) ;
//   - rappel périodique (DiskRenotifyInterval) tant que le breach persiste ;
//   - rétablissement (breach → ok) : notification de recovery ;
//   - statut unknown : jamais de notification (mesure indisponible ≠ incident),
//     et l'état n'est pas modifié (un unknown transitoire ne masque pas le breach).
//
// Retourne (notifier, nouvel état). Le caller n'envoie que si notifier == true
// et met TOUJOURS à jour son état avec le retour.
func ShouldNotifyDisk(state DiskWatchState, current string, now time.Time) (bool, DiskWatchState) {
	if current == domain.FreshnessStatusUnknown || current == "" {
		return false, state
	}
	inBreach := current == domain.FreshnessStatusWarn || current == domain.FreshnessStatusCritical
	wasBreach := state.LastStatus == domain.FreshnessStatusWarn || state.LastStatus == domain.FreshnessStatusCritical

	next := state
	next.LastStatus = current

	switch {
	case inBreach && current != state.LastStatus:
		// Entrée en breach ou changement de sévérité (warn↔critical).
		next.LastNotifiedAt = now
		return true, next
	case inBreach && now.Sub(state.LastNotifiedAt) >= DiskRenotifyInterval:
		// Rappel : le breach persiste depuis plus d'un intervalle.
		next.LastNotifiedAt = now
		return true, next
	case !inBreach && wasBreach:
		// Rétablissement — informer que l'incident est clos.
		next.LastNotifiedAt = now
		return true, next
	default:
		return false, next
	}
}
