package analysis

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// Renforcement M3 : les cas-garde « < 2 joueurs humains → extrêmes vides ».
// La désignation MVP/LVP sur scoreboard complet est couverte par
// citations_composite_test.go ; ici on verrouille les bornes.
func TestComputeMVPLVP_GuardBranches(t *testing.T) {
	cases := []struct {
		name string
		rows []domain.ScoreboardRaw
	}{
		{"vide", nil},
		{"un seul humain", []domain.ScoreboardRaw{{XUID: "2533274800000001"}}},
		{"que des bots", []domain.ScoreboardRaw{{XUID: "bid(1.0)"}, {XUID: "bid(2.0)"}}},
		{
			"un humain + des bots (< 2 humains)",
			[]domain.ScoreboardRaw{{XUID: "2533274800000001"}, {XUID: "bid(1.0)"}, {XUID: "bid(2.0)"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeMVPLVP(tc.rows)
			if got.MVPXUID != "" || got.LVPXUID != "" {
				t.Errorf("attendu extrêmes vides, got MVP=%q LVP=%q", got.MVPXUID, got.LVPXUID)
			}
		})
	}
}
