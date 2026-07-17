package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// briefingOutcomeRaw : raw row datée du briefing réduite à un outcome (les stats
// K/D/A/perf ne comptent pas pour les séries). daysAgo décroissant = plus récent.
func briefingOutcomeRaw(id string, daysAgo, outcome int) domain.MatchHistoryRawRow {
	return briefingRaw(id, daysAgo, outcome, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène classée")
}

// Série de victoires en TÊTE de scope (chronologique W W W L W) : meilleure série
// = 3 (les 3 plus anciens), l'ordre d'entrée est brouillé pour prouver le tri.
func TestBuildBriefingStreaks_HeadStreak(t *testing.T) {
	scope := []domain.MatchHistoryRawRow{
		briefingOutcomeRaw("d", 2, domain.OutcomeLoss),
		briefingOutcomeRaw("a", 5, domain.OutcomeWin),
		briefingOutcomeRaw("e", 1, domain.OutcomeWin),
		briefingOutcomeRaw("c", 3, domain.OutcomeWin),
		briefingOutcomeRaw("b", 4, domain.OutcomeWin),
	}
	s := buildBriefingStreaks(scope)
	if s == nil {
		t.Fatal("streaks attendu non nil (rows datées présentes)")
	}
	if s.BestWinStreak != 3 {
		t.Errorf("BestWinStreak = %d, attendu 3 (série en tête)", s.BestWinStreak)
	}
	if s.WorstLossStreak != 1 {
		t.Errorf("WorstLossStreak = %d, attendu 1", s.WorstLossStreak)
	}
}

// Série de victoires en QUEUE (chronologique L L W W W W) : meilleure série = 4
// (les plus récents), pire série de défaites = 2 (en tête).
func TestBuildBriefingStreaks_TailStreak(t *testing.T) {
	scope := []domain.MatchHistoryRawRow{
		briefingOutcomeRaw("f", 1, domain.OutcomeWin),
		briefingOutcomeRaw("a", 6, domain.OutcomeLoss),
		briefingOutcomeRaw("c", 4, domain.OutcomeWin),
		briefingOutcomeRaw("b", 5, domain.OutcomeLoss),
		briefingOutcomeRaw("e", 2, domain.OutcomeWin),
		briefingOutcomeRaw("d", 3, domain.OutcomeWin),
	}
	s := buildBriefingStreaks(scope)
	if s == nil {
		t.Fatal("streaks attendu non nil")
	}
	if s.BestWinStreak != 4 {
		t.Errorf("BestWinStreak = %d, attendu 4 (série en queue)", s.BestWinStreak)
	}
	if s.WorstLossStreak != 2 {
		t.Errorf("WorstLossStreak = %d, attendu 2 (série en tête)", s.WorstLossStreak)
	}
}

// Scope 100 % victoires : pire série de défaites absente (0 → omitempty).
func TestBuildBriefingStreaks_AllWinsNoLossStreak(t *testing.T) {
	var scope []domain.MatchHistoryRawRow
	for i := 0; i < 6; i++ {
		scope = append(scope, briefingOutcomeRaw("w"+string(rune('a'+i)), 10-i, domain.OutcomeWin))
	}
	s := buildBriefingStreaks(scope)
	if s == nil {
		t.Fatal("streaks attendu non nil")
	}
	if s.BestWinStreak != 6 {
		t.Errorf("BestWinStreak = %d, attendu 6", s.BestWinStreak)
	}
	if s.WorstLossStreak != 0 {
		t.Errorf("WorstLossStreak = %d, attendu 0 (aucune défaite)", s.WorstLossStreak)
	}
}

// Une série de victoires est rompue par TOUT autre outcome (nul, abandon), pas
// seulement une défaite (P-9).
func TestBuildBriefingStreaks_BrokenByAnyOutcome(t *testing.T) {
	// Chronologique : W W N W (nul en 3e position rompt la série).
	scope := []domain.MatchHistoryRawRow{
		briefingOutcomeRaw("a", 4, domain.OutcomeWin),
		briefingOutcomeRaw("b", 3, domain.OutcomeWin),
		briefingOutcomeRaw("c", 2, domain.OutcomeDraw),
		briefingOutcomeRaw("d", 1, domain.OutcomeWin),
	}
	s := buildBriefingStreaks(scope)
	if s == nil {
		t.Fatal("streaks attendu non nil")
	}
	if s.BestWinStreak != 2 {
		t.Errorf("BestWinStreak = %d, attendu 2 (rompue par le nul)", s.BestWinStreak)
	}
}

// Les rows non datées sont écartées avant le calcul des séries (P-9). Deux
// défaites SANS date ne doivent pas produire de série de défaites.
func TestBuildBriefingStreaks_UndatedRowsDiscarded(t *testing.T) {
	scope := []domain.MatchHistoryRawRow{
		briefingOutcomeRaw("a", 3, domain.OutcomeWin),
		briefingOutcomeRaw("b", 2, domain.OutcomeWin),
		briefingOutcomeRaw("c", 1, domain.OutcomeWin),
	}
	// 2 défaites non datées (StartTime nil) — doivent être ignorées.
	for _, id := range []string{"x", "y"} {
		r := briefingOutcomeRaw(id, 0, domain.OutcomeLoss)
		r.StartTime = nil
		scope = append(scope, r)
	}
	s := buildBriefingStreaks(scope)
	if s == nil {
		t.Fatal("streaks attendu non nil (3 rows datées)")
	}
	if s.BestWinStreak != 3 {
		t.Errorf("BestWinStreak = %d, attendu 3", s.BestWinStreak)
	}
	if s.WorstLossStreak != 0 {
		t.Errorf("WorstLossStreak = %d, attendu 0 (défaites non datées écartées)", s.WorstLossStreak)
	}
}

// Aucune row datée → nil (P-9).
func TestBuildBriefingStreaks_NilWhenNoDatedRows(t *testing.T) {
	var scope []domain.MatchHistoryRawRow
	for _, id := range []string{"x", "y", "z"} {
		r := briefingOutcomeRaw(id, 0, domain.OutcomeWin)
		r.StartTime = nil
		scope = append(scope, r)
	}
	if s := buildBriefingStreaks(scope); s != nil {
		t.Fatalf("streaks attendu nil (aucune row datée), got %+v", s)
	}
}

// Compteurs de dominance : chaque DominanceFlag 1..5 est compté sur son bucket.
func TestBuildBriefingDominance_Counts(t *testing.T) {
	flags := []int{
		analysis.DominanceFlagDomination,
		analysis.DominanceFlagDomination,
		analysis.DominanceFlagHumiliation,
		analysis.DominanceFlagRemontada,
		analysis.DominanceFlagRemontada,
		analysis.DominanceFlagRemontada,
		analysis.DominanceFlagDebacle,
		analysis.DominanceFlagContreRemontada,
		analysis.DominanceFlagNone, // ignoré
	}
	var scope []domain.MatchHistoryRawRow
	for i, f := range flags {
		r := briefingOutcomeRaw("m"+string(rune('a'+i)), 20-i, domain.OutcomeWin)
		r.DominanceFlag = f
		scope = append(scope, r)
	}
	d := buildBriefingDominance(scope)
	if d == nil {
		t.Fatal("dominance attendu non nil")
	}
	if d.Dominations != 2 {
		t.Errorf("Dominations = %d, attendu 2", d.Dominations)
	}
	if d.Humiliations != 1 {
		t.Errorf("Humiliations = %d, attendu 1", d.Humiliations)
	}
	if d.Remontadas != 3 {
		t.Errorf("Remontadas = %d, attendu 3", d.Remontadas)
	}
	if d.Debandades != 1 {
		t.Errorf("Debandades = %d, attendu 1", d.Debandades)
	}
	if d.ContreRemontadas != 1 {
		t.Errorf("ContreRemontadas = %d, attendu 1", d.ContreRemontadas)
	}
}

// Dominance tous-zéro → nil (P-9, dégradation par omission).
func TestBuildBriefingDominance_NilWhenAllZero(t *testing.T) {
	var scope []domain.MatchHistoryRawRow
	for i := 0; i < 5; i++ {
		// DominanceFlag laissé à 0 par défaut (briefingOutcomeRaw ne le pose pas).
		scope = append(scope, briefingOutcomeRaw("m"+string(rune('a'+i)), 10-i, domain.OutcomeWin))
	}
	if d := buildBriefingDominance(scope); d != nil {
		t.Fatalf("dominance attendu nil (tous compteurs à zéro), got %+v", d)
	}
}

// Dataset hétérogène réaliste (outcomes mélangés + dominance + une row non datée)
// via buildExplorerBriefing : séries et moments forts cohérents, non nil.
func TestBuildExplorerBriefing_StreaksAndDominanceHeterogeneous(t *testing.T) {
	// Chronologique (daysAgo décroissant) : W W W L L W N W W  + 1 abandon dominé.
	type spec struct {
		daysAgo int
		outcome int
		flag    int
	}
	specs := []spec{
		{30, domain.OutcomeWin, analysis.DominanceFlagDomination},
		{29, domain.OutcomeWin, analysis.DominanceFlagNone},
		{28, domain.OutcomeWin, analysis.DominanceFlagDomination},
		{27, domain.OutcomeLoss, analysis.DominanceFlagHumiliation},
		{26, domain.OutcomeLoss, analysis.DominanceFlagNone},
		{25, domain.OutcomeWin, analysis.DominanceFlagRemontada},
		{24, domain.OutcomeDraw, analysis.DominanceFlagNone},
		{23, domain.OutcomeWin, analysis.DominanceFlagNone},
		{22, domain.OutcomeWin, analysis.DominanceFlagNone},
		{21, domain.OutcomeLoss, analysis.DominanceFlagNone},
		{20, domain.OutcomeWin, analysis.DominanceFlagContreRemontada},
	}
	var filtered []domain.MatchHistoryRawRow
	for i, sp := range specs {
		r := briefingOutcomeRaw("m"+string(rune('a'+i)), sp.daysAgo, sp.outcome)
		r.DominanceFlag = sp.flag
		filtered = append(filtered, r)
	}
	// Une row non datée (doit être écartée des séries, comptée en dominance).
	undated := briefingOutcomeRaw("und", 0, domain.OutcomeLoss)
	undated.StartTime = nil
	filtered = append(filtered, undated)

	b := svcWithRanked(false).buildExplorerBriefing(context.Background(), filtered, filtered)
	if b == nil {
		t.Fatal("briefing nil")
	}
	if b.LowSample {
		t.Fatal("scope ≥ seuil : ne doit pas être low sample")
	}
	if b.Streaks == nil {
		t.Fatal("Streaks attendu non nil")
	}
	// Chronologique W W W L L W N W W L W : meilleure série de victoires = 3 (tête).
	if b.Streaks.BestWinStreak != 3 {
		t.Errorf("BestWinStreak = %d, attendu 3", b.Streaks.BestWinStreak)
	}
	// Pire série de défaites = 2 (L L au milieu). La défaite non datée est écartée.
	if b.Streaks.WorstLossStreak != 2 {
		t.Errorf("WorstLossStreak = %d, attendu 2", b.Streaks.WorstLossStreak)
	}
	if b.Dominance == nil {
		t.Fatal("Dominance attendu non nil")
	}
	if b.Dominance.Dominations != 2 || b.Dominance.Humiliations != 1 ||
		b.Dominance.Remontadas != 1 || b.Dominance.ContreRemontadas != 1 {
		t.Errorf("dominance = %+v, compteurs inattendus", b.Dominance)
	}
}

// Sous low_sample, les modules Streaks/Dominance sont omis (retour anticipé).
func TestBuildExplorerBriefing_StreaksOmittedWhenLowSample(t *testing.T) {
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 8; i++ {
		r := briefingOutcomeRaw("m"+string(rune('a'+i)), 10-i, domain.OutcomeWin)
		r.DominanceFlag = analysis.DominanceFlagDomination
		filtered = append(filtered, r)
	}
	b := svcWithRanked(false).buildExplorerBriefing(context.Background(), filtered, filtered)
	if b == nil {
		t.Fatal("briefing nil")
	}
	if !b.LowSample {
		t.Fatal("attendu LowSample (scope < seuil modules)")
	}
	if b.Streaks != nil || b.Dominance != nil {
		t.Errorf("Streaks/Dominance doivent être nil sous low_sample, got streaks=%+v dominance=%+v", b.Streaks, b.Dominance)
	}
}
