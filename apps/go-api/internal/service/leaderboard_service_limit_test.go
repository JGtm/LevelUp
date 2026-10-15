package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

// La limite transmise au repo est plafonnée à domain.MaxLeaderboardLimit, pour le
// classement mondial comme pour les classements de stats ; en deçà elle passe telle
// quelle, et une limite absente ou négative prend la valeur par défaut.
func TestLeaderboardService_LimitIsCapped(t *testing.T) {
	cases := []struct {
		name      string
		category  domain.LeaderboardCategory
		limit     int
		wantLimit int
	}{
		{"csr-world énorme", domain.LeaderboardCSRWorld, 1_000_000_000, domain.MaxLeaderboardLimit},
		{"csr-world plafond+1", domain.LeaderboardCSRWorld, domain.MaxLeaderboardLimit + 1, domain.MaxLeaderboardLimit},
		{"csr-world plafond", domain.LeaderboardCSRWorld, domain.MaxLeaderboardLimit, domain.MaxLeaderboardLimit},
		{"csr-world ordinaire", domain.LeaderboardCSRWorld, 100, 100},
		{"csr-world négative", domain.LeaderboardCSRWorld, -5, defaultLeaderboardLimit},
		{"stats énorme", domain.LeaderboardKills, 1_000_000_000, domain.MaxLeaderboardLimit},
		{"stats ordinaire", domain.LeaderboardKills, 100, 100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockLeaderboardRepo{}
			svc := NewLeaderboardService(repo)
			_, err := svc.GetPage(context.Background(), domain.LeaderboardRequest{
				Category: string(tc.category), Season: "s", Playlist: "p", Limit: tc.limit,
			})
			if err != nil {
				t.Fatalf("GetPage : %v", err)
			}
			if repo.lastLimit != tc.wantLimit {
				t.Errorf("limite transmise au repo = %d, want %d", repo.lastLimit, tc.wantLimit)
			}
		})
	}
}
