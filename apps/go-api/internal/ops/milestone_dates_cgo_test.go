// Package ops — milestone_dates_cgo_test.go : backfill des dates de jalons.
// Tests du moteur PUR (sans DB) + intégration DuckDB (shared + stats temporaires).
package ops

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis"
)

func day(y int, mo time.Month, d, h int) time.Time {
	return time.Date(y, mo, d, h, 0, 0, 0, time.UTC)
}

func crossingMatches() []MilestoneCrossingMatch {
	return []MilestoneCrossingMatch{
		{PlayedAt: day(2026, 1, 1, 10), Win: true, Kills: 60, Assists: 0, Accuracy: 0.60, DamageDealt: 2000, DamageTaken: 1000, Deaths: 5},
		{PlayedAt: day(2026, 1, 1, 12), Win: false, Kills: 50, Accuracy: 0.40, DamageDealt: 4000, DamageTaken: 3000, Deaths: 8},
		{PlayedAt: day(2026, 1, 2, 10), Win: true, Kills: 40, Accuracy: 0.70, DamageDealt: 1500, DamageTaken: 900, Deaths: 4},
		{PlayedAt: day(2026, 1, 3, 10), Win: true, Kills: 30, Accuracy: 0.55, DamageDealt: 1200, DamageTaken: 800, Deaths: 3},
	}
}

const testHP = 225.0

func TestComputeMilestoneCrossings_PureMetrics(t *testing.T) {
	matches := crossingMatches()
	third := day(2026, 1, 2, 10) // C
	second := day(2026, 1, 1, 12)

	targets := []MilestoneTarget{
		{MilestoneID: "mp3", Metric: "matches_played", Threshold: 3},
		{MilestoneID: "kills120", Metric: "kills", Threshold: 120}, // 60+50+40=150 au 3e
		{MilestoneID: "wins2", Metric: "wins", Threshold: 2},       // 2e victoire = C
		{MilestoneID: "assists1", Metric: "assists", Threshold: 1000},
		{MilestoneID: "days2", Metric: "accuracy_threshold_days", Threshold: 2}, // 2e jour qualifiant = C
		{MilestoneID: "unknown", Metric: "foo_bar", Threshold: 1},
		{MilestoneID: "mp99", Metric: "matches_played", Threshold: 99}, // jamais atteint
	}
	got := ComputeMilestoneCrossings(matches, targets, testHP)

	expectAt := func(id string, want time.Time) {
		if got[id] == nil {
			t.Errorf("%s: crossing nil (attendu %v)", id, want)
			return
		}
		if !got[id].Equal(want) {
			t.Errorf("%s: crossing = %v, want %v", id, *got[id], want)
		}
	}
	expectNil := func(id string) {
		if got[id] != nil {
			t.Errorf("%s: crossing = %v, want nil", id, *got[id])
		}
	}
	expectAt("mp3", third)
	expectAt("kills120", third)
	expectAt("wins2", third)
	expectAt("days2", third)
	expectNil("assists1") // 0 assist cumulé -> jamais atteint
	expectNil("unknown")  // métrique inconnue
	expectNil("mp99")     // seuil hors de portée
	_ = second
}

func TestComputeMilestoneCrossings_CombatMetrics(t *testing.T) {
	// OC = hp*(kills + assists/3)/damage_dealt ; DR = damage_taken/(hp*deaths).
	// hp=225. Match1 OC=225*10/2000=1.125 (>=0.83) ; DR=1500/(225*4)=1.667 (>=1.59).
	matches := []MilestoneCrossingMatch{
		{PlayedAt: day(2026, 2, 1, 10), Kills: 10, DamageDealt: 2000, DamageTaken: 1500, Deaths: 4}, // OC ok, DR ok
		{PlayedAt: day(2026, 2, 2, 10), Kills: 5, DamageDealt: 3000, DamageTaken: 300, Deaths: 3},   // OC=0.375 no, DR=0.44 no
		{PlayedAt: day(2026, 2, 3, 10), Kills: 12, DamageDealt: 2500, DamageTaken: 2000, Deaths: 5}, // OC=1.08 ok, DR=1.78 ok
	}
	targets := []MilestoneTarget{
		{MilestoneID: "prec2", Metric: "combat_precision_matches", Threshold: 2},
		{MilestoneID: "end2", Metric: "combat_endurance_matches", Threshold: 2},
		{MilestoneID: "exc2", Metric: "combat_excellence_matches", Threshold: 2},
	}
	got := ComputeMilestoneCrossings(matches, targets, testHP)
	want := day(2026, 2, 3, 10) // 2e match qualifiant pour chaque = match 3
	for _, id := range []string{"prec2", "end2", "exc2"} {
		if got[id] == nil || !got[id].Equal(want) {
			t.Errorf("%s: crossing = %v, want %v", id, got[id], want)
		}
	}
}

// ─── Intégration DuckDB ──────────────────────────────────────────────────────

func TestBackfillMilestoneDates_DryRunThenApply(t *testing.T) {
	ctx := context.Background()
	shared := openPurgeTestDB(t, "shared_matches_v2")
	stats := openPurgeTestDB(t, "stats")

	if _, err := shared.Exec(`
		CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP,
			start_time_utc TIMESTAMPTZ
		);
		CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR, outcome INTEGER,
			kills INTEGER, headshot_kills INTEGER, assists INTEGER, deaths INTEGER,
			accuracy DOUBLE, damage_dealt DOUBLE, damage_taken DOUBLE
		);
	`); err != nil {
		t.Fatalf("create shared: %v", err)
	}
	insMatch := func(id string, ts string, outcome, kills, hs, assists, deaths int, acc, dd, dt float64) {
		if _, err := shared.Exec(`INSERT INTO match_registry (match_id, start_time_utc) VALUES (?, ?)`, id, ts); err != nil {
			t.Fatalf("insert registry %s: %v", id, err)
		}
		if _, err := shared.Exec(`INSERT INTO match_participants
			(match_id, xuid, outcome, kills, headshot_kills, assists, deaths, accuracy, damage_dealt, damage_taken)
			VALUES (?, 'x1', ?, ?, ?, ?, ?, ?, ?, ?)`, id, outcome, kills, hs, assists, deaths, acc, dd, dt); err != nil {
			t.Fatalf("insert participant %s: %v", id, err)
		}
	}
	insMatch("A", "2026-01-01 10:00:00+00", 2, 60, 5, 0, 5, 0.60, 2000, 1000)
	insMatch("B", "2026-01-01 12:00:00+00", 3, 50, 4, 0, 8, 0.40, 4000, 3000)
	insMatch("C", "2026-01-02 10:00:00+00", 2, 40, 3, 0, 4, 0.70, 1500, 900)
	insMatch("D", "2026-01-03 10:00:00+00", 2, 30, 2, 0, 3, 0.55, 1200, 800)

	if _, err := stats.Exec(`
		CREATE TABLE milestone_earned (
			user_id VARCHAR NOT NULL, title_slug VARCHAR NOT NULL, milestone_id VARCHAR NOT NULL,
			earned_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			PRIMARY KEY (user_id, title_slug, milestone_id)
		);
		CREATE INDEX idx_ms_earned_user_title ON milestone_earned(user_id, title_slug);
	`); err != nil {
		t.Fatalf("create milestone_earned: %v", err)
	}
	for _, id := range []string{"mp3", "kills120", "unknown"} {
		if _, err := stats.Exec(`INSERT INTO milestone_earned (user_id, title_slug, milestone_id, earned_at)
			VALUES ('x1', 'halo_infinite', ?, '2099-01-01 00:00:00')`, id); err != nil {
			t.Fatalf("seed earned %s: %v", id, err)
		}
	}

	catalog := map[string]MilestoneTarget{
		"mp3":      {MilestoneID: "mp3", Metric: "matches_played", Threshold: 3},
		"kills120": {MilestoneID: "kills120", Metric: "kills", Threshold: 120},
		// "unknown" absent du catalogue -> non dérivable -> NULL
	}

	// ── Dry-run : ne mute rien. ──
	dry, err := BackfillMilestoneDates(ctx, shared, stats, "halo_infinite", catalog, testHP, false)
	if err != nil {
		t.Fatalf("dry: %v", err)
	}
	if len(dry) != 1 || dry[0].Updated != 2 || dry[0].Nulled != 1 {
		t.Fatalf("dry result = %+v (attendu Updated=2 Nulled=1)", dry)
	}
	var seedCount int
	if err := stats.QueryRow(`SELECT COUNT(*) FROM milestone_earned WHERE earned_at = '2099-01-01 00:00:00'`).Scan(&seedCount); err != nil {
		t.Fatalf("count seed: %v", err)
	}
	if seedCount != 3 {
		t.Errorf("dry-run a muté des dates (%d encore au seed, attendu 3)", seedCount)
	}

	// ── Apply : dates recalculées + NULL pour non dérivable. ──
	res, err := BackfillMilestoneDates(ctx, shared, stats, "halo_infinite", catalog, testHP, true)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res[0].Updated != 2 || res[0].Nulled != 1 {
		t.Fatalf("apply result = %+v", res[0])
	}
	// mp3 et kills120 -> 2026-01-02 (match C).
	for _, id := range []string{"mp3", "kills120"} {
		var d string
		if err := stats.QueryRow(`SELECT strftime(earned_at, '%Y-%m-%d') FROM milestone_earned WHERE milestone_id = ?`, id).Scan(&d); err != nil {
			t.Fatalf("select %s: %v", id, err)
		}
		if d != "2026-01-02" {
			t.Errorf("%s earned_at date = %s (attendu 2026-01-02)", id, d)
		}
	}
	// unknown -> NULL.
	var nullCount int
	if err := stats.QueryRow(`SELECT COUNT(*) FROM milestone_earned WHERE milestone_id='unknown' AND earned_at IS NULL`).Scan(&nullCount); err != nil {
		t.Fatalf("count null: %v", err)
	}
	if nullCount != 1 {
		t.Errorf("unknown devrait avoir earned_at NULL, got count=%d", nullCount)
	}
}

// ─── Exclusion de la Campagne (E2 du plan .ai/V7.5/PLAN_BACKLOG_2026-10-15.md, DB-27) ───
//
// Un match d'arène (arena1, 01/09) et un match de Campagne PLUS RÉCENT (camp1, 02/09), joués par
// le même joueur, lus par les chargeurs d'ops qui nourrissent une donnée servie à l'UI (dates de
// paliers, corpus de la démo). Halo 5 : la Campagne disparaît ; Halo Infinite (aucune variante
// masquée) : le résolveur est neutre, les deux matchs restent.

const campagneOpsXUID = "2533274800000001"

var campagneOpsTitres = []struct {
	slug    string
	masquee bool
}{{"halo_5", true}, {"halo_infinite", false}}

// seedCampagneOps crée un shared fichier (arène + Campagne) et le ferme : les lecteurs d'ops le
// rouvrent par leur chemin.
func seedCampagneOps(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "shared.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	camp := analysis.CampaignExcludedVariantIDs("halo_5")[0]
	for _, s := range []string{
		`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP,
			start_time_utc TIMESTAMPTZ, game_variant_id VARCHAR)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, outcome INTEGER, kills INTEGER,
			headshot_kills INTEGER, assists INTEGER, deaths INTEGER, accuracy DOUBLE, damage_dealt DOUBLE,
			damage_taken DOUBLE)`,
		`INSERT INTO match_registry VALUES
			('arena1', '2026-09-01 10:00:00', '2026-09-01 10:00:00+00', 'aaaaaaaa-0000-0000-0000-000000000001'),
			('camp1', '2026-09-02 10:00:00', '2026-09-02 10:00:00+00', '` + camp + `')`,
		`INSERT INTO match_participants VALUES
			('arena1', '` + campagneOpsXUID + `', 2, 5, 1, 2, 1, 0.5, 500, 100),
			('camp1', '` + campagneOpsXUID + `', 2, 50, 5, 9, 0, 0.9, 5000, 100)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("fixture Campagne : %v\n%s", err, s)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close shared: %v", err)
	}
	return path
}

func campagneOpsNombre(masquee bool) int {
	if masquee {
		return 1
	}
	return 2
}

// E2.6 — dates de paliers : le franchissement se lit sur les matchs que l'app affiche.
func TestCampaignExclusion_LoadCrossingMatches(t *testing.T) {
	for _, tc := range campagneOpsTitres {
		path := seedCampagneOps(t)
		db, err := sql.Open("duckdb", path+"?access_mode=read_only")
		if err != nil {
			t.Fatalf("%s : open : %v", tc.slug, err)
		}
		got, err := loadCrossingMatches(context.Background(), db, campagneOpsXUID, tc.slug)
		_ = db.Close()
		if err != nil {
			t.Fatalf("%s : %v", tc.slug, err)
		}
		if len(got) != campagneOpsNombre(tc.masquee) {
			t.Errorf("%s : %d matchs de franchissement, attendu %d", tc.slug, len(got), campagneOpsNombre(tc.masquee))
		}
	}
}

// E2.6 — corpus de la démo : les N matchs récents ne portent pas la Campagne.
func TestCampaignExclusion_SelectRecentMatchIDs(t *testing.T) {
	for _, tc := range campagneOpsTitres {
		got, err := selectRecentMatchIDs(context.Background(), seedCampagneOps(t), campagneOpsXUID, tc.slug, 10)
		if err != nil {
			t.Fatalf("%s : %v", tc.slug, err)
		}
		if len(got) != campagneOpsNombre(tc.masquee) {
			t.Errorf("%s : matchs récents %v, attendu %d", tc.slug, got, campagneOpsNombre(tc.masquee))
		}
	}
}
