package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

// --- mock ---

type mockLeaderboardRepo struct {
	csrWorld []domain.LeaderboardEntry
	stats    []domain.LeaderboardEntry
	local    []domain.LeaderboardEntry
	err      error

	lastCategory domain.LeaderboardCategory
	lastLimit    int
}

func (m *mockLeaderboardRepo) GetLocalLeaderboard(_ context.Context, _, _, _ string) ([]domain.LeaderboardEntry, error) {
	return m.local, m.err
}

func (m *mockLeaderboardRepo) GetCSRWorldLeaderboard(_ context.Context, _, _, _ string, limit int) ([]domain.LeaderboardEntry, error) {
	m.lastCategory = domain.LeaderboardCSRWorld
	m.lastLimit = limit
	return m.csrWorld, m.err
}

func (m *mockLeaderboardRepo) GetStatLeaderboard(_ context.Context, _ string, category domain.LeaderboardCategory, _, _ string, limit int) ([]domain.LeaderboardEntry, error) {
	m.lastCategory = category
	m.lastLimit = limit
	return m.stats, m.err
}

func (m *mockLeaderboardRepo) GetWorldLeaderboardCatalog(_ context.Context, _ string) (domain.LeaderboardCatalog, error) {
	return domain.LeaderboardCatalog{}, m.err
}

// TestLeaderboardService_NoCapability_EmptyNot500 (PMT-7 oracle b) : un titre sans
// CapWorldLeaderboard (ou inconnu) dégrade en vide + 200 — JAMAIS 500, et sans
// appeler le repo. Gating par capability, pas par comparaison de slug.
func TestLeaderboardService_NoCapability_EmptyNot500(t *testing.T) {
	repo := &mockLeaderboardRepo{csrWorld: []domain.LeaderboardEntry{{Rank: 1, Gamertag: "X"}}}
	svc := NewLeaderboardService(repo)

	resp, err := svc.GetPage(context.Background(), domain.LeaderboardRequest{TitleSlug: "unknown_title_no_cap"})
	if err != nil {
		t.Fatalf("titre sans capability devrait dégrader (pas err) : %v", err)
	}
	if len(resp.Entries) != 0 {
		t.Errorf("titre sans CapWorldLeaderboard → %d entrées, want 0 (vide+200)", len(resp.Entries))
	}
	if resp.TitleSlug != "unknown_title_no_cap" {
		t.Errorf("TitleSlug résolu = %q, want unknown_title_no_cap", resp.TitleSlug)
	}
}

// TestLeaderboardService_MissingSeasonOrPlaylist_EmptyNot500 (Lot 4.1) : le
// classement mondial se définit par un COUPLE (saison, playlist). Un couple
// incomplet n'est pas une panne : réponse vide + 200, le repo n'est même pas
// appelé (son erreur « season et playlist requis » ne doit plus remonter en 500).
func TestLeaderboardService_MissingSeasonOrPlaylist_EmptyNot500(t *testing.T) {
	cases := []struct{ name, season, playlist string }{
		{"aucun des deux", "", ""},
		{"saison manquante", "", "p"},
		{"playlist manquante", "s", ""},
		{"blancs seulement", "   ", "  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Le repo échoue systématiquement : si le service l'appelait, le test
			// verrait l'erreur (c'est le 500 qu'on supprime).
			repo := &mockLeaderboardRepo{err: errors.New("le repo ne doit pas être appelé")}
			svc := NewLeaderboardService(repo)

			resp, err := svc.GetPage(context.Background(), domain.LeaderboardRequest{
				Season: tc.season, Playlist: tc.playlist,
			})
			if err != nil {
				t.Fatalf("couple incomplet doit dégrader en vide, got err: %v", err)
			}
			if len(resp.Entries) != 0 {
				t.Errorf("entries = %d, want 0", len(resp.Entries))
			}
			if resp.TotalLocal != 0 {
				t.Errorf("total = %d, want 0", resp.TotalLocal)
			}
			if resp.Category != string(domain.LeaderboardCSRWorld) {
				t.Errorf("category = %q, want csr-world", resp.Category)
			}
			if repo.lastCategory != "" {
				t.Errorf("le repo a été appelé (%q) alors que le couple est incomplet", repo.lastCategory)
			}
		})
	}
}

// TestLeaderboardService_StatCategory_NoSeasonPlaylist_StillQueries : la garde
// 4.1 ne concerne QUE le classement mondial — les catégories de stats agrégées
// acceptent des filtres vides (saison/playlist optionnelles).
func TestLeaderboardService_StatCategory_NoSeasonPlaylist_StillQueries(t *testing.T) {
	repo := &mockLeaderboardRepo{stats: []domain.LeaderboardEntry{{Rank: 1, Gamertag: "Alice", Value: 42}}}
	svc := NewLeaderboardService(repo)

	resp, err := svc.GetPage(context.Background(), domain.LeaderboardRequest{Category: string(domain.LeaderboardKills)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastCategory != domain.LeaderboardKills {
		t.Errorf("le repo de stats doit être appelé, lastCategory = %q", repo.lastCategory)
	}
	if len(resp.Entries) != 1 {
		t.Errorf("entries = %d, want 1", len(resp.Entries))
	}
}

// --- tests ---

// La catégorie par défaut route vers le classement CSR mondial.
func TestLeaderboardService_DefaultsToCSRWorld(t *testing.T) {
	entries := []domain.LeaderboardEntry{
		{Rank: 1, Gamertag: "Twissted Mindss", CSRValue: 2180},
		{Rank: 2, Gamertag: "OR81TAL", CSRValue: 2097},
	}
	repo := &mockLeaderboardRepo{csrWorld: entries}
	svc := NewLeaderboardService(repo)

	resp, err := svc.GetPage(context.Background(), domain.LeaderboardRequest{Season: "s", Playlist: "p"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastCategory != domain.LeaderboardCSRWorld {
		t.Errorf("expected csr-world routing, got %q", repo.lastCategory)
	}
	if resp.Category != string(domain.LeaderboardCSRWorld) {
		t.Errorf("expected response category csr-world, got %q", resp.Category)
	}
	if repo.lastLimit != defaultLeaderboardLimit {
		t.Errorf("expected default limit %d, got %d", defaultLeaderboardLimit, repo.lastLimit)
	}
	if len(resp.Entries) != 2 || resp.Entries[0].Gamertag != "Twissted Mindss" {
		t.Fatalf("unexpected entries: %+v", resp.Entries)
	}
}

// Une catégorie de stat route vers l'agrégation match_participants.
func TestLeaderboardService_StatCategory(t *testing.T) {
	repo := &mockLeaderboardRepo{stats: []domain.LeaderboardEntry{{Rank: 1, Gamertag: "Alice", Value: 42}}}
	svc := NewLeaderboardService(repo)

	resp, err := svc.GetPage(context.Background(), domain.LeaderboardRequest{
		Category: string(domain.LeaderboardKDA), Limit: 25,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastCategory != domain.LeaderboardKDA {
		t.Errorf("expected kda routing, got %q", repo.lastCategory)
	}
	if repo.lastLimit != 25 {
		t.Errorf("expected limit 25, got %d", repo.lastLimit)
	}
	if len(resp.Entries) != 1 || resp.Entries[0].Gamertag != "Alice" {
		t.Fatalf("unexpected entries: %+v", resp.Entries)
	}
}

func TestLeaderboardService_Empty(t *testing.T) {
	svc := NewLeaderboardService(&mockLeaderboardRepo{csrWorld: []domain.LeaderboardEntry{}})
	resp, err := svc.GetPage(context.Background(), domain.LeaderboardRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(resp.Entries))
	}
}
