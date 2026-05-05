package analysis

import (
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// fakeCategoryPrefixes : stub pour les tests. Retourne quelques cas réalistes
// + une catégorie inconnue pour vérifier la dégradation.
func fakeCategoryPrefixes(cat string) []string {
	switch cat {
	case "BTB":
		return []string{"BTB", "BTB Heavies"}
	case "Ranked":
		return []string{"Ranked"}
	case "Husky":
		return []string{"Husky Raid"}
	case "":
		return nil
	}
	return nil // catégorie inconnue
}

func TestBuildNeighborsWhereClause_Empty(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		spec *domain.MatchFilterSpec
	}{
		{"nil spec", nil},
		{"spec sans champs", &domain.MatchFilterSpec{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildNeighborsWhereClause(tt.spec, fakeCategoryPrefixes)
			if got.SQL != "" || len(got.Args) != 0 {
				t.Errorf("expected empty result, got SQL=%q Args=%v", got.SQL, got.Args)
			}
		})
	}
}

func TestBuildNeighborsWhereClause_Playlist(t *testing.T) {
	t.Parallel()
	pl := "Ranked Arena"
	got := BuildNeighborsWhereClause(&domain.MatchFilterSpec{PlaylistName: &pl}, fakeCategoryPrefixes)
	if !strings.Contains(got.SQL, "mr.playlist_name = ?") {
		t.Errorf("SQL missing playlist clause: %q", got.SQL)
	}
	if len(got.Args) != 1 || got.Args[0] != "Ranked Arena" {
		t.Errorf("Args want [Ranked Arena], got %v", got.Args)
	}
}

func TestBuildNeighborsWhereClause_ModeCategory_BTB(t *testing.T) {
	t.Parallel()
	cat := "BTB"
	got := BuildNeighborsWhereClause(&domain.MatchFilterSpec{ModeCategory: &cat}, fakeCategoryPrefixes)
	// Doit générer 2 préfixes × 2 clauses chacune = 4 clauses OR
	if !strings.Contains(got.SQL, "mr.pair_name = ?") {
		t.Errorf("SQL missing pair_name equality: %q", got.SQL)
	}
	if !strings.Contains(got.SQL, "mr.pair_name LIKE ?") {
		t.Errorf("SQL missing pair_name LIKE: %q", got.SQL)
	}
	if len(got.Args) != 4 {
		t.Errorf("Args length want 4 (2 prefixes × 2 patterns), got %d : %v", len(got.Args), got.Args)
	}
	// Args attendus : "BTB", "BTB:%", "BTB Heavies", "BTB Heavies:%"
	expectedArgs := []any{"BTB", "BTB:%", "BTB Heavies", "BTB Heavies:%"}
	for i, want := range expectedArgs {
		if got.Args[i] != want {
			t.Errorf("Args[%d] = %v, want %v", i, got.Args[i], want)
		}
	}
}

func TestBuildNeighborsWhereClause_ModeCategory_Unknown(t *testing.T) {
	t.Parallel()
	cat := "InventedCategory"
	got := BuildNeighborsWhereClause(&domain.MatchFilterSpec{ModeCategory: &cat}, fakeCategoryPrefixes)
	if got.SQL != "" {
		t.Errorf("catégorie inconnue : SQL doit être vide, got %q", got.SQL)
	}
	if len(got.IgnoredFilters) != 1 || got.IgnoredFilters[0] != "mode_category" {
		t.Errorf("IgnoredFilters want [mode_category], got %v", got.IgnoredFilters)
	}
}

func TestBuildNeighborsWhereClause_ModeCategory_NilResolver(t *testing.T) {
	t.Parallel()
	cat := "BTB"
	got := BuildNeighborsWhereClause(&domain.MatchFilterSpec{ModeCategory: &cat}, nil)
	if got.SQL != "" {
		t.Errorf("resolver nil : SQL doit être vide, got %q", got.SQL)
	}
	if len(got.IgnoredFilters) != 1 || got.IgnoredFilters[0] != "mode_category" {
		t.Errorf("IgnoredFilters want [mode_category], got %v", got.IgnoredFilters)
	}
}

func TestBuildNeighborsWhereClause_DateFromTo(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 1, 23, 59, 59, 0, time.UTC)
	got := BuildNeighborsWhereClause(
		&domain.MatchFilterSpec{DateFrom: &from, DateTo: &to},
		fakeCategoryPrefixes,
	)
	if !strings.Contains(got.SQL, ">=") || !strings.Contains(got.SQL, "<=") {
		t.Errorf("SQL must contain >= and <=, got %q", got.SQL)
	}
	if !strings.Contains(got.SQL, "AT TIME ZONE 'UTC'") {
		t.Errorf("SQL must use canonical timezone pattern, got %q", got.SQL)
	}
	if len(got.Args) != 2 {
		t.Errorf("Args want 2 dates, got %d", len(got.Args))
	}
}

func TestBuildNeighborsWhereClause_Outcome(t *testing.T) {
	t.Parallel()
	cases := []struct {
		label    string
		wantCode int
		ignored  bool
	}{
		{"win", 2, false},
		{"loss", 3, false},
		{"draw", 1, false},
		{"dnf", 4, false},
		{"BAD", 0, true},
		{"", 0, true}, // chaîne vide n'entre pas (skip silencieux), pas de "ignored" non plus
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.label, func(t *testing.T) {
			outcome := tc.label
			got := BuildNeighborsWhereClause(&domain.MatchFilterSpec{Outcome: &outcome}, fakeCategoryPrefixes)

			if tc.label == "" {
				// Empty string : on ne tente même pas, pas d'ignoré
				if got.SQL != "" {
					t.Errorf("empty outcome: SQL want empty, got %q", got.SQL)
				}
				return
			}

			if tc.ignored {
				if got.SQL != "" {
					t.Errorf("invalid outcome %q: SQL want empty, got %q", tc.label, got.SQL)
				}
				if len(got.IgnoredFilters) != 1 || got.IgnoredFilters[0] != "outcome" {
					t.Errorf("invalid outcome %q: IgnoredFilters want [outcome], got %v", tc.label, got.IgnoredFilters)
				}
				return
			}

			if !strings.Contains(got.SQL, "mp.outcome = ?") {
				t.Errorf("outcome %q: SQL missing clause, got %q", tc.label, got.SQL)
			}
			if len(got.Args) != 1 || got.Args[0] != tc.wantCode {
				t.Errorf("outcome %q: Args want [%d], got %v", tc.label, tc.wantCode, got.Args)
			}
		})
	}
}

func TestBuildNeighborsWhereClause_SessionID_Ignored(t *testing.T) {
	t.Parallel()
	sid := "session-abc"
	got := BuildNeighborsWhereClause(&domain.MatchFilterSpec{SessionID: &sid}, fakeCategoryPrefixes)
	// Phase 2b initial : session_id n'est pas implémenté côté SQL (player DB
	// non jointe dans Q25). On vérifie qu'il est marqué "ignored".
	if got.SQL != "" {
		t.Errorf("session_id seul : SQL doit être vide, got %q", got.SQL)
	}
	if len(got.IgnoredFilters) != 1 || got.IgnoredFilters[0] != "session_id" {
		t.Errorf("IgnoredFilters want [session_id], got %v", got.IgnoredFilters)
	}
}

func TestBuildNeighborsWhereClause_Combined(t *testing.T) {
	t.Parallel()
	pl := "Ranked Arena"
	cat := "Ranked"
	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	outcome := "win"
	got := BuildNeighborsWhereClause(&domain.MatchFilterSpec{
		PlaylistName: &pl,
		ModeCategory: &cat,
		DateFrom:     &from,
		Outcome:      &outcome,
	}, fakeCategoryPrefixes)

	if !strings.HasPrefix(got.SQL, " AND ") {
		t.Errorf("SQL must start with ' AND ', got %q", got.SQL)
	}
	// 4 clauses jointes par AND
	if strings.Count(got.SQL, " AND ") < 4 {
		t.Errorf("SQL must contain ≥4 AND tokens, got %d in %q", strings.Count(got.SQL, " AND "), got.SQL)
	}
	// Ranked → 1 préfixe × 2 patterns = 2 args, +1 playlist, +1 from, +1 outcome = 5
	if len(got.Args) != 5 {
		t.Errorf("Args want 5, got %d : %v", len(got.Args), got.Args)
	}
}

func TestIsValidOutcomeLabel(t *testing.T) {
	t.Parallel()
	for _, ok := range []string{"win", "loss", "draw", "dnf"} {
		if !IsValidOutcomeLabel(ok) {
			t.Errorf("IsValidOutcomeLabel(%q) want true", ok)
		}
	}
	for _, bad := range []string{"", "victory", "WIN", "abandoned"} {
		if IsValidOutcomeLabel(bad) {
			t.Errorf("IsValidOutcomeLabel(%q) want false", bad)
		}
	}
}
