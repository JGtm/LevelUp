package analysis

import (
	"math"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

func float64PtrKPI(v float64) *float64 { return &v }

func intPtrKPI(v int) *int { return &v }

func mkRow(kills, deaths, assists, timePlayed int, outcome canonical.Outcome, accuracy, avgLife *float64) canonical.PlayerMatchRow {
	return canonical.PlayerMatchRow{
		Self: canonical.MatchParticipant{
			Outcome:        outcome,
			Kills:          intPtrKPI(kills),
			Deaths:         intPtrKPI(deaths),
			Assists:        intPtrKPI(assists),
			TimePlayed:     intPtrKPI(timePlayed),
			Accuracy:       accuracy,
			AvgLifeSeconds: avgLife,
		},
	}
}

// mkRowWithSkill construit une row avec un skill snapshot pour tester
// RankDelta. Passer ratingType="" pour omettre le snapshot.
func mkRowWithSkill(ratingType canonical.RatingType, delta *float64) canonical.PlayerMatchRow {
	row := canonical.PlayerMatchRow{
		Self: canonical.MatchParticipant{Outcome: canonical.OutcomeWin},
	}
	if ratingType != "" {
		row.Enrichment.SkillSnapshot = &canonical.SkillSnapshot{
			RatingType: ratingType,
			Delta:      delta,
		}
	}
	return row
}

func TestComputeKPIStats_Empty(t *testing.T) {
	t.Parallel()
	got := ComputeKPIStats(nil, 225)
	if got.MatchesCount != 0 {
		t.Errorf("empty: want 0 matches, got %d", got.MatchesCount)
	}
}

func TestComputeKPIStats_BasicAggregation(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		mkRow(10, 5, 2, 600, canonical.OutcomeWin, float64PtrKPI(45.0), float64PtrKPI(30.0)),
		mkRow(8, 7, 1, 400, canonical.OutcomeLoss, float64PtrKPI(50.0), float64PtrKPI(20.0)),
		mkRow(4, 3, 5, 500, canonical.OutcomeTie, nil, nil),
	}
	got := ComputeKPIStats(rows, 225)

	if got.MatchesCount != 3 {
		t.Errorf("MatchesCount: want 3, got %d", got.MatchesCount)
	}
	if got.TotalPlaySeconds != 1500 {
		t.Errorf("TotalPlaySeconds: want 1500, got %d", got.TotalPlaySeconds)
	}
	if math.Abs(got.AvgMatchSeconds-500) > 1e-9 {
		t.Errorf("AvgMatchSeconds: want 500, got %v", got.AvgMatchSeconds)
	}

	// Kills total = 22, deaths = 15, assists = 8, sur 3 matchs
	if math.Abs(got.KillsPerGame-22.0/3.0) > 1e-9 {
		t.Errorf("KillsPerGame: want 22/3, got %v", got.KillsPerGame)
	}
	if math.Abs(got.DeathsPerGame-5.0) > 1e-9 {
		t.Errorf("DeathsPerGame: want 5, got %v", got.DeathsPerGame)
	}

	// PerMinute : 1500s = 25 minutes
	if math.Abs(got.KillsPerMinute-22.0/25.0) > 1e-9 {
		t.Errorf("KillsPerMinute: want 22/25, got %v", got.KillsPerMinute)
	}

	// Accuracy : moyenne sur les 2 samples (45 + 50) / 2 = 47.5
	if math.Abs(got.AvgAccuracy-47.5) > 1e-9 {
		t.Errorf("AvgAccuracy: want 47.5, got %v", got.AvgAccuracy)
	}
	// AvgLife : (30 + 20) / 2 = 25
	if math.Abs(got.AvgLifeSeconds-25.0) > 1e-9 {
		t.Errorf("AvgLifeSeconds: want 25, got %v", got.AvgLifeSeconds)
	}

	if got.Outcomes.Wins != 1 || got.Outcomes.Losses != 1 || got.Outcomes.Ties != 1 || got.Outcomes.DNF != 0 {
		t.Errorf("Outcomes: want W1/L1/T1/DNF0, got %+v", got.Outcomes)
	}
}

func TestComputeKPIStats_AllNilFieldsTolere(t *testing.T) {
	t.Parallel()
	// Row sans aucun pointer renseigne -> les agregats sont 0 mais MatchesCount=1.
	row := canonical.PlayerMatchRow{
		Self: canonical.MatchParticipant{Outcome: canonical.OutcomeWin},
	}
	got := ComputeKPIStats([]canonical.PlayerMatchRow{row}, 225)
	if got.MatchesCount != 1 {
		t.Errorf("MatchesCount: want 1, got %d", got.MatchesCount)
	}
	if got.KillsPerGame != 0 || got.AvgAccuracy != 0 {
		t.Errorf("nil fields should not contribute, got KPG=%v Acc=%v",
			got.KillsPerGame, got.AvgAccuracy)
	}
	if got.Outcomes.Wins != 1 {
		t.Errorf("Win should still be counted, got %+v", got.Outcomes)
	}
}

func TestComputeKPIStats_OutcomeBreakdown(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		mkRow(10, 5, 0, 600, canonical.OutcomeWin, nil, nil),
		mkRow(10, 5, 0, 600, canonical.OutcomeWin, nil, nil),
		mkRow(5, 10, 0, 600, canonical.OutcomeLoss, nil, nil),
		mkRow(5, 5, 0, 600, canonical.OutcomeTie, nil, nil),
		mkRow(0, 1, 0, 60, canonical.OutcomeDNF, nil, nil),
		mkRow(0, 0, 0, 0, canonical.Outcome(""), nil, nil), // outcome vide ignore
	}
	got := ComputeKPIStats(rows, 225)
	if got.Outcomes.Wins != 2 || got.Outcomes.Losses != 1 || got.Outcomes.Ties != 1 || got.Outcomes.DNF != 1 {
		t.Errorf("outcomes: want W2/L1/T1/DNF1, got %+v", got.Outcomes)
	}
}

// =============================================================================
// RankDelta
// =============================================================================

func TestComputeKPIStats_RankDelta_CSR_SumsSignedDeltas(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		mkRowWithSkill(canonical.RatingTypeCSR, float64PtrKPI(15.0)),
		mkRowWithSkill(canonical.RatingTypeCSR, float64PtrKPI(-8.0)),
		mkRowWithSkill(canonical.RatingTypeCSR, float64PtrKPI(20.0)),
	}
	got := ComputeKPIStats(rows, 225)
	if got.RankDelta == nil {
		t.Fatal("RankDelta: got nil, want CSR delta")
	}
	if got.RankDelta.Kind != "csr" {
		t.Errorf("Kind: want csr, got %q", got.RankDelta.Kind)
	}
	if math.Abs(got.RankDelta.Value-27.0) > 1e-9 {
		t.Errorf("Value: want 27 (15-8+20), got %v", got.RankDelta.Value)
	}
	if got.RankDelta.Count != 3 {
		t.Errorf("Count: want 3, got %d", got.RankDelta.Count)
	}
}

func TestComputeKPIStats_RankDelta_LUSR(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		mkRowWithSkill(canonical.RatingTypeLUSR, float64PtrKPI(0.05)),
		mkRowWithSkill(canonical.RatingTypeLUSR, float64PtrKPI(-0.02)),
	}
	got := ComputeKPIStats(rows, 225)
	if got.RankDelta == nil {
		t.Fatal("RankDelta: got nil, want LUSR delta")
	}
	if got.RankDelta.Kind != "lusr" {
		t.Errorf("Kind: want lusr, got %q", got.RankDelta.Kind)
	}
	if math.Abs(got.RankDelta.Value-0.03) > 1e-9 {
		t.Errorf("Value: want 0.03, got %v", got.RankDelta.Value)
	}
	if got.RankDelta.Count != 2 {
		t.Errorf("Count: want 2, got %d", got.RankDelta.Count)
	}
}

// TestComputeKPIStats_RankDelta_NilWhenNoRatedMatches : aucun snapshot ou aucun
// delta renseigne -> RankDelta nil (la card "Delta rang" sera cachee cote front).
func TestComputeKPIStats_RankDelta_NilWhenNoRatedMatches(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		mkRowWithSkill("", nil),                      // aucun snapshot
		mkRowWithSkill(canonical.RatingTypeCSR, nil), // snapshot sans delta
	}
	got := ComputeKPIStats(rows, 225)
	if got.RankDelta != nil {
		t.Errorf("RankDelta: want nil, got %+v", got.RankDelta)
	}
}

// TestComputeKPIStats_RankDelta_MixedScopeKeepsDominantKind : cas pathologique
// (scope qui mixe matchs classes et non classes). On retient le type avec le
// plus de matchs (CSR ici, 2 vs 1).
func TestComputeKPIStats_RankDelta_MixedScopeKeepsDominantKind(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		mkRowWithSkill(canonical.RatingTypeCSR, float64PtrKPI(10.0)),
		mkRowWithSkill(canonical.RatingTypeCSR, float64PtrKPI(-5.0)),
		mkRowWithSkill(canonical.RatingTypeLUSR, float64PtrKPI(0.1)),
	}
	got := ComputeKPIStats(rows, 225)
	if got.RankDelta == nil || got.RankDelta.Kind != "csr" {
		t.Fatalf("want csr (majority), got %+v", got.RankDelta)
	}
	if got.RankDelta.Count != 2 {
		t.Errorf("Count: want 2 (csr only), got %d", got.RankDelta.Count)
	}
	if math.Abs(got.RankDelta.Value-5.0) > 1e-9 {
		t.Errorf("Value: want 5 (10-5, lusr ignore), got %v", got.RankDelta.Value)
	}
}

// TestComputeKPIStats_RankDelta_TieBrokenByCSR : 1 match CSR + 1 match LUSR
// -> egalite, CSR l'emporte (priorite competitive).
func TestComputeKPIStats_RankDelta_TieBrokenByCSR(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		mkRowWithSkill(canonical.RatingTypeLUSR, float64PtrKPI(0.5)),
		mkRowWithSkill(canonical.RatingTypeCSR, float64PtrKPI(7.0)),
	}
	got := ComputeKPIStats(rows, 225)
	if got.RankDelta == nil || got.RankDelta.Kind != "csr" {
		t.Fatalf("tie broken by CSR: want csr, got %+v", got.RankDelta)
	}
	if got.RankDelta.Count != 1 || math.Abs(got.RankDelta.Value-7.0) > 1e-9 {
		t.Errorf("CSR bucket only: want Count=1 Value=7, got %+v", got.RankDelta)
	}
}

func TestComputeKPIStats_ZeroPlaySecondsNoPanicOnPerMin(t *testing.T) {
	t.Parallel()
	// Tous les TimePlayed = 0 -> *PerMinute restent 0 (pas de division par zero).
	rows := []canonical.PlayerMatchRow{
		mkRow(10, 5, 0, 0, canonical.OutcomeWin, nil, nil),
	}
	got := ComputeKPIStats(rows, 225)
	if got.KillsPerMinute != 0 || got.DeathsPerMinute != 0 || got.AssistsPerMinute != 0 {
		t.Errorf("zero play seconds: per-min should be 0, got %+v",
			[]float64{got.KillsPerMinute, got.DeathsPerMinute, got.AssistsPerMinute})
	}
}

// =============================================================================
// ComputeKPIs (legacy HomeMatchRow) — découplage offensif / défensif (P6)
// =============================================================================

func float64PtrHM(v float64) *float64 { return &v }

// TestComputeKPIs_OffensiveConversionDecoupledFromDamageTaken : régression H5.
// Avec damage_dealt présent mais damage_taken == nil (Halo 5 n'a pas de
// damage_taken), l'AvgOffensiveConversion doit être calculé (> 0) tandis que
// l'AvgDefensiveResistance reste nil (pas de DR fabriquée).
func TestComputeTeamAvgKPIs_Empty(t *testing.T) {
	t.Parallel()
	if got := ComputeTeamAvgKPIs(nil); got != nil {
		t.Errorf("nil map: got %+v, want nil", got)
	}
	if got := ComputeTeamAvgKPIs(map[string]*domain.KPIStats{}); got != nil {
		t.Errorf("empty map: got %+v, want nil", got)
	}
}

func TestComputeTeamAvgKPIs_OnlyNilEntries(t *testing.T) {
	t.Parallel()
	in := map[string]*domain.KPIStats{
		"a": nil,
		"b": nil,
	}
	if got := ComputeTeamAvgKPIs(in); got != nil {
		t.Errorf("only nil entries: got %+v, want nil", got)
	}
}

func TestComputeTeamAvgKPIs_SingleEntry_MirrorsValues(t *testing.T) {
	t.Parallel()
	src := &domain.KPIStats{
		MatchesCount:     10,
		TotalPlaySeconds: 6540,
		AvgMatchSeconds:  654,
		KillsPerGame:     8.7,
		KillsPerMinute:   1.0,
		DeathsPerGame:    10.8,
		DeathsPerMinute:  1.24,
		AssistsPerGame:   4.5,
		AssistsPerMinute: 0.52,
		AvgAccuracy:      46.92,
		AvgLifeSeconds:   37,
	}
	src.Outcomes.Wins = 3
	src.Outcomes.Losses = 7

	got := ComputeTeamAvgKPIs(map[string]*domain.KPIStats{"me": src})
	if got == nil {
		t.Fatal("single entry: got nil")
	}
	if got.KillsPerGame != 8.7 {
		t.Errorf("KillsPerGame: got %v, want 8.7", got.KillsPerGame)
	}
	if got.AvgAccuracy != 46.92 {
		t.Errorf("AvgAccuracy: got %v, want 46.92", got.AvgAccuracy)
	}
	// Outcomes mis a zero (sans signification en moyenne).
	if got.Outcomes.Wins != 0 || got.Outcomes.Losses != 0 {
		t.Errorf("Outcomes should be zero in avg: got %+v", got.Outcomes)
	}
}

func TestComputeTeamAvgKPIs_ThreeEntries_FieldByFieldMean(t *testing.T) {
	t.Parallel()
	in := map[string]*domain.KPIStats{
		"a": {KillsPerGame: 6.0, DeathsPerGame: 12.0, AvgAccuracy: 40.0, AvgLifeSeconds: 30, KillsPerMinute: 0.6, MatchesCount: 10, TotalPlaySeconds: 6000, AvgMatchSeconds: 600},
		"b": {KillsPerGame: 9.0, DeathsPerGame: 10.0, AvgAccuracy: 50.0, AvgLifeSeconds: 40, KillsPerMinute: 0.9, MatchesCount: 10, TotalPlaySeconds: 6000, AvgMatchSeconds: 600},
		"c": {KillsPerGame: 12.0, DeathsPerGame: 8.0, AvgAccuracy: 60.0, AvgLifeSeconds: 50, KillsPerMinute: 1.2, MatchesCount: 10, TotalPlaySeconds: 6000, AvgMatchSeconds: 600},
	}
	got := ComputeTeamAvgKPIs(in)
	if got == nil {
		t.Fatal("3 entries: got nil")
	}
	approxEqualKPI := func(t *testing.T, name string, gotV, wantV float64) {
		t.Helper()
		if math.Abs(gotV-wantV) > 1e-9 {
			t.Errorf("%s: got %v, want %v", name, gotV, wantV)
		}
	}
	approxEqualKPI(t, "KillsPerGame", got.KillsPerGame, 9.0)
	approxEqualKPI(t, "DeathsPerGame", got.DeathsPerGame, 10.0)
	approxEqualKPI(t, "AvgAccuracy", got.AvgAccuracy, 50.0)
	approxEqualKPI(t, "AvgLifeSeconds", got.AvgLifeSeconds, 40.0)
	approxEqualKPI(t, "KillsPerMinute", got.KillsPerMinute, 0.9)
}

func TestComputeTeamAvgKPIs_IgnoresNilEntry(t *testing.T) {
	t.Parallel()
	in := map[string]*domain.KPIStats{
		"a":   {KillsPerGame: 6.0},
		"nil": nil,
		"b":   {KillsPerGame: 12.0},
	}
	got := ComputeTeamAvgKPIs(in)
	if got == nil {
		t.Fatal("got nil, want avg of 2 valid entries")
	}
	if math.Abs(got.KillsPerGame-9.0) > 1e-9 {
		t.Errorf("KillsPerGame ignoring nil: got %v, want 9.0 (mean of 6 and 12)", got.KillsPerGame)
	}
}

// =============================================================================
// CombatProfile dans ComputeKPIStats
// =============================================================================

func intPtrDmg(v int) *int { return &v }

func mkRowWithDamage(kills, assists, deaths int, dmgDealt, dmgTaken int) canonical.PlayerMatchRow {
	return canonical.PlayerMatchRow{
		Self: canonical.MatchParticipant{
			Kills:       intPtrDmg(kills),
			Assists:     intPtrDmg(assists),
			Deaths:      intPtrDmg(deaths),
			DamageDealt: intPtrDmg(dmgDealt),
			DamageTaken: intPtrDmg(dmgTaken),
		},
	}
}

func TestComputeKPIStats_CombatProfile_SetWhenDamagePresent(t *testing.T) {
	t.Parallel()
	// 20 matchs avec données dégâts valides → CombatProfile non nil.
	// 10 kills, 0 assists, 5 deaths, 2000 dmg dealt, 1800 dmg taken par match.
	rows := make([]canonical.PlayerMatchRow, 20)
	for i := range rows {
		rows[i] = mkRowWithDamage(10, 0, 5, 2000, 1800)
	}
	got := ComputeKPIStats(rows, 225)
	if got.CombatProfile == nil {
		t.Fatal("CombatProfile: want non-nil with 20 rows of damage data")
	}
	if got.AvgOffensiveConversion == nil || *got.AvgOffensiveConversion <= 0 {
		t.Errorf("AvgOffensiveConversion: want > 0, got %v", got.AvgOffensiveConversion)
	}
	if got.AvgDefensiveResistance == nil || *got.AvgDefensiveResistance <= 0 {
		t.Errorf("AvgDefensiveResistance: want > 0, got %v", got.AvgDefensiveResistance)
	}
	// Avec ≥15 matchs → styles calculés.
	if got.CombatProfile.StyleOffensive == nil {
		t.Error("CombatProfile.StyleOffensive: want non-nil with 20 matchs")
	}
	if got.CombatProfile.StyleDefensive == nil {
		t.Error("CombatProfile.StyleDefensive: want non-nil with 20 matchs")
	}
}

// TestComputeKPIStats_OC_IsAggregate_NotMeanOfRatios : régression du bug "Rendement
// identique". L'OC doit être l'agrégat volume-pondéré, PAS la moyenne des ratios par
// match (qui sur-pondère un petit match très efficace) — et = 225 / DmgPerKill.
func TestComputeKPIStats_OC_IsAggregate_NotMeanOfRatios(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		mkRowWithDamage(30, 0, 5, 4000, 2000), // gros volume
		mkRowWithDamage(1, 0, 1, 50, 1000),    // petit pic d'efficacité (OC/match = 4.5)
	}
	got := ComputeKPIStats(rows, 225)
	// Agrégat : 225 × Σ(frags+assists/3) / Σdégâts = 225 × 31 / 4050 ≈ 1.72.
	// Moyenne des ratios donnerait (1.6875 + 4.5)/2 ≈ 3.09 → bug.
	if got.AvgOffensiveConversion == nil {
		t.Fatal("AvgOffensiveConversion nil")
	}
	if math.Abs(*got.AvgOffensiveConversion-1.72) > 0.01 {
		t.Errorf("AvgOC = %v, want ≈1.72 (agrégat). Si ≈3.09 → régression mean-of-ratios", *got.AvgOffensiveConversion)
	}
	// Invariant : dégâts par frag-équivalent = Σdégâts/(Σfrags+Σassists/3), et
	// AvgOC = 225 / DmgPerKill.
	if got.CombatProfile == nil || got.CombatProfile.DmgPerKill == nil {
		t.Fatal("CombatProfile.DmgPerKill nil")
	}
	wantDPK := 4050.0 / 31.0
	if math.Abs(*got.CombatProfile.DmgPerKill-wantDPK) > 1e-6 {
		t.Errorf("DmgPerKill = %v, want %v (frag-équivalent)", *got.CombatProfile.DmgPerKill, wantDPK)
	}
	if math.Abs(*got.AvgOffensiveConversion-225.0/(*got.CombatProfile.DmgPerKill)) > 0.01 {
		t.Errorf("invariant rompu : AvgOC (%v) != 225/DmgPerKill (%v)",
			*got.AvgOffensiveConversion, 225.0/(*got.CombatProfile.DmgPerKill))
	}
}

func TestComputeKPIStats_CombatProfile_NilWhenNoDamageData(t *testing.T) {
	t.Parallel()
	// Rows sans DamageDealt/DamageTaken → CombatProfile nil.
	rows := []canonical.PlayerMatchRow{
		mkRow(10, 5, 2, 600, canonical.OutcomeWin, nil, nil),
		mkRow(8, 7, 1, 400, canonical.OutcomeLoss, nil, nil),
	}
	got := ComputeKPIStats(rows, 225)
	if got.CombatProfile != nil {
		t.Errorf("CombatProfile: want nil without damage data, got %+v", got.CombatProfile)
	}
}

func TestComputeKPIStats_CombatProfile_NilStylesBelow15Matches(t *testing.T) {
	t.Parallel()
	// 14 matchs avec dommages → CombatProfile non-nil mais styles nil (< seuil).
	rows := make([]canonical.PlayerMatchRow, 14)
	for i := range rows {
		rows[i] = mkRowWithDamage(10, 0, 5, 2000, 1800)
	}
	got := ComputeKPIStats(rows, 225)
	if got.CombatProfile == nil {
		t.Fatal("CombatProfile: want non-nil with damage data (even < 15 matchs)")
	}
	if got.CombatProfile.StyleOffensive != nil || got.CombatProfile.StyleDefensive != nil {
		t.Errorf("styles should be nil with < 15 matchs: off=%v def=%v",
			got.CombatProfile.StyleOffensive, got.CombatProfile.StyleDefensive)
	}
}

func TestComputeKPIStats_CombatProfile_PaceRatio_WiresStyleActivity(t *testing.T) {
	t.Parallel()
	// pace_ratio ∈ [1.08, 1.25[ → StyleActivity = "actif" (≥ 15 matchs + damage).
	ratio := 1.10
	rows := make([]canonical.PlayerMatchRow, 20)
	for i := range rows {
		r := mkRowWithDamage(10, 0, 5, 2000, 1800)
		r.Enrichment.EngagementPaceRatio = &ratio
		rows[i] = r
	}
	got := ComputeKPIStats(rows, 225)
	if got.CombatProfile == nil {
		t.Fatal("CombatProfile: want non-nil")
	}
	if got.CombatProfile.AvgPaceRatio == nil {
		t.Fatal("AvgPaceRatio: want non-nil when EngagementPaceRatio set on rows")
	}
	if *got.CombatProfile.AvgPaceRatio != ratio {
		t.Errorf("AvgPaceRatio = %.2f, want %.2f", *got.CombatProfile.AvgPaceRatio, ratio)
	}
	if got.CombatProfile.StyleActivity == nil || *got.CombatProfile.StyleActivity != domain.CombatStyleActivityActif {
		t.Errorf("StyleActivity: want %q (ratio=1.10), got %v", domain.CombatStyleActivityActif, got.CombatProfile.StyleActivity)
	}
}

func TestComputeKPIStats_CombatProfile_NilPaceRatio_NilStyleActivity(t *testing.T) {
	t.Parallel()
	// Pas de EngagementPaceRatio → AvgPaceRatio nil → StyleActivity nil.
	rows := make([]canonical.PlayerMatchRow, 20)
	for i := range rows {
		rows[i] = mkRowWithDamage(10, 0, 5, 2000, 1800)
	}
	got := ComputeKPIStats(rows, 225)
	if got.CombatProfile == nil {
		t.Fatal("CombatProfile: want non-nil with damage")
	}
	if got.CombatProfile.AvgPaceRatio != nil {
		t.Errorf("AvgPaceRatio: want nil when no EngagementPaceRatio, got %v", *got.CombatProfile.AvgPaceRatio)
	}
	if got.CombatProfile.StyleActivity != nil {
		t.Errorf("StyleActivity: want nil when AvgPaceRatio nil, got %v", *got.CombatProfile.StyleActivity)
	}
}
