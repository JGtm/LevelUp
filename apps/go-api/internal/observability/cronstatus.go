// Package observability — cronstatus.go : registre central du statut des crons
// (plan monitoring A6, DC-5) + heartbeats de features.
//
// Chaque cron rapporte ses exécutions via ReportCronRun (nom, début, erreur,
// durée) : le registre mémorise last_run / last_success / last_error /
// consecutive_failures, et relaie vers un Sink optionnel (câblé au boot vers la
// persistance cron_runs de la base monitoring — le registre reste utilisable
// sans store, ex. tests).
//
// Heartbeats (DC-5, liste fermée côté monitoring) : timestamp expvar
// heartbeat_{feature} posé au PASSAGE RÉEL dans le code (le cas « hook câblé
// mais jamais invoqué » devient visible).
package observability

import (
	"sort"
	"sync"
	"time"
)

// CronStatusRecord est l'état courant d'un cron depuis le boot.
type CronStatusRecord struct {
	Name                string
	LastRunAt           time.Time
	LastSuccessAt       time.Time
	LastError           string
	ConsecutiveFailures int
	Runs                int64
	LastDurationMs      int64
}

// CronRunSink relaie chaque exécution vers une persistance (best-effort).
type CronRunSink func(name string, startedAt time.Time, ok bool, errStr string, durationMs int64)

type cronRegistry struct {
	mu      sync.Mutex
	records map[string]*CronStatusRecord
	sink    CronRunSink
}

var cronReg = &cronRegistry{records: map[string]*CronStatusRecord{}}

// SetCronRunSink câble la persistance des exécutions (une fois, au boot).
func SetCronRunSink(sink CronRunSink) {
	cronReg.mu.Lock()
	cronReg.sink = sink
	cronReg.mu.Unlock()
}

// ReportCronRun enregistre une exécution de cron (err nil = succès).
func ReportCronRun(name string, startedAt time.Time, err error, durationMs int64) {
	cronReg.mu.Lock()
	rec, ok := cronReg.records[name]
	if !ok {
		rec = &CronStatusRecord{Name: name}
		cronReg.records[name] = rec
	}
	rec.Runs++
	rec.LastRunAt = startedAt
	rec.LastDurationMs = durationMs
	if err != nil {
		rec.LastError = err.Error()
		rec.ConsecutiveFailures++
	} else {
		rec.LastError = ""
		rec.ConsecutiveFailures = 0
		rec.LastSuccessAt = startedAt
	}
	sink := cronReg.sink
	cronReg.mu.Unlock()

	if sink != nil {
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		sink(name, startedAt, err == nil, errStr, durationMs)
	}
}

// CronStatusSnapshot retourne l'état courant trié par nom (copie).
func CronStatusSnapshot() []CronStatusRecord {
	cronReg.mu.Lock()
	out := make([]CronStatusRecord, 0, len(cronReg.records))
	for _, r := range cronReg.records {
		out = append(out, *r)
	}
	cronReg.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ResetCronStatus vide le registre (tests).
func ResetCronStatus() {
	cronReg.mu.Lock()
	cronReg.records = map[string]*CronStatusRecord{}
	cronReg.sink = nil
	cronReg.mu.Unlock()
}

// ─── Heartbeats de features (DC-5) ─────────────────────────────────────────

// Heartbeat pose le timestamp « vu vivant » d'une feature (expvar
// heartbeat_{feature}, unix seconds). À appeler au passage RÉEL dans le code.
func Heartbeat(feature string) {
	SetInt("heartbeat_"+feature, time.Now().Unix())
}

// HeartbeatUnix lit le dernier heartbeat d'une feature (0 = jamais vu).
func HeartbeatUnix(feature string) int64 {
	return LoadCounter("heartbeat_" + feature)
}
