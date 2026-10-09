package teammates

// teammates_service_trends_membres_test.go — membres de la vue Escouade des tendances :
// unicité par xuid et service sans dépendances injectées.

import (
	"context"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

func gamertagsDesMembres(resp domain.TrendsPageResponse) []string {
	out := make([]string, 0, len(resp.Members))
	for _, m := range resp.Members {
		out = append(out, m.XUID+"="+m.Gamertag)
	}
	return out
}

func TestGetSquadTrends_DeuxGamertagsUnMemeXUID(t *testing.T) {
	repo := repoTendances()
	repo.lookupAliases = map[string]string{"alias": "xa"}
	resp, err := nouveauServiceTendances(&lecturesComptees{mockSquadRepo: repo}).GetSquadTrends(
		context.Background(), "px",
		domain.TrendsQueryRequest{View: "squad", SelectedGamertags: []string{"AllyA", "Alias"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(gamertagsDesMembres(resp), ","); got != "px=Test,xa=AllyA" {
		t.Fatalf("membres = %s", got)
	}
	if valeur365(t, resp, "kda", "Alias") != nil {
		t.Error("le doublon ne doit pas avoir de ligne")
	}
	if v := valeur365(t, resp, "squad_share_of_team_kills", ""); v == nil || *v > 1 {
		t.Errorf("part de l'équipe = %v, ne doit pas dépasser 1", v)
	}
	if v := valeur365(t, resp, "member_share_of_squad_kills", "AllyA"); v == nil || *v > 1 {
		t.Errorf("part d'AllyA = %v", v)
	}
}

func TestGetSquadTrends_GamertagDuJoueurPrincipalIgnore(t *testing.T) {
	repo := repoTendances()
	repo.lookupAliases = map[string]string{"moi": "px"}
	resp, err := nouveauServiceTendances(&lecturesComptees{mockSquadRepo: repo}).GetSquadTrends(
		context.Background(), "px",
		domain.TrendsQueryRequest{View: "squad", SelectedGamertags: []string{"Moi"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Indicators) != 0 || resp.Indicators == nil || resp.View != domain.TrendsViewSquad {
		t.Errorf("attendu la réponse sans coéquipier : %+v", resp)
	}
	if got := strings.Join(gamertagsDesMembres(resp), ","); got != "px=Test" {
		t.Errorf("membres = %s", got)
	}
}

func TestGetSquadTrends_MembresOrdreDeLaRequete(t *testing.T) {
	resp, err := nouveauServiceTendances(&lecturesComptees{mockSquadRepo: repoTendances()}).GetSquadTrends(
		context.Background(), "px",
		domain.TrendsQueryRequest{View: "squad", SelectedGamertags: []string{"AllyB", "AllyA"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(gamertagsDesMembres(resp), ","); got != "px=Test,xb=AllyB,xa=AllyA" {
		t.Errorf("membres = %s", got)
	}
}

func TestGetSquadTrends_SansWithTrends(t *testing.T) {
	svc := NewTeammatesService(&lecturesComptees{mockSquadRepo: repoTendances()}, nil).
		WithPlayerMatchesRepo(newSynthMockFromRows(historiqueTendances(), nil), "halo_infinite", "Test")
	_, err := svc.GetSquadTrends(context.Background(), "px",
		domain.TrendsQueryRequest{View: "squad", SelectedGamertags: []string{"AllyA"}})
	if err == nil || !strings.Contains(err.Error(), "WithTrends") {
		t.Fatalf("erreur = %v, attendu une erreur explicite", err)
	}
}
