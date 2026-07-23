package adminstate

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Déclencheurs d'une action journalisée.
const (
	TriggerManual = "manual" // action lancée à la main depuis l'admin (bouton)
	TriggerCron   = "tick"   // cycle périodique du scheduler
)

// Issues d'une exécution journalisée.
const (
	OutcomeOK    = "ok"
	OutcomeError = "error"
)

// Noms canoniques des actions globales journalisées. Partagés côté Go ; le front
// mappe LES MÊMES clés sous chaque bouton d'action globale (page Qualité données
// + actions rapides de l'onglet État).
const (
	ActionRegistryNames   = "registry_names_backfill"
	ActionCatalogRefresh  = "catalog_refresh"
	ActionLyingBitsReset  = "lying_bits_reset"
	ActionCatalogUGCDrain = "catalog_ugc_drain"
	ActionDataHealth      = "data_health"
	ActionSyncCycle       = "sync_cycle"
)

// Outcome mappe une erreur d'exécution vers l'issue journalisée (point unique de
// la règle err→outcome — évite la duplication sur chaque call-site).
func Outcome(err error) string {
	if err != nil {
		return OutcomeError
	}
	return OutcomeOK
}

// ActionRecord est la dernière exécution connue d'une action.
type ActionRecord struct {
	LastRunAt time.Time `json:"last_run_at"`
	Outcome   string    `json:"outcome"` // OutcomeOK | OutcomeError
	Trigger   string    `json:"trigger"` // TriggerManual | TriggerCron
}

// ActionJournal mémorise la dernière exécution de chaque action globale et la
// persiste atomiquement (survit au reboot). Thread-safe : le scheduler (cycle de
// sync) et les handlers/services d'action écrivent concurremment.
type ActionJournal struct {
	store   *FileStore
	mu      sync.Mutex
	entries map[string]ActionRecord
}

// NewActionJournal crée un journal adossé au FileStore donné (vide jusqu'à Load).
func NewActionJournal(store *FileStore) *ActionJournal {
	return &ActionJournal{store: store, entries: map[string]ActionRecord{}}
}

// Load réhydrate le journal depuis le disque (à appeler une fois au boot).
// Fichier absent → journal vide (pas d'erreur). Corruption → erreur remontée
// (le caller LOG et démarre à vide, sans écraser le fichier).
func (j *ActionJournal) Load(ctx context.Context) error {
	if j == nil || j.store == nil {
		return nil
	}
	var entries map[string]ActionRecord
	found, err := j.store.Load(&entries)
	if err != nil {
		return err
	}
	if !found || entries == nil {
		return nil
	}
	j.mu.Lock()
	j.entries = entries
	j.mu.Unlock()
	slog.InfoContext(ctx, "adminstate: journal des actions réhydraté", "actions", len(entries))
	return nil
}

// Record enregistre la dernière exécution d'une action et persiste le journal.
// Nil-safe (journal non câblé → no-op). La sauvegarde est effectuée SOUS le
// verrou (sérialise le read-modify-write : pas de perte de mise à jour entre
// deux actions concourantes). Un échec d'écriture est LOGGÉ (jamais avalé) mais
// ne remonte pas : l'action a réussi, la persistance est best-effort.
func (j *ActionJournal) Record(ctx context.Context, action, outcome, trigger string) {
	if j == nil || j.store == nil {
		return
	}
	j.mu.Lock()
	j.entries[action] = ActionRecord{LastRunAt: time.Now().UTC(), Outcome: outcome, Trigger: trigger}
	snapshot := make(map[string]ActionRecord, len(j.entries))
	for k, v := range j.entries {
		snapshot[k] = v
	}
	err := j.store.Save(snapshot)
	j.mu.Unlock()

	if err != nil {
		slog.ErrorContext(ctx, "adminstate: persistance du journal des actions échouée (état mémoire conservé)",
			"action", action, "err", err)
	}
}

// Entry retourne la dernière exécution connue d'une action (ok=false si jamais).
func (j *ActionJournal) Entry(action string) (ActionRecord, bool) {
	if j == nil {
		return ActionRecord{}, false
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	rec, ok := j.entries[action]
	return rec, ok
}

// Snapshot retourne une copie de toutes les entrées (pour l'endpoint admin).
func (j *ActionJournal) Snapshot() map[string]ActionRecord {
	out := map[string]ActionRecord{}
	if j == nil {
		return out
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	for k, v := range j.entries {
		out[k] = v
	}
	return out
}
