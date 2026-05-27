//go:build cgo

package sync

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/platform/duckdb"
)

func TestIsLUSRV2Enabled(t *testing.T) {
	original := os.Getenv(lusrV2EnvFlag)
	t.Cleanup(func() { _ = os.Setenv(lusrV2EnvFlag, original) })

	cases := []struct {
		envVal string
		want   bool
	}{
		{"", false},
		{"0", false},
		{"false", false},
		{"no", false},
		{"random_junk", false},
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"yes", true},
		{"YES", true},
		{"  1  ", true}, // trim
	}
	for _, c := range cases {
		t.Run(c.envVal, func(t *testing.T) {
			_ = os.Setenv(lusrV2EnvFlag, c.envVal)
			if got := IsLUSRV2Enabled(); got != c.want {
				t.Errorf("IsLUSRV2Enabled() with env=%q = %v, want %v", c.envVal, got, c.want)
			}
		})
	}
}

func TestOutcomeToTeamResult(t *testing.T) {
	cases := []struct {
		name    string
		outcome int
		want    skillv2.TeamResult
		wantOK  bool
	}{
		{"Win", 2, skillv2.TeamWin, true},
		{"Tie", 1, skillv2.TeamDraw, true},
		{"Loss", 3, skillv2.TeamLoss, true},
		{"DNF skipped", 4, 0, false},
		{"unknown_0", 0, 0, false},
		{"unknown_99", 99, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := outcomeToTeamResult(c.outcome)
			if ok != c.wantOK {
				t.Errorf("outcomeToTeamResult(%d) ok = %v, want %v", c.outcome, ok, c.wantOK)
			}
			if ok && got != c.want {
				t.Errorf("outcomeToTeamResult(%d) = %v, want %v", c.outcome, got, c.want)
			}
		})
	}
}

// openShadowTestDB ouvre une DuckDB en mémoire avec les tables LUSR v2 +
// match_registry + match_participants + xuid_aliases (le minimum pour
// tester le shadow runner end-to-end).
func openShadowTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	const ddl = `
		CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP,
			start_time_utc TIMESTAMPTZ,
			pair_name VARCHAR,
			is_ranked BOOLEAN DEFAULT FALSE,
			is_firefight BOOLEAN DEFAULT FALSE,
			duration_seconds INTEGER
		);
		CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			team_id INTEGER,
			outcome INTEGER,
			kills DOUBLE,
			deaths DOUBLE,
			PRIMARY KEY (match_id, xuid)
		);
		CREATE SEQUENCE player_skill_state_v2_seq START 1;
		CREATE TABLE player_skill_state_v2 (
			id              BIGINT DEFAULT nextval('player_skill_state_v2_seq') PRIMARY KEY,
			xuid            VARCHAR NOT NULL,
			playlist_group  VARCHAR NOT NULL,
			mu              DOUBLE  NOT NULL,
			sigma           DOUBLE  NOT NULL,
			experience      INTEGER NOT NULL DEFAULT 0,
			last_match_id   VARCHAR,
			last_match_at   TIMESTAMP,
			written_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE VIEW player_skill_state_v2_latest AS
		SELECT s.*
		FROM player_skill_state_v2 s
		JOIN (
			SELECT xuid, playlist_group, MAX(written_at) AS max_written_at
			FROM player_skill_state_v2
			GROUP BY xuid, playlist_group
		) m ON s.xuid = m.xuid AND s.playlist_group = m.playlist_group AND s.written_at = m.max_written_at;
	`
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("DDL: %v", err)
	}
	return db
}

func TestBuildTwoTeamRosters_TwoTeamsOK(t *testing.T) {
	db := openShadowTestDB(t)
	// 4 joueurs, 2 par équipe avec kills/deaths.
	for _, q := range []struct {
		xuid          string
		team          int
		kills, deaths int
	}{
		{"a1", 0, 15, 5}, {"a2", 0, 10, 8}, {"b1", 1, 8, 12}, {"b2", 1, 6, 13},
	} {
		_, err := db.Exec("INSERT INTO match_participants(match_id, xuid, team_id, outcome, kills, deaths) VALUES (?, ?, ?, 2, ?, ?)",
			"m1", q.xuid, q.team, q.kills, q.deaths)
		if err != nil {
			t.Fatal(err)
		}
	}
	teamA, teamB, ok := buildTwoTeamRosters(context.Background(), db, "m1", 0)
	if !ok {
		t.Fatal("expected 2-team match to be accepted")
	}
	if len(teamA) != 2 || len(teamB) != 2 {
		t.Errorf("rosters sizes: A=%d, B=%d, want 2/2", len(teamA), len(teamB))
	}
	wantA := map[string]bool{"a1": true, "a2": true}
	for _, m := range teamA {
		if !wantA[m.xuid] {
			t.Errorf("teamA contient xuid inattendu : %v", m.xuid)
		}
		// Phase 3c : kills/deaths doivent être chargés
		if m.kills == nil {
			t.Errorf("teamA[%s] : kills nil, attendu chargé depuis DB", m.xuid)
		}
		if m.deaths == nil {
			t.Errorf("teamA[%s] : deaths nil, attendu chargé depuis DB", m.xuid)
		}
	}
}

func TestBuildTwoTeamRosters_FFAReject(t *testing.T) {
	db := openShadowTestDB(t)
	// FFA : 4 joueurs, 4 équipes différentes.
	for i := 0; i < 4; i++ {
		_, err := db.Exec("INSERT INTO match_participants(match_id, xuid, team_id, outcome, kills, deaths) VALUES (?, ?, ?, 2, 5, 5)",
			"m1", string(rune('a'+i)), i)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, _, ok := buildTwoTeamRosters(context.Background(), db, "m1", 0); ok {
		t.Error("FFA (4 teams) should be rejected")
	}
}

func TestBuildTwoTeamRosters_NoMatch(t *testing.T) {
	db := openShadowTestDB(t)
	if _, _, ok := buildTwoTeamRosters(context.Background(), db, "missing", 0); ok {
		t.Error("missing match_id should be rejected")
	}
}

func TestRunLUSRV2Shadow_FlagOff_NoOp(t *testing.T) {
	original := os.Getenv(lusrV2EnvFlag)
	t.Cleanup(func() { _ = os.Setenv(lusrV2EnvFlag, original) })
	_ = os.Setenv(lusrV2EnvFlag, "0")

	db := openShadowTestDB(t)
	n, err := RunLUSRV2Shadow(context.Background(), db, "anyxuid")
	if err != nil {
		t.Fatalf("RunLUSRV2Shadow: %v", err)
	}
	if n != 0 {
		t.Errorf("flag off should be no-op, got n=%d", n)
	}
}

func TestRunLUSRV2Shadow_FullFlow_2v2Match(t *testing.T) {
	original := os.Getenv(lusrV2EnvFlag)
	t.Cleanup(func() { _ = os.Setenv(lusrV2EnvFlag, original) })
	_ = os.Setenv(lusrV2EnvFlag, "1")

	db := openShadowTestDB(t)
	// 1 match social, 4 joueurs (2v2), pair_name "Slayer" → arena_slayer.
	startTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	_, err := db.Exec(`INSERT INTO match_registry
		(match_id, start_time, start_time_utc, pair_name, is_ranked, is_firefight, duration_seconds)
		VALUES (?, ?, ?, 'Slayer', FALSE, FALSE, 600)`,
		"m1", startTime, startTime)
	if err != nil {
		t.Fatalf("insert match_registry: %v", err)
	}
	for _, q := range []struct {
		xuid          string
		team          int
		outcome       int
		kills, deaths int
	}{
		{"owner", 0, 2, 18, 6}, {"teammate1", 0, 2, 12, 9},
		{"opp1", 1, 3, 7, 14}, {"opp2", 1, 3, 8, 14},
	} {
		_, err := db.Exec(`INSERT INTO match_participants
			(match_id, xuid, team_id, outcome, kills, deaths) VALUES (?, ?, ?, ?, ?, ?)`,
			"m1", q.xuid, q.team, q.outcome, q.kills, q.deaths)
		if err != nil {
			t.Fatalf("insert participant: %v", err)
		}
	}

	processed, err := RunLUSRV2Shadow(context.Background(), db, "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2Shadow: %v", err)
	}
	if processed != 1 {
		t.Errorf("processed = %d, want 1", processed)
	}

	// État écrit dans player_skill_state_v2 pour TOUS les joueurs.
	repo := duckdb.NewSkillV2Repo(db)
	priors := skillv2.DefaultPriors()
	for _, xuid := range []string{"owner", "teammate1", "opp1", "opp2"} {
		st, err := repo.LoadState(context.Background(), xuid, "arena_slayer")
		if err != nil {
			t.Errorf("LoadState(%s): %v", xuid, err)
			continue
		}
		if st == nil {
			t.Errorf("state nil for %s, expected created", xuid)
			continue
		}
		if st.Experience != 1 {
			t.Errorf("%s experience = %d, want 1", xuid, st.Experience)
		}
		if xuid == "owner" || xuid == "teammate1" {
			if st.Mu <= priors.Mu0 {
				t.Errorf("winner %s μ = %v, expected > %v", xuid, st.Mu, priors.Mu0)
			}
		} else {
			if st.Mu >= priors.Mu0 {
				t.Errorf("loser %s μ = %v, expected < %v", xuid, st.Mu, priors.Mu0)
			}
		}
	}

	// Re-run : watermark → 0 traités.
	processed2, err := RunLUSRV2Shadow(context.Background(), db, "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2Shadow second pass: %v", err)
	}
	if processed2 != 0 {
		t.Errorf("second pass should be no-op (watermark), got n=%d", processed2)
	}
}

func TestRunLUSRV2Shadow_RankedSkipped(t *testing.T) {
	original := os.Getenv(lusrV2EnvFlag)
	t.Cleanup(func() { _ = os.Setenv(lusrV2EnvFlag, original) })
	_ = os.Setenv(lusrV2EnvFlag, "1")

	db := openShadowTestDB(t)
	startTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	_, err := db.Exec(`INSERT INTO match_registry
		(match_id, start_time, start_time_utc, pair_name, is_ranked, is_firefight, duration_seconds)
		VALUES (?, ?, ?, 'Ranked: Arena Slayer', TRUE, FALSE, 600)`,
		"m_ranked", startTime, startTime)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	_, err = db.Exec(`INSERT INTO match_participants
		(match_id, xuid, team_id, outcome, kills, deaths) VALUES (?, ?, ?, ?, ?, ?)`,
		"m_ranked", "owner", 0, 2, 10, 8)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	processed, err := RunLUSRV2Shadow(context.Background(), db, "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2Shadow: %v", err)
	}
	if processed != 0 {
		t.Errorf("ranked match should be filtered by SQL, processed=%d", processed)
	}
}
