// Package api — registry_action_journal.go : câblage + lecture du journal des
// actions globales du dashboard admin (C2). Écriture par les runners d'action
// (à côté des compteurs observability) ; lecture par GET /admin/actions/journal.
package wire

import (
	"context"
	"sort"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/adminstate"
)

// WithActionJournal branche le journal des actions globales (persistance JSON
// hors DuckDB, survit au reboot). Partagé avec le scheduler (cycle de sync).
// Nil → les runners ne journalisent pas et l'endpoint sert un journal vide.
func (r *ServiceRegistry) WithActionJournal(j *adminstate.ActionJournal) *ServiceRegistry {
	r.actionJournal = j
	return r
}

// journalAction enregistre la dernière exécution MANUELLE d'une action globale.
// Point unique de la règle err→issue (adminstate.Outcome) pour tous les runners
// — évite la duplication du mapping. Nil-safe (journal non câblé → no-op).
func (r *ServiceRegistry) journalAction(ctx context.Context, action string, err error) {
	r.actionJournal.Record(ctx, action, adminstate.Outcome(err), adminstate.TriggerManual)
}

// ActionJournalReport agrège le journal des actions globales pour l'endpoint
// admin (dernière exécution / issue / déclencheur par action, trié par nom).
// Nil-safe : journal non câblé → réponse vide (jamais d'erreur).
func (r *ServiceRegistry) ActionJournalReport(ctx context.Context) domain.AdminActionJournalResponse {
	_ = ctx
	resp := domain.AdminActionJournalResponse{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Actions:     []domain.AdminActionJournalEntry{},
	}
	snap := r.actionJournal.Snapshot()
	names := make([]string, 0, len(snap))
	for name := range snap {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		rec := snap[name]
		lastRun := ""
		if !rec.LastRunAt.IsZero() {
			lastRun = rec.LastRunAt.UTC().Format(time.RFC3339)
		}
		resp.Actions = append(resp.Actions, domain.AdminActionJournalEntry{
			Action:    name,
			LastRunAt: lastRun,
			Outcome:   rec.Outcome,
			Trigger:   rec.Trigger,
		})
	}
	return resp
}
