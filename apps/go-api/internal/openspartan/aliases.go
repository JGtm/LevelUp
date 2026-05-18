package openspartan

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// XuidAliasRow is one row of the OpenSpartan XuidAliases table — a mapping
// from xuid to last-known gamertag, with a source label and timestamps.
type XuidAliasRow struct {
	XUID     string
	Gamertag string
	LastSeen *time.Time
	Source   string
}

// LoadXuidAliases reads every row of the XuidAliases table, if present.
// Missing table is treated as "no aliases" (returns empty slice, no error)
// — XuidAliases is optional in older OpenSpartan schemas.
func (r *Reader) LoadXuidAliases(ctx context.Context) ([]XuidAliasRow, error) {
	r.mu.Lock()
	closed := r.closed
	r.mu.Unlock()
	if closed {
		return nil, ErrReaderClosed
	}

	hasTable, err := tableExists(ctx, r.db, "XuidAliases")
	if err != nil {
		return nil, err
	}
	if !hasTable {
		return nil, nil
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT Xuid, Gamertag, LastSeen, Source FROM XuidAliases`)
	if err != nil {
		return nil, fmt.Errorf("openspartan: query XuidAliases: %w", err)
	}
	defer rows.Close()

	var out []XuidAliasRow
	for rows.Next() {
		var (
			xuid, gamertag sql.NullString
			lastSeenStr    sql.NullString
			source         sql.NullString
		)
		if err := rows.Scan(&xuid, &gamertag, &lastSeenStr, &source); err != nil {
			return nil, fmt.Errorf("openspartan: scan XuidAliases: %w", err)
		}
		if !xuid.Valid || !gamertag.Valid {
			continue
		}
		row := XuidAliasRow{
			XUID:     xuid.String,
			Gamertag: gamertag.String,
		}
		if source.Valid {
			row.Source = source.String
		}
		if lastSeenStr.Valid {
			if t, err := time.Parse(time.RFC3339Nano, lastSeenStr.String); err == nil {
				row.LastSeen = &t
			} else if t, err := time.Parse(time.RFC3339, lastSeenStr.String); err == nil {
				row.LastSeen = &t
			}
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("openspartan: iterate XuidAliases: %w", err)
	}
	return out, nil
}

// AliasMap is a convenience helper that builds a {xuid: gamertag} lookup map
// from the aliases table. Missing entries simply absent from the map.
func (r *Reader) AliasMap(ctx context.Context) (map[string]string, error) {
	rows, err := r.LoadXuidAliases(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, a := range rows {
		out[a.XUID] = a.Gamertag
	}
	return out, nil
}
