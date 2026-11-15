//go:build integration

package profilerepo

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

const (
	testXUID  = "xuid-profilerepo"
	testOther = "xuid-autre-joueur"
	testTitle = "halo_infinite"
)

// repoEnv : bases migrées d'un joueur (joueur, partagée, référentiels) sur fichiers temporaires.
func repoEnv(t *testing.T, withMetadata bool) *duckdb.PlayerDB {
	t.Helper()
	dir := t.TempDir()
	open := func(name string, target migration.TargetDB) *duckdb.DB {
		path := filepath.Join(dir, name+".duckdb")
		raw, err := sql.Open("duckdb", path)
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		if err := migration.RunForDB(raw, target); err != nil {
			_ = raw.Close()
			t.Fatalf("migrate %s: %v", name, err)
		}
		_ = raw.Close()
		db, err := duckdb.OpenReadWrite(path)
		if err != nil {
			t.Fatalf("reopen %s: %v", name, err)
		}
		t.Cleanup(func() { _ = db.Close() })
		return db
	}
	shared := open("shared", migration.TargetShared)
	player := open("stats", migration.TargetPlayer)
	pdb := &duckdb.PlayerDB{
		Player: player, Shared: shared, SharedReader: duckdb.LegacySharedReader(shared),
		XUID: testXUID, TitleSlug: testTitle,
	}
	if withMetadata {
		pdb.Metadata = open("metadata", migration.TargetMetadata)
	}
	ctx := context.Background()
	if _, err := player.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS personal_score_awards (
			id INTEGER, match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL, award_name VARCHAR NOT NULL,
			award_category VARCHAR, award_count INTEGER DEFAULT 1, award_score INTEGER DEFAULT 0,
			generation_id BIGINT NOT NULL DEFAULT 0, is_tombstone BOOLEAN DEFAULT FALSE);
		CREATE OR REPLACE VIEW personal_score_awards_latest AS
		SELECT * EXCLUDE (rk) FROM (
			SELECT *, DENSE_RANK() OVER (PARTITION BY match_id, xuid ORDER BY generation_id DESC) AS rk
			FROM personal_score_awards)
		WHERE rk = 1 AND NOT is_tombstone`); err != nil {
		t.Fatalf("personal_score_awards: %v", err)
	}
	return pdb
}

var day0 = time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)

// seedMatch écrit un match (début day0 - daysAgo) joué par le joueur, et par un autre joueur si withOther.
func seedMatch(t *testing.T, pdb *duckdb.PlayerDB, id string, daysAgo int, withOther bool) {
	t.Helper()
	ctx := context.Background()
	start := day0.AddDate(0, 0, -daysAgo)
	if _, err := pdb.Shared.Exec(ctx, `INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)`, id, start); err != nil {
		t.Fatalf("match_registry %s: %v", id, err)
	}
	for _, x := range []struct {
		xuid  string
		kills int
		on    bool
	}{{testXUID, 10, true}, {testOther, 3, withOther}} {
		if !x.on {
			continue
		}
		if _, err := pdb.Shared.Exec(ctx, `
			INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome, kills, deaths, assists,
				personal_score, max_killing_spree, time_played_seconds)
			VALUES (?, ?, 'gt', 1, 2, ?, 4, 2, 1000, 3, 600)`, id, x.xuid, x.kills); err != nil {
			t.Fatalf("match_participants %s: %v", id, err)
		}
	}
	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO match_skill_rank (match_id, rating_type, rating_value, rating_deviation, start_time)
		VALUES (?, 'LUSR', ?, 120, ?)`, id, 1500.0+float64(daysAgo), start); err != nil {
		t.Fatalf("match_skill_rank %s: %v", id, err)
	}
}

func TestRepo_LecturesDuJoueur(t *testing.T) {
	pdb := repoEnv(t, false)
	seedMatch(t, pdb, "m-1", 3, false)
	seedMatch(t, pdb, "m-2", 2, true)
	seedMatch(t, pdb, "m-3", 1, false)
	r := New(pdb)
	ctx := context.Background()

	if r.TitleSlug() != testTitle {
		t.Errorf("TitleSlug = %q", r.TitleSlug())
	}

	lusr, err := r.LatestLUSR(ctx)
	if err != nil {
		t.Fatalf("LatestLUSR: %v", err)
	}
	if lusr.MatchesCount != 3 || lusr.Mu != 1501 || lusr.Sigma != 120 || lusr.LastMatchAt == nil {
		t.Errorf("LatestLUSR = %+v, attendu 3 matchs, mu 1501 (le plus récent), sigma 120", lusr)
	}

	series, err := r.LUSRSeries(ctx, day0.AddDate(0, 0, -10), day0)
	if err != nil || len(series) != 3 || series[0] != 1503 || series[2] != 1501 {
		t.Errorf("LUSRSeries = %v (err %v), attendu [1503 1502 1501] en ordre chronologique", series, err)
	}
	pts, err := r.LUSRSeriesPoints(ctx, day0.AddDate(0, 0, -2).Add(-time.Hour), day0)
	if err != nil || len(pts) != 2 || !pts[0].At.Before(pts[1].At) {
		t.Errorf("LUSRSeriesPoints = %v (err %v), attendu 2 points datés croissants", pts, err)
	}

	n, err := r.CountWindowMatches(ctx, testXUID, day0.AddDate(0, 0, -10), day0)
	if err != nil || n != 3 {
		t.Errorf("CountWindowMatches = %d (err %v), attendu 3", n, err)
	}
	if n, _ := r.CountWindowMatches(ctx, testOther, day0.AddDate(0, 0, -10), day0); n != 1 {
		t.Errorf("CountWindowMatches(autre) = %d, attendu 1", n)
	}

	base, err := r.RadarBase(ctx, testXUID, day0.AddDate(0, 0, -10), day0)
	if err != nil || base.MatchCount != 3 || base.AvgKills == nil || *base.AvgKills != 10 || base.AvgKillingSpree == nil || *base.AvgKillingSpree != 3 {
		t.Errorf("RadarBase = %+v (err %v)", base, err)
	}

	all, err := r.WindowMatchIDs(ctx, day0.AddDate(0, 0, -10), day0)
	if err != nil || len(all) != 3 {
		t.Errorf("WindowMatchIDs = %v (err %v), attendu les 3 matchs du registre", all, err)
	}
	mine, err := r.PlayerWindowMatchIDs(ctx, testOther, day0.AddDate(0, 0, -10), day0)
	if err != nil || len(mine) != 1 || mine[0] != "m-2" {
		t.Errorf("PlayerWindowMatchIDs(autre) = %v (err %v), attendu [m-2]", mine, err)
	}

	times, err := r.MatchStartTimes(ctx, testXUID, day0.AddDate(0, 0, -10), day0)
	if err != nil || len(times) != 3 || !times[0].Before(times[1]) || !times[1].Before(times[2]) {
		t.Errorf("MatchStartTimes = %v (err %v), attendu 3 instants croissants", times, err)
	}
	days, err := r.MatchDays(ctx, testXUID, day0.AddDate(0, 0, -10), day0)
	if err != nil || len(days) != 3 {
		t.Errorf("MatchDays = %v (err %v), attendu 3 lignes", days, err)
	}

	// Fenêtre sans match : listes vides, pas d'erreur.
	if empty, err := r.WindowMatchIDs(ctx, day0.AddDate(1, 0, 0), day0.AddDate(1, 0, 1)); err != nil || len(empty) != 0 {
		t.Errorf("fenêtre vide : %v (err %v)", empty, err)
	}
}

func TestRepo_AwardTotals(t *testing.T) {
	pdb := repoEnv(t, false)
	ctx := context.Background()
	for _, row := range []struct {
		match, xuid, award string
		count              int
	}{{"m-1", testXUID, "flag_captured", 2}, {"m-2", testXUID, "flag_captured", 1}, {"m-2", testXUID, "zero", 0}, {"m-2", testOther, "flag_captured", 9}} {
		if _, err := pdb.Player.Exec(ctx, `INSERT INTO personal_score_awards (match_id, xuid, award_name, award_count) VALUES (?, ?, ?, ?)`,
			row.match, row.xuid, row.award, row.count); err != nil {
			t.Fatalf("insert award: %v", err)
		}
	}
	totals, err := New(pdb).AwardTotals(ctx, testXUID, []string{"m-1", "m-2"})
	if err != nil {
		t.Fatalf("AwardTotals: %v", err)
	}
	if len(totals) != 1 || totals[0].Name != "flag_captured" || totals[0].Total != 3 {
		t.Errorf("AwardTotals = %+v, attendu un seul cumul flag_captured = 3 (cumul nul omis, autre joueur ignoré)", totals)
	}
}

func TestRepo_SuggestionTemplates(t *testing.T) {
	ctx := context.Background()

	_, available, err := New(repoEnv(t, false)).SuggestionTemplates(ctx, testTitle)
	if err != nil || available {
		t.Errorf("sans référentiels : available=%v err=%v, attendu indisponible sans erreur", available, err)
	}

	pdb := repoEnv(t, true)
	if _, err := pdb.Metadata.Exec(ctx, `
		INSERT INTO challenge_template (id, title_slug, metric, window_type, cadence, eval_type,
			normal_target, heroic_target, legendary_target, mythic_target, lusr_components, is_long_term, schema_version)
		VALUES ('t1', ?, 'kills', 'matches', 'daily', 'sum', 1, 2, 3, 4, 'win_factor,accuracy_delta', TRUE, 1)`, testTitle); err != nil {
		t.Fatalf("insert template: %v", err)
	}
	templates, available, err := New(pdb).SuggestionTemplates(ctx, testTitle)
	if err != nil || !available || len(templates) != 1 || templates[0].ID != "t1" ||
		len(templates[0].LUSRComponents) != 2 || !templates[0].IsLongTerm {
		t.Errorf("SuggestionTemplates = %+v available=%v err=%v", templates, available, err)
	}
	if other, _, _ := New(pdb).SuggestionTemplates(ctx, "halo_5"); len(other) != 0 {
		t.Errorf("les modèles d'un autre titre ne doivent pas sortir : %+v", other)
	}
}

func TestForPlayer_NilQuandLesBasesSontInutilisables(t *testing.T) {
	if ForPlayer(nil) != nil {
		t.Error("ForPlayer(nil) doit rendre un magasin nil")
	}
	if ForPlayer(&duckdb.PlayerDB{}) != nil {
		t.Error("ForPlayer sans base joueur doit rendre un magasin nil")
	}
	if ForPlayer(repoEnv(t, false)) == nil {
		t.Error("ForPlayer d'un joueur complet ne doit pas être nil")
	}
}
