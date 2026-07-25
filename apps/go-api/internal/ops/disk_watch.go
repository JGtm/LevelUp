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

// diskConfirmTicks : nombre d'observations CONSÉCUTIVES d'une AMÉLIORATION
// (dé-escalade / rétablissement) avant de la confirmer. Débounce anti-oscillation :
// l'espace libre mesuré fluctue naturellement (WAL DuckDB, fichiers temporaires,
// logs rotés) ; un volume qui stationne à ±0,5 % d'un seuil produirait sinon une
// paire alerte/rétablissement à chaque tick (~96 messages/jour). Les AGGRAVATIONS
// ne sont jamais débouncées (un disque qui se remplit doit alerter tout de suite).
const diskConfirmTicks = 2

// DiskWatchState est l'état de la boucle de surveillance. PERSISTÉ hors DuckDB
// (JSON, PathResolver.DiskWatchStatePath) et réhydraté au boot : sans cela l'état
// repartait de zéro à chaque redémarrage du conteneur (deploy à chaque push main,
// crash-loop disque plein) → le chemin « boot en warn/critical » re-notifiait à
// CHAQUE boot = rafale d'alertes Discord identiques. La persistance conserve
// l'anti-spam (dernier statut confirmé + timestamp) et la garantie « une seule
// notification de rétablissement » à travers les restarts. Les tags JSON sont
// nécessaires à la sérialisation FileStore.
type DiskWatchState struct {
	// LastStatus est le dernier statut CONFIRMÉ (ok/warn/critical). Vide au boot.
	LastStatus string `json:"last_status"`
	// LastNotifiedAt est l'horodatage de la dernière notification envoyée.
	LastNotifiedAt time.Time `json:"last_notified_at"`
	// PendingImprove : statut d'AMÉLIORATION observé, en attente de confirmation
	// (débounce). Vide dès qu'un tick revient au statut confirmé ou aggrave.
	PendingImprove string `json:"pending_improve"`
	// PendingImproveTicks : nb d'observations consécutives de PendingImprove.
	PendingImproveTicks int `json:"pending_improve_ticks"`
}

// diskSeverity ordonne les statuts (ok < warn < critical) pour distinguer
// aggravation (immédiate) et amélioration (débouncée).
func diskSeverity(status string) int {
	switch status {
	case domain.FreshnessStatusWarn:
		return 1
	case domain.FreshnessStatusCritical:
		return 2
	default: // ok
		return 0
	}
}

// ShouldNotifyDisk décide si le statut courant justifie une notification :
//   - aggravation (ok→warn, warn→critical) : notification IMMÉDIATE ;
//   - amélioration (dé-escalade / rétablissement) : confirmée sur diskConfirmTicks
//     observations consécutives (débounce anti-oscillation) puis notifiée ;
//   - breach persistant (statut stable == dernier confirmé, toujours en warn/critical) :
//     AUCUN rappel (décision V72-31, 2026-07-25 : la notification d'entrée en alerte
//     suffit ; une aggravation ultérieure re-notifie via le cas ci-dessus) ;
//   - statut unknown : jamais de notification (mesure indisponible ≠ incident),
//     et l'état n'est pas modifié (un unknown transitoire ne masque ni le breach
//     ni un débounce en cours).
//
// Retourne (notifier, nouvel état). Le caller n'envoie que si notifier == true
// et met TOUJOURS à jour son état avec le retour.
func ShouldNotifyDisk(state DiskWatchState, current string, now time.Time) (bool, DiskWatchState) {
	if current == domain.FreshnessStatusUnknown || current == "" {
		return false, state
	}
	next := state

	// Boot (aucun statut confirmé) : le premier relevé est confirmé d'emblée.
	if state.LastStatus == "" {
		next.LastStatus = current
		next.PendingImprove = ""
		next.PendingImproveTicks = 0
		if diskSeverity(current) > 0 {
			next.LastNotifiedAt = now
			return true, next
		}
		return false, next
	}

	sevCur, sevLast := diskSeverity(current), diskSeverity(state.LastStatus)
	switch {
	case sevCur > sevLast:
		// Aggravation : confirmée immédiatement, annule tout débounce d'amélioration.
		next.LastStatus = current
		next.PendingImprove = ""
		next.PendingImproveTicks = 0
		next.LastNotifiedAt = now
		return true, next

	case sevCur < sevLast:
		// Amélioration : débounce (diskConfirmTicks observations consécutives).
		if current == state.PendingImprove {
			next.PendingImproveTicks = state.PendingImproveTicks + 1
		} else {
			next.PendingImprove = current
			next.PendingImproveTicks = 1
		}
		if next.PendingImproveTicks >= diskConfirmTicks {
			next.LastStatus = current
			next.PendingImprove = ""
			next.PendingImproveTicks = 0
			next.LastNotifiedAt = now
			return true, next
		}
		// Pas encore confirmée : statut confirmé inchangé, pas de notification.
		return false, next

	default:
		// Statut stable (== dernier confirmé) : un débounce d'amélioration en cours
		// est réinitialisé (l'amélioration ne s'est pas maintenue). Aucun rappel même
		// en breach persistant (warn/critical) — décision V72-31 : l'entrée en alerte
		// a déjà notifié une fois, et une aggravation ultérieure re-notifie via le cas
		// sevCur > sevLast ci-dessus.
		next.PendingImprove = ""
		next.PendingImproveTicks = 0
		return false, next
	}
}
