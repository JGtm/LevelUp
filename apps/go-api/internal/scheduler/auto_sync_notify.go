// Package scheduler — auto_sync_notify.go : notification Discord de fin de cycle
// auto-sync (« nouveaux matchs synchronisés »).
//
// CONTEXTE (signalement 2026-07-25) : aucune notif Discord de sync de nouveaux
// matchs n'arrivait depuis le VPS. Cause : le chemin auto-sync (scheduler,
// in-process) n'appelait JAMAIS notify.NotifySync — seul le CLI `levelup
// notify-sync` le faisait. Le toggle discord_notify_sync existe pourtant (défaut
// ON) et est exposé dans les réglages. On honore donc ce toggle en émettant UNE
// notification par cycle périodique quand au moins un joueur a de nouveaux matchs.
//
// Pourquoi ce chemin plutôt que le relais des catégories in-app
// (notifications/external) : la catégorie « nouveaux matchs » (match_synced) n'est
// (a) pas émise par l'auto-sync (seul le handler HTTP l'émet) et (b) volontairement
// hors DefaultForwardedCategories (relais coach uniquement, verrouillé par le
// garde-rail TestDefaultForwardedCategories_MirrorsCoach). Le relais est de plus
// gaté par discord_notify_coach (décision vie privée). notify.NotifySync est le
// mécanisme dédié « fin de sync », déjà testé, i18n FR/EN, sans changement de
// garde-rail — voie propre et à surface minimale.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/notify"
	"levelup/go-api/internal/platform/adminstate"
)

// notifyDiscordSyncCycle envoie UNE notification Discord listant les joueurs ayant
// reçu de nouveaux matchs lors de ce cycle auto-sync. Best-effort strict :
// LoadNotifyConfig + NotifySync sont failsafe (no-op si webhook absent ou toggle
// off), jamais bloquant ni paniquant.
//
// Gaté sur le déclencheur PÉRIODIQUE (tick) uniquement : les cycles manuels
// (bouton admin / diag) ne pinguent pas Discord — évite le bruit pendant les tests
// et cadre avec le besoin « notif de sync depuis le VPS » (sync de fond).
//
// N'émet RIEN si aucun nouveau match ce cycle (anti-bruit : pas de « déjà à jour »
// répété à chaque tick).
func (s *AutoSyncScheduler) notifyDiscordSyncCycle(ctx context.Context, trigger string, cycleStart time.Time, res *RunOnceResult) {
	if trigger != adminstate.TriggerCron {
		return // seul le cycle périodique notifie (pas les runs manuels/diag)
	}
	cfg := notify.LoadNotifyConfig(s.cfg.AppSettingsPath)
	if cfg.WebhookURL == "" || !cfg.NotifySync {
		return // webhook absent ou toggle discord_notify_sync OFF — no-op silencieux
	}
	players := s.collectCycleNewMatchPlayers(cycleStart, res)
	total := 0
	for _, p := range players {
		total += p.MatchesSynced
	}
	if total == 0 {
		return // aucun nouveau match ce cycle
	}
	slog.DebugContext(ctx, "auto_sync: notification Discord de fin de cycle",
		"players", len(players), "total_matches", total)
	// success=true (la notif ne part que sur des insertions réussies) ; skipIdle=false
	// (players ne contient QUE des joueurs avec de nouveaux matchs, jamais idle).
	notify.NotifySync(cfg, "sync_delta", cycleStart, time.Now(), players, true, false)
}

// collectCycleNewMatchPlayers lit le Snapshot (joueurs live/filet) et délègue la
// fusion à mergeCycleNewMatchPlayers (pure, testable sans Snapshot).
func (s *AutoSyncScheduler) collectCycleNewMatchPlayers(cycleStart time.Time, res *RunOnceResult) []notify.PlayerSyncResult {
	return mergeCycleNewMatchPlayers(cycleStart, res.newMatchPlayers, s.Snapshot().Players)
}

// mergeCycleNewMatchPlayers fusionne, dédupliqués par gamertag, les joueurs ayant
// inséré >= 1 nouveau match ce cycle depuis les deux chemins du cycle :
//   - engine : joueurs moteur (Infinite) du pipeline V2, comptés dans
//     CycleResult.PerPlayer (non enregistrés dans playerOutcomes) ;
//   - snapshot : joueurs live-only (Halo 5) + filet de boot, passés par
//     syncPlayer/recordOutcome, filtrés sur CE cycle (AttemptedAt >= cycleStart).
//
// Pure (aucune I/O) → testable directement.
func mergeCycleNewMatchPlayers(cycleStart time.Time, engine []notify.PlayerSyncResult, snapshot []PlayerOutcomeDetail) []notify.PlayerSyncResult {
	seen := make(map[string]struct{})
	var out []notify.PlayerSyncResult
	for _, p := range engine {
		if p.MatchesSynced <= 0 {
			continue
		}
		if _, dup := seen[p.Gamertag]; dup {
			continue
		}
		seen[p.Gamertag] = struct{}{}
		out = append(out, p)
	}
	for _, d := range snapshot {
		if d.Outcome != "ok" || d.MatchesInserted <= 0 || d.AttemptedAt.Before(cycleStart) {
			continue
		}
		if _, dup := seen[d.Gamertag]; dup {
			continue
		}
		seen[d.Gamertag] = struct{}{}
		out = append(out, notify.PlayerSyncResult{Gamertag: d.Gamertag, MatchesSynced: d.MatchesInserted})
	}
	return out
}
