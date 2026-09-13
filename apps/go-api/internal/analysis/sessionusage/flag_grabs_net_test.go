package sessionusage

// flag_grabs_net_test.go — les dénominateurs de la grandeur, et ses silences.
//
// Trois de ces cas viennent de la revue adversariale du 2026-09-13 et fixent des défauts
// MESURÉS, pas des hypothèses : la part qui dépassait 100 %, le camp 0 confondu avec un xuid
// absent, et la couverture qui comptait les matchs d'autres familles.

import "testing"

func rowsCTF() []FlagGrabsNetRow {
	return []FlagGrabsNetRow{
		// m1 : le joueur suivi (camp 0), un allié (camp 0) et un adversaire (camp 1).
		{MatchID: "m1", XUID: "moi", Raw: 9, Net: 3, Openings: 25, WindowMS: 1500},
		{MatchID: "m1", XUID: "allie", Raw: 4, Net: 4, Openings: 25, WindowMS: 1500},
		{MatchID: "m1", XUID: "adverse", Raw: 7, Net: 2, Openings: 25, WindowMS: 1500},
		// m2 : le joueur suivi seul.
		{MatchID: "m2", XUID: "moi", Raw: 5, Net: 5, Openings: 6, WindowMS: 1500},
	}
}

func entreeCTF(rows []FlagGrabsNetRow, famille int) FlagGrabsNetInput {
	return FlagGrabsNetInput{
		Rows:              rows,
		PlayerXUID:        "moi",
		MatchesFlagFamily: famille,
		PlayerTeam:        map[string]int{"m1": 0, "m2": 0},
		TeamOf: map[string]map[string]int{
			"m1": {"moi": 0, "allie": 0, "adverse": 1},
			"m2": {"moi": 0},
		},
	}
}

func TestComputeFlagGrabsNet_TotauxEtDenominateurs(t *testing.T) {
	// TROIS matchs de la famille drapeau dans la session, DEUX mesurés : l'écart est la
	// couverture du film, et le bloc doit le dire au lieu de le taire.
	b := ComputeFlagGrabsNet(entreeCTF(rowsCTF(), 3))
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
	if b.MatchesMeasured != 2 || b.MatchesWithFlagFamily != 3 {
		t.Errorf("couverture = %d/%d, want 2/3", b.MatchesMeasured, b.MatchesWithFlagFamily)
	}
	// Les ouvertures se comptent UNE FOIS PAR MATCH, pas une fois par ligne.
	if b.OpeningsTotal != 31 {
		t.Errorf("ouvertures = %d, want 31 (25 + 6, une fois par match)", b.OpeningsTotal)
	}
	if b.WindowSeconds != 1.5 {
		t.Errorf("fenêtre = %v, want 1.5", b.WindowSeconds)
	}
	if b.PlayerShareOfTeamPct == nil || *b.PlayerShareOfTeamPct < 66.6 || *b.PlayerShareOfTeamPct > 66.7 {
		t.Errorf("part = %v, want ~66,7 %% (8 sur 12)", b.PlayerShareOfTeamPct)
	}
}

// TestComputeFlagGrabsNet_PartNeDepassePasCent — LE DÉFAUT DE LA REVUE. Le joueur joue deux
// matchs, mais un seul a un camp connu : compter ses prises sur les deux et celles de son camp
// sur un seul faisait une part de 267 %.
func TestComputeFlagGrabsNet_PartNeDepassePasCent(t *testing.T) {
	in := FlagGrabsNetInput{
		Rows: []FlagGrabsNetRow{
			{MatchID: "m1", XUID: "moi", Raw: 4, Net: 3, Openings: 10, WindowMS: 1500},
			{MatchID: "m1", XUID: "allie", Raw: 2, Net: 2, Openings: 10, WindowMS: 1500},
			// m2 : camp INCONNU (absent de PlayerTeam).
			{MatchID: "m2", XUID: "moi", Raw: 12, Net: 10, Openings: 20, WindowMS: 1500},
		},
		PlayerXUID:        "moi",
		MatchesFlagFamily: 2,
		PlayerTeam:        map[string]int{"m1": 0},
		TeamOf:            map[string]map[string]int{"m1": {"moi": 0, "allie": 0}},
	}
	b := ComputeFlagGrabsNet(in)
	if b.PlayerTotal != 13 {
		t.Errorf("total joueur (tout le scope mesuré) = %d, want 13", b.PlayerTotal)
	}
	// Le couple COMPARABLE ne porte que m1.
	if b.PlayerTeamScopeTotal != 3 || b.TeamTotal != 5 {
		t.Errorf("couple comparable = (%d, %d), want (3, 5)", b.PlayerTeamScopeTotal, b.TeamTotal)
	}
	if b.MatchesTeamKnown != 1 {
		t.Errorf("matchs à camp connu = %d, want 1", b.MatchesTeamKnown)
	}
	if b.PlayerShareOfTeamPct == nil || *b.PlayerShareOfTeamPct <= 0 || *b.PlayerShareOfTeamPct > 100 {
		t.Fatalf("part = %v, want une valeur dans ]0 ; 100]", b.PlayerShareOfTeamPct)
	}
	if *b.PlayerShareOfTeamPct < 59.9 || *b.PlayerShareOfTeamPct > 60.1 {
		t.Errorf("part = %v, want 60 %% (3 sur 5)", *b.PlayerShareOfTeamPct)
	}
}

// TestComputeFlagGrabsNet_XuidAbsentDeTeamOfNestPasLeCampZero — le camp 0 est un camp
// LÉGITIME : un xuid absent de la table des camps ne doit pas en hériter.
func TestComputeFlagGrabsNet_XuidAbsentDeTeamOfNestPasLeCampZero(t *testing.T) {
	in := FlagGrabsNetInput{
		Rows: []FlagGrabsNetRow{
			{MatchID: "m1", XUID: "moi", Raw: 4, Net: 3, Openings: 10, WindowMS: 1500},
			// `fantome` n'est PAS dans TeamOf : son camp est inconnu, pas « 0 ».
			{MatchID: "m1", XUID: "fantome", Raw: 9, Net: 9, Openings: 10, WindowMS: 1500},
		},
		PlayerXUID:        "moi",
		MatchesFlagFamily: 1,
		PlayerTeam:        map[string]int{"m1": 0},
		TeamOf:            map[string]map[string]int{"m1": {"moi": 0}},
	}
	b := ComputeFlagGrabsNet(in)
	if b.TeamTotal != 3 {
		t.Errorf("camp = %d, want 3 — le xuid absent de TeamOf a été compté dans l'équipe 0", b.TeamTotal)
	}
	if b.LobbyTotal != 12 {
		t.Errorf("lobby = %d, want 12 (le joueur inconnu reste dans le lobby)", b.LobbyTotal)
	}
}

// TestComputeFlagGrabsNet_DeuxFenetresNePublientAucune — un scope partiellement re-projeté
// après un changement de règle n'a pas UNE fenêtre ; en publier une mentirait sur l'autre.
func TestComputeFlagGrabsNet_DeuxFenetresNePublientAucune(t *testing.T) {
	rows := rowsCTF()
	rows[3].WindowMS = 3000
	b := ComputeFlagGrabsNet(entreeCTF(rows, 3))
	if b.WindowSeconds != 0 {
		t.Errorf("fenêtre = %v, want 0 (le scope en mêle deux)", b.WindowSeconds)
	}
}

// TestComputeFlagGrabsNet_ScopeSansPriseEstOmis — pas de bloc à zéro : un scope sans drapeau
// ou sans film lu n'a rien à montrer.
func TestComputeFlagGrabsNet_ScopeSansPriseEstOmis(t *testing.T) {
	if b := ComputeFlagGrabsNet(entreeCTF(nil, 3)); b != nil {
		t.Errorf("bloc publié sur un scope vide : %+v", b)
	}
}

// TestComputeFlagGrabsNet_CampSansPriseNaPasDePart — pas de part sur un dénominateur nul, et
// surtout pas « 0 % ».
func TestComputeFlagGrabsNet_CampSansPriseNaPasDePart(t *testing.T) {
	in := FlagGrabsNetInput{
		Rows:              []FlagGrabsNetRow{{MatchID: "m1", XUID: "adverse", Raw: 3, Net: 1, Openings: 4, WindowMS: 1500}},
		PlayerXUID:        "moi",
		MatchesFlagFamily: 1,
		PlayerTeam:        map[string]int{"m1": 0},
		TeamOf:            map[string]map[string]int{"m1": {"adverse": 1}},
	}
	b := ComputeFlagGrabsNet(in)
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

// TestComputeFlagGrabsNet_ZeroMesureEstUneMesure — un joueur du roster à zéro prise a une
// ligne, et elle compte dans les dénominateurs comme n'importe quelle autre.
func TestComputeFlagGrabsNet_ZeroMesureEstUneMesure(t *testing.T) {
	in := FlagGrabsNetInput{
		Rows: []FlagGrabsNetRow{
			{MatchID: "m1", XUID: "moi", Raw: 0, Net: 0, Openings: 8, WindowMS: 1500},
			{MatchID: "m1", XUID: "allie", Raw: 6, Net: 4, Openings: 8, WindowMS: 1500},
		},
		PlayerXUID:        "moi",
		MatchesFlagFamily: 1,
		PlayerTeam:        map[string]int{"m1": 0},
		TeamOf:            map[string]map[string]int{"m1": {"moi": 0, "allie": 0}},
	}
	b := ComputeFlagGrabsNet(in)
	if b == nil {
		t.Fatal("bloc nil alors que le match est mesuré")
	}
	if b.MatchesMeasured != 1 {
		t.Errorf("matchs mesurés = %d, want 1 — un match où le joueur suivi n'a rien pris reste mesuré", b.MatchesMeasured)
	}
	if b.PlayerShareOfTeamPct == nil || *b.PlayerShareOfTeamPct != 0 {
		t.Errorf("part = %v, want 0 %% — le camp a pris, le joueur non : c'est une mesure", b.PlayerShareOfTeamPct)
	}
}
