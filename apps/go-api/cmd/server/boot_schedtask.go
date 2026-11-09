// boot_schedtask.go — la tâche planifiée refresh-only du serveur desktop (lot L21.e, ADR 0045 D-10) : pose la
// tâche Windows quand le réglage `scheduled_refresh` passe actif, la retire quand il passe coupé, vérifie son
// état au démarrage et l'expose à la page des réglages. Ne tourne que sous la capacité ScheduledRefresh
// (profil desktop). La mécanique de schtasks est dans internal/platform/schedtask ; ce fichier la branche sur
// les réglages vivants.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"levelup/go-api/internal/platform/schedtask"
)

// taskManager : ce que le réconciliateur demande à la tâche (satisfait par *schedtask.Manager).
type taskManager interface {
	Query(ctx context.Context) (schedtask.State, error)
	Ensure(ctx context.Context) error
	Remove(ctx context.Context) error
}

// launchScheduledRefresh lance la boucle de la tâche planifiée sous la capacité ScheduledRefresh. Elle tourne
// sous le contexte des tâches de fond et dans leur groupe d'attente.
func launchScheduledRefresh(st *bootState) {
	if !st.cfg.Capabilities.ScheduledRefresh {
		return
	}
	ctx := st.schedulerCtx
	obs := schedtask.DefaultObserved()
	exe, err := os.Executable()
	if err != nil {
		slog.ErrorContext(ctx, "schedtask: exécutable du serveur introuvable, tâche planifiée indisponible", "err", err)
		obs.Set(schedtask.StateError, schedtask.ErrorCodeSchtasks)
		return
	}
	roots := st.cfg.Roots()
	mgr := &schedtask.Manager{
		Runner: schedtask.ExecRunner{},
		Spec: schedtask.Spec{
			ExePath: exe, DataRoot: roots.Data, ResourcesRoot: roots.Resources, LogsRoot: roots.Logs,
		},
	}
	st.schedulerWG.Add(1)
	go func() {
		defer st.schedulerWG.Done()
		runScheduledRefreshLoop(ctx, st.settingsStore, newRefreshReconciler(mgr, obs), nil, shellSettingsEvery)
	}()
	slog.InfoContext(ctx, "schedtask: tâche planifiée refresh-only armée (réglage scheduled_refresh)",
		"tache", mgr.TaskName(), "periode", schedtask.Interval)
}

// refreshReconciler ramène la tâche planifiée à l'état que le réglage demande.
type refreshReconciler struct {
	mgr     taskManager
	obs     *schedtask.Observed
	applied *bool // la dernière valeur du réglage traitée ; nil avant la vérification du démarrage
}

func newRefreshReconciler(mgr taskManager, obs *schedtask.Observed) *refreshReconciler {
	return &refreshReconciler{mgr: mgr, obs: obs}
}

// reconcile traite la valeur voulue du réglage.
//
// Au démarrage (première valeur), réglage actif : la tâche est TOUJOURS réécrite (`/F`, un seul appel), présente
// ou non : une réinstallation dans un autre dossier ou une racine changée laisserait sinon une tâche périmée
// affichée « présente » ; réglage coupé et tâche présente (reste d'une installation précédente), elle est
// retirée ; réglage coupé et tâche absente, rien. Ensuite, chaque CHANGEMENT du réglage pose ou retire la tâche. Une valeur déjà traitée ne refait rien : un échec n'est
// pas retenté à chaque tic (le réglage qui bascule, ou le prochain démarrage, retente), il est journalisé et
// exposé.
func (r *refreshReconciler) reconcile(ctx context.Context, want bool) {
	first := r.applied == nil
	if !first && *r.applied == want {
		return
	}
	r.applied = &want
	state, err := r.mgr.Query(ctx)
	if err != nil {
		slog.WarnContext(ctx, "schedtask: état de la tâche planifiée illisible", "err", err)
		r.obs.Set(schedtask.StateError, schedtask.ErrorCodeSchtasks)
		return
	}
	if first && !want && state == schedtask.StateAbsent {
		r.obs.Set(state, "")
		return
	}
	if err := r.apply(ctx, want); err != nil {
		code := schedtask.ErrorCodeSchtasks
		if errors.Is(err, schedtask.ErrUnsupportedPath) {
			code = schedtask.ErrorCodePath
		}
		slog.WarnContext(ctx, "schedtask: tâche planifiée non mise à jour", "voulue", want, "err", err)
		r.obs.Set(schedtask.StateError, code)
		return
	}
	final, err := r.mgr.Query(ctx)
	if err != nil {
		slog.WarnContext(ctx, "schedtask: état de la tâche planifiée illisible après mise à jour", "err", err)
		r.obs.Set(schedtask.StateError, schedtask.ErrorCodeSchtasks)
		return
	}
	slog.InfoContext(ctx, "schedtask: tâche planifiée mise à jour", "voulue", want, "etat", string(final))
	r.obs.Set(final, "")
}

// apply pose ou retire la tâche.
func (r *refreshReconciler) apply(ctx context.Context, want bool) error {
	if want {
		return r.mgr.Ensure(ctx)
	}
	return r.mgr.Remove(ctx)
}

// runScheduledRefreshLoop relit le réglage à chaque tic (ticks nil : un ticker de every, arrêté avec la boucle)
// et le traite. Un réglage illisible ne change rien et se journalise une fois par panne. Bloquant : à lancer
// dans une goroutine suivie.
func runScheduledRefreshLoop(ctx context.Context, store settingsLoader, r *refreshReconciler, ticks <-chan time.Time, every time.Duration) {
	if ticks == nil {
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		ticks = ticker.C
	}
	failing := false
	poll := func() {
		cfg, err := store.Load()
		if err != nil {
			if !failing {
				slog.WarnContext(ctx, "schedtask: réglages illisibles, la tâche planifiée reste comme elle est", "err", err)
			}
			failing = true
			return
		}
		failing = false
		r.reconcile(ctx, cfg.ScheduledRefresh)
	}
	poll()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			poll()
		}
	}
}
