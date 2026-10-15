package killcollector

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

// RebuildPairsAfterPhantomPurge recalcule les couples de chaque match de `matchIDs` et rend le
// nombre de matchs reecrits. Idempotent : le calcul est deterministe (ordre total des evenements).
func RebuildPairsAfterPhantomPurge(ctx context.Context, db *sql.DB, matchIDs []string) (int, error) {
	pers := persist.NewEventsCompletionPersister(db)
	done := 0
	for _, id := range matchIDs {
		pairs, err := couplesRecalcules(ctx, db, id)
		if err != nil {
			return done, err
		}
		if err := pers.RebuildKillerVictimPairs(ctx, id, pairs); err != nil {
			return done, err
		}
		done++
	}
	return done, nil
}

// StalePairsReport : le bilan de [RebuildStalePairs].
type StalePairsReport struct {
	// Examined : matchs qui portent au moins un kill ou une mort dans `highlight_events`.
	Examined int
	// MatchIDs : ceux dont les couples stockes different de l appariement courant, dans l ordre
	// des identifiants.
	MatchIDs []string
	// Rewritten : ceux qui ont ete reecrits (zero en lecture seule).
	Rewritten int
}

// RebuildStalePairs confronte les couples stockes de CHAQUE match a l appariement courant
// (`analysis.ComputeKillerVictimPairs`) et, quand `ecrire`, reecrit ceux qui different. Sert la
// suite d un changement de l appariement : les couples ecrits par l appariement d avant restent
// en base jusqu a ce passage. Idempotent : un second passage ne trouve plus rien.
func RebuildStalePairs(ctx context.Context, db *sql.DB, ecrire bool) (StalePairsReport, error) {
	var rep StalePairsReport
	ids, err := matchsAvecKillsOuMorts(ctx, db)
	if err != nil {
		return rep, err
	}
	pers := persist.NewEventsCompletionPersister(db)
	for _, id := range ids {
		rep.Examined++
		pairs, err := couplesRecalcules(ctx, db, id)
		if err != nil {
			return rep, err
		}
		stockes, err := couplesStockes(ctx, db, id)
		if err != nil {
			return rep, err
		}
		if memesCouples(stockes, pairs) {
			continue
		}
		rep.MatchIDs = append(rep.MatchIDs, id)
		if !ecrire {
			continue
		}
		if err := pers.RebuildKillerVictimPairs(ctx, id, pairs); err != nil {
			return rep, err
		}
		rep.Rewritten++
	}
	return rep, nil
}

// couplesRecalcules : les couples d un match selon l appariement courant.
func couplesRecalcules(ctx context.Context, db *sql.DB, matchID string) ([]persist.KVPairCompletion, error) {
	raw, err := phantomRepairEvents(ctx, db, matchID)
	if err != nil {
		return nil, err
	}
	pairs := analysis.ComputeKillerVictimPairs(raw, analysis.KillDeathPairToleranceMS)
	rows := make([]persist.KVPairCompletion, 0, len(pairs))
	for _, p := range pairs {
		rows = append(rows, persist.KVPairCompletion{
			KillerXUID: p.KillerXUID, KillerGamertag: p.KillerGT,
			VictimXUID: p.VictimXUID, VictimGamertag: p.VictimGT, TimeMS: p.TimeMS,
		})
	}
	return rows, nil
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
