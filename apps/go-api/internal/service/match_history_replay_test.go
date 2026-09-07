// Package service — match_history_replay_test.go : la colonne « Rejeu » des tableaux
// de matchs et son filtre `replay_scope`. Deux invariants s'y jouent :
//   - la présence d'artefact est résolue UNE FOIS par requête (un listing), jamais un
//     accès disque par ligne — le stub ci-dessous COMPTE les appels ;
//   - le filtre a 3 états, comme le scope escouade dont il copie la forme.
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// stubReplayService sert un ensemble fixe et compte les listings : c'est le garde-fou
// du « un seul os.ReadDir par requête ».
type stubReplayService struct {
	shortIDs []string
	err      error
	calls    int
}

func (s *stubReplayService) GetReplay(context.Context, string) (replaydoc.ReplayDocument, error) {
	return replaydoc.ReplayDocument{}, port.ErrReplayNotAvailable
}

func (s *stubReplayService) IsAvailable(context.Context, string) bool { return false }

func (s *stubReplayService) AvailableSet(context.Context) (port.ReplayAvailability, error) {
	s.calls++
	if s.err != nil {
		return port.ReplayAvailability{}, s.err
	}
	set := make(port.ReplayAvailability, len(s.shortIDs))
	for _, id := range s.shortIDs {
		set[title.FilmShortMatchID(id)] = struct{}{}
	}
	return set, nil
}

func (s *stubReplayService) MapBackground(context.Context, string) (*replaydoc.MapBackground, error) {
	return nil, port.ErrMapBackgroundNotAvailable
}

func (s *stubReplayService) MapBackgroundImage(context.Context, string) ([]byte, error) {
	return nil, port.ErrMapBackgroundNotAvailable
}

func (s *stubReplayService) MapBackgroundForMap(context.Context, string) (*replaydoc.MapBackground, error) {
	return nil, port.ErrMapBackgroundNotAvailable
}

func (s *stubReplayService) MapBackgroundImageForMap(context.Context, string) ([]byte, error) {
	return nil, port.ErrMapBackgroundNotAvailable
}

func (s *stubReplayService) MapCallouts(context.Context, string) (*replaydoc.MapCalloutsEntry, error) {
	return nil, port.ErrMapCalloutsNotAvailable
}

// replayTestRows : trois matchs, dont un seul aura un artefact.
func replayTestRows() []domain.MatchHistoryRawRow {
	now := time.Now()
	return []domain.MatchHistoryRawRow{
		{MatchID: "aaaa1111-0000-4000-8000-000000000001", StartTime: &now, Outcome: 2},
		{MatchID: "bbbb2222-0000-4000-8000-000000000002", StartTime: &now, Outcome: 3},
		{MatchID: "cccc3333-0000-4000-8000-000000000003", StartTime: &now, Outcome: 2},
	}
}

func replayRowMatchIDs(items []domain.MatchHistoryRow) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.MatchID)
	}
	return out
}

// TestGetPage_HasReplay_UnSeulListing — has_replay est publié sur la bonne ligne, et le
// service n'interroge la présence qu'UNE fois pour toute la requête.
func TestGetPage_HasReplay_UnSeulListing(t *testing.T) {
	stub := &stubReplayService{shortIDs: []string{"aaaa1111"}}
	svc := NewMatchHistoryService(&mockMatchHistoryRepo{rows: replayTestRows()}, "GT").WithReplay(stub)

	resp, err := svc.GetPage(context.Background(), domain.MatchHistoryQueryRequest{})
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	if stub.calls != 1 {
		t.Fatalf("un seul listing attendu par requête, %d appels", stub.calls)
	}
	byID := map[string]bool{}
	for _, it := range resp.Table.Items {
		byID[it.MatchID] = it.HasReplay
	}
	if !byID["aaaa1111-0000-4000-8000-000000000001"] {
		t.Error("le match avec artefact doit porter has_replay")
	}
	if byID["bbbb2222-0000-4000-8000-000000000002"] || byID["cccc3333-0000-4000-8000-000000000003"] {
		t.Error("un match sans artefact ne doit jamais porter has_replay")
	}
}

// TestGetPage_ReplayScope_TroisEtats — la forme exacte du scope escouade : vide = tous,
// "with" = avec rejeu, "without" = sans.
func TestGetPage_ReplayScope_TroisEtats(t *testing.T) {
	cases := []struct {
		scope string
		want  []string
	}{
		{"", []string{
			"aaaa1111-0000-4000-8000-000000000001",
			"bbbb2222-0000-4000-8000-000000000002",
			"cccc3333-0000-4000-8000-000000000003",
		}},
		{"with", []string{"aaaa1111-0000-4000-8000-000000000001"}},
		{"without", []string{
			"bbbb2222-0000-4000-8000-000000000002",
			"cccc3333-0000-4000-8000-000000000003",
		}},
	}
	for _, c := range cases {
		svc := NewMatchHistoryService(&mockMatchHistoryRepo{rows: replayTestRows()}, "GT").
			WithReplay(&stubReplayService{shortIDs: []string{"aaaa1111"}})
		resp, err := svc.GetPage(context.Background(), domain.MatchHistoryQueryRequest{ReplayScope: c.scope})
		if err != nil {
			t.Fatalf("scope %q: %v", c.scope, err)
		}
		got := replayRowMatchIDs(resp.Table.Items)
		if len(got) != len(c.want) {
			t.Fatalf("scope %q: %d lignes attendues, %d obtenues (%v)", c.scope, len(c.want), len(got), got)
		}
		for _, want := range c.want {
			found := false
			for _, g := range got {
				if g == want {
					found = true
				}
			}
			if !found {
				t.Errorf("scope %q: %s manquant dans %v", c.scope, want, got)
			}
		}
		if resp.Summary.TotalMatchesScoped != len(c.want) {
			t.Errorf("scope %q: total_matches_scoped=%d, attendu %d",
				c.scope, resp.Summary.TotalMatchesScoped, len(c.want))
		}
	}
}

// TestGetPage_Replay_DegradationPropre — service non câblé (titre sans rejeu) ou listing
// en échec : la page se sert SANS la colonne, jamais en erreur.
func TestGetPage_Replay_DegradationPropre(t *testing.T) {
	// Sans service injecté.
	resp, err := NewMatchHistoryService(&mockMatchHistoryRepo{rows: replayTestRows()}, "GT").
		GetPage(context.Background(), domain.MatchHistoryQueryRequest{})
	if err != nil {
		t.Fatalf("service non câblé : %v", err)
	}
	for _, it := range resp.Table.Items {
		if it.HasReplay {
			t.Error("service non câblé : aucune ligne ne doit porter has_replay")
		}
	}

	// Listing en échec : réponse servie, colonne éteinte.
	failing := &stubReplayService{err: errors.New("dossier illisible")}
	resp, err = NewMatchHistoryService(&mockMatchHistoryRepo{rows: replayTestRows()}, "GT").
		WithReplay(failing).
		GetPage(context.Background(), domain.MatchHistoryQueryRequest{})
	if err != nil {
		t.Fatalf("listing en échec : la page doit être servie, obtenu %v", err)
	}
	if len(resp.Table.Items) != 3 {
		t.Errorf("listing en échec : 3 lignes attendues, %d", len(resp.Table.Items))
	}
	for _, it := range resp.Table.Items {
		if it.HasReplay {
			t.Error("listing en échec : aucune ligne ne doit porter has_replay")
		}
	}
}

// TestFilterByExplorerReplayScope_EnsembleVide — un titre sans artefact : « avec rejeu »
// ne garde rien, « sans rejeu » garde tout. C'est la vérité mesurée, pas une panne.
func TestFilterByExplorerReplayScope_EnsembleVide(t *testing.T) {
	rows := replayTestRows()
	if got := filterByExplorerReplayScope(rows, scopeWithReplay, nil); len(got) != 0 {
		t.Errorf("ensemble vide + with : 0 ligne attendue, %d", len(got))
	}
	if got := filterByExplorerReplayScope(rows, scopeWithoutReplay, nil); len(got) != len(rows) {
		t.Errorf("ensemble vide + without : %d lignes attendues, %d", len(rows), len(got))
	}
	if got := filterByExplorerReplayScope(rows, "bogus", nil); len(got) != len(rows) {
		t.Errorf("scope inconnu = no-op : %d lignes attendues, %d", len(rows), len(got))
	}
}
