package sync

// highlight_phantoms_repair.go — LA SUITE DE `migration.PurgeHighlightPhantoms` : les couples
// `killer_victim_pairs` des matchs touches se recalculent depuis `highlight_events` assaini.
//
// Les couples sont apparies au plus proche : un fantome avait pu prendre la mort d un vrai kill
// (l ordre de tri des evenements de meme instant n etait pas total), donc il ne suffit pas de
// retirer les couples du fantome — il faut re-apparier. Les gamertags viennent des couples
// existants du match (a defaut, le xuid : `victim_gamertag` est NOT NULL) ; la base credit, elle,
// se reconstruit ensuite par `backfill-killsource`, qui lit l annuaire.

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/persist"
)

// phantomPairToleranceMS : la fenetre d appariement kill <-> death, celle de la completion combat.
const phantomPairToleranceMS = int64(5)

// RebuildPairsAfterPhantomPurge recalcule les couples de chaque match de `matchIDs` et rend le
// nombre de matchs reecrits. Idempotent : le calcul est deterministe (ordre total des evenements).
func RebuildPairsAfterPhantomPurge(ctx context.Context, db *sql.DB, matchIDs []string) (int, error) {
	pers := persist.NewEventsCompletionPersister(db)
	done := 0
	for _, id := range matchIDs {
		raw, err := phantomRepairEvents(ctx, db, id)
		if err != nil {
			return done, err
		}
		pairs := analysis.ComputeKillerVictimPairs(raw, phantomPairToleranceMS)
		rows := make([]persist.KVPairCompletion, 0, len(pairs))
		for _, p := range pairs {
			rows = append(rows, persist.KVPairCompletion{
				KillerXUID: p.KillerXUID, KillerGamertag: p.KillerGT,
				VictimXUID: p.VictimXUID, VictimGamertag: p.VictimGT, TimeMS: p.TimeMS,
			})
		}
		if err := pers.RebuildKillerVictimPairs(ctx, id, rows); err != nil {
			return done, err
		}
		done++
	}
	return done, nil
}

// phantomRepairEvents : kills et morts du match (doublons exacts ecartes), nommes d apres les
// couples existants.
func phantomRepairEvents(ctx context.Context, db *sql.DB, matchID string) ([]analysis.RawEvent, error) {
	noms, err := phantomRepairNames(ctx, db, matchID)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT xuid, LOWER(event_type), COALESCE(time_ms, 0) FROM highlight_events
		WHERE match_id = ? AND LOWER(event_type) IN (?, ?) AND xuid IS NOT NULL AND xuid <> ''
		ORDER BY 3, 1, 2`, matchID, highlightevent.EventTypeKill, highlightevent.EventTypeDeath)
	if err != nil {
		return nil, fmt.Errorf("repair-highlight-phantoms: evenements %s: %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()
	var out []analysis.RawEvent
	for rows.Next() {
		var ev analysis.RawEvent
		if err := rows.Scan(&ev.XUID, &ev.EventType, &ev.TimeMS); err != nil {
			return nil, fmt.Errorf("repair-highlight-phantoms: scan %s: %w", matchID, err)
		}
		ev.Gamertag = noms[ev.XUID]
		if ev.Gamertag == "" {
			ev.Gamertag = ev.XUID
		}
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repair-highlight-phantoms: lignes %s: %w", matchID, err)
	}
	return out, nil
}

func phantomRepairNames(ctx context.Context, db *sql.DB, matchID string) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT killer_xuid, MAX(killer_gamertag) FROM killer_victim_pairs
		WHERE match_id = ? AND killer_gamertag IS NOT NULL GROUP BY 1
		UNION
		SELECT victim_xuid, MAX(victim_gamertag) FROM killer_victim_pairs
		WHERE match_id = ? AND victim_gamertag IS NOT NULL GROUP BY 1`, matchID, matchID)
	if err != nil {
		return nil, fmt.Errorf("repair-highlight-phantoms: noms %s: %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()
	noms := map[string]string{}
	for rows.Next() {
		var x, g string
		if err := rows.Scan(&x, &g); err != nil {
			return nil, fmt.Errorf("repair-highlight-phantoms: noms scan %s: %w", matchID, err)
		}
		if _, ok := noms[x]; !ok {
			noms[x] = g
		}
	}
	return noms, rows.Err()
}
