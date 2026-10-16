// Package handlers_test — leaderboard_limit_test.go : plafond serveur du paramètre
// `limit` du classement.
//
// Contrat : quelle que soit la valeur de `limit`, la réponse porte au plus
// domain.MaxLeaderboardLimit entrées et le repo ne reçoit jamais une limite
// supérieure (il dimensionne ses allocations dessus).
package handlers_test

import (
	"context"
	"net/http"
	"testing"

	"levelup/go-api/internal/domain"
)

// limitEchoRepo rend autant d'entrées que la limite demandée (au plus
// echoCeiling, pour que le test ne s'alloue pas lui-même des gigaoctets) et
// retient la limite reçue.
type limitEchoRepo struct {
	strictLeaderboardRepo
	gotLimit int
}

const echoCeiling = 5000

func (r *limitEchoRepo) GetCSRWorldLeaderboard(_ context.Context, _, _, _ string, limit int) ([]domain.LeaderboardEntry, error) {
	r.gotLimit = limit
	n := min(limit, echoCeiling)
	out := make([]domain.LeaderboardEntry, n)
	for i := range out {
		out[i] = domain.LeaderboardEntry{Rank: i + 1, Gamertag: "P"}
	}
	return out, nil
}

func TestLeaderboardPage_LimitIsCappedServerSide(t *testing.T) {
	const couple = "?season=csrseason13-3&playlist=pl-a"
	cases := []struct {
		name        string
		limit       string
		wantEntries int
	}{
		{"milliard", "&limit=1000000000", domain.MaxLeaderboardLimit},
		{"juste au-dessus du plafond", "&limit=201", domain.MaxLeaderboardLimit},
		{"au plafond", "&limit=200", 200},
		{"valeur du front", "&limit=100", 100},
		{"absent", "", 50},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &limitEchoRepo{}
			code, body := getLeaderboard(t, newLeaderboardRouter(repo), couple+tc.limit)
			if code != http.StatusOK {
				t.Fatalf("status = %d, want 200", code)
			}
			if len(body.Entries) != tc.wantEntries {
				t.Errorf("entrées = %d, want %d", len(body.Entries), tc.wantEntries)
			}
			if repo.gotLimit > domain.MaxLeaderboardLimit {
				t.Errorf("le repo a reçu limit=%d, au-dessus du plafond %d", repo.gotLimit, domain.MaxLeaderboardLimit)
			}
		})
	}
}
