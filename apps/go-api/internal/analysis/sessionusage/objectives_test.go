package sessionusage

// objectives_test.go — le bloc objectifs : rôles agrégés toutes familles, blocs
// par famille avec rôles filtrés (dérivés de narrative, jamais d'une liste
// locale), lignes d'escouade, parités, omission sans ligne.

import (
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
)

func objectivesInputDeTest() ObjectivesInput {
	return ObjectivesInput{
		PlayerXUID: "P",
		SquadXUIDs: []string{"A"},
		Rows: []ObjectiveRow{
			{MatchID: "m1", XUID: "P", Family: narrative.FamilyCTF, Take: 2, Defend: 1, HoldSeconds: 30},
			{MatchID: "m1", XUID: "A", Family: narrative.FamilyCTF, Take: 1},
			{MatchID: "m1", XUID: "E1", Family: narrative.FamilyCTF, Take: 3, Defend: 2, HoldSeconds: 40},
			{MatchID: "m2", XUID: "P", Family: narrative.FamilyExtraction, Take: 1},
			{MatchID: "m2", XUID: "E1", Family: narrative.FamilyExtraction, Defend: 2},
		},
		PlayerTeam: map[string]int{"m1": 0, "m2": 0},
		TeamOf: map[string]map[string]int{
			"m1": {"P": 0, "A": 0, "E1": 1},
			"m2": {"P": 0, "E1": 1},
		},
		TeamSize:  map[string]int{"m1": 2, "m2": 4},
		LobbySize: map[string]int{"m1": 4, "m2": 8},
	}
}

func findRole(t *testing.T, roles []domain.SessionObjectiveRoleMetric, role narrative.ObjectiveRole) domain.SessionObjectiveRoleMetric {
	t.Helper()
	for i := range roles {
		if roles[i].Role == string(role) {
			return roles[i]
		}
	}
	t.Fatalf("rôle %q absent : %+v", role, roles)
	return domain.SessionObjectiveRoleMetric{}
}

func TestComputeObjectives_RolesToutesFamilles(t *testing.T) {
	out := ComputeObjectives(objectivesInputDeTest())
	if out == nil {
		t.Fatal("bloc nil, attendu un bloc")
	}
	if out.MatchesWithObjectives != 2 {
		t.Errorf("matchs à objectifs = %d, attendu 2", out.MatchesWithObjectives)
	}
	take := findRole(t, out.Roles, narrative.ObjectiveRoleTake)
	// Lobby = 2+1+3+1 = 7 ; joueur = 3 (m1 + m2) ; camp = 4 (P 3 + A 1).
	if take.PlayerTotal != 3 || !closeTo(take.TeamTotal, 4) || take.LobbyTotal != 7 {
		t.Errorf("prendre (joueur, camp, lobby) = (%v, %v, %v), attendu (3, 4, 7)",
			take.PlayerTotal, take.TeamTotal, take.LobbyTotal)
	}
	if !closeTo(take.TeamShareOfLobbyPct, 100*4.0/7) || !closeTo(take.PlayerShareOfTeamPct, 75) {
		t.Errorf("parts prendre = (%v, %v), attendu (57.14, 75)",
			take.TeamShareOfLobbyPct, take.PlayerShareOfTeamPct)
	}
	hold := findRole(t, out.Roles, narrative.ObjectiveRoleHold)
	if !hold.IsDuration {
		t.Error("tenir doit porter IsDuration (secondes, pas comptes)")
	}
	if hold.PlayerTotal != 30 || hold.LobbyTotal != 70 {
		t.Errorf("tenir (joueur, lobby) = (%v, %v), attendu (30, 70)", hold.PlayerTotal, hold.LobbyTotal)
	}
	// Parités du scope objectifs : effectifs moyens (2+4)/2=3 et (4+8)/2=6.
	if !closeTo(out.TeamParityPct, 100/3.0) || !closeTo(out.LobbyParityPct, 100/6.0) {
		t.Errorf("parités = (%v, %v), attendu (33.33, 16.67)", out.TeamParityPct, out.LobbyParityPct)
	}
}

func TestComputeObjectives_FamillesEtRolesFiltresDepuisNarrative(t *testing.T) {
	out := ComputeObjectives(objectivesInputDeTest())
	if len(out.Families) != 2 {
		t.Fatalf("familles = %+v, attendu ctf + extraction", out.Families)
	}
	var ctf, extraction *domain.SessionObjectiveFamilyBlock
	for i := range out.Families {
		switch out.Families[i].Family {
		case string(narrative.FamilyCTF):
			ctf = &out.Families[i]
		case string(narrative.FamilyExtraction):
			extraction = &out.Families[i]
		}
	}
	if ctf == nil || extraction == nil {
		t.Fatalf("familles = %+v", out.Families)
	}
	if len(ctf.Roles) != 3 {
		t.Errorf("ctf : %d rôles, attendu 3 (prendre, défendre, tenir)", len(ctf.Roles))
	}
	// Extraction n'a AUCUNE colonne de durée (narrative.ObjectiveFamilyHoldColumns)
	// : le rôle « tenir » doit être ABSENT du bloc, pas servi à zéro.
	if len(extraction.Roles) != 2 {
		t.Errorf("extraction : %d rôles (%+v), attendu 2 (tenir absent)", len(extraction.Roles), extraction.Roles)
	}
	for _, r := range extraction.Roles {
		if r.Role == string(narrative.ObjectiveRoleHold) {
			t.Error("extraction publie un rôle tenir — zéro structurel interdit")
		}
	}
	// Le scope d'une famille est SES matchs : take ctf lobby = 6, take extraction lobby = 1.
	if take := findRole(t, ctf.Roles, narrative.ObjectiveRoleTake); take.LobbyTotal != 6 {
		t.Errorf("prendre ctf lobby = %v, attendu 6", take.LobbyTotal)
	}
	if take := findRole(t, extraction.Roles, narrative.ObjectiveRoleTake); take.LobbyTotal != 1 {
		t.Errorf("prendre extraction lobby = %v, attendu 1", take.LobbyTotal)
	}
}

func TestComputeObjectives_LignesEscouade(t *testing.T) {
	out := ComputeObjectives(objectivesInputDeTest())
	take := findRole(t, out.Roles, narrative.ObjectiveRoleTake)
	if len(take.Squad) != 1 || take.Squad[0].XUID != "A" {
		t.Fatalf("lignes escouade = %+v, attendu une ligne pour A", take.Squad)
	}
	l := take.Squad[0]
	if l.Total != 1 || !closeTo(l.ShareOfTeamPct, 25) {
		t.Errorf("ligne A = %+v, attendu total 1, part équipe 25 %%", l)
	}
	if l.Per10Min != nil {
		t.Error("les rôles d'objectif n'ont pas de cadence — Per10Min doit rester nil")
	}
}

// C3 : un scope objectifs dont AUCUN match n'a de camp connu (FFA, participants
// absents) ne publie aucune grandeur d'équipe — nil, jamais &0.
func TestComputeObjectives_CampInconnuPartoutPartsEquipeNil(t *testing.T) {
	in := ObjectivesInput{
		PlayerXUID: "P",
		Rows: []ObjectiveRow{
			{MatchID: "m1", XUID: "P", Family: narrative.FamilyOddball, HoldSeconds: 30, Take: 1},
			{MatchID: "m1", XUID: "E1", Family: narrative.FamilyOddball, HoldSeconds: 40},
		},
		PlayerTeam: map[string]int{}, // camp inconnu partout
		TeamOf:     map[string]map[string]int{},
		TeamSize:   map[string]int{},
		LobbySize:  map[string]int{"m1": 4},
	}
	out := ComputeObjectives(in)
	if out == nil {
		t.Fatal("bloc nil, attendu un bloc (les grandeurs joueur/lobby restent mesurables)")
	}
	hold := findRole(t, out.Roles, narrative.ObjectiveRoleHold)
	if hold.TeamTotal != nil {
		t.Errorf("camp = %v, attendu nil (aucun camp connu — jamais un 0 inventé)", *hold.TeamTotal)
	}
	if hold.TeamShareOfLobbyPct != nil || hold.PlayerShareOfTeamPct != nil {
		t.Errorf("parts d'équipe = (%v, %v), attendu nil (jamais &0)",
			hold.TeamShareOfLobbyPct, hold.PlayerShareOfTeamPct)
	}
	if !closeTo(hold.PlayerShareOfLobbyPct, 100*30.0/70) {
		t.Errorf("joueur/lobby = %v, attendu 42.86 (le lobby, lui, mesure)", hold.PlayerShareOfLobbyPct)
	}
}

// C1 (objectifs) : scope mêlant un match à camp connu et un match à camp
// inconnu — les grandeurs d'équipe se calculent sur le seul match à camp connu
// (numérateurs ET dénominateurs), sinon joueur/équipe dépasserait 100 %.
func TestComputeObjectives_ScopeMixteResteDansLeScope(t *testing.T) {
	in := ObjectivesInput{
		PlayerXUID: "P",
		Rows: []ObjectiveRow{
			{MatchID: "m1", XUID: "P", Family: narrative.FamilyCTF, Take: 1},
			{MatchID: "m1", XUID: "E1", Family: narrative.FamilyCTF, Take: 1},
			{MatchID: "ffa", XUID: "P", Family: narrative.FamilyCTF, Take: 5},
			{MatchID: "ffa", XUID: "X", Family: narrative.FamilyCTF, Take: 5},
		},
		PlayerTeam: map[string]int{"m1": 0},
		TeamOf:     map[string]map[string]int{"m1": {"P": 0, "E1": 1}},
		TeamSize:   map[string]int{"m1": 1},
		LobbySize:  map[string]int{"m1": 2, "ffa": 5},
	}
	out := ComputeObjectives(in)
	take := findRole(t, out.Roles, narrative.ObjectiveRoleTake)
	if take.PlayerTotal != 6 || take.LobbyTotal != 12 {
		t.Errorf("joueur/lobby = (%v, %v), attendu (6, 12) — scope complet", take.PlayerTotal, take.LobbyTotal)
	}
	if !closeTo(take.TeamTotal, 1) {
		t.Errorf("camp = %v, attendu 1 (m1 seul)", take.TeamTotal)
	}
	if !closeTo(take.PlayerShareOfTeamPct, 100) {
		t.Errorf("joueur/équipe = %v, attendu 100 %% — un croisement de scopes donnerait 600 %%", take.PlayerShareOfTeamPct)
	}
	if !closeTo(take.TeamShareOfLobbyPct, 50) {
		t.Errorf("camp/lobby = %v, attendu 50 %% (1/2 sur m1) — dilué il vaudrait 8.33 %%", take.TeamShareOfLobbyPct)
	}
}

func TestComputeObjectives_SansLigneLeBlocEstOmis(t *testing.T) {
	if out := ComputeObjectives(ObjectivesInput{PlayerXUID: "P"}); out != nil {
		t.Errorf("bloc = %+v, attendu nil (session sans mode à objectif)", out)
	}
}
