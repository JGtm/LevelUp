//go:build cgo

package skill

// skill_v2_fixtures_test.go — jeu de matchs des tests du lot perf L6 (pré-filtre du
// filigrane sous lecteur, rafale unique, parité des écritures) : un match =
// match_registry + participants, dans une base au schéma shadowSchemaDDL.

import (
	"database/sql"
	"testing"
	"time"
)

// Paires de mode du jeu et leur chaîne LUSR (classifier Infinite câblé par TestMain).
const (
	pairSlayer  = "Arena:Slayer on Bazaar"            // arena_slayer
	pairCTF     = "Arena:CTF on Recharge"             // arena_objectif
	pairBTB     = "BTB:Slayer on Fragmentation"       // btb
	pairFiesta  = "Fiesta:Slayer on Bazaar"           // chaos
	pairNoChain = "Gruntpocalypse:Slayer on Deadlock" // aucune chaîne LUSR
)

// fixtureBase est l'origine des horaires. Chaque match a sa minute propre : aucune
// égalité de start_time, donc un ordre SQL des candidats total et identique d'une
// base à l'autre.
var fixtureBase = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

func fixtureAt(minute int) time.Time { return fixtureBase.Add(time.Duration(minute) * time.Minute) }

// shadowSeat est une ligne match_participants (outcome : code Halo, 1 nul,
// 2 victoire, 3 défaite, 4 abandon).
type shadowSeat struct {
	xuid          string
	team, outcome int
	kills, deaths int
}

// shadowFixture est un match du jeu.
type shadowFixture struct {
	id, pair string
	start    time.Time
	seats    []shadowSeat
}

// seats2v2 : owner+mate (équipe 0) contre opp1+opp2 (équipe 1). ownerOutcome est
// l'issue du camp du owner ; l'autre camp reçoit l'issue miroir.
func seats2v2(ownerOutcome, ownerKills, ownerDeaths int) []shadowSeat {
	mirror := map[int]int{1: 1, 2: 3, 3: 2}[ownerOutcome]
	return []shadowSeat{
		{"owner", 0, ownerOutcome, ownerKills, ownerDeaths},
		{"mate", 0, ownerOutcome, 11, 9},
		{"opp1", 1, mirror, 9, 12},
		{"opp2", 1, mirror, 7, 13},
	}
}

// seatsThreeTeams : trois équipes humaines → non notable (lusrSkipNonTwoTeam).
func seatsThreeTeams() []shadowSeat {
	return []shadowSeat{
		{"owner", 0, 2, 12, 7}, {"mate", 0, 2, 9, 8},
		{"opp1", 1, 3, 8, 10}, {"opp2", 2, 3, 6, 10},
	}
}

// seatsThreeVsOne : trois contre un → non notable (lusrSkipImbalance).
func seatsThreeVsOne() []shadowSeat {
	return []shadowSeat{
		{"owner", 0, 2, 14, 3}, {"mate", 0, 2, 12, 4}, {"mate2", 0, 2, 10, 5},
		{"opp1", 1, 3, 4, 20},
	}
}

// seatsOwnerQuit : le owner abandonne (outcome 4) → issue non notable
// (lusrSkipNonTwoTeam).
func seatsOwnerQuit() []shadowSeat {
	return []shadowSeat{
		{"owner", 0, 4, 3, 5}, {"mate", 0, 3, 7, 9},
		{"opp1", 1, 2, 11, 6}, {"opp2", 1, 2, 9, 7},
	}
}

// seedShadowFixtures insère les matchs dans la base (schéma shadowSchemaDDL).
func seedShadowFixtures(t *testing.T, db *sql.DB, fixtures ...shadowFixture) {
	t.Helper()
	for _, f := range fixtures {
		if _, err := db.Exec(`INSERT INTO match_registry
			(match_id, start_time, start_time_utc, pair_name, is_ranked, is_firefight, duration_seconds)
			VALUES (?, ?, ?, ?, FALSE, FALSE, 600)`, f.id, f.start, f.start, f.pair); err != nil {
			t.Fatalf("seed match_registry(%s): %v", f.id, err)
		}
		for _, s := range f.seats {
			if _, err := db.Exec(`INSERT INTO match_participants
				(match_id, xuid, team_id, outcome, kills, deaths) VALUES (?, ?, ?, ?, ?, ?)`,
				f.id, s.xuid, s.team, s.outcome, s.kills, s.deaths); err != nil {
				t.Fatalf("seed participant(%s,%s): %v", f.id, s.xuid, err)
			}
		}
	}
}

// countStateRows compte TOUTES les lignes écrites dans player_skill_state_v2 (table
// brute : on mesure les écritures, pas l'état courant).
func countStateRows(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM player_skill_state_v2`).Scan(&n); err != nil {
		t.Fatalf("count player_skill_state_v2: %v", err)
	}
	return n
}
