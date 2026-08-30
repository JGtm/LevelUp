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
//
// Ce fichier porte AUSSI les deux orchestrateurs d'étape du post-sync
// (postSyncFilmSteps, en fin de fichier) : sélection → COLLECT → FLUSH. Taille
// > 500 L assumée (exemption CLAUDE.md §Seuils) : le ratchet
// archlint.TestSyncRootPackageFrozen interdit tout NOUVEAU fichier à la racine
// de internal/sync/, et ces orchestrateurs n'ont de sens qu'à côté des phases
// qu'ils pilotent. Prochaine extraction du cluster convergence → sous-package
// dédié (ADR 0027), ce fichier partira en bloc.

package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/killcollector"
	"levelup/go-api/internal/sync/replayartifacts"
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

// filterEventsStillMissing re-vérifie, SOUS le burst writer (sérialisé par
// dblease), que chaque match du lot a toujours besoin d'une convergence events
// (events_loaded=false et events_empty=false). Ferme la fenêtre TOCTOU ouverte
// par l'étape 1 contention : les post-syncs de joueurs tournent désormais en
// parallèle SANS être sérialisés par un lease global — la work-list du joueur B
// (sélectionnée en RO) peut contenir un match partagé que le joueur A vient de
// converger. Sans ce re-check, B dupliquerait les highlight_events (INSERT OR
// IGNORE = no-op de dédup en prod, PK auto-seq).
//
// Depuis le split COLLECT/FLUSH (v7.3) la fenêtre à couvrir est PLUS LARGE :
// elle va de la sélection RO jusqu'au flush, en incluant le téléchargement du
// film (hors lease). Ce re-check reste donc l'unique garde anti-duplication et
// DOIT être appelé sous le writer, juste avant d'écrire.
// Weapons n'a pas besoin de l'équivalent : écriture par génération auto-réparante
// (v_weapon_kills ne lit que la génération MAX).
func filterEventsStillMissing(ctx context.Context, sharedDB *sql.DB, ids []string) []string {
	if len(ids) == 0 || sharedDB == nil {
		return ids
	}
	cond := "COALESCE(events_loaded, FALSE) = FALSE"
	if hasEventsEmptyColumn(ctx, sharedDB) {
		cond += " AND COALESCE(events_empty, FALSE) = FALSE"
	}
	out := make([]string, 0, len(ids))
	for _, mid := range ids {
		var still bool
		err := sharedDB.QueryRowContext(ctx,
			"SELECT "+cond+" FROM match_registry WHERE match_id = ?", mid).Scan(&still)
		if err != nil {
			// Best-effort : match absent du registry ou probe indisponible →
			// laisser passer (ProcessHighlightEvents gère les cas terminaux).
			out = append(out, mid)
			continue
		}
		if still {
			out = append(out, mid)
		}
	}
	return out
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
	done, pairs := convergePSACollect(ctx, playerDB, client, xuid, ids)
	flushAliasPairs(ctx, sharedDB, xuid, pairs)
	return done
}

// aliasPair : upsert shared.xuid_aliases différé (étape 1 contention — le fetch
// réseau PSA ne tient plus le writer shared ; les alias sont bufferisés puis
// flushés en burst court par le caller).
type aliasPair struct{ xuid, gamertag string }

// convergePSACollect : phase fetch + écritures PLAYER de la convergence PSA.
// AUCUNE écriture shared — les alias extraits des JSONs sont RETOURNÉS pour un
// flush en burst par le caller. Sémantique par match inchangée (best-effort,
// stamp terminal psa_checked_at côté player).
func convergePSACollect(ctx context.Context, playerDB *sql.DB, client HaloClient, xuid string, ids []string) (int, []aliasPair) {
	done := 0
	var pairs []aliasPair
	for _, mid := range ids {
		if ctx.Err() != nil {
			break
		}
		matchJSON, err := client.GetMatchStats(ctx, mid)
		if err != nil {
			slog.WarnContext(ctx, "convergence: PSA fetch échoué", "match_id", mid, "err", err)
			continue
		}
		pairs = append(pairs, extractAliasPairsFromMatchJSON(matchJSON)...)
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
	return done, pairs
}

// flushAliasPairs upserte les alias bufferisés dans shared.xuid_aliases (burst
// court, SQL UpsertXUIDAlias INCHANGÉ). Best-effort — même sémantique que
// l'ancien upsert inline (erreurs individuelles ignorées, sharedDB nil = no-op).
func flushAliasPairs(ctx context.Context, sharedDB *sql.DB, xuid string, pairs []aliasPair) {
	if sharedDB == nil || len(pairs) == 0 {
		return
	}
	aliases := 0
	for _, p := range pairs {
		if err := UpsertXUIDAlias(ctx, sharedDB, p.xuid, p.gamertag); err == nil {
			aliases++
		}
	}
	if aliases > 0 {
		// Cumul boot-level pour le dashboard monitoring (rattrapage convergence).
		observability.AddIntT(ctxkeys.TitleSlug(ctx), "convergence_aliases_upserted_total", int64(aliases))
		slog.InfoContext(ctx, "convergence: alias upsertés depuis les JSONs PSA",
			"xuid", xuid, "aliases", aliases)
	}
}

// upsertAliasesFromMatchJSON upserte un alias xuid→gamertag dans
// shared.xuid_aliases pour chaque participant du JSON match. Best-effort
// (sharedDB nil ou erreurs individuelles ignorées — UpsertXUIDAlias normalise
// déjà les bots). Retourne le nombre d'upserts réussis.
func upsertAliasesFromMatchJSON(ctx context.Context, sharedDB *sql.DB, matchJSON map[string]any) int {
	if sharedDB == nil {
		return 0
	}
	n := 0
	for _, p := range extractAliasPairsFromMatchJSON(matchJSON) {
		if err := UpsertXUIDAlias(ctx, sharedDB, p.xuid, p.gamertag); err == nil {
			n++
		}
	}
	return n
}

// extractAliasPairsFromMatchJSON extrait les paires (xuid, gamertag) des
// participants d'un JSON match — parsing PUR, aucune écriture (étape 1
// contention : la phase collect n'a pas besoin du writer shared).
func extractAliasPairsFromMatchJSON(matchJSON map[string]any) []aliasPair {
	players, _ := matchJSON["Players"].([]any)
	var out []aliasPair
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
		out = append(out, aliasPair{xuid: pid, gamertag: gt})
	}
	return out
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

// convergeEventsCollect — phase COLLECT de la convergence events : re-fetch et
// parse les highlight events des matchs events_loaded=false (matchs insérés par
// le watcher d'un teammate, ou film pas encore propagé au 1er passage), HORS de
// tout lease shared. Aucune écriture, aucune lecture DB.
//
// Best-effort par match, sémantique identique à l'ancienne boucle inline : un
// fetch ou un parse KO est loggé et le match sort simplement du lot (il reste
// events_loaded=false → repris au cycle suivant). Un échec ne jette JAMAIS le
// reste du lot.
//
// IMPÉRATIF : ce chemin ne touche pas au flag events_loaded avant écriture —
// JAMAIS de routage via ReplayHighlightEventsForMatches (qui clear events_loaded
// AVANT → combiné à l'INSERT OR IGNORE non-déduplicant en prod, dupliquerait les
// highlight_events).
func convergeEventsCollect(ctx context.Context, client HaloClient, ids []string) []collectedHighlightEvents {
	out := make([]collectedHighlightEvents, 0, len(ids))
	for _, mid := range ids {
		if ctx.Err() != nil {
			break
		}
		collected, err := collectHighlightEvents(ctx, client, mid)
		if err != nil {
			slog.WarnContext(ctx, "convergence: events échoué", "match_id", mid, "err", err)
			continue
		}
		out = append(out, collected)
	}
	return out
}

// convergeEventsFlush — phase FLUSH de la convergence events : burst writer
// COURT, écritures shared seules. Re-filtre le lot SOUS le lease
// (filterEventsStillMissing, anti-TOCTOU multi-joueurs : un post-sync parallèle
// a pu converger ces matchs pendant notre téléchargement) puis persiste.
// Idempotent : un match events_loaded=true n'est jamais réécrit.
// flushHighlightEvents gère le terminal no-film (>30j → MarkNoFilmDefinitive,
// sort du set). Retourne le nombre de matchs flushés sans erreur.
func convergeEventsFlush(ctx context.Context, sharedDB *sql.DB, collected []collectedHighlightEvents) int {
	if len(collected) == 0 {
		return 0
	}
	ids := make([]string, 0, len(collected))
	for _, c := range collected {
		ids = append(ids, c.matchID)
	}
	still := make(map[string]bool, len(ids))
	for _, mid := range filterEventsStillMissing(ctx, sharedDB, ids) {
		still[mid] = true
	}
	done := 0
	for _, c := range collected {
		if ctx.Err() != nil {
			break
		}
		if !still[c.matchID] {
			continue // convergé par un post-sync parallèle entre le collect et ce flush
		}
		if err := flushHighlightEvents(ctx, sharedDB, c, nil); err != nil {
			slog.WarnContext(ctx, "convergence: events échoué", "match_id", c.matchID, "err", err)
			continue
		}
		done++
	}
	return done
}

// ─────────────────────────────────────────────────────────────────────────────
// Orchestration post-sync des deux étapes « film » (1.54 events, 1.55 weapons)
// ─────────────────────────────────────────────────────────────────────────────
//
// PROBLÈME RÉSOLU (v7.3) : ces deux étapes téléchargeaient et parsaient le film
// SOUS le burst d'écriture shared. Le writer RW était donc tenu pendant tout le
// réseau (~300-500 ms/match events, plusieurs secondes/match weapons) → les
// « writer RW tenu > 2 s » observés 3-5 fois par cycle d'auto-sync en prod, avec
// les lectures HTTP gatées pendant ce temps. Le chunking bornait la fenêtre mais
// ne changeait pas sa NATURE : elle restait proportionnelle au réseau.
//
// PATRON CIBLE (identique au split 4b de la convergence PSA) :
//
//	SELECT   → segment de LECTURE court (work-list)
//	COLLECT  → fetch réseau + parse + lignes prêtes, AUCUN lease shared tenu
//	           (les lectures shared nécessaires passent par un segment Read)
//	FLUSH    → burst writer COURT : écriture des lignes collectées, rien d'autre
//
// La fenêtre RW ne dépend donc plus que du coût SQL du flush. Le chunking est
// conservé à l'identique (3 events / 2 weapons) — il borne désormais la MÉMOIRE
// (nombre de films collectés avant flush), plus la durée du lease.
//
// Invariants préservés :
//   - anti-TOCTOU events : filterEventsStillMissing reste appelé SOUS le writer,
//     juste avant l'écriture (cf. convergeEventsFlush) ;
//   - releases sous defer : le flush garde son `defer releaseW()` ;
//   - labels de télémétrie inchangés : sync_v2_postsync/{events,weapons} ;
//   - sémantique d'erreur par match : un fetch KO ne jette pas le lot.

// postSyncFilmSteps regroupe les dépendances communes aux étapes film du
// post-sync (struct plutôt que 6 paramètres à plat — seuil projet : 5).
type postSyncFilmSteps struct {
	engine   *SyncEngine
	playerDB *sql.DB
	shared   *SharedAccess
	client   HaloClient
	result   *domain.PostSyncResult
}

// withRead ouvre un segment de LECTURE shared court (délègue au helper canonique
// du pipeline — même WARN + trackFatalErr, même dégradation best-effort).
func (s postSyncFilmSteps) withRead(ctx context.Context, step string, fn func(sharedDB *sql.DB)) {
	s.engine.withSharedRead(ctx, s.shared, s.result, step, fn)
}

// runEventsConvergence — étape 1.54 : convergence des highlight_events des matchs
// events_loaded=false (matchs insérés par le watcher d'un teammate, ou film pas
// encore propagé au 1er passage). Le sync primaire ne charge que les matchs
// NOUVEAUX ; ce backlog ne se résorbait jamais depuis la décommission du heal
// (2026-06-01). Idempotent (un match events_loaded=true n'est pas resélectionné)
// + terminal no-film 30j. DOIT précéder weapon kills (qui dérivent de
// highlight_events).
func (s postSyncFilmSteps) runEventsConvergence(ctx context.Context) {
	e := s.engine
	var eventsWork []string
	s.withRead(ctx, "events_select", func(sharedDB *sql.DB) {
		eventsWork = selectMatchesMissingEvents(ctx, s.playerDB, sharedDB, e.xuid)
	})
	// Jauge "roue de secours" : en régime stationnaire ces compteurs doivent
	// PLAFONNER (convergence = filet exceptionnel). S'ils croissent en continu,
	// c'est que le 1er passage laisse des trous récurrents → durcir l'ingestion.
	// Lisibles sur /debug/vars (expvar "levelup").
	observability.AddIntT(ctxkeys.TitleSlug(ctx), "convergence_events_pending_total", int64(len(eventsWork)))
	if len(eventsWork) == 0 {
		return
	}
	total := 0
	for start := 0; start < len(eventsWork); start += postsyncEventsBurstChunk {
		end := min(start+postsyncEventsBurstChunk, len(eventsWork))
		// COLLECT : fetch + parse HORS de tout lease shared.
		collected := convergeEventsCollect(ctx, s.client, eventsWork[start:end])
		if len(collected) == 0 {
			continue // tout le lot a échoué au fetch → rien à écrire, pas de burst
		}
		// FLUSH : burst writer court, écritures seules.
		wdb, releaseW, werr := s.shared.Write(ctx, "events")
		if werr != nil {
			slog.WarnContext(ctx, "post-sync: burst events indisponible — reste du backlog reporté",
				"gamertag", e.gamertag, "remaining", len(eventsWork)-start, "err", werr)
			trackFatalErr(s.result, "events burst", werr)
			break
		}
		// Corps du flush sous closure : releaseW en defer, donc la fenêtre RW est
		// rendue même si l'écriture panique. Moment nominal de libération
		// inchangé — fin de l'itération, avant le lot suivant.
		func() {
			defer releaseW()
			total += convergeEventsFlush(ctx, wdb, collected)
		}()
	}
	s.result.ConvergedEvents = total
	observability.AddIntT(ctxkeys.TitleSlug(ctx), "convergence_events_processed_total", int64(total))
	slog.InfoContext(ctx, "post-sync: convergence events",
		"gamertag", e.gamertag, "selected", len(eventsWork), "processed", total)
}

// runWeaponKills — étape 1.55 : pipeline film weapon kills. Convergent : nouveaux
// matchs (insertedIDs) ∪ backlog incomplet (bits weapon non posés), bornés. La
// sélection weapons se fait APRÈS la convergence events pour que highlight_events
// soit peuplé. Best-effort : films absents (404/410) normaux pour les vieux
// matchs. Garde bit-honnête préservée (MBitWeaponKills posé seulement si ≥1 ligne
// insérée, cf. flushWeaponKillsForMatch).
func (s postSyncFilmSteps) runWeaponKills(ctx context.Context, insertedIDs []string) {
	e := s.engine
	var weaponBacklog []string
	s.withRead(ctx, "weapons_select", func(sharedDB *sql.DB) {
		weaponBacklog = selectMatchesMissingWeapons(ctx, s.playerDB, sharedDB, e.xuid)
	})
	observability.AddIntT(ctxkeys.TitleSlug(ctx), "convergence_weapons_pending_total", int64(len(weaponBacklog)))
	weaponWork := mergeUniqMatchIDs(insertedIDs, weaponBacklog)
	if len(weaponWork) == 0 {
		return
	}
	totalDone, totalNoFilm := 0, 0
	for start := 0; start < len(weaponWork); start += postsyncWeaponsBurstChunk {
		end := min(start+postsyncWeaponsBurstChunk, len(weaponWork))
		// COLLECT : download film + corrélation. Les lectures shared dont dépend
		// la corrélation (highlight_events, match_participants) passent par un
		// segment de LECTURE — jamais par le writer. Le segment est relâché AVANT
		// le burst (garde anti-deadlock de SharedAccess.Write).
		var collected []collectedWeaponKills
		readOK := false
		s.withRead(ctx, "weapons_collect", func(roDB *sql.DB) {
			readOK = true
			collected = collectWeaponKillsChunk(ctx, roDB, s.client, e.xuid, weaponWork[start:end])
		})
		if !readOK {
			// Lecture shared indisponible (déjà loggée + trackFatalErr par withRead) :
			// inutile de réessayer lot après lot, on reporte le reste du backlog.
			break
		}
		if len(collected) == 0 {
			continue // tout le lot a échoué au fetch film → rien à écrire, pas de burst
		}
		// FLUSH : burst writer court, écritures seules.
		wdb, releaseW, werr := s.shared.Write(ctx, "weapons")
		if werr != nil {
			slog.WarnContext(ctx, "post-sync: burst weapons indisponible — reste du backlog reporté",
				"gamertag", e.gamertag, "remaining", len(weaponWork)-start, "err", werr)
			trackFatalErr(s.result, "weapons burst", werr)
			break
		}
		var done, noFilm int
		func() {
			defer releaseW()
			done, noFilm = flushWeaponKillsChunk(ctx, wdb, collected)
		}()
		totalDone += done
		totalNoFilm += noFilm
		if ctx.Err() != nil {
			// Annulation en cours de cycle : on arrête proprement (le reste du
			// backlog est repris au cycle suivant — étapes idempotentes).
			slog.WarnContext(ctx, "post-sync: weapon kills interrompu", "gamertag", e.gamertag, "err", ctx.Err())
			trackFatalErr(s.result, "weapon kills", ctx.Err())
			break
		}
	}
	s.result.WeaponKillsProcessed = totalDone
	s.result.WeaponKillsNoFilm = totalNoFilm
	observability.AddIntT(ctxkeys.TitleSlug(ctx), "convergence_weapons_processed_total", int64(totalDone))
	if totalDone > 0 || totalNoFilm > 0 {
		slog.InfoContext(ctx, "post-sync: weapon kills",
			"gamertag", e.gamertag, "done", totalDone, "no_film", totalNoFilm)
	}
}

// runKillSource — étape 1.57 : la SOURCE DU KILL des matchs insérés, puis le backlog.
//
// POURQUOI ICI, ET PAS DANS UN OUTIL SÉPARÉ. Cette donnée n'avait qu'un producteur : une
// sous-commande manuelle qui ne lisait que les films déjà en cache. Le cache a cessé d'être
// alimenté le 2026-04-07 et personne ne l'a vu pendant cinq mois — `assist_known` est resté
// FALSE sur tout match synchronisé depuis, et deux blocs de l'app se sont retirés sans un log
// (`.ai/V7.5/REGISTRE_ASSISTANCES_2026-08-29.md`). Une donnée qui ne se remplit que si
// quelqu'un lance une commande finit toujours par ne plus se remplir.
//
// Elle vient APRÈS runWeaponKills, pour la même raison que celui-ci vient après la
// convergence events : le film du match est disponible et récent, et le roster que le
// décodage doit joindre (`v_gamertag_lookup`) est à jour.
//
// TOUTE la logique vit dans internal/sync/killcollector (ratchet K3c : le neuf n'entre pas à
// la racine du god-package) ; ici on ne fait que câbler les dépendances du moteur.
func (s postSyncFilmSteps) runKillSource(ctx context.Context, insertedIDs []string) int {
	e := s.engine
	if e.killSource == nil {
		return 0
	}
	// GetFilmChunks est une capacité OPTIONNELLE du client (assertion, pas extension de
	// HaloClient : les mocks des autres étapes n'ont pas à la porter).
	//
	// ⚠ SON ABSENCE SE JOURNALISE ET SE COMPTE, ELLE NE SE TAIT PAS. Une première version
	// faisait un `return` nu : les deux clients de production ne portaient pas la méthode,
	// l'étape ne s'exécutait nulle part, et RIEN ne le disait — le défaut même que cette
	// étape corrige, reproduit dans son propre câblage. Les clients réels sont désormais
	// vérifiés à la COMPILATION (kill_source_wiring_test.go) ; ce compteur couvre le cas
	// qu'aucune assertion statique ne peut couvrir : un client injecté à l'exécution.
	fetcher, ok := s.client.(killcollector.FilmChunkFetcher)
	if !ok {
		observability.IncCounter(killcollector.CompteurPostSyncClientSansFilm)
		slog.WarnContext(ctx, "post-sync: kill source désarmée — le client ne porte pas GetFilmChunks",
			"gamertag", e.gamertag, "client", fmt.Sprintf("%T", s.client),
			"consequence", "assist_known restera FALSE sur les matchs de ce cycle")
		return 0
	}
	return killcollector.RunPostSync(ctx, e.killSource, killcollector.PostSyncDeps{
		Fetcher:    fetcher,
		LocalCache: e.localFilmCache,
		WithRead:   s.withRead,
		// Le writer est acquis PAR MATCH et relâché aussitôt (burst court) : le collecteur
		// résout son roster en lecture AVANT, donc les deux segments ne se chevauchent
		// jamais — même garde anti-deadlock que le burst weapons.
		AcquireWriter: func(c context.Context) (*sql.DB, func(), error) {
			return s.shared.Write(c, "killsource")
		},
		TitleSlug: e.titleSlug,
		Gamertag:  e.gamertag,
	}, insertedIDs)
}

// runReplayArtifacts — étape 1.58 : pont disque film + artefacts de rejeu 2D des matchs
// insérés. TOUTE la logique vit dans internal/sync/replayartifacts (ratchet K3c : le neuf
// n'entre pas à la racine du god-package) ; ici on ne fait que câbler les dépendances du
// moteur sur son API. Étape absente si le wiring n'a pas installé le hook (production).
func (s postSyncFilmSteps) runReplayArtifacts(ctx context.Context, insertedIDs []string) {
	e := s.engine
	if e.replayArtifacts == nil || len(insertedIDs) == 0 {
		return
	}
	// GetFilmChunks est une capacité OPTIONNELLE du client (assertion, pas extension de
	// HaloClient — les mocks des autres étapes n'ont pas à la porter). Son absence
	// n'interdit QUE la construction locale : mettre en file ne télécharge aucun film
	// (c'est l'ouvrier qui le fera), donc ce chemin-là reste ouvert.
	fetcher, _ := s.client.(replayartifacts.ChunksFetcher)
	replayartifacts.Run(ctx, replayartifacts.Deps{
		BuildOne:        e.replayArtifacts.BuildOne,
		Fetcher:         fetcher,
		WithRead:        s.withRead,
		MetaDB:          e.metaDB,
		RepoRoot:        e.repoRoot,
		TitleSlug:       e.titleSlug,
		Gamertag:        e.gamertag,
		CacheRoot:       e.replayArtifacts.CacheRoot,
		RetentionMonths: e.replayArtifacts.Months(),
		Placement:       e.replayArtifacts.Placement(),
		Enqueue:         e.replayArtifacts.Enqueue,
	}, insertedIDs)
}
