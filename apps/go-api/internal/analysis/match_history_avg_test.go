package analysis_test

import (
	"testing"

	"levelup/go-api/internal/analysis"
)

func TestComputeModeCategory_Firefight(t *testing.T) {
	cat := analysis.ComputeModeCategory("ranked", false, false)
	if cat != "pvp_ranked" {
		t.Errorf("attendu pvp_ranked, obtenu %s", cat)
	}
	catPVE := analysis.ComputeModeCategory("ranked", true, false)
	if catPVE != "pve" {
		t.Errorf("attendu pve pour firefight, obtenu %s", catPVE)
	}
}

func TestComputeModeCategory_Ranked(t *testing.T) {
	cat := analysis.ComputeModeCategory("slayer", false, true)
	if cat != "pvp_ranked" {
		t.Errorf("attendu pvp_ranked, obtenu %s", cat)
	}
}

func TestComputeModeCategory_Unknown(t *testing.T) {
	cat := analysis.ComputeModeCategory("", false, false)
	if cat != "pvp_unranked" {
		t.Errorf("attendu pvp_unranked pour mode vide, obtenu %s", cat)
	}
}

func TestComputeModeCategoryAverages_Empty(t *testing.T) {
	result := analysis.ComputeModeCategoryAverages(nil, "pvp_ranked")
	if result != nil {
		t.Errorf("attendu nil pour historique vide, obtenu %v", result)
	}
}

func TestComputeModeCategoryAverages_NoMatch(t *testing.T) {
	history := []analysis.HistoryRow{
		{ModeCategory: "slayer", IsRanked: false, Kills: 10, Deaths: 5, Assists: 2},
	}
	result := analysis.ComputeModeCategoryAverages(history, "pvp_ranked")
	if result != nil {
		t.Errorf("attendu nil car aucun match ranked, obtenu %v", result)
	}
}

func TestComputeModeCategoryAverages_Average(t *testing.T) {
	history := []analysis.HistoryRow{
		{ModeCategory: "slayer", IsRanked: true, Kills: 10, Deaths: 5, Assists: 2},
		{ModeCategory: "slayer", IsRanked: true, Kills: 20, Deaths: 10, Assists: 4},
	}
	result := analysis.ComputeModeCategoryAverages(history, "pvp_ranked")
	if result == nil {
		t.Fatal("attendu un résultat, obtenu nil")
	}
	if result.MatchCount != 2 {
		t.Errorf("attendu MatchCount=2, obtenu %d", result.MatchCount)
	}
	if result.AvgKills != 15.0 {
		t.Errorf("attendu AvgKills=15.0, obtenu %f", result.AvgKills)
	}
	if result.AvgDeaths != 7.5 {
		t.Errorf("attendu AvgDeaths=7.5, obtenu %f", result.AvgDeaths)
	}
}
