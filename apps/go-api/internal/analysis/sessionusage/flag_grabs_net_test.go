package sessionusage

// flag_grabs_net_test.go — les dénominateurs de la grandeur, et ses deux silences.

import "testing"

func rowsCTF() []FlagGrabsNetRow {
	return []FlagGrabsNetRow{
		// m1 : le joueur suivi (camp 0) et un adversaire (camp 1).
		{MatchID: "m1", XUID: "moi", Raw: 9, Net: 3, WindowMS: 1500},
		{MatchID: "m1", XUID: "allie", Raw: 4, Net: 4, WindowMS: 1500},
		{MatchID: "m1", XUID: "adverse", Raw: 7, Net: 2, WindowMS: 1500},
		// m2 : le joueur suivi seul.
		{MatchID: "m2", XUID: "moi", Raw: 5, Net: 5, WindowMS: 1500},
	}
}

func contexteCTF() (map[string]int, map[string]map[string]int) {
	playerTeam := map[string]int{"m1": 0, "m2": 0}
	teamOf := map[string]map[string]int{
		"m1": {"moi": 0, "allie": 0, "adverse": 1},
		"m2": {"moi": 0},
	}
	return playerTeam, teamOf
}

func TestComputeFlagGrabsNet_TotauxEtDenominateurs(t *testing.T) {
	pt, to := contexteCTF()
	// TROIS matchs à objectif dans le scope, DEUX mesurés : l'écart est la couverture
	// du film, et le bloc doit le dire au lieu de le taire.
	b := ComputeFlagGrabsNet(rowsCTF(), "moi", 3, pt, to)
	if b == nil {
		t.Fatal("bloc nil alors que des prises existent")
	}
	if b.PlayerTotal != 8 || b.PlayerRawTotal != 14 {
		t.Errorf("joueur = (%d net, %d brut), want (8, 14)", b.PlayerTotal, b.PlayerRawTotal)
	}
	if b.TeamTotal != 12 || b.TeamRawTotal != 18 {
		t.Errorf("camp = (%d net, %d brut), want (12, 18)", b.TeamTotal, b.TeamRawTotal)
	}
	if b.LobbyTotal != 14 || b.LobbyRawTotal != 25 {
		t.Errorf("lobby = (%d net, %d brut), want (14, 25)", b.LobbyTotal, b.LobbyRawTotal)
	}
	if b.MatchesMeasured != 2 || b.MatchesWithObjectives != 3 {
		t.Errorf("couverture = %d/%d, want 2/3", b.MatchesMeasured, b.MatchesWithObjectives)
	}
	if b.WindowSeconds != 1.5 {
		t.Errorf("fenêtre = %v, want 1.5", b.WindowSeconds)
	}
	if b.PlayerShareOfTeamPct == nil || *b.PlayerShareOfTeamPct < 66.6 || *b.PlayerShareOfTeamPct > 66.7 {
		t.Errorf("part = %v, want ~66,7 %% (8 sur 12)", b.PlayerShareOfTeamPct)
	}
}

// TestComputeFlagGrabsNet_DeuxFenetresNePublientAucune — un scope partiellement re-projeté
// après un changement de règle n'a pas UNE fenêtre ; en publier une mentirait sur l'autre.
func TestComputeFlagGrabsNet_DeuxFenetresNePublientAucune(t *testing.T) {
	pt, to := contexteCTF()
	rows := rowsCTF()
	rows[3].WindowMS = 3000
	b := ComputeFlagGrabsNet(rows, "moi", 3, pt, to)
	if b.WindowSeconds != 0 {
		t.Errorf("fenêtre = %v, want 0 (le scope en mêle deux)", b.WindowSeconds)
	}
}

// TestComputeFlagGrabsNet_ScopeSansPriseEstOmis — pas de bloc à zéro : un scope sans drapeau
// ou sans film lu n'a rien à montrer.
func TestComputeFlagGrabsNet_ScopeSansPriseEstOmis(t *testing.T) {
	pt, to := contexteCTF()
	if b := ComputeFlagGrabsNet(nil, "moi", 3, pt, to); b != nil {
		t.Errorf("bloc publié sur un scope vide : %+v", b)
	}
}

// TestComputeFlagGrabsNet_CampSansPriseNaPasDePart — pas de part sur un dénominateur nul, et
// surtout pas « 0 % ».
func TestComputeFlagGrabsNet_CampSansPriseNaPasDePart(t *testing.T) {
	rows := []FlagGrabsNetRow{{MatchID: "m1", XUID: "adverse", Raw: 3, Net: 1, WindowMS: 1500}}
	pt := map[string]int{"m1": 0}
	to := map[string]map[string]int{"m1": {"adverse": 1}}
	b := ComputeFlagGrabsNet(rows, "moi", 1, pt, to)
	if b == nil {
		t.Fatal("bloc nil alors qu'une prise existe dans le lobby")
	}
	if b.PlayerShareOfTeamPct != nil {
		t.Errorf("part publiée (%v) alors que le camp n'a pris aucun drapeau", *b.PlayerShareOfTeamPct)
	}
	if b.TeamTotal != 0 || b.LobbyTotal != 1 {
		t.Errorf("camp=%d lobby=%d, want 0 et 1", b.TeamTotal, b.LobbyTotal)
	}
}

// TestComputeFlagGrabsNet_CampInconnuNeCompteNiPourNiContre — un match dont le camp du joueur
// n'est pas connu (FFA, historique incomplet) ne peut pas alimenter un total d'équipe.
func TestComputeFlagGrabsNet_CampInconnuNeCompteNiPourNiContre(t *testing.T) {
	rows := []FlagGrabsNetRow{{MatchID: "mX", XUID: "moi", Raw: 4, Net: 2, WindowMS: 1500}}
	b := ComputeFlagGrabsNet(rows, "moi", 1, map[string]int{}, map[string]map[string]int{})
	if b.TeamTotal != 0 {
		t.Errorf("camp = %d, want 0 (camp inconnu)", b.TeamTotal)
	}
	if b.PlayerTotal != 2 || b.LobbyTotal != 2 {
		t.Errorf("joueur=%d lobby=%d, want 2 et 2", b.PlayerTotal, b.LobbyTotal)
	}
}
