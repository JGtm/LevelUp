// Package api — registry_match_steps.go : le DÉLAI DU DÉBRIEF côté serveur — vidage du
// collecteur des instants mesurés (observability.ReportMatchStep) vers la base monitoring, et
// lecture du rapport joint au panneau de la file de construction.
package wire

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

// matchDelaysLimit : matchs servis par le tableau « délai du débrief » de l'admin.
const matchDelaysLimit = 50

// flushMatchSteps écrit dans le store les instants captés depuis le dernier vidage. Un échec
// remet les instants en tête du collecteur pour le vidage suivant : ils portent l'heure de
// l'événement, les écrire en retard ne fausse rien.
func (r *ServiceRegistry) flushMatchSteps(ctx context.Context) {
	if r.monitoringStore == nil {
		return
	}
	events := observability.DrainMatchSteps()
	if err := r.monitoringStore.RecordMatchSteps(ctx, events); err != nil {
		monitoringLog.WarnContext(ctx, "admin_monitoring: écriture des instants mesurés échouée, reportée",
			"err", err, "instants", len(events))
		observability.RequeueMatchSteps(events)
	}
}

// attachMatchDelays joint au rapport de la file le délai du débrief des matchs récents, lu APRÈS
// un vidage du collecteur : l'admin voit les instants de la dernière minute sans attendre la
// boucle de flush.
func (r *ServiceRegistry) attachMatchDelays(ctx context.Context, resp *domain.AdminBuildQueueResponse) error {
	if r.monitoringStore == nil {
		return nil
	}
	r.flushMatchSteps(ctx)
	delais, err := r.monitoringStore.MatchDelays(ctx, matchDelaysLimit)
	if err != nil {
		return err
	}
	resp.MatchDelays = delais
	return nil
}
