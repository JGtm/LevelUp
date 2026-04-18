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

	"levelup/go-api/internal/analysis"
)

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
		_ = MarkWeaponKillsDone(sharedDB, matchID, true)
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
	kills, err := getKillsForPlayer(sharedDB, matchID, xuid)
	if err != nil {
		return true, fmt.Errorf("BackfillWeaponKillsForMatch kills(%s): %w", matchID, err)
	}
	if len(kills) == 0 {
		_ = MarkWeaponKillsDone(sharedDB, matchID, false)
		return true, nil
	}

	xuidToPI, err := getXuidToPI(sharedDB, matchID)
	if err != nil {
		slog.Warn("backfill_weapons: xuidToPI non disponible", "match_id", matchID, "err", err)
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

	if err := InsertWeaponKills(sharedDB, matchID, xuid, rows); err != nil {
		return true, fmt.Errorf("BackfillWeaponKillsForMatch insert(%s): %w", matchID, err)
	}
	if err := MarkWeaponKillsDone(sharedDB, matchID, false); err != nil {
		slog.Warn("backfill_weapons: MarkWeaponKillsDone failed", "match_id", matchID, "err", err)
	}

	return true, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Méthode SyncEngine
// ─────────────────────────────────────────────────────────────────────────────

// BackfillWeaponKillsForMatches traite une liste de matchs (film pipeline).
// Retourne (done, noFilm, error).
func (e *SyncEngine) BackfillWeaponKillsForMatches(
	ctx context.Context,
	matchIDs []string,
) (done, noFilm int, err error) {
	sharedDB, err := OpenSharedDB(e.sharedDBPath)
	if err != nil {
		return 0, 0, fmt.Errorf("BackfillWeaponKillsForMatches OpenSharedDB: %w", err)
	}
	defer sharedDB.Close()

	client := NewHaloAPIClient(e.tokens.SpartanToken, e.tokens.ClearanceToken, 3)

	for _, matchID := range matchIDs {
		if ctx.Err() != nil {
			break
		}
		found, procErr := BackfillWeaponKillsForMatch(ctx, client, sharedDB, matchID, e.xuid)
		if procErr != nil {
			slog.Warn("backfill_weapons: erreur match", "match_id", matchID, "err", procErr)
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
// Utilise highlight_events (event_type='Killed') pour les kills du joueur.
func getKillsForPlayer(db *sql.DB, matchID, xuid string) ([]analysis.Kill, error) {
	rows, err := db.Query(`
		SELECT
			he.time_ms,
			CASE WHEN kv.weapon_type = 'melee' THEN TRUE ELSE FALSE END AS is_melee,
			CASE WHEN kv.weapon_type = 'grenade' THEN TRUE ELSE FALSE END AS is_grenade
		FROM highlight_events he
		LEFT JOIN killer_victim_pairs kv
			ON kv.match_id = he.match_id AND kv.killer_xuid = he.xuid
			AND ABS(COALESCE(kv.time_ms, 0) - COALESCE(he.time_ms, 0)) < 300
		WHERE he.match_id = ? AND he.xuid = ? AND he.event_type = 'Killed'
		ORDER BY he.time_ms`, matchID, xuid)
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
// l'ordre des participants (team_id ASC, rank_in_team ASC).
func getXuidToPI(db *sql.DB, matchID string) (map[string]int, error) {
	rows, err := db.Query(`
		SELECT xuid FROM match_participants
		WHERE match_id = ?
		ORDER BY team_id, rank_in_team NULLS LAST`, matchID)
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
