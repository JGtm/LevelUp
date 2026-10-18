// Package service — home_service_analysis_pending_test.go : l'indicateur « analyse du film en
// cours » (RecentMatchItem.AnalysisPending) sur la page d'accueil COMPLÈTE, repo simulé.
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

const (
	idAnalyseEnCours = "aaaa1111-0000-4000-8000-000000000001"
	idAnalyseFinie   = "bbbb2222-0000-4000-8000-000000000002"
)

func pendingParID(items []domain.RecentMatchItem) map[string]bool {
	out := map[string]bool{}
	for _, it := range items {
		out[it.MatchID] = it.AnalysisPending
	}
	return out
}

// TestGetHomePage_AnalysisPending_DeBoutEnBout — deux matchs récents, le premier sans film
// analysé, le second terminé : seul le premier porte le signal, sur la page servie.
func TestGetHomePage_AnalysisPending_DeBoutEnBout(t *testing.T) {
	repo := homeReplayRepo(time.Now())
	sonde := &probeRepoFactice{probes: map[string]domain.AnalysisProbe{
		idAnalyseEnCours: synchroIlYA(time.Hour, func(p *domain.AnalysisProbe) { p.KillSourceDone, p.HasCitationRows = false, false }),
		idAnalyseFinie:   synchroIlYA(time.Hour, nil),
	}}
	stub := &stubReplayService{shortIDs: []string{"bbbb2222"}}
	svc := withHomeMock(NewHomeService(repo), repo).WithReplay(stub).
		WithAnalysisPending(resolveurDeTest(sonde, true))

	resp, err := svc.GetHomePage(context.Background(), "TestGT", "fr")
	if err != nil {
		t.Fatalf("GetHomePage: %v", err)
	}
	got := pendingParID(resp.RecentMatches)
	if !got[idAnalyseEnCours] || got[idAnalyseFinie] || len(got) != 2 {
		t.Fatalf("analysis_pending par tuile = %v", got)
	}
	if sonde.appels != 1 {
		t.Fatalf("une seule lecture d'avancement attendue par page, %d", sonde.appels)
	}
	if stub.calls != 1 {
		t.Fatalf("un seul listing de rejeux par page, %d", stub.calls)
	}
}

// TestGetHomePage_AnalysisPending_SansResolveurOuEnEchec — titre sans la capability (pas de
// résolveur) ou lecture en échec : la page est servie, aucun indicateur.
func TestGetHomePage_AnalysisPending_SansResolveurOuEnEchec(t *testing.T) {
	repo := homeReplayRepo(time.Now())
	resp, err := withHomeMock(NewHomeService(repo), repo).GetHomePage(context.Background(), "TestGT", "fr")
	if err != nil {
		t.Fatalf("sans résolveur : %v", err)
	}
	for id, p := range pendingParID(resp.RecentMatches) {
		if p {
			t.Errorf("sans résolveur : %s ne doit pas être en cours", id)
		}
	}

	repo = homeReplayRepo(time.Now())
	sonde := &probeRepoFactice{err: errors.New("registre illisible")}
	resp, err = withHomeMock(NewHomeService(repo), repo).WithAnalysisPending(resolveurDeTest(sonde, true)).
		GetHomePage(context.Background(), "TestGT", "fr")
	if err != nil {
		t.Fatalf("lecture en échec : la page doit être servie, %v", err)
	}
	for id, p := range pendingParID(resp.RecentMatches) {
		if p {
			t.Errorf("lecture en échec : %s ne doit pas être en cours", id)
		}
	}
}
