// Package scheduler — auto_sync_catchup.go : le rattrapage d'un cycle de sync hors de l'horloge
// périodique (ADR 0045 D-10).
//
// Le ticker de Run compte en temps monotone : sous Windows (et sous Linux) il ne compte pas la veille,
// et un poste qui dort huit heures ne tire pas son tick à l'heure. La séquence de réveil
// (internal/wakeup) demande alors un rattrapage : [AutoSyncScheduler.CatchUpDue] dit si la dernière
// synchronisation connue est plus vieille que l'intervalle, [AutoSyncScheduler.RequestCatchUp] fait
// tirer UN tick tout de suite, dans la boucle de Run (jamais en parallèle d'un tick périodique).
//
// Rien ici ne se déclenche seul : sans appelant, le comportement de Run est celui d'avant. Le
// profil qui appelle est décidé par la capacité de déploiement, pas par ce paquet.
package scheduler

import "time"

// catchUpQueue : taille de la file des rattrapages demandés. Un seul en attente suffit : deux
// demandes avant que la boucle les lise se confondent en un tick.
const catchUpQueue = 1

// CatchUpDue dit si un cycle de rattrapage est dû à l'instant now : l'auto-sync est actif ET la
// dernière synchronisation connue (le dernier cycle, relu du disque au boot, cf. RehydrateFromDisk)
// date de plus que l'intervalle courant. Aucun cycle connu (instance neuve ou instantané perdu)
// vaut « dû ». Rend aussi la date du dernier cycle (zéro si inconnue) et l'intervalle, pour le journal.
//
// L'âge se mesure à l'horloge MURALE : un horodatage pris par ce processus porte une lecture
// monotone qui ne compte pas la veille, et la différence de deux lectures monotones sous-estimerait
// justement l'absence qu'on cherche (ADR 0045 D-10).
func (s *AutoSyncScheduler) CatchUpDue(now time.Time) (due bool, last time.Time, interval time.Duration) {
	cfg, err := s.settings.Load()
	if err != nil {
		return false, time.Time{}, 0 // le tick ignorerait la demande : Run journalise déjà cette erreur
	}
	interval = resolveInterval(cfg.SpnkrAutoSyncIntervalMinutes, cfg.SpnkrAutoSyncIntervalHours)
	s.snapshotMu.RLock()
	last = s.lastCycleAt
	s.snapshotMu.RUnlock()
	if !cfg.SpnkrAutoSyncEnabled {
		return false, last, interval
	}
	if last.IsZero() {
		return true, last, interval
	}
	return now.Round(0).Sub(last.Round(0)) > interval, last, interval
}

// RequestCatchUp demande à la boucle de Run un tick immédiat ; reason alimente le journal. Ne
// bloque jamais : rend false si une demande attend déjà (le tick à venir couvre celle-ci). Sans
// boucle de Run en cours, la demande attend son démarrage.
func (s *AutoSyncScheduler) RequestCatchUp(reason string) bool {
	select {
	case s.catchUpCh <- reason:
		return true
	default:
		return false
	}
}
