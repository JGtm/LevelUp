package openspartan

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// buildFixtureDB creates a minimal but realistic OpenSpartan SQLite database
// at the requested filename inside dir. The schema mirrors what real grunt
// databases expose (canonical tables with ResponseBody TEXT columns).
//
// The fixture inserts three matches:
//   - match A: ownerXUID + otherHuman1 + 1 bot
//   - match B: ownerXUID + otherHuman2
//   - match C: ownerXUID alone
//
// After construction the owner XUID appears in 3 matches while every other
// human appears in 1 — making it unambiguous for mostFrequentHumanXUID.
//
// The returned path is absolute.
func buildFixtureDB(t *testing.T, dir, filename, ownerXUID string) string {
	t.Helper()
	path := filepath.Join(dir, filename)

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatalf("open fixture db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	stmts := []string{
		`CREATE TABLE MatchStats (
			ResponseBody TEXT,
			MatchId TEXT GENERATED ALWAYS AS (json_extract(ResponseBody, '$.MatchId')) VIRTUAL
		)`,
		`CREATE UNIQUE INDEX IDX_MATCH_STATS ON MatchStats (MatchId)`,
		`CREATE TABLE PlayerMatchStats (
			ResponseBody TEXT,
			MatchId TEXT
		)`,
		`CREATE TABLE HighlightEvents (
			MatchId TEXT NOT NULL,
			ResponseBody TEXT NOT NULL
		)`,
		`CREATE TABLE CacheMeta (key TEXT PRIMARY KEY, value TEXT, updated_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE XuidAliases (Xuid TEXT PRIMARY KEY, Gamertag TEXT NOT NULL, LastSeen TEXT, Source TEXT, UpdatedAt TEXT)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("create fixture schema (%s): %v", s, err)
		}
	}

	otherHuman1 := "2533274801010001"
	otherHuman2 := "2533274801010002"
	botID := "bid(8589934592-100)"

	mkMatch := func(matchID string, startTime time.Time, humanXUIDs []string, includeBot bool) MatchStats {
		t.Helper()
		players := make([]Player, 0, len(humanXUIDs)+1)
		for i, x := range humanXUIDs {
			players = append(players, Player{
				PlayerID:   "xuid(" + x + ")",
				PlayerType: 1,
				LastTeamID: 0,
				Outcome:    2,
				Rank:       i + 1,
				ParticipationInfo: ParticipationInfo{
					FirstJoinedTime:     startTime,
					PresentAtBeginning:  true,
					PresentAtCompletion: true,
					TimePlayed:          "PT10M",
				},
				PlayerTeamStats: []PlayerTeamStat{{
					TeamID: 0,
					Stats: StatsBundle{CoreStats: CoreStats{
						Kills:   10,
						Deaths:  5,
						Assists: 2,
						KDA:     7.5,
						Medals: []MedalAward{
							{NameID: 3546244406, Count: 1},
							{NameID: 622331684, Count: 2},
						},
					}},
				}},
			})
		}
		if includeBot {
			players = append(players, Player{
				PlayerID:   botID,
				PlayerType: 2,
				LastTeamID: 1,
				Outcome:    3,
				Rank:       4,
				ParticipationInfo: ParticipationInfo{
					FirstJoinedTime:     startTime,
					PresentAtBeginning:  true,
					PresentAtCompletion: true,
					TimePlayed:          "PT10M",
				},
				PlayerTeamStats: []PlayerTeamStat{{
					TeamID: 1,
					Stats:  StatsBundle{CoreStats: CoreStats{Kills: 3, Deaths: 12}},
				}},
			})
		}
		return MatchStats{
			MatchID: matchID,
			MatchInfo: MatchInfo{
				StartTime:           startTime,
				EndTime:             startTime.Add(10 * time.Minute),
				Duration:            "PT10M",
				LifecycleMode:       3,
				GameVariantCategory: 6,
				LevelID:             "b963a5ed-a8d0-4475-a47e-67430c56b3bd",
				MapVariant:          AssetRef{AssetKind: 2, AssetID: "map-id", VersionID: "map-v"},
				UgcGameVariant:      AssetRef{AssetKind: 6, AssetID: "variant-id", VersionID: "variant-v"},
				PlayableDuration:    "PT10M",
				TeamsEnabled:        true,
				TeamScoringEnabled:  true,
			},
			Teams:   []Team{{TeamID: 0, Outcome: 2, Rank: 1}, {TeamID: 1, Outcome: 3, Rank: 2}},
			Players: players,
		}
	}

	insertMatch := func(ms MatchStats) {
		body, err := json.Marshal(ms)
		if err != nil {
			t.Fatalf("marshal MatchStats: %v", err)
		}
		if _, err := db.Exec(`INSERT INTO MatchStats(ResponseBody) VALUES (?)`, string(body)); err != nil {
			t.Fatalf("insert MatchStats: %v", err)
		}
		pms := map[string]any{
			"Value": []PlayerMatchStatsValue{{
				ID:         "xuid(" + ownerXUID + ")",
				ResultCode: 0,
				Result:     &PlayerMatchStatsResult{TeamMmr: 1234.5},
			}},
		}
		pmsBody, err := json.Marshal(pms)
		if err != nil {
			t.Fatalf("marshal PlayerMatchStats: %v", err)
		}
		if _, err := db.Exec(`INSERT INTO PlayerMatchStats(ResponseBody, MatchId) VALUES (?, ?)`,
			string(pmsBody), ms.MatchID); err != nil {
			t.Fatalf("insert PlayerMatchStats: %v", err)
		}
		// One synthetic highlight event per match.
		hl := map[string]any{
			"event_type": "Kill",
			"time_ms":    12345,
			"xuid":       ownerXUID,
			"gamertag":   "TestOwner",
			"type_hint":  1,
		}
		hlBody, _ := json.Marshal(hl)
		if _, err := db.Exec(`INSERT INTO HighlightEvents(MatchId, ResponseBody) VALUES (?, ?)`,
			ms.MatchID, string(hlBody)); err != nil {
			t.Fatalf("insert HighlightEvents: %v", err)
		}
	}

	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	insertMatch(mkMatch("11111111-aaaa-bbbb-cccc-000000000001", t0,
		[]string{ownerXUID, otherHuman1}, true))
	insertMatch(mkMatch("22222222-aaaa-bbbb-cccc-000000000002", t0.Add(time.Hour),
		[]string{ownerXUID, otherHuman2}, false))
	insertMatch(mkMatch("33333333-aaaa-bbbb-cccc-000000000003", t0.Add(2*time.Hour),
		[]string{ownerXUID}, false))

	// Persist owner in CacheMeta as a fallback signal.
	if _, err := db.Exec(`INSERT INTO CacheMeta(key, value) VALUES (?, ?)`,
		"current_user_xuid", ownerXUID); err != nil {
		t.Fatalf("insert CacheMeta: %v", err)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	return abs
}

// buildEmptyDB writes a minimal SQLite file with no OpenSpartan tables, used to
// negative-test the detector.
func buildEmptyDB(t *testing.T, dir, filename string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatalf("open empty db: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE Unrelated(id INTEGER)`); err != nil {
		t.Fatalf("create unrelated: %v", err)
	}
	_ = db.Close()
	abs, _ := filepath.Abs(path)
	return abs
}

// matchCountDirect is a small helper for the tests below.
func matchCountDirect(t *testing.T, r *Reader) int {
	t.Helper()
	n, err := r.MatchCount(context.Background())
	if err != nil {
		t.Fatalf("MatchCount: %v", err)
	}
	return n
}
