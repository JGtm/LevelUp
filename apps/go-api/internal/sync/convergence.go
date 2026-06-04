// Package sync — convergence.go : convergence du pipeline d'enrichissement.
//
// Principe (2026-06-04) : le sync ne "passe" pas une fois sur un match, il
// CONVERGE — son unité de travail est « ce match est complètement enrichi ».
// Chaque cycle, on dérive le travail du LEDGER (match_registry.events_loaded +
// backfill_completed bits, via le moteur de sélection bitmask-aware existant
// FindMatchesMissingData) plutôt que des seuls matchs fraîchement insérés. Un
// match déjà complet n'est jamais resélectionné (idempotent) ; un match
// incomplet est repris au cycle suivant jusqu'à complétion ou état terminal
// (no-film > 30j → MarkNoFilmDefinitive). Ce N'EST PAS un heal : pas d'ON
// CONFLICT/UPDATE sur shared (INSERT-pur / DELETE-then-INSERT sérialisé par
// lease), c'est intrinsèque au cycle de sync.
//
// Ordre imposé : events AVANT weapons — weapon_kills DÉRIVENT de highlight_events
// (getKillsForPlayer lit highlight_events).

package sync

import (
	"context"
	"database/sql"
	"log/slog"
)

// convergenceHorizon borne le nombre de matchs incomplets repris par cycle
// (les plus récents d'abord — ORDER BY start_time DESC LIMIT). Les téléchargements
// film (weapons) dominent le coût ; un backlog plus grand se résorbe sur
// plusieurs cycles. Les vieux matchs sans film sortent du set via le terminal
// no-film 30j (MarkNoFilmDefinitive), donc le set converge vers 0.
const convergenceHorizon = 50

// selectMatchesMissingEvents retourne les match_ids du joueur dont les
// highlight events ne sont pas chargés (events_loaded=false), bornés et triés
// par récence. Réutilise FindMatchesMissingData (aucun SQL nouveau).
func selectMatchesMissingEvents(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) []string {
	scope := &SyncScope{Events: true, MaxMatches: convergenceHorizon, DetectionMode: "or"}
	scope.Resolve()
	ids, err := FindMatchesMissingData(ctx, playerDB, sharedDB, xuid, scope)
	if err != nil {
		slog.WarnContext(ctx, "convergence: sélection events incomplets échouée", "xuid", xuid, "err", err)
		return nil
	}
	return ids
}

// selectMatchesMissingWeapons : idem pour weapon_kills (bits MBitWeaponKills /
// MBitWeaponKillsNoFilm non posés). À appeler APRÈS convergeEvents.
func selectMatchesMissingWeapons(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) []string {
	scope := &SyncScope{Weapons: true, MaxMatches: convergenceHorizon, DetectionMode: "or"}
	scope.Resolve()
	ids, err := FindMatchesMissingData(ctx, playerDB, sharedDB, xuid, scope)
	if err != nil {
		slog.WarnContext(ctx, "convergence: sélection weapons incomplets échouée", "xuid", xuid, "err", err)
		return nil
	}
	return ids
}

// hasConvergenceBacklog indique s'il reste des matchs à converger (events OU
// weapons incomplets). Sert à déclencher le post-sync même quand aucun nouveau
// match n'a été inséré : le sync n'a pas "fini" tant que tout n'est pas enrichi.
func hasConvergenceBacklog(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) bool {
	return len(selectMatchesMissingEvents(ctx, playerDB, sharedDB, xuid)) > 0 ||
		len(selectMatchesMissingWeapons(ctx, playerDB, sharedDB, xuid)) > 0
}

// convergeEvents re-fetch les highlight events des matchs events_loaded=false
// (matchs insérés par le watcher d'un teammate, ou film pas encore propagé au
// 1er passage). Idempotent : un match events_loaded=true n'est jamais
// resélectionné. ProcessHighlightEvents gère le terminal no-film (>30j →
// MarkNoFilmDefinitive, sort du set). Retourne le nombre de matchs traités.
//
// IMPÉRATIF : router via ProcessHighlightEvents (qui ne touche pas le flag avant
// écriture), JAMAIS via ReplayHighlightEventsForMatches (qui clear events_loaded
// AVANT → combiné à l'INSERT OR IGNORE non-déduplicant en prod, dupliquerait les
// highlight_events). globalDB=nil : l'upsert d'alias xbox est best-effort et
// déjà fait au sync primaire.
func convergeEvents(ctx context.Context, sharedDB *sql.DB, client HaloClient, ids []string) int {
	done := 0
	for _, mid := range ids {
		if ctx.Err() != nil {
			break
		}
		if err := ProcessHighlightEvents(ctx, client, sharedDB, nil, mid, nil); err != nil {
			slog.WarnContext(ctx, "convergence: events échoué", "match_id", mid, "err", err)
			continue
		}
		done++
	}
	return done
}
