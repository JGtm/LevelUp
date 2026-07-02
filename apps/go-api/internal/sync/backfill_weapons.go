// Package sync — backfill_weapons.go : pipeline weapon kills via film Halo.
//
// Sprint 41 T2 : connecte le package analysis/ au moteur de sync.
//
// Flux :
//  1. GetMatchFilm → manifest + chunks binaires
//  2. BuildWeaponTimelines (analysis) → timelines armes par chunk
//  3. ScanFireEventsAll (analysis) → fire events de tous les joueurs
//  4. getKillsForPlayer (shared DB) → kills à attribuer
//  5. CorrelateKillsGlobal (analysis) → KillAttribution par kill
//  6. ReconcileAPIAggregates (analysis) → ajustements depuis API counts
//  7. InsertWeaponKills → écriture dans weapon_kills
//  8. MarkWeaponKillsDone → mise à jour bitmask
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
)

// weaponBackfillParallelism plafonne les matchs traités en parallèle par
// processWeaponKillsInline. Le parallélisme est NETWORK-ONLY (download film +
// corrélation en mémoire, throttle réel par le rate limiter HTTP). Les écritures
// weapon_kills (DELETE-then-INSERT par match_id) sont sérialisées par le lease
// shared + MaxOpenConns(1) → pas de concurrence sur l'index ART malgré le DELETE.
// NB : weapon_kills n'est PAS append-only (pas de id/written_at/_latest) — la
// sûreté vient de la sérialisation, pas du schéma. 24 = valeur Phase 3.6. Ex-
// healParallelismNetworkOnly, relocalisé ici à la décommission des heals (2026-06-01).
const weaponBackfillParallelism = 24

// ─────────────────────────────────────────────────────────────────────────────
// Pipeline principal
// ─────────────────────────────────────────────────────────────────────────────

// BackfillWeaponKillsForMatch exécute le pipeline complet pour un match.
// Retourne (filmFound, error) : filmFound=false si le film est absent (404/410).
func BackfillWeaponKillsForMatch(
	ctx context.Context,
	client HaloClient,
	sharedDB *sql.DB,
	matchID, xuid string,
) (bool, error) {
	// 1. Télécharger les chunks film.
	rawChunks, found, err := client.GetMatchFilm(ctx, matchID)
	if err != nil {
		return false, fmt.Errorf("BackfillWeaponKillsForMatch film(%s): %w", matchID, err)
	}
	if !found {
		_ = MarkWeaponKillsDone(ctx, sharedDB, matchID, true)
		return false, nil
	}

	// 2. Convertir filmChunkData → analysis.ChunkData.
	chunks := make(map[int]analysis.ChunkData, len(rawChunks))
	for idx, fc := range rawChunks {
		chunks[idx] = analysis.ChunkData{
			Data:       fc.Data,
			StartMS:    fc.StartMS,
			DurationMS: fc.DurationMS,
		}
	}

	// 3. Construire les timelines armes.
	timelines, chunksSorted := analysis.BuildWeaponTimelines(chunks)

	// 4. Scanner tous les fire events depuis les chunks.
	var allFireEvents []analysis.FireEvent
	for _, idx := range chunksSorted {
		cd := chunks[idx]
		evs := analysis.ScanFireEventsAll(cd.Data, cd.StartMS, cd.DurationMS)
		allFireEvents = append(allFireEvents, evs...)
	}

	// 5. Récupérer les kills et le mapping xuid → player_index.
	kills, err := getKillsForPlayer(ctx, sharedDB, matchID, xuid)
	if err != nil {
		return true, fmt.Errorf("BackfillWeaponKillsForMatch kills(%s): %w", matchID, err)
	}
	if len(kills) == 0 {
		// Pas de kills à corréler côté film → on ne MARQUE PAS bit21 (= "weapon_kills
		// chargés"). Marquer ce bit alors qu'aucune ligne n'a été insérée a vidé
		// silencieusement la table pour ~1010 matchs en mai 2026 (cf. thought_log
		// 2026-05-09). Le match peut être retraité plus tard si highlight_events
		// arrivent. Pas de bit22 non plus : le film EST disponible, c'est juste
		// que la corrélation côté DB n'est pas prête.
		return true, nil
	}

	xuidToPI, err := getXuidToPI(ctx, sharedDB, matchID)
	if err != nil {
		slog.WarnContext(ctx, "backfill_weapons: xuidToPI non disponible", "match_id", matchID, "err", err)
		xuidToPI = map[string]int{}
	}

	// 6. Corréler kills → attributions.
	attrs := analysis.CorrelateKillsGlobal(
		kills,
		allFireEvents,
		xuidToPI,
		timelines.Timeline,
		timelines.SwapPIs,
		timelines.Timing,
		chunksSorted,
		matchID,
		timelines.TimelineNS,
	)

	// 7. Convertir en WeaponKillRow pour l'insertion.
	rows := attributionsToRows(attrs, xuid)

	if err := InsertWeaponKills(ctx, sharedDB, matchID, xuid, rows); err != nil {
		return true, fmt.Errorf("BackfillWeaponKillsForMatch insert(%s): %w", matchID, err)
	}
	// On ne pose bit21 QUE si quelque chose a effectivement été inséré.
	// Garde-fou anti-faux-positif : un MarkWeaponKillsDone systématique a
	// causé la perte de ~1010 matchs en mai 2026 (cf. thought_log 2026-05-09).
	if len(rows) > 0 {
		if err := MarkWeaponKillsDone(ctx, sharedDB, matchID, false); err != nil {
			slog.WarnContext(ctx, "backfill_weapons: MarkWeaponKillsDone failed", "match_id", matchID, "err", err)
		}
	}

	return true, nil
}

// BackfillWeaponKillsForMatchAll exécute le pipeline complet pour un match,
// traitant TOUS les participants en une seule passe (film téléchargé une fois).
// Retourne (filmFound, error) : filmFound=false si film absent (404/410).
func BackfillWeaponKillsForMatchAll(
	ctx context.Context,
	client HaloClient,
	sharedDB *sql.DB,
	matchID string,
) (bool, error) {
	// 1. Télécharger les chunks film.
	rawChunks, found, err := client.GetMatchFilm(ctx, matchID)
	if err != nil {
		return false, fmt.Errorf("BackfillWeaponKillsForMatchAll film(%s): %w", matchID, err)
	}
	if !found {
		_ = MarkWeaponKillsDone(ctx, sharedDB, matchID, true)
		return false, nil
	}

	// 2. Convertir filmChunkData → analysis.ChunkData.
	chunks := make(map[int]analysis.ChunkData, len(rawChunks))
	for idx, fc := range rawChunks {
		chunks[idx] = analysis.ChunkData{
			Data:       fc.Data,
			StartMS:    fc.StartMS,
			DurationMS: fc.DurationMS,
		}
	}

	// 3. Construire les timelines armes.
	timelines, chunksSorted := analysis.BuildWeaponTimelines(chunks)

	// 4. Scanner tous les fire events depuis les chunks.
	var allFireEvents []analysis.FireEvent
	for _, idx := range chunksSorted {
		cd := chunks[idx]
		evs := analysis.ScanFireEventsAll(cd.Data, cd.StartMS, cd.DurationMS)
		allFireEvents = append(allFireEvents, evs...)
	}

	// 5. Récupérer les kills de TOUS les participants.
	allKills, err := getAllKillsForMatch(ctx, sharedDB, matchID)
	if err != nil {
		return true, fmt.Errorf("BackfillWeaponKillsForMatchAll getAllKills(%s): %w", matchID, err)
	}
	if len(allKills) == 0 {
		// Pas de kills à corréler → ne PAS poser bit21 (cf. thought_log 2026-05-09).
		return true, nil
	}

	// 6. Récupérer le mapping xuid → player_index (covers tous les participants).
	xuidToPI, err := getXuidToPI(ctx, sharedDB, matchID)
	if err != nil {
		slog.WarnContext(ctx, "backfill_weapons_all: xuidToPI non disponible", "match_id", matchID, "err", err)
		xuidToPI = map[string]int{}
	}

	// 7. Corréler kills → attributions (une seule fois, tous les kills).
	attrs := analysis.CorrelateKillsGlobal(
		allKills,
		allFireEvents,
		xuidToPI,
		timelines.Timeline,
		timelines.SwapPIs,
		timelines.Timing,
		chunksSorted,
		matchID,
		timelines.TimelineNS,
	)

	// 8. Récupérer les participants et insérer par xuid.
	xuids, err := getMatchParticipantXuids(ctx, sharedDB, matchID)
	if err != nil {
		return true, fmt.Errorf("BackfillWeaponKillsForMatchAll getParticipants(%s): %w", matchID, err)
	}

	totalInserted := 0
	for _, xuid := range xuids {
		rows := attributionsToRows(attrs, xuid)
		if err := InsertWeaponKills(ctx, sharedDB, matchID, xuid, rows); err != nil {
			slog.WarnContext(ctx, "backfill_weapons_all: insert failed", "match_id", matchID, "xuid", xuid, "err", err)
			continue
		}
		totalInserted += len(rows)
	}

	// 9. Marquer le match comme done UNIQUEMENT si au moins une ligne a été insérée.
	// Garde-fou anti-faux-positif : marquer bit21 alors que totalInserted == 0
	// laisse croire au sweep --weapons que le match est traité, alors que les
	// données existantes ont peut-être été préservées (par InsertWeaponKills
	// no-op) sans qu'on sache si elles couvrent tout. Si rien n'a été inséré,
	// on laisse le bit non-set pour permettre un retry.
	if totalInserted > 0 {
		if err := MarkWeaponKillsDone(ctx, sharedDB, matchID, false); err != nil {
			slog.WarnContext(ctx, "backfill_weapons_all: MarkWeaponKillsDone failed", "match_id", matchID, "err", err)
		}
	} else {
		slog.InfoContext(ctx, "backfill_weapons_all: 0 lignes insérées, bit21 non posé",
			"match_id", matchID, "kills", len(allKills))
	}

	return true, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Méthode SyncEngine
// ─────────────────────────────────────────────────────────────────────────────

// processWeaponKillsInline traite une liste de matchs sur des DBs déjà ouvertes
// (i.e. sans acquérir de lease). Utilisé depuis le PostSync chain où le shared
// lease est déjà détenu. Coût : 1 download film par match. Films absents
// (404/410) sont silencieux (matchs trop anciens).
//
// Parallélisé via errgroup.SetLimit(weaponBackfillParallelism=24) — plan
// Phase 3.0 (gain ~150-200s/cycle Madina) puis Phase 3.6 bump 8→24. Le
// parallélisme est network-only ; les writes weapon_kills (DELETE-then-INSERT)
// sont sérialisés par le lease shared + MaxOpenConns(1), donc sans conflit ART
// malgré le DELETE sur table indexée (weapon_kills n'est PAS append-only). Cf.
// .ai/PLAN_SYNC_CONCURRENCY_STABILIZATION.md §3.0+§3.6 + handoff §3 priorité 1.
//
// Contrat de sortie (lock par TDD avant impl) :
//   - matchIDs vide → (0, 0, nil)
//   - Tous films présents → done = len(matchIDs), noFilm = 0
//   - Tous films absents → done = 0, noFilm = len(matchIDs)
//   - 1 call par match_id (ni duplicate ni perte)
//   - Idempotent sur runs consécutifs
//   - ctx.Cancel → retourne ctx.Err() (les goroutines en cours abortent)
//   - Erreurs par-match : loggées WARN, best-effort (ne propage pas)
func processWeaponKillsInline(
	ctx context.Context,
	sharedDB *sql.DB,
	client HaloClient,
	xuid string,
	matchIDs []string,
) (done, noFilm int, err error) {
	if len(matchIDs) == 0 {
		return 0, 0, nil
	}
	var mu sync.Mutex
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(weaponBackfillParallelism)
	for _, matchID := range matchIDs {
		matchID := matchID // capture pour closure
		eg.Go(func() error {
			if egCtx.Err() != nil {
				return egCtx.Err()
			}
			found, procErr := BackfillWeaponKillsForMatch(egCtx, client, sharedDB, matchID, xuid)
			if procErr != nil {
				slog.WarnContext(egCtx, "weapon_kills: erreur match", "match_id", matchID, "err", procErr)
				return nil // best-effort : on n'interrompt pas les autres goroutines
			}
			mu.Lock()
			if found {
				done++
			} else {
				noFilm++
			}
			mu.Unlock()
			return nil
		})
	}
	_ = eg.Wait()
	// Propager ctx.Err() si cancellation pendant la run (préserve la sémantique
	// de l'ancienne version séquentielle qui retournait ctx.Err()).
	if ctx.Err() != nil {
		return done, noFilm, ctx.Err()
	}
	return done, noFilm, nil
}

// BackfillWeaponKillsForMatches traite une liste de matchs (film pipeline).
// Retourne (done, noFilm, error).
func (e *SyncEngine) BackfillWeaponKillsForMatches(
	ctx context.Context,
	matchIDs []string,
) (done, noFilm int, err error) {
	// Sprint B1 commit 13a : acquireSharedWriter centralise lease + open
	// (Provider en B-swap pour coordonner avec les readers HTTP en cours).
	sharedDB, releaseShared, err := e.acquireSharedWriter(ctxkeys.WithDBWriterLabel(ctx, "backfill_weapon_kills"))
	if err != nil {
		return 0, 0, fmt.Errorf("BackfillWeaponKillsForMatches: %w", err)
	}
	defer releaseShared()

	client := NewHaloAPIClient(e.tokens.SpartanToken, e.tokens.ClearanceToken, 3)

	for _, matchID := range matchIDs {
		if ctx.Err() != nil {
			break
		}
		// Traiter TOUS les participants du match en une seule passe.
		found, procErr := BackfillWeaponKillsForMatchAll(ctx, client, sharedDB, matchID)
		if procErr != nil {
			slog.WarnContext(ctx, "backfill_weapons: erreur match", "match_id", matchID, "err", procErr)
			continue
		}
		if found {
			done++
		} else {
			noFilm++
		}
	}
	return done, noFilm, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers DB
// ─────────────────────────────────────────────────────────────────────────────

// getKillsForPlayer récupère les kills d'un joueur dans un match depuis shared DB.
// Utilise highlight_events (event_type='kill' ou 'melee_kill' ou 'grenade_kill').
//
// Note : `killer_victim_pairs` n'a pas de colonne `weapon_type` — la
// distinction melee/grenade vient de l'event_type lui-même dans le film.
func getKillsForPlayer(ctx context.Context, db *sql.DB, matchID, xuid string) ([]analysis.Kill, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			time_ms,
			MAX(CASE WHEN LOWER(COALESCE(event_type, '')) LIKE '%melee%' THEN TRUE ELSE FALSE END) AS is_melee,
			MAX(CASE WHEN LOWER(COALESCE(event_type, '')) LIKE '%grenade%' THEN TRUE ELSE FALSE END) AS is_grenade
		FROM highlight_events
		WHERE match_id = ? AND xuid = ?
		  AND LOWER(COALESCE(event_type, '')) LIKE '%kill%'
		GROUP BY time_ms
		ORDER BY time_ms`, matchID, xuid)
	if err != nil {
		return nil, fmt.Errorf("getKillsForPlayer: %w", err)
	}
	defer rows.Close()

	var kills []analysis.Kill
	for rows.Next() {
		var (
			timeMS    *int
			isMelee   bool
			isGrenade bool
		)
		if err := rows.Scan(&timeMS, &isMelee, &isGrenade); err != nil {
			continue
		}
		tms := 0
		if timeMS != nil {
			tms = *timeMS
		}
		kills = append(kills, analysis.Kill{
			MatchID:   matchID,
			XUID:      xuid,
			TimeMS:    tms,
			IsMelee:   isMelee,
			IsGrenade: isGrenade,
		})
	}
	return kills, rows.Err()
}

// getXuidToPI construit le mapping xuid → player_index en se basant sur
// l'ordre des participants (team_id ASC, rank ASC).
func getXuidToPI(ctx context.Context, db *sql.DB, matchID string) (map[string]int, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT xuid FROM match_participants
		WHERE match_id = ?
		ORDER BY team_id, rank NULLS LAST`, matchID)
	if err != nil {
		return nil, fmt.Errorf("getXuidToPI: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	idx := 0
	for rows.Next() {
		var xu string
		if err := rows.Scan(&xu); err != nil {
			continue
		}
		result[xu] = idx
		idx++
	}
	return result, rows.Err()
}

// getAllKillsForMatch récupère les kills de TOUS les participants d'un match.
// Utile pour le backfill multi-joueurs : au lieu d'appeler getKillsForPlayer
// par xuid, on récupère une seule fois tous les kills et on itère sur les xuids.
func getAllKillsForMatch(ctx context.Context, db *sql.DB, matchID string) ([]analysis.Kill, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			xuid, time_ms,
			MAX(CASE WHEN LOWER(COALESCE(event_type, '')) LIKE '%melee%' THEN TRUE ELSE FALSE END) AS is_melee,
			MAX(CASE WHEN LOWER(COALESCE(event_type, '')) LIKE '%grenade%' THEN TRUE ELSE FALSE END) AS is_grenade
		FROM highlight_events
		WHERE match_id = ?
		  AND LOWER(COALESCE(event_type, '')) LIKE '%kill%'
		GROUP BY xuid, time_ms
		ORDER BY time_ms`, matchID)
	if err != nil {
		return nil, fmt.Errorf("getAllKillsForMatch: %w", err)
	}
	defer rows.Close()

	var kills []analysis.Kill
	for rows.Next() {
		var (
			xuid      string
			timeMS    *int
			isMelee   bool
			isGrenade bool
		)
		if err := rows.Scan(&xuid, &timeMS, &isMelee, &isGrenade); err != nil {
			continue
		}
		tms := 0
		if timeMS != nil {
			tms = *timeMS
		}
		kills = append(kills, analysis.Kill{
			MatchID:   matchID,
			XUID:      xuid,
			TimeMS:    tms,
			IsMelee:   isMelee,
			IsGrenade: isGrenade,
		})
	}
	return kills, rows.Err()
}

// getMatchParticipantXuids retourne la liste des xuids participants à un match.
func getMatchParticipantXuids(ctx context.Context, db *sql.DB, matchID string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT xuid FROM match_participants WHERE match_id = ?
		ORDER BY xuid`, matchID)
	if err != nil {
		return nil, fmt.Errorf("getMatchParticipantXuids: %w", err)
	}
	defer rows.Close()

	var xuids []string
	for rows.Next() {
		var xuid string
		if err := rows.Scan(&xuid); err != nil {
			continue
		}
		xuids = append(xuids, xuid)
	}
	return xuids, rows.Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// Conversion
// ─────────────────────────────────────────────────────────────────────────────

// attributionsToRows filtre les attributions pour le joueur et les convertit en WeaponKillRow.
func attributionsToRows(attrs []analysis.KillAttribution, xuid string) []WeaponKillRow {
	rows := make([]WeaponKillRow, 0, len(attrs))
	for _, a := range attrs {
		if a.XUID != xuid {
			continue
		}
		rows = append(rows, WeaponKillRow{
			TimeMS:          a.TimeMS,
			WeaponID:        a.WeaponID,
			ReconciledAs:    a.ReconciledAs,
			DeltaMS:         a.DeltaMS,
			Confidence:      a.Confidence,
			AttributionPath: a.AttributionPath,
			SwapDetected:    a.SwapDetected,
			DelayedDamage:   a.DelayedDamage,
			PlayerIndex:     a.PlayerIndex,
		})
	}
	return rows
}
