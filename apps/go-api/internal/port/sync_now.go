package port

import (
	"errors"
	"fmt"
	"time"
)

// Refus de SyncNowService.StartSyncNow, reconnus par errors.Is.
var (
	// ErrSyncCycleBusy : un cycle de synchronisation tourne déjà (périodique ou demandé).
	ErrSyncCycleBusy = errors.New("sync cycle already running")
	// ErrSyncCycleUnavailable : aucun cycle ne peut être lancé (ordonnanceur absent, arrêt en cours).
	ErrSyncCycleUnavailable = errors.New("sync cycle unavailable")
	// ErrSyncCycleCooldown : un cycle s'est terminé il y a moins que le délai minimal entre deux cycles
	// demandés ; l'erreur concrète est un *SyncCooldownError, qui porte le délai restant.
	ErrSyncCycleCooldown = errors.New("sync cycle cooldown")
)

// SyncCooldownError est le refus d'un cycle demandé trop tôt après la fin d'un cycle (toute origine).
type SyncCooldownError struct {
	// RetryAfter : délai restant avant que la demande soit acceptée ; toujours positif.
	RetryAfter time.Duration
}

func (e *SyncCooldownError) Error() string {
	return fmt.Sprintf("%v: retry after %s", ErrSyncCycleCooldown, e.RetryAfter.Round(time.Second))
}

// Is fait reconnaître ErrSyncCycleCooldown par errors.Is.
func (e *SyncCooldownError) Is(target error) bool { return target == ErrSyncCycleCooldown }

// SyncNowService lance, à la demande, le cycle de synchronisation de l'ordonnanceur : le même que le
// cycle périodique (rattrapage de l'historique compris), pour tous les joueurs suivis.
type SyncNowService interface {
	// StartSyncNow démarre un cycle en arrière-plan et rend aussitôt la main. ErrSyncCycleBusy si un
	// cycle tourne déjà, *SyncCooldownError si un cycle s'est terminé trop récemment, ErrSyncCycleUnavailable
	// si l'ordonnanceur ne peut pas en lancer. Un clic explicite lance le cycle même quand la
	// synchronisation automatique est désactivée.
	StartSyncNow() error
	// CycleRunning dit si un cycle tourne en ce moment, quelle qu'en soit l'origine (cycle périodique ou
	// demandé) : l'interface s'en sert pour garder le bouton « en cours » jusqu'à la fin du cycle.
	CycleRunning() bool
	// ManualWindow rend la fin du dernier cycle connu (toute origine ; zéro si aucun) et l'instant à partir
	// duquel un cycle demandé est de nouveau accepté (zéro = accepté tout de suite).
	ManualWindow() (lastCycleAt, availableAt time.Time)
}
