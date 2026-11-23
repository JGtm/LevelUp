package teammates

// teammates_squad_impact_temoin_test.go — TÉMOIN du tableau d'impact par session : les valeurs
// ci-dessous sont celles que l'ancien historique par soirée et l'ancienne matrice d'impact
// rendaient sur ces fixtures, quand le nouveau calcul leur était confronté avant leur suppression
// (comptes par rôle et points par session et par joueur ; totaux = matrice restreinte aux matchs
// à libellé de session). Elles se lisent maintenant en valeurs figées.

import (
	"context"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// ajouterMatchSansSession ajoute à la fixture un match de la population SANS libellé de session
// (frags, morts et équipe alliée comme les autres matchs de la fixture) et rend sa ligne.
func ajouterMatchSansSession(repo *mockSquadRepo) domain.SquadMatchRow {
	id := "U-m0"
	row := domain.SquadMatchRow{
		MatchID: id, StartTime: time.Date(2026, 9, 2, 21, 0, 0, 0, time.UTC), Outcome: domain.OutcomeWin,
		Kills: 10, Deaths: 5, Assists: 3, TimePlayedSecs: 600, IsWithFriends: true, MapUI: "Aquarius",
	}
	repo.squadRows = append(repo.squadRows, row)
	repo.impactRows = append(repo.impactRows,
		domain.ImpactEventRow{MatchID: id, XUID: "x_main", EventType: "kill", TimeMS: 10_000},
		domain.ImpactEventRow{MatchID: id, XUID: "x_ally", EventType: "death", TimeMS: 20_000},
		domain.ImpactEventRow{MatchID: id, XUID: "x_ally", EventType: "kill", TimeMS: 300_000},
		domain.ImpactEventRow{MatchID: id, XUID: "x_main", EventType: "kill", TimeMS: 350_000},
		domain.ImpactEventRow{MatchID: id, XUID: "x_main", EventType: "death", TimeMS: 351_000},
	)
	repo.allyRows = append(repo.allyRows,
		domain.AllyParticipant{MatchID: id, XUID: "x_main", Gamertag: "Main", Kills: 10, Deaths: 5, Assists: 3, Outcome: domain.OutcomeWin},
		domain.AllyParticipant{MatchID: id, XUID: "x_ally", Gamertag: "Ally", Kills: 4, Deaths: 7, Assists: 1, Outcome: domain.OutcomeWin},
		domain.AllyParticipant{MatchID: id, XUID: "x_ns", Gamertag: "NS", Kills: 2, Deaths: 3, Assists: 0, Outcome: domain.OutcomeWin},
	)
	return row
}

// comptesParRole indexe les comptes d'un joueur par rôle, comptes nuls ignorés.
func comptesParRole(counts []domain.SquadImpactSessionsRoleCount) map[string]int {
	out := map[string]int{}
	for _, c := range counts {
		if c.Count != 0 {
			out[c.Role] = c.Count
		}
	}
	return out
}

// témoinParSession : les valeurs de l'ancien historique, identiques sur S1 et S2 (fixture
// symétrique), par joueur.
var temoinParSession = map[string]struct {
	points float64
	counts map[string]int
}{
	"Main": {9.5, map[string]int{"clutch_finisher": 2, "first_blood": 3, "top_killer": 3, "last_casualty": 1, "kamikaze": 3}},
	"Ally": {-4, map[string]int{"thief": 1, "first_group_death": 3, "last_group_kill": 3}},
}

// témoinTotaux : les valeurs de l'ancienne matrice restreinte aux matchs à libellé de session
// (S1 + S2), par joueur.
var temoinTotaux = map[string]struct {
	points float64
	counts map[string]int
}{
	"Main": {19, map[string]int{"clutch_finisher": 4, "first_blood": 6, "top_killer": 6, "last_casualty": 2, "kamikaze": 6}},
	"Ally": {-8, map[string]int{"thief": 2, "first_group_death": 6, "last_group_kill": 6}},
}

// TestImpactSessions_TemoinValeursFigees : sur les sessions S1 (précédente) et S2 (affichée) de la
// fixture, plus un match de la population sans libellé de session (qui valait +4,5 à Main et -1 à
// Ally dans l'ancienne matrice sur toute la population : 23,5 et -9),
//   - par session et par joueur, comptes par rôle, points, matchs joués, début et issues égalent
//     l'ancien historique ;
//   - les totaux égalent l'ancienne matrice restreinte aux matchs à libellé : le match sans
//     libellé n'entre pas dans le tableau.
func TestImpactSessions_TemoinValeursFigees(t *testing.T) {
	repo, all := soireesFixture()
	sansSession := ajouterMatchSansSession(repo)
	svc := &TeammatesService{repo: repo, titleSlug: "halo_infinite", gamertag: "Main"}
	tm := []domain.TeammateRow{{Gamertag: "Ally", XUID: strPtr("x_ally")}}
	population := append(lignesDeSession(all, "S2"), sansSession)
	timeline := append(slices.Clone(all), sansSession)
	sessions := svc.buildSquadImpact(context.Background(), impactEscouade{
		rows: population, timeline: timeline, evenings: soireesDImpact(population, timeline, true),
		mainXUID: "x_main", selected: []string{"Ally"}, teammates: tm, allies: repo.allyRows,
	})
	if sessions == nil || len(sessions.Sessions) != 2 {
		t.Fatalf("deux sessions attendues, obtenu %+v", sessions)
	}
	for i, label := range []string{"S2", "S1"} {
		s := sessions.Sessions[i]
		victoires := 0
		for _, m := range s.Matches {
			if m.Outcome == domain.OutcomeWin {
				victoires++
			}
		}
		wantStart := map[string]string{"S1": "2026-09-01T19:00:00Z", "S2": "2026-09-02T19:00:00Z"}[label]
		if s.SessionLabel != label || s.StartTime != wantStart || len(s.Matches) != 3 || victoires != 2 {
			t.Fatalf("session %d : %s %s, %d matchs, %d victoires ; attendu %s %s, 3 matchs, 2 victoires",
				i, s.SessionLabel, s.StartTime, len(s.Matches), victoires, label, wantStart)
		}
		for _, p := range s.Players {
			want := temoinParSession[p.Player]
			if p.Points != want.points || !maps.Equal(comptesParRole(p.Counts), want.counts) || p.MatchesPlayed != 3 {
				t.Errorf("%s / %s : %v points, %v, %d matchs joués ; attendu %v points, %v, 3 matchs joués",
					label, p.Player, p.Points, comptesParRole(p.Counts), p.MatchesPlayed, want.points, want.counts)
			}
		}
	}
	for _, tot := range sessions.Totals {
		want := temoinTotaux[tot.Player]
		if tot.Points != want.points || !maps.Equal(comptesParRole(tot.Counts), want.counts) || tot.MatchesPlayed != 6 {
			t.Errorf("total de %s : %v points, %v, %d matchs joués ; attendu %v points, %v, 6 matchs joués",
				tot.Player, tot.Points, comptesParRole(tot.Counts), tot.MatchesPlayed, want.points, want.counts)
		}
	}

	// Le match sans libellé est compté et journalisé une fois.
	if n := matchsSansSession(population); n != 1 {
		t.Errorf("matchs sans libellé : %d, attendu 1", n)
	}
	logs := withCapturedLogs(t, func() { journaliserMatchsSansSession(context.Background(), population) })
	if strings.Count(logs, "teammates_squad_impact_sessions_unlabelled_matches") != 1 || !strings.Contains(logs, `"matches":1`) {
		t.Errorf("une ligne de journal attendue avec 1 match, obtenu : %s", logs)
	}
	if logs := withCapturedLogs(t, func() { journaliserMatchsSansSession(context.Background(), all) }); logs != "" {
		t.Errorf("aucun match sans libellé : aucun journal attendu, obtenu : %s", logs)
	}
}
