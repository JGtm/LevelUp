package teammates

// teammates_data_issues_test.go — les chargements best-effort dégradés doivent
// être REMONTÉS (domain.DataIssue) et non plus avalés dans un warn : sans ça, la
// page Escouade affiche des nombres amputés qu'aucun rejeu ne reproduit.
//
// Couvre aussi la population unique du heatmap (plus de re-filtrage privé).

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

func issueCodes(issues []domain.DataIssue) map[string]string {
	out := make(map[string]string, len(issues))
	for _, it := range issues {
		out[it.Code] = it.Detail
	}
	return out
}

// Option composition exacte activée MAIS chargement des équipes alliées en
// échec : la page reste servie sur la population « ensemble » (dégradation
// gracieuse) et le dit explicitement.
func TestGetPage_DataIssue_MainTeamParticipantsFailure(t *testing.T) {
	repo := newExtraTeammateRepo()
	repo.allyErr = errors.New("duckdb: read failed")
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(
		newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test",
	)

	resp, err := svc.GetPage(context.Background(), "px", domain.TeammatesQueryRequest{
		SelectedGamertags:      []string{"AllyA", "AllyB"},
		FilterExactComposition: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	codes := issueCodes(resp.DataIssues)
	if _, ok := codes[domain.DataIssueMainTeamParticipants]; !ok {
		t.Errorf("DataIssues doit signaler %q, got %v", domain.DataIssueMainTeamParticipants, resp.DataIssues)
	}
	// Population non filtrée : les 2 sessions restent exposées.
	if len(resp.CompositionSessions) != 2 {
		t.Errorf("dégradation gracieuse : want 2 sessions (intersection brute), got %d", len(resp.CompositionSessions))
	}
}

// Matchs d'un coéquipier illisibles : il est retiré de la population commune —
// le compteur doit devenir suspect pour l'utilisateur, pas silencieusement faux.
func TestGetPage_DataIssue_TeammateMatchesFailure(t *testing.T) {
	repo := newExtraTeammateRepo()
	repo.squadErr = errors.New("player db locked")
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(
		newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test",
	)

	resp, err := svc.GetPage(context.Background(), "px", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"AllyA", "AllyB"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	codes := issueCodes(resp.DataIssues)
	detail, ok := codes[domain.DataIssueTeammateMatches]
	if !ok {
		t.Fatalf("DataIssues doit signaler %q, got %v", domain.DataIssueTeammateMatches, resp.DataIssues)
	}
	if detail != "AllyA" && detail != "AllyB" {
		t.Errorf("le détail doit identifier le coéquipier concerné, got %q", detail)
	}
}

// Historique par carte illisible : les références historiques manquent.
func TestGetPage_DataIssue_MapStatsFailure(t *testing.T) {
	repo := newExtraTeammateRepo()
	repo.mapStatsErr = errors.New("shared db busy")
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(
		newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test",
	)

	resp, err := svc.GetPage(context.Background(), "px", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"AllyA", "AllyB"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := issueCodes(resp.DataIssues)[domain.DataIssueMapStats]; !ok {
		t.Errorf("DataIssues doit signaler %q, got %v", domain.DataIssueMapStats, resp.DataIssues)
	}
}

// Page saine : aucun bruit dans DataIssues (le champ est omis du JSON).
func TestGetPage_NoDataIssuesWhenHealthy(t *testing.T) {
	repo := newExtraTeammateRepo()
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(
		newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test",
	)

	resp, err := svc.GetPage(context.Background(), "px", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"AllyA", "AllyB"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.DataIssues != nil {
		t.Errorf("page saine : DataIssues doit être nil, got %v", resp.DataIssues)
	}
}

// Heatmap : la ligne d'un coéquipier couvre EXACTEMENT la population reçue
// (allSquadRows). Avant le 2026-08-02, un re-filtrage privé par sessionMatchIDs
// (calculé sur les matchs du joueur principal) rétrécissait les lignes
// coéquipiers sous celle du main — un des écarts du 11/8/6/5.
func TestBuildSquadMapHeatmap_TeammateCoversSamePopulationAsMain(t *testing.T) {
	perf := 60.0
	streets := "Streets"
	mainSquadRows := []domain.SquadMatchRow{
		{MatchID: "m1", PerformanceScore: &perf, MapUI: streets, IsWithFriends: true},
		{MatchID: "m2", PerformanceScore: &perf, MapUI: streets, IsWithFriends: true},
	}
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			// m3 n'appartient pas à la population escouade : il ne doit pas compter.
			"friend1": {rowWithMap("m1", streets, 40), rowWithMap("m2", streets, 40), rowWithMap("m3", streets, 99)},
		},
	}
	svc := &TeammatesService{squadLoader: loader, titleSlug: "halo_infinite", gamertag: "main"}

	heatmap := svc.buildSquadMapHeatmap(context.Background(), mainSquadRows, []string{"friend1"}, &dataIssues{})
	if heatmap == nil {
		t.Fatal("heatmap should be non-nil")
	}
	var mainCell, friendCell *domain.SquadMapHeatmapCell
	for i := range heatmap.Cells {
		c := &heatmap.Cells[i]
		switch c.Player {
		case "main":
			mainCell = c
		case "friend1":
			friendCell = c
		}
	}
	if mainCell == nil || friendCell == nil {
		t.Fatalf("cells main+friend1 attendues, got %+v", heatmap.Cells)
	}
	if mainCell.MatchCount != 2 || friendCell.MatchCount != 2 {
		t.Errorf("même population attendue : main=%d friend=%d (want 2/2)", mainCell.MatchCount, friendCell.MatchCount)
	}
}

// Heatmap : LoadFor en échec → ligne vide ET dégradation remontée.
func TestBuildSquadMapHeatmap_LoadFailureIsReported(t *testing.T) {
	perf := 60.0
	streets := "Streets"
	mainSquadRows := []domain.SquadMatchRow{
		{MatchID: "m1", PerformanceScore: &perf, MapUI: streets, IsWithFriends: true},
	}
	loader := &fakeSquadLoader{errByGT: map[string]error{"friend1": errors.New("player db unavailable")}}
	svc := &TeammatesService{squadLoader: loader, titleSlug: "halo_infinite", gamertag: "main"}

	issues := &dataIssues{}
	if heatmap := svc.buildSquadMapHeatmap(context.Background(), mainSquadRows, []string{"friend1"}, issues); heatmap == nil {
		t.Fatal("heatmap should be non-nil")
	}
	detail, ok := issueCodes(issues.list())[domain.DataIssueHeatmapTeammate]
	if !ok || detail != "friend1" {
		t.Errorf("échec LoadFor : want issue %q/friend1, got %v", domain.DataIssueHeatmapTeammate, issues.list())
	}
}
