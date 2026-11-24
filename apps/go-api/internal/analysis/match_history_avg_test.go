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
