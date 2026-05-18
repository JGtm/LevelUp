package openspartan

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// buildAuxOnlyFixture writes a SQLite OpenSpartan file with the canonical
// tables but populated only for the aux paths under test (XuidAliases,
// Friends, HighlightEvents, CacheMeta). MatchStats / PlayerMatchStats are
// present (required by detectSchema) but left empty.
func buildAuxOnlyFixture(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "aux.db")
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, ddl := range []string{
		`CREATE TABLE MatchStats (
			ResponseBody TEXT,
			MatchId TEXT GENERATED ALWAYS AS (json_extract(ResponseBody, '$.MatchId')) VIRTUAL
		)`,
		`CREATE TABLE PlayerMatchStats (ResponseBody TEXT, MatchId TEXT)`,
		`CREATE TABLE HighlightEvents (MatchId TEXT NOT NULL, ResponseBody TEXT NOT NULL)`,
		`CREATE TABLE XuidAliases (Xuid TEXT PRIMARY KEY, Gamertag TEXT NOT NULL, LastSeen TEXT, Source TEXT, UpdatedAt TEXT)`,
		`CREATE TABLE Friends (id INTEGER PRIMARY KEY AUTOINCREMENT, owner_xuid TEXT NOT NULL, friend_xuid TEXT NOT NULL, friend_gamertag TEXT, nickname TEXT, added_at TEXT)`,
		`CREATE TABLE CacheMeta (key TEXT PRIMARY KEY, value TEXT, updated_at TEXT)`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("ddl: %v", err)
		}
	}
	for _, a := range []struct{ x, g, src, ls string }{
		{"2533274823110022", "TestOwner", "api", "2026-01-01T12:00:00Z"},
		{"2533274801010001", "PlayerB", "api", "2026-01-02T09:00:00Z"},
		{"", "ShouldBeSkipped", "api", ""}, // invalid row (empty XUID) — skipped
	} {
		if _, err := db.Exec(`INSERT INTO XuidAliases(Xuid, Gamertag, LastSeen, Source) VALUES (?, ?, ?, ?)`,
			a.x, a.g, a.ls, a.src); err != nil && a.x != "" {
			t.Fatalf("insert XuidAliases: %v", err)
		}
	}
	for _, f := range []struct{ owner, fr, gt string }{
		{"2533274823110022", "2533274801010001", "PlayerB"},
		{"2533274823110022", "2533274802020002", "PlayerC"},
	} {
		if _, err := db.Exec(`INSERT INTO Friends(owner_xuid, friend_xuid, friend_gamertag) VALUES (?, ?, ?)`,
			f.owner, f.fr, f.gt); err != nil {
			t.Fatalf("insert Friends: %v", err)
		}
	}
	for matchID, body := range map[string]string{
		"m1": `{"event_type":"kill","time_ms":1234,"xuid":2533274823110022,"type_hint":50}`,
		"m2": `{"event_type":"medal","time_ms":4567,"xuid":2533274823110022,"type_hint":12}`,
		"m3": `{"event_type":"death","time_ms":7890,"xuid":2533274801010001,"type_hint":1}`,
	} {
		if _, err := db.Exec(`INSERT INTO HighlightEvents(MatchId, ResponseBody) VALUES (?, ?)`, matchID, body); err != nil {
			t.Fatalf("insert HighlightEvents: %v", err)
		}
	}
	if _, err := db.Exec(`INSERT INTO CacheMeta(key, value) VALUES (?, ?)`,
		"current_user_xuid", "2533274823110022"); err != nil {
		t.Fatalf("insert CacheMeta: %v", err)
	}
	abs, _ := filepath.Abs(path)
	return abs
}

func TestLoadXuidAliases_ReturnsRows(t *testing.T) {
	path := buildAuxOnlyFixture(t, t.TempDir())
	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	aliases, err := r.LoadXuidAliases(context.Background())
	if err != nil {
		t.Fatalf("LoadXuidAliases: %v", err)
	}
	if len(aliases) != 2 {
		t.Fatalf("len: want 2 (invalid XUID skipped), got %d", len(aliases))
	}
	wantSource := "api"
	for _, a := range aliases {
		if a.XUID == "" || a.Gamertag == "" {
			t.Errorf("alias has empty fields: %+v", a)
		}
		if a.Source != wantSource {
			t.Errorf("Source: want %q, got %q", wantSource, a.Source)
		}
	}
}

func TestLoadXuidAliases_ParsesLastSeenTimestamp(t *testing.T) {
	path := buildAuxOnlyFixture(t, t.TempDir())
	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	aliases, err := r.LoadXuidAliases(context.Background())
	if err != nil {
		t.Fatalf("LoadXuidAliases: %v", err)
	}
	var owner XuidAliasRow
	for _, a := range aliases {
		if a.XUID == "2533274823110022" {
			owner = a
		}
	}
	if owner.LastSeen == nil {
		t.Fatal("owner LastSeen should be parsed from the RFC3339 string")
	}
	if owner.LastSeen.Year() != 2026 || owner.LastSeen.Month() != 1 {
		t.Errorf("LastSeen year/month: want 2026/01, got %v", owner.LastSeen)
	}
}

func TestAliasMap_BuildsXUIDtoGamertagLookup(t *testing.T) {
	path := buildAuxOnlyFixture(t, t.TempDir())
	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	m, err := r.AliasMap(context.Background())
	if err != nil {
		t.Fatalf("AliasMap: %v", err)
	}
	if m["2533274823110022"] != "TestOwner" {
		t.Errorf("want TestOwner, got %q", m["2533274823110022"])
	}
	if m["2533274801010001"] != "PlayerB" {
		t.Errorf("want PlayerB, got %q", m["2533274801010001"])
	}
	if _, found := m["unknown"]; found {
		t.Error("unknown xuids should not be in the map")
	}
}

func TestLoadFriends_ReturnsRows(t *testing.T) {
	path := buildAuxOnlyFixture(t, t.TempDir())
	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	friends, err := r.LoadFriends(context.Background())
	if err != nil {
		t.Fatalf("LoadFriends: %v", err)
	}
	if len(friends) != 2 {
		t.Fatalf("len: want 2, got %d", len(friends))
	}
	for _, f := range friends {
		if f.OwnerXUID != "2533274823110022" {
			t.Errorf("OwnerXUID: want 2533274823110022, got %q", f.OwnerXUID)
		}
		if f.FriendXUID == "" {
			t.Error("FriendXUID should not be empty")
		}
	}
}

func TestLoadFriends_MissingTableReturnsNilNoError(t *testing.T) {
	// Build a minimal OpenSpartan DB with the canonical tables but NO Friends table.
	dir := t.TempDir()
	path := filepath.Join(dir, "no_friends.db")
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	for _, ddl := range []string{
		`CREATE TABLE MatchStats (ResponseBody TEXT, MatchId TEXT GENERATED ALWAYS AS (json_extract(ResponseBody, '$.MatchId')) VIRTUAL)`,
		`CREATE TABLE PlayerMatchStats (ResponseBody TEXT, MatchId TEXT)`,
		`CREATE TABLE HighlightEvents (MatchId TEXT NOT NULL, ResponseBody TEXT NOT NULL)`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("ddl: %v", err)
		}
	}
	_ = db.Close()

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	friends, err := r.LoadFriends(context.Background())
	if err != nil {
		t.Fatalf("LoadFriends should not error on missing table: %v", err)
	}
	if friends != nil {
		t.Errorf("want nil slice when table absent, got %v", friends)
	}
}

func TestHighlights_YieldsAllRowsOrderedByMatchID(t *testing.T) {
	path := buildAuxOnlyFixture(t, t.TempDir())
	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	var collected []HighlightRow
	for hl, err := range r.Highlights(context.Background()) {
		if err != nil {
			t.Fatalf("Highlights yielded error: %v", err)
		}
		collected = append(collected, hl)
	}
	if len(collected) != 3 {
		t.Fatalf("len: want 3, got %d", len(collected))
	}
	// Ordered by MatchId ascending — "m1" < "m2" < "m3" lexicographically.
	for i, want := range []string{"m1", "m2", "m3"} {
		if collected[i].MatchID != want {
			t.Errorf("position %d: want match_id %s, got %s", i, want, collected[i].MatchID)
		}
		if len(collected[i].RawJSON) == 0 {
			t.Errorf("position %d: RawJSON should be preserved", i)
		}
	}
}

func TestHighlights_StopsOnYieldFalse(t *testing.T) {
	path := buildAuxOnlyFixture(t, t.TempDir())
	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	count := 0
	for _, err := range r.Highlights(context.Background()) {
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		count++
		break
	}
	if count != 1 {
		t.Fatalf("want iteration to stop at 1, got %d", count)
	}
}

func TestHighlightCount_ReturnsRowCount(t *testing.T) {
	path := buildAuxOnlyFixture(t, t.TempDir())
	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	n, err := r.HighlightCount(context.Background())
	if err != nil {
		t.Fatalf("HighlightCount: %v", err)
	}
	if n != 3 {
		t.Errorf("want 3, got %d", n)
	}
}

func TestXuidFromCacheMeta_ReturnsStoredXUID(t *testing.T) {
	path := buildAuxOnlyFixture(t, t.TempDir())
	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	got, err := r.xuidFromCacheMeta(context.Background())
	if err != nil {
		t.Fatalf("xuidFromCacheMeta: %v", err)
	}
	if got != "2533274823110022" {
		t.Errorf("want 2533274823110022, got %q", got)
	}
}

func TestConfidence_String(t *testing.T) {
	cases := map[Confidence]string{
		ConfidenceNone:   "none",
		ConfidenceLow:    "low",
		ConfidenceMedium: "medium",
		ConfidenceHigh:   "high",
		Confidence(999):  "none", // default branch
	}
	for c, want := range cases {
		if got := c.String(); got != want {
			t.Errorf("Confidence(%d).String() = %q, want %q", c, got, want)
		}
	}
}

func TestTableExists_FindsExistingAndAbsent(t *testing.T) {
	path := buildAuxOnlyFixture(t, t.TempDir())
	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	ctx := context.Background()
	ok, err := tableExists(ctx, r.db, "CacheMeta")
	if err != nil {
		t.Fatalf("tableExists CacheMeta: %v", err)
	}
	if !ok {
		t.Error("CacheMeta should exist in fixture")
	}
	ok, err = tableExists(ctx, r.db, "DoesNotExist")
	if err != nil {
		t.Fatalf("tableExists DoesNotExist: %v", err)
	}
	if ok {
		t.Error("DoesNotExist should be absent")
	}
}
