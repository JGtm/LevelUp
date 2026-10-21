package teammates

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// Un MMR à 0 (match classé abandonné) est une valeur absente : le tableau de l'escouade n'en
// tire ni MMR d'équipe, ni écart. Le lecteur SQL rend TeamMMR à 0 pour « inconnu » (colonne non
// nullable) ; c'est ici, avant l'affichage, que ce 0 doit redevenir absent.
func TestBuildSquadMatchHistory_ZeroMMRIsAbsent(t *testing.T) {
	enemy := 1100.0
	zero := 0.0
	cases := []struct {
		name          string
		team          float64
		enemy         *float64
		wantTeam      bool
		wantEnemy     bool
		wantDeltaMMR  bool
		wantDeltaWant float64
	}{
		{"match classé abandonné : tout à 0", 0, &zero, false, false, false, 0},
		{"équipe à 0, adversaire réel : pas d'écart", 0, &enemy, false, true, false, 0},
		{"équipe réelle, adversaire absent : pas d'écart", 1200, nil, true, false, false, 0},
		{"les deux réels : écart", 1200, &enemy, true, true, true, 100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hist := buildSquadMatchHistory([]domain.SquadMatchRow{
				{MatchID: "m1", TeamMMR: tc.team, EnemyMMR: tc.enemy},
			}, nil, "halo_infinite", nil, nil)
			if len(hist) != 1 {
				t.Fatalf("1 ligne attendue, got %d", len(hist))
			}
			h := hist[0]
			if (h.TeamMMRAvg != nil) != tc.wantTeam {
				t.Errorf("TeamMMRAvg présent = %v, attendu %v (%v)", h.TeamMMRAvg != nil, tc.wantTeam, h.TeamMMRAvg)
			}
			if (h.EnemyMMRAvg != nil) != tc.wantEnemy {
				t.Errorf("EnemyMMRAvg présent = %v, attendu %v", h.EnemyMMRAvg != nil, tc.wantEnemy)
			}
			if (h.DeltaMMR != nil) != tc.wantDeltaMMR {
				t.Fatalf("DeltaMMR présent = %v, attendu %v", h.DeltaMMR != nil, tc.wantDeltaMMR)
			}
			if tc.wantDeltaMMR && *h.DeltaMMR != tc.wantDeltaWant {
				t.Errorf("DeltaMMR = %v, attendu %v", *h.DeltaMMR, tc.wantDeltaWant)
			}
		})
	}
}

func TestBuildMatchSeries_ZeroMMRIsAbsent(t *testing.T) {
	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	got := buildMatchSeries([]domain.SquadMatchRow{
		{MatchID: "m0", StartTime: ts, TeamMMR: 0},
		{MatchID: "m1", StartTime: ts, TeamMMR: 1200},
	})
	if len(got) != 2 {
		t.Fatalf("2 points attendus, got %d", len(got))
	}
	if got[0].TeamMMRAvg != nil {
		t.Errorf("MMR à 0 attendu absent, got %v", *got[0].TeamMMRAvg)
	}
	if got[1].TeamMMRAvg == nil || *got[1].TeamMMRAvg != 1200 {
		t.Errorf("MMR réel conservé, got %v", got[1].TeamMMRAvg)
	}
}
