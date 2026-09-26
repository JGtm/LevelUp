package sessionusage

// usage_overview_test.go — LE BLOC « SERVI OU GÂCHÉ » AU GRAIN PÉRIODE (étapes E5
// et E6 du PLAN_EQUIPEMENT_GACHIS_2026-09-09) : une ligne par famille pour la
// Synthèse, une ligne par joueur suivi pour l'Escouade, et les comptes exclusifs
// des deux donuts (décisions P9, P10, P11).

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// overviewDeTest — deux matchs mesurés, dont UN SEUL à camp connu.
//
//	m1 (camp connu) : P (moi) prend 3 murs, en pose 1, en lâche 1 -> gardé 1
//	                  P prend AUSSI 1 capteur et en CONSOMME la charge -> utilisé 1
//	                  F (ami suivi) prend 2 murs, en pose 2
//	                  A (allié NON suivi) prend 1 mur, le lâche
//	                  E1 (eux) prend 4 murs, les lâche tous
//	                  socles d'arme : P 2, F 1, A 3, E1 5
//	m2 (FFA, camp INCONNU) : P prend 1 capteur et le garde ; X en consomme 2
//
// LE CAPTEUR EST LÀ POUR DISTINGUER LES DEUX CANAUX (correction C1, 2026-09-10) :
// il n'engendre AUCUNE pièce, son « utilisé » se lit donc sur les CONSOMMATIONS
// (`SpentByFamily`) et jamais sur les poses. Une fixture qui ne porterait que du
// mur — seule famille encore lue sur ses poses — laisserait passer les deux règles.
func overviewDeTest() OverviewInput {
	return OverviewInput{
		PlayerXUID:  "P",
		FriendXUIDs: []string{"F"},
		Matches: []MatchInput{
			{
				MatchID: "m1", Measured: true,
				PlayerTeam: intp(0),
				TeamOf:     map[string]int{"P": 0, "F": 0, "A": 0, "E1": 1},
				TeamSize:   3, LobbySize: 4,
				Players: []PlayerRow{
					{
						MatchID: "m1", XUID: "P", PadPickups: 2,
						DeployedByFamily: map[string]int{"wall": 1},
						SpentByFamily:    map[string]int{"sensor": 1},
						TakenByFamily:    map[string]int{"wall": 3, "sensor": 1},
						DroppedByFamily:  map[string]int{"wall": 1},
						KeptByFamily:     map[string]int{"wall": 1},
					},
					{
						MatchID: "m1", XUID: "F", PadPickups: 1,
						DeployedByFamily: map[string]int{"wall": 2},
						TakenByFamily:    map[string]int{"wall": 2},
					},
					{
						MatchID: "m1", XUID: "A", PadPickups: 3,
						TakenByFamily:   map[string]int{"wall": 1},
						DroppedByFamily: map[string]int{"wall": 1},
					},
					{
						MatchID: "m1", XUID: "E1", PadPickups: 5,
						TakenByFamily:   map[string]int{"wall": 4},
						DroppedByFamily: map[string]int{"wall": 4},
					},
				},
			},
			{
				MatchID: "m2", Measured: true,
				TeamOf: map[string]int{}, LobbySize: 2,
				Players: []PlayerRow{
					{
						MatchID: "m2", XUID: "P", PadPickups: 1,
						TakenByFamily: map[string]int{"sensor": 1},
						KeptByFamily:  map[string]int{"sensor": 1},
					},
					{
						MatchID: "m2", XUID: "X",
						SpentByFamily: map[string]int{"sensor": 2},
						TakenByFamily: map[string]int{"sensor": 2},
					},
				},
			},
		},
	}
}

func findFamily(t *testing.T, lines []domain.EquipmentUsageFamilyLine, key string) domain.EquipmentUsageFamilyLine {
	t.Helper()
	for _, l := range lines {
		if l.FamilyKey == key {
			return l
		}
	}
	t.Fatalf("famille %q absente du bloc (%d lignes)", key, len(lines))
	return domain.EquipmentUsageFamilyLine{}
}

// TestOverview_FamillesTroisIssuesEtDeuxTauxDeReference — la ligne de famille porte
// les MÊMES grandeurs que la page Sessions, et les deux références M'EXCLUENT (P7).
func TestOverview_FamillesTroisIssuesEtDeuxTauxDeReference(t *testing.T) {
	out := ComputeUsageOverview(overviewDeTest())
	if !out.Available {
		t.Fatalf("bloc indisponible : %q", out.UnavailableReason)
	}
	if out.MatchesMeasured != 2 || out.MatchesTotal != 2 {
		t.Errorf("couverture = %d/%d, attendu 2/2", out.MatchesMeasured, out.MatchesTotal)
	}
	wall := findFamily(t, out.Families, "wall")
	if wall.Used != 1 || wall.Kept != 1 || wall.Dropped != 1 || wall.Taken != 3 {
		t.Errorf("mur = (utilisé %v, gardé %v, lâché %v, pris %v), attendu (1, 1, 1, 3)",
			wall.Used, wall.Kept, wall.Dropped, wall.Taken)
	}
	// Le reste de mon équipe = F (2 posés sur 2) + A (0 sur 1) = 2/3.
	if !closeTo(wall.TeammatesUsedRatePct, 200.0/3) {
		t.Errorf("taux du reste de mon équipe = %v, attendu 66,67 %%", wall.TeammatesUsedRatePct)
	}
	// Eux = E1 : 0 utilisé sur 4. Un 0 % MESURÉ, pas un nil.
	if !closeTo(wall.OpponentsUsedRatePct, 0) {
		t.Errorf("taux de eux = %v, attendu 0 %%", wall.OpponentsUsedRatePct)
	}
}

// TestOverview_FamillesTrieesDuPlusPrisAuMoinsPris — décision P9 : l'axe est en
// comptes, les lignes se lisent de haut en bas par volume.
func TestOverview_FamillesTrieesDuPlusPrisAuMoinsPris(t *testing.T) {
	out := ComputeUsageOverview(overviewDeTest())
	if len(out.Families) != 2 {
		t.Fatalf("familles = %d, attendu 2 (mur, capteur)", len(out.Families))
	}
	if out.Families[0].FamilyKey != "wall" || out.Families[1].FamilyKey != "sensor" {
		t.Errorf("ordre = %q puis %q, attendu wall puis sensor",
			out.Families[0].FamilyKey, out.Families[1].FamilyKey)
	}
}

// TestOverview_LesQuatrePartsFontLeLobby — décisions P10/P11 : les parts sont
// EXCLUSIVES et leur somme est le lobby. Le Go publie des COMPTES, jamais de part
// en pourcentage (le front fait les parts).
func TestOverview_LesQuatrePartsFontLeLobby(t *testing.T) {
	out := ComputeUsageOverview(overviewDeTest())
	eq := out.EquipmentParties
	if eq == nil {
		t.Fatal("comptes du donut équipement absents")
	}
	// Moi = 3 murs (posé/gardé/lâché) + 1 capteur CONSOMMÉ : la charge consommée
	// est un objet utilisé, même sans aucune pose (correction C1).
	if eq.Player != 4 || eq.Friends != 2 || eq.RestOfTeam != 1 || eq.Opponents != 4 {
		t.Errorf("parts équipement = (moi %v, amis %v, reste %v, eux %v), attendu (4, 2, 1, 4)",
			eq.Player, eq.Friends, eq.RestOfTeam, eq.Opponents)
	}
	if somme := eq.Player + eq.Friends + eq.RestOfTeam + eq.Opponents; somme != eq.LobbyTotal {
		t.Errorf("somme des parts %v != lobby %v", somme, eq.LobbyTotal)
	}
	pad := out.WeaponPadParties
	if pad == nil {
		t.Fatal("comptes du donut armes spéciales absents")
	}
	if pad.Player != 2 || pad.Friends != 1 || pad.RestOfTeam != 3 || pad.Opponents != 5 {
		t.Errorf("parts socles = (moi %v, amis %v, reste %v, eux %v), attendu (2, 1, 3, 5)",
			pad.Player, pad.Friends, pad.RestOfTeam, pad.Opponents)
	}
	if somme := pad.Player + pad.Friends + pad.RestOfTeam + pad.Opponents; somme != pad.LobbyTotal {
		t.Errorf("somme des parts %v != lobby %v", somme, pad.LobbyTotal)
	}
	if len(eq.ByFriend) != 1 || eq.ByFriend[0].XUID != "F" || eq.ByFriend[0].Value != 2 {
		t.Errorf("ventilation par ami = %+v, attendu [{F 2}]", eq.ByFriend)
	}
}

// TestOverview_UneLignePar JoueurSuivi — étape E6.1 : le joueur de la route EN TÊTE,
// puis les coéquipiers suivis dans l'ordre reçu. Les issues y sont TOUTES FAMILLES
// CONFONDUES et portent sur TOUT le scope mesuré (le capteur du match FFA compte).
func TestOverview_UneLigneParJoueurSuivi(t *testing.T) {
	out := ComputeUsageOverview(overviewDeTest())
	if len(out.Players) != 2 {
		t.Fatalf("lignes joueur = %d, attendu 2 (P puis F)", len(out.Players))
	}
	moi := out.Players[0]
	if moi.XUID != "P" {
		t.Fatalf("première ligne = %q, attendu le joueur de la route P", moi.XUID)
	}
	// 1 mur posé + 1 capteur CONSOMMÉ = 2 utilisés ; 5 pris (3 murs + 2 capteurs).
	if moi.Used != 2 || moi.Kept != 2 || moi.Dropped != 1 || moi.Taken != 5 {
		t.Errorf("moi = (utilisé %v, gardé %v, lâché %v, pris %v), attendu (2, 2, 1, 5)",
			moi.Used, moi.Kept, moi.Dropped, moi.Taken)
	}
	if moi.PadPickups != 3 {
		t.Errorf("socles pris par moi = %v, attendu 3", moi.PadPickups)
	}
	ami := out.Players[1]
	if ami.XUID != "F" || ami.Used != 2 || ami.Kept != 0 || ami.Dropped != 0 {
		t.Errorf("ami = %+v, attendu F (utilisé 2, gardé 0, lâché 0)", ami)
	}
	if ami.PadPickups != 1 {
		t.Errorf("socles pris par l'ami = %v, attendu 1", ami.PadPickups)
	}
}

// TestOverview_FFAIntegral_AucunePartInventee — un scope entièrement FFA n'a ni
// « reste de mon équipe » ni « eux » : les deux donuts sont ABSENTS, jamais servis
// à zéro. Les familles, elles, restent lues (elles ne dépendent pas du camp).
func TestOverview_FFAIntegral_AucunePartInventee(t *testing.T) {
	in := overviewDeTest()
	in.Matches = in.Matches[1:] // ne garde que le match FFA
	out := ComputeUsageOverview(in)
	if out.EquipmentParties != nil || out.WeaponPadParties != nil {
		t.Errorf("donuts servis sur un scope sans camp connu : %+v / %+v",
			out.EquipmentParties, out.WeaponPadParties)
	}
	sensor := findFamily(t, out.Families, "sensor")
	if sensor.Kept != 1 || sensor.Taken != 1 {
		t.Errorf("capteur = (gardé %v, pris %v), attendu (1, 1)", sensor.Kept, sensor.Taken)
	}
	if sensor.TeammatesUsedRatePct != nil || sensor.OpponentsUsedRatePct != nil {
		t.Errorf("taux de référence servis sans camp connu : %v / %v",
			sensor.TeammatesUsedRatePct, sensor.OpponentsUsedRatePct)
	}
}

// TestOverview_AucunMatchMesure — bloc PRÉSENT et disponible, mais vide : « matchs
// mesurés 0/N » doit s'afficher, la couverture des films n'est jamais totale.
func TestOverview_AucunMatchMesure(t *testing.T) {
	out := ComputeUsageOverview(OverviewInput{
		PlayerXUID: "P",
		Matches:    []MatchInput{{MatchID: "m1"}, {MatchID: "m2"}},
	})
	if !out.Available {
		t.Fatalf("bloc indisponible : %q", out.UnavailableReason)
	}
	if out.MatchesMeasured != 0 || out.MatchesTotal != 2 {
		t.Errorf("couverture = %d/%d, attendu 0/2", out.MatchesMeasured, out.MatchesTotal)
	}
	if len(out.Families) != 0 || out.EquipmentParties != nil || len(out.Players) != 0 {
		t.Error("grandeurs servies sans aucun match mesuré")
	}
}

// TestOverview_ScopeDesDonutsEstCeluiDuCampCONNU — règle de scope de computeMetric
// appliquée aux donuts : numérateurs ET dénominateurs sur les seuls matchs à camp
// connu. Les capteurs du match FFA n'entrent donc PAS dans le donut, alors qu'ils
// entrent dans la ligne de famille et dans la ligne du joueur.
//
// Le total du donut PIN AUSSI LA RÈGLE D'USAGE (correction C1) : le capteur consommé
// par P dans m1 y compte pour 1, alors que ses poses valent zéro. Lu sur les poses,
// ce donut afficherait 10.
func TestOverview_ScopeDesDonutsEstCeluiDuCampConnu(t *testing.T) {
	out := ComputeUsageOverview(overviewDeTest())
	if out.EquipmentParties.LobbyTotal != 11 {
		t.Errorf("lobby du donut = %v, attendu 11 (les 3 objets du match FFA sont hors scope, "+
			"le capteur consommé de m1 compte)", out.EquipmentParties.LobbyTotal)
	}
	if out.WeaponPadParties.LobbyTotal != 11 {
		t.Errorf("lobby du donut socles = %v, attendu 11", out.WeaponPadParties.LobbyTotal)
	}
}
