// Package v2 — known_loader_history_test.go : KnownLoader rend l'état de l'historique et le
// marque-page lus sur la base joueur ouverte (DuckDB réelle, sync_meta posée par historystate).
package v2

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/sync/historystate"
)

func TestKnownLoaderV2_EtatDeLHistorique(t *testing.T) {
	cursor := historystate.Cursor{Offset: 400, LastMatchID: "m-400"}
	cases := []struct {
		name       string
		withMeta   bool
		arrange    func(ctx context.Context, db *sql.DB)
		wantIncomp bool
		wantCursor *historystate.Cursor
	}{
		{"clé de fin absente : incomplet, sans marque-page", true, func(context.Context, *sql.DB) {}, true, nil},
		{"incomplet avec marque-page", true, func(ctx context.Context, db *sql.DB) { historystate.SaveCursor(ctx, db, cursor) }, true, &cursor},
		{"fin rangée : complet", true, func(ctx context.Context, db *sql.DB) { historystate.MarkComplete(ctx, db, "alice") }, false, nil},
		{"fin rangée et marque-page résiduel : complet", true, func(ctx context.Context, db *sql.DB) {
			historystate.SaveCursor(ctx, db, cursor)
			historystate.MarkComplete(ctx, db, "alice")
		}, false, nil},
		{"lecture impossible (sans sync_meta) : réputé complet", false, func(context.Context, *sql.DB) {}, false, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			playerDB, _ := setupTestPlayerDB(t, nil)
			if tc.withMeta {
				if _, err := playerDB.SQLDb().Exec(`CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR, updated_at TIMESTAMP)`); err != nil {
					t.Fatal(err)
				}
			}
			tc.arrange(ctx, playerDB.SQLDb())
			sharedDB := setupTestSharedDB(t, map[string][]string{"999": {"m1"}})
			opener := func(context.Context, string) (*sql.DB, func(), error) { return playerDB.SQLDb(), func() {}, nil }

			st, err := NewKnownLoader(opener, SharedBorrower(nil, func() *sql.DB { return sharedDB })).
				LoadKnown(ctx, PlayerProfile{Gamertag: "alice", XUID: "999"})
			if err != nil {
				t.Fatalf("LoadKnown: %v", err)
			}
			if st.History.Incomplete != tc.wantIncomp {
				t.Errorf("Incomplete = %v, want %v", st.History.Incomplete, tc.wantIncomp)
			}
			switch {
			case tc.wantCursor == nil && st.History.Cursor != nil:
				t.Errorf("Cursor = %+v, want nil", *st.History.Cursor)
			case tc.wantCursor != nil && (st.History.Cursor == nil || *st.History.Cursor != *tc.wantCursor):
				t.Errorf("Cursor = %v, want %+v", st.History.Cursor, *tc.wantCursor)
			}
			if !st.Known["m1"] {
				t.Error("l'ensemble connu n'est plus rendu avec l'état")
			}
		})
	}
}
