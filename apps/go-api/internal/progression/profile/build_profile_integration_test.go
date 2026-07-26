//go:build integration

// build_profile_integration_test.go — tests d'intégration de BuildProfile()
// avec DuckDB in-memory.

package profile

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

const (
	testXUID  = "xuid-profile-test"
	testGT    = "ProfileTester"
	testTitle = "halo_infinite"
)

// setupProfileEnv crée un PlayerDB minimal pour exercer BuildProfile :
//   - Player (stats.duckdb) avec shared attaché en RW
//   - Metadata (metadata.duckdb) pour les templates (catalogue vide en test
//     simple, ou seeded pour tester les suggestions)
//
// Pattern aligné sur post_sync_progression_test.go.setupProgressionEnv.
func setupProfileEnv(t *testing.T) *duckdb.PlayerDB {
	t.Helper()
	dir := t.TempDir()

	openAndMigrate := func(name string, target migration.TargetDB) *duckdb.DB {
		path := filepath.Join(dir, name+".duckdb")
		raw, err := sql.Open("duckdb", path)
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		if err := migration.RunForDB(raw, target); err != nil {
			raw.Close()
			t.Fatalf("migrate %s: %v", name, err)
		}
		raw.Close()
		db, err := duckdb.OpenReadWrite(path)
		if err != nil {
			t.Fatalf("reopen %s: %v", name, err)
		}
		return db
	}

	// Topologie post-ADR 0016 : shared ouvert en conn RW dédiée (pas d'ATTACH
	// sur player). SharedReader = LegacySharedReader(shared) en test.
	shared := openAndMigrate("shared_matches_v2", migration.TargetShared)
	meta := openAndMigrate("metadata", migration.TargetMetadata)
	player := openAndMigrate("stats", migration.TargetPlayer)

	ctx := context.Background()

	// personal_score_awards n'est pas dans le registry de migrations (legacy
	// schema géré par sync.EnsurePlayerSchema). On le crée à la main pour les
	// tests qui consomment ce qui sera la source des axes radar enrichis
	// par awards.toml (V2 §2).
	// Append-only #23046 (Phase 2) : born append-only (generation_id + is_tombstone) ;
	// les readers (applyAwardsRadarAxes) lisent la vue personal_score_awards_latest.
	if _, err := player.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS personal_score_awards (
			id             INTEGER,
			match_id       VARCHAR NOT NULL,
			xuid           VARCHAR NOT NULL,
			award_name     VARCHAR NOT NULL,
			award_category VARCHAR,
			award_count    INTEGER DEFAULT 1,
			award_score    INTEGER DEFAULT 0,
			generation_id  BIGINT NOT NULL DEFAULT 0,
			is_tombstone   BOOLEAN DEFAULT FALSE
		)
	`); err != nil {
		t.Fatalf("create personal_score_awards: %v", err)
	}
	if _, err := player.Exec(ctx, `CREATE OR REPLACE VIEW personal_score_awards_latest AS
		SELECT * EXCLUDE (rk) FROM (
			SELECT *, DENSE_RANK() OVER (PARTITION BY match_id, xuid ORDER BY generation_id DESC) AS rk
			FROM personal_score_awards)
		WHERE rk = 1 AND NOT is_tombstone`); err != nil {
		t.Fatalf("create personal_score_awards_latest view: %v", err)
	}

	pdb := &duckdb.PlayerDB{
		Player: player, Shared: shared, Metadata: meta,
		SharedReader: duckdb.LegacySharedReader(shared),
		XUID:         testXUID, Gamertag: testGT, TitleSlug: testTitle,
	}
	t.Cleanup(func() {
		player.Close()
		shared.Close()
		meta.Close()
	})
	return pdb
}

func seedMatchesForProfile(t *testing.T, pdb *duckdb.PlayerDB, now time.Time, count int) {
	t.Helper()
	ctx := context.Background()
	componentNames := []string{
		"kills_vs_expected", "deaths_vs_expected", "win_factor", "damage_efficiency",
		"accuracy_delta", "medal_exploit", "offensive_conversion", "defensive_resistance",
	}
	for i := 0; i < count; i++ {
		matchID := zeropadProfile(i, 4)
		startTime := now.AddDate(0, 0, -i)
		if _, err := pdb.Shared.Exec(ctx, `
			INSERT INTO match_registry (match_id, start_time)
			VALUES (?, ?)
		`, matchID, startTime); err != nil {
			t.Fatalf("match_registry %s: %v", matchID, err)
		}
		if _, err := pdb.Shared.Exec(ctx, `
			INSERT INTO match_participants (
				match_id, xuid, gamertag, team_id, outcome,
				kills, deaths, assists, personal_score, max_killing_spree,
				time_played_seconds
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, matchID, testXUID, testGT, 1, 2, 12+i%4, 8, 3, 1500, 5, 600); err != nil {
			t.Fatalf("match_participants %s: %v", matchID, err)
		}
		mu := 1500.0 + float64(i%20)
		if _, err := pdb.Player.Exec(ctx, `
			INSERT INTO match_skill_rank (match_id, rating_type, rating_value, rating_deviation, start_time)
			VALUES (?, 'LUSR', ?, 120, ?)
		`, matchID, mu, startTime); err != nil {
			t.Fatalf("match_skill_rank %s: %v", matchID, err)
		}
		// V2 §1 : peupler lusr_component_history (simule la live write de
		// sync.upsertLUSRRatings). Valeurs déterministes pour permettre
		// l'assertion sur current_avg / top20 / trend.
		for j, comp := range componentNames {
			value := 0.4 + float64((i+j)%6)*0.08 // valeurs [0.4..0.84]
			weight := 0.1 + float64(j)*0.02
			if _, err := pdb.Player.Exec(ctx, `
				INSERT INTO lusr_component_history (match_id, component_name, value, weight, computed_at)
				VALUES (?, ?, ?, ?, ?)
			`, matchID, comp, value, weight, startTime); err != nil {
				t.Fatalf("lusr_component_history %s/%s: %v", matchID, comp, err)
			}
		}
	}
}

func zeropadProfile(n, w int) string {
	out := make([]byte, w)
	for i := w - 1; i >= 0; i-- {
		out[i] = byte('0' + n%10)
		n /= 10
	}
	return string(out)
}

// TestBuildProfile_EmptyDB_HasEnoughDataFalse vérifie le cas dégradé.
func TestBuildProfile_EmptyDB_HasEnoughDataFalse(t *testing.T) {
	pdb := setupProfileEnv(t)
	svc := NewServiceFromPlayerDB(pdb)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)

	prof, err := svc.BuildProfile(context.Background(), testXUID, testTitle, 30, now)
	if err != nil {
		t.Fatalf("BuildProfile: %v", err)
	}
	if prof.HasEnoughData {
		t.Errorf("HasEnoughData on empty DB: got true, want false")
	}
	if prof.MatchesAnalyzed != 0 {
		t.Errorf("MatchesAnalyzed: got %d, want 0", prof.MatchesAnalyzed)
	}
}

// TestBuildProfile_WithMatches_FillsSections vérifie qu'au-dessus de
// MinMatchesForProfile, les sections sont remplies.
func TestBuildProfile_WithMatches_FillsSections(t *testing.T) {
	pdb := setupProfileEnv(t)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	seedMatchesForProfile(t, pdb, now, MinMatchesForProfile+5)

	svc := NewServiceFromPlayerDB(pdb)
	prof, err := svc.BuildProfile(context.Background(), testXUID, testTitle, 60, now)
	if err != nil {
		t.Fatalf("BuildProfile: %v", err)
	}
	if !prof.HasEnoughData {
		t.Errorf("HasEnoughData: want true, got false (matches=%d)", prof.MatchesAnalyzed)
	}
	if prof.MatchesAnalyzed < MinMatchesForProfile {
		t.Errorf("MatchesAnalyzed: got %d, want >= %d", prof.MatchesAnalyzed, MinMatchesForProfile)
	}
	if len(prof.RadarAxes) == 0 {
		t.Error("RadarAxes: empty")
	}
	if prof.LUSR.Mu == 0 {
		t.Error("LUSR.Mu: zero")
	}
	if prof.Tier.IsEmpty() {
		t.Error("Tier: empty")
	}
	if prof.EngagementSnap.Tier == "" {
		t.Error("EngagementSnap.Tier: empty")
	}
}

// TestBuildProfile_LUSRComponentsPopulated vérifie que loadLUSRComponentsBreakdown
// lit effectivement lusr_component_history (V2 §1) et remplit les 8 composantes.
func TestBuildProfile_LUSRComponentsPopulated(t *testing.T) {
	pdb := setupProfileEnv(t)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	seedMatchesForProfile(t, pdb, now, MinMatchesForProfile+5)

	svc := NewServiceFromPlayerDB(pdb)
	prof, err := svc.BuildProfile(context.Background(), testXUID, testTitle, 60, now)
	if err != nil {
		t.Fatalf("BuildProfile: %v", err)
	}
	if !prof.HasEnoughData {
		t.Fatalf("HasEnoughData: want true (matches=%d)", prof.MatchesAnalyzed)
	}
	if len(prof.LUSRComponents) != 8 {
		t.Fatalf("LUSRComponents: got %d, want 8", len(prof.LUSRComponents))
	}
	// Au moins une composante doit avoir current_avg > 0 (les valeurs seed
	// sont déterministes dans [0.4..0.84]).
	hasData := false
	for _, c := range prof.LUSRComponents {
		if c.CurrentAvg > 0 || c.PersonalTop20 > 0 {
			hasData = true
		}
	}
	if !hasData {
		t.Errorf("aucun current_avg/top20 > 0 dans LUSRComponents")
	}
}

// TestBuildProfile_ObjectiveAxisFromIndex vérifie le nouveau modèle de l'axe
// Objective (plan PLAN_AXE_OBJECTIFS_INDEX) : il vient de l'index par
// opportunité (match_objective_stats_latest, shared) et NON des
// personal_score_awards — sans match à objectif dans la fenêtre, l'axe est
// RETIRÉ même avec un AwardMappingSet mappant des awards « objective » ;
// awards.toml reste la source des AUTRES axes (support boosté ici).
func TestBuildProfile_ObjectiveAxisFromIndex(t *testing.T) {
	pdb := setupProfileEnv(t)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	seedMatchesForProfile(t, pdb, now, MinMatchesForProfile+5)

	// Seed personal_score_awards : awards objectifs (ne doivent PLUS alimenter
	// l'axe) + kill_assist (doit toujours booster support via le mapping).
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		matchID := zeropadProfile(i, 4)
		if _, err := pdb.Player.Exec(ctx, `
			INSERT INTO personal_score_awards (match_id, xuid, award_name, award_category, award_count, award_score)
			VALUES (?, ?, 'flag_captured', 'objective', 1, 200)
		`, matchID, testXUID); err != nil {
			t.Fatalf("insert award: %v", err)
		}
		if _, err := pdb.Player.Exec(ctx, `
			INSERT INTO personal_score_awards (match_id, xuid, award_name, award_category, award_count, award_score)
			VALUES (?, ?, 'kill_assist', 'support', 4, 200)
		`, matchID, testXUID); err != nil {
			t.Fatalf("insert award: %v", err)
		}
	}
	awardSet := mappings.NewAwardMappingSet("halo_infinite", 1, map[string]mappings.AwardMapping{
		"flag_captured": {AwardName: "flag_captured", Axes: []string{"objective"}, Weight: 5.0},
		"kill_assist":   {AwardName: "kill_assist", Axes: []string{"support"}, Weight: 1.0},
	})

	// Sans ligne match_objective_stats : axe objective RETIRÉ (pas de 0 trompeur),
	// malgré les awards « objective » mappés.
	svcNoObj := NewServiceFromPlayerDB(pdb).WithAwardMapping(awardSet)
	profNoObj, err := svcNoObj.BuildProfile(context.Background(), testXUID, testTitle, 60, now)
	if err != nil {
		t.Fatalf("BuildProfile sans lignes objectifs: %v", err)
	}
	if _, present := axisLookup(profNoObj, "objective"); present {
		t.Errorf("axe objective présent sans match à objectif dans la fenêtre — want retiré")
	}
	supportNoObj, _ := axisLookup(profNoObj, "support")

	// Awards toujours actifs sur les AUTRES axes : support boosté vs sans mapping.
	profNoMap, err := NewServiceFromPlayerDB(pdb).BuildProfile(context.Background(), testXUID, testTitle, 60, now)
	if err != nil {
		t.Fatalf("BuildProfile sans mapping: %v", err)
	}
	supportNoMap, _ := axisLookup(profNoMap, "support")
	if supportNoObj <= supportNoMap {
		t.Errorf("support avec mapping (%.2f) devrait dépasser sans mapping (%.2f)", supportNoObj, supportNoMap)
	}

	// Seed 2 lignes objectifs CTF (shared) sur des matchs de la fenêtre → axe
	// présent, alimenté par l'index par opportunité.
	for i, mid := range []string{zeropadProfile(0, 4), zeropadProfile(1, 4)} {
		if _, err := pdb.Shared.Exec(ctx, `
			INSERT INTO match_objective_stats (match_id, xuid, flag_captures, flag_secures, time_as_flag_carrier_seconds)
			VALUES (?, ?, ?, ?, ?)
		`, mid, testXUID, 2+i, 3, 25.0); err != nil {
			t.Fatalf("insert match_objective_stats %s: %v", mid, err)
		}
	}
	profObj, err := svcNoObj.BuildProfile(context.Background(), testXUID, testTitle, 60, now)
	if err != nil {
		t.Fatalf("BuildProfile avec lignes objectifs: %v", err)
	}
	objVal, present := axisLookup(profObj, "objective")
	if !present || objVal <= 0 {
		t.Errorf("axe objective avec lignes objectifs: present=%v val=%.2f, want présent et > 0", present, objVal)
	}
}

// axisLookup retourne (valeur, présent) pour un axe du radar du profil.
func axisLookup(prof *PlayerProfile, axis string) (float64, bool) {
	for _, a := range prof.RadarAxes {
		if a.Axis == axis {
			return a.Value, true
		}
	}
	return 0, false
}

// TestBuildProfile_SkillTrendPopulated vérifie que la sparkline de tendance LUSR
// (DEC-5/D2) est peuplée depuis match_skill_rank_latest et datée en jours UTC.
func TestBuildProfile_SkillTrendPopulated(t *testing.T) {
	pdb := setupProfileEnv(t)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	seedMatchesForProfile(t, pdb, now, 25) // 25 jours notés → LOWESS effectif

	svc := NewServiceFromPlayerDB(pdb)
	prof, err := svc.BuildProfile(context.Background(), testXUID, testTitle, 60, now)
	if err != nil {
		t.Fatalf("BuildProfile: %v", err)
	}
	if len(prof.SkillTrend) == 0 {
		t.Fatalf("SkillTrend: vide, want peuplé (>= 3 points de rating dans 90 j)")
	}
	for i, p := range prof.SkillTrend {
		if len(p.Date) != 10 { // YYYY-MM-DD
			t.Errorf("point %d: Date %q, want format YYYY-MM-DD", i, p.Date)
		}
		if p.Value <= 0 {
			t.Errorf("point %d: Value %v, want > 0 (échelle points LUSR)", i, p.Value)
		}
		if i > 0 && p.Date < prof.SkillTrend[i-1].Date {
			t.Errorf("SkillTrend non trié ASC en %d: %s < %s", i, p.Date, prof.SkillTrend[i-1].Date)
		}
	}
}

// TestLoadActivityCalendar_CountsDistinctDays vérifie le calendrier d'activité
// (DEC-5/D3) : 1 match/jour seedé → 1 jour par entrée, comptes = 1, tri ASC.
func TestLoadActivityCalendar_CountsDistinctDays(t *testing.T) {
	pdb := setupProfileEnv(t)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	seedMatchesForProfile(t, pdb, now, 10) // 1 match/jour sur 10 jours distincts

	svc := NewServiceFromPlayerDB(pdb)
	cal, err := svc.LoadActivityCalendar(context.Background(), testXUID, now.AddDate(0, 0, -90), now)
	if err != nil {
		t.Fatalf("LoadActivityCalendar: %v", err)
	}
	if len(cal.Days) != 10 {
		t.Fatalf("Days: got %d, want 10 (jours distincts)", len(cal.Days))
	}
	for _, d := range cal.Days {
		if d.Count != 1 {
			t.Errorf("jour %s: count %d, want 1", d.Date, d.Count)
		}
		if len(d.Date) != 10 {
			t.Errorf("jour %q: format YYYY-MM-DD attendu", d.Date)
		}
	}
	for i := 1; i < len(cal.Days); i++ {
		if cal.Days[i-1].Date > cal.Days[i].Date {
			t.Errorf("Days non trié ASC en %d: %s > %s", i, cal.Days[i-1].Date, cal.Days[i].Date)
		}
	}
	if cal.Since == "" || cal.Until == "" {
		t.Errorf("Since/Until vides: %q / %q", cal.Since, cal.Until)
	}
}

// TestLoadActivityCalendar_EmptyWindow vérifie le cas vide (aucun match).
func TestLoadActivityCalendar_EmptyWindow(t *testing.T) {
	pdb := setupProfileEnv(t)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)

	svc := NewServiceFromPlayerDB(pdb)
	cal, err := svc.LoadActivityCalendar(context.Background(), testXUID, now.AddDate(0, 0, -90), now)
	if err != nil {
		t.Fatalf("LoadActivityCalendar: %v", err)
	}
	if len(cal.Days) != 0 {
		t.Errorf("Days: got %d, want 0 (DB vide)", len(cal.Days))
	}
}

// TestLoad_V2Compat vérifie que Load() reste compatible avec le caller V2
// (post_sync_progression.go). MinMatchesForRating=10 → 10+ matchs requis.
func TestLoad_V2Compat(t *testing.T) {
	pdb := setupProfileEnv(t)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	seedMatchesForProfile(t, pdb, now, MinMatchesForRating+5)

	svc := NewService(pdb.Player) // V2 path : constructor minimal
	prof, err := svc.Load(context.Background(), testXUID, testTitle, 30, now)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if prof.LUSR.MatchesCount < MinMatchesForRating {
		t.Errorf("MatchesCount: got %d, want >= %d", prof.LUSR.MatchesCount, MinMatchesForRating)
	}
	if prof.Tier.IsEmpty() {
		t.Error("Tier: empty after Load() with enough matches")
	}
}
