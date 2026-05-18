package openspartan

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"iter"
)

// HighlightRow is one row of the OpenSpartan HighlightEvents table, surfaced
// with both the MatchId index and the raw JSON body. The mapper turns this
// into a HighlightEventRow for DuckDB v6.
type HighlightRow struct {
	MatchID string
	RawJSON json.RawMessage
}

// Highlights returns an iterator that yields every row from the
// HighlightEvents table, ordered by MatchId. Missing table is handled
// gracefully by yielding nothing.
//
// The iterator stops early on ctx cancellation, on a fatal SQL error, or
// when the yield function returns false. Per-row scan errors are surfaced
// via the (zero, err) yield pair; the caller decides whether to continue.
func (r *Reader) Highlights(ctx context.Context) iter.Seq2[HighlightRow, error] {
	return func(yield func(HighlightRow, error) bool) {
		r.mu.Lock()
		closed := r.closed
		r.mu.Unlock()
		if closed {
			yield(HighlightRow{}, ErrReaderClosed)
			return
		}
		hasTable, err := tableExists(ctx, r.db, "HighlightEvents")
		if err != nil {
			yield(HighlightRow{}, err)
			return
		}
		if !hasTable {
			return
		}
		rows, err := r.db.QueryContext(ctx,
			`SELECT MatchId, ResponseBody FROM HighlightEvents ORDER BY MatchId`)
		if err != nil {
			yield(HighlightRow{}, fmt.Errorf("openspartan: query HighlightEvents: %w", err))
			return
		}
		defer rows.Close()

		for rows.Next() {
			select {
			case <-ctx.Done():
				yield(HighlightRow{}, ctx.Err())
				return
			default:
			}
			var (
				matchID sql.NullString
				body    []byte
			)
			if err := rows.Scan(&matchID, &body); err != nil {
				if !yield(HighlightRow{}, fmt.Errorf("openspartan: scan HighlightEvents: %w", err)) {
					return
				}
				continue
			}
			if !matchID.Valid {
				continue
			}
			hr := HighlightRow{
				MatchID: matchID.String,
				RawJSON: append(json.RawMessage(nil), body...),
			}
			if !yield(hr, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(HighlightRow{}, fmt.Errorf("openspartan: iterate HighlightEvents: %w", err))
		}
	}
}

// HighlightCount returns the total number of HighlightEvents rows. Useful
// for progress reporting before iteration. Returns 0 when the table is
// absent.
func (r *Reader) HighlightCount(ctx context.Context) (int, error) {
	r.mu.Lock()
	closed := r.closed
	r.mu.Unlock()
	if closed {
		return 0, ErrReaderClosed
	}
	hasTable, err := tableExists(ctx, r.db, "HighlightEvents")
	if err != nil {
		return 0, err
	}
	if !hasTable {
		return 0, nil
	}
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM HighlightEvents`).Scan(&n); err != nil {
		return 0, fmt.Errorf("openspartan: count HighlightEvents: %w", err)
	}
	return n, nil
}
