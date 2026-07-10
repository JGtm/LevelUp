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

// TestModeTaxonomy_NilSafeAndDelegates verrouille le contrat du seam F1 : une
// ModeTaxonomy zéro-value ne panique jamais (dégradation gracieuse pour un titre
// sans classification) et une taxonomie injectée délègue bien.
func TestModeTaxonomy_NilSafe(t *testing.T) {
	var zero ModeTaxonomy // titre sans taxonomie
	if got := zero.Classify("Arena:Slayer on X"); got != "" {
		t.Errorf("Classify zéro-value = %q, want \"\"", got)
	}
	if got := zero.Prefixes("BTB"); got != nil {
		t.Errorf("Prefixes zéro-value = %v, want nil", got)
	}
	if got := zero.KnownPrefixes(); got != nil {
		t.Errorf("KnownPrefixes zéro-value = %v, want nil", got)
	}
}

func TestModeTaxonomy_Delegates(t *testing.T) {
	tax := ModeTaxonomy{
		InferCategory: func(pn string) string {
			if strings.HasPrefix(pn, "BTB") {
				return "BTB"
			}
			return "Other"
		},
		PrefixesFor: fakeCategoryPrefixes,
		AllPrefixes: func() []string { return []string{"BTB", "Ranked", "Husky Raid"} },
		Other:       "Other",
	}
	if got := tax.Classify("BTB Heavies on Y"); got != "BTB" {
		t.Errorf("Classify = %q, want BTB", got)
	}
	if got := tax.Classify("Arena:Slayer"); got != "Other" {
		t.Errorf("Classify = %q, want Other", got)
	}
	if got := tax.Prefixes("Ranked"); len(got) != 1 || got[0] != "Ranked" {
		t.Errorf("Prefixes(Ranked) = %v", got)
	}
	if got := tax.KnownPrefixes(); len(got) != 3 {
		t.Errorf("KnownPrefixes = %v, want 3", got)
	}
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
	got := BuildNeighborsWhereClause(
		&domain.MatchFilterSpec{PlaylistNames: []string{"Ranked Arena"}},
		fakeCategoryPrefixes,
	)
	if !strings.Contains(got.SQL, "mr.playlist_name IN (?)") {
		t.Errorf("SQL missing playlist IN clause: %q", got.SQL)
	}
	if len(got.Args) != 1 || got.Args[0] != "Ranked Arena" {
		t.Errorf("Args want [Ranked Arena], got %v", got.Args)
	}
}

func TestBuildNeighborsWhereClause_MultiPlaylist(t *testing.T) {
	t.Parallel()
	got := BuildNeighborsWhereClause(
		&domain.MatchFilterSpec{PlaylistNames: []string{"Ranked Arena", "Big Team Battle"}},
		fakeCategoryPrefixes,
	)
	if !strings.Contains(got.SQL, "mr.playlist_name IN (?, ?)") {
		t.Errorf("SQL want 2-placeholder IN, got %q", got.SQL)
	}
	if len(got.Args) != 2 || got.Args[0] != "Ranked Arena" || got.Args[1] != "Big Team Battle" {
		t.Errorf("Args want [Ranked Arena, Big Team Battle], got %v", got.Args)
	}
}

func TestBuildNeighborsWhereClause_Playlist_SkipsEmpty(t *testing.T) {
	t.Parallel()
	// Slice avec valeurs vides → filtrées ; pas de placeholder fantôme.
	got := BuildNeighborsWhereClause(
		&domain.MatchFilterSpec{PlaylistNames: []string{"", "  ", "Fiesta"}},
		fakeCategoryPrefixes,
	)
	if !strings.Contains(got.SQL, "mr.playlist_name IN (?)") {
		t.Errorf("SQL want single placeholder (empties filtered), got %q", got.SQL)
	}
	if len(got.Args) != 1 || got.Args[0] != "Fiesta" {
		t.Errorf("Args want [Fiesta], got %v", got.Args)
	}
}

func TestBuildNeighborsWhereClause_ModeCategory_BTB(t *testing.T) {
	t.Parallel()
	got := BuildNeighborsWhereClause(
		&domain.MatchFilterSpec{ModeCategories: []string{"BTB"}},
		fakeCategoryPrefixes,
	)
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

func TestBuildNeighborsWhereClause_MultiModeCategory(t *testing.T) {
	t.Parallel()
	// BTB (2 préfixes) + Ranked (1 préfixe) = 3 préfixes × 2 patterns = 6 args,
	// le tout dans un seul OR.
	got := BuildNeighborsWhereClause(
		&domain.MatchFilterSpec{ModeCategories: []string{"BTB", "Ranked"}},
		fakeCategoryPrefixes,
	)
	if strings.Count(got.SQL, " OR ") != 5 {
		t.Errorf("SQL want 5 OR tokens (6 clauses), got %d in %q", strings.Count(got.SQL, " OR "), got.SQL)
	}
	if len(got.Args) != 6 {
		t.Errorf("Args want 6 (3 prefixes × 2 patterns), got %d : %v", len(got.Args), got.Args)
	}
	if len(got.IgnoredFilters) != 0 {
		t.Errorf("aucune catégorie ignorée attendue, got %v", got.IgnoredFilters)
	}
}

func TestBuildNeighborsWhereClause_MultiModeCategory_PartialResolve(t *testing.T) {
	t.Parallel()
	// BTB résout, "InventedCategory" non → on garde BTB, pas d'ignored
	// (au moins une catégorie a matché).
	got := BuildNeighborsWhereClause(
		&domain.MatchFilterSpec{ModeCategories: []string{"BTB", "InventedCategory"}},
		fakeCategoryPrefixes,
	)
	if len(got.Args) != 4 {
		t.Errorf("Args want 4 (BTB seul résout), got %d : %v", len(got.Args), got.Args)
	}
	if len(got.IgnoredFilters) != 0 {
		t.Errorf("partial resolve : pas d'ignored (BTB a matché), got %v", got.IgnoredFilters)
	}
}

func TestBuildNeighborsWhereClause_ModeCategory_Unknown(t *testing.T) {
	t.Parallel()
	got := BuildNeighborsWhereClause(
		&domain.MatchFilterSpec{ModeCategories: []string{"InventedCategory"}},
		fakeCategoryPrefixes,
	)
	if got.SQL != "" {
		t.Errorf("catégorie inconnue : SQL doit être vide, got %q", got.SQL)
	}
	if len(got.IgnoredFilters) != 1 || got.IgnoredFilters[0] != "mode_category" {
		t.Errorf("IgnoredFilters want [mode_category], got %v", got.IgnoredFilters)
	}
}

func TestBuildNeighborsWhereClause_ModeCategory_NilResolver(t *testing.T) {
	t.Parallel()
	got := BuildNeighborsWhereClause(&domain.MatchFilterSpec{ModeCategories: []string{"BTB"}}, nil)
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

func TestBuildNeighborsWhereClause_WithPlayerXuid(t *testing.T) {
	t.Parallel()
	xuid := "2533274791785593"
	got := BuildNeighborsWhereClause(
		&domain.MatchFilterSpec{WithPlayerXuid: &xuid},
		fakeCategoryPrefixes,
	)
	if !strings.Contains(got.SQL, "EXISTS (SELECT 1 FROM match_participants mp2") {
		t.Errorf("SQL missing EXISTS clause: %q", got.SQL)
	}
	if !strings.Contains(got.SQL, "mp2.match_id = mr.match_id") {
		t.Errorf("SQL missing match_id correlation: %q", got.SQL)
	}
	if !strings.Contains(got.SQL, "mp2.xuid = ?") {
		t.Errorf("SQL missing xuid placeholder: %q", got.SQL)
	}
	if len(got.Args) != 1 || got.Args[0] != xuid {
		t.Errorf("Args want [%s], got %v", xuid, got.Args)
	}
}

func TestBuildNeighborsWhereClause_Combined(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	outcome := "win"
	got := BuildNeighborsWhereClause(&domain.MatchFilterSpec{
		PlaylistNames:  []string{"Ranked Arena"},
		ModeCategories: []string{"Ranked"},
		DateFrom:       &from,
		Outcome:        &outcome,
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
