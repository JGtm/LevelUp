package duckdb

// checkpoint_held.go : CHECKPOINT périodique des bases que ce processus tient en lecture-écriture.
//
// Pourquoi : entre deux CHECKPOINT, les écritures vivent dans le WAL ; une coupure brutale
// (courant, arrêt forcé) rend ce WAL à la réouverture, et un WAL d'une base qui a déjà eu des
// incidents de rejeu (#7659, ADR 0021) coûte plus qu'un fichier propre. Un poste de bureau est
// coupé sans préavis, ce qu'un serveur hébergé n'est pas : le profil desktop CHECKPOINT donc
// toutes les quelques minutes les bases qu'il tient en RW.
//
// Règles :
//   - seul le processus qui tient le handle RW s'en charge (le cache process-wide de ce paquet),
//     jamais un second processus, jamais une ouverture propre ;
//   - le CHECKPOINT passe sous le bail d'écriture de la base (`dblease`) : un lot d'écriture en
//     cours (persist, sync, bascule B-swap de shared_matches) le tient, la base est alors sautée
//     et comptée, jamais attendue longtemps ;
//   - un CHECKPOINT qui échoue est journalisé (WARN, avec la base) et compté, jamais avalé ;
//   - la liste des bases est figée avant la prise du bail, et un handle RW peut être fermé entre
//     les deux (B-swap : releaseWriter ferme le handle AVANT de rendre le bail) : l'état « fermé »
//     est donc relu APRÈS la prise du bail, et une base fermée est sautée, pas comptée en échec ;
//   - l'arrêt du serveur annule le contexte, donc la requête (duckdb_interrupt) : un CHECKPOINT
//     coupé ainsi est compté à part (`interrupted`), ce n'est pas une panne de la base. DuckDB
//     rend un CHECKPOINT atomique (un CHECKPOINT interrompu laisse la base et son WAL tels qu'ils
//     étaient) et CloseAll checkpointe à la fermeture : rien n'est perdu.

import (
	"context"
	"expvar"
	"log/slog"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/platform/dblease"
)

// checkpointLeaseWait borne l'attente du bail d'écriture d'une base : au-delà, un lot d'écriture
// est en cours et la base est sautée jusqu'au prochain passage.
const checkpointLeaseWait = 2 * time.Second

// checkpointPerDBTimeout borne un CHECKPOINT (un WAL de plusieurs dizaines de Mo prend
// quelques secondes sur un disque lent).
const checkpointPerDBTimeout = 30 * time.Second

// metricPeriodicCheckpoint compte les passages du CHECKPOINT périodique, par issue : `ok`,
// `failed` (CHECKPOINT refusé ou en erreur), `skipped_busy` (bail d'écriture tenu),
// `skipped_closed` (handle fermé entre la liste et le bail), `interrupted` (contexte annulé
// pendant le CHECKPOINT : arrêt du serveur). Exposé sous /debug/vars.
var metricPeriodicCheckpoint = expvar.NewMap("levelup.checkpoint.periodic")

// metricPeriodicCheckpointMs : durée du dernier passage complet (ms).
var metricPeriodicCheckpointMs = expvar.NewFloat("levelup.checkpoint.periodic_last_duration_ms")

// CheckpointReport résume un passage de CheckpointHeld.
type CheckpointReport struct {
	// Done : bases checkpointées.
	Done int
	// Failed : bases dont le CHECKPOINT a échoué (journalisé).
	Failed int
	// Busy : bases sautées parce que leur bail d'écriture était tenu.
	Busy int
	// Closed : bases sautées parce que leur handle a été fermé entre la liste et le bail.
	Closed int
	// Interrupted : CHECKPOINT coupés par l'annulation du contexte (arrêt du serveur).
	Interrupted int
	// Duration : durée du passage entier.
	Duration time.Duration
}

// heldRWHandles rend, triés par chemin, les handles RW ouverts du cache process-wide, hors ceux
// de skip (chemins déjà couverts par un autre CHECKPOINT périodique).
func heldRWHandles(skip map[string]bool) []*DB {
	openDBsMu.Lock()
	defer openDBsMu.Unlock()
	var out []*DB
	for key, cached := range openDBs {
		if !strings.HasPrefix(key, rwCacheKeyPrefix) || cached.db == nil || cached.db.closed.Load() {
			continue
		}
		if skip[cached.db.path] {
			continue
		}
		out = append(out, cached.db)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].path < out[j].path })
	return out
}

// CheckpointHeld exécute un CHECKPOINT sur chaque base tenue en RW par ce processus, sous son
// bail d'écriture. skip : chemins à ne pas traiter. Ne rend jamais d'erreur : chaque échec est
// journalisé et compté, le rapport le dit.
func CheckpointHeld(ctx context.Context, skip map[string]bool) CheckpointReport {
	start := time.Now()
	var report CheckpointReport
	for _, db := range heldRWHandles(skip) {
		if ctx.Err() != nil {
			break
		}
		switch checkpointOne(ctx, db) {
		case checkpointDone:
			report.Done++
		case checkpointFailed:
			report.Failed++
		case checkpointBusy:
			report.Busy++
		case checkpointClosed:
			report.Closed++
		case checkpointInterrupted:
			report.Interrupted++
		}
	}
	report.Duration = time.Since(start)
	metricPeriodicCheckpointMs.Set(float64(report.Duration.Milliseconds()))
	if report.Done+report.Failed+report.Busy+report.Closed+report.Interrupted > 0 {
		slog.InfoContext(ctx, "duckdb: checkpoint périodique",
			"done", report.Done, "failed", report.Failed, "busy", report.Busy,
			"closed", report.Closed, "interrupted", report.Interrupted,
			"duration_ms", report.Duration.Milliseconds())
	}
	return report
}

type checkpointOutcome int

const (
	checkpointDone checkpointOutcome = iota
	checkpointFailed
	checkpointBusy
	checkpointClosed
	checkpointInterrupted
)

// checkpointOne : bail d'écriture (attente bornée), CHECKPOINT, métrique et journal de la base.
func checkpointOne(ctx context.Context, db *DB) checkpointOutcome {
	release, err := dblease.AcquireLease(db.path, checkpointLeaseWait)
	if err != nil {
		metricPeriodicCheckpoint.Add("skipped_busy", 1)
		slog.DebugContext(ctx, "duckdb: checkpoint périodique sauté, écriture en cours", "path", db.path, "err", err)
		return checkpointBusy
	}
	defer release()

	// Relu APRÈS le bail : la liste a été faite avant, et releaseWriter ferme le handle RW avant de
	// rendre le bail. Un CHECKPOINT sur ce handle échouerait en « database is closed ».
	if db.closed.Load() {
		metricPeriodicCheckpoint.Add("skipped_closed", 1)
		slog.DebugContext(ctx, "duckdb: checkpoint périodique sauté, handle fermé entre-temps", "path", db.path)
		return checkpointClosed
	}

	cpCtx, cancel := context.WithTimeout(ctx, checkpointPerDBTimeout)
	defer cancel()
	start := time.Now()
	if _, err := db.loadSQL().ExecContext(cpCtx, "CHECKPOINT"); err != nil {
		if ctx.Err() != nil {
			metricPeriodicCheckpoint.Add("interrupted", 1)
			slog.InfoContext(ctx, "duckdb: checkpoint périodique interrompu par l'arrêt",
				"path", db.path, "duration_ms", time.Since(start).Milliseconds(), "err", err)
			return checkpointInterrupted
		}
		metricPeriodicCheckpoint.Add("failed", 1)
		slog.WarnContext(ctx, "duckdb: checkpoint périodique échoué",
			"path", db.path, "duration_ms", time.Since(start).Milliseconds(), "err", err)
		return checkpointFailed
	}
	metricPeriodicCheckpoint.Add("ok", 1)
	slog.DebugContext(ctx, "duckdb: checkpoint périodique", "path", db.path, "duration_ms", time.Since(start).Milliseconds())
	return checkpointDone
}
