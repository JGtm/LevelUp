package teammates

// teammates_squad_impact_temoin_test.go — TÉMOIN D'ÉQUIVALENCE du tableau d'impact par session
// avec l'ancienne matrice d'impact et l'ancien historique par soirée, écrit AVANT leur suppression
// : mêmes fixtures, mêmes rôles, mêmes comptes et mêmes points.

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/squadimpact"
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

// TestImpactSessions_TemoinAncienHistoriqueEtAncienneMatrice : sur les deux sessions S1 (précédente)
// et S2 (affichée) de la fixture, plus un match de la population sans libellé de session,
//   - par session et par joueur, les comptes par rôle et les points du nouveau bloc égalent ceux de
//     l'ancien historique, et ses victoires comptent celles de l'ancien historique ;
//   - les totaux égalent l'ancienne matrice restreinte aux matchs porteurs d'un libellé ;
//   - l'ancienne matrice sur TOUTE la population en diffère (le match sans libellé sort du tableau).
func TestImpactSessions_TemoinAncienHistoriqueEtAncienneMatrice(t *testing.T) {
	repo, all := soireesFixture()
	sansSession := ajouterMatchSansSession(repo)
	svc := &TeammatesService{repo: repo, titleSlug: "halo_infinite", gamertag: "Main"}
	tm := []domain.TeammateRow{{Gamertag: "Ally", XUID: strPtr("x_ally")}}
	population := append(lignesDeSession(all, "S2"), sansSession)
	timeline := append(slices.Clone(all), sansSession)
	in := impactEscouade{
		rows: population, timeline: timeline, evenings: soireesDImpact(population, timeline, true),
		mainXUID: "x_main", selected: []string{"Ally"}, teammates: tm, allies: repo.allyRows,
	}
	squad := resolveSquadScope(in.rows, svc.gamertag, in.mainXUID, in.teammates).gtByXUID
	roles := svc.rolesDesMatchs(context.Background(), in, groupesDImpact(in.rows, in.evenings), squad)
	players := joueursDeLEscouade(svc.gamertag, in.selected)

	history := squadimpact.BuildHistory(squadimpact.HistoryInput{Evenings: in.evenings, Players: players, Roles: roles})
	sessions := sessionsDImpact(in, players, squad, roles)
	if history == nil || sessions == nil || len(history.Evenings) != 2 || len(sessions.Sessions) != 2 {
		t.Fatalf("deux sessions attendues : historique %+v, bloc %+v", history, sessions)
	}

	// Par session (le bloc : la plus récente d'abord ; l'historique : la plus ancienne d'abord).
	for i, s := range sessions.Sessions {
		ev := history.Evenings[len(history.Evenings)-1-i]
		victoires := 0
		for _, m := range s.Matches {
			if m.Outcome == domain.OutcomeWin {
				victoires++
			}
		}
		if s.SessionLabel != ev.SessionLabel || s.StartTime != ev.StartTime || len(s.Matches) != ev.Matches || victoires != ev.Wins {
			t.Fatalf("session %d : bloc %s %s %d matchs %d victoires, historique %s %s %d matchs %d victoires",
				i, s.SessionLabel, s.StartTime, len(s.Matches), victoires, ev.SessionLabel, ev.StartTime, ev.Matches, ev.Wins)
		}
		for j, p := range s.Players {
			old := ev.Players[j]
			attendu := map[string]int{}
			for _, r := range old.Roles {
				attendu[r.Role] = r.Count
			}
			if p.Player != old.Player || p.Points != old.Points || !equalCounts(comptesParRole(p.Counts), attendu) {
				t.Errorf("%s / %s : bloc %v pour %v points, historique %v pour %v points",
					s.SessionLabel, p.Player, comptesParRole(p.Counts), p.Points, attendu, old.Points)
			}
		}
	}

	// Totaux = ancienne matrice restreinte aux matchs porteurs d'un libellé (les lignes de `all`).
	restreinte := assemblerMatrice(all, players, roles)
	complete := assemblerMatrice(append(slices.Clone(all), sansSession), players, roles)
	if restreinte == nil || complete == nil {
		t.Fatalf("matrices nulles : restreinte %v, complète %v", restreinte, complete)
	}
	score := func(m *domain.SquadImpactMatrix) map[string]domain.SquadImpactPlayerSummary {
		out := map[string]domain.SquadImpactPlayerSummary{}
		for _, p := range m.Players {
			out[p.Player] = p
		}
		return out
	}
	rest, comp := score(restreinte), score(complete)
	differe := false
	for _, tot := range sessions.Totals {
		old := rest[tot.Player]
		attendu := map[string]int{}
		for _, c := range old.Counts {
			if c.Count != 0 {
				attendu[c.BadgeKey] = c.Count
			}
		}
		if tot.Points != old.Score || !equalCounts(comptesParRole(tot.Counts), attendu) {
			t.Errorf("total de %s : bloc %v pour %v points, matrice restreinte %v pour %v points",
				tot.Player, comptesParRole(tot.Counts), tot.Points, attendu, old.Score)
		}
		differe = differe || comp[tot.Player].Score != old.Score
	}
	if !differe {
		t.Error("la matrice sur toute la population doit différer des totaux : le match sans libellé porte des rôles")
	}

	// Le témoin ne compare pas du vide : des rôles et des matchs joués existent bien.
	if len(comptesParRole(sessions.Totals[0].Counts)) == 0 || sessions.Totals[0].MatchesPlayed != 6 ||
		len(comptesParRole(sessions.Totals[1].Counts)) == 0 || sessions.Totals[1].MatchesPlayed != 6 {
		t.Errorf("totaux vides ou matchs joués ≠ 6 : %+v", sessions.Totals)
	}

	// D-9 : le match sans libellé est compté et journalisé une fois.
	if n := matchsSansSession(population); n != 1 {
		t.Errorf("matchs sans libellé : %d, attendu 1", n)
	}
	logs := withCapturedLogs(t, func() { journaliserMatchsSansSession(context.Background(), population) })
	if strings.Count(logs, "teammates_squad_impact_sessions_unlabelled_matches") != 1 || !strings.Contains(logs, `"matches":1`) {
		t.Errorf("une ligne de journal attendue avec 1 match, obtenu :\n%s", logs)
	}
	if logs := withCapturedLogs(t, func() { journaliserMatchsSansSession(context.Background(), all) }); logs != "" {
		t.Errorf("aucun match sans libellé : aucun journal attendu, obtenu :\n%s", logs)
	}
}

func equalCounts(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
