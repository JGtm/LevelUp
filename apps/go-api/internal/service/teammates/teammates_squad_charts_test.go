package teammates

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// rowWithPersonalScore construit une PlayerMatchRow avec PersonalScore renseigné.
func rowWithPersonalScore(matchID string, ts time.Time, kills, deaths, assists, timePlayed, personalScore int) canonical.PlayerMatchRow {
	k, d, a, tp, ps := kills, deaths, assists, timePlayed, personalScore
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{MatchID: matchID, StartedAtUTC: ts, Outcome: canonical.OutcomeWin},
		Self: canonical.MatchParticipant{
			Outcome:       canonical.OutcomeWin,
			Kills:         &k,
			Deaths:        &d,
			Assists:       &a,
			TimePlayed:    &tp,
			PersonalScore: &ps,
		},
	}
}

// ---------- buildSquadSessionTimeline (teammates.04) ----------

func TestBuildSquadSessionTimeline_GroupAndAggregate(t *testing.T) {
	p1, p2, p3 := 70.0, 50.0, 90.0
	sessA := "A"
	sessB := "B"
	t1 := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	t2 := t1.Add(10 * time.Minute)
	t3 := t1.Add(time.Hour)

	matches := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t1, SessionLabel: &sessA, Outcome: domain.OutcomeWin, PerformanceScore: &p1, TeamMMR: 1500},
		{MatchID: "m2", StartTime: t2, SessionLabel: &sessA, Outcome: domain.OutcomeLoss, PerformanceScore: &p2, TeamMMR: 1480},
		{MatchID: "m3", StartTime: t3, SessionLabel: &sessB, Outcome: domain.OutcomeWin, PerformanceScore: &p3, TeamMMR: 1550},
		// duplicate by match_id (autre teammate sur même match) — doit être dédupliqué
		{MatchID: "m1", StartTime: t1, SessionLabel: &sessA, Outcome: domain.OutcomeWin, PerformanceScore: &p1, TeamMMR: 1500},
	}
	pts := buildSquadSessionTimeline(matches)
	if len(pts) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(pts))
	}
	// Tri chronologique : sessA (t1) avant sessB (t3).
	if pts[0].SessionLabel != "A" || pts[1].SessionLabel != "B" {
		t.Errorf("expected [A, B] order, got [%s, %s]", pts[0].SessionLabel, pts[1].SessionLabel)
	}
	a := pts[0]
	if a.MatchCount != 2 {
		t.Errorf("A: expected match_count=2 (dédup), got %d", a.MatchCount)
	}
	if a.Wins != 1 || a.Losses != 1 {
		t.Errorf("A: expected wins=1 losses=1, got wins=%d losses=%d", a.Wins, a.Losses)
	}
	if a.SquadPerf != 60.0 {
		t.Errorf("A: expected squad_perf=60 ((70+50)/2), got %f", a.SquadPerf)
	}
	if a.WinRate == nil || *a.WinRate != 0.5 {
		t.Errorf("A: expected win_rate=0.5, got %v", a.WinRate)
	}
	if a.TeamMMRAvg == nil || *a.TeamMMRAvg != 1490.0 {
		t.Errorf("A: expected team_mmr_avg=1490, got %v", a.TeamMMRAvg)
	}
	b := pts[1]
	if b.MatchCount != 1 || b.SquadPerf != 90.0 {
		t.Errorf("B: expected match_count=1 squad_perf=90, got %d / %f", b.MatchCount, b.SquadPerf)
	}
}

func TestBuildSquadSessionTimeline_NoSessionLabelBucket(t *testing.T) {
	matches := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: time.Now(), SessionLabel: nil, Outcome: domain.OutcomeWin, TeamMMR: 1500},
	}
	pts := buildSquadSessionTimeline(matches)
	if len(pts) != 1 || pts[0].SessionLabel != "(no session)" {
		t.Errorf("expected single bucket '(no session)', got %v", pts)
	}
}

func TestBuildSquadSessionTimeline_PerfNilLeavesZero(t *testing.T) {
	sessA := "A"
	matches := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: time.Now(), SessionLabel: &sessA, Outcome: domain.OutcomeWin, PerformanceScore: nil},
	}
	pts := buildSquadSessionTimeline(matches)
	if len(pts) != 1 {
		t.Fatalf("expected 1 point, got %d", len(pts))
	}
	if pts[0].SquadPerf != 0 {
		t.Errorf("expected squad_perf=0 when no score, got %f", pts[0].SquadPerf)
	}
}

// ---------- buildSquadPerMinuteStats helpers (teammates.14) ----------

func TestPerMinuteEntry_RoundingAndMatchCount(t *testing.T) {
	// Sanity check : un agrégat 60s × 1 kill ⇒ 1.0 kpm.
	// On vérifie ici uniquement la propriété domain.
	e := domain.SquadPerMinuteEntry{
		Player:           "Me",
		KillsPerMinute:   1.5,
		DeathsPerMinute:  0.7,
		AssistsPerMinute: 0.3,
		MatchCount:       8,
	}
	if e.Player != "Me" || e.MatchCount != 8 {
		t.Errorf("unexpected entry shape: %+v", e)
	}
}

// TestBuildSquadPerMinuteStats_DistinctValuesPerTeammate cadenasse le bug
// bound-to-main : sans squadLoader.LoadFor (résolution per-gamertag), les rows
// teammate étaient en réalité celles du main → toutes les barres affichaient
// la même valeur kpm/dpm/apm pour tout le squad.
func TestBuildSquadPerMinuteStats_DistinctValuesPerTeammate(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	// Main : 10/5/2 sur m1 (600s), agrégé depuis allSquadRows.
	// Teammate friend1 : 4/8/6 sur m1 (600s).
	// Teammate friend2 : 1/3/15 sur m1 (600s).
	mainRows := []canonical.PlayerMatchRow{
		rowWithStatsXUID("xuid-main", "m1", t0, canonical.OutcomeWin, 10, 5, 2, 600, 50, 80),
	}
	f1Rows := []canonical.PlayerMatchRow{
		rowWithStatsXUID("xuid-f1", "m1", t0, canonical.OutcomeWin, 4, 8, 6, 600, 40, 50),
	}
	f2Rows := []canonical.PlayerMatchRow{
		rowWithStatsXUID("xuid-f2", "m1", t0, canonical.OutcomeWin, 1, 3, 15, 600, 35, 45),
	}
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main":    mainRows,
			"friend1": f1Rows,
			"friend2": f2Rows,
		},
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", squadLoader: loader}

	allSquadRows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0, Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5, Assists: 2, TimePlayedSecs: 600},
	}
	got := svc.buildSquadPerMinuteStats(
		context.Background(),
		allSquadRows,
		"main",
		[]string{"friend1", "friend2"},
		nil,
	)
	if len(got) != 3 {
		t.Fatalf("want 3 entries (main + 2 friends), got %d", len(got))
	}
	byPlayer := make(map[string]domain.SquadPerMinuteEntry, len(got))
	for _, e := range got {
		byPlayer[e.Player] = e
	}
	main := byPlayer["main"]
	f1 := byPlayer["friend1"]
	f2 := byPlayer["friend2"]
	// 600s = 10 minutes : kpm = kills / 10.
	wantKPM := map[string]float64{"main": 1.0, "friend1": 0.4, "friend2": 0.1}
	for gt, want := range wantKPM {
		if got := byPlayer[gt].KillsPerMinute; got != want {
			t.Errorf("%s: KillsPerMinute want %.2f, got %.2f", gt, want, got)
		}
	}
	// Bug cadenassé : les 3 valeurs doivent être distinctes (sinon : rows teammate
	// = rows main → mêmes barres pour tout le squad).
	if main.KillsPerMinute == f1.KillsPerMinute || main.KillsPerMinute == f2.KillsPerMinute {
		t.Errorf("teammates ont la même kpm que le main (%.2f) → bug bound-to-main",
			main.KillsPerMinute)
	}
	if f1.AssistsPerMinute == f2.AssistsPerMinute {
		t.Errorf("friend1 et friend2 ont la même apm (%.2f) → bug bound-to-main",
			f1.AssistsPerMinute)
	}
}

// TestBuildSquadPerMinuteStats_NoSquadLoader_TeammatesEmpty verifie que sans
// squadLoader, les coequipiers ressortent avec une entry vide (pas de panic,
// pas de fallback bound-to-main).
func TestBuildSquadPerMinuteStats_NoSquadLoader_TeammatesEmpty(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main"} // squadLoader == nil
	allSquadRows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0, Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5, Assists: 2, TimePlayedSecs: 600},
	}
	got := svc.buildSquadPerMinuteStats(
		context.Background(),
		allSquadRows,
		"main",
		[]string{"friend1"},
		nil,
	)
	if len(got) != 2 {
		t.Fatalf("want 2 entries (main + friend1 placeholder), got %d", len(got))
	}
	for _, e := range got {
		if e.Player == "friend1" && (e.KillsPerMinute != 0 || e.MatchCount != 0) {
			t.Errorf("friend1 sans squadLoader : entry doit être vide, got %+v", e)
		}
	}
}

// ---------- buildSquadPerformanceSeries (teammates.XX) ----------

// TestBuildSquadPerformanceSeries_DistinctPointsPerPlayer cadenasse le bug
// bound-to-main : la série de chaque coéquipier affichait les stats du main
// (kills/deaths/assists identiques) car playerMatchesRepo ignorait l'arg gt.
func TestBuildSquadPerformanceSeries_DistinctPointsPerPlayer(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main":    {rowWithStatsXUID("xuid-main", "m1", t0, canonical.OutcomeWin, 10, 5, 2, 600, 50, 80)},
			"friend1": {rowWithStatsXUID("xuid-f1", "m1", t0, canonical.OutcomeWin, 3, 9, 7, 600, 35, 40)},
		},
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", squadLoader: loader}
	allSquadRows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0, Kills: 10, Deaths: 5},
		{MatchID: "m1", StartTime: t0, Kills: 3, Deaths: 9},
	}
	got := svc.buildSquadPerformanceSeries(context.Background(), allSquadRows, "main", "", []string{"friend1"}, nil)
	if len(got) != 2 {
		t.Fatalf("want 2 series (main + friend1), got %d", len(got))
	}
	mainSeries := got["main"]
	f1Series := got["friend1"]
	if len(mainSeries) == 0 || len(f1Series) == 0 {
		t.Fatalf("both series must be non-empty: main=%d friend1=%d", len(mainSeries), len(f1Series))
	}
	if mainSeries[0].Kills == f1Series[0].Kills {
		t.Errorf("kills identiques (main=%d friend1=%d) → bug bound-to-main",
			mainSeries[0].Kills, f1Series[0].Kills)
	}
}

// TestBuildSquadPerformanceSeries_NoSquadLoader_EmptyResult verifie que sans
// squadLoader, la fonction retourne nil sans panic (loadFor skips all gamertags).
func TestBuildSquadPerformanceSeries_NoSquadLoader_EmptyResult(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main"}
	allSquadRows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0},
		{MatchID: "m1", StartTime: t0},
	}
	got := svc.buildSquadPerformanceSeries(context.Background(), allSquadRows, "main", "", []string{"friend1"}, nil)
	if got != nil {
		t.Errorf("sans squadLoader : want nil, got %d entries", len(got))
	}
}

// TestBuildSquadPerformanceSeries_PopulatesEfficiencyFields vérifie que
// RendementOffensif et ResistanceDefensive sont calculés quand DamageDealt /
// DamageTaken sont disponibles dans le canonical row.
func TestBuildSquadPerformanceSeries_PopulatesEfficiencyFields(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	// 10 kills, 5 deaths, 3 assists, damageDealt=2300, damageTaken=1200
	// rendement = 225 × (10 + 3/3) / 2300 = 225×11/2300 ≈ 1.076 → round2 = 1.08
	// résistance = 1200 / (225×5) = 1200/1125 ≈ 1.067 → round2 = 1.07
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main":    {rowWithDamage("m1", t0, 10, 5, 3, 600, 0, 2300, 1200)},
			"friend1": {rowWithStatsXUID("xuid-f1", "m1", t0, canonical.OutcomeWin, 5, 5, 2, 600, 35, 40)},
		},
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", squadLoader: loader}
	allSquadRows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0},
	}
	got := svc.buildSquadPerformanceSeries(context.Background(), allSquadRows, "main", "", []string{"friend1"}, nil)
	if got == nil {
		t.Fatal("résultat nil inattendu")
	}
	pts := got["main"]
	if len(pts) == 0 {
		t.Fatal("série main vide")
	}
	pt := pts[0]
	if pt.RendementOffensif == nil {
		t.Error("RendementOffensif nil alors que DamageDealt disponible")
	} else if got, want := *pt.RendementOffensif, 1.08; got != want {
		t.Errorf("RendementOffensif = %.2f, want %.2f", got, want)
	}
	if pt.ResistanceDefensive == nil {
		t.Error("ResistanceDefensive nil alors que DamageTaken disponible")
	} else if got, want := *pt.ResistanceDefensive, 1.07; got != want {
		t.Errorf("ResistanceDefensive = %.2f, want %.2f", got, want)
	}
}

// TestBuildSquadPerformanceSeries_PopulatesKillTypeBreakdown vérifie que la
// ventilation par type de frag (mêlée / arme lourde / grenade) est extraite du
// canonical row vers le point de série — alimente les barres empilées
// « Répartition des frags » de la page Contributions. Les pointeurs restent nil
// pour un joueur sans ventilation (pas de fallback 0).
func TestBuildSquadPerformanceSeries_PopulatesKillTypeBreakdown(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	mainRow := rowWithStatsXUID("xuid-main", "m1", t0, canonical.OutcomeWin, 10, 5, 2, 600, 50, 80)
	melee, pw, gren := 2, 1, 3
	mainRow.Self.MeleeKills = &melee
	mainRow.Self.PowerWeaponKills = &pw
	mainRow.Self.GrenadeKills = &gren
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main":    {mainRow},
			"friend1": {rowWithStatsXUID("xuid-f1", "m1", t0, canonical.OutcomeWin, 5, 5, 2, 600, 35, 40)},
		},
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", squadLoader: loader}
	allSquadRows := []domain.SquadMatchRow{{MatchID: "m1", StartTime: t0}}
	got := svc.buildSquadPerformanceSeries(context.Background(), allSquadRows, "main", "", []string{"friend1"}, nil)
	if got == nil {
		t.Fatal("résultat nil inattendu")
	}
	pts := got["main"]
	if len(pts) == 0 {
		t.Fatal("série main vide")
	}
	pt := pts[0]
	if pt.MeleeKills == nil || *pt.MeleeKills != 2 {
		t.Errorf("MeleeKills = %v, want 2", pt.MeleeKills)
	}
	if pt.PowerWeaponKills == nil || *pt.PowerWeaponKills != 1 {
		t.Errorf("PowerWeaponKills = %v, want 1", pt.PowerWeaponKills)
	}
	if pt.GrenadeKills == nil || *pt.GrenadeKills != 3 {
		t.Errorf("GrenadeKills = %v, want 3", pt.GrenadeKills)
	}
	f1 := got["friend1"]
	if len(f1) == 0 {
		t.Fatal("série friend1 vide")
	}
	if f1[0].MeleeKills != nil {
		t.Errorf("friend1 MeleeKills = %v, want nil (non renseigné)", *f1[0].MeleeKills)
	}
}

// ---------- buildSquadSynergyRadar (teammates.11) ----------

// TestBuildSquadSynergyRadar_DistinctAxesPerPlayer cadenasse le bug bound-to-main :
// combat/survival/support affichaient les mêmes valeurs pour tout le squad car
// makeMateRaw chargeait les rows du main pour chaque gamertag.
func TestBuildSquadSynergyRadar_DistinctAxesPerPlayer(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	// Main : 20 kills (combat élevé), friend1 : 2 kills (combat bas).
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main":    {rowWithStatsXUID("xuid-main", "m1", t0, canonical.OutcomeWin, 20, 3, 5, 600, 60, 80)},
			"friend1": {rowWithStatsXUID("xuid-f1", "m1", t0, canonical.OutcomeWin, 2, 3, 15, 600, 35, 40)},
		},
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", squadLoader: loader}
	// 1 match présent 1 fois >= len(selectedGamertags)=1 → sharedMatches ok.
	allSquadRows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0, Kills: 20, Deaths: 3, Assists: 5, TimePlayedSecs: 600},
	}
	got := svc.buildSquadSynergyRadar(context.Background(), allSquadRows, "main", []string{"friend1"})
	if len(got) != 2 {
		t.Fatalf("want 2 series (main + friend1), got %d", len(got))
	}
	byPlayer := make(map[string]domain.SquadSynergyRadarSeries, 2)
	for _, s := range got {
		byPlayer[s.Player] = s
	}
	axisVal := func(s domain.SquadSynergyRadarSeries, axis string) float64 {
		for _, a := range s.Axes {
			if a.Axis == axis {
				return a.Raw
			}
		}
		return 0
	}
	mainCombat := axisVal(byPlayer["main"], "combat")
	f1Combat := axisVal(byPlayer["friend1"], "combat")
	if mainCombat == f1Combat {
		t.Errorf("axis combat identique (%.2f) pour main et friend1 → bug bound-to-main", mainCombat)
	}
	// Main a largement plus de kills → combat plus élevé.
	if mainCombat <= f1Combat {
		t.Errorf("main combat (%.2f) devrait être > friend1 (%.2f)", mainCombat, f1Combat)
	}
}

// ---------- buildSquadIntensityProfile xuid resolution ----------

// TestBuildSquadIntensityProfile_NoSquadLoader_NoPanic verifie que sans squadLoader
// la résolution xuid ne tente pas d'appeler playerMatchesRepo (nil → pas de panic),
// et la section est soit nil (événements absent) soit présente sans crash.
func TestBuildSquadIntensityProfile_NoSquadLoader_NoPanic(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	repo := &mockSquadRepo{impactRows: nil}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", repo: repo}
	allSquadRows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0},
		{MatchID: "m2", StartTime: t0.Add(time.Hour)},
		{MatchID: "m3", StartTime: t0.Add(2 * time.Hour)},
	}
	// Pas de panic attendu — retourne nil car pas d'events.
	got := svc.buildSquadIntensityProfile(context.Background(), allSquadRows, "main", []string{"friend1"}, "all")
	if got != nil {
		t.Errorf("sans events : want nil, got profile avec %d options", len(got.Options))
	}
}

// ---------- buildSquadFirstEvents T0 (teammates.17, §4.A-bis) ----------

// TestBuildSquadFirstEvents_AppliesT0Shift cadenasse le fix §4.A-bis : le chart
// premier frag / première mort doit retrancher le countdown pré-match (T0).
// Un kill à 50s de film, T0=28s → 22s de gameplay → bin plus précoce qu'en
// chronologie brute (50s).
func TestBuildSquadFirstEvents_AppliesT0Shift(t *testing.T) {
	start := time.Date(2026, 4, 6, 18, 0, 0, 0, time.UTC)
	mainXUID := "x_main"
	repo := &mockSquadRepo{
		impactRows: []domain.ImpactEventRow{
			{MatchID: "m1", XUID: mainXUID, EventType: analysis.EventTypeKill, TimeMS: 50000},
		},
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", repo: repo}

	t0ms := int64(28000)
	withT0 := []domain.SquadMatchRow{{MatchID: "m1", StartTime: start, DurationSeconds: 600, T0Ms: &t0ms}}
	noT0 := []domain.SquadMatchRow{{MatchID: "m1", StartTime: start, DurationSeconds: 600}} // T0Ms nil

	got := svc.buildSquadFirstEvents(context.Background(), withT0, "main", mainXUID, nil)
	raw := svc.buildSquadFirstEvents(context.Background(), noT0, "main", mainXUID, nil)
	if got == nil || raw == nil {
		t.Fatal("résultats non nil attendus (1 kill du main)")
	}

	binIdxLabel := func(r *domain.SquadFirstEvents) (int, string) {
		for _, row := range r.Rows {
			if row.Player != "main" {
				continue
			}
			for i, c := range row.KillCounts {
				if c > 0 {
					return i, r.BinLabels[i]
				}
			}
		}
		return -1, ""
	}
	gotIdx, gotLabel := binIdxLabel(got)
	rawIdx, rawLabel := binIdxLabel(raw)
	if gotIdx < 0 || rawIdx < 0 {
		t.Fatalf("kill du main introuvable : gotIdx=%d rawIdx=%d", gotIdx, rawIdx)
	}
	if gotIdx >= rawIdx {
		t.Errorf("T0 doit ramener le kill à un bin plus précoce : avec T0 idx=%d (%s), brut idx=%d (%s)",
			gotIdx, gotLabel, rawIdx, rawLabel)
	}
	// Valeurs précises : 22s gameplay → bin "30s" ; 50s film → bin "1m00s".
	if gotLabel != "30s" {
		t.Errorf("avec T0 (22s gameplay) : label want \"30s\", got %q", gotLabel)
	}
	if rawLabel != "1m00s" {
		t.Errorf("brut (50s film) : label want \"1m00s\", got %q", rawLabel)
	}
}

// TestBuildSquadFirstEvents_SkipsPreGameplayEvents : un event situé dans le
// countdown (TimeMS < T0) → temps corrigé négatif, doit être ignoré. Sinon il
// polluerait le sentinel -1 de firstKillS et masquerait le vrai premier frag
// (qui serait alors perdu → chart nil).
func TestBuildSquadFirstEvents_SkipsPreGameplayEvents(t *testing.T) {
	start := time.Date(2026, 4, 6, 18, 0, 0, 0, time.UTC)
	mainXUID := "x_main"
	repo := &mockSquadRepo{
		impactRows: []domain.ImpactEventRow{
			{MatchID: "m1", XUID: mainXUID, EventType: analysis.EventTypeKill, TimeMS: 10000}, // countdown → -18s
			{MatchID: "m1", XUID: mainXUID, EventType: analysis.EventTypeKill, TimeMS: 40000}, // gameplay → 12s
		},
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", repo: repo}
	t0ms := int64(28000)
	rows := []domain.SquadMatchRow{{MatchID: "m1", StartTime: start, DurationSeconds: 600, T0Ms: &t0ms}}

	got := svc.buildSquadFirstEvents(context.Background(), rows, "main", mainXUID, nil)
	if got == nil {
		t.Fatal("résultat non nil attendu : le frag à 40s (gameplay 12s) doit subsister")
	}
	total, firstBin := 0, -1
	for _, row := range got.Rows {
		if row.Player != "main" {
			continue
		}
		for i, c := range row.KillCounts {
			total += c
			if c > 0 && firstBin == -1 {
				firstBin = i
			}
		}
	}
	if total != 1 {
		t.Errorf("first_kill count want 1 (event countdown ignoré), got %d", total)
	}
	if firstBin != 0 || got.BinLabels[0] != "15s" {
		t.Errorf("first kill à 12s → bin0 \"15s\", got bin=%d labels=%v", firstBin, got.BinLabels)
	}
}

// TestBuildSquadIntensityProfile_AppliesT0AndSkipsCountdown (extension §4.A-bis,
// teammates.13) : le profil d'intensité doit retrancher le T0 et EXCLURE les
// events du countdown (TimeMS<0 après correction). Un kill countdown + un kill
// gameplay sur le même match → 1 seul bucket non nul (le gameplay), pas 2.
func TestBuildSquadIntensityProfile_AppliesT0AndSkipsCountdown(t *testing.T) {
	base := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	mainXUID := "x_main"
	repo := &mockSquadRepo{
		impactRows: []domain.ImpactEventRow{
			{MatchID: "m1", XUID: mainXUID, EventType: analysis.EventTypeKill, TimeMS: 10000},  // countdown → -18s → exclu
			{MatchID: "m1", XUID: mainXUID, EventType: analysis.EventTypeKill, TimeMS: 300000}, // gameplay → 272s
		},
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", repo: repo}
	t0ms := int64(28000)
	// >= intensityMinMatches (3) matchs ; events seulement sur m1.
	rows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: base, DurationSeconds: 600, T0Ms: &t0ms},
		{MatchID: "m2", StartTime: base.Add(time.Hour), DurationSeconds: 600, T0Ms: &t0ms},
		{MatchID: "m3", StartTime: base.Add(2 * time.Hour), DurationSeconds: 600, T0Ms: &t0ms},
	}
	got := svc.buildSquadIntensityProfile(context.Background(), rows, "main", nil, "Tous")
	if got == nil {
		t.Fatal("profil non nil attendu (kill gameplay présent sur m1)")
	}
	allRows := got.Rows["all"]
	var m1 *domain.SquadIntensityMatchRow
	for i := range allRows {
		if allRows[i].MatchID == "m1" {
			m1 = &allRows[i]
		}
	}
	if m1 == nil {
		t.Fatal("ligne m1 introuvable dans le toggle \"all\"")
	}
	nonZero := 0
	for _, p := range m1.Phases {
		if p > 0 {
			nonZero++
		}
	}
	if nonZero != 1 {
		t.Errorf("m1 : 1 seul bucket non nul attendu (kill countdown exclu), got %d (phases=%v)", nonZero, m1.Phases)
	}
}

// ---------- impactScoreWeights (teammates.07) ----------

func TestImpactScoreWeights_Coverage(t *testing.T) {
	for _, badge := range impactBadgeOrd {
		if _, ok := impactScoreWeights[badge]; !ok {
			t.Errorf("badge %q manque dans impactScoreWeights", badge)
		}
	}
	// Sanity : les weights matchent la spec (cf. .ai/charts_specs/teammates/07).
	expected := map[string]float64{
		"clutch_finisher":   2.0,
		"first_blood":       2.0,
		"last_casualty":     -2.0,
		"silent_hero":       1.5,
		"false_brother":     -1.5,
		"last_group_kill":   -1.0,
		"first_group_death": -1.0,
		"kamikaze":          -1.0,
		"top_killer":        1.0,
	}
	for k, v := range expected {
		if impactScoreWeights[k] != v {
			t.Errorf("%s: weight=%v, want %v", k, impactScoreWeights[k], v)
		}
	}
}

// ---------- buildSquadMapHeatmap (teammates.03) ----------

// rowWithMap est un helper local (pas de collision avec rowWithStats) qui
// construit une PlayerMatchRow canonique avec une map et un perf score.
func rowWithMap(matchID, mapName string, perf float64) canonical.PlayerMatchRow {
	p := perf
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID: matchID,
			Map: &canonical.AssetReference{
				Kind:         "map",
				ID:           mapName,
				DefaultLabel: mapName,
			},
		},
		Enrichment: canonical.PlayerMatchEnrichment{
			PerformanceScore: &p,
		},
	}
}

// TestBuildSquadMapHeatmap_TeammatePerfsAreDistinctFromMain cadenasse le bug
// "bound-to-main" du PlayerMatchesAdapter : si buildSquadMapHeatmap chargeait
// les rows des coéquipiers via playerMatchesRepo (qui ignore l'arg gamertag),
// chaque ligne du heatmap aurait les mêmes perfs que le main player. Le fix
// (commit fabc3b3c) bascule sur squadLoader.LoadFor qui résout la PlayerDB
// par gamertag.
func TestBuildSquadMapHeatmap_TeammatePerfsAreDistinctFromMain(t *testing.T) {
	mainPerf := 80.0
	teammatePerf := 50.0
	streets := "Streets"
	mainSquadRows := []domain.SquadMatchRow{
		{MatchID: "m1", PerformanceScore: &mainPerf, MapUI: streets, IsWithFriends: true},
	}
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"friend1": {rowWithMap("m1", streets, teammatePerf)},
		},
	}
	svc := &TeammatesService{
		squadLoader: loader,
		titleSlug:   "halo_infinite",
		gamertag:    "main",
	}

	heatmap := svc.buildSquadMapHeatmap(context.Background(), mainSquadRows, []string{"friend1"}, nil)
	if heatmap == nil {
		t.Fatal("heatmap should be non-nil")
	}
	if len(heatmap.Players) != 2 || heatmap.Players[0] != "main" || heatmap.Players[1] != "friend1" {
		t.Fatalf("Players want [main, friend1], got %v", heatmap.Players)
	}

	// Cell main + Streets : perf=80
	// Cell friend1 + Streets : perf=50 (PAS 80 — bug bound-to-main)
	var mainCell, friendCell *domain.SquadMapHeatmapCell
	for i := range heatmap.Cells {
		c := &heatmap.Cells[i]
		if c.MapUI != streets {
			continue
		}
		switch c.Player {
		case "main":
			mainCell = c
		case "friend1":
			friendCell = c
		}
	}
	if mainCell == nil || friendCell == nil {
		t.Fatalf("cells main+friend1 expected, got %+v", heatmap.Cells)
	}
	if mainCell.PerfAvg == nil || friendCell.PerfAvg == nil {
		t.Fatalf("PerfAvg should be set for both: main=%v friend=%v", mainCell.PerfAvg, friendCell.PerfAvg)
	}
	if *mainCell.PerfAvg != mainPerf {
		t.Errorf("main PerfAvg: want %v, got %v", mainPerf, *mainCell.PerfAvg)
	}
	if *friendCell.PerfAvg != teammatePerf {
		t.Errorf("friend1 PerfAvg: want %v (sa propre perf), got %v — bug bound-to-main probable",
			teammatePerf, *friendCell.PerfAvg)
	}
	if *mainCell.PerfAvg == *friendCell.PerfAvg {
		t.Errorf("main et friend1 ont des perfs identiques (%v) → heatmap retourne les rows du main pour tous", *mainCell.PerfAvg)
	}
}

// TestBuildSquadImpactMatrix_TeamWideAllyDropped : le scoreboard impact est
// calculé en team-wide alliée (parité Python compute_single_match_impact).
// Si un badge (ex. top_killer) tombe sur un allié NON-squad, il ne doit PAS
// apparaître dans le scoreboard ni se reporter sur un squad member.
//
// Setup : main + A (squad). NS = allié non-squad. NS a 12 kills (top killer
// global de l'équipe alliée), main a 5, A a 4. Le badge top_killer va donc à
// NS et doit être droppé. silent_hero (max_assists+min_deaths) tombe sur A
// (max_assists=8, min_deaths=0) → doit apparaître dans le scoreboard.
func TestBuildSquadImpactMatrix_TeamWideAllyDropped(t *testing.T) {
	mainXUID := "x_main"
	matchID := "m_session_1"
	startTime := time.Date(2026, 4, 6, 18, 0, 0, 0, time.UTC)

	// Mock repo : retourne 3 alliés (main, A=squad, NS=non-squad). NS a le
	// max kills donc top_killer. A a max assists + min deaths donc silent_hero.
	repo := &mockSquadRepo{
		allyRows: []domain.AllyParticipant{
			{MatchID: matchID, XUID: mainXUID, Gamertag: "main", Kills: 5, Deaths: 2, Assists: 3, Outcome: domain.OutcomeWin},
			{MatchID: matchID, XUID: "x_a", Gamertag: "A", Kills: 4, Deaths: 0, Assists: 8, Outcome: domain.OutcomeWin},
			{MatchID: matchID, XUID: "x_ns", Gamertag: "NS", Kills: 12, Deaths: 1, Assists: 1, Outcome: domain.OutcomeWin},
		},
	}
	svc := &TeammatesService{
		repo:      repo,
		titleSlug: "halo_infinite",
		gamertag:  "main",
	}

	allSquadRows := []domain.SquadMatchRow{
		{MatchID: matchID, StartTime: startTime, Outcome: domain.OutcomeWin},
	}
	matrix := svc.buildSquadImpactMatrix(context.Background(), allSquadRows, mainXUID, "main", []string{"A"})
	if matrix == nil {
		t.Fatal("matrix should be non-nil")
	}

	// top_killer va sur NS (non-squad) → ne doit apparaître nulle part.
	for _, c := range matrix.Cells {
		for _, k := range c.BadgeKeys {
			if k == "top_killer" {
				t.Errorf("top_killer ne doit pas apparaître (tombe sur NS, allié non-squad) — trouvé sur %s", c.Player)
			}
		}
	}
	// Aucun joueur ne doit avoir un compte top_killer non nul dans la summary.
	for _, p := range matrix.Players {
		for _, b := range p.Counts {
			if b.BadgeKey == "top_killer" && b.Count != 0 {
				t.Errorf("player %s : top_killer count=%d, want 0 (badge tombé sur non-squad)", p.Player, b.Count)
			}
		}
	}

	// silent_hero doit aller à A (max assists=8, min deaths=0 sur l'équipe alliée).
	foundSilentOnA := false
	for _, c := range matrix.Cells {
		if c.Player != "A" {
			continue
		}
		for _, k := range c.BadgeKeys {
			if k == "silent_hero" {
				foundSilentOnA = true
			}
		}
	}
	if !foundSilentOnA {
		t.Error("silent_hero attendu sur A (max assists + min deaths parmi alliés)")
	}
}

// TestBuildSquadImpactMatrix_TeamWideAllySwapsBadge : si un allié non-squad
// est plus extrême que le squad member, c'est lui qui reçoit le badge en
// team-wide → le squad member NE DOIT PAS recevoir le badge "à défaut".
//
// Setup défaite : main perd, A (squad) a 5 deaths, NS (non-squad) a 9 deaths.
// false_brother en team-wide va sur NS, donc A ne le reçoit pas.
func TestBuildSquadImpactMatrix_TeamWideNoFallback(t *testing.T) {
	mainXUID := "x_main"
	matchID := "m_session_2"
	startTime := time.Date(2026, 4, 6, 19, 0, 0, 0, time.UTC)

	repo := &mockSquadRepo{
		allyRows: []domain.AllyParticipant{
			{MatchID: matchID, XUID: mainXUID, Gamertag: "main", Kills: 4, Deaths: 3, Assists: 1, Outcome: domain.OutcomeLoss},
			{MatchID: matchID, XUID: "x_a", Gamertag: "A", Kills: 3, Deaths: 5, Assists: 2, Outcome: domain.OutcomeLoss},
			{MatchID: matchID, XUID: "x_ns", Gamertag: "NS", Kills: 2, Deaths: 9, Assists: 0, Outcome: domain.OutcomeLoss},
		},
	}
	svc := &TeammatesService{
		repo:      repo,
		titleSlug: "halo_infinite",
		gamertag:  "main",
	}
	allSquadRows := []domain.SquadMatchRow{
		{MatchID: matchID, StartTime: startTime, Outcome: domain.OutcomeLoss},
	}
	matrix := svc.buildSquadImpactMatrix(context.Background(), allSquadRows, mainXUID, "main", []string{"A"})

	// false_brother doit aller à NS (max deaths=9, min assists=0). Donc :
	// - A ne doit PAS recevoir false_brother malgré ses 5 deaths (squad-only,
	//   il l'aurait reçu en logique squad-only buggée).
	// - matrix peut être nil si AUCUN badge ne tombe sur un squad member.
	if matrix == nil {
		// OK : aucun badge alloué à un squad member, matrice vide donc nil.
		return
	}
	for _, c := range matrix.Cells {
		for _, k := range c.BadgeKeys {
			if k == "false_brother" {
				t.Errorf("false_brother ne doit pas apparaître (tombe sur NS en team-wide) — trouvé sur %s", c.Player)
			}
		}
	}
}

// TestBuildSquadMapHeatmap_NoCapOnMaps : le cap top 15 a été retiré (commit
// fabc3b3c). 18 cartes en input → 18 cartes en output (toutes affichées).
func TestBuildSquadMapHeatmap_NoCapOnMaps(t *testing.T) {
	rows := make([]domain.SquadMatchRow, 0, 18)
	perf := 60.0
	for i := 0; i < 18; i++ {
		rows = append(rows, domain.SquadMatchRow{
			MatchID:          string(rune('a' + i)),
			PerformanceScore: &perf,
			MapUI:            string(rune('A'+i)) + "_map",
		})
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main"}
	heatmap := svc.buildSquadMapHeatmap(context.Background(), rows, nil, nil)
	if heatmap == nil {
		t.Fatal("heatmap should be non-nil")
	}
	if len(heatmap.MapsTopN) != 18 {
		t.Errorf("toutes les cartes attendues (18), got %d → cap top 15 réintroduit ?", len(heatmap.MapsTopN))
	}
}

// TestBuildSquadMapHeatmap_MapsOrderedByFirstAppearance verrouille le tri des
// cartes par ordre CHRONOLOGIQUE de première apparition (verbatim utilisateur :
// « pas dans l'ordre et je veux pas de regroupement »). "Early" (1 match) est
// jouée avant "Mid" (2 matchs) avant "Late" (3 matchs) : l'ordre fréquence serait
// l'inverse (Late>Mid>Early), donc ce test distingue les deux tris.
func TestBuildSquadMapHeatmap_MapsOrderedByFirstAppearance(t *testing.T) {
	base := time.Date(2026, 5, 1, 18, 0, 0, 0, time.UTC)
	perf := 60.0
	rows := []domain.SquadMatchRow{
		{MatchID: "e1", StartTime: base, MapUI: "Early", PerformanceScore: &perf},
		{MatchID: "m1", StartTime: base.Add(1 * time.Hour), MapUI: "Mid", PerformanceScore: &perf},
		{MatchID: "m2", StartTime: base.Add(90 * time.Minute), MapUI: "Mid", PerformanceScore: &perf},
		{MatchID: "l1", StartTime: base.Add(2 * time.Hour), MapUI: "Late", PerformanceScore: &perf},
		{MatchID: "l2", StartTime: base.Add(150 * time.Minute), MapUI: "Late", PerformanceScore: &perf},
		{MatchID: "l3", StartTime: base.Add(3 * time.Hour), MapUI: "Late", PerformanceScore: &perf},
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main"}
	heatmap := svc.buildSquadMapHeatmap(context.Background(), rows, nil, nil)
	if heatmap == nil {
		t.Fatal("heatmap should be non-nil")
	}
	want := []string{"Early", "Mid", "Late"}
	if len(heatmap.MapsTopN) != len(want) {
		t.Fatalf("want %d cartes, got %d (%v)", len(want), len(heatmap.MapsTopN), heatmap.MapsTopN)
	}
	for i, w := range want {
		if heatmap.MapsTopN[i] != w {
			t.Errorf("MapsTopN[%d]: want %q, got %q (ordre chronologique cassé: %v)",
				i, w, heatmap.MapsTopN[i], heatmap.MapsTopN)
		}
	}
}

// ---------- SynergyOffensiveConversion / SynergyDefensiveResistance ----------

func TestSynergyOffensiveConversion_ZeroDamage(t *testing.T) {
	if v := SynergyOffensiveConversion(10, 5, 0, 225); v != 0 {
		t.Errorf("zero damage → want 0, got %v", v)
	}
}

func TestSynergyOffensiveConversion_TypicalMatch(t *testing.T) {
	// 225 × (10 + 5/3) / 2500 ≈ 1.05
	high := SynergyOffensiveConversion(10, 5, 2500, 225)
	low := SynergyOffensiveConversion(3, 1, 2500, 225)
	if high <= 0 {
		t.Errorf("typical OC: want > 0, got %v", high)
	}
	if high <= low {
		t.Errorf("more kills → higher OC: high=%v should > low=%v", high, low)
	}
}

func TestSynergyDefensiveResistance_ZeroBoth(t *testing.T) {
	if v := SynergyDefensiveResistance(0, 0, 225); v != 0 {
		t.Errorf("zero both → want 0, got %v", v)
	}
}

func TestSynergyDefensiveResistance_ZeroDeaths(t *testing.T) {
	// 0 mort avec damage pris → score parfait (au-delà du P80)
	v := SynergyDefensiveResistance(1000, 0, 225)
	if v <= 0 {
		t.Errorf("zero deaths + damage → want > 0 (parfait), got %v", v)
	}
}

func TestSynergyDefensiveResistance_TypicalMatch(t *testing.T) {
	// 1000 DT / (225 × 4) ≈ 1.11
	v := SynergyDefensiveResistance(1000, 4, 225)
	if v <= 0.5 || v > 3.0 {
		t.Errorf("typical DR: want in (0.5, 3.0], got %v", v)
	}
}

// ---------- Radar synergie — axe Impact (rendement offensif) ----------

// rowWithDamage construit une PlayerMatchRow avec PersonalScore, DamageDealt et DamageTaken.
func rowWithDamage(matchID string, ts time.Time, kills, deaths, assists, timePlayed, personalScore, damageDealt, damageTaken int) canonical.PlayerMatchRow {
	r := rowWithPersonalScore(matchID, ts, kills, deaths, assists, timePlayed, personalScore)
	dd, dt := damageDealt, damageTaken
	r.Self.DamageDealt = &dd
	r.Self.DamageTaken = &dt
	return r
}

// TestBuildSquadSynergyRadar_ImpactIsOffensiveConversion vérifie que l'axe impact
// est calculé comme rendement offensif 225×(K+A/3)/DD et non comme pts/min.
func TestBuildSquadSynergyRadar_ImpactIsOffensiveConversion(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	// main : 10K, 5A, DD=2500 → OC = 225*(10+5/3)/2500 ≈ 1.05
	// friend1 : 3K, 1A, DD=3000 → OC = 225*(3+1/3)/3000 ≈ 0.25
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main":    {rowWithDamage("m1", t0, 10, 3, 5, 600, 3000, 2500, 1000)},
			"friend1": {rowWithDamage("m1", t0, 3, 3, 1, 600, 1500, 3000, 1500)},
		},
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", squadLoader: loader}
	allRows := []domain.SquadMatchRow{{MatchID: "m1", StartTime: t0, Kills: 10, Deaths: 3}}

	got := svc.buildSquadSynergyRadar(context.Background(), allRows, "main", []string{"friend1"})
	if len(got) != 2 {
		t.Fatalf("want 2 series, got %d", len(got))
	}
	axisRaw := func(s domain.SquadSynergyRadarSeries, axis string) float64 {
		for _, a := range s.Axes {
			if a.Axis == axis {
				return a.Raw
			}
		}
		return -1
	}
	byPlayer := map[string]domain.SquadSynergyRadarSeries{}
	for _, s := range got {
		byPlayer[s.Player] = s
	}
	mainImpact := axisRaw(byPlayer["main"], "impact")
	f1Impact := axisRaw(byPlayer["friend1"], "impact")
	if mainImpact <= 0 {
		t.Errorf("main impact devrait être > 0 (OC > 0), got %v", mainImpact)
	}
	if mainImpact <= f1Impact {
		t.Errorf("main OC supérieur → impact devrait être > friend1: main=%v f1=%v", mainImpact, f1Impact)
	}
}

// TestBuildSquadSynergyRadar_SurvivalIsDefensiveResistance vérifie que l'axe survival
// est calculé comme résistance défensive ΣDT/(225×ΣD).
func TestBuildSquadSynergyRadar_SurvivalIsDefensiveResistance(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	// main : DT=2000, 5 deaths → DR = 2000/(225×5) ≈ 1.78 (haut)
	// friend1 : DT=3000, 10 deaths → DR = 3000/(225×10) ≈ 1.33 (bas)
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main":    {rowWithDamage("m1", t0, 8, 5, 3, 600, 2500, 2000, 2000)},
			"friend1": {rowWithDamage("m1", t0, 6, 10, 4, 600, 2000, 2500, 3000)},
		},
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", squadLoader: loader}
	allRows := []domain.SquadMatchRow{{MatchID: "m1", StartTime: t0, Kills: 8, Deaths: 5}}

	got := svc.buildSquadSynergyRadar(context.Background(), allRows, "main", []string{"friend1"})
	if len(got) != 2 {
		t.Fatalf("want 2 series, got %d", len(got))
	}
	axisRaw := func(s domain.SquadSynergyRadarSeries, axis string) float64 {
		for _, a := range s.Axes {
			if a.Axis == axis {
				return a.Raw
			}
		}
		return -1
	}
	byPlayer := map[string]domain.SquadSynergyRadarSeries{}
	for _, s := range got {
		byPlayer[s.Player] = s
	}
	mainSurvival := axisRaw(byPlayer["main"], "survival")
	f1Survival := axisRaw(byPlayer["friend1"], "survival")
	if mainSurvival <= 0 {
		t.Errorf("main survival devrait être > 0 (DR > 0), got %v", mainSurvival)
	}
	if mainSurvival <= f1Survival {
		t.Errorf("main DR supérieur → survival devrait être > friend1: main=%v f1=%v", mainSurvival, f1Survival)
	}
}

// ---------- Radar synergie — axe Objective via PSA ----------

// TestBuildSquadSynergyRadar_ObjectiveSumsFromPSA vérifie que l'axe objective
// est alimenté depuis LoadObjectiveScores (scores PSA catégorie "objective"),
// et non toujours à 0.
func TestBuildSquadSynergyRadar_ObjectiveSumsFromPSA(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main":    {rowWithPersonalScore("m1", t0, 5, 3, 2, 600, 2000)},
			"friend1": {rowWithPersonalScore("m1", t0, 3, 4, 8, 600, 1500)},
		},
		// main a des pts objectif sur m1, friend1 n'en a pas.
		objByGT: map[string]map[string]int{
			"main": {"m1": 700},
		},
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", squadLoader: loader}
	allRows := []domain.SquadMatchRow{{MatchID: "m1", StartTime: t0, Kills: 5, Deaths: 3}}

	got := svc.buildSquadSynergyRadar(context.Background(), allRows, "main", []string{"friend1"})
	if len(got) != 2 {
		t.Fatalf("want 2 series, got %d", len(got))
	}
	axisRaw := func(s domain.SquadSynergyRadarSeries, axis string) float64 {
		for _, a := range s.Axes {
			if a.Axis == axis {
				return a.Raw
			}
		}
		return -1
	}
	byPlayer := map[string]domain.SquadSynergyRadarSeries{}
	for _, s := range got {
		byPlayer[s.Player] = s
	}
	mainObj := axisRaw(byPlayer["main"], "objective")
	f1Obj := axisRaw(byPlayer["friend1"], "objective")
	if mainObj != 700 {
		t.Errorf("main objective raw: want 700, got %v", mainObj)
	}
	if f1Obj != 0 {
		t.Errorf("friend1 objective raw: want 0 (aucun PSA), got %v", f1Obj)
	}
}

// ---------- Radar synergie — axe Score (score personnel par minute) ----------

// TestBuildSquadSynergyRadar_ScoreIsPersonalScorePerMinute vérifie que l'axe score
// est le score personnel par minute jouée (ΣPS / (Σtime_played/60)) : vivant et
// différenciant entre joueurs, et non plus le résiduel medals/streaks ≈ 0.
func TestBuildSquadSynergyRadar_ScoreIsPersonalScorePerMinute(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	// main : PS=2000 sur 600 s → 200/min ; friend1 : PS=1000 sur 600 s → 100/min.
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main":    {rowWithPersonalScore("m1", t0, 5, 3, 2, 600, 2000)},
			"friend1": {rowWithPersonalScore("m1", t0, 5, 3, 2, 600, 1000)},
		},
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", squadLoader: loader}
	allRows := []domain.SquadMatchRow{{MatchID: "m1", StartTime: t0, Kills: 5, Deaths: 3}}

	got := svc.buildSquadSynergyRadar(context.Background(), allRows, "main", []string{"friend1"})
	if len(got) != 2 {
		t.Fatalf("want 2 series, got %d", len(got))
	}
	axis := func(s domain.SquadSynergyRadarSeries, name string) domain.SquadSynergyRadarAxis {
		for _, a := range s.Axes {
			if a.Axis == name {
				return a
			}
		}
		return domain.SquadSynergyRadarAxis{Raw: -1, Value: -1}
	}
	byPlayer := map[string]domain.SquadSynergyRadarSeries{}
	for _, s := range got {
		byPlayer[s.Player] = s
	}
	mainScore := axis(byPlayer["main"], "score")
	f1Score := axis(byPlayer["friend1"], "score")
	if mainScore.Raw != 200 {
		t.Errorf("main score raw (PS/min): want 200, got %v", mainScore.Raw)
	}
	if f1Score.Raw != 100 {
		t.Errorf("friend1 score raw (PS/min): want 100, got %v", f1Score.Raw)
	}
	if mainScore.Value <= f1Score.Value {
		t.Errorf("main PS/min supérieur → score value devrait être > friend1: main=%v f1=%v",
			mainScore.Value, f1Score.Value)
	}
	if f1Score.Value <= 0 {
		t.Errorf("friend1 score value devrait être vivant (> 0), got %v", f1Score.Value)
	}
}

// ---------- Régression 851e10ef5 : input dédupliqué + 2 coéquipiers ----------
//
// Depuis intersectSquadRowsByMatchID, allSquadRows arrive dédupliqué (1 row par
// match). L'ancienne heuristique « occurrences >= len(selectedGamertags) »
// rendait synergy radar / performance series / medal digest TOUJOURS vides dès
// 2 coéquipiers sélectionnés (1 >= 2 faux). Ces tests cadenassent le fix.

func TestBuildSquadPerformanceSeries_DedupedInput_TwoTeammates(t *testing.T) {
	t0 := time.Date(2026, 6, 9, 19, 14, 0, 0, time.UTC)
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main":    {rowWithStatsXUID("xuid-main", "m1", t0, canonical.OutcomeWin, 10, 5, 2, 600, 50, 80)},
			"friend1": {rowWithStatsXUID("xuid-f1", "m1", t0, canonical.OutcomeWin, 3, 9, 7, 600, 35, 40)},
			"friend2": {rowWithStatsXUID("xuid-f2", "m1", t0, canonical.OutcomeWin, 6, 6, 1, 600, 45, 60)},
		},
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", squadLoader: loader}
	// 1 seule row par match : sortie d'intersectSquadRowsByMatchID.
	allSquadRows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0, Kills: 10, Deaths: 5},
	}
	got := svc.buildSquadPerformanceSeries(context.Background(), allSquadRows, "main", "", []string{"friend1", "friend2"}, nil)
	if len(got) != 3 {
		t.Fatalf("want 3 series (main + 2 friends), got %d — régression heuristique occurrences", len(got))
	}
	for _, gt := range []string{"main", "friend1", "friend2"} {
		if len(got[gt]) == 0 {
			t.Errorf("série %q vide alors que le match m1 est dans l'intersection", gt)
		}
	}
}

func TestBuildSquadSynergyRadar_DedupedInput_TwoTeammates(t *testing.T) {
	t0 := time.Date(2026, 6, 9, 19, 14, 0, 0, time.UTC)
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main":    {rowWithStatsXUID("xuid-main", "m1", t0, canonical.OutcomeWin, 10, 5, 2, 600, 50, 80)},
			"friend1": {rowWithStatsXUID("xuid-f1", "m1", t0, canonical.OutcomeWin, 3, 9, 7, 600, 35, 40)},
			"friend2": {rowWithStatsXUID("xuid-f2", "m1", t0, canonical.OutcomeWin, 6, 6, 1, 600, 45, 60)},
		},
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", squadLoader: loader}
	allRows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0, Kills: 10, Deaths: 5},
	}
	got := svc.buildSquadSynergyRadar(context.Background(), allRows, "main", []string{"friend1", "friend2"})
	if len(got) != 3 {
		t.Fatalf("want 3 series radar (main + 2 friends), got %d — régression heuristique occurrences", len(got))
	}
}

func TestCollectSharedMatchIDsForDigest_DedupedInput(t *testing.T) {
	t0 := time.Date(2026, 6, 9, 19, 14, 0, 0, time.UTC)
	rows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0},
		{MatchID: "m2", StartTime: t0},
		{MatchID: "m2", StartTime: t0}, // doublon résiduel toléré
	}
	got := collectSharedMatchIDsForDigest(rows)
	if len(got) != 2 {
		t.Fatalf("want 2 match_ids distincts, got %d (%v)", len(got), got)
	}
}
