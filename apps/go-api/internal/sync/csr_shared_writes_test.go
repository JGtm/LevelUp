// Package sync — csr_shared_writes_test.go : tests unitaires de
// ExtractAllSharedCSRRows + tests intégration de UpsertSharedCSRs.
//
// Option A du plan pipeline CSR. Couvre la capture per-participant des CSR
// au sync (vs. legacy single-player path via ExtractCSRRowIfRanked).
package sync

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ─── ExtractAllSharedCSRRows (unit) ─────────────────────────────────────────

func mkReg(matchID, seasonID string, isRanked bool) *MatchRegistryRow {
	r := &MatchRegistryRow{
		MatchID:   matchID,
		StartTime: time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC),
		IsRanked:  isRanked,
	}
	if seasonID != "" {
		s := seasonID
		r.SeasonID = &s
	}
	return r
}

func mkSkill(tier string, value float64, subTier, remaining int, preValue float64) *MatchSkillData {
	skill := &MatchSkillData{
		PostMatchCSR: &CSRRankSnapshot{
			Value:                       value,
			Tier:                        tier,
			SubTier:                     subTier,
			MeasurementMatchesRemaining: remaining,
		},
	}
	if preValue >= 0 {
		skill.PreMatchCSR = &CSRRankSnapshot{Value: preValue, Tier: tier, SubTier: subTier}
	}
	return skill
}

func TestExtractAllSharedCSRRows_RankedMatch_AllParticipants(t *testing.T) {
	reg := mkReg("m1", "CsrSeason13-1", true)
	skill := map[string]*MatchSkillData{
		"xuid-A": mkSkill("Gold", 1100, 4, 0, 1085),    // matured, delta=+15
		"xuid-B": mkSkill("Onyx", 1850, 0, 0, 1800),    // Onyx no subTier
		"xuid-C": mkSkill("", 0, 0, 3, -1),             // placement (remaining=3)
		"xuid-D": nil,                                  // skip
		"xuid-E": {PostMatchCSR: nil, PreMatchCSR: nil}, // skip (no PostMatchCSR)
	}
	rows := ExtractAllSharedCSRRows(reg, skill)
	if len(rows) != 3 {
		t.Fatalf("want 3 rows (A,B,C — D nil + E no PostMatchCSR skipped), got %d", len(rows))
	}
	byXUID := map[string]SharedMatchCSRRow{}
	for _, r := range rows {
		byXUID[r.XUID] = r
	}
	// xuid-A : matured Gold 4 + delta calculé
	a := byXUID["xuid-A"]
	if a.Tier != "Gold" || a.SubTier != 4 {
		t.Errorf("A: want Gold/4, got %s/%d", a.Tier, a.SubTier)
	}
	if a.RatingValue == nil || *a.RatingValue != 1100 {
		t.Errorf("A: rating want 1100, got %v", a.RatingValue)
	}
	if a.RatingDelta == nil || *a.RatingDelta != 15 {
		t.Errorf("A: delta want +15, got %v", a.RatingDelta)
	}
	if a.SeasonID != "CsrSeason13-1" {
		t.Errorf("A: season want CsrSeason13-1, got %s", a.SeasonID)
	}
	// xuid-C : placement remaining=3 → tier=Placement, label formatté, value nil
	c := byXUID["xuid-C"]
	if c.Tier != "Placement" {
		t.Errorf("C: tier want Placement, got %s", c.Tier)
	}
	if c.RatingValue != nil {
		t.Errorf("C: rating_value want nil (placement), got %v", *c.RatingValue)
	}
	if c.MeasurementMatchesRemaining != 3 {
		t.Errorf("C: remaining want 3, got %d", c.MeasurementMatchesRemaining)
	}
	if c.TierLabel != "Placement (3 restants)" {
		t.Errorf("C: label want \"Placement (3 restants)\", got %q", c.TierLabel)
	}
}

func TestExtractAllSharedCSRRows_NonRankedMatch_ReturnsEmpty(t *testing.T) {
	reg := mkReg("m1", "CsrSeason13-1", false)
	skill := map[string]*MatchSkillData{"x": mkSkill("Gold", 1100, 4, 0, -1)}
	if rows := ExtractAllSharedCSRRows(reg, skill); len(rows) != 0 {
		t.Errorf("non-ranked match should return 0 rows, got %d", len(rows))
	}
}

func TestExtractAllSharedCSRRows_TruncatedPayload_Skipped(t *testing.T) {
	// Tier vide + remaining=0 = payload tronqué → skip (cohérent avec
	// ExtractCSRRowIfRanked garde-fou).
	reg := mkReg("m1", "CsrSeason13-1", true)
	skill := map[string]*MatchSkillData{
		"xuid-A": mkSkill("", 0, 0, 0, -1),       // tronqué → skip
		"xuid-B": mkSkill("Gold", 1100, 4, 0, -1), // OK
	}
	rows := ExtractAllSharedCSRRows(reg, skill)
	if len(rows) != 1 {
		t.Errorf("want 1 (B only, A truncated), got %d", len(rows))
	}
}

func TestExtractAllSharedCSRRows_EmptySeasonID_StoredAsEmpty(t *testing.T) {
	reg := mkReg("m1", "", true) // pas de season
	skill := map[string]*MatchSkillData{"x": mkSkill("Gold", 1100, 4, 0, -1)}
	rows := ExtractAllSharedCSRRows(reg, skill)
	if len(rows) != 1 || rows[0].SeasonID != "" {
		t.Errorf("want SeasonID=\"\", got %v", rows)
	}
}

func TestExtractAllSharedCSRRows_NilArgs(t *testing.T) {
	if rows := ExtractAllSharedCSRRows(nil, nil); rows != nil {
		t.Errorf("nil args should return nil, got %v", rows)
	}
	if rows := ExtractAllSharedCSRRows(mkReg("m1", "s", true), nil); rows != nil {
		t.Errorf("nil skill should return nil, got %v", rows)
	}
}

// ─── UpsertSharedCSRs (integration) ─────────────────────────────────────────

func openTempSharedForCSR(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := EnsureSharedSchema(db); err != nil {
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	return db
}

func TestUpsertSharedCSRs_InsertsBatch(t *testing.T) {
	db := openTempSharedForCSR(t)
	v1, v2 := 1100.0, 1850.0
	d1 := 15.0
	rows := []SharedMatchCSRRow{
		{MatchID: "m1", XUID: "xA", RatingType: "CSR", RatingValue: &v1, Tier: "Gold", SubTier: 4, TierLabel: "Or 4", RatingDelta: &d1, SeasonID: "CsrSeason13-1", StartTime: time.Now()},
		{MatchID: "m1", XUID: "xB", RatingType: "CSR", RatingValue: &v2, Tier: "Onyx", SubTier: 0, TierLabel: "Onyx 1850", SeasonID: "CsrSeason13-1", StartTime: time.Now()},
		{MatchID: "m1", XUID: "xC", RatingType: "CSR", Tier: "Placement", MeasurementMatchesRemaining: 3, TierLabel: "Placement (3 restants)", SeasonID: "CsrSeason13-1", StartTime: time.Now()},
	}
	if err := UpsertSharedCSRs(db, rows); err != nil {
		t.Fatalf("UpsertSharedCSRs: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_csrs WHERE match_id='m1'`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Errorf("want 3 rows in match_csrs, got %d", count)
	}
}

func TestUpsertSharedCSRs_UpdateOnConflict(t *testing.T) {
	db := openTempSharedForCSR(t)
	v1 := 1100.0
	row := SharedMatchCSRRow{MatchID: "m1", XUID: "xA", RatingType: "CSR", RatingValue: &v1, Tier: "Gold", SubTier: 4, TierLabel: "Or 4", SeasonID: "CsrSeason13-1", StartTime: time.Now()}
	if err := UpsertSharedCSRs(db, []SharedMatchCSRRow{row}); err != nil {
		t.Fatalf("1er insert: %v", err)
	}
	// Re-write avec valeur différente → UPSERT doit mettre à jour
	v2 := 1200.0
	d := 100.0
	row.RatingValue = &v2
	row.SubTier = 5
	row.TierLabel = "Or 5"
	row.RatingDelta = &d
	if err := UpsertSharedCSRs(db, []SharedMatchCSRRow{row}); err != nil {
		t.Fatalf("UPSERT: %v", err)
	}
	var gotValue float64
	var gotSubTier int
	var gotLabel string
	if err := db.QueryRow(`SELECT rating_value, sub_tier, tier_label FROM match_csrs WHERE match_id='m1' AND xuid='xA'`).
		Scan(&gotValue, &gotSubTier, &gotLabel); err != nil {
		t.Fatalf("select: %v", err)
	}
	if gotValue != 1200 || gotSubTier != 5 || gotLabel != "Or 5" {
		t.Errorf("UPSERT effet: value=%v sub=%d label=%q (want 1200/5/Or 5)", gotValue, gotSubTier, gotLabel)
	}
}

func TestUpsertSharedCSRs_EmptyRows_NoOp(t *testing.T) {
	db := openTempSharedForCSR(t)
	if err := UpsertSharedCSRs(db, nil); err != nil {
		t.Errorf("nil rows: want nil error, got %v", err)
	}
	if err := UpsertSharedCSRs(db, []SharedMatchCSRRow{}); err != nil {
		t.Errorf("empty rows: want nil error, got %v", err)
	}
}

func TestUpsertSharedCSRs_NullableSeasonID(t *testing.T) {
	db := openTempSharedForCSR(t)
	row := SharedMatchCSRRow{
		MatchID: "m1", XUID: "xA", RatingType: "CSR",
		Tier: "Placement", MeasurementMatchesRemaining: 3, TierLabel: "Placement (3)",
		SeasonID: "", // explicitement vide
		StartTime: time.Now(),
	}
	if err := UpsertSharedCSRs(db, []SharedMatchCSRRow{row}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	var sid sql.NullString
	if err := db.QueryRow(`SELECT season_id FROM match_csrs WHERE match_id='m1' AND xuid='xA'`).Scan(&sid); err != nil {
		t.Fatalf("select: %v", err)
	}
	if sid.Valid {
		t.Errorf("season_id should be NULL when SeasonID is empty string, got %q", sid.String)
	}
}

// ─── E2E : sync match ranked avec N participants → N rows shared.match_csrs ──

func TestEndToEnd_ExtractAndUpsert_AllParticipants(t *testing.T) {
	db := openTempSharedForCSR(t)
	reg := mkReg("m-e2e", "CsrSeason13-1", true)
	skill := map[string]*MatchSkillData{
		"xuid-1": mkSkill("Gold", 1100, 4, 0, 1085),
		"xuid-2": mkSkill("Diamond", 1500, 3, 0, 1495),
		"xuid-3": mkSkill("Onyx", 1850, 0, 0, 1830),
		"xuid-4": mkSkill("", 0, 0, 5, -1), // placement (S3+ : remaining=5)
	}
	rows := ExtractAllSharedCSRRows(reg, skill)
	if err := UpsertSharedCSRs(db, rows); err != nil {
		t.Fatalf("E2E upsert: %v", err)
	}
	var total, placement, matured int
	if err := db.QueryRow(`
		SELECT
			COUNT(*),
			SUM(CASE WHEN tier='Placement' THEN 1 ELSE 0 END),
			SUM(CASE WHEN tier IN ('Gold','Diamond','Onyx') THEN 1 ELSE 0 END)
		FROM match_csrs WHERE match_id='m-e2e'
	`).Scan(&total, &placement, &matured); err != nil {
		t.Fatalf("query: %v", err)
	}
	if total != 4 || placement != 1 || matured != 3 {
		t.Errorf("counts: total=%d placement=%d matured=%d (want 4/1/3)", total, placement, matured)
	}
	// Vérifier delta présent pour xuid-2 (Diamond +5)
	var delta sql.NullFloat64
	if err := db.QueryRow(`SELECT rating_delta FROM match_csrs WHERE match_id='m-e2e' AND xuid='xuid-2'`).Scan(&delta); err != nil {
		t.Fatalf("scan delta: %v", err)
	}
	if !delta.Valid || delta.Float64 != 5 {
		t.Errorf("xuid-2 delta: want +5, got %v", delta)
	}
}
