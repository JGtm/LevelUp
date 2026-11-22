package squadimpact

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"levelup/go-api/internal/domain"
)

// seance — les matchs de la session label avec l'issue donnée (2 victoire, 3 défaite) par rang.
func seance(label string, day, hour int, outcomes ...int) []EveningMatch {
	out := make([]EveningMatch, 0, len(outcomes))
	for i, o := range outcomes {
		out = append(out, EveningMatch{
			MatchID: fmt.Sprintf("%s-%d", label, i), SessionLabel: label,
			StartTime: at(day, hour, 10*i), Outcome: o,
		})
	}
	return out
}

func compte(counts []domain.SquadImpactSessionsRoleCount, role string) int {
	for _, c := range counts {
		if c.Role == role {
			return c.Count
		}
	}
	return -1
}

// TestBuildSessions_CumulParSession : sessions de la plus récente à la plus ancienne, matchs
// chronologiques malgré une entrée en désordre, comptes dans RoleOrder (zéros compris), points au
// barème, rôles match par match dans l'ordre de RoleOrder, totaux sommés sur les sessions.
func TestBuildSessions_CumulParSession(t *testing.T) {
	s1 := seance("S1", 1, 19, 2, 3, 2)
	s2 := seance("S2", 2, 19, 2, 2)
	// Entrée en désordre : le match le plus récent de S1 d'abord.
	s1[0], s1[2] = s1[2], s1[0]
	evenings := SelectEvenings(slices.Concat(s1, s2), nil)
	roles := map[string][]Attribution{
		"S1-0": {{Player: "Main", Role: RoleKamikaze}, {Player: "Main", Role: RoleFirstBlood}},
		"S1-1": {{Player: "Main", Role: RoleKamikaze}, {Player: "A", Role: RoleLastCasualty}},
		"S1-2": {{Player: "Inconnu", Role: RoleFirstBlood}},
		"S2-0": {{Player: "A", Role: RoleClutchFinisher}},
	}
	allied := map[string]map[string]bool{
		"S1-0": {"Main": true, "A": true}, "S1-1": {"Main": true, "A": true}, "S1-2": {"Main": true},
		"S2-0": {"Main": true, "A": true}, "S2-1": {"Main": true},
	}
	got := BuildSessions(SessionsInput{Evenings: evenings, Players: []string{"Main", "A", "B"}, Roles: roles, Allied: allied})
	if got == nil || len(got.Sessions) != 2 {
		t.Fatalf("deux sessions attendues, obtenu %+v", got)
	}
	if got.Sessions[0].SessionLabel != "S2" || got.Sessions[1].SessionLabel != "S1" {
		t.Fatalf("ordre %s, %s : attendu S2 puis S1 (la plus récente d'abord)", got.Sessions[0].SessionLabel, got.Sessions[1].SessionLabel)
	}
	if !reflect.DeepEqual(got.RoleOrder, RoleOrder()) || len(got.Scale) != len(stackOrder) ||
		got.Scale[0] != (domain.SquadImpactSessionsRoleWeight{Role: RoleClutchFinisher, Points: 2}) {
		t.Errorf("role_order %v, barème %v", got.RoleOrder, got.Scale)
	}
	one := got.Sessions[1]
	if one.StartTime != "2026-09-01T19:00:00Z" {
		t.Errorf("début %s", one.StartTime)
	}
	wantMatches := []domain.SquadImpactSessionsMatch{{MatchID: "S1-0", Outcome: 2}, {MatchID: "S1-1", Outcome: 3}, {MatchID: "S1-2", Outcome: 2}}
	if !reflect.DeepEqual(one.Matches, wantMatches) {
		t.Errorf("matchs de S1 %+v, attendu %+v (chronologiques)", one.Matches, wantMatches)
	}
	main := one.Players[0]
	if main.Player != "Main" || main.MatchesPlayed != 3 || main.Points != 1 {
		t.Errorf("Main sur S1 = %+v, attendu 3 matchs joués, net +1", main)
	}
	if len(main.Counts) != len(RoleOrder()) || compte(main.Counts, RoleFirstBlood) != 1 || compte(main.Counts, RoleKamikaze) != 2 ||
		compte(main.Counts, RoleThief) != 0 {
		t.Errorf("comptes de Main %+v", main.Counts)
	}
	wantPer := []domain.SquadImpactSessionsPerMatch{
		{MatchID: "S1-0", Roles: []string{RoleFirstBlood, RoleKamikaze}},
		{MatchID: "S1-1", Roles: []string{RoleKamikaze}},
	}
	if !reflect.DeepEqual(main.PerMatch, wantPer) {
		t.Errorf("rôles par match de Main %+v, attendu %+v (RoleOrder dans un match, S1-2 absent)", main.PerMatch, wantPer)
	}
	if a := one.Players[1]; a.MatchesPlayed != 2 || a.Points != -2 {
		t.Errorf("A sur S1 = %+v, attendu 2 matchs joués, net -2", a)
	}
	if b := one.Players[2]; b.MatchesPlayed != 0 || b.Points != 0 || len(b.PerMatch) != 0 || b.PerMatch == nil {
		t.Errorf("B sur S1 = %+v, attendu aucun match, per_match vide non nul", b)
	}
	// Totaux : somme des deux sessions.
	tm, ta := got.Totals[0], got.Totals[1]
	if tm.Player != "Main" || tm.MatchesPlayed != 5 || tm.Points != 1 || compte(tm.Counts, RoleKamikaze) != 2 {
		t.Errorf("total de Main %+v, attendu 5 matchs joués, net +1", tm)
	}
	if ta.MatchesPlayed != 3 || ta.Points != 0 || compte(ta.Counts, RoleClutchFinisher) != 1 || compte(ta.Counts, RoleLastCasualty) != 1 {
		t.Errorf("total de A %+v, attendu 3 matchs joués, net 0 (+2 -2)", ta)
	}
}

// TestBuildSessions_JoueurAbsentEtRoleVautPresence : un joueur hors de l'équipe alliée n'a pas
// joué le match ; un rôle attribué dans un match y vaut présence, de sorte que 0 match joué
// implique net et comptes nuls.
func TestBuildSessions_JoueurAbsentEtRoleVautPresence(t *testing.T) {
	evenings := SelectEvenings(seance("S1", 1, 19, 2, 2), nil)
	roles := map[string][]Attribution{"S1-0": {{Player: "A", Role: RoleSilentHero}}}
	allied := map[string]map[string]bool{"S1-0": {"Main": true}, "S1-1": {"Main": true}}
	got := BuildSessions(SessionsInput{Evenings: evenings, Players: []string{"Main", "A"}, Roles: roles, Allied: allied})
	if got == nil {
		t.Fatal("bloc attendu")
	}
	main, a := got.Sessions[0].Players[0], got.Sessions[0].Players[1]
	if main.MatchesPlayed != 2 || main.Points != 0 {
		t.Errorf("Main = %+v, attendu 2 matchs, net 0", main)
	}
	if a.MatchesPlayed != 1 || a.Points != 1.5 {
		t.Errorf("A = %+v, attendu 1 match (le rôle vaut présence), net +1,5", a)
	}
}

// TestBuildSessions_SessionSansRole : une session sans rôle reste publiée (matchs, zéros) tant
// qu'une autre en a ; sans aucun rôle sur l'ensemble, le bloc se retire ; sans joueur ni session
// aussi.
func TestBuildSessions_SessionSansRole(t *testing.T) {
	evenings := SelectEvenings(slices.Concat(seance("S1", 1, 19, 2), seance("S2", 2, 19, 2, 2, 2)), nil)
	roles := map[string][]Attribution{"S1-0": {{Player: "Main", Role: RoleFirstBlood}}}
	got := BuildSessions(SessionsInput{Evenings: evenings, Players: []string{"Main"}, Roles: roles})
	if got == nil || len(got.Sessions) != 2 {
		t.Fatalf("deux sessions attendues, obtenu %+v", got)
	}
	vide := got.Sessions[0]
	if vide.SessionLabel != "S2" || len(vide.Matches) != 3 || vide.Players[0].Points != 0 ||
		vide.Players[0].PerMatch == nil || len(vide.Players[0].Counts) != len(RoleOrder()) {
		t.Errorf("session sans rôle %+v", vide)
	}
	if h := BuildSessions(SessionsInput{Evenings: evenings, Players: []string{"Main"}}); h != nil {
		t.Errorf("aucun rôle : attendu nil, obtenu %+v", h)
	}
	if h := BuildSessions(SessionsInput{Players: []string{"Main"}, Roles: roles}); h != nil {
		t.Errorf("aucune session : attendu nil, obtenu %+v", h)
	}
	if h := BuildSessions(SessionsInput{Evenings: evenings, Roles: roles}); h != nil {
		t.Errorf("aucun joueur : attendu nil, obtenu %+v", h)
	}
}

// TestBuildSessions_EgalitesEtPlusDeOnzeSessions : à égalité l'ordre d'entrée des joueurs est
// gardé ; sur quatorze sessions SelectEvenings en garde onze et le bloc les publie toutes, la
// plus récente d'abord, totaux sommés sur ces onze seulement.
func TestBuildSessions_EgalitesEtPlusDeOnzeSessions(t *testing.T) {
	var all []EveningMatch
	for d := 1; d <= 14; d++ {
		all = append(all, seance(fmt.Sprintf("D%02d", d), d, 19, 2, 2)...)
	}
	evenings := SelectEvenings(all, all)
	roles := map[string][]Attribution{}
	for _, e := range evenings {
		roles[e.Matches[0].MatchID] = []Attribution{{Player: "Main", Role: RoleFirstBlood}, {Player: "A", Role: RoleFirstBlood}}
	}
	// Un match de D01 (hors des onze) porte aussi un rôle : il ne doit pas entrer dans les totaux.
	roles["D01-0"] = []Attribution{{Player: "Main", Role: RoleFirstBlood}}
	got := BuildSessions(SessionsInput{Evenings: evenings, Players: []string{"Main", "A"}, Roles: roles})
	if got == nil || len(got.Sessions) != domain.SquadImpactSessionsMax {
		t.Fatalf("%v sessions, attendu %d", got, domain.SquadImpactSessionsMax)
	}
	if got.Sessions[0].SessionLabel != "D14" || got.Sessions[10].SessionLabel != "D04" {
		t.Errorf("de %s à %s, attendu de D14 à D04", got.Sessions[0].SessionLabel, got.Sessions[10].SessionLabel)
	}
	if got.Totals[0].Points != 22 || got.Totals[1].Points != 22 || compte(got.Totals[0].Counts, RoleFirstBlood) != 11 {
		t.Errorf("totaux %+v / %+v, attendu 22 points et 11 Premier sang chacun", got.Totals[0], got.Totals[1])
	}
	if got.Players[0] != "Main" || got.Totals[0].Player != "Main" || got.Totals[1].Player != "A" {
		t.Errorf("l'ordre des joueurs d'entrée est gardé : %v", got.Players)
	}
}

// TestBuildSessions_RolesDuBaremeDansLOrdre : tout rôle du barème figure dans RoleOrder, donc
// dans les comptes (sinon un rôle compté en points serait invisible dans les colonnes).
func TestBuildSessions_RolesDuBaremeDansLOrdre(t *testing.T) {
	for role := range weights {
		if !slices.Contains(roleOrder, role) {
			t.Errorf("le rôle %s du barème est absent de RoleOrder", role)
		}
	}
}
