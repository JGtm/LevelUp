package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

const (
	helloLiteral = "hello"
	aquariusMap  = "Aquarius"
)

// ---------- applySessionFilter ----------

func TestApplySessionFilter_Empty(t *testing.T) {
	rows := []domain.FilterMatchRow{
		{MatchID: "m1"},
		{MatchID: "m2"},
	}
	sf := domain.SessionsFilter{}
	got := applySessionFilter(rows, sf)
	if len(got) != 2 {
		t.Errorf("expected 2 (no filter), got %d", len(got))
	}
}

func TestApplySessionFilter_ByLabel(t *testing.T) {
	lbl1, lbl2 := "session-A", "session-B"
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", SessionLabel: &lbl1},
		{MatchID: "m2", SessionLabel: &lbl2},
		{MatchID: "m3"}, // no session
	}
	picked := "session-A"
	sf := domain.SessionsFilter{PickedSessionLabel: &picked}
	got := applySessionFilter(rows, sf)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].MatchID != "m1" {
		t.Errorf("expected m1, got %s", got[0].MatchID)
	}
}

func TestApplySessionFilter_PickedSessions(t *testing.T) {
	lbl1, lbl2 := "s1", "s2"
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", SessionLabel: &lbl1},
		{MatchID: "m2", SessionLabel: &lbl2},
	}
	sf := domain.SessionsFilter{PickedSessions: []string{"s1", "s2"}}
	got := applySessionFilter(rows, sf)
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}
}

// Régression : en prod, pme.session_id est strconv.Itoa(int) ("1", "2", …)
// alors que session_label est un horodatage formaté ("30/04/2026 18:30 (12)").
// FilterOmnibar.SessionPill envoie le session_id dans picked_sessions ; si le
// backend ne matche QUE par SessionLabel, picked_sessions=["1"] ne capture
// jamais une row dont SessionLabel="30/04/2026 18:30 (12)" → counter à 0.
func TestApplySessionFilter_PickedSessions_ByID(t *testing.T) {
	id1, id2 := "1", "2"
	lbl1, lbl2 := "30/04/2026 18:30 (12)", "01/05/2026 14:00 (8)"
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", SessionID: &id1, SessionLabel: &lbl1},
		{MatchID: "m2", SessionID: &id2, SessionLabel: &lbl2},
		{MatchID: "m3"}, // pas de session
	}
	sf := domain.SessionsFilter{PickedSessions: []string{"1"}}
	got := applySessionFilter(rows, sf)
	if len(got) != 1 {
		t.Fatalf("expected 1 (id match), got %d", len(got))
	}
	if got[0].MatchID != "m1" {
		t.Errorf("expected m1, got %s", got[0].MatchID)
	}
}

func TestApplySessionFilter_PickedSessions_MixedIDAndLabel(t *testing.T) {
	id1, id2 := "1", "2"
	lbl1, lbl2 := "30/04/2026 18:30 (12)", "01/05/2026 14:00 (8)"
	id3 := "3"
	lbl3 := "02/05/2026 19:00 (5)"
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", SessionID: &id1, SessionLabel: &lbl1},
		{MatchID: "m2", SessionID: &id2, SessionLabel: &lbl2},
		{MatchID: "m3", SessionID: &id3, SessionLabel: &lbl3},
	}
	// SessionMultiSelect (squad) envoie un label, SessionPill envoie un id —
	// les deux doivent fonctionner dans la même requête sans casser l'autre.
	sf := domain.SessionsFilter{PickedSessions: []string{"1", "01/05/2026 14:00 (8)"}}
	got := applySessionFilter(rows, sf)
	if len(got) != 2 {
		t.Fatalf("expected 2 (id match + label match), got %d", len(got))
	}
}

// ---------- applyExperienceFilter ----------

func TestApplyExperienceFilter_Empty(t *testing.T) {
	rows := []domain.FilterMatchRow{
		{MatchID: "m1"},
	}
	got := applyExperienceFilter(rows, nil)
	if len(got) != 1 {
		t.Errorf("expected 1 (no filter), got %d", len(got))
	}
}

func TestApplyExperienceFilter_PVE(t *testing.T) {
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", IsFirefight: true},
		{MatchID: "m2", IsFirefight: false, IsRanked: false},
	}
	got := applyExperienceFilter(rows, []string{"PVE"})
	if len(got) != 1 {
		t.Fatalf("expected 1 PVE match, got %d", len(got))
	}
	if got[0].MatchID != "m1" {
		t.Errorf("expected m1, got %s", got[0].MatchID)
	}
}

func TestApplyExperienceFilter_Ranked(t *testing.T) {
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", IsFirefight: false, IsRanked: true},
		{MatchID: "m2", IsFirefight: false, IsRanked: false},
	}
	got := applyExperienceFilter(rows, []string{"classé"})
	if len(got) != 1 {
		t.Fatalf("expected 1 ranked match, got %d", len(got))
	}
	if got[0].MatchID != "m1" {
		t.Errorf("expected m1, got %s", got[0].MatchID)
	}
}

// ---------- derefStr ----------

func TestDerefStr_Nil(t *testing.T) {
	if got := derefStr(nil); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestDerefStr_Value(t *testing.T) {
	s := helloLiteral
	if got := derefStr(&s); got != helloLiteral {
		t.Errorf("expected hello, got %s", got)
	}
}

// ---------- applyCascadeFilter ----------

func TestApplyCascadeFilter_NoFilters(t *testing.T) {
	rows := []domain.FilterMatchRow{
		{MatchID: "m1"},
		{MatchID: "m2"},
	}
	cf := domain.CascadeFilter{}
	got := applyCascadeFilter(rows, cf)
	if len(got) != 2 {
		t.Errorf("expected 2, got %d", len(got))
	}
}

// ---------- applyPeriodFilter ----------

func TestApplyPeriodFilter_NoFilter(t *testing.T) {
	rows := []domain.FilterMatchRow{{MatchID: "m1"}}
	got := applyPeriodFilter(rows, domain.PeriodInput{})
	if len(got) != 1 {
		t.Errorf("expected 1, got %d", len(got))
	}
}

func TestApplyPeriodFilter_StartDate(t *testing.T) {
	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	start := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", StartTime: &t1},
		{MatchID: "m2", StartTime: &t2},
	}
	got := applyPeriodFilter(rows, domain.PeriodInput{StartDate: &start})
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].MatchID != "m2" {
		t.Errorf("expected m2, got %s", got[0].MatchID)
	}
}

func TestApplyPeriodFilter_EndDate(t *testing.T) {
	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", StartTime: &t1},
		{MatchID: "m2", StartTime: &t2},
	}
	got := applyPeriodFilter(rows, domain.PeriodInput{EndDate: &end})
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].MatchID != "m1" {
		t.Errorf("expected m1, got %s", got[0].MatchID)
	}
}

func TestApplyPeriodFilter_NilStartTime(t *testing.T) {
	rows := []domain.FilterMatchRow{
		{MatchID: "m1"}, // nil StartTime
	}
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	got := applyPeriodFilter(rows, domain.PeriodInput{StartDate: &start})
	if len(got) != 0 {
		t.Errorf("expected 0 (nil StartTime skipped), got %d", len(got))
	}
}

// ---------- filterBySet ----------

func TestFilterBySet_EmptyValues(t *testing.T) {
	rows := []domain.FilterMatchRow{{MatchID: "m1"}}
	got := filterBySet(rows, nil, playlistUI)
	if len(got) != 1 {
		t.Errorf("expected 1, got %d", len(got))
	}
}

func TestFilterBySet_WithValues(t *testing.T) {
	pl1, pl2 := "Ranked", "Social"
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", PlaylistName: &pl1},
		{MatchID: "m2", PlaylistName: &pl2},
	}
	got := filterBySet(rows, []string{"Ranked"}, playlistUI)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].MatchID != "m1" {
		t.Errorf("expected m1, got %s", got[0].MatchID)
	}
}

// ---------- mapUI ----------

func TestMapUI_FRPreferred(t *testing.T) {
	en, fr := aquariusMap, "Verseau"
	row := domain.FilterMatchRow{MapName: &en, MapNameFR: &fr}
	if got := mapUI(row); got != "Verseau" {
		t.Errorf("expected FR, got %s", got)
	}
}

func TestMapUI_FallbackEN(t *testing.T) {
	en := aquariusMap
	row := domain.FilterMatchRow{MapName: &en}
	if got := mapUI(row); got != aquariusMap {
		t.Errorf("expected EN fallback, got %s", got)
	}
}

// ---------- compareMatchHistoryRows ----------

func TestCompareMatchHistoryRows_OutcomeCode(t *testing.T) {
	a := domain.MatchHistoryRow{OutcomeCode: 1, StartTime: time.Now()}
	b := domain.MatchHistoryRow{OutcomeCode: 2, StartTime: time.Now()}
	if !compareMatchHistoryRows(a, b, "outcome_code") {
		t.Error("expected a < b for outcome_code")
	}
}

func TestCompareMatchHistoryRows_TeamMMR(t *testing.T) {
	m1, m2 := 1200.0, 1500.0
	a := domain.MatchHistoryRow{TeamMMR: &m1, StartTime: time.Now()}
	b := domain.MatchHistoryRow{TeamMMR: &m2, StartTime: time.Now()}
	if !compareMatchHistoryRows(a, b, "team_mmr") {
		t.Error("expected a < b for team_mmr")
	}
}

func TestCompareMatchHistoryRows_DefaultStartTime(t *testing.T) {
	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	a := domain.MatchHistoryRow{StartTime: t1}
	b := domain.MatchHistoryRow{StartTime: t2}
	if !compareMatchHistoryRows(a, b, "start_time") {
		t.Error("expected a < b for start_time")
	}
}

func TestCompareMatchHistoryRows_DeltaMMR(t *testing.T) {
	d1, d2 := -50.0, 100.0
	a := domain.MatchHistoryRow{DeltaMMR: &d1, StartTime: time.Now()}
	b := domain.MatchHistoryRow{DeltaMMR: &d2, StartTime: time.Now()}
	if !compareMatchHistoryRows(a, b, "delta_mmr") {
		t.Error("expected a < b for delta_mmr")
	}
}

func TestCompareMatchHistoryRows_WinRate(t *testing.T) {
	w1, w2 := 40.0, 60.0
	a := domain.MatchHistoryRow{WinRateHist: &w1, StartTime: time.Now()}
	b := domain.MatchHistoryRow{WinRateHist: &w2, StartTime: time.Now()}
	if !compareMatchHistoryRows(a, b, "win_rate_hist") {
		t.Error("expected a < b for win_rate_hist")
	}
}

func TestCompareMatchHistoryRows_PerfScore(t *testing.T) {
	p1, p2 := 30, 80
	a := domain.MatchHistoryRow{PerformanceScoreRelative: &p1, StartTime: time.Now()}
	b := domain.MatchHistoryRow{PerformanceScoreRelative: &p2, StartTime: time.Now()}
	if !compareMatchHistoryRows(a, b, "performance_score_relative") {
		t.Error("expected a < b for perf score")
	}
}
