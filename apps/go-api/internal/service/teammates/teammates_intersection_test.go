package teammates

// teammates_intersection_test.go — tests de la composition EXACTE (intersection)
// dans GetPage + sessions composition-aware. Couvre le bug "coéquipier ajouté à
// une session qu'il n'a pas jouée" : la page Escouade unionnait les matchs par
// coéquipier au lieu de les intersecter.

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// makeSquadRowSess enrichit makeSquadRow avec un SessionLabel + StartTime précis
// (pour tester le regroupement/tri des sessions de composition).
func makeSquadRowSess(matchID, mapUI string, outcome int, session string, ts time.Time) domain.SquadMatchRow {
	r := makeSquadRow(matchID, mapUI, outcome)
	r.SessionLabel = strPtr(session)
	r.StartTime = ts
	return r
}

// TestGetPage_TwoTeammates_OnlySharedMatchesInCharts : avec 2 coéquipiers, seuls
// les matchs joués par TOUS (intersection) alimentent les charts. C'est le
// scénario Chocoboflor/Madina : un match joué sans Madina ne doit pas survivre.
func TestGetPage_TwoTeammates_OnlySharedMatchesInCharts(t *testing.T) {
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "ta", Gamertag: "AllyA", GamesTogether: 10},
			{XUID: "tb", Gamertag: "AllyB", GamesTogether: 10},
		},
		squadRowsByTeammate: map[string][]domain.SquadMatchRow{
			// AllyA a joué m1,m2,m3 avec le main.
			"ta": {
				makeSquadRow("m1", "Bazaar", domain.OutcomeWin),
				makeSquadRow("m2", "Aquarius", domain.OutcomeWin),
				makeSquadRow("m3", "Recharge", domain.OutcomeLoss),
			},
			// AllyB a joué m2,m3,m4. Intersection = {m2, m3}.
			"tb": {
				makeSquadRow("m2", "Aquarius", domain.OutcomeWin),
				makeSquadRow("m3", "Recharge", domain.OutcomeLoss),
				makeSquadRow("m4", "Streets", domain.OutcomeWin),
			},
		},
	}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(
		newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test",
	)

	resp, err := svc.GetPage(context.Background(), "px", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"AllyA", "AllyB"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	maps := map[string]bool{}
	for _, row := range resp.MapBreakdown {
		maps[row.MapUI] = true
	}
	if len(resp.MapBreakdown) != 2 {
		t.Fatalf("expected 2 maps (intersection m2+m3), got %d (%v)", len(resp.MapBreakdown), maps)
	}
	if maps["Bazaar"] || maps["Streets"] {
		t.Errorf("matchs non partagés fuités dans le breakdown: %v", maps)
	}
	if !maps["Aquarius"] || !maps["Recharge"] {
		t.Errorf("matchs partagés manquants dans le breakdown: %v", maps)
	}
}

// TestGetPage_TwoTeammates_EmptyIntersection_NoChartsSoloHeader : si la
// composition exacte n'a jamais joué ensemble, aucun chart n'est produit, le
// header repasse en solo, et les sessions de composition sont vides.
func TestGetPage_TwoTeammates_EmptyIntersection_NoChartsSoloHeader(t *testing.T) {
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "ta", Gamertag: "AllyA", GamesTogether: 10},
			{XUID: "tb", Gamertag: "AllyB", GamesTogether: 10},
		},
		squadRowsByTeammate: map[string][]domain.SquadMatchRow{
			"ta": {makeSquadRow("m1", "Bazaar", domain.OutcomeWin)},
			"tb": {makeSquadRow("m2", "Aquarius", domain.OutcomeWin)},
		},
	}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(
		newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test",
	)

	resp, err := svc.GetPage(context.Background(), "px", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"AllyA", "AllyB"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.MapBreakdown) != 0 {
		t.Errorf("expected no map breakdown (intersection vide), got %d", len(resp.MapBreakdown))
	}
	if len(resp.CompositionSessions) != 0 {
		t.Errorf("expected no composition sessions, got %d", len(resp.CompositionSessions))
	}
	if resp.LatestCompositionSession != "" {
		t.Errorf("expected empty latest composition session, got %q", resp.LatestCompositionSession)
	}
}

// TestGetPage_TwoTeammates_CompositionSessions : les sessions exposées sont
// celles de l'intersection, triées DESC, et latest_composition_session pointe la
// plus récente.
func TestGetPage_TwoTeammates_CompositionSessions(t *testing.T) {
	tOld := time.Date(2026, 6, 1, 20, 0, 0, 0, time.UTC)
	tNew := time.Date(2026, 6, 8, 20, 0, 0, 0, time.UTC)
	shared := []domain.SquadMatchRow{
		makeSquadRowSess("m1", "Bazaar", domain.OutcomeWin, "S_old", tOld),
		makeSquadRowSess("m2", "Aquarius", domain.OutcomeWin, "S_new", tNew),
	}
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "ta", Gamertag: "AllyA", GamesTogether: 10},
			{XUID: "tb", Gamertag: "AllyB", GamesTogether: 10},
		},
		squadRowsByTeammate: map[string][]domain.SquadMatchRow{
			"ta": shared,
			"tb": shared,
		},
	}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(
		newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test",
	)

	resp, err := svc.GetPage(context.Background(), "px", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"AllyA", "AllyB"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.CompositionSessions) != 2 {
		t.Fatalf("expected 2 composition sessions, got %d", len(resp.CompositionSessions))
	}
	if resp.CompositionSessions[0].Label != "S_new" {
		t.Errorf("expected latest session S_new first (tri DESC), got %q", resp.CompositionSessions[0].Label)
	}
	if resp.LatestCompositionSession != "S_new" {
		t.Errorf("expected LatestCompositionSession=S_new, got %q", resp.LatestCompositionSession)
	}
}

// TestIntersectSquadRowsByMatchID couvre le helper directement : 0/1/N sets +
// dédup intra-set.
func TestIntersectSquadRowsByMatchID(t *testing.T) {
	rA := []domain.SquadMatchRow{
		makeSquadRow("m1", "Bazaar", domain.OutcomeWin),
		makeSquadRow("m2", "Aquarius", domain.OutcomeWin),
		makeSquadRow("m2", "Aquarius", domain.OutcomeWin), // doublon intra-set
		makeSquadRow("m3", "Recharge", domain.OutcomeLoss),
	}
	rB := []domain.SquadMatchRow{
		makeSquadRow("m2", "Aquarius", domain.OutcomeWin),
		makeSquadRow("m3", "Recharge", domain.OutcomeLoss),
		makeSquadRow("m4", "Streets", domain.OutcomeWin),
	}

	if got := intersectSquadRowsByMatchID(nil); got != nil {
		t.Errorf("0 set: expected nil, got %v", got)
	}

	one := intersectSquadRowsByMatchID([][]domain.SquadMatchRow{rA})
	if len(one) != 3 {
		t.Errorf("1 set: expected 3 (dédupliqué), got %d", len(one))
	}

	two := intersectSquadRowsByMatchID([][]domain.SquadMatchRow{rA, rB})
	ids := map[string]bool{}
	for _, r := range two {
		ids[r.MatchID] = true
	}
	if len(two) != 2 || !ids["m2"] || !ids["m3"] {
		t.Errorf("2 sets: expected {m2,m3}, got %v", ids)
	}
}

// TestBuildCompositionSessionLabels couvre le builder : regroupement par label,
// dédup par match_id, exclusion des labels vides/nil, bornes start/end, tri DESC.
func TestBuildCompositionSessionLabels(t *testing.T) {
	t1 := time.Date(2026, 6, 1, 19, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 1, 21, 30, 0, 0, time.UTC) // même session S_old, plus tard
	t3 := time.Date(2026, 6, 8, 20, 0, 0, 0, time.UTC)  // S_new, plus récent
	rows := []domain.SquadMatchRow{
		makeSquadRowSess("m1", "Bazaar", domain.OutcomeWin, "S_old", t1),
		makeSquadRowSess("m2", "Aquarius", domain.OutcomeWin, "S_old", t2),
		makeSquadRowSess("m2", "Aquarius", domain.OutcomeWin, "S_old", t2), // doublon match_id
		makeSquadRowSess("m3", "Recharge", domain.OutcomeLoss, "S_new", t3),
		makeSquadRow("m4", "Streets", domain.OutcomeWin), // SessionLabel nil → ignoré
	}

	out := buildCompositionSessionLabels(rows)
	if len(out) != 2 {
		t.Fatalf("expected 2 sessions (S_old, S_new), got %d", len(out))
	}
	// Tri DESC : S_new en tête.
	if out[0].Label != "S_new" || out[1].Label != "S_old" {
		t.Fatalf("expected [S_new, S_old] (tri DESC), got [%s, %s]", out[0].Label, out[1].Label)
	}
	// Bornes de S_old : started=t1, ended=t2 (dédup du doublon m2).
	if old := out[1]; !old.StartedAt.Equal(t1) || !old.EndedAt.Equal(t2) {
		t.Errorf("S_old bounds: expected [%v,%v], got [%v,%v]", t1, t2, old.StartedAt, old.EndedAt)
	}
}
