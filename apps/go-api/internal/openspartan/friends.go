package openspartan

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// FriendRow is one row of the OpenSpartan Friends table — a self-declared
// "this user is my Xbox friend" pair. The mapper does NOT translate this
// into a DuckDB table; instead the import service stashes the full list as
// JSON for the future MULTIUSER_ACL sprint to consume.
type FriendRow struct {
	OwnerXUID      string
	FriendXUID     string
	FriendGamertag string
	Nickname       string
	AddedAt        *time.Time
}

// LoadFriends reads every row of the Friends table, if present. Missing
// table or empty table both yield (nil, nil) — Friends is optional in older
// OpenSpartan schemas.
func (r *Reader) LoadFriends(ctx context.Context) ([]FriendRow, error) {
	r.mu.Lock()
	closed := r.closed
	r.mu.Unlock()
	if closed {
		return nil, ErrReaderClosed
	}

	hasTable, err := tableExists(ctx, r.db, "Friends")
	if err != nil {
		return nil, err
	}
	if !hasTable {
		return nil, nil
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT owner_xuid, friend_xuid, friend_gamertag, nickname, added_at FROM Friends`)
	if err != nil {
		return nil, fmt.Errorf("openspartan: query Friends: %w", err)
	}
	defer rows.Close()

	var out []FriendRow
	for rows.Next() {
		var (
			owner, friend       sql.NullString
			gamertag, nickname  sql.NullString
			addedAtStr          sql.NullString
		)
		if err := rows.Scan(&owner, &friend, &gamertag, &nickname, &addedAtStr); err != nil {
			return nil, fmt.Errorf("openspartan: scan Friends: %w", err)
		}
		if !owner.Valid || !friend.Valid {
			continue
		}
		row := FriendRow{
			OwnerXUID:      owner.String,
			FriendXUID:     friend.String,
			FriendGamertag: stringOrEmpty(gamertag),
			Nickname:       stringOrEmpty(nickname),
		}
		if addedAtStr.Valid {
			for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
				if t, err := time.Parse(layout, addedAtStr.String); err == nil {
					row.AddedAt = &t
					break
				}
			}
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("openspartan: iterate Friends: %w", err)
	}
	return out, nil
}

func stringOrEmpty(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}
