// Package scheduler — auto_sync_convergence.go : passe de convergence des highlight
// events (rattrapage best-effort des matchs à events incomplets) + warn des claims gate
// anormalement anciens. Extrait de auto_sync.go (K2c, 2026-07-06) pour scinder le god-file
// (scheduler / engine factory / métriques-convergence).
package scheduler

import (
	"context"
	"log/slog"
)

// eventsConvergenceEnabled : kill-switch (défaut ON). LEVELUP_EVENTS_CONVERGENCE=0
// désactive la passe scheduler ET le trigger immédiat. La valeur est résolue au boot
// dans config.AppConfig (source unique — CR A6).
//
// Cycle de vie du kill-switch :
//   - basculement défaut ON : dès la livraison de la convergence events
//     (idempotente append-only, ADR 0026/0027) ;
//   - date cible de retrait : >= 2026-Q4 ;
//   - critère mesurable : aucun rollback `=0` activé + aucune erreur récurrente
//     « auto_sync: convergence events échouée » sur >= 1 trimestre + res.Processed
//     converge vers 0 (backlog de highlight_events incomplets rattrapé). Alors
//     retirer le flag en gardant la passe active.
func (s *AutoSyncScheduler) eventsConvergenceEnabled() bool {
	return s.cfg.EventsConvergence
}

// convergencePerCycleLimit borne le nombre de matchs traités par la passe de
// convergence events à chaque tick (le reste est repris au tick suivant). Résolu au
// boot depuis LEVELUP_EVENTS_CONVERGENCE_MAX (config.AppConfig).
func (s *AutoSyncScheduler) convergencePerCycleLimit() int {
	return s.cfg.EventsConvergenceMax
}

// runEventsConvergencePass lance une passe bornée de convergence events pour un
// joueur, SOUS le claim déjà tenu par syncPlayer (gate=nil → pas de re-claim).
// Gate sur le pool (tests sans pool → skip) + kill-switch. Best-effort.
func (s *AutoSyncScheduler) runEventsConvergencePass(ctx context.Context, gamertag, xuid string) {
	if s.pool == nil || !s.eventsConvergenceEnabled() {
		return
	}
	engine := s.BuildEngine(ctx, gamertag, xuid)
	res, err := engine.RunEventsConvergence(ctx, nil, s.convergencePerCycleLimit())
	if err != nil {
		slog.WarnContext(ctx, "auto_sync: convergence events échouée", "gamertag", gamertag, "err", err)
		return
	}
	if res.EventsWritten > 0 || res.NoFilmFinal > 0 || res.Skipped > 0 {
		slog.InfoContext(ctx, "auto_sync: convergence events",
			"gamertag", gamertag, "detected", res.Detected,
			"events_written", res.EventsWritten, "no_film_final", res.NoFilmFinal,
			"skipped", res.Skipped, "dominance_updated", res.DominanceUpdated)
	}
}

// TriggerEventsConvergence lance une passe de convergence events IMMÉDIATE pour un
// joueur (ex. juste après un import OpenSpartan), en réutilisant le pool d'auth.
// Cède au sync live par lot (gate=SyncGate). Conçue pour un appel en goroutine
// (ne tient pas de claim externe). No-op si pool absent ou kill-switch off.
func (s *AutoSyncScheduler) TriggerEventsConvergence(ctx context.Context, gamertag, xuid string) {
	if s == nil || s.pool == nil || !s.eventsConvergenceEnabled() {
		return
	}
	engine := s.BuildEngine(ctx, gamertag, xuid)
	res, err := engine.RunEventsConvergence(ctx, s.SyncGate, 0) // tous les incomplets, cède au live
	if err != nil {
		slog.WarnContext(ctx, "trigger convergence events échouée", "gamertag", gamertag, "err", err)
		return
	}
	slog.InfoContext(ctx, "trigger convergence events terminée",
		"gamertag", gamertag, "detected", res.Detected,
		"events_written", res.EventsWritten, "no_film_final", res.NoFilmFinal,
		"skipped", res.Skipped, "ceded", res.Ceded)
}

// warnStaleGateClaims émet un WARN par claim du gate anormalement ancien
// (potentiellement fuité : release jamais appelé → joueur jamais re-synchronisé).
// No-op si aucun gate (NopSyncGate renvoie un cliché vide).
func (s *AutoSyncScheduler) warnStaleGateClaims(ctx context.Context) {
	if s.SyncGate == nil {
		return
	}
	for _, cl := range s.SyncGate.GateSnapshot().Claims {
		if cl.Stale {
			slog.WarnContext(ctx, "sync_gate: claim potentiellement fuité (tenu anormalement longtemps)",
				"gamertag", cl.Gamertag, "source", cl.Source, "age_ms", cl.AgeMs)
		}
	}
}
