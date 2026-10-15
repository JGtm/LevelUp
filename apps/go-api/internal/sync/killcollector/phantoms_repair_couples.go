package killcollector

// phantoms_repair_couples.go — LES LECTURES DE [RebuildStalePairs] : quels matchs portent des
// kills ou des morts, et quels couples ils portent deja.

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"slices"

	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/persist"
)

// matchsAvecKillsOuMorts : les matchs dont `highlight_events` porte au moins un kill ou une mort,
// dans l ordre des identifiants.
func matchsAvecKillsOuMorts(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT match_id FROM highlight_events
		WHERE LOWER(event_type) IN (?, ?) AND xuid IS NOT NULL AND xuid <> ''
		ORDER BY 1`, highlightevent.EventTypeKill, highlightevent.EventTypeDeath)
	if err != nil {
		return nil, fmt.Errorf("couples a recalculer: matchs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("couples a recalculer: scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// coupleCle : ce qui distingue un couple d un autre — tueur, victime, instant. Les gamertags n en
// font pas partie : ils viennent des couples existants.
type coupleCle struct {
	tueur, victime string
	ms             int64
}

// couplesStockes : les couples que `killer_victim_pairs` porte pour un match.
func couplesStockes(ctx context.Context, db *sql.DB, matchID string) ([]coupleCle, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT killer_xuid, victim_xuid, COALESCE(time_ms, 0) FROM killer_victim_pairs
		WHERE match_id = ?`, matchID)
	if err != nil {
		return nil, fmt.Errorf("couples a recalculer: couples stockes %s: %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()
	var out []coupleCle
	for rows.Next() {
		var c coupleCle
		if err := rows.Scan(&c.tueur, &c.victime, &c.ms); err != nil {
			return nil, fmt.Errorf("couples a recalculer: scan %s: %w", matchID, err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// memesCouples : les couples stockes sont-ils exactement ceux du calcul, multiplicite comprise ?
func memesCouples(stockes []coupleCle, calcules []persist.KVPairCompletion) bool {
	if len(stockes) != len(calcules) {
		return false
	}
	b := make([]coupleCle, 0, len(calcules))
	for _, p := range calcules {
		b = append(b, coupleCle{tueur: p.KillerXUID, victime: p.VictimXUID, ms: p.TimeMS})
	}
	a := slices.Clone(stockes)
	trierCouples(a)
	trierCouples(b)
	return slices.Equal(a, b)
}

// trierCouples : l ordre total des couples (instant, tueur, victime).
func trierCouples(cs []coupleCle) {
	slices.SortFunc(cs, func(x, y coupleCle) int {
		return cmp.Or(cmp.Compare(x.ms, y.ms), cmp.Compare(x.tueur, y.tueur), cmp.Compare(x.victime, y.victime))
	})
}
