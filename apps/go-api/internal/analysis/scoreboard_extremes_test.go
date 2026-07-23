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

// TestComputeMVPLVP_ExcludesMechanicKills verrouille B2 : les frags issus de
// mécaniques (assassinat, charge spartane / shoulder_bash, coup au sol /
// ground_pound) ne comptent PAS dans la valeur de frags qui départage le MVP/LVP.
//
// Scénario à 2 joueurs, seules les colonnes Frags (w2) et Passes (w1) sont
// actives (deaths/perfect nuls et identiques → ignorées, le reste nil → NaN).
//   - a : 20 frags bruts, dont 18 mécaniques (6+6+6) → 2 frags "de tir" retenus.
//   - b : 8 frags nus.
//
// Cas « brut » (contrôle, sans mécaniques) : a tient la meilleure cellule Frags
// (20 > 8) et rafle le MVP. Cas « exclusion » : b devient le meilleur tireur
// (8 > 2) et bascule MVP — c'est l'inversion qui prouve l'exclusion.
func TestComputeMVPLVP_ExcludesMechanicKills(t *testing.T) {
	ip := func(v int) *int { return &v }
	const xuidA, xuidB = "2533274800000001", "2533274800000002"

	cases := []struct {
		name    string
		rows    []domain.ScoreboardRaw
		wantMVP string
		wantLVP string
	}{
		{
			name: "brut (contrôle) : a MVP sur ses 20 frags",
			rows: []domain.ScoreboardRaw{
				{XUID: xuidA, Kills: 20, Assists: 10},
				{XUID: xuidB, Kills: 8, Assists: 5},
			},
			wantMVP: xuidA,
			wantLVP: xuidB,
		},
		{
			name: "exclusion : 18 frags mécaniques retirés → b MVP",
			rows: []domain.ScoreboardRaw{
				{XUID: xuidA, Kills: 20, Assists: 10,
					AssassinationKills: ip(6), ShoulderBashKills: ip(6), GroundPoundKills: ip(6)},
				{XUID: xuidB, Kills: 8, Assists: 5},
			},
			wantMVP: xuidB,
			wantLVP: xuidA,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeMVPLVP(tc.rows)
			if got.MVPXUID != tc.wantMVP {
				t.Errorf("MVP = %q, want %q", got.MVPXUID, tc.wantMVP)
			}
			if got.LVPXUID != tc.wantLVP {
				t.Errorf("LVP = %q, want %q", got.LVPXUID, tc.wantLVP)
			}
		})
	}
}
