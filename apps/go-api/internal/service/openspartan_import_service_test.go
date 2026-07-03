package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	_ "modernc.org/sqlite"

	"levelup/go-api/internal/sync"
)

const (
	testOwnerXUID = "2533274823110022"
	// testOtherXUID must be lexicographically GREATER than testOwnerXUID
	// so that, in a 1-match fixture where both have the same frequency,
	// the mostFrequentHumanXUID tie-breaker (smallest-first) picks the owner.
	testOtherXUID = "2533274999999999"
)

// buildMinimalFixtureDB writes a tiny but realistic OpenSpartan SQLite at
// dir/<filename>: 1 match with 2 humans + 1 bot + 1 highlight + 2 aliases +
// 1 friend. Sufficient to exercise every import branch end-to-end.
func buildMinimalFixtureDB(t *testing.T, dir, filename, ownerXUID string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatalf("open sqlite fixture: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, ddl := range []string{
		`CREATE TABLE MatchStats (
			ResponseBody TEXT,
			MatchId TEXT GENERATED ALWAYS AS (json_extract(ResponseBody, '$.MatchId')) VIRTUAL
		)`,
		`CREATE TABLE PlayerMatchStats (ResponseBody TEXT, MatchId TEXT)`,
		`CREATE TABLE HighlightEvents (MatchId TEXT NOT NULL, ResponseBody TEXT NOT NULL)`,
		`CREATE TABLE CacheMeta (key TEXT PRIMARY KEY, value TEXT, updated_at TEXT)`,
		`CREATE TABLE XuidAliases (Xuid TEXT PRIMARY KEY, Gamertag TEXT NOT NULL, LastSeen TEXT, Source TEXT, UpdatedAt TEXT)`,
		`CREATE TABLE Friends (id INTEGER PRIMARY KEY AUTOINCREMENT, owner_xuid TEXT NOT NULL, friend_xuid TEXT NOT NULL, friend_gamertag TEXT, nickname TEXT, added_at TEXT)`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("DDL %q: %v", ddl, err)
		}
	}

	matchID := "11111111-aaaa-bbbb-cccc-000000000001"
	start := time.Date(2026, 1, 2, 20, 18, 1, 0, time.UTC)
	end := start.Add(12 * time.Minute)

	matchStats := map[string]any{
		"MatchId": matchID,
		"MatchInfo": map[string]any{
			"StartTime": start.Format(time.RFC3339Nano), "EndTime": end.Format(time.RFC3339Nano),
			"Duration": "PT12M", "PlayableDuration": "PT12M",
			"LifecycleMode": 3, "GameVariantCategory": 6,
			"SeasonId":       "CsrSeason13-1", // match classé → CSR par-match (RankRecap)
			"LevelId":        "level-1",
			"MapVariant":     map[string]any{"AssetKind": 2, "AssetId": "map-1", "VersionId": "map-v1"},
			"UgcGameVariant": map[string]any{"AssetKind": 6, "AssetId": "variant-1", "VersionId": "variant-v1"},
		},
		"Teams": []any{
			map[string]any{"TeamId": 0, "Outcome": 2, "Rank": 1,
				"Stats": map[string]any{"CoreStats": map[string]any{"Score": 50, "PersonalScore": 5000}}},
			map[string]any{"TeamId": 1, "Outcome": 3, "Rank": 2,
				"Stats": map[string]any{"CoreStats": map[string]any{"Score": 47, "PersonalScore": 4321}}},
		},
		"Players": []any{
			map[string]any{
				"PlayerId": "xuid(" + ownerXUID + ")", "PlayerType": 1,
				"LastTeamId": 0, "Outcome": 2, "Rank": 1,
				"ParticipationInfo": map[string]any{"TimePlayed": "PT12M"},
				"PlayerTeamStats": []any{map[string]any{
					"TeamId": 0,
					"Stats": map[string]any{"CoreStats": map[string]any{
						"Score": 19, "Kills": 19, "Deaths": 11, "Assists": 3, "KDA": 9.0,
						"Accuracy": 60.32, "ShotsFired": 436, "ShotsHit": 263,
						"DamageDealt": 4889, "DamageTaken": 4159, "HeadshotKills": 11,
						"MaxKillingSpree": 6, "AverageLifeDuration": "PT46S",
						"Medals": []any{map[string]any{"NameId": 3546244406, "Count": 2}},
					}},
				}},
			},
			map[string]any{
				"PlayerId": "xuid(" + testOtherXUID + ")", "PlayerType": 1,
				"LastTeamId": 1, "Outcome": 3, "Rank": 3,
				"ParticipationInfo": map[string]any{"TimePlayed": "PT12M"},
				"PlayerTeamStats": []any{map[string]any{
					"TeamId": 1,
					"Stats": map[string]any{"CoreStats": map[string]any{
						"Kills": 5, "Deaths": 12, "Medals": []any{map[string]any{"NameId": 622331684, "Count": 1}},
					}},
				}},
			},
			map[string]any{
				"PlayerId": "bid(123-1)", "PlayerType": 2,
				"LastTeamId": 1, "Outcome": 3, "Rank": 4,
				"ParticipationInfo": map[string]any{"TimePlayed": "PT12M"},
				"PlayerTeamStats":   []any{map[string]any{"TeamId": 1, "Stats": map[string]any{"CoreStats": map[string]any{}}}},
			},
		},
	}
	body, _ := json.Marshal(matchStats)
	if _, err := db.Exec(`INSERT INTO MatchStats(ResponseBody) VALUES (?)`, string(body)); err != nil {
		t.Fatalf("insert MatchStats: %v", err)
	}

	pms := map[string]any{"Value": []any{map[string]any{
		"Id": "xuid(" + ownerXUID + ")", "ResultCode": 0,
		"Result": map[string]any{
			"TeamMmr": 1041.7,
			// RankRecap : présent uniquement sur les matchs classés → source du CSR par-match.
			"RankRecap": map[string]any{
				"PreMatchCsr":  map[string]any{"Value": 1450, "Tier": "Gold", "SubTier": 4, "MeasurementMatchesRemaining": 0},
				"PostMatchCsr": map[string]any{"Value": 1465, "Tier": "Gold", "SubTier": 5, "MeasurementMatchesRemaining": 0},
			},
		},
	}}}
	pmsBody, _ := json.Marshal(pms)
	if _, err := db.Exec(`INSERT INTO PlayerMatchStats(ResponseBody, MatchId) VALUES (?, ?)`, string(pmsBody), matchID); err != nil {
		t.Fatalf("insert PlayerMatchStats: %v", err)
	}

	hl, _ := json.Marshal(map[string]any{
		"event_type": "kill", "time_ms": 46832,
		"xuid": 2533274823110022, "type_hint": 50,
	})
	if _, err := db.Exec(`INSERT INTO HighlightEvents(MatchId, ResponseBody) VALUES (?, ?)`, matchID, string(hl)); err != nil {
		t.Fatalf("insert HighlightEvents: %v", err)
	}

	for _, a := range []struct{ x, g string }{
		{ownerXUID, "TestOwner"},
		{testOtherXUID, "OtherPlayer"},
	} {
		if _, err := db.Exec(`INSERT INTO XuidAliases(Xuid, Gamertag, Source, UpdatedAt) VALUES (?, ?, 'api', ?)`,
			a.x, a.g, time.Now().UTC().Format(time.RFC3339)); err != nil {
			t.Fatalf("insert XuidAliases: %v", err)
		}
	}
	if _, err := db.Exec(`INSERT INTO Friends(owner_xuid, friend_xuid, friend_gamertag) VALUES (?, ?, ?)`,
		ownerXUID, testOtherXUID, "OtherPlayer"); err != nil {
		t.Fatalf("insert Friends: %v", err)
	}

	abs, _ := filepath.Abs(path)
	return abs
}

func setupSharedDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpPath := filepath.Join(t.TempDir(), "shared.duckdb")
	db, err := sql.Open("duckdb", tmpPath)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := sync.EnsureSharedSchema(t.Context(), db); err != nil {
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	// highlight_events lives in a separate migration step
	// (internal/migration/steps_shared.go applyDropHighlightEventsGamertag).
	// We mirror that DDL here so the integration test exercises the same
	// columns InsertHighlightEvents writes to.
	if _, err := db.Exec(`
		CREATE SEQUENCE IF NOT EXISTS highlight_events_id_seq;
		CREATE TABLE IF NOT EXISTS highlight_events (
			id         INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
			match_id   VARCHAR NOT NULL,
			event_type VARCHAR NOT NULL,
			time_ms    INTEGER,
			xuid       VARCHAR,
			type_hint  INTEGER,
			raw_json   VARCHAR
		);
		CREATE INDEX IF NOT EXISTS idx_highlight_match ON highlight_events(match_id);
	`); err != nil {
		t.Fatalf("create highlight_events: %v", err)
	}
	// Colonnes ajoutées par migration title-owned (match_intensity/backfill_completed
	// sur match_registry ; backfill_bits + mécaniques de kill Halo 5 sur
	// match_participants), absentes de sharedSchemaSQL statique mais écrites par
	// persist.SharedPersister.Persist (E1 route l'import via ce persister). Sans ce
	// mirroring : Binder Error "column ..." — même patch que sync/patchSharedSchemaForBatch.
	for _, col := range []string{"match_intensity DOUBLE", "backfill_completed BIGINT DEFAULT 0"} {
		if _, err := db.Exec("ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS " + col); err != nil {
			t.Fatalf("patch match_registry %s: %v", col, err)
		}
	}
	for _, col := range []string{
		"backfill_bits INTEGER",
		"assassination_kills SMALLINT DEFAULT 0",
		"ground_pound_kills SMALLINT DEFAULT 0",
		"shoulder_bash_kills SMALLINT DEFAULT 0",
	} {
		if _, err := db.Exec("ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS " + col); err != nil {
			t.Fatalf("patch match_participants %s: %v", col, err)
		}
	}
	return db
}

func TestOpenSpartanImport_EndToEnd_WritesAllRowFamilies(t *testing.T) {
	dir := t.TempDir()
	fixturePath := buildMinimalFixtureDB(t, dir, testOwnerXUID+".db", testOwnerXUID)
	sharedDB := setupSharedDB(t)
	stashDir := filepath.Join(dir, "stash")

	svc := NewOpenSpartanImportServiceForTest(sharedDB)
	result, err := svc.Import(context.Background(), testOwnerXUID, fixturePath, ImportOptions{
		Source: "test", StashDir: stashDir,
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	if result.DetectedOwnerXUID != testOwnerXUID {
		t.Errorf("DetectedOwnerXUID: want %s, got %s", testOwnerXUID, result.DetectedOwnerXUID)
	}
	if result.InsertedMatches != 1 {
		t.Errorf("InsertedMatches: want 1, got %d", result.InsertedMatches)
	}
	if result.InsertedParticipants != 2 {
		t.Errorf("InsertedParticipants: want 2 humans, got %d", result.InsertedParticipants)
	}
	if result.InsertedMedals != 2 {
		t.Errorf("InsertedMedals: want 2, got %d", result.InsertedMedals)
	}
	if result.InsertedHighlights != 1 {
		t.Errorf("InsertedHighlights: want 1, got %d", result.InsertedHighlights)
	}
	if result.InsertedCSRs != 1 {
		t.Errorf("InsertedCSRs: want 1 (owner RankRecap classé), got %d", result.InsertedCSRs)
	}
	// Le CSR par-match est écrit dans shared.match_csrs depuis RankRecap, avec le
	// season_id du match (même quand reg.IsRanked n'est pas encore résolu à l'import).
	var csrSeason, csrTier sql.NullString
	var csrValue sql.NullFloat64
	if err := sharedDB.QueryRow(
		`SELECT season_id, tier, rating_value FROM match_csrs WHERE xuid = ?`, testOwnerXUID).
		Scan(&csrSeason, &csrTier, &csrValue); err != nil {
		t.Fatalf("query match_csrs: %v", err)
	}
	if csrSeason.String != "CsrSeason13-1" {
		t.Errorf("match_csrs.season_id: want CsrSeason13-1, got %q", csrSeason.String)
	}
	if !csrValue.Valid || csrValue.Float64 != 1465 {
		t.Errorf("match_csrs.rating_value: want 1465, got %v", csrValue)
	}
	if csrTier.String != "Gold" {
		t.Errorf("match_csrs.tier: want Gold, got %q", csrTier.String)
	}
	// is_ranked corrigé à l'import via la présence du RankRecap (sinon faux car
	// PlaylistName non résolu) → exclut correctement ce match du recompute LUSR.
	var isRanked sql.NullBool
	if err := sharedDB.QueryRow(
		`SELECT is_ranked FROM match_registry WHERE match_id = ?`,
		"11111111-aaaa-bbbb-cccc-000000000001").Scan(&isRanked); err != nil {
		t.Fatalf("query registry is_ranked: %v", err)
	}
	if !isRanked.Valid || !isRanked.Bool {
		t.Error("match_registry.is_ranked devrait être TRUE (RankRecap présent → classé)")
	}
	if result.InsertedAliases != 2 {
		t.Errorf("InsertedAliases: want 2, got %d", result.InsertedAliases)
	}
	if result.StashedFriends != 1 {
		t.Errorf("StashedFriends: want 1, got %d", result.StashedFriends)
	}
	if len(result.Errors) > 0 {
		t.Errorf("expected no errors, got %d: %+v", len(result.Errors), result.Errors)
	}

	// Verify rows landed in DuckDB.
	verifyCount(t, sharedDB, "match_registry", 1)
	verifyCount(t, sharedDB, "match_participants", 2)
	verifyCount(t, sharedDB, "medals_earned", 2)
	verifyCount(t, sharedDB, "highlight_events", 1)
	verifyCount(t, sharedDB, "xuid_aliases", 2)

	// Verify the participant rows carry the gamertag resolved from XuidAliases.
	var gt sql.NullString
	if err := sharedDB.QueryRow(`SELECT gamertag FROM match_participants WHERE xuid = ?`, testOwnerXUID).Scan(&gt); err != nil {
		t.Fatalf("query gamertag: %v", err)
	}
	if !gt.Valid || gt.String != "TestOwner" {
		t.Errorf("owner gamertag should be 'TestOwner' (resolved from aliases), got %q", gt.String)
	}

	// Friends stash file written.
	stashPath := filepath.Join(stashDir, testOwnerXUID, "stash", "openspartan_friends.json")
	if _, err := os.Stat(stashPath); err != nil {
		t.Fatalf("stash file should exist at %s: %v", stashPath, err)
	}
}

func TestOpenSpartanImport_RejectsXUIDMismatch(t *testing.T) {
	dir := t.TempDir()
	fixturePath := buildMinimalFixtureDB(t, dir, testOwnerXUID+".db", testOwnerXUID)
	sharedDB := setupSharedDB(t)

	svc := NewOpenSpartanImportServiceForTest(sharedDB)
	_, err := svc.Import(context.Background(), "9999999999999999", fixturePath, ImportOptions{StashDir: dir})
	if !errors.Is(err, ErrXUIDMismatch) {
		t.Fatalf("want ErrXUIDMismatch, got %v", err)
	}
	// Nothing should have been written to DuckDB.
	verifyCount(t, sharedDB, "match_registry", 0)
}

func TestOpenSpartanImport_DryRunCountsButDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	fixturePath := buildMinimalFixtureDB(t, dir, testOwnerXUID+".db", testOwnerXUID)
	sharedDB := setupSharedDB(t)

	svc := NewOpenSpartanImportServiceForTest(sharedDB)
	result, err := svc.Import(context.Background(), testOwnerXUID, fixturePath, ImportOptions{
		DryRun: true, StashDir: filepath.Join(dir, "stash"),
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if result.InsertedMatches != 1 || result.InsertedParticipants != 2 {
		t.Errorf("dry-run counts mismatch: %+v", result)
	}
	verifyCount(t, sharedDB, "match_registry", 0)
	stashPath := filepath.Join(dir, "stash", testOwnerXUID, "stash", "openspartan_friends.json")
	if _, err := os.Stat(stashPath); err == nil {
		t.Error("dry-run should not write the friends stash")
	}
}

func TestOpenSpartanImport_OnProgressInvokedPerMatch(t *testing.T) {
	dir := t.TempDir()
	fixturePath := buildMinimalFixtureDB(t, dir, testOwnerXUID+".db", testOwnerXUID)
	sharedDB := setupSharedDB(t)
	svc := NewOpenSpartanImportServiceForTest(sharedDB)
	calls := 0
	_, err := svc.Import(context.Background(), testOwnerXUID, fixturePath, ImportOptions{
		StashDir: filepath.Join(dir, "stash"),
		OnProgress: func(parsed, total int) {
			calls++
			if total != 1 {
				t.Errorf("OnProgress total: want 1, got %d", total)
			}
			if parsed != calls {
				t.Errorf("OnProgress parsed should equal call count, got %d on call %d", parsed, calls)
			}
		},
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if calls != 1 {
		t.Errorf("OnProgress should fire once per match (1), got %d", calls)
	}
}

func verifyCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Errorf("table %s rows: want %d, got %d", table, want, got)
	}
}
