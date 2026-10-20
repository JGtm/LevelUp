package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

// TestGetRelationsPage_Assists : le bloc d'assistances est posé sur les relations présentes
// dans la map, objet à zéro compris (« zéro assistance » sur des matchs porteurs) ; une
// relation absente de la map reste nil — jamais un objet fabriqué « 0 assistance ».
func TestGetRelationsPage_Assists(t *testing.T) {
	repo := &mockRelationsRepo{
		rows: []domain.RelationRawRow{
			{XUID: "x1", Gamertag: "Mesure", TotalMatches: 5, TeammateCount: 5},
			{XUID: "x2", Gamertag: "Absent", TotalMatches: 5, TeammateCount: 5},
			{XUID: "x3", Gamertag: "Zero", TotalMatches: 5, TeammateCount: 5},
		},
		assistsByXUID: map[string]domain.RelationAssists{
			"x1": {MyFrags: 40, PartnerFrags: 30,
				Received: domain.AssistTiers{Total: 7, Low: 1, Mid: 2, High: 4},
				Given:    domain.AssistTiers{Total: 5, Low: 2, Mid: 2, High: 1}},
			"x3": {MyFrags: 12, PartnerFrags: 9},
		},
	}
	page, err := NewRelationsService(repo).withNow(fixedNow()).
		GetRelationsPage(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	byGT := map[string]domain.RelationInsight{}
	for _, r := range page.Relations {
		byGT[r.Gamertag] = r
	}
	got := byGT["Mesure"].Assists
	if got == nil || got.Received.High != 4 || got.Given.Total != 5 || got.MyFrags != 40 {
		t.Fatalf("Mesure assists = %+v", got)
	}
	if byGT["Absent"].Assists != nil {
		t.Fatal("relation absente de la map : assists doit rester nil")
	}
	if z := byGT["Zero"].Assists; z == nil || z.Received.Total != 0 || z.MyFrags != 12 {
		t.Fatalf("relation à zéro assistance : objet attendu, got %+v", z)
	}
}

// TestGetRelationsPage_AssistsErrorIsBestEffort : une erreur de lecture des assistances
// ne fait pas échouer la page.
func TestGetRelationsPage_AssistsErrorIsBestEffort(t *testing.T) {
	repo := &mockRelationsRepo{
		rows:       []domain.RelationRawRow{{XUID: "x1", Gamertag: "A", TotalMatches: 5, TeammateCount: 5}},
		assistsErr: errors.New("boom"),
	}
	page, err := NewRelationsService(repo).withNow(fixedNow()).
		GetRelationsPage(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("la page ne doit pas échouer : %v", err)
	}
	if len(page.Relations) != 1 || page.Relations[0].Assists != nil {
		t.Fatalf("relations = %+v", page.Relations)
	}
}

// TestAttachEncounterAssists : même contrat côté vue match.
func TestAttachEncounterAssists(t *testing.T) {
	rows := []domain.MatchEncounterRow{{XUID: "a"}, {XUID: "b"}}
	attachEncounterAssists(rows, map[string]domain.RelationAssists{
		"a": {MyFrags: 9, Received: domain.AssistTiers{Total: 3}},
	})
	if rows[0].Assists == nil || rows[0].Assists.Received.Total != 3 {
		t.Fatalf("a = %+v", rows[0].Assists)
	}
	if rows[1].Assists != nil {
		t.Fatal("b absent de la map : nil attendu")
	}
	attachEncounterAssists(rows, nil) // map nil : aucune panique
}
