// Tests pour les fonctions computeAvailable* qui calculent les counts
// cascade-aware (sémantique OR) sur les 5 dimensions Explorer-spécifiques :
// outcome, perf_tier, skill_tier, ranked_context, squad_scope.
//
// Stratégie : un dataset minimal de 8 rows couvrant chaque combinaison testable.
// Chaque test instancie une MatchHistoryQueryRequest avec un sous-ensemble de
// filtres et vérifie les counts retournés.
package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// ptrFloat retourne un pointeur sur la valeur — pratique pour les fixtures.
func ptrFloat(v float64) *float64 { return &v }

// fixtureExplorerRows construit un dataset varié pour tester les 5 dimensions.
//
//	idx | outcome | perf | skill   | ranked | with_friends | firefight
//	----|---------|------|---------|--------|--------------|----------
//	 0  |  2 Win  | 90   | Diamond |  yes   |   yes        |  no
//	 1  |  2 Win  | 85   | Diamond |  yes   |   no         |  no
//	 2  |  3 Loss | 40   | Onyx    |  yes   |   yes        |  no
//	 3  |  3 Loss | 60   | Gold    |  no    |   yes        |  no
//	 4  |  1 Tie  | 50   | nil     |  no    |   no         |  no
//	 5  |  4 DNF  | nil  | nil     |  no    |   no         |  no
//	 6  |  2 Win  | 95   | Onyx    |  yes   |   yes        |  no
//	 7  |  2 Win  | 70   | nil     |  no    |   yes        |  yes (PVE)
//
// Tiers perf calculés par analysis.PerfTier sur PerformanceScore (seuils 80/65/50/35) :
//   - 85, 90, 95 → tier 1 (≥80)
//   - 70         → tier 2 (≥65)
//   - 50, 60     → tier 3 (≥50)
//   - 40         → tier 4 (≥35)
//   - <35        → tier 5
//   - nil        → exclu
func fixtureExplorerRows() []domain.MatchHistoryRawRow {
	return []domain.MatchHistoryRawRow{
		{MatchID: "m0", Outcome: 2, PerformanceScore: ptrFloat(90), SkillTier: ptrStr("Diamond"), IsRanked: true, IsWithFriends: true},
		{MatchID: "m1", Outcome: 2, PerformanceScore: ptrFloat(85), SkillTier: ptrStr("Diamond"), IsRanked: true, IsWithFriends: false},
		{MatchID: "m2", Outcome: 3, PerformanceScore: ptrFloat(40), SkillTier: ptrStr("Onyx"), IsRanked: true, IsWithFriends: true},
		{MatchID: "m3", Outcome: 3, PerformanceScore: ptrFloat(60), SkillTier: ptrStr("Gold"), IsRanked: false, IsWithFriends: true},
		{MatchID: "m4", Outcome: 1, PerformanceScore: ptrFloat(50), IsRanked: false, IsWithFriends: false},
		{MatchID: "m5", Outcome: 4, PerformanceScore: nil, IsRanked: false, IsWithFriends: false},
		{MatchID: "m6", Outcome: 2, PerformanceScore: ptrFloat(95), SkillTier: ptrStr("Onyx"), IsRanked: true, IsWithFriends: true},
		{MatchID: "m7", Outcome: 2, PerformanceScore: ptrFloat(70), IsRanked: false, IsWithFriends: true, IsFirefight: true},
	}
}

// findCount cherche le LabelValue dont Value == value et retourne son Count.
// Retourne -1 si introuvable (échec assertion).
func findCount(opts []domain.LabelValue, value string) int {
	for _, o := range opts {
		if o.Value == value {
			return o.Count
		}
	}
	return -1
}

// ---------------------------------------------------------------------------
// computeAvailableOutcomes
// ---------------------------------------------------------------------------

func TestComputeAvailableOutcomes_NoFilter(t *testing.T) {
	rows := fixtureExplorerRows()
	got := computeAvailableOutcomes(rows, domain.MatchHistoryQueryRequest{}, nil)

	// 4 Wins (m0/m1/m6/m7), 2 Loss (m2/m3), 1 Tie (m4), 1 DNF (m5).
	cases := map[string]int{"2": 4, "3": 2, "1": 1, "4": 1}
	for v, want := range cases {
		if got := findCount(got, v); got != want {
			t.Errorf("outcome %s: want %d, got %d", v, want, got)
		}
	}
}

func TestComputeAvailableOutcomes_OneSelectedAddsOR(t *testing.T) {
	rows := fixtureExplorerRows()
	// Si selected={Win=2}, count(Loss=3) doit refléter "Win OR Loss" = 6.
	req := domain.MatchHistoryQueryRequest{OutcomeFilter: []int{2}}
	got := computeAvailableOutcomes(rows, req, nil)

	if c := findCount(got, "2"); c != 4 {
		t.Errorf("Win already selected: want 4, got %d", c)
	}
	if c := findCount(got, "3"); c != 6 {
		t.Errorf("Loss added: Win OR Loss = 6, got %d", c)
	}
	if c := findCount(got, "1"); c != 5 {
		t.Errorf("Tie added: Win OR Tie = 5, got %d", c)
	}
}

// ---------------------------------------------------------------------------
// computeAvailablePerfTiers
// ---------------------------------------------------------------------------

func TestComputeAvailablePerfTiers_NoFilter(t *testing.T) {
	rows := fixtureExplorerRows()
	got := computeAvailablePerfTiers(rows, domain.MatchHistoryQueryRequest{}, nil)

	// tier 1 (≥80) : m0(90), m1(85), m6(95) = 3
	// tier 2 (65..80) : m7(70) = 1
	// tier 3 (50..65) : m3(60), m4(50) = 2
	// tier 4 (35..50) : m2(40) = 1
	// tier 5 (<35) : 0
	cases := map[string]int{"1": 3, "2": 1, "3": 2, "4": 1, "5": 0}
	for v, want := range cases {
		if c := findCount(got, v); c != want {
			t.Errorf("perf tier %s: want %d, got %d", v, want, c)
		}
	}
}

func TestComputeAvailablePerfTiers_OneSelectedAddsOR(t *testing.T) {
	rows := fixtureExplorerRows()
	req := domain.MatchHistoryQueryRequest{PerfTiers: []int{1}}
	got := computeAvailablePerfTiers(rows, req, nil)

	if c := findCount(got, "1"); c != 3 {
		t.Errorf("tier 1 already selected: want 3, got %d", c)
	}
	if c := findCount(got, "2"); c != 4 { // tier1(3) OR tier2(1) = 4
		t.Errorf("tier 2 added: 1 OR 2 = 4, got %d", c)
	}
}

// ---------------------------------------------------------------------------
// computeAvailableSkillTiers
// ---------------------------------------------------------------------------

func TestComputeAvailableSkillTiers_NoRankedContext_AllZero(t *testing.T) {
	rows := fixtureExplorerRows()
	got := computeAvailableSkillTiers(rows, domain.MatchHistoryQueryRequest{}, nil)

	// Sans ranked_context, skill_tier n'est pas applicable → tous les counts = 0.
	for _, opt := range got {
		if opt.Count != 0 {
			t.Errorf("skill tier %s: want 0 sans ranked_context, got %d", opt.Value, opt.Count)
		}
	}
}

func TestComputeAvailableSkillTiers_RankedContext_NonZero(t *testing.T) {
	rows := fixtureExplorerRows()
	req := domain.MatchHistoryQueryRequest{RankedContext: "ranked"}
	got := computeAvailableSkillTiers(rows, req, nil)

	// Avec ranked_context="ranked" : m0,m1,m2,m6 sont IsRanked=true.
	//   Diamond : m0+m1 = 2
	//   Onyx    : m2+m6 = 2
	//   Gold    : 0 (m3 est unranked, exclu par filterByRankedContext("ranked"))
	cases := map[string]int{"Diamond": 2, "Onyx": 2, "Gold": 0, "Bronze": 0}
	for v, want := range cases {
		if c := findCount(got, v); c != want {
			t.Errorf("skill tier %s: want %d, got %d", v, want, c)
		}
	}
}

// ---------------------------------------------------------------------------
// computeAvailableRankedContexts
// ---------------------------------------------------------------------------

func TestComputeAvailableRankedContexts_NoFilter(t *testing.T) {
	rows := fixtureExplorerRows()
	got := computeAvailableRankedContexts(rows, domain.MatchHistoryQueryRequest{}, nil)

	// IsRanked=true : m0,m1,m2,m6 = 4
	// IsRanked=false : m3,m4,m5,m7 = 4
	// "" (all) : 8
	cases := map[string]int{"": 8, "ranked": 4, "unranked": 4}
	for v, want := range cases {
		if c := findCount(got, v); c != want {
			t.Errorf("ranked %q: want %d, got %d", v, want, c)
		}
	}
}

func TestComputeAvailableRankedContexts_WithOutcomeFilter(t *testing.T) {
	rows := fixtureExplorerRows()
	// Filtre Win uniquement → m0,m1,m6,m7 (4 wins).
	// Parmi ces 4 : ranked=m0,m1,m6 = 3 ; unranked=m7 = 1 ; all=4.
	req := domain.MatchHistoryQueryRequest{OutcomeFilter: []int{2}}
	got := computeAvailableRankedContexts(rows, req, nil)

	cases := map[string]int{"": 4, "ranked": 3, "unranked": 1}
	for v, want := range cases {
		if c := findCount(got, v); c != want {
			t.Errorf("ranked %q with Win filter: want %d, got %d", v, want, c)
		}
	}
}

// ---------------------------------------------------------------------------
// computeAvailableSquadScopes
// ---------------------------------------------------------------------------

func TestComputeAvailableSquadScopes_NoFilter(t *testing.T) {
	rows := fixtureExplorerRows()
	got := computeAvailableSquadScopes(rows, domain.MatchHistoryQueryRequest{}, nil)

	// IsWithFriends=true : m0,m2,m3,m6,m7 = 5 (squad)
	// IsWithFriends=false : m1,m4,m5 = 3 (solo)
	// all : 8
	cases := map[string]int{"": 8, "solo": 3, "squad": 5}
	for v, want := range cases {
		if c := findCount(got, v); c != want {
			t.Errorf("squad scope %q: want %d, got %d", v, want, c)
		}
	}
}

func TestComputeAvailableSquadScopes_RespectsOtherFilters(t *testing.T) {
	rows := fixtureExplorerRows()
	// Filtre Win uniquement (4 matchs : m0,m1,m6,m7).
	//   solo  : m1 = 1
	//   squad : m0,m6,m7 = 3
	//   all   : 4
	req := domain.MatchHistoryQueryRequest{OutcomeFilter: []int{2}}
	got := computeAvailableSquadScopes(rows, req, nil)

	cases := map[string]int{"": 4, "solo": 1, "squad": 3}
	for v, want := range cases {
		if c := findCount(got, v); c != want {
			t.Errorf("squad scope %q with Win filter: want %d, got %d", v, want, c)
		}
	}
}

// ---------------------------------------------------------------------------
// Stabilité des structures retournées
// ---------------------------------------------------------------------------

func TestComputeAvailable_AllReturnExpectedCardinalities(t *testing.T) {
	rows := fixtureExplorerRows()
	req := domain.MatchHistoryQueryRequest{}

	checks := []struct {
		name string
		got  []domain.LabelValue
		want int
	}{
		{"outcomes", computeAvailableOutcomes(rows, req, nil), 4},
		{"perf_tiers", computeAvailablePerfTiers(rows, req, nil), 5},
		{"skill_tiers", computeAvailableSkillTiers(rows, req, nil), 6},
		{"ranked_contexts", computeAvailableRankedContexts(rows, req, nil), 3},
		{"squad_scopes", computeAvailableSquadScopes(rows, req, nil), 3},
	}
	for _, c := range checks {
		if len(c.got) != c.want {
			t.Errorf("%s: want %d options, got %d", c.name, c.want, len(c.got))
		}
		// Toutes les options doivent avoir un Label non vide.
		for _, opt := range c.got {
			if opt.Label == "" {
				t.Errorf("%s: option value=%q has empty label", c.name, opt.Value)
			}
		}
	}
}
