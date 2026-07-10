//go:build cgo

package skill

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
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

func TestIsLUSRV2Canonical(t *testing.T) {
	original := os.Getenv(lusrCanonicalEnvFlag)
	t.Cleanup(func() { _ = os.Setenv(lusrCanonicalEnvFlag, original) })

	cases := []struct {
		envVal string
		want   bool
	}{
		{"", false},     // défaut = v1 canonical
		{"LUSR", false}, // explicite v1
		{"LUSR_V2", true},
		{"lusr_v2", true}, // case insensitive (UPPER trim)
		{"  LUSR_V2  ", true},
		{"random", false},
		{"V2", false}, // doit être exactement "LUSR_V2"
	}
	for _, c := range cases {
		t.Run(c.envVal, func(t *testing.T) {
			_ = os.Setenv(lusrCanonicalEnvFlag, c.envVal)
			if got := IsLUSRV2Canonical(); got != c.want {
				t.Errorf("IsLUSRV2Canonical() with env=%q = %v, want %v", c.envVal, got, c.want)
			}
		})
	}
}

// TestDefaultOnFlags vérifie que squad offset et cross-mode coupling sont ON par
// défaut (décision produit 2026-05-28) : actifs sauf si explicitement "0"/"false"/"no".
func TestDefaultOnFlags(t *testing.T) {
	cases := []struct {
		envVal string
		want   bool
	}{
		{"", true}, // défaut = ON
		{"1", true},
		{"true", true},
		{"yes", true},
		{"random", true}, // seul "0"/"false"/"no" désactive
		{"0", false},
		{"false", false},
		{"no", false},
		{"  0  ", false}, // trim
		{"FALSE", false}, // case-insensitive
	}
	for _, fn := range []struct {
		name string
		env  string
		f    func() bool
	}{
		{"SquadOffset", lusrSquadOffsetEnvFlag, IsLUSRV2SquadOffsetEnabled},
		{"ModeCoupling", lusrModeCouplingEnvFlag, IsLUSRV2ModeCouplingEnabled},
	} {
		original := os.Getenv(fn.env)
		t.Run(fn.name, func(t *testing.T) {
			t.Cleanup(func() { _ = os.Setenv(fn.env, original) })
			for _, c := range cases {
				_ = os.Setenv(fn.env, c.envVal)
				if got := fn.f(); got != c.want {
					t.Errorf("%s avec env=%q = %v, want %v", fn.name, c.envVal, got, c.want)
				}
			}
		})
	}
}

func TestIsTeamImbalanceTooHigh(t *testing.T) {
	// Critère : |nA - nB| > 1 → trop déséquilibré → skip.
	cases := []struct {
		nA, nB int
		want   bool
		why    string
	}{
		{4, 4, false, "4v4 équilibré"},
		{1, 1, false, "1v1"},
		{4, 3, false, "diff 1 = OK (quit normal)"},
		{4, 5, false, "diff 1 = OK (late join)"},
		{2, 1, false, "diff 1"},
		{4, 2, true, "diff 2 → skip"},
		{4, 6, true, "diff 2 → skip"},
		{5, 8, true, "diff 3 → skip"},
		{3, 8, true, "diff 5 → skip"},
		{0, 4, true, "équipe vide → skip"},
		{4, 0, true, "équipe vide → skip"},
	}
	for _, c := range cases {
		got := isTeamImbalanceTooHigh(c.nA, c.nB)
		if got != c.want {
			t.Errorf("isTeamImbalanceTooHigh(%d, %d) = %v, want %v (%s)", c.nA, c.nB, got, c.want, c.why)
		}
	}
}

func TestConcurrentTeamSize(t *testing.T) {
	// present_at_beginning = TRUE → occupait un slot au coup d'envoi (compté).
	// FALSE → remplaçant (occupe un slot libéré, jamais un slot en plus).
	start := func(present bool) rosterMember {
		return rosterMember{presentAtStart: sql.NullBool{Bool: present, Valid: true}}
	}
	cases := []struct {
		name string
		team []rosterMember
		want int
	}{
		{"4v4 tous présents au coup d'envoi", []rosterMember{start(true), start(true), start(true), start(true)}, 4},
		{"4 partants + 2 remplaçants → 4 concurrents (cas du bug)", []rosterMember{start(true), start(true), start(true), start(true), start(false), start(false)}, 4},
		{"4 partants + 4 remplaçants (8 lignes) → 4 concurrents (match 0ba4aa2b)", []rosterMember{start(true), start(true), start(true), start(true), start(false), start(false), start(false), start(false)}, 4},
		{"present_at_beginning non renseigné → fallback len(team)", []rosterMember{{}, {}, {}, {}}, 4},
		{"équipe vide", nil, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := concurrentTeamSize(c.team); got != c.want {
				t.Errorf("concurrentTeamSize(%s) = %d, want %d", c.name, got, c.want)
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

func TestIdentifyPrimaryQuitter_PicksSmallestTimePlayed(t *testing.T) {
	mkQuitter := func(xuid string, timePlayed float64) rosterMember {
		return rosterMember{
			xuid:           xuid,
			leftInProgress: sql.NullBool{Bool: true, Valid: true},
			timePlayedSecs: sql.NullFloat64{Float64: timePlayed, Valid: true},
		}
	}
	mkPlayer := func(xuid string, timePlayed float64) rosterMember {
		return rosterMember{
			xuid:           xuid,
			timePlayedSecs: sql.NullFloat64{Float64: timePlayed, Valid: true},
			presentAtStart: sql.NullBool{Bool: true, Valid: true},
			presentAtEnd:   sql.NullBool{Bool: true, Valid: true},
		}
	}
	cases := []struct {
		name        string
		teamA       []rosterMember
		teamB       []rosterMember
		wantPrimary string
	}{
		{
			name:        "aucun quitter",
			teamA:       []rosterMember{mkPlayer("a1", 600)},
			teamB:       []rosterMember{mkPlayer("b1", 600)},
			wantPrimary: "",
		},
		{
			name:        "un seul quitter",
			teamA:       []rosterMember{mkQuitter("a1", 120), mkPlayer("a2", 600)},
			teamB:       []rosterMember{mkPlayer("b1", 600)},
			wantPrimary: "a1",
		},
		{
			name:        "deux quitters meme equipe — premier = plus petit temps",
			teamA:       []rosterMember{mkQuitter("a1", 90), mkQuitter("a2", 250), mkPlayer("a3", 600)},
			teamB:       []rosterMember{mkPlayer("b1", 600)},
			wantPrimary: "a1",
		},
		{
			name:        "quitters equipes adverses — choisit le premier global",
			teamA:       []rosterMember{mkQuitter("a1", 300)},
			teamB:       []rosterMember{mkQuitter("b1", 80)},
			wantPrimary: "b1",
		},
		{
			name:        "quitter sans time_played — exclu du tri",
			teamA:       []rosterMember{{xuid: "a1", leftInProgress: sql.NullBool{Bool: true, Valid: true}}},
			teamB:       []rosterMember{mkQuitter("b1", 200)},
			wantPrimary: "b1",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := identifyPrimaryQuitter(c.teamA, c.teamB)
			if got != c.wantPrimary {
				t.Errorf("identifyPrimaryQuitter = %q, want %q", got, c.wantPrimary)
			}
		})
	}
}

func TestIdentifyPrimaryQuitter_PrefersLastLeaveTimeOverTimePlayed(t *testing.T) {
	t0 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	mkQuitterTs := func(xuid string, leaveTime time.Time, timePlayed float64) rosterMember {
		return rosterMember{
			xuid:           xuid,
			leftInProgress: sql.NullBool{Bool: true, Valid: true},
			lastLeaveTime:  sql.NullTime{Time: leaveTime, Valid: true},
			timePlayedSecs: sql.NullFloat64{Float64: timePlayed, Valid: true},
		}
	}
	// Cas critique : a1 a plus petit time_played (90s) MAIS quitté plus tard
	// que b1 (leave_t = t0+5min vs t0+1min). Le primary doit être b1 grâce
	// au timestamp absolu — c'est lui qui est parti EN PREMIER.
	teamA := []rosterMember{mkQuitterTs("a1", t0.Add(5*time.Minute), 90)}
	teamB := []rosterMember{mkQuitterTs("b1", t0.Add(1*time.Minute), 300)}
	got := identifyPrimaryQuitter(teamA, teamB)
	if got != "b1" {
		t.Errorf("identifyPrimaryQuitter = %q, want b1 (leave_time prime sur time_played)", got)
	}
}

func TestIdentifyPrimaryQuitter_MixedTimestampCoverage(t *testing.T) {
	t0 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	// a1 a un timestamp (post-backfill), b1 n'en a pas (pre-backfill).
	// Le primary doit être a1 — on NE compare PAS timestamps vs time_played
	// dans un même match (cf. doc identifyPrimaryQuitter).
	withTs := rosterMember{
		xuid:           "a1",
		leftInProgress: sql.NullBool{Bool: true, Valid: true},
		lastLeaveTime:  sql.NullTime{Time: t0.Add(8 * time.Minute), Valid: true},
		timePlayedSecs: sql.NullFloat64{Float64: 480, Valid: true},
	}
	withoutTs := rosterMember{
		xuid:           "b1",
		leftInProgress: sql.NullBool{Bool: true, Valid: true},
		timePlayedSecs: sql.NullFloat64{Float64: 60, Valid: true},
	}
	got := identifyPrimaryQuitter([]rosterMember{withTs}, []rosterMember{withoutTs})
	if got != "a1" {
		t.Errorf("identifyPrimaryQuitter = %q, want a1 (timestamp prime sur fallback)", got)
	}
}

func TestScaledQuitDelta_PrimaryFullSecondaryHalf(t *testing.T) {
	const base = 2.5
	// primary = "a1" → delta plein.
	if got := scaledQuitDelta("a1", "a1", base); got != base {
		t.Errorf("primary delta = %v, want %v", got, base)
	}
	// secondaire → 50%.
	if got := scaledQuitDelta("a2", "a1", base); got != base*0.5 {
		t.Errorf("secondary delta = %v, want %v", got, base*0.5)
	}
	// primary inconnu ("") → fallback delta plein (matchs anciens sans time_played).
	if got := scaledQuitDelta("a2", "", base); got != base {
		t.Errorf("fallback (primary empty) delta = %v, want %v", got, base)
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

	if _, err := db.Exec(shadowSchemaDDL); err != nil {
		t.Fatalf("DDL: %v", err)
	}
	return db
}

// shadowSchemaDDL : schéma minimal partagé par les DB de test du shadow (in-memory
// via openShadowTestDB, fichier via openShadowTestFileDB pour le test read-only).
const shadowSchemaDDL = `
		CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP,
			start_time_utc TIMESTAMPTZ,
			real_start_time TIMESTAMP,
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
			present_at_beginning BOOLEAN,
			present_at_completion BOOLEAN,
			joined_in_progress BOOLEAN,
			left_in_progress BOOLEAN,
			first_joined_time TIMESTAMPTZ,
			last_leave_time TIMESTAMPTZ,
			time_played_seconds DOUBLE,
			PRIMARY KEY (match_id, xuid)
		);
		CREATE TABLE killer_victim_pairs (
			match_id VARCHAR,
			killer_xuid VARCHAR,
			victim_xuid VARCHAR,
			kill_count INTEGER DEFAULT 1,
			time_ms INTEGER
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
		CREATE SEQUENCE lusr_hyperparams_v2_seq START 1;
		CREATE TABLE lusr_hyperparams_v2 (
			id              BIGINT DEFAULT nextval('lusr_hyperparams_v2_seq') PRIMARY KEY,
			playlist_group  VARCHAR NOT NULL,
			name            VARCHAR NOT NULL,
			value           DOUBLE  NOT NULL,
			source          VARCHAR NOT NULL,
			written_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE VIEW lusr_hyperparams_v2_latest AS
		SELECT h.*
		FROM lusr_hyperparams_v2 h
		JOIN (
			SELECT playlist_group, name, MAX(written_at) AS max_written_at
			FROM lusr_hyperparams_v2
			GROUP BY playlist_group, name
		) m ON h.playlist_group = m.playlist_group AND h.name = m.name AND h.written_at = m.max_written_at;
		CREATE SEQUENCE player_squad_offset_seq START 1;
		CREATE TABLE player_squad_offset (
			id              BIGINT DEFAULT nextval('player_squad_offset_seq') PRIMARY KEY,
			xuid            VARCHAR NOT NULL,
			partner_xuid    VARCHAR NOT NULL,
			playlist_group  VARCHAR NOT NULL,
			offset_value    DOUBLE  NOT NULL,
			match_count     INTEGER NOT NULL DEFAULT 0,
			source          VARCHAR NOT NULL,
			written_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE VIEW player_squad_offset_latest AS
		SELECT o.*
		FROM player_squad_offset o
		JOIN (
			SELECT xuid, partner_xuid, playlist_group, MAX(written_at) AS max_written_at
			FROM player_squad_offset
			GROUP BY xuid, partner_xuid, playlist_group
		) m ON o.xuid = m.xuid AND o.partner_xuid = m.partner_xuid
		     AND o.playlist_group = m.playlist_group AND o.written_at = m.max_written_at;
	`

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
	n, err := RunLUSRV2Shadow(context.Background(), nil, newPinnedSharedAccessor(db), "anyxuid")
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

	processed, err := RunLUSRV2Shadow(context.Background(), nil, newPinnedSharedAccessor(db), "owner")
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
	processed2, err := RunLUSRV2Shadow(context.Background(), nil, newPinnedSharedAccessor(db), "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2Shadow second pass: %v", err)
	}
	if processed2 != 0 {
		t.Errorf("second pass should be no-op (watermark), got n=%d", processed2)
	}
}

// TestRunLUSRV2ShadowOwnerOnly_TeammateDoesNotBlockOwner : fix autonomie live
// (2026-06-08). En owner-only, traiter un coéquipier en premier ne doit PAS
// avancer le watermark du owner ni le faire sauter — chaque joueur obtient sa
// couverture seul. (Avec l'ancien persist 2-équipes, le owner aurait été skippé
// car le run du coéquipier aurait avancé son watermark.)
func TestRunLUSRV2ShadowOwnerOnly_TeammateDoesNotBlockOwner(t *testing.T) {
	original := os.Getenv(lusrV2EnvFlag)
	t.Cleanup(func() { _ = os.Setenv(lusrV2EnvFlag, original) })
	_ = os.Setenv(lusrV2EnvFlag, "1")

	db := openShadowTestDB(t)
	startTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	if _, err := db.Exec(`INSERT INTO match_registry
		(match_id, start_time, start_time_utc, pair_name, is_ranked, is_firefight, duration_seconds)
		VALUES (?, ?, ?, 'Slayer', FALSE, FALSE, 600)`, "m1", startTime, startTime); err != nil {
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
		if _, err := db.Exec(`INSERT INTO match_participants
			(match_id, xuid, team_id, outcome, kills, deaths) VALUES (?, ?, ?, ?, ?, ?)`,
			"m1", q.xuid, q.team, q.outcome, q.kills, q.deaths); err != nil {
			t.Fatalf("insert participant: %v", err)
		}
	}

	repo := duckdb.NewSkillV2Repo(db)
	ctx := context.Background()

	// 1) Coéquipier traité en premier (owner-only).
	n1, err := RunLUSRV2ShadowOwnerOnly(ctx, nil, newPinnedSharedAccessor(db), "teammate1")
	if err != nil {
		t.Fatalf("owner-only teammate1: %v", err)
	}
	if n1 != 1 {
		t.Fatalf("teammate1 processed = %d, want 1", n1)
	}
	// Le owner ne doit PAS avoir d'état : owner-only n'a persisté que teammate1,
	// donc le watermark du owner n'a pas bougé.
	if st, _ := repo.LoadState(ctx, "owner", "arena_slayer"); st != nil {
		t.Errorf("owner ne devrait pas avoir d'état après le run owner-only de teammate1, got μ=%v", st.Mu)
	}

	// 2) Owner traité ensuite : NE doit PAS être bloqué (autonomie).
	n2, err := RunLUSRV2ShadowOwnerOnly(ctx, nil, newPinnedSharedAccessor(db), "owner")
	if err != nil {
		t.Fatalf("owner-only owner: %v", err)
	}
	if n2 != 1 {
		t.Fatalf("owner processed = %d, want 1 (coéquipier ne doit pas bloquer)", n2)
	}
	if st, _ := repo.LoadState(ctx, "owner", "arena_slayer"); st == nil {
		t.Error("owner devrait avoir un état après son propre run")
	}
}

// TestRunLUSRV2Shadow_Canonical_WritesLegacyLUSRRow vérifie la Stratégie C :
// quand LEVELUP_LUSR_CANONICAL=LUSR_V2, le shadow runner écrit dans le
// playerDB match_skill_rank (slot rating_type='LUSR') en plus de
// player_skill_state_v2.
func TestRunLUSRV2Shadow_Canonical_WritesLegacyLUSRRow(t *testing.T) {
	origEnabled := os.Getenv(lusrV2EnvFlag)
	origCanonical := os.Getenv(lusrCanonicalEnvFlag)
	t.Cleanup(func() {
		_ = os.Setenv(lusrV2EnvFlag, origEnabled)
		_ = os.Setenv(lusrCanonicalEnvFlag, origCanonical)
	})
	_ = os.Setenv(lusrV2EnvFlag, "1")
	_ = os.Setenv(lusrCanonicalEnvFlag, "LUSR_V2")

	sharedDB := openShadowTestDB(t)
	playerDB := openCanonicalPlayerTestDB(t)

	// Insert un match social 2v2.
	startTime := time.Date(2025, 2, 1, 14, 0, 0, 0, time.UTC)
	_, err := sharedDB.Exec(`INSERT INTO match_registry
		(match_id, start_time, start_time_utc, pair_name, is_ranked, is_firefight, duration_seconds)
		VALUES (?, ?, ?, 'Slayer', FALSE, FALSE, 600)`,
		"m_canon", startTime, startTime)
	if err != nil {
		t.Fatalf("insert match_registry: %v", err)
	}
	for _, q := range []struct {
		xuid          string
		team, outcome int
		kills, deaths int
	}{
		{"owner", 0, 2, 18, 6}, {"teammate", 0, 2, 12, 9},
		{"opp1", 1, 3, 7, 14}, {"opp2", 1, 3, 8, 14},
	} {
		_, err := sharedDB.Exec(`INSERT INTO match_participants
			(match_id, xuid, team_id, outcome, kills, deaths) VALUES (?, ?, ?, ?, ?, ?)`,
			"m_canon", q.xuid, q.team, q.outcome, q.kills, q.deaths)
		if err != nil {
			t.Fatalf("insert participant: %v", err)
		}
	}

	processed, err := RunLUSRV2Shadow(context.Background(), playerDB, newPinnedSharedAccessor(sharedDB), "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2Shadow: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}

	// Vérifie qu'une row existe pour rating_type='LUSR' ET rating_type='LUSR_V2'
	// (Stratégie C dual-row).
	var countLUSR, countV2 int
	if err := playerDB.QueryRow(`SELECT
		COUNT(*) FILTER (WHERE rating_type = 'LUSR'),
		COUNT(*) FILTER (WHERE rating_type = 'LUSR_V2')
		FROM match_skill_rank WHERE match_id = ?`, "m_canon").
		Scan(&countLUSR, &countV2); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if countLUSR != 1 || countV2 != 1 {
		t.Errorf("dual-row attendu : LUSR=%d LUSR_V2=%d (want 1+1)", countLUSR, countV2)
	}

	// Vérifie que la row LUSR a le bon mapping.
	var ratingType, tier string
	var ratingValue float64
	err = playerDB.QueryRow(`SELECT rating_type, rating_value, tier
		FROM match_skill_rank
		WHERE match_id = ? AND rating_type = 'LUSR'`, "m_canon").
		Scan(&ratingType, &ratingValue, &tier)
	if err != nil {
		t.Fatalf("no LUSR row found in match_skill_rank: %v", err)
	}
	if ratingValue < 1000 || ratingValue > 2500 {
		t.Errorf("rating_value = %v hors plage v1 [1000, 2500]", ratingValue)
	}
	t.Logf("OK : dual-row owner posterior écrit LUSR=%d LUSR_V2=%d rating=%v tier=%s",
		countLUSR, countV2, ratingValue, tier)
}

// openCanonicalPlayerTestDB : DuckDB :memory: avec match_skill_rank schema
// (équivalent au schéma append-only Phase 2.F).
func openCanonicalPlayerTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb player: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	const ddl = `
		CREATE SEQUENCE match_skill_rank_id_seq;
		CREATE TABLE match_skill_rank (
			id              BIGINT DEFAULT nextval('match_skill_rank_id_seq') PRIMARY KEY,
			match_id        VARCHAR NOT NULL,
			rating_type     VARCHAR NOT NULL,
			rating_value    FLOAT,
			rating_deviation FLOAT,
			tier            VARCHAR,
			tier_fr         VARCHAR,
			sub_tier        SMALLINT,
			tier_label      VARCHAR,
			rating_delta    FLOAT,
			playlist_group  VARCHAR,
			expected_win_prob FLOAT,
			start_time      TIMESTAMP,
			written_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE VIEW match_skill_rank_latest AS
		  SELECT * FROM match_skill_rank
		  QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, rating_type
		                             ORDER BY written_at DESC) = 1;
	`
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("DDL match_skill_rank: %v", err)
	}
	return db
}

// TestRunDualRowSentinel_DetectsInconsistencies vérifie que la sentinelle
// signale correctement les états :
//   - LUSR seul   → ignoré (héritage v1 légitime)
//   - LUSR + V2   → both_present (cas nominal)
//   - LUSR_V2 seul → inconsistance (incrémente expvar)
func TestRunDualRowSentinel_DetectsInconsistencies(t *testing.T) {
	db := openCanonicalPlayerTestDB(t)

	// Reset compteur pour assertion exacte.
	dualRowInconsistencies.Set(0)

	// Match 1 : LUSR seul (héritage v1).
	_, err := db.Exec(`INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, playlist_group)
		VALUES ('m_legacy', 'LUSR', 1500, 'arena_slayer')`)
	if err != nil {
		t.Fatalf("insert m_legacy: %v", err)
	}
	// Match 2 : dual-row correct.
	_, err = db.Exec(`INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, playlist_group) VALUES
		('m_both', 'LUSR', 1700, 'arena_slayer'),
		('m_both', 'LUSR_V2', 1700, 'arena_slayer')`)
	if err != nil {
		t.Fatalf("insert m_both: %v", err)
	}
	// Match 3 : LUSR_V2 seul (bug — devrait avoir un LUSR aussi).
	_, err = db.Exec(`INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, playlist_group)
		VALUES ('m_orphan', 'LUSR_V2', 1900, 'arena_slayer')`)
	if err != nil {
		t.Fatalf("insert m_orphan: %v", err)
	}

	report, err := RunDualRowSentinel(context.Background(), db)
	if err != nil {
		t.Fatalf("RunDualRowSentinel: %v", err)
	}
	if report.MatchesScanned != 3 {
		t.Errorf("MatchesScanned = %d, want 3", report.MatchesScanned)
	}
	if report.OnlyLUSR != 1 {
		t.Errorf("OnlyLUSR = %d, want 1", report.OnlyLUSR)
	}
	if report.BothPresent != 1 {
		t.Errorf("BothPresent = %d, want 1", report.BothPresent)
	}
	if report.OnlyLUSRV2 != 1 {
		t.Errorf("OnlyLUSRV2 = %d, want 1 (bug à signaler)", report.OnlyLUSRV2)
	}
	if len(report.SampleInconsistent) != 1 || report.SampleInconsistent[0] != "m_orphan" {
		t.Errorf("SampleInconsistent = %v, want [m_orphan]", report.SampleInconsistent)
	}
	if got := dualRowInconsistencies.Value(); got != 1 {
		t.Errorf("expvar dualRowInconsistencies = %d, want 1", got)
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
	processed, err := RunLUSRV2Shadow(context.Background(), nil, newPinnedSharedAccessor(db), "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2Shadow: %v", err)
	}
	if processed != 0 {
		t.Errorf("ranked match should be filtered by SQL, processed=%d", processed)
	}
}

// TestRunLUSRV2Shadow_Phase4_CrossModeLeak vérifie qu'avec LEVELUP_LUSR_V2_MODE_COUPLING=1
// activé, un match dans le mode "arena_slayer" leak son delta vers les autres
// modes (e.g., "btb") du même joueur, capé à w_d · delta.
func TestRunLUSRV2Shadow_Phase4_CrossModeLeak(t *testing.T) {
	origEnabled := os.Getenv(lusrV2EnvFlag)
	origCoupling := os.Getenv(lusrModeCouplingEnvFlag)
	t.Cleanup(func() {
		_ = os.Setenv(lusrV2EnvFlag, origEnabled)
		_ = os.Setenv(lusrModeCouplingEnvFlag, origCoupling)
	})
	_ = os.Setenv(lusrV2EnvFlag, "1")
	_ = os.Setenv(lusrModeCouplingEnvFlag, "1")

	db := openShadowTestDB(t)
	repo := duckdb.NewSkillV2Repo(db)
	ctx := context.Background()

	// Seed un état pré-existant pour owner dans le mode "btb" avant qu'il
	// joue un match dans "arena_slayer". μ = 24 (déjà un peu en-dessous du prior).
	pretime := time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC)
	if err := repo.UpsertState(ctx, domain.SkillV2State{
		XUID: "owner", PlaylistGroup: "btb",
		Mu: 24.0, Sigma: 6.0, Experience: 5,
		LastMatchAt: &pretime,
	}); err != nil {
		t.Fatalf("seed btb state: %v", err)
	}

	// Match 2v2 slayer où owner gagne avec gros stats (carry).
	startTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	_, err := db.Exec(`INSERT INTO match_registry
		(match_id, start_time, start_time_utc, pair_name, is_ranked, is_firefight, duration_seconds)
		VALUES (?, ?, ?, 'Slayer', FALSE, FALSE, 600)`,
		"m_phase4", startTime, startTime)
	if err != nil {
		t.Fatalf("insert match_registry: %v", err)
	}
	for _, q := range []struct {
		xuid          string
		team, outcome int
		kills, deaths int
	}{
		{"owner", 0, 2, 25, 4}, {"teammate", 0, 2, 10, 9},
		{"opp1", 1, 3, 6, 16}, {"opp2", 1, 3, 8, 14},
	} {
		_, err := db.Exec(`INSERT INTO match_participants
			(match_id, xuid, team_id, outcome, kills, deaths) VALUES (?, ?, ?, ?, ?, ?)`,
			"m_phase4", q.xuid, q.team, q.outcome, q.kills, q.deaths)
		if err != nil {
			t.Fatalf("insert participant: %v", err)
		}
	}

	if _, err := RunLUSRV2Shadow(ctx, nil, newPinnedSharedAccessor(db), "owner"); err != nil {
		t.Fatalf("RunLUSRV2Shadow: %v", err)
	}

	// État slayer écrit (mode primaire).
	slayer, err := repo.LoadState(ctx, "owner", "arena_slayer")
	if err != nil || slayer == nil {
		t.Fatalf("slayer state non créé: %v", err)
	}
	deltaSlayer := slayer.Mu - 25.0 // prior μ_0 = 25
	if deltaSlayer <= 0 {
		t.Fatalf("expected positive delta slayer (carry win), got %v", deltaSlayer)
	}

	// État btb mis à jour par leak. Δ_btb attendu = 0.3 · delta_slayer.
	btb, err := repo.LoadState(ctx, "owner", "btb")
	if err != nil || btb == nil {
		t.Fatalf("btb state introuvable après leak: %v", err)
	}
	expectedBtbMu := 24.0 + skillv2.DefaultModeCouplingWeight*deltaSlayer
	if diff := btb.Mu - expectedBtbMu; diff > 0.01 || diff < -0.01 {
		t.Errorf("btb μ after leak = %v, want %v (= 24.0 + 0.3 · %v)",
			btb.Mu, expectedBtbMu, deltaSlayer)
	}
	// σ inchangé.
	if btb.Sigma != 6.0 {
		t.Errorf("btb σ = %v, want 6.0 (leak ne modifie pas σ)", btb.Sigma)
	}
}

// TestRunLUSRV2Shadow_Phase4_ExplicitlyOff vérifie qu'avec le flag explicitement
// désactivé ("0"), les autres modes ne sont PAS modifiés. (Le cross-mode leak est
// désormais ON par défaut — décision produit 2026-05-28.)
func TestRunLUSRV2Shadow_Phase4_ExplicitlyOff(t *testing.T) {
	origEnabled := os.Getenv(lusrV2EnvFlag)
	origCoupling := os.Getenv(lusrModeCouplingEnvFlag)
	t.Cleanup(func() {
		_ = os.Setenv(lusrV2EnvFlag, origEnabled)
		_ = os.Setenv(lusrModeCouplingEnvFlag, origCoupling)
	})
	_ = os.Setenv(lusrV2EnvFlag, "1")
	_ = os.Setenv(lusrModeCouplingEnvFlag, "0") // explicitement désactivé

	db := openShadowTestDB(t)
	repo := duckdb.NewSkillV2Repo(db)
	ctx := context.Background()

	pretime := time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC)
	if err := repo.UpsertState(ctx, domain.SkillV2State{
		XUID: "owner", PlaylistGroup: "btb",
		Mu: 24.0, Sigma: 6.0, Experience: 5,
		LastMatchAt: &pretime,
	}); err != nil {
		t.Fatalf("seed btb: %v", err)
	}

	startTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	_, err := db.Exec(`INSERT INTO match_registry
		(match_id, start_time, start_time_utc, pair_name, is_ranked, is_firefight, duration_seconds)
		VALUES (?, ?, ?, 'Slayer', FALSE, FALSE, 600)`,
		"m_phase4_off", startTime, startTime)
	if err != nil {
		t.Fatalf("insert match: %v", err)
	}
	for _, q := range []struct {
		xuid          string
		team, outcome int
		kills, deaths int
	}{
		{"owner", 0, 2, 25, 4}, {"teammate", 0, 2, 10, 9},
		{"opp1", 1, 3, 6, 16}, {"opp2", 1, 3, 8, 14},
	} {
		_, err := db.Exec(`INSERT INTO match_participants
			(match_id, xuid, team_id, outcome, kills, deaths) VALUES (?, ?, ?, ?, ?, ?)`,
			"m_phase4_off", q.xuid, q.team, q.outcome, q.kills, q.deaths)
		if err != nil {
			t.Fatalf("insert participant: %v", err)
		}
	}
	if _, err := RunLUSRV2Shadow(ctx, nil, newPinnedSharedAccessor(db), "owner"); err != nil {
		t.Fatalf("RunLUSRV2Shadow: %v", err)
	}

	btb, err := repo.LoadState(ctx, "owner", "btb")
	if err != nil || btb == nil {
		t.Fatalf("btb state: %v", err)
	}
	if btb.Mu != 24.0 {
		t.Errorf("btb μ = %v, want 24.0 (flag off, no leak)", btb.Mu)
	}
}

// seedCanonical2v2 insère un match social 2v2 (owner+teammate vs opp1+opp2)
// dans le sharedDB de test. Utilisé par les tests Sprint 1.A.
// seedH5Match : match 2v2 façon Halo 5 — start_time_utc renseigné, start_time
// (legacy) NULL, pair_name VIDE (h5 n'en a pas). Vérifie le chemin title-generic.
func seedH5Match(t *testing.T, sharedDB *sql.DB, matchID string, start time.Time) {
	t.Helper()
	if _, err := sharedDB.Exec(`INSERT INTO match_registry
		(match_id, start_time, start_time_utc, pair_name, is_ranked, is_firefight, duration_seconds)
		VALUES (?, NULL, ?, '', FALSE, FALSE, 600)`, matchID, start); err != nil {
		t.Fatalf("insert match_registry h5: %v", err)
	}
	for _, q := range []struct {
		xuid          string
		team, outcome int
		kills, deaths int
	}{
		{"owner", 0, 2, 18, 6}, {"teammate", 0, 2, 12, 9},
		{"opp1", 1, 3, 7, 14}, {"opp2", 1, 3, 8, 14},
	} {
		if _, err := sharedDB.Exec(`INSERT INTO match_participants
			(match_id, xuid, team_id, outcome, kills, deaths) VALUES (?, ?, ?, ?, ?, ?)`,
			matchID, q.xuid, q.team, q.outcome, q.kills, q.deaths); err != nil {
			t.Fatalf("insert participant h5: %v", err)
		}
	}
}

// TestRunLUSRV2Shadow_Halo5_TitleAwareChain — LUSR v2 pour Halo 5 : un match
// canonique (start_time_utc renseigné, start_time legacy NULL, pas de pair_name)
// DOIT être traité (filtres start_time title-generic) et l'état écrit sous la
// chaîne du classifier TITLE-AWARE (h5_arena), PAS le bucket Infinite arena_slayer.
// Prouve que la v2 (k/d/a/outcome, sans MMR) tourne pour h5.
func TestRunLUSRV2Shadow_Halo5_TitleAwareChain(t *testing.T) {
	origEnabled := os.Getenv(lusrV2EnvFlag)
	t.Cleanup(func() { _ = os.Setenv(lusrV2EnvFlag, origEnabled) })
	_ = os.Setenv(lusrV2EnvFlag, "1")

	// Classifier h5 dédié (chaîne unique). Nettoyé après le test (test interne →
	// accès direct à la map du seam).
	SetLUSRChainClassifierForTitle("halo_5", func(string) string { return "h5_arena" })
	t.Cleanup(func() { delete(lusrChainClassifiersByTitle, "halo_5") })

	// Le registre runtime par défaut ne connaît qu'Infinite hors boot : on
	// l'enrichit d'un descripteur h5 (CapLUSR) pour le test (swap + restore).
	oldReg := title.DefaultRegistry()
	reg := title.NewRegistry()
	reg.Register(&title.TitleDescriptor{
		Slug: "halo_5", Name: "Halo 5", Status: title.StatusActive,
		Capabilities: []title.Capability{title.CapLUSR},
	})
	title.SetDefaultRegistry(reg)
	t.Cleanup(func() { title.SetDefaultRegistry(oldReg) })

	sharedDB := openShadowTestDB(t)
	seedH5Match(t, sharedDB, "h5_m1", time.Date(2025, 11, 1, 14, 0, 0, 0, time.UTC))

	ctx := ctxkeys.WithTitleSlug(context.Background(), "halo_5")
	processed, err := RunLUSRV2ShadowOwnerOnly(ctx, nil, newPinnedSharedAccessor(sharedDB), "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2ShadowOwnerOnly: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1 (le match h5 doit passer les filtres start_time_utc)", processed)
	}

	var group string
	var mu, sigma float64
	if err := sharedDB.QueryRow(
		`SELECT playlist_group, mu, sigma FROM player_skill_state_v2_latest WHERE xuid = 'owner'`,
	).Scan(&group, &mu, &sigma); err != nil {
		t.Fatalf("read state: %v", err)
	}
	if group != "h5_arena" {
		t.Errorf("playlist_group = %q, want h5_arena (classifier title-aware, pas le bucket Infinite)", group)
	}
	if mu <= 0 || sigma <= 0 {
		t.Errorf("μ=%v σ=%v, attendu un rating valide", mu, sigma)
	}
}

func seedCanonical2v2(t *testing.T, sharedDB *sql.DB, matchID string, start time.Time) {
	t.Helper()
	if _, err := sharedDB.Exec(`INSERT INTO match_registry
		(match_id, start_time, start_time_utc, pair_name, is_ranked, is_firefight, duration_seconds)
		VALUES (?, ?, ?, 'Slayer', FALSE, FALSE, 600)`, matchID, start, start); err != nil {
		t.Fatalf("insert match_registry: %v", err)
	}
	for _, q := range []struct {
		xuid          string
		team, outcome int
		kills, deaths int
	}{
		{"owner", 0, 2, 18, 6}, {"teammate", 0, 2, 12, 9},
		{"opp1", 1, 3, 7, 14}, {"opp2", 1, 3, 8, 14},
	} {
		if _, err := sharedDB.Exec(`INSERT INTO match_participants
			(match_id, xuid, team_id, outcome, kills, deaths) VALUES (?, ?, ?, ?, ?, ?)`,
			matchID, q.xuid, q.team, q.outcome, q.kills, q.deaths); err != nil {
			t.Fatalf("insert participant: %v", err)
		}
	}
}

// TestRunLUSRV2Shadow_Canonical_StoresExpectedWinProb (Sprint 1.A) : la row
// canonical LUSR doit porter un expected_win_prob dans [0,1].
func TestRunLUSRV2Shadow_Canonical_StoresExpectedWinProb(t *testing.T) {
	origEnabled := os.Getenv(lusrV2EnvFlag)
	origCanonical := os.Getenv(lusrCanonicalEnvFlag)
	t.Cleanup(func() {
		_ = os.Setenv(lusrV2EnvFlag, origEnabled)
		_ = os.Setenv(lusrCanonicalEnvFlag, origCanonical)
	})
	_ = os.Setenv(lusrV2EnvFlag, "1")
	_ = os.Setenv(lusrCanonicalEnvFlag, "LUSR_V2")

	sharedDB := openShadowTestDB(t)
	playerDB := openCanonicalPlayerTestDB(t)
	seedCanonical2v2(t, sharedDB, "m_winprob", time.Date(2025, 3, 1, 14, 0, 0, 0, time.UTC))

	if _, err := RunLUSRV2Shadow(context.Background(), playerDB, newPinnedSharedAccessor(sharedDB), "owner"); err != nil {
		t.Fatalf("RunLUSRV2Shadow: %v", err)
	}

	var winProb sql.NullFloat64
	if err := playerDB.QueryRow(`SELECT expected_win_prob FROM match_skill_rank
		WHERE match_id = ? AND rating_type = 'LUSR'`, "m_winprob").Scan(&winProb); err != nil {
		t.Fatalf("read expected_win_prob: %v", err)
	}
	if !winProb.Valid {
		t.Fatal("expected_win_prob NULL, attendu une valeur")
	}
	if winProb.Float64 < 0 || winProb.Float64 > 1 {
		t.Errorf("expected_win_prob = %v hors [0,1]", winProb.Float64)
	}
}

// TestRunLUSRV2Shadow_Canonical_FirstMatchFallback (Sprint 1.A) : si aucun
// joueur n'a d'historique, tous sont seedés au prior (μ=25) → équipes
// équilibrées → proba ≈ 0.5, et aucune panic (fallback priors testé).
func TestRunLUSRV2Shadow_Canonical_FirstMatchFallback(t *testing.T) {
	origEnabled := os.Getenv(lusrV2EnvFlag)
	origCanonical := os.Getenv(lusrCanonicalEnvFlag)
	t.Cleanup(func() {
		_ = os.Setenv(lusrV2EnvFlag, origEnabled)
		_ = os.Setenv(lusrCanonicalEnvFlag, origCanonical)
	})
	_ = os.Setenv(lusrV2EnvFlag, "1")
	_ = os.Setenv(lusrCanonicalEnvFlag, "LUSR_V2")

	sharedDB := openShadowTestDB(t)
	playerDB := openCanonicalPlayerTestDB(t)
	seedCanonical2v2(t, sharedDB, "m_firstmatch", time.Date(2025, 3, 2, 14, 0, 0, 0, time.UTC))

	if _, err := RunLUSRV2Shadow(context.Background(), playerDB, newPinnedSharedAccessor(sharedDB), "owner"); err != nil {
		t.Fatalf("RunLUSRV2Shadow: %v", err)
	}

	var winProb sql.NullFloat64
	if err := playerDB.QueryRow(`SELECT expected_win_prob FROM match_skill_rank
		WHERE match_id = ? AND rating_type = 'LUSR'`, "m_firstmatch").Scan(&winProb); err != nil {
		t.Fatalf("read expected_win_prob: %v", err)
	}
	if !winProb.Valid {
		t.Fatal("expected_win_prob NULL, attendu une valeur")
	}
	// Équipes entièrement seedées au prior → match équilibré → proche de 0.5.
	if d := winProb.Float64 - 0.5; d > 0.2 || d < -0.2 {
		t.Errorf("first-match balanced : expected_win_prob = %v, attendu ≈ 0.5 (±0.2)", winProb.Float64)
	}
}

// TestRunLUSRV2Shadow_UsesEmpiricalDrawProb (Sprint 1.B) prouve que la draw
// probability empirique seedée dans lusr_hyperparams_v2 est réellement lue par
// le runner. Sur un match nul 2v2 parfaitement symétrique SANS counts (pure
// TrueSkill), μ reste à 25 par symétrie mais σ dépend de la marge de draw ε,
// donc de DrawProbability. Un draw "moins surprenant" (proba haute) réduit moins
// σ → σ plus grand. On compare deux runs identiques : seul l'hyperparam diffère.
func TestRunLUSRV2Shadow_UsesEmpiricalDrawProb(t *testing.T) {
	original := os.Getenv(lusrV2EnvFlag)
	t.Cleanup(func() { _ = os.Setenv(lusrV2EnvFlag, original) })
	_ = os.Setenv(lusrV2EnvFlag, "1")

	runDrawMatch := func(seedHighDrawProb bool) (mu, sigma float64) {
		db := openShadowTestDB(t)
		if seedHighDrawProb {
			if _, err := db.Exec(`INSERT INTO lusr_hyperparams_v2
				(playlist_group, name, value, source)
				VALUES ('arena_slayer', 'draw_probability_empirical', 0.5, 'test')`); err != nil {
				t.Fatalf("seed hyperparam: %v", err)
			}
		}
		start := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
		if _, err := db.Exec(`INSERT INTO match_registry
			(match_id, start_time, start_time_utc, pair_name, is_ranked, is_firefight, duration_seconds)
			VALUES ('m_draw', ?, ?, 'Slayer', FALSE, FALSE, 600)`, start, start); err != nil {
			t.Fatalf("insert match_registry: %v", err)
		}
		// Match nul (outcome=1 partout), SANS kills/deaths → counts nil → pure TS.
		for _, q := range []struct {
			xuid string
			team int
		}{{"owner", 0}, {"teammate", 0}, {"opp1", 1}, {"opp2", 1}} {
			if _, err := db.Exec(`INSERT INTO match_participants
				(match_id, xuid, team_id, outcome) VALUES ('m_draw', ?, ?, 1)`,
				q.xuid, q.team); err != nil {
				t.Fatalf("insert participant: %v", err)
			}
		}
		if _, err := RunLUSRV2Shadow(context.Background(), nil, newPinnedSharedAccessor(db), "owner"); err != nil {
			t.Fatalf("RunLUSRV2Shadow: %v", err)
		}
		st, err := duckdb.NewSkillV2Repo(db).LoadState(context.Background(), "owner", "arena_slayer")
		if err != nil || st == nil {
			t.Fatalf("LoadState owner: %v", err)
		}
		return st.Mu, st.Sigma
	}

	_, sigmaDefault := runDrawMatch(false)
	_, sigmaSeeded := runDrawMatch(true)

	if sigmaSeeded <= sigmaDefault {
		t.Errorf("draw_probability empirique haute → draw moins surprenant → σ doit rester PLUS grand : seeded σ=%v doit > default σ=%v",
			sigmaSeeded, sigmaDefault)
	}
}

// TestRunLUSRV2Shadow_AppliesSquadOffset (Sprint 1.C) prouve la correction
// d'escouade : avec un offset de synergie positif owner↔teammate, quand le
// squad GAGNE, le μ individuel du owner monte MOINS que sans offset (la victoire
// est en partie attribuée à la synergie, pas au skill individuel → anti-inflation).
// Gated : flag LEVELUP_LUSR_V2_SQUAD_OFFSET=1.
func TestRunLUSRV2Shadow_AppliesSquadOffset(t *testing.T) {
	origEnabled := os.Getenv(lusrV2EnvFlag)
	origSquad := os.Getenv(lusrSquadOffsetEnvFlag)
	t.Cleanup(func() {
		_ = os.Setenv(lusrV2EnvFlag, origEnabled)
		_ = os.Setenv(lusrSquadOffsetEnvFlag, origSquad)
	})
	_ = os.Setenv(lusrV2EnvFlag, "1")
	_ = os.Setenv(lusrSquadOffsetEnvFlag, "1")

	// runWin joue un match 2v2 où owner+teammate GAGNENT (sans counts) et
	// retourne le μ posterior du owner. Si withOffset, seede +1.5 owner↔teammate.
	runWin := func(withOffset bool) float64 {
		db := openShadowTestDB(t)
		if withOffset {
			for _, p := range [][2]string{{"owner", "teammate"}, {"teammate", "owner"}} {
				if _, err := db.Exec(`INSERT INTO player_squad_offset
					(xuid, partner_xuid, playlist_group, offset_value, match_count, source)
					VALUES (?, ?, 'arena_slayer', 1.5, 20, 'test')`, p[0], p[1]); err != nil {
					t.Fatalf("seed squad offset: %v", err)
				}
			}
		}
		start := time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC)
		if _, err := db.Exec(`INSERT INTO match_registry
			(match_id, start_time, start_time_utc, pair_name, is_ranked, is_firefight, duration_seconds)
			VALUES ('m_squad', ?, ?, 'Slayer', FALSE, FALSE, 600)`, start, start); err != nil {
			t.Fatalf("insert match_registry: %v", err)
		}
		// owner+teammate gagnent (outcome 2), opp1+opp2 perdent (3). Pas de counts.
		for _, q := range []struct {
			xuid          string
			team, outcome int
		}{{"owner", 0, 2}, {"teammate", 0, 2}, {"opp1", 1, 3}, {"opp2", 1, 3}} {
			if _, err := db.Exec(`INSERT INTO match_participants
				(match_id, xuid, team_id, outcome) VALUES ('m_squad', ?, ?, ?)`,
				q.xuid, q.team, q.outcome); err != nil {
				t.Fatalf("insert participant: %v", err)
			}
		}
		if _, err := RunLUSRV2Shadow(context.Background(), nil, newPinnedSharedAccessor(db), "owner"); err != nil {
			t.Fatalf("RunLUSRV2Shadow: %v", err)
		}
		st, err := duckdb.NewSkillV2Repo(db).LoadState(context.Background(), "owner", "arena_slayer")
		if err != nil || st == nil {
			t.Fatalf("LoadState owner: %v", err)
		}
		return st.Mu
	}

	muNoOffset := runWin(false)
	muWithOffset := runWin(true)

	// Les deux montent (victoire) au-dessus du prior 25.
	if muNoOffset <= 25.0 {
		t.Fatalf("victoire sans offset : μ owner = %v, attendu > 25", muNoOffset)
	}
	// Anti-inflation : avec offset positif, μ individuel monte MOINS.
	if !(muWithOffset < muNoOffset) {
		t.Errorf("offset squad +1.5 → μ owner doit monter MOINS : withOffset=%v doit < noOffset=%v",
			muWithOffset, muNoOffset)
	}
}

// TestRunLUSRV2Shadow_Canonical_RatingDelta (Sprint 3.B) : sur 2 matchs
// successifs du même groupe, la 1re row LUSR a rating_delta NULL (pas de
// précédent) et la 2e a un rating_delta non-nul (= rating - rating précédent).
func TestRunLUSRV2Shadow_Canonical_RatingDelta(t *testing.T) {
	origEnabled := os.Getenv(lusrV2EnvFlag)
	origCanonical := os.Getenv(lusrCanonicalEnvFlag)
	t.Cleanup(func() {
		_ = os.Setenv(lusrV2EnvFlag, origEnabled)
		_ = os.Setenv(lusrCanonicalEnvFlag, origCanonical)
	})
	_ = os.Setenv(lusrV2EnvFlag, "1")
	_ = os.Setenv(lusrCanonicalEnvFlag, "LUSR_V2")

	sharedDB := openShadowTestDB(t)
	playerDB := openCanonicalPlayerTestDB(t)
	// 2 matchs successifs (m1 puis m2), même groupe arena_slayer.
	seedCanonical2v2(t, sharedDB, "m1", time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))
	seedCanonical2v2(t, sharedDB, "m2", time.Date(2025, 6, 1, 13, 0, 0, 0, time.UTC))

	if _, err := RunLUSRV2Shadow(context.Background(), playerDB, newPinnedSharedAccessor(sharedDB), "owner"); err != nil {
		t.Fatalf("RunLUSRV2Shadow: %v", err)
	}

	readDelta := func(matchID string) sql.NullFloat64 {
		var d sql.NullFloat64
		if err := playerDB.QueryRow(`SELECT rating_delta FROM match_skill_rank
			WHERE match_id = ? AND rating_type = 'LUSR'`, matchID).Scan(&d); err != nil {
			t.Fatalf("read rating_delta %s: %v", matchID, err)
		}
		return d
	}
	if d1 := readDelta("m1"); d1.Valid {
		t.Errorf("m1 (1er match) : rating_delta = %v, attendu NULL (pas de précédent)", d1.Float64)
	}
	if d2 := readDelta("m2"); !d2.Valid {
		t.Error("m2 (2e match) : rating_delta NULL, attendu une valeur (= rating - précédent)")
	}
}

// TestRunLUSRV2Shadow_QuitContext_LeadingAtQuit (Sprint 2.A) : un joueur qui
// quitte alors que son équipe MENAIT (mais perd le match) doit subir la pénalité
// FORTE (unrelated) — pas la pénalité modérée du fallback outcome-final (perte).
// On compare avec timeline de frags (menait au quit) vs sans timeline (fallback).
func TestRunLUSRV2Shadow_QuitContext_LeadingAtQuit(t *testing.T) {
	orig := os.Getenv(lusrV2EnvFlag)
	t.Cleanup(func() { _ = os.Setenv(lusrV2EnvFlag, orig) })
	_ = os.Setenv(lusrV2EnvFlag, "1")

	start := time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC)
	quitTime := start.Add(60 * time.Second) // owner part à 60s

	runQuit := func(withTimeline bool) float64 {
		db := openShadowTestDB(t)
		if _, err := db.Exec(`INSERT INTO match_registry
			(match_id, start_time, start_time_utc, pair_name, is_ranked, is_firefight, duration_seconds)
			VALUES ('m_quit', ?, ?, 'Slayer', FALSE, FALSE, 600)`, start, start); err != nil {
			t.Fatalf("insert match_registry: %v", err)
		}
		// owner (team0) quitte ; team0 PERD (outcome 3) vs team1 (2). Pas de counts.
		if _, err := db.Exec(`INSERT INTO match_participants
			(match_id, xuid, team_id, outcome, left_in_progress, last_leave_time)
			VALUES ('m_quit', 'owner', 0, 3, TRUE, ?)`, quitTime); err != nil {
			t.Fatalf("insert owner: %v", err)
		}
		for _, q := range []struct {
			xuid          string
			team, outcome int
		}{{"teammate", 0, 3}, {"opp1", 1, 2}, {"opp2", 1, 2}} {
			if _, err := db.Exec(`INSERT INTO match_participants
				(match_id, xuid, team_id, outcome) VALUES ('m_quit', ?, ?, ?)`,
				q.xuid, q.team, q.outcome); err != nil {
				t.Fatalf("insert participant: %v", err)
			}
		}
		if withTimeline {
			// Au moment du quit (60000ms) team0 mène 3-1.
			for _, f := range []struct {
				killer string
				tms    int
			}{{"owner", 10000}, {"teammate", 20000}, {"owner", 30000}, {"opp1", 15000}} {
				if _, err := db.Exec(`INSERT INTO killer_victim_pairs
					(match_id, killer_xuid, victim_xuid, kill_count, time_ms)
					VALUES ('m_quit', ?, 'victim', 1, ?)`, f.killer, f.tms); err != nil {
					t.Fatalf("insert frag: %v", err)
				}
			}
		}
		if _, err := RunLUSRV2Shadow(context.Background(), nil, newPinnedSharedAccessor(db), "owner"); err != nil {
			t.Fatalf("RunLUSRV2Shadow: %v", err)
		}
		st, err := duckdb.NewSkillV2Repo(db).LoadState(context.Background(), "owner", "arena_slayer")
		if err != nil || st == nil {
			t.Fatalf("LoadState owner: %v", err)
		}
		return st.Mu
	}

	muWithTimeline := runQuit(true) // menait au quit → unrelated (2.5, capé à QuitPenaltyMuCap=1.5)
	muFallback := runQuit(false)    // pas de timeline → fallback outcome (Loss → related 1.0)

	if !(muWithTimeline < muFallback) {
		t.Errorf("menait au quit → pénalité forte → μ plus bas : withTimeline=%v doit < fallback=%v",
			muWithTimeline, muFallback)
	}
	// Écart attendu = min(DefaultQuitDeltaUnrelated, QuitPenaltyMuCap) - DefaultQuitDeltaRelated
	// = 1.5 - 1.0 = 0.5 depuis le cap quit [2026-05-31] (appliqué post-EP, EP identique).
	if diff := muFallback - muWithTimeline; diff < 0.49 || diff > 0.51 {
		t.Errorf("écart de pénalité attendu ≈ 0.5 (cap unrelated 1.5 − related 1.0), got %v", diff)
	}
}

// TestRunLUSRV2Shadow_Canonical_WriteFailHoldsWatermark (fix désync 2026-06-07)
// prouve le gate anti-désync : si l'écriture canonical (player DB) échoue, le
// watermark v2 (player_skill_state_v2, shared) n'avance PAS → le match est
// ré-essayé au cycle suivant et finit par recevoir sa ligne LUSR. Reproduit le
// scénario "database is closed" du 03/06 (player DB momentanément fermée) qui
// avait créé un trou LUSR permanent.
func TestRunLUSRV2Shadow_Canonical_WriteFailHoldsWatermark(t *testing.T) {
	origEnabled := os.Getenv(lusrV2EnvFlag)
	origCanonical := os.Getenv(lusrCanonicalEnvFlag)
	t.Cleanup(func() {
		_ = os.Setenv(lusrV2EnvFlag, origEnabled)
		_ = os.Setenv(lusrCanonicalEnvFlag, origCanonical)
	})
	_ = os.Setenv(lusrV2EnvFlag, "1")
	_ = os.Setenv(lusrCanonicalEnvFlag, "LUSR_V2")

	sharedDB := openShadowTestDB(t)
	seedCanonical2v2(t, sharedDB, "m_gate", time.Date(2025, 8, 1, 14, 0, 0, 0, time.UTC))

	ctx := context.Background()
	repo := duckdb.NewSkillV2Repo(sharedDB)

	// 1) Player DB FERMÉE → writeCanonicalLUSRRow échoue ("database is closed").
	brokenPlayerDB, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open broken playerDB: %v", err)
	}
	_ = brokenPlayerDB.Close() // fermeture volontaire = reproduit l'incident 03/06

	processed, err := RunLUSRV2Shadow(ctx, brokenPlayerDB, newPinnedSharedAccessor(sharedDB), "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2Shadow (broken): %v", err)
	}
	if processed != 0 {
		t.Errorf("écriture échouée → processed doit être 0 (gate), got %d", processed)
	}
	// Watermark NON avancé : aucun état v2 pour owner (le gate a tenu le watermark).
	if st, _ := repo.LoadState(ctx, "owner", "arena_slayer"); st != nil {
		t.Errorf("watermark ne doit PAS avancer après échec d'écriture, state=%+v", st)
	}

	// 2) Retry avec une player DB saine → le match tenu passe ET écrit la ligne LUSR.
	goodPlayerDB := openCanonicalPlayerTestDB(t)
	processed2, err := RunLUSRV2Shadow(ctx, goodPlayerDB, newPinnedSharedAccessor(sharedDB), "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2Shadow (retry): %v", err)
	}
	if processed2 != 1 {
		t.Errorf("retry doit traiter le match tenu, processed=%d want 1", processed2)
	}
	var countLUSR int
	if err := goodPlayerDB.QueryRow(`SELECT COUNT(*) FROM match_skill_rank
		WHERE match_id = ? AND rating_type = 'LUSR'`, "m_gate").Scan(&countLUSR); err != nil {
		t.Fatalf("count LUSR: %v", err)
	}
	if countLUSR != 1 {
		t.Errorf("après retry, ligne LUSR attendue = 1, got %d", countLUSR)
	}
	// Watermark avancé maintenant que l'écriture a réussi.
	if st, _ := repo.LoadState(ctx, "owner", "arena_slayer"); st == nil {
		t.Error("watermark doit avoir avancé après le retry réussi")
	}
}

// TestRunLUSRV2Shadow_Canonical_HeldGroupSkipsLaterMatches (fix anti-gap multi-matchs
// 2026-06-07) prouve que si un match ANCIEN d'un groupe échoue son écriture
// canonical, les matchs PLUS RÉCENTS du même groupe sont SAUTÉS ce run — sinon
// leur persist avancerait le watermark de groupe par-dessus le match non écrit,
// le rendant skippedAlready à jamais (gap LUSR permanent, la classe de bug d'origine).
func TestRunLUSRV2Shadow_Canonical_HeldGroupSkipsLaterMatches(t *testing.T) {
	origEnabled := os.Getenv(lusrV2EnvFlag)
	origCanonical := os.Getenv(lusrCanonicalEnvFlag)
	t.Cleanup(func() {
		_ = os.Setenv(lusrV2EnvFlag, origEnabled)
		_ = os.Setenv(lusrCanonicalEnvFlag, origCanonical)
	})
	_ = os.Setenv(lusrV2EnvFlag, "1")
	_ = os.Setenv(lusrCanonicalEnvFlag, "LUSR_V2")

	sharedDB := openShadowTestDB(t)
	// 2 matchs du MÊME groupe (pair_name 'Slayer' → arena_slayer), m_old < m_new.
	seedCanonical2v2(t, sharedDB, "m_old", time.Date(2025, 9, 1, 12, 0, 0, 0, time.UTC))
	seedCanonical2v2(t, sharedDB, "m_new", time.Date(2025, 9, 1, 13, 0, 0, 0, time.UTC))

	ctx := context.Background()
	repo := duckdb.NewSkillV2Repo(sharedDB)

	// 1) Player DB fermée → toute écriture canonical échoue.
	canonicalWriteHeldWatermark.Set(0)
	brokenPlayerDB, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open broken playerDB: %v", err)
	}
	_ = brokenPlayerDB.Close()

	processed, err := RunLUSRV2Shadow(ctx, brokenPlayerDB, newPinnedSharedAccessor(sharedDB), "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2Shadow (broken): %v", err)
	}
	if processed != 0 {
		t.Errorf("processed = %d, want 0 (groupe tenu)", processed)
	}
	// CŒUR DU FIX : m_old échoue → groupe tenu → m_new SAUTÉ avant toute tentative
	// d'écriture. Donc UNE seule tentative (m_old). Sans heldGroups, m_new tenterait
	// aussi (compteur=2) et son persist aurait avancé le watermark par-dessus m_old.
	if got := canonicalWriteHeldWatermark.Value(); got != 1 {
		t.Errorf("canonicalWriteHeldWatermark = %d, want 1 (m_new doit être sauté, pas tenté)", got)
	}
	if st, lerr := repo.LoadState(ctx, "owner", "arena_slayer"); lerr != nil {
		t.Fatalf("LoadState: %v", lerr)
	} else if st != nil {
		t.Errorf("watermark ne doit pas avoir avancé, state=%+v", st)
	}

	// 2) Retry avec player DB saine → les DEUX matchs sont traités, EN ORDRE, aucun gap.
	goodPlayerDB := openCanonicalPlayerTestDB(t)
	processed2, err := RunLUSRV2Shadow(ctx, goodPlayerDB, newPinnedSharedAccessor(sharedDB), "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2Shadow (retry): %v", err)
	}
	if processed2 != 2 {
		t.Errorf("retry processed = %d, want 2 (m_old puis m_new)", processed2)
	}
	for _, mid := range []string{"m_old", "m_new"} {
		var n int
		if err := goodPlayerDB.QueryRow(`SELECT COUNT(*) FROM match_skill_rank
			WHERE match_id = ? AND rating_type = 'LUSR'`, mid).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", mid, err)
		}
		if n != 1 {
			t.Errorf("%s : ligne LUSR = %d, want 1 (aucun gap, aucun doublon)", mid, n)
		}
	}
}

// TestLoadPreviousLUSRRating_ChronologicalOrder (fix delta figé 2026-06-11) : le
// rating "précédent" est le match CHRONOLOGIQUEMENT antérieur (start_time), PAS le
// dernier écrit (written_at — fragile : NULL/désordre → bug delta +75 sur Choco).
// Vérifie aussi l'exclusion du match courant (retry append-only).
func TestLoadPreviousLUSRRating_ChronologicalOrder(t *testing.T) {
	playerDB := openCanonicalPlayerTestDB(t)
	ctx := context.Background()
	t1 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 1, 1, 13, 0, 0, 0, time.UTC)
	if _, err := playerDB.Exec(`INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, playlist_group, start_time, written_at) VALUES
		('m1', 'LUSR', 1500, 'arena_slayer', TIMESTAMP '2025-01-01 12:00:00', TIMESTAMP '2025-01-01 12:00:00'),
		('m2', 'LUSR', 1600, 'arena_slayer', TIMESTAMP '2025-01-01 13:00:00', TIMESTAMP '2025-01-01 13:00:00')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	// Précédent de m2 (start 13:00) = m1 (start 12:00 < 13:00).
	if prev := loadPreviousLUSRRating(ctx, playerDB, "arena_slayer", "m2", t2); prev == nil || *prev != 1500 {
		t.Errorf("prev(m2) = %v, want 1500 (m1)", prev)
	}
	// Doublon m2 (retry append, written_at PLUS RÉCENT mais même start_time) : m2
	// reste exclu (match_id ET start_time pas < 13:00) → toujours m1. C'est le cœur
	// du fix : un written_at plus récent ne doit PAS détourner le "précédent".
	if _, err := playerDB.Exec(`INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, playlist_group, start_time, written_at)
		VALUES ('m2', 'LUSR', 1600, 'arena_slayer', TIMESTAMP '2025-01-01 13:00:00', TIMESTAMP '2025-01-01 14:00:00')`); err != nil {
		t.Fatalf("insert dup: %v", err)
	}
	if prev := loadPreviousLUSRRating(ctx, playerDB, "arena_slayer", "m2", t2); prev == nil || *prev != 1500 {
		t.Errorf("avec doublon m2, prev(m2) = %v, want 1500 (m1)", prev)
	}
	// m1 est le PREMIER chronologiquement → aucun précédent (nil), MÊME si m2 existe
	// (postérieur). L'ancien tri written_at DESC aurait retourné m2 à tort.
	if prev := loadPreviousLUSRRating(ctx, playerDB, "arena_slayer", "m1", t1); prev != nil {
		t.Errorf("prev(m1) = %v, want nil (m1 premier ; m2 postérieur)", *prev)
	}
}

// TestLoadPreviousDisplayedOrdinal_ChronologicalOrder — l'hystérésis lit le palier
// du match CHRONOLOGIQUEMENT précédent (start_time), pas le dernier écrit. -1 si
// aucun précédent (premier match), même si un match postérieur existe.
func TestLoadPreviousDisplayedOrdinal_ChronologicalOrder(t *testing.T) {
	playerDB := openCanonicalPlayerTestDB(t)
	ctx := context.Background()
	bnd := skillv2.DefaultTierBoundaries()
	t1 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 1, 1, 13, 0, 0, 0, time.UTC)
	if _, err := playerDB.Exec(`INSERT INTO match_skill_rank
		(match_id, rating_type, tier, sub_tier, playlist_group, start_time, written_at) VALUES
		('m1', 'LUSR', 'Gold', 3, 'arena_slayer', TIMESTAMP '2025-01-01 12:00:00', TIMESTAMP '2025-01-01 12:00:00'),
		('m2', 'LUSR', 'Gold', 5, 'arena_slayer', TIMESTAMP '2025-01-01 13:00:00', TIMESTAMP '2025-01-01 13:00:00')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	// Précédent de m2 (13:00) = m1 (Gold 3).
	wantM1 := skillv2.TierOrdinal(bnd, "Gold", 3)
	if got := loadPreviousDisplayedOrdinal(ctx, playerDB, "arena_slayer", "m2", t2, bnd); got != wantM1 {
		t.Errorf("prev ordinal(m2) = %d, want %d (m1 Gold 3)", got, wantM1)
	}
	// m1 est le premier chronologiquement → -1, même si m2 (postérieur) existe.
	if got := loadPreviousDisplayedOrdinal(ctx, playerDB, "arena_slayer", "m1", t1, bnd); got != -1 {
		t.Errorf("prev ordinal(m1) = %d, want -1 (premier match)", got)
	}
}

// TestRunLUSRV2ShadowOwnerOnly_DoesNotOverwriteTeammateState (recovery 2026-06-07)
// prouve que le mode owner-only persiste UNIQUEMENT l'état du joueur traité : un
// coéquipier tracké conserve son état v2 (zéro écrasement croisé), tandis que le
// joueur owner obtient bien sa ligne canonique + son état. C'est ce qui permet au
// backfill par-joueur de donner une couverture complète à TOUS sans contamination.
func TestRunLUSRV2ShadowOwnerOnly_DoesNotOverwriteTeammateState(t *testing.T) {
	origEnabled := os.Getenv(lusrV2EnvFlag)
	origCanonical := os.Getenv(lusrCanonicalEnvFlag)
	t.Cleanup(func() {
		_ = os.Setenv(lusrV2EnvFlag, origEnabled)
		_ = os.Setenv(lusrCanonicalEnvFlag, origCanonical)
	})
	_ = os.Setenv(lusrV2EnvFlag, "1")
	_ = os.Setenv(lusrCanonicalEnvFlag, "LUSR_V2")

	sharedDB := openShadowTestDB(t)
	playerDB := openCanonicalPlayerTestDB(t)
	ctx := context.Background()
	repo := duckdb.NewSkillV2Repo(sharedDB)

	// Pré-seed l'état d'un coéquipier tracké — il NE doit PAS être écrasé.
	pretime := time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC)
	if err := repo.UpsertState(ctx, domain.SkillV2State{
		XUID: "teammate", PlaylistGroup: "arena_slayer",
		Mu: 30.0, Sigma: 3.0, Experience: 42, LastMatchAt: &pretime,
	}); err != nil {
		t.Fatalf("seed teammate: %v", err)
	}

	// Match owner+teammate vs opp1+opp2 (pair_name 'Slayer' → arena_slayer).
	seedCanonical2v2(t, sharedDB, "m_oo", time.Date(2025, 9, 2, 12, 0, 0, 0, time.UTC))

	processed, err := RunLUSRV2ShadowOwnerOnly(ctx, playerDB, newPinnedSharedAccessor(sharedDB), "owner")
	if err != nil {
		t.Fatalf("RunLUSRV2ShadowOwnerOnly: %v", err)
	}
	if processed != 1 {
		t.Errorf("processed = %d, want 1", processed)
	}

	// Owner : ligne canonique écrite + état créé.
	var n int
	if err := playerDB.QueryRow(`SELECT COUNT(*) FROM match_skill_rank
		WHERE match_id = 'm_oo' AND rating_type = 'LUSR'`).Scan(&n); err != nil {
		t.Fatalf("count owner LUSR: %v", err)
	}
	if n != 1 {
		t.Errorf("owner LUSR row = %d, want 1", n)
	}
	if st, _ := repo.LoadState(ctx, "owner", "arena_slayer"); st == nil {
		t.Error("état owner manquant après run owner-only")
	}

	// Coéquipier : état STRICTEMENT INCHANGÉ (mu=30, exp=42) — pas écrasé.
	tst, terr := repo.LoadState(ctx, "teammate", "arena_slayer")
	if terr != nil || tst == nil {
		t.Fatalf("état teammate disparu: %v", terr)
	}
	if tst.Mu != 30.0 || tst.Experience != 42 {
		t.Errorf("état teammate écrasé par le run owner-only: mu=%v exp=%d (want 30.0 / 42)", tst.Mu, tst.Experience)
	}
}
