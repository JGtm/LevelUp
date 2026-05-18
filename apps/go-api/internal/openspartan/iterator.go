package openspartan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log/slog"
)

// Matches returns an iterator that yields one *ParsedMatch per row in
// MatchStats, joining PlayerMatchStats on MatchId when a corresponding row
// exists. Matches are emitted in ascending chronological order (oldest first).
//
// The yielded ParsedMatch contains both the typed payload and the raw JSON
// bytes, so downstream code can re-emit fields the package did not bother to
// type.
//
// Iteration short-circuits when:
//   - the context is cancelled,
//   - the yield function returns false,
//   - a fatal SQL error is encountered.
//
// Per-row parse errors are surfaced via the (nil, err) yield pair: the caller
// decides whether to skip and continue (return true) or stop (return false).
// The reader must remain open for the lifetime of the iterator.
func (r *Reader) Matches(ctx context.Context) iter.Seq2[*ParsedMatch, error] {
	return func(yield func(*ParsedMatch, error) bool) {
		r.mu.Lock()
		closed := r.closed
		r.mu.Unlock()
		if closed {
			yield(nil, ErrReaderClosed)
			return
		}
		rows, err := r.db.QueryContext(ctx, `
			SELECT ms.ResponseBody, pms.ResponseBody
			FROM MatchStats AS ms
			LEFT JOIN PlayerMatchStats AS pms USING (MatchId)
			ORDER BY json_extract(ms.ResponseBody, '$.MatchInfo.StartTime') ASC`)
		if err != nil {
			yield(nil, fmt.Errorf("openspartan: query matches: %w", err))
			return
		}
		defer rows.Close()

		for rows.Next() {
			select {
			case <-ctx.Done():
				yield(nil, ctx.Err())
				return
			default:
			}
			var matchStatsBody, playerStatsBody []byte
			if err := rows.Scan(&matchStatsBody, &playerStatsBody); err != nil {
				slog.Warn("openspartan: scan match row failed", "err", err)
				if !yield(nil, fmt.Errorf("openspartan: scan match row: %w", err)) {
					return
				}
				continue
			}
			pm, err := parseMatch(matchStatsBody, playerStatsBody)
			if err != nil {
				slog.Warn("openspartan: parse match failed", "err", err)
			}
			if !yield(pm, err) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			slog.Error("openspartan: iterate matches failed", "err", err)
			yield(nil, fmt.Errorf("openspartan: iterate matches: %w", err))
		}
	}
}

// parseMatch decodes one MatchStats.ResponseBody (and an optional
// PlayerMatchStats.ResponseBody) into a *ParsedMatch. The raw bytes are
// always preserved on the returned struct, even when typed decoding of the
// secondary payload fails — callers can fall back to the raw form.
func parseMatch(matchStatsBody, playerStatsBody []byte) (*ParsedMatch, error) {
	if len(matchStatsBody) == 0 {
		return nil, errors.New("openspartan: empty MatchStats.ResponseBody")
	}
	pm := &ParsedMatch{
		RawMatchStats: append(json.RawMessage(nil), matchStatsBody...),
	}
	if err := json.Unmarshal(matchStatsBody, &pm.Stats); err != nil {
		return nil, fmt.Errorf("openspartan: unmarshal MatchStats: %w", err)
	}
	pm.MatchID = pm.Stats.MatchID

	if len(playerStatsBody) > 0 {
		pm.RawPlayerStats = append(json.RawMessage(nil), playerStatsBody...)
		// PlayerMatchStats.ResponseBody wraps the per-player array under "Value".
		var wrap struct {
			Value []PlayerMatchStatsValue `json:"Value"`
		}
		// Typed decoding is best-effort: malformed wrapper does not invalidate
		// the match — the raw bytes are still available via pm.RawPlayerStats.
		if err := json.Unmarshal(playerStatsBody, &wrap); err == nil {
			pm.PlayerStats = wrap.Value
		}
	}
	return pm, nil
}
