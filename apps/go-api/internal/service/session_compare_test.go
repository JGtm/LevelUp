package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

func ptr(s string) *string { return &s }

func makeMatch(label string, kills, deaths int, outcome *int) legacymatch.StatsMatchRow {
	return legacymatch.StatsMatchRow{
		SessionLabel: &label,
		Kills:        kills,
		Deaths:       deaths,
		Outcome:      outcome,
		StartTime:    time.Now(),
	}
}

func TestExtractSessionLabels(t *testing.T) {
	matches := []legacymatch.StatsMatchRow{
		makeMatch("S1", 10, 5, nil),
		makeMatch("S2", 8, 6, nil),
		makeMatch("S1", 12, 4, nil),
	}
	labels := extractSessionLabels(matches)
	if len(labels) != 2 {
		t.Fatalf("expected 2, got %d", len(labels))
	}
}

func TestExtractSessionLabels_NoLabels(t *testing.T) {
	labels := extractSessionLabels(nil)
	if len(labels) != 0 {
		t.Fatalf("expected 0, got %d", len(labels))
	}
}

func TestLastOrNil(t *testing.T) {
	labels := []string{"S1", "S2", "S3"}
	if got := lastOrNil(labels, nil); got != "S3" {
		t.Fatalf("expected S3, got %s", got)
	}
	if got := lastOrNil(labels, ptr("override")); got != "override" {
		t.Fatalf("expected override, got %s", got)
	}
	if got := lastOrNil(nil, nil); got != "" {
		t.Fatalf("expected empty, got %s", got)
	}
}

func TestSecondLastOrNil(t *testing.T) {
	labels := []string{"S1", "S2", "S3"}
	if got := secondLastOrNil(labels, nil); got != "S2" {
		t.Fatalf("expected S2, got %s", got)
	}
	if got := secondLastOrNil(labels, ptr("override")); got != "override" {
		t.Fatalf("expected override, got %s", got)
	}
	if got := secondLastOrNil([]string{"S1"}, nil); got != "" {
		t.Fatalf("expected empty, got %s", got)
	}
}

func TestFilterBySession(t *testing.T) {
	matches := []legacymatch.StatsMatchRow{
		makeMatch("S1", 10, 5, nil),
		makeMatch("S2", 8, 6, nil),
		makeMatch("S1", 12, 4, nil),
	}
	filtered := filterBySession(matches, "S1")
	if len(filtered) != 2 {
		t.Fatalf("expected 2, got %d", len(filtered))
	}
}

func TestFilterBySession_NoLabel(t *testing.T) {
	filtered := filterBySession(nil, "S1")
	if filtered != nil {
		t.Fatal("expected nil")
	}
}

func TestBuildCompareEntry_Nil(t *testing.T) {
	entry := buildCompareEntry(nil, "S1", 225)
	if entry != nil {
		t.Fatal("expected nil for empty matches")
	}
}

func TestBuildCompareEntry_WithMatches(t *testing.T) {
	win := analysis.OutcomeWin
	loss := analysis.OutcomeLoss
	matches := []legacymatch.StatsMatchRow{
		makeMatch("S1", 15, 5, &win),
		makeMatch("S1", 10, 8, &loss),
		makeMatch("S1", 20, 3, &win),
	}
	entry := buildCompareEntry(matches, "S1", 225)
	if entry == nil {
		t.Fatal("expected non-nil")
	}
	if entry.TotalMatches != 3 {
		t.Fatalf("expected 3, got %d", entry.TotalMatches)
	}
	if entry.Wins != 2 {
		t.Fatalf("expected 2 wins, got %d", entry.Wins)
	}
}

func TestBuildCompareEntry_AvgAccuracy(t *testing.T) {
	acc := func(v float64) *float64 { return &v }
	// Accuracy stockée en pourcentage 0..100 (cf. averageAccuracy) ; le contrat AvgAccuracy
	// est 0..1 (ADR 0006) → la moyenne est normalisée /100.
	m1 := makeMatch("S1", 15, 5, nil)
	m1.Accuracy = acc(50.0)
	m2 := makeMatch("S1", 10, 8, nil)
	m2.Accuracy = acc(60.0)
	m3 := makeMatch("S1", 20, 3, nil) // pas de précision → ignoré dans la moyenne

	entry := buildCompareEntry([]legacymatch.StatsMatchRow{m1, m2, m3}, "S1", 225)
	if entry == nil || entry.AvgAccuracy == nil {
		t.Fatalf("AvgAccuracy attendu non-nil, got %+v", entry)
	}
	if *entry.AvgAccuracy != 0.55 { // (50 + 60) / 2 / 100, arrondi 3 décimales
		t.Errorf("AvgAccuracy = %v, want 0.55 (moyenne des matchs avec précision)", *entry.AvgAccuracy)
	}

	// Aucune précision → nil (pas de KPI à afficher).
	noAcc := buildCompareEntry([]legacymatch.StatsMatchRow{makeMatch("S1", 10, 5, nil)}, "S1", 225)
	if noAcc.AvgAccuracy != nil {
		t.Errorf("AvgAccuracy attendu nil sans données précision, got %v", *noAcc.AvgAccuracy)
	}
}

func TestWinRate(t *testing.T) {
	win := analysis.OutcomeWin
	loss := analysis.OutcomeLoss
	matches := []legacymatch.StatsMatchRow{
		makeMatch("S1", 10, 5, &win),
		makeMatch("S1", 8, 6, &loss),
	}
	rate := winRate(matches)
	if rate != 50 {
		t.Fatalf("expected 50, got %f", rate)
	}
}

func TestAvgKD(t *testing.T) {
	matches := []legacymatch.StatsMatchRow{
		makeMatch("S1", 10, 5, nil),
		makeMatch("S1", 20, 10, nil),
	}
	kd := avgKD(matches)
	if kd != 2.0 {
		t.Fatalf("expected 2.0, got %f", kd)
	}
}

func TestBuildCompareMetrics_TwoSessions(t *testing.T) {
	win := analysis.OutcomeWin
	a := []legacymatch.StatsMatchRow{makeMatch("S1", 15, 5, &win)}
	b := []legacymatch.StatsMatchRow{makeMatch("S2", 10, 10, nil)}
	metrics := buildCompareMetrics(a, b)
	if len(metrics) < 4 {
		t.Fatalf("expected >=4 metrics, got %d", len(metrics))
	}
}

func TestEffectiveKDA(t *testing.T) {
	// Chemin nominal : le KDA stocké est rendu tel quel.
	precomputed := 1.75
	if got := effectiveKDA(legacymatch.StatsMatchRow{KDA: &precomputed}); got == nil || *got != precomputed {
		t.Fatalf("expected precomputed KDA, got %#v", got)
	}
	// Fallback = FDA NET per-match ((kills + assists/3) − deaths), JAMAIS un quotient.
	// 9 kills, 0 assist, 0 mort → 9 + 0/3 − 0 = 9.
	if got := effectiveKDA(legacymatch.StatsMatchRow{Kills: 9, Deaths: 0}); got == nil || *got != 9 {
		t.Fatalf("expected net fallback 9, got %#v", got)
	}
	// 9 kills, 0 assist, 4 morts → 9 + 0/3 − 4 = 5 (et NON 9/4 = 2.25).
	if got := effectiveKDA(legacymatch.StatsMatchRow{Kills: 9, Deaths: 4}); got == nil || *got != 5 {
		t.Fatalf("expected net fallback 5, got %#v", got)
	}
	// 10 kills, 6 assists, 8 morts → 10 + 6/3 − 8 = 4 (vérifie le terme assists/3).
	if got := effectiveKDA(legacymatch.StatsMatchRow{Kills: 10, Assists: 6, Deaths: 8}); got == nil || *got != 4 {
		t.Fatalf("expected net fallback 4 with assists, got %#v", got)
	}
}

func TestClassifySessionCategory(t *testing.T) {
	tests := []struct {
		name  string
		match legacymatch.StatsMatchRow
		want  string
	}{
		{name: "firefight", match: legacymatch.StatsMatchRow{PlaylistName: "Firefight Normal"}, want: "Firefight"},
		{name: "ranked", match: legacymatch.StatsMatchRow{IsRanked: true, PlaylistName: "Arena"}, want: "Ranked"},
		{name: "btb", match: legacymatch.StatsMatchRow{PlaylistName: "Big Team Battle"}, want: "BTB"},
		{name: "arena", match: legacymatch.StatsMatchRow{PlaylistName: "Quick Play"}, want: "Arena"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifySessionCategory(tt.match); got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

// TestBuildCompareEntry_DerivedMetrics couvre les champs WinRate/KDR/KillsPerMatch
// ajoutés (Phase 2) — alimentés par les mêmes helpers que compare_metrics.
func TestBuildCompareEntry_DerivedMetrics(t *testing.T) {
	win := analysis.OutcomeWin
	loss := analysis.OutcomeLoss
	matches := []legacymatch.StatsMatchRow{
		makeMatch("S1", 10, 5, &win),
		makeMatch("S1", 6, 5, &loss),
	}
	entry := buildCompareEntry(matches, "S1", 225)
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.WinRate != 50 { // 1 win / 2 matchs
		t.Fatalf("WinRate: want 50, got %v", entry.WinRate)
	}
	if entry.KDR != 16.0/10.0 { // (10+6) kills / (5+5) deaths
		t.Fatalf("KDR: want 1.6, got %v", entry.KDR)
	}
	if entry.KillsPerMatch != 8 { // 16 kills / 2 matchs
		t.Fatalf("KillsPerMatch: want 8, got %v", entry.KillsPerMatch)
	}
	// KDA de session = agrégat NET par match ((Σk + Σa/3) − Σd) / nb_matchs.
	// (16 + 0/3 − 10) / 2 = 3.0 — JAMAIS Σk/Σd = 1.6.
	if entry.KDA == nil || *entry.KDA != 3.0 {
		t.Fatalf("KDA: want net 3.0, got %v", entry.KDA)
	}
}

// TestBuildCompareEntry_FragAggregates couvre les agrégats du radar de frags en
// AGRÉGATS DE SESSION : spree = MAX atteint, HS/PK = TOTAUX de la session.
func TestBuildCompareEntry_FragAggregates(t *testing.T) {
	spree := func(v int) *int { return &v }
	rows := []legacymatch.StatsMatchRow{
		{SessionLabel: ptr("S1"), StartTime: time.Now(), MaxKillingSpree: spree(6), HeadshotKills: spree(2), PerfectKills: spree(0)},
		{SessionLabel: ptr("S1"), StartTime: time.Now(), MaxKillingSpree: spree(9), HeadshotKills: spree(4), PerfectKills: spree(1)},
		{SessionLabel: ptr("S1"), StartTime: time.Now(), MaxKillingSpree: spree(3), HeadshotKills: spree(1), PerfectKills: spree(2)},
	}
	entry := buildCompareEntry(rows, "S1", 225)
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.MaxKillingSpree == nil || *entry.MaxKillingSpree != 9 { // max(6,9,3)
		t.Fatalf("MaxKillingSpree: want 9 (max session), got %v", entry.MaxKillingSpree)
	}
	if entry.TotalHeadshotKills == nil || *entry.TotalHeadshotKills != 7 { // 2+4+1
		t.Fatalf("TotalHeadshotKills: want 7 (total session), got %v", entry.TotalHeadshotKills)
	}
	if entry.TotalPerfectKills == nil || *entry.TotalPerfectKills != 3 { // 0+1+2
		t.Fatalf("TotalPerfectKills: want 3 (total session), got %v", entry.TotalPerfectKills)
	}
}

// TestBuildCompareEntry_AvgLifeSeconds couvre l'agrégat durée de vie moyenne
// (moyenne des AvgLifeSeconds par match) projeté pour la KPI "Durée de vie".
func TestBuildCompareEntry_AvgLifeSeconds(t *testing.T) {
	l1, l2 := 30.0, 50.0
	rows := []legacymatch.StatsMatchRow{
		{SessionLabel: ptr("S1"), StartTime: time.Now(), AvgLifeSeconds: &l1},
		{SessionLabel: ptr("S1"), StartTime: time.Now(), AvgLifeSeconds: &l2},
	}
	entry := buildCompareEntry(rows, "S1", 225)
	if entry == nil || entry.AvgLifeSeconds == nil {
		t.Fatal("expected non-nil AvgLifeSeconds")
	}
	if *entry.AvgLifeSeconds != 40 { // (30 + 50) / 2
		t.Fatalf("AvgLifeSeconds: want 40, got %v", *entry.AvgLifeSeconds)
	}
}

// TestBuildSessionParticipationProfile_ObjectiveAndScore couvre les axes Objective
// (somme PSA) et Score (score personnel par minute jouée) du profil de participation.
// Objective reste piloté par le PSA ; Score = ΣPS / (Σtime_played/60), désormais
// indépendant du PSA (avant : résiduel medals/streaks ≈ 0, axe mort).
func TestBuildSessionParticipationProfile_ObjectiveAndScore(t *testing.T) {
	dd, dt := 3000.0, 2000.0
	ps := 2000
	tp := 600 // 10 min → PS/min = 2000 / 10 = 200
	rows := []legacymatch.StatsMatchRow{
		{MatchID: "m1", Kills: 10, Assists: 4, Deaths: 5, PersonalScore: &ps, TimePlayedSeconds: &tp, DamageDealt: &dd, DamageTaken: &dt},
	}
	axisVal := func(axes []domain.SessionParticipationAxis, name string) float64 {
		for _, a := range axes {
			if a.Name == name {
				return a.Value
			}
		}
		return -1
	}
	// Sans scores PSA → Objective à 0 (dégradation gracieuse) ; Score reste calculé.
	without := buildSessionParticipationProfile(rows, nil, 225)
	if v := axisVal(without, "objective"); v != 0 {
		t.Fatalf("Objective sans PSA: want 0, got %v", v)
	}
	// Score = 200/min normalisé (195×1.25=243.75) → ~82/100, vivant et > 0.
	if v := axisVal(without, "score"); v <= 0 {
		t.Fatalf("Score PS/min: want > 0, got %v", v)
	}
	// Avec PSA → Objective > 0 ; Score inchangé (ne dépend plus du PSA/résiduel).
	with := buildSessionParticipationProfile(rows, map[string]int{"m1": 500}, 225)
	if v := axisVal(with, "objective"); v <= 0 {
		t.Fatalf("Objective avec PSA: want > 0, got %v", v)
	}
	if got, exp := axisVal(with, "score"), axisVal(without, "score"); got != exp {
		t.Fatalf("Score ne doit pas dépendre du PSA: with=%v without=%v", got, exp)
	}
}

// TestBuildSessionParticipationProfile_ScoreZeroWithoutTime : sans time_played
// renseigné, l'axe Score (PS/min) dégrade proprement à 0 plutôt que de diviser
// par zéro.
func TestBuildSessionParticipationProfile_ScoreZeroWithoutTime(t *testing.T) {
	ps := 2000
	rows := []legacymatch.StatsMatchRow{
		{MatchID: "m1", Kills: 10, Assists: 4, Deaths: 5, PersonalScore: &ps},
	}
	profile := buildSessionParticipationProfile(rows, nil, 225)
	for _, a := range profile {
		if a.Name == "score" && a.Value != 0 {
			t.Fatalf("Score sans time_played: want 0, got %v", a.Value)
		}
	}
}

// TestBuildCompareEntry_FragAggregates_AllNil : aucun match n'a de stats de frags
// → agrégats nil (le radar dégrade en série vide côté front), pas de zéro trompeur.
func TestBuildCompareEntry_FragAggregates_AllNil(t *testing.T) {
	entry := buildCompareEntry([]legacymatch.StatsMatchRow{
		{SessionLabel: ptr("S1"), StartTime: time.Now(), Kills: 5, Deaths: 3},
	}, "S1", 225)
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.MaxKillingSpree != nil || entry.TotalHeadshotKills != nil || entry.TotalPerfectKills != nil {
		t.Fatalf("expected nil frag aggregates, got spree=%v hs=%v pk=%v",
			entry.MaxKillingSpree, entry.TotalHeadshotKills, entry.TotalPerfectKills)
	}
}

// TestBuildSessionDetailRows_EnrichedFields couvre les colonnes enrichies (Phase 3)
// projetées depuis StatsMatchRow : map FR-préférée, ΔMMR, perf_tier, durée, rating,
// + (lot P2) ModeUI localisé/normalisé et Δ rang par match.
func TestBuildSessionDetailRows_EnrichedFields(t *testing.T) {
	team, enemy, perf, rating, ratingDelta := 1500.0, 1400.0, 72.0, 1450.0, 12.5
	dur := 540
	dd, dt := 3000.0, 1500.0
	rk := 3
	row := legacymatch.StatsMatchRow{
		MatchID:           "m1",
		StartTime:         time.Now(),
		Kills:             10,
		Deaths:            5,
		MapName:           "Live Fire",
		MapNameFR:         "Tir réel",
		PairName:          "Arena:Slayer on Live Fire",
		TeamMMR:           &team,
		EnemyMMR:          &enemy,
		PerfScoreComputed: &perf,
		TimePlayedSeconds: &dur,
		SkillRatingValue:  &rating,
		SkillRatingType:   "csr",
		SkillRatingDelta:  &ratingDelta,
		DamageDealt:       &dd,
		DamageTaken:       &dt,
		Rank:              &rk,
	}
	out := buildSessionDetailRows([]legacymatch.StatsMatchRow{row}, nil, "fr", nil, nil)
	if len(out) != 1 {
		t.Fatalf("expected 1 row, got %d", len(out))
	}
	r := out[0]
	if r.MapName != "Tir réel" {
		t.Fatalf("MapName: want FR-preferred 'Tir réel', got %q", r.MapName)
	}
	wantMode := derefString(analysis.ResolveModeUI(&row.PairName, &row.PairNameFR))
	if wantMode == "" || r.ModeUI != wantMode {
		t.Fatalf("ModeUI: want %q (via ResolveModeUI), got %q", wantMode, r.ModeUI)
	}
	if r.SkillRatingDelta == nil || *r.SkillRatingDelta != 12.5 {
		t.Fatalf("SkillRatingDelta: want 12.5, got %v", r.SkillRatingDelta)
	}
	if r.DeltaMMR == nil || *r.DeltaMMR != 100 { // 1500 - 1400
		t.Fatalf("DeltaMMR: want 100, got %v", r.DeltaMMR)
	}
	if r.PerfTier != int(analysis.PerfTier(perf)) {
		t.Fatalf("PerfTier: want %d, got %d", int(analysis.PerfTier(perf)), r.PerfTier)
	}
	if r.DurationSeconds == nil || *r.DurationSeconds != 540 {
		t.Fatalf("DurationSeconds: want 540, got %v", r.DurationSeconds)
	}
	if r.SkillRatingValue == nil || *r.SkillRatingValue != 1450 {
		t.Fatalf("SkillRatingValue: want 1450, got %v", r.SkillRatingValue)
	}
	if r.SkillRatingType != "csr" {
		t.Fatalf("SkillRatingType: want csr, got %q", r.SkillRatingType)
	}
	if r.DamageDealt == nil || *r.DamageDealt != 3000 {
		t.Fatalf("DamageDealt: want 3000, got %v", r.DamageDealt)
	}
	if r.DamageTaken == nil || *r.DamageTaken != 1500 {
		t.Fatalf("DamageTaken: want 1500, got %v", r.DamageTaken)
	}
	if r.Placement == nil || *r.Placement != 3 { // projeté depuis StatsMatchRow.Rank
		t.Fatalf("Placement: want 3, got %v", r.Placement)
	}
}

// TestBuildSessionDetailRows_Locale couvre la résolution FR/EN des libellés
// cartes/modes/playlists selon la locale (aligné Home/Explorer).
func TestBuildSessionDetailRows_Locale(t *testing.T) {
	row := legacymatch.StatsMatchRow{
		MatchID:        "m1",
		StartTime:      time.Now(),
		MapName:        "Live Fire",
		MapNameFR:      "Tir réel",
		PlaylistName:   "Ranked Arena",
		PlaylistNameFR: "Arène classée",
		PairName:       "Arena:Slayer on Live Fire",
		PairNameFR:     "Arène:Massacre sur Tir réel",
	}
	en := buildSessionDetailRows([]legacymatch.StatsMatchRow{row}, nil, "en", nil, nil)[0]
	if en.MapName != "Live Fire" {
		t.Fatalf("EN MapName: want 'Live Fire', got %q", en.MapName)
	}
	if en.PlaylistName != "Ranked Arena" {
		t.Fatalf("EN PlaylistName: want 'Ranked Arena', got %q", en.PlaylistName)
	}
	if want := derefString(analysis.ResolveModeUI(&row.PairName, nil)); en.ModeUI != want {
		t.Fatalf("EN ModeUI: want %q (EN normalisé), got %q", want, en.ModeUI)
	}

	fr := buildSessionDetailRows([]legacymatch.StatsMatchRow{row}, nil, "fr", nil, nil)[0]
	if fr.MapName != "Tir réel" {
		t.Fatalf("FR MapName: want 'Tir réel', got %q", fr.MapName)
	}
	if fr.PlaylistName != "Arène classée" {
		t.Fatalf("FR PlaylistName: want 'Arène classée', got %q", fr.PlaylistName)
	}
	if want := derefString(analysis.ResolveModeUI(&row.PairName, &row.PairNameFR)); fr.ModeUI != want {
		t.Fatalf("FR ModeUI: want %q, got %q", want, fr.ModeUI)
	}
}

// TestBuildSessionDetailRows_NilEnrichment vérifie la dégradation gracieuse :
// pas de MMR/perf/rating → champs nil/zéro, pas de panic.
func TestBuildSessionDetailRows_NilEnrichment(t *testing.T) {
	row := legacymatch.StatsMatchRow{MatchID: "m1", StartTime: time.Now(), Kills: 3, Deaths: 2}
	out := buildSessionDetailRows([]legacymatch.StatsMatchRow{row}, nil, "fr", nil, nil)
	if len(out) != 1 {
		t.Fatalf("expected 1 row, got %d", len(out))
	}
	r := out[0]
	if r.DeltaMMR != nil {
		t.Fatalf("DeltaMMR: want nil (no MMR), got %v", *r.DeltaMMR)
	}
	if r.PerfTier != 0 {
		t.Fatalf("PerfTier: want 0 (no perf score), got %d", r.PerfTier)
	}
	// Sans attendu → KdaExpected nil (null-safety).
	if r.KdaExpected != nil {
		t.Fatalf("KdaExpected: want nil (no expected), got %v", *r.KdaExpected)
	}
}

// TestBuildSessionDetailRows_ExpectedFDA vérifie la projection de l'écart au FDA
// attendu (K/D depuis StatsMatchRow, assists depuis le batch) → KdaExpected.
func TestBuildSessionDetailRows_ExpectedFDA(t *testing.T) {
	ke, de := 9.0, 4.0
	assistsExp := 3.0
	row := legacymatch.StatsMatchRow{
		MatchID: "m1", StartTime: time.Now(), Kills: 10, Deaths: 4,
		KillsExpected: &ke, DeathsExpected: &de,
	}
	out := buildSessionDetailRows([]legacymatch.StatsMatchRow{row}, nil, "fr",
		map[string]*float64{"m1": &assistsExp}, nil)
	if len(out) != 1 {
		t.Fatalf("expected 1 row, got %d", len(out))
	}
	r := out[0]
	if r.KillsExpected == nil || *r.KillsExpected != ke {
		t.Fatalf("KillsExpected = %v, want %v", r.KillsExpected, ke)
	}
	if r.AssistsExpected == nil || *r.AssistsExpected != assistsExp {
		t.Fatalf("AssistsExpected = %v, want %v", r.AssistsExpected, assistsExp)
	}
	// 9 + 3/3 - 4 = 6
	if r.KdaExpected == nil || *r.KdaExpected != 6.0 {
		t.Fatalf("KdaExpected = %v, want 6", r.KdaExpected)
	}
}
