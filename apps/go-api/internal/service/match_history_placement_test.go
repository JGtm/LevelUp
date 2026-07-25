package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/sync"
)

func TestParsePlacementRemaining(t *testing.T) {
	cases := []struct {
		label string
		want  int
	}{
		{"Placement (4 restants)", 4},
		{"Placement (10 restants)", 10},
		{"Placement (0 restant)", 0},
		{"Placement", 10},  // fallback : pas de "(N restant)"
		{"Diamant IV", 10}, // fallback : pas un label de placement
		{"", 10},
		{"Placement (999 restants)", 10}, // out of bounds → fallback
	}
	for _, c := range cases {
		if got := parsePlacementRemaining(c.label); got != c.want {
			t.Errorf("parsePlacementRemaining(%q) = %d, want %d", c.label, got, c.want)
		}
	}
}

func TestApplyCSRPlacements_threshold5(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{
		{MatchID: "m1", SkillRatingType: strPtr("CSR"), SkillTierLabel: strPtr("Placement (4 restants)"), SeasonID: strPtr("CsrSeason13-1")},
		{MatchID: "m2", SkillRatingType: strPtr("CSR"), SkillTierLabel: strPtr("Diamant IV"), SeasonID: strPtr("CsrSeason13-1")}, // tier officiel → pas placement
		{MatchID: "m3", SkillRatingType: strPtr("LUSR"), SkillTierLabel: strPtr("Placement (3 restants)")},                       // LUSR → pas concerné par applyCSRPlacements
		{MatchID: "m4"}, // pas de rating → ignore
	}
	csrResolver := func(_ context.Context, _ string) int { return 5 }
	if got := applyCSRPlacements(context.Background(), rows, csrResolver); got != 1 {
		t.Errorf("count want 1, got %d", got)
	}

	if rows[0].PlacementDone == nil || *rows[0].PlacementDone != 1 {
		t.Errorf("m1: PlacementDone want 1 (= 5-4), got %v", rows[0].PlacementDone)
	}
	if rows[0].PlacementTotal == nil || *rows[0].PlacementTotal != 5 {
		t.Errorf("m1: PlacementTotal want 5, got %v", rows[0].PlacementTotal)
	}
	if rows[1].PlacementDone != nil {
		t.Errorf("m2: tier officiel ne doit pas être en placement, got %v", rows[1].PlacementDone)
	}
	if rows[2].PlacementDone != nil {
		t.Errorf("m3: LUSR ne doit pas être affecté par applyCSRPlacements, got %v", rows[2].PlacementDone)
	}
}

func TestApplyCSRPlacements_threshold10_legacySeason(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{
		{MatchID: "m1", SkillRatingType: strPtr("CSR"), SkillTierLabel: strPtr("Placement (7 restants)"), SeasonID: strPtr("CsrSeason1")},
	}
	csrResolver := func(_ context.Context, seasonID string) int {
		if seasonID == "CsrSeason1" {
			return 10
		}
		return 5
	}
	applyCSRPlacements(context.Background(), rows, csrResolver)
	if rows[0].PlacementDone == nil || *rows[0].PlacementDone != 3 {
		t.Errorf("PlacementDone want 3 (= 10-7), got %v", rows[0].PlacementDone)
	}
	if rows[0].PlacementTotal == nil || *rows[0].PlacementTotal != 10 {
		t.Errorf("PlacementTotal want 10, got %v", rows[0].PlacementTotal)
	}
}

func TestApplyLUSRPlacements_first10PerChain(t *testing.T) {
	// 12 matchs Slayer Arena (chaîne "arena_slayer") sans LUSR, ordre temporel
	// inverse pour valider le tri ASC dans la fonction.
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := []domain.MatchHistoryRawRow{}
	for i := 12; i >= 1; i-- {
		ts := base.Add(time.Duration(i) * time.Hour)
		rows = append(rows, domain.MatchHistoryRawRow{
			MatchID:   "slayer-" + string(rune('a'+i-1)),
			StartTime: &ts,
			PairName:  strPtr("Arena:Slayer on Bazaar"),
		})
	}

	if got := applyLUSRPlacements(rows); got != 10 {
		t.Errorf("applyLUSRPlacements returned %d, want 10", got)
	}

	// Les 10 matchs les plus anciens doivent être marqués 1/10 → 10/10.
	// Tri attendu : matchs i=1..10 (les plus anciens car +i heures), donc on
	// trie d'abord les rows par start_time ASC pour vérifier.
	placementCount := 0
	for _, r := range rows {
		if r.PlacementDone != nil {
			placementCount++
			if r.PlacementTotal == nil || *r.PlacementTotal != 10 {
				t.Errorf("%s: PlacementTotal want 10, got %v", r.MatchID, r.PlacementTotal)
			}
		}
	}
	if placementCount != 10 {
		t.Errorf("want 10 matchs en placement, got %d", placementCount)
	}
}

func TestApplyMatchPlacements_csrAndLusrCombined(t *testing.T) {
	ts := time.Now()
	rows := []domain.MatchHistoryRawRow{
		// CSR placement → done = 5-2 = 3, total = 5
		{MatchID: "csr1", StartTime: &ts, PairName: strPtr("Ranked:Slayer"),
			SkillRatingType: strPtr("CSR"), SkillTierLabel: strPtr("Placement (2 restants)"),
			SeasonID: strPtr("CsrSeason13-1")},
		// LUSR placement (chaîne arena_slayer, premier match) → 1/10
		{MatchID: "lusr1", StartTime: &ts, PairName: strPtr("Arena:Slayer on Bazaar")},
		// Tier officiel CSR → ignoré
		{MatchID: "csr_full", StartTime: &ts, PairName: strPtr("Ranked:Slayer"),
			SkillRatingType: strPtr("CSR"), SkillTierLabel: strPtr("Diamant IV")},
	}
	csrResolver := func(_ context.Context, _ string) int { return 5 }
	applyMatchPlacements(context.Background(), rows, csrResolver)

	if rows[0].PlacementDone == nil || *rows[0].PlacementDone != 3 || *rows[0].PlacementTotal != 5 {
		t.Errorf("csr1 want 3/5, got done=%v total=%v", rows[0].PlacementDone, rows[0].PlacementTotal)
	}
	if rows[1].PlacementDone == nil || *rows[1].PlacementDone != 1 || *rows[1].PlacementTotal != 10 {
		t.Errorf("lusr1 want 1/10, got done=%v total=%v", rows[1].PlacementDone, rows[1].PlacementTotal)
	}
	if rows[2].PlacementDone != nil {
		t.Errorf("csr_full (tier officiel) ne doit pas être en placement, got %v", rows[2].PlacementDone)
	}
}

func TestApplyMatchPlacements_nilCsrResolver(t *testing.T) {
	rows := []domain.MatchHistoryRawRow{
		{MatchID: "m1", SkillRatingType: strPtr("CSR"), SkillTierLabel: strPtr("Placement (2 restants)")},
	}
	// csrThreshold = nil → fallback à defaultCSRPlacementThreshold (5)
	applyMatchPlacements(context.Background(), rows, nil)
	if rows[0].PlacementTotal == nil || *rows[0].PlacementTotal != defaultCSRPlacementThreshold {
		t.Errorf("fallback threshold want %d, got %v", defaultCSRPlacementThreshold, rows[0].PlacementTotal)
	}
	if rows[0].PlacementDone == nil || *rows[0].PlacementDone != 3 {
		t.Errorf("done want 3 (5-2), got %v", rows[0].PlacementDone)
	}
}

func TestApplyLUSRPlacements_skipMatchesWithLUSRorCSR(t *testing.T) {
	ts := time.Now()
	rows := []domain.MatchHistoryRawRow{
		{MatchID: "m1", StartTime: &ts, PairName: strPtr("Arena:Slayer"), SkillRatingType: strPtr("LUSR")},
		{MatchID: "m2", StartTime: &ts, PairName: strPtr("Arena:Slayer"), SkillRatingType: strPtr("CSR")},
		{MatchID: "m3", StartTime: &ts, PairName: strPtr("Arena:Slayer")}, // sans rating → placement
	}
	applyLUSRPlacements(rows)
	if rows[0].PlacementDone != nil {
		t.Errorf("m1 (LUSR existant) ne doit pas être en placement")
	}
	if rows[1].PlacementDone != nil {
		t.Errorf("m2 (CSR existant) ne doit pas être en placement")
	}
	if rows[2].PlacementDone == nil || *rows[2].PlacementDone != 1 {
		t.Errorf("m3 (sans rating, chaîne LUSR) doit être placement 1/10, got %v", rows[2].PlacementDone)
	}
}

func TestApplyLUSRPlacements_skipRankedAndFirefight(t *testing.T) {
	ts := time.Now()
	rows := []domain.MatchHistoryRawRow{
		{MatchID: "ranked", StartTime: &ts, PairName: strPtr("Ranked:Slayer")},     // chaîne "" (Ranked → CSR)
		{MatchID: "firefight", StartTime: &ts, PairName: strPtr("Firefight:KOTH")}, // chaîne "" (Firefight → PvE)
		{MatchID: "btb", StartTime: &ts, PairName: strPtr("BTB:CTF on Highpower")}, // chaîne "btb"
	}
	applyLUSRPlacements(rows)
	if rows[0].PlacementDone != nil {
		t.Errorf("ranked ne doit pas être placement LUSR")
	}
	if rows[1].PlacementDone != nil {
		t.Errorf("firefight ne doit pas être placement LUSR")
	}
	if rows[2].PlacementDone == nil || *rows[2].PlacementDone != 1 {
		t.Errorf("btb doit être placement 1/10, got %v", rows[2].PlacementDone)
	}
}

// TestApplyLUSRPlacements_bigTeamBattle_fewerThan10_userReportedScenario reproduit
// le signalement V72-32 : un joueur ayant joué SEULEMENT quelques matchs Big Team
// Battle (chaîne "btb") au total, dont plusieurs "hier". Confirme le mécanisme
// exact — < sync.LUSRPlacementThreshold (10) matchs dans LA CHAÎNE BTB, pas un
// total global de matchs joués — et que TOUS les matchs BTB de la chaîne (donc
// ceux d'hier) reçoivent PlacementDone/PlacementTotal, jamais nil silencieux.
// C'est ce signal que consomme ExplorerMatchesTable.placement.tsx (front) pour
// afficher « En placement » sur Perf/ΔPerf/Note à la place du "-".
func TestApplyLUSRPlacements_bigTeamBattle_fewerThan10_userReportedScenario(t *testing.T) {
	base := time.Date(2026, 7, 20, 18, 0, 0, 0, time.UTC)
	// 6 matchs BTB au total (< 10) : 3 "il y a plusieurs jours" + 3 "hier"
	// (24/07), aucun n'a encore de LUSR (chaîne trop jeune).
	rows := []domain.MatchHistoryRawRow{
		{MatchID: "btb-old-1", StartTime: tPtr(base), PairName: strPtr("BTB:CTF on Highpower")},
		{MatchID: "btb-old-2", StartTime: tPtr(base.Add(24 * time.Hour)), PairName: strPtr("BTB:Slayer on Behemoth")},
		{MatchID: "btb-old-3", StartTime: tPtr(base.Add(48 * time.Hour)), PairName: strPtr("BTB:Stockpile on Deadlock")},
		{MatchID: "btb-yday-1", StartTime: tPtr(base.Add(96 * time.Hour)), PairName: strPtr("BTB:CTF on Fragmentation")},
		{MatchID: "btb-yday-2", StartTime: tPtr(base.Add(97 * time.Hour)), PairName: strPtr("BTB:Slayer on Highpower")},
		{MatchID: "btb-yday-3", StartTime: tPtr(base.Add(98 * time.Hour)), PairName: strPtr("BTB:Stockpile on Behemoth")},
		// Un match Ranked le même jour ne doit pas polluer la chaîne "btb".
		{MatchID: "ranked-same-day", StartTime: tPtr(base.Add(50 * time.Hour)), PairName: strPtr("Ranked:Slayer")},
	}
	count := applyLUSRPlacements(rows)
	if count != 6 {
		t.Fatalf("6 matchs BTB attendus en placement (< 10 dans la chaîne), got %d", count)
	}
	for i, want := range []int{1, 2, 3, 4, 5, 6} {
		r := rows[i]
		if r.PlacementDone == nil || *r.PlacementDone != want {
			t.Errorf("%s: PlacementDone want %d, got %v", r.MatchID, want, r.PlacementDone)
		}
		if r.PlacementTotal == nil || *r.PlacementTotal != sync.LUSRPlacementThreshold {
			t.Errorf("%s: PlacementTotal want %d, got %v", r.MatchID, sync.LUSRPlacementThreshold, r.PlacementTotal)
		}
	}
	if rows[6].PlacementDone != nil {
		t.Errorf("ranked-same-day (chaîne différente) ne doit pas être en placement BTB, got %v", rows[6].PlacementDone)
	}
}

func tPtr(t time.Time) *time.Time { return &t }

// f64Ptr : helper local pour les PerformanceScore des tests de placement perf.
func f64Ptr(v float64) *float64 { return &v }

// btbRow construit un match BTB (chaîne perf "btb") avec un perf_score et un
// LUSR optionnels. lusr=true simule une chaîne LUSR DÉJÀ calibrée (le cas JGtm).
func btbRow(id string, at time.Time, perf *float64, lusr bool) domain.MatchHistoryRawRow {
	r := domain.MatchHistoryRawRow{
		MatchID:          id,
		StartTime:        tPtr(at),
		PairName:         strPtr("BTB:CTF on Highpower"),
		Outcome:          domain.OutcomeWin,
		PerformanceScore: perf,
	}
	if lusr {
		r.SkillRatingType = strPtr("LUSR")
		r.SkillTierLabel = strPtr("Or II")
	}
	return r
}

// TestApplyPerfPlacements_jgtmBTB_lusrEstabli_chaineSousLeSeuil reproduit LE cas
// signalé (V72-34, vérifié sur les données réelles de JGtm) : chaîne BTB de 8 matchs
// perf-éligibles, LUSR DÉJÀ établi (Rang "Or II") sur chacun. Le placement LUSR ne
// couvre pas ces matchs (applyLUSRPlacements saute ceux qui ont un LUSR) — c'est
// précisément le trou que le signal perf dédié comble : Perf/ΔPerf doivent afficher
// « En placement (8/10) » alors que la colonne Note reste inchangée.
func TestApplyPerfPlacements_jgtmBTB_lusrEstabli_chaineSousLeSeuil(t *testing.T) {
	base := time.Date(2026, 7, 24, 19, 0, 0, 0, time.UTC)
	rows := make([]domain.MatchHistoryRawRow, 0, 8)
	for i := 0; i < 8; i++ {
		rows = append(rows, btbRow("btb-"+strconv.Itoa(i), base.Add(time.Duration(i)*time.Hour), nil, true))
	}
	if got := applyPerfPlacements(context.Background(), rows); got != 8 {
		t.Fatalf("8 matchs BTB attendus en placement perf, got %d", got)
	}
	// done = TAILLE DE LA CHAÎNE (8), identique sur tous les matchs — progression de
	// calibration, pas un rang par-match.
	for i := range rows {
		r := &rows[i]
		if r.PerfPlacementDone == nil || *r.PerfPlacementDone != 8 {
			t.Errorf("%s: PerfPlacementDone want 8, got %v", r.MatchID, r.PerfPlacementDone)
		}
		if r.PerfPlacementTotal == nil || *r.PerfPlacementTotal != sync.MinMatchesPerChainForRelative {
			t.Errorf("%s: PerfPlacementTotal want %d, got %v", r.MatchID, sync.MinMatchesPerChainForRelative, r.PerfPlacementTotal)
		}
		// Le signal de classement (colonne Note) reste vierge : LUSR établi.
		if r.PlacementDone != nil {
			t.Errorf("%s: PlacementDone doit rester nil (LUSR établi), got %v", r.MatchID, r.PlacementDone)
		}
	}
}

// TestApplyPerfPlacements_chaineCalibree_aucunSignal : ≥ 10 matchs éligibles dans la
// chaîne → la chaîne est calibrée. Un match sans perf_score y est un trou structurel
// ou un retard de sync : AUCUN champ perf_placement (on ne masque pas un défaut de
// fraîcheur derrière un faux état de placement).
func TestApplyPerfPlacements_chaineCalibree_aucunSignal(t *testing.T) {
	base := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	rows := make([]domain.MatchHistoryRawRow, 0, 12)
	for i := 0; i < 11; i++ {
		rows = append(rows, btbRow("scored-"+strconv.Itoa(i), base.Add(time.Duration(i)*time.Hour), f64Ptr(55), true))
	}
	// 12e match de la chaîne, sans perf_score (retard de sync).
	rows = append(rows, btbRow("lagging", base.Add(50*time.Hour), nil, true))

	applyPerfPlacements(context.Background(), rows)
	for i := range rows {
		if rows[i].PerfPlacementDone != nil || rows[i].PerfPlacementTotal != nil {
			t.Errorf("%s: chaîne calibrée (12 matchs) → aucun signal perf attendu, got %v/%v",
				rows[i].MatchID, rows[i].PerfPlacementDone, rows[i].PerfPlacementTotal)
		}
	}
}

// TestApplyPerfPlacements_matchAvecPerfScore_aucunSignal : un match qui a DÉJÀ sa
// perf n'est jamais annoncé « en placement », même si sa chaîne est courte (cas
// limite : la chaîne vient d'atteindre le seuil).
func TestApplyPerfPlacements_matchAvecPerfScore_aucunSignal(t *testing.T) {
	base := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	rows := []domain.MatchHistoryRawRow{
		btbRow("with-score", base, f64Ptr(72), true),
		btbRow("without-score", base.Add(time.Hour), nil, true),
	}
	applyPerfPlacements(context.Background(), rows)
	if rows[0].PerfPlacementDone != nil {
		t.Errorf("match avec perf_score ne doit jamais porter le signal, got %v", rows[0].PerfPlacementDone)
	}
	// L'autre match de la chaîne (2 éligibles < 10) est bien marqué 2/10.
	if rows[1].PerfPlacementDone == nil || *rows[1].PerfPlacementDone != 2 {
		t.Errorf("without-score: PerfPlacementDone want 2 (taille de chaîne), got %v", rows[1].PerfPlacementDone)
	}
}

// TestApplyPerfPlacements_dnfEtExclus_nonComptesEtNonMarques : les matchs DNF
// (outcome=4) et exclus manuellement ne sont pas perf-éligibles — miroir du
// périmètre du batch : WHERE de loadHistoryForPerf (COALESCE(outcome,0) != 4)
// + filtre loadExcludedMatchIDs appliqué à allMatches avant la boucle de chaîne
// (sync/performance.go). Ils ne gonflent pas le compteur de chaîne et ne
// reçoivent jamais le badge. 3e critère couvert par
// TestApplyPerfPlacements_sansStartTime_nonCompte.
func TestApplyPerfPlacements_dnfEtExclus_nonComptesEtNonMarques(t *testing.T) {
	base := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	rows := []domain.MatchHistoryRawRow{
		btbRow("ok-1", base, nil, false),
		btbRow("ok-2", base.Add(time.Hour), nil, false),
	}
	dnf := btbRow("dnf", base.Add(2*time.Hour), nil, false)
	dnf.Outcome = domain.OutcomeDNF
	excluded := btbRow("excluded", base.Add(3*time.Hour), nil, false)
	excluded.IsExcluded = true
	rows = append(rows, dnf, excluded)

	applyPerfPlacements(context.Background(), rows)

	// 2 éligibles seulement → les 2 matchs OK portent 2/10.
	for _, i := range []int{0, 1} {
		if rows[i].PerfPlacementDone == nil || *rows[i].PerfPlacementDone != 2 {
			t.Errorf("%s: PerfPlacementDone want 2 (DNF/exclus non comptés), got %v",
				rows[i].MatchID, rows[i].PerfPlacementDone)
		}
	}
	for _, i := range []int{2, 3} {
		if rows[i].PerfPlacementDone != nil {
			t.Errorf("%s: match non éligible ne doit pas porter le signal, got %v",
				rows[i].MatchID, rows[i].PerfPlacementDone)
		}
	}
}

// TestApplyPerfPlacements_sansStartTime_nonCompte : 3e critère d'éligibilité du
// batch, jusqu'ici absent du placement (contre-revue V7.2). loadHistoryForPerf
// filtre `mr.start_time IS NOT NULL` : un match sans start_time n'entre PAS dans
// l'historique de chaîne du batch. Le compter côté placement affichait un « X/10 »
// supérieur au compteur réel du batch (le badge « En placement » aurait annoncé
// 3/10 quand le batch n'en voit que 2).
func TestApplyPerfPlacements_sansStartTime_nonCompte(t *testing.T) {
	base := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	rows := []domain.MatchHistoryRawRow{
		btbRow("ok-1", base, nil, false),
		btbRow("ok-2", base.Add(time.Hour), nil, false),
	}
	noStart := btbRow("sans-start-time", base.Add(2*time.Hour), nil, false)
	noStart.StartTime = nil
	rows = append(rows, noStart)

	applyPerfPlacements(context.Background(), rows)

	for _, i := range []int{0, 1} {
		if rows[i].PerfPlacementDone == nil || *rows[i].PerfPlacementDone != 2 {
			t.Errorf("%s: PerfPlacementDone want 2 (match sans start_time non compté par le batch), got %v",
				rows[i].MatchID, rows[i].PerfPlacementDone)
		}
	}
	if rows[2].PerfPlacementDone != nil {
		t.Errorf("sans-start-time: match hors périmètre du batch, aucun signal attendu, got %v",
			rows[2].PerfPlacementDone)
	}
}

// TestApplyPerfPlacements_chainesIndependantes : le comptage est PAR CHAÎNE. Une
// chaîne courte (btb) reçoit le signal sans que la chaîne longue (ranked) soit
// affectée, et les compteurs ne se mélangent pas.
func TestApplyPerfPlacements_chainesIndependantes(t *testing.T) {
	base := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	var rows []domain.MatchHistoryRawRow
	// 3 matchs BTB sans perf (chaîne courte).
	for i := 0; i < 3; i++ {
		rows = append(rows, btbRow("btb-"+strconv.Itoa(i), base.Add(time.Duration(i)*time.Hour), nil, false))
	}
	// 11 matchs Ranked scorés (chaîne calibrée, chaîne perf "ranked").
	for i := 0; i < 11; i++ {
		rows = append(rows, domain.MatchHistoryRawRow{
			MatchID:          "ranked-" + strconv.Itoa(i),
			StartTime:        tPtr(base.Add(time.Duration(100+i) * time.Hour)),
			PairName:         strPtr("Ranked:Slayer"),
			IsRanked:         true,
			Outcome:          domain.OutcomeWin,
			PerformanceScore: f64Ptr(60),
		})
	}
	applyPerfPlacements(context.Background(), rows)

	for i := 0; i < 3; i++ {
		if rows[i].PerfPlacementDone == nil || *rows[i].PerfPlacementDone != 3 {
			t.Errorf("%s: want 3/10 (chaîne btb isolée), got %v", rows[i].MatchID, rows[i].PerfPlacementDone)
		}
	}
	for i := 3; i < len(rows); i++ {
		if rows[i].PerfPlacementDone != nil {
			t.Errorf("%s: chaîne ranked calibrée+scorée → aucun signal, got %v",
				rows[i].MatchID, rows[i].PerfPlacementDone)
		}
	}
}
