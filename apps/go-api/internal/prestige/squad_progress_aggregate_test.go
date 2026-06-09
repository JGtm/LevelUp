package prestige

import "testing"

// progressByXUID indexe un résultat d'agrégation par xuid (pour les assertions).
func progressByXUID(ps []SquadParticipantProgress) map[string]SquadParticipantProgress {
	m := make(map[string]SquadParticipantProgress, len(ps))
	for _, p := range ps {
		m[p.Xuid] = p
	}
	return m
}

func TestAggregateSquadProgress_CumulativeOverCountingMatches(t *testing.T) {
	roster := []string{xA, xB, xC}
	matches := []SquadMatchMetric{
		// compté : trio complet
		{MatchID: "m1", Xuids: []string{xA, xB, xC}, Values: map[string]float64{xA: 10, xB: 5, xC: 2}},
		// compté : trio + random (random ignoré)
		{MatchID: "m2", Xuids: []string{xA, xB, xC, xR}, Values: map[string]float64{xA: 4, xB: 1, xC: 3}},
		// NON compté : C absent
		{MatchID: "m3", Xuids: []string{xA, xB, xR}, Values: map[string]float64{xA: 99, xB: 99}},
	}
	got := progressByXUID(AggregateSquadProgress(roster, nil, matches, 0))

	if got[xA].Value != 14 || got[xA].Matches != 2 {
		t.Errorf("A: value=%v matches=%d, want 14/2", got[xA].Value, got[xA].Matches)
	}
	if got[xB].Value != 6 || got[xB].Matches != 2 {
		t.Errorf("B: value=%v matches=%d, want 6/2", got[xB].Value, got[xB].Matches)
	}
	if got[xC].Value != 5 || got[xC].Matches != 2 {
		t.Errorf("C: value=%v matches=%d, want 5/2", got[xC].Value, got[xC].Matches)
	}
}

func TestAggregateSquadProgress_NoOverlapExcludesTrioForDuo(t *testing.T) {
	// Duo {A,B}, C est un autre coéquipier connu. Le match trio est disqualifié.
	roster := []string{xA, xB}
	other := toXUIDSet([]string{xC})
	matches := []SquadMatchMetric{
		{MatchID: "m1", Xuids: []string{xA, xB, xC}, Values: map[string]float64{xA: 50, xB: 50}}, // disqualifié
		{MatchID: "m2", Xuids: []string{xA, xB, xR}, Values: map[string]float64{xA: 3, xB: 4}},   // compté
	}
	got := progressByXUID(AggregateSquadProgress(roster, other, matches, 0))
	if got[xA].Value != 3 || got[xB].Value != 4 {
		t.Errorf("duo cumul = A:%v B:%v, want 3/4 (match trio exclu)", got[xA].Value, got[xB].Value)
	}
}

func TestAggregateSquadProgress_Completion(t *testing.T) {
	roster := []string{xA, xB}
	matches := []SquadMatchMetric{
		{MatchID: "m1", Xuids: []string{xA, xB}, Values: map[string]float64{xA: 6, xB: 2}},
		{MatchID: "m2", Xuids: []string{xA, xB}, Values: map[string]float64{xA: 5, xB: 1}},
	}
	got := progressByXUID(AggregateSquadProgress(roster, nil, matches, 10))
	// A : 11 >= 10 → complété ; B : 3 < 10 → non
	if !got[xA].Completed {
		t.Errorf("A devrait être complété (11 >= 10)")
	}
	if got[xB].Completed {
		t.Errorf("B ne devrait pas être complété (3 < 10)")
	}
}

func TestAggregateSquadProgress_PartialContribution(t *testing.T) {
	// B ne contribue pas au m2 (absent de Values) → son cumul/compteur s'arrête.
	roster := []string{xA, xB}
	matches := []SquadMatchMetric{
		{MatchID: "m1", Xuids: []string{xA, xB}, Values: map[string]float64{xA: 2, xB: 7}},
		{MatchID: "m2", Xuids: []string{xA, xB}, Values: map[string]float64{xA: 3}},
	}
	got := progressByXUID(AggregateSquadProgress(roster, nil, matches, 0))
	if got[xA].Matches != 2 || got[xA].Value != 5 {
		t.Errorf("A: %v/%d, want 5/2", got[xA].Value, got[xA].Matches)
	}
	if got[xB].Matches != 1 || got[xB].Value != 7 {
		t.Errorf("B: %v/%d, want 7/1", got[xB].Value, got[xB].Matches)
	}
}

func TestAggregateSquadProgress_EmptyRoster(t *testing.T) {
	got := AggregateSquadProgress(nil, nil, []SquadMatchMetric{
		{MatchID: "m1", Xuids: []string{xA}, Values: map[string]float64{xA: 1}},
	}, 0)
	if len(got) != 0 {
		t.Errorf("roster vide → aucune progression, got %d", len(got))
	}
}
