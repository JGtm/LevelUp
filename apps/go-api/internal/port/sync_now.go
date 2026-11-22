package port

import "errors"

// Refus de SyncNowService.StartSyncNow, reconnus par errors.Is.
var (
	// ErrSyncCycleBusy : un cycle de synchronisation tourne déjà (périodique ou demandé).
	ErrSyncCycleBusy = errors.New("sync cycle already running")
	// ErrSyncCycleUnavailable : aucun cycle ne peut être lancé (ordonnanceur absent, arrêt en cours).
	ErrSyncCycleUnavailable = errors.New("sync cycle unavailable")
)

// SyncNowService lance, à la demande, le cycle de synchronisation de l'ordonnanceur : le même que le
// cycle périodique (rattrapage de l'historique compris), pour tous les joueurs suivis.
type SyncNowService interface {
	// StartSyncNow démarre un cycle en arrière-plan et rend aussitôt la main. ErrSyncCycleBusy si un
	// cycle tourne déjà, ErrSyncCycleUnavailable si l'ordonnanceur ne peut pas en lancer. Un clic
	// explicite lance le cycle même quand la synchronisation automatique est désactivée.
	StartSyncNow() error
	// CycleRunning dit si un cycle tourne en ce moment, quelle qu'en soit l'origine (cycle périodique ou
	// demandé) : l'interface s'en sert pour garder le bouton « en cours » jusqu'à la fin du cycle.
	CycleRunning() bool
}
