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

// hasConvergenceBacklog indique s'il reste des matchs à converger (enrichment
// manquant, events OU weapons incomplets). Sert à déclencher le post-sync même
// quand aucun nouveau match n'a été inséré : le sync n'a pas "fini" tant que
// tout n'est pas enrichi.
func hasConvergenceBacklog(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) bool {
	return countSharedMatchesMissingEnrichment(ctx, playerDB, sharedDB, xuid) > 0 ||
		len(selectMatchesMissingEvents(ctx, playerDB, sharedDB, xuid)) > 0 ||
		len(selectMatchesMissingWeapons(ctx, playerDB, sharedDB, xuid)) > 0
}

// countSharedMatchesMissingEnrichment compte les matchs présents dans
// shared.match_participants pour ce xuid mais SANS row player_match_enrichment.
//
// Cas couvert (gate 2026-06-10) : cycle « pur skip » — tous les matchs du
// joueur ont été insérés en shared par le watcher d'un coéquipier (delta-skip
// via loadKnownMatchIDs source 2), donc matchesInserted=0 pour ce joueur. Si
// par ailleurs ses scores existants sont complets et events/weapons chargés,
// AUCUN déclencheur ne lançait le pipeline → ensurePlayerEnrichmentRows ne
// tournait jamais → enrichment manquant à durée indéterminée (la convergence
// observée en prod reposait accidentellement sur des scores NULL cold-start
// qui maintenaient needsScoreRefresh=true). Ce compteur rend le déclenchement
// déterministe.
//
// Implémentation en diff Go (2 requêtes) : playerDB et sharedDB sont deux
// connexions distinctes, pas de cross-join SQL possible sans ATTACH.
func countSharedMatchesMissingEnrichment(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) int {
	if playerDB == nil || sharedDB == nil || xuid == "" {
		return 0
	}
	known := make(map[string]struct{}, 512)
	rows, err := playerDB.QueryContext(ctx, `SELECT match_id FROM player_match_enrichment`)
	if err != nil {
		slog.WarnContext(ctx, "convergence: lecture player_match_enrichment échouée", "xuid", xuid, "err", err)
		return 0
	}
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr == nil {
			known[id] = struct{}{}
		}
	}
	_ = rows.Close()

	// Cast défensif xuid || '' aligné sur loadKnownMatchIDs (drift de type
	// UBIGINT vs VARCHAR possible sur un titre futur).
	shared, err := sharedDB.QueryContext(ctx,
		`SELECT DISTINCT match_id FROM match_participants WHERE xuid || '' = ? AND match_id IS NOT NULL`, xuid)
	if err != nil {
		slog.WarnContext(ctx, "convergence: lecture shared.match_participants échouée", "xuid", xuid, "err", err)
		return 0
	}
	defer shared.Close()
	missing := 0
	for shared.Next() {
		var id string
		if scanErr := shared.Scan(&id); scanErr != nil {
			continue
		}
		if _, ok := known[id]; !ok {
			missing++
		}
	}
	if missing > 0 {
		slog.InfoContext(ctx, "convergence: enrichment manquant détecté (matchs insérés par un coéquipier)",
			"xuid", xuid, "missing", missing)
	}
	return missing
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
