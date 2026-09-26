// Package service — home_service_replay_test.go : le lien « Rejeu 2D » des tuiles de
// match de l'Accueil (RecentMatchItem.HasReplay). Deux invariants, les mêmes que pour
// les tableaux (match_history_replay_test.go) :
//   - la présence d'artefact est résolue UNE FOIS par requête pour TOUTES les tuiles
//     (récents + favoris) — le stub partagé COMPTE les listings ;
//   - service non câblé ou listing en échec = tuiles servies sans lien, jamais un 500.
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/legacymatch"
)

// homeReplayRepo : deux matchs récents, dont un seul aura un artefact de rejeu.
func homeReplayRepo(now time.Time) *mockHomeRepo {
	return &mockHomeRepo{
		matches: []legacymatch.HomeMatchRow{
			{MatchID: "aaaa1111-0000-4000-8000-000000000001", StartTime: now, MapName: "Aquarius", PairName: "Slayer", Outcome: 2, Kills: 10, Deaths: 5},
			{MatchID: "bbbb2222-0000-4000-8000-000000000002", StartTime: now.Add(-1 * time.Hour), MapName: "Streets", PairName: "CTF", Outcome: 3, Kills: 5, Deaths: 10},
		},
	}
}

// TestGetHomePage_HasReplay_UnSeulListing — has_replay est publié sur la bonne tuile,
// et le service n'interroge la présence qu'UNE fois pour toute la page (les deux
// listes, récents et favoris, partagent le même ensemble).
func TestGetHomePage_HasReplay_UnSeulListing(t *testing.T) {
	repo := homeReplayRepo(time.Now())
	stub := &stubReplayService{shortIDs: []string{"aaaa1111"}}
	svc := withHomeMock(NewHomeService(repo), repo).WithReplay(stub)

	resp, err := svc.GetHomePage(context.Background(), "TestGT", "fr")
	if err != nil {
		t.Fatalf("GetHomePage: %v", err)
	}
	if stub.calls != 1 {
		t.Fatalf("un seul listing attendu par requête, %d appels", stub.calls)
	}
	byID := map[string]bool{}
	for _, it := range resp.RecentMatches {
		byID[it.MatchID] = it.HasReplay
	}
	if !byID["aaaa1111-0000-4000-8000-000000000001"] {
		t.Error("le match avec artefact doit porter has_replay")
	}
	if byID["bbbb2222-0000-4000-8000-000000000002"] {
		t.Error("un match sans artefact ne doit jamais porter has_replay")
	}
}

// TestGetHomePage_Replay_DegradationPropre — service non câblé (titre sans rejeu) ou
// listing en échec : la page se sert SANS lien, jamais en erreur.
func TestGetHomePage_Replay_DegradationPropre(t *testing.T) {
	// Sans service injecté.
	repo := homeReplayRepo(time.Now())
	resp, err := withHomeMock(NewHomeService(repo), repo).
		GetHomePage(context.Background(), "TestGT", "fr")
	if err != nil {
		t.Fatalf("service non câblé : %v", err)
	}
	for _, it := range resp.RecentMatches {
		if it.HasReplay {
			t.Error("service non câblé : aucune tuile ne doit porter has_replay")
		}
	}

	// Listing en échec : page servie, lien éteint.
	repo = homeReplayRepo(time.Now())
	failing := &stubReplayService{err: errors.New("dossier illisible")}
	resp, err = withHomeMock(NewHomeService(repo), repo).WithReplay(failing).
		GetHomePage(context.Background(), "TestGT", "fr")
	if err != nil {
		t.Fatalf("listing en échec : la page doit être servie, obtenu %v", err)
	}
	if len(resp.RecentMatches) != 2 {
		t.Errorf("listing en échec : 2 tuiles attendues, %d", len(resp.RecentMatches))
	}
	for _, it := range resp.RecentMatches {
		if it.HasReplay {
			t.Error("listing en échec : aucune tuile ne doit porter has_replay")
		}
	}
}
