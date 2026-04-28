// Tests purs pour BuildSessionLabelsList et filterMatchHistoryRowsBySoloSessions.
//
// Couverture :
//   - BuildSessionLabelsList : split solo/squad, bornes min/max, tri DESC,
//     skip label vide, multi-titres ignorés (le helper est title-agnostic).
//   - filterMatchHistoryRowsBySoloSessions : labels vide = passthrough,
//     squad rows exclues, label nil exclu, set match case-sensitive.
package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// BuildSessionLabelsList
// ---------------------------------------------------------------------------

func TestBuildSessionLabelsList_Empty(t *testing.T) {
	got := BuildSessionLabelsList(nil)
	if len(got.Solo) != 0 || len(got.Squad) != 0 {
		t.Errorf("expected empty Solo+Squad, got solo=%d squad=%d", len(got.Solo), len(got.Squad))
	}
}

func TestBuildSessionLabelsList_SkipsEmptyLabel(t *testing.T) {
	inputs := []SessionLabelInput{
		{Label: "", StartTime: time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC), IsWithFriends: false},
		{Label: "S1", StartTime: time.Date(2026, 4, 28, 11, 0, 0, 0, time.UTC), IsWithFriends: false},
	}
	got := BuildSessionLabelsList(inputs)
	if len(got.Solo) != 1 {
		t.Fatalf("empty label should be skipped: solo=%d", len(got.Solo))
	}
	if got.Solo[0].Label != "S1" {
		t.Errorf("expected S1 only, got %q", got.Solo[0].Label)
	}
}

func TestBuildSessionLabelsList_SplitSoloSquad(t *testing.T) {
	t1 := time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(2 * time.Hour)
	inputs := []SessionLabelInput{
		{Label: "Solo-A", StartTime: t1, IsWithFriends: false},
		{Label: "Squad-B", StartTime: t2, IsWithFriends: true},
	}
	got := BuildSessionLabelsList(inputs)
	if len(got.Solo) != 1 || got.Solo[0].Label != "Solo-A" {
		t.Errorf("Solo: expected [Solo-A], got %+v", got.Solo)
	}
	if len(got.Squad) != 1 || got.Squad[0].Label != "Squad-B" {
		t.Errorf("Squad: expected [Squad-B], got %+v", got.Squad)
	}
}

func TestBuildSessionLabelsList_MinMaxPerLabel(t *testing.T) {
	// Plusieurs matchs même label : StartedAt = min, EndedAt = max.
	t1 := time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(1 * time.Hour)
	t3 := t1.Add(3 * time.Hour)
	inputs := []SessionLabelInput{
		{Label: "S1", StartTime: t2, IsWithFriends: false}, // milieu
		{Label: "S1", StartTime: t1, IsWithFriends: false}, // début (min)
		{Label: "S1", StartTime: t3, IsWithFriends: false}, // fin (max)
	}
	got := BuildSessionLabelsList(inputs)
	if len(got.Solo) != 1 {
		t.Fatalf("expected 1 solo entry, got %d", len(got.Solo))
	}
	if !got.Solo[0].StartedAt.Equal(t1) {
		t.Errorf("StartedAt: expected min %v, got %v", t1, got.Solo[0].StartedAt)
	}
	if !got.Solo[0].EndedAt.Equal(t3) {
		t.Errorf("EndedAt: expected max %v, got %v", t3, got.Solo[0].EndedAt)
	}
}

func TestBuildSessionLabelsList_SortedDESC(t *testing.T) {
	// 3 sessions solo → tri par StartedAt DESC (la plus récente en tête).
	t1 := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)
	inputs := []SessionLabelInput{
		{Label: "S-Old", StartTime: t1, IsWithFriends: false},
		{Label: "S-Newest", StartTime: t3, IsWithFriends: false},
		{Label: "S-Mid", StartTime: t2, IsWithFriends: false},
	}
	got := BuildSessionLabelsList(inputs)
	if len(got.Solo) != 3 {
		t.Fatalf("expected 3 solo entries, got %d", len(got.Solo))
	}
	if got.Solo[0].Label != "S-Newest" {
		t.Errorf("first should be newest, got %q", got.Solo[0].Label)
	}
	if got.Solo[2].Label != "S-Old" {
		t.Errorf("last should be oldest, got %q", got.Solo[2].Label)
	}
}

// ---------------------------------------------------------------------------
// filterMatchHistoryRowsBySoloSessions
// ---------------------------------------------------------------------------

func makeRow(matchID, sessionLabel string, isWithFriends bool) domain.MatchHistoryRawRow {
	var lbl *string
	if sessionLabel != "" {
		s := sessionLabel
		lbl = &s
	}
	return domain.MatchHistoryRawRow{
		MatchID:       matchID,
		SessionLabel:  lbl,
		IsWithFriends: isWithFriends,
	}
}

func TestFilterMatchHistoryRowsBySoloSessions_EmptyLabels_Passthrough(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{makeRow("m1", "S1", false)}
	got := filterMatchHistoryRowsBySoloSessions(rows, nil)
	if len(got) != 1 {
		t.Errorf("empty labels list should passthrough, got %d", len(got))
	}
}

func TestFilterMatchHistoryRowsBySoloSessions_KeepsSoloOnly(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{
		makeRow("m1", "S1", false), // solo S1 — keep
		makeRow("m2", "S1", true),  // squad S1 — exclude (squad)
		makeRow("m3", "S2", false), // solo S2 — exclude (label not in allow)
		makeRow("m4", "", false),   // solo no-label — exclude (nil session)
	}
	got := filterMatchHistoryRowsBySoloSessions(rows, []string{"S1"})
	if len(got) != 1 {
		t.Fatalf("expected 1 row, got %d", len(got))
	}
	if got[0].MatchID != "m1" {
		t.Errorf("expected m1, got %s", got[0].MatchID)
	}
}

func TestFilterMatchHistoryRowsBySoloSessions_MultipleLabels(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{
		makeRow("m1", "S1", false),
		makeRow("m2", "S2", false),
		makeRow("m3", "S3", false),
	}
	got := filterMatchHistoryRowsBySoloSessions(rows, []string{"S1", "S3"})
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
}

func TestFilterMatchHistoryRowsBySoloSessions_CaseSensitive(t *testing.T) {
	// Les labels sont des identifiants normalisés côté ingestion — match case-sensitive.
	rows := []domain.MatchHistoryRawRow{makeRow("m1", "Session-A", false)}
	got := filterMatchHistoryRowsBySoloSessions(rows, []string{"session-a"})
	if len(got) != 0 {
		t.Errorf("case mismatch should not match, got %d", len(got))
	}
}

func TestFilterMatchHistoryRowsBySoloSessions_PreservesOrder(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{
		makeRow("m1", "S1", false),
		makeRow("m2", "S2", false), // squad — excluded by allow-list
		makeRow("m3", "S1", false),
	}
	got := filterMatchHistoryRowsBySoloSessions(rows, []string{"S1"})
	if len(got) != 2 || got[0].MatchID != "m1" || got[1].MatchID != "m3" {
		t.Errorf("expected [m1, m3] in order, got %v", got)
	}
}
