package teammates

// teammates_squad_impact_shields_test.go — le rôle « Os de verre » dans le tableau d'impact par
// session : un match non mesuré n'en donne pas.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

// boucliersDeLaFixture pose les bris de soireesFixture : S1-m0 Ally (9 bris contre 5), S1-m1 égalité
// à 6 bris départagée par les recharges (Ally 1 contre 2), S1-m2 non mesuré, S2-m0 l'allié hors
// escouade (20 bris), S2-m1 Main (8 contre 3), S2-m2 aucun bris.
func boucliersDeLaFixture() []domain.MatchShieldRow {
	return []domain.MatchShieldRow{
		{MatchID: "S1-m0", XUID: "x_ally", Breaks: 9}, {MatchID: "S1-m0", XUID: "x_main", Breaks: 5},
		{MatchID: "S1-m1", XUID: "x_ally", Breaks: 6, Recharges: 1}, {MatchID: "S1-m1", XUID: "x_main", Breaks: 6, Recharges: 2},
		{MatchID: "S2-m0", XUID: "x_ns", Breaks: 20}, {MatchID: "S2-m0", XUID: "x_ally", Breaks: 2},
		{MatchID: "S2-m1", XUID: "x_main", Breaks: 8}, {MatchID: "S2-m1", XUID: "x_ally", Breaks: 3},
		{MatchID: "S2-m2", XUID: "x_main"}, {MatchID: "S2-m2", XUID: "x_ally", Recharges: 4},
	}
}

// compteOsDeVerre rend le compte du rôle « Os de verre » d'un joueur sur une session, -1 si le
// joueur ou la session manque.
func compteOsDeVerre(sessions *domain.SquadImpactSessions, label, joueur string) int {
	for _, s := range sessions.Sessions {
		if s.SessionLabel != label {
			continue
		}
		for _, p := range s.Players {
			if p.Player == joueur {
				return compteDuRole(p.Counts, "glass_jaw")
			}
		}
	}
	return -1
}

// compteDuRole rend le compte d'un rôle dans une liste de comptes, -1 si le rôle manque.
func compteDuRole(counts []domain.SquadImpactSessionsRoleCount, role string) int {
	for _, c := range counts {
		if c.Role == role {
			return c.Count
		}
	}
	return -1
}

// TestBuildSquadImpact_OsDeVerre : le rôle tombe sur l'allié au plus de bris de chaque match mesuré
// (départage par les recharges) et n'est pas rendu à l'escouade quand il tombe sur un allié hors
// escouade ni sans bris allié ; les sessions le comptent par session et dans les totaux.
func TestBuildSquadImpact_OsDeVerre(t *testing.T) {
	repo, all := soireesFixture()
	repo.shields = boucliersDeLaFixture()
	svc := &TeammatesService{repo: repo, titleSlug: "halo_infinite", gamertag: "Main"}
	tm := []domain.TeammateRow{{Gamertag: "Ally", XUID: strPtr("x_ally")}}
	s2 := lignesDeSession(all, "S2")
	sessions := svc.buildSquadImpact(context.Background(), impactEscouade{
		rows: s2, timeline: all, evenings: soireesDImpact(s2, all, true),
		mainXUID: "x_main", selected: []string{"Ally"}, teammates: tm, allies: repo.allyRows,
	})
	if sessions == nil || len(sessions.Sessions) != 2 {
		t.Fatalf("deux sessions attendues, obtenu %+v", sessions)
	}
	wantAlly := map[string]int{"S1": 2, "S2": 0}
	wantMain := map[string]int{"S1": 0, "S2": 1}
	for _, label := range []string{"S1", "S2"} {
		if got := compteOsDeVerre(sessions, label, "Ally"); got != wantAlly[label] {
			t.Errorf("%s : Os de verre d'Ally %d, attendu %d", label, got, wantAlly[label])
		}
		if got := compteOsDeVerre(sessions, label, "Main"); got != wantMain[label] {
			t.Errorf("%s : Os de verre de Main %d, attendu %d", label, got, wantMain[label])
		}
	}
	if got := compteDuRole(sessions.Totals[1].Counts, "glass_jaw"); sessions.Totals[1].Player != "Ally" || got != 2 {
		t.Errorf("total d'Ally %+v : Os de verre %d, attendu 2", sessions.Totals[1], got)
	}
}

// TestBuildSquadImpact_OsDeVerreSansMesureOuEnEchec : sans mesure de bouclier, ou quand la lecture
// échoue, aucun « Os de verre » n'est attribué et le reste du tableau se publie.
func TestBuildSquadImpact_OsDeVerreSansMesureOuEnEchec(t *testing.T) {
	for nom, config := range map[string]func(*mockSquadRepo){
		"sans mesure":      func(*mockSquadRepo) {},
		"lecture en échec": func(r *mockSquadRepo) { r.shields = boucliersDeLaFixture(); r.shieldsErr = errors.New("boom") },
	} {
		repo, all := soireesFixture()
		config(repo)
		sessions := dedieSessions(t, repo, lignesDeSession(all, "S1"))
		if sessions == nil {
			t.Fatalf("%s : bloc nul", nom)
		}
		for _, j := range []string{"Main", "Ally"} {
			if got := compteOsDeVerre(sessions, "S1", j); got != 0 {
				t.Errorf("%s : Os de verre de %s %d, attendu 0", nom, j, got)
			}
		}
	}
}
