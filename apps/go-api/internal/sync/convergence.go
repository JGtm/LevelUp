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

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
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
// manquant, PSA non tentés, events OU weapons incomplets). Sert à déclencher le
// post-sync même quand aucun nouveau match n'a été inséré : le sync n'a pas
// "fini" tant que tout n'est pas enrichi.
func hasConvergenceBacklog(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) bool {
	return countSharedMatchesMissingEnrichment(ctx, playerDB, sharedDB, xuid) > 0 ||
		len(selectMatchesMissingPSA(ctx, playerDB)) > 0 ||
		len(selectMatchesMissingEvents(ctx, playerDB, sharedDB, xuid)) > 0 ||
		len(selectMatchesMissingWeapons(ctx, playerDB, sharedDB, xuid)) > 0
}

// selectMatchesMissingPSA retourne les matchs enrichis dont les
// personal_score_awards n'ont JAMAIS été tentés (psa_checked_at IS NULL) et
// qui n'ont aucune row PSA. Cas nominal : matchs delta-skippés (insérés en
// shared par un coéquipier) — seul le traitement per-match écrivait les PSA,
// confirmé par le gate invariants (psa_missing, 2026-06-10).
//
// Le marqueur terminal psa_checked_at garantit qu'un match sans PersonalScores
// extractibles (NameIds inconnus, vieux matchs) n'est tenté qu'UNE fois.
// Borné par convergenceHorizon comme les autres sélections.
func selectMatchesMissingPSA(ctx context.Context, playerDB *sql.DB) []string {
	// match_id IS NOT NULL dans le sous-select : défense contre les player DB
	// legacy sans contraintes (un NULL rendrait le NOT IN faux pour TOUTES les
	// rows → convergence silencieusement désactivée). ORDER BY created_at DESC :
	// les matchs récents d'abord, aligné sur le contrat convergenceHorizon.
	rows, err := playerDB.QueryContext(ctx, `
		SELECT e.match_id
		-- Append-only #23046 : _latest. psa_checked_at vit sur le stage 'psa' ; sur la
		-- table brute les rows des autres stages ont psa_checked_at NULL → ce filtre
		-- retournerait TOUS les matchs même déjà checkés → re-fetch PSA infini.
		FROM player_match_enrichment_latest e
		WHERE e.psa_checked_at IS NULL
		  AND e.match_id NOT IN (
		      SELECT DISTINCT match_id FROM personal_score_awards WHERE match_id IS NOT NULL)
		ORDER BY e.created_at DESC NULLS LAST
		LIMIT ?`, convergenceHorizon)
	if err != nil {
		slog.WarnContext(ctx, "convergence: sélection PSA manquants échouée", "err", err)
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr != nil {
			slog.WarnContext(ctx, "convergence: scan PSA manquant échoué", "err", scanErr)
			continue
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "convergence: itération PSA manquants interrompue", "err", err)
	}
	return out
}

// convergePSA re-fetch le JSON match des matchs sans PSA et extrait les
// PersonalScores du joueur (ExtractPersonalScoreAwards + InsertPersonalScoreAwards,
// idempotent DELETE+INSERT). Stampe psa_checked_at dans TOUS les cas où le
// fetch a réussi (même 0 award) — état terminal. En cas d'échec fetch, pas de
// stamp : retenté au cycle suivant.
//
// Convergence OPPORTUNISTE des alias (2026-06-10) : le JSON fetché contient le
// gamertag de TOUS les participants — on upserte shared.xuid_aliases au
// passage, à coût API nul. Résorbe le backlog d'alias des vieux matchs
// (113 xuids absents au moment de l'audit). Depuis 2026-06-19, shared.xuid_aliases
// est l'unique store (le store global xbox_aliases a été consolidé puis supprimé) :
// ce chemin convergent en est l'alimentation principale, avec
// match_participants/killer_victim_pairs (lus par v_gamertag_lookup).
func convergePSA(ctx context.Context, playerDB, sharedDB *sql.DB, client HaloClient, xuid string, ids []string) int {
	done := 0
	aliases := 0
	for _, mid := range ids {
		if ctx.Err() != nil {
			break
		}
		matchJSON, err := client.GetMatchStats(ctx, mid)
		if err != nil {
			slog.WarnContext(ctx, "convergence: PSA fetch échoué", "match_id", mid, "err", err)
			continue
		}
		aliases += upsertAliasesFromMatchJSON(ctx, sharedDB, matchJSON)
		awards := ExtractPersonalScoreAwards(matchJSON, mid, xuid)
		if len(awards) > 0 {
			if err := InsertPersonalScoreAwards(ctx, playerDB, mid, xuid, awards); err != nil {
				slog.WarnContext(ctx, "convergence: PSA insert échoué", "match_id", mid, "err", err)
				continue
			}
		}
		// Append-only #23046 : INSERT pur stage='psa' (marqueur terminal). La vue
		// merge expose psa_checked_at par match ; selectMatchesMissingPSA lit _latest.
		if _, err := playerDB.ExecContext(ctx,
			`INSERT INTO player_match_enrichment (match_id, psa_checked_at, stage) VALUES (?, now(), 'psa')`, mid); err != nil {
			slog.WarnContext(ctx, "convergence: PSA stamp échoué", "match_id", mid, "err", err)
			continue
		}
		done++
	}
	if aliases > 0 {
		// Cumul boot-level pour le dashboard monitoring (rattrapage convergence).
		observability.AddIntT(ctxkeys.TitleSlug(ctx), "convergence_aliases_upserted_total", int64(aliases))
		slog.InfoContext(ctx, "convergence: alias upsertés depuis les JSONs PSA",
			"xuid", xuid, "aliases", aliases)
	}
	return done
}

// upsertAliasesFromMatchJSON upserte un alias xuid→gamertag dans
// shared.xuid_aliases pour chaque participant du JSON match. Best-effort
// (sharedDB nil ou erreurs individuelles ignorées — UpsertXUIDAlias normalise
// déjà les bots). Retourne le nombre d'upserts réussis.
func upsertAliasesFromMatchJSON(ctx context.Context, sharedDB *sql.DB, matchJSON map[string]any) int {
	if sharedDB == nil {
		return 0
	}
	players, _ := matchJSON["Players"].([]any)
	n := 0
	for _, p := range players {
		player, ok := p.(map[string]any)
		if !ok {
			continue
		}
		pid := extractXUID(asString(player["PlayerId"]))
		gt := asString(player["Gamertag"])
		if gt == "" {
			gt = asString(player["PlayerName"])
		}
		if pid == "" || gt == "" {
			continue
		}
		if err := UpsertXUIDAlias(ctx, sharedDB, pid, gt); err == nil {
			n++
		}
	}
	return n
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
	rows, err := playerDB.QueryContext(ctx, `SELECT match_id FROM player_match_enrichment_latest`)
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
	if iterErr := rows.Err(); iterErr != nil {
		// Itération tronquée = known partiel = sur-déclenchement possible du
		// pipeline (bénin, le heal est idempotent) — mais on le trace.
		slog.WarnContext(ctx, "convergence: itération enrichment interrompue", "xuid", xuid, "err", iterErr)
	}
	_ = rows.Close()

	// Cast défensif xuid || '' aligné sur loadKnownMatchIDs ET sur
	// ensurePlayerEnrichmentRows (le repareur) — même prédicat partout, sinon
	// un drift de type ferait diverger déclencheur et réparateur (re-trigger
	// infini sans convergence).
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
	if iterErr := shared.Err(); iterErr != nil {
		slog.WarnContext(ctx, "convergence: itération participants interrompue", "xuid", xuid, "err", iterErr)
	}
	if missing > 0 {
		slog.InfoContext(ctx, "convergence: enrichment manquant détecté (matchs insérés par un coéquipier)",
			"xuid", xuid, "missing", missing)
	}
	return missing
}

// CountSharedMatchesMissingEnrichment est le wrapper EXPORTÉ de
// countSharedMatchesMissingEnrichment, pour les pipelines de sync DÉDIÉS hors du
// package sync (runner live Halo 5 — internal/games/halo_5/livesync). Même
// sémantique title-agnostic : nombre de matchs présents en shared.match_participants
// pour ce xuid mais SANS row player_match_enrichment.
//
// Sert de garde bon-marché « backlog d'enrichment à 0 insert » : un match du titre
// inséré en shared par le sync d'un coéquipier (delta-skip chez ce joueur) n'est
// jamais enrichi tant qu'aucun cycle ne déclenche le reconcile full-shared
// (BackfillEnrichmentFromShared). > 0 → le runner doit relancer son hook post-score
// même à matchesInserted=0 (mirroir de hasConvergenceBacklog côté Infinite).
func CountSharedMatchesMissingEnrichment(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) int {
	return countSharedMatchesMissingEnrichment(ctx, playerDB, sharedDB, xuid)
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
// highlight_events).
func convergeEvents(ctx context.Context, sharedDB *sql.DB, client HaloClient, ids []string) int {
	done := 0
	for _, mid := range ids {
		if ctx.Err() != nil {
			break
		}
		if err := ProcessHighlightEvents(ctx, client, sharedDB, mid, nil); err != nil {
			slog.WarnContext(ctx, "convergence: events échoué", "match_id", mid, "err", err)
			continue
		}
		done++
	}
	return done
}
