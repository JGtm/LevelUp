package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// ---------- formatDateFR ----------

func TestFormatDateFR_Zero(t *testing.T) {
	if got := formatDateFR(time.Time{}); got != "-" {
		t.Errorf("expected '-', got %s", got)
	}
}

func TestFormatDateFR_Valid(t *testing.T) {
	d := time.Date(2025, 3, 15, 14, 30, 0, 0, time.UTC)
	if got := formatDateFR(d); got != "15/03/2025 14:30" {
		t.Errorf("expected '15/03/2025 14:30', got %s", got)
	}
}

// ---------- formatLifeSeconds ----------

func TestFormatLifeSeconds_Nil(t *testing.T) {
	if got := formatLifeSeconds(nil); got != "0:00" {
		t.Errorf("expected '0:00', got %s", got)
	}
}

func TestFormatLifeSeconds_Negative(t *testing.T) {
	v := -5.0
	if got := formatLifeSeconds(&v); got != "0:00" {
		t.Errorf("expected '0:00', got %s", got)
	}
}

func TestFormatLifeSeconds_Normal(t *testing.T) {
	v := 125.0
	if got := formatLifeSeconds(&v); got != "2:05" {
		t.Errorf("expected '2:05', got %s", got)
	}
}

// (buildMatchURL supprimé en F3 : la construction d'URL Waypoint vit désormais
// dans halo_infinite.AssetURLAdapter.{Match,PlayerMatch}WebURL, testée là-bas.)

// ---------- buildPeriodLabel ----------

func TestBuildPeriodLabel_SessionMode(t *testing.T) {
	in := domain.FilterContextInput{FilterMode: "sessions"}
	if got := buildPeriodLabel(in); got != nil {
		t.Errorf("expected nil for sessions mode, got %v", *got)
	}
}

func TestBuildPeriodLabel_NoDates(t *testing.T) {
	if got := buildPeriodLabel(domain.FilterContextInput{}); got != nil {
		t.Errorf("expected nil, got %v", *got)
	}
}

func TestBuildPeriodLabel_StartAndEnd(t *testing.T) {
	s := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	e := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	in := domain.FilterContextInput{Period: domain.PeriodInput{StartDate: &s, EndDate: &e}}
	got := buildPeriodLabel(in)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if *got != "01/01/2025 → 01/03/2025" {
		t.Errorf("got %s", *got)
	}
}

func TestBuildPeriodLabel_StartOnly(t *testing.T) {
	s := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	in := domain.FilterContextInput{Period: domain.PeriodInput{StartDate: &s}}
	got := buildPeriodLabel(in)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if *got != "01/01/2025" {
		t.Errorf("got %s", *got)
	}
}

// ---------- coalesce ----------

func TestCoalesce_FirstNonEmpty(t *testing.T) {
	a := "hello"
	if got := coalesce(&a, nil); got != "hello" {
		t.Errorf("got %s", got)
	}
}

func TestCoalesce_FallbackB(t *testing.T) {
	b := "world"
	if got := coalesce(nil, &b); got != "world" {
		t.Errorf("got %s", got)
	}
}

func TestCoalesce_BothNil(t *testing.T) {
	if got := coalesce(nil, nil); got != "" {
		t.Errorf("got %s", got)
	}
}

// ---------- paginate ----------

func TestPaginate_FirstPage(t *testing.T) {
	items := make([]domain.MatchHistoryRow, 25)
	for i := range items {
		items[i].MatchID = "m"
	}
	meta, page := paginate(items, domain.PaginationRequest{Page: 1, PageSize: 10})
	if len(page) != 10 {
		t.Errorf("expected 10, got %d", len(page))
	}
	if meta.Total != 25 {
		t.Errorf("expected total 25, got %d", meta.Total)
	}
	if !meta.HasNext {
		t.Error("expected HasNext=true")
	}
}

func TestPaginate_LastPartialPage(t *testing.T) {
	items := make([]domain.MatchHistoryRow, 25)
	_, page := paginate(items, domain.PaginationRequest{Page: 3, PageSize: 10})
	if len(page) != 5 {
		t.Errorf("expected 5, got %d", len(page))
	}
}

func TestPaginate_EmptyItems(t *testing.T) {
	meta, page := paginate(nil, domain.PaginationRequest{Page: 1, PageSize: 10})
	if len(page) != 0 || meta.Total != 0 {
		t.Errorf("expected empty result")
	}
}

// ---------- normalizeInput ----------

func TestNormalizeInput_SwapDates(t *testing.T) {
	s := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	e := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	in := domain.FilterContextInput{Period: domain.PeriodInput{StartDate: &s, EndDate: &e}}
	out := normalizeInput(in)
	if !out.Period.StartDate.Equal(e) || !out.Period.EndDate.Equal(s) {
		t.Error("expected dates swapped")
	}
}

// ---------- buildSessionOptions ----------

func TestBuildSessionOptions_Empty(t *testing.T) {
	got := buildSessionOptions(nil, domain.CascadeFilter{})
	if len(got.AllSessions) != 0 {
		t.Errorf("expected 0 sessions")
	}
}

func TestBuildSessionOptions_MixedSoloSquad(t *testing.T) {
	lbl1, lbl2 := "Session A", "Session B"
	sid1, sid2 := "s1", "s2"
	// 2 matchs par session (sinon filtrées par minListedSessionMatches).
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", SessionLabel: &lbl1, SessionID: &sid1, IsWithFriends: false},
		{MatchID: "m2", SessionLabel: &lbl1, SessionID: &sid1, IsWithFriends: false},
		{MatchID: "m3", SessionLabel: &lbl2, SessionID: &sid2, IsWithFriends: true},
		{MatchID: "m4", SessionLabel: &lbl2, SessionID: &sid2, IsWithFriends: true},
	}
	got := buildSessionOptions(rows, domain.CascadeFilter{})
	if len(got.AllSessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(got.AllSessions))
	}
	if len(got.SoloLabels) != 1 || len(got.SquadLabels) != 1 {
		t.Errorf("solo=%d squad=%d", len(got.SoloLabels), len(got.SquadLabels))
	}
}

func TestBuildSessionOptions_DropsSingleMatch(t *testing.T) {
	lblKeep, lblDrop := "Session Keep", "Session Drop"
	sidKeep, sidDrop := "sk", "sd"
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", SessionLabel: &lblKeep, SessionID: &sidKeep, IsWithFriends: false},
		{MatchID: "m2", SessionLabel: &lblKeep, SessionID: &sidKeep, IsWithFriends: false},
		{MatchID: "m3", SessionLabel: &lblDrop, SessionID: &sidDrop, IsWithFriends: true}, // 1 match → exclu
	}
	got := buildSessionOptions(rows, domain.CascadeFilter{})
	if len(got.AllSessions) != 1 || got.AllSessions[0].Label != lblKeep {
		t.Fatalf("expected only %q listed, got %+v", lblKeep, got.AllSessions)
	}
	if len(got.SquadLabels) != 0 {
		t.Errorf("single-match squad session should be dropped, got SquadLabels=%v", got.SquadLabels)
	}
}
