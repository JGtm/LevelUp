package narrative

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// ctfTeam rend `n` joueurs du camp 0 qui ont tous la même action (une capture), plus un adversaire.
func ctfTeam(n int) []ObjectiveTeamRow {
	rows := make([]ObjectiveTeamRow, 0, n+1)
	for i := 0; i < n; i++ {
		rows = append(rows, ObjectiveTeamRow{
			XUID: "j" + string(rune('a'+i)), TeamID: 0, Family: FamilyCTF, TimePlayedSeconds: 600,
			Values: map[string]float64{"flag_captures": 1},
		})
	}
	return append(rows, ObjectiveTeamRow{
		XUID: "adverse", TeamID: 1, Family: FamilyCTF, TimePlayedSeconds: 600,
		Values: map[string]float64{"flag_captures": 9},
	})
}

// La part équitable donne 50, quelle que soit la taille de l'équipe (4 contre 4 ou 8 contre 8).
func TestObjectiveMatchScore_PartEquitableDonne50(t *testing.T) {
	for _, n := range []int{4, 8} {
		got, ok := ObjectiveMatchScore(ctfTeam(n), "ja")
		if !ok || !almostEqual(got, 50) {
			t.Errorf("équipe de %d : note %v ok=%v, want 50", n, got, ok)
		}
	}
}

// Le double de la part équitable, ou plus, sature à 100 ; l'axe ne dépasse jamais 100. Une capture
// pèse 5 points : à 4 joueurs, la part équitable est 25 % et la saturation tombe à 50 % de l'équipe.
func TestObjectiveMatchScore_SatureA100(t *testing.T) {
	cases := []struct {
		name     string
		captures float64
		want     float64
	}{
		// 10 points sur 25 : part 0,4 x 4 / 2 = 0,8.
		{"sous la borne", 2, 80},
		// 15 points sur 30 : part 0,5 x 4 / 2 = 1 pile, la borne elle-même.
		{"sur la borne", 3, 100},
		// 25 points sur 40 : part 0,625 x 4 / 2 = 1,25, écrêté à 100.
		{"au-dessus de la borne", 5, 100},
	}
	for _, c := range cases {
		rows := ctfTeam(4)
		rows[0].Values = map[string]float64{"flag_captures": c.captures}
		got, ok := ObjectiveMatchScore(rows, "ja")
		if !ok || !almostEqual(got, c.want) {
			t.Errorf("%s : note du porteur = %v ok=%v, want %v", c.name, got, ok, c.want)
		}
	}
}

// Un joueur qui n'a rien pris de l'objectif de son équipe a 0 : une mesure, pas une absence.
func TestObjectiveMatchScore_AucuneActionDonneZero(t *testing.T) {
	rows := ctfTeam(4)
	rows[1].Values = map[string]float64{}
	got, ok := ObjectiveMatchScore(rows, "jb")
	if !ok || got != 0 {
		t.Fatalf("note = %v ok=%v, want 0 mesuré", got, ok)
	}
	// Une équipe entière sans action : 0 pour tous, toujours mesuré.
	for i := range rows {
		rows[i].Values = map[string]float64{}
	}
	if got, ok := ObjectiveMatchScore(rows, "ja"); !ok || got != 0 {
		t.Fatalf("équipe sans action : %v ok=%v, want 0 mesuré", got, ok)
	}
}

// Une escorte (retours, sécurisations, frags de porteurs) n'est pas punie de ne pas porter : sa
// note suit sa part du SOUTIEN quand elle dépasse sa part de l'objectif, et le temps de port nul
// n'y change rien (l'ancien indice par famille le pénalisait).
func TestObjectiveMatchScore_EscorteSuitLaPartDuSoutien(t *testing.T) {
	rows := []ObjectiveTeamRow{
		{XUID: "porteur", TeamID: 0, Family: FamilyCTF, TimePlayedSeconds: 600, Values: map[string]float64{
			"flag_captures": 2, "flag_steals": 2, "time_as_flag_carrier_seconds": 120,
		}}, // objectif : 14 points, soutien : 0
		{XUID: "escorte", TeamID: 0, Family: FamilyCTF, TimePlayedSeconds: 600, Values: map[string]float64{
			"flag_returns": 3, "flag_secures": 2,
		}}, // objectif : 6,5 points, soutien : 6,5 points
		{XUID: "tueur", TeamID: 0, Family: FamilyCTF, TimePlayedSeconds: 600, Values: map[string]float64{}},
		{XUID: "libre", TeamID: 0, Family: FamilyCTF, TimePlayedSeconds: 600, Values: map[string]float64{}},
	}
	share, _ := ComputeObjectiveTeamShare(rows, "escorte")
	if !almostEqual(share.SupportShare, 1) || share.Share >= share.SupportShare {
		t.Fatalf("part du soutien %v, part de l'objectif %v : le soutien doit dominer", share.SupportShare, share.Share)
	}
	got, _ := ObjectiveMatchScore(rows, "escorte")
	if !almostEqual(got, 100) { // part du soutien 1 x 4 joueurs / 2 > 1
		t.Fatalf("note de l'escorte = %v, want 100 (soutien)", got)
	}
	// Sans le soutien (ligne de soutien retirée), la même escorte ne vaudrait que sa part d'objectif.
	if only := ObjectiveScoreOf(ObjectiveTeamShare{Share: share.Share, TeamSize: share.TeamSize}); only >= got {
		t.Fatalf("la part de l'objectif seule (%v) ne doit pas égaler la note avec soutien (%v)", only, got)
	}
}

// Halo 5 (aucune ligne), mode sans objectif, joueur absent ou sans camp : note ABSENTE, jamais 0.
func TestObjectiveMatchScore_SansDonneesAbsente(t *testing.T) {
	if _, ok := ObjectiveMatchScore(nil, "x"); ok {
		t.Error("aucune ligne : la note doit être absente")
	}
	if _, ok := ObjectiveMatchScore([]ObjectiveTeamRow{{XUID: "x", TeamID: 0}}, "x"); ok {
		t.Error("aucune famille : la note doit être absente")
	}
	if _, ok := ObjectiveMatchScore(ctfTeam(4), "inconnu"); ok {
		t.Error("joueur absent des lignes : la note doit être absente")
	}
	rows := ctfTeam(4)
	rows[0].TeamID = NoObjectiveTeam
	if _, ok := ObjectiveMatchScore(rows, "ja"); ok {
		t.Error("camp inconnu : la note doit être absente")
	}
}

// La moyenne ne compte que les matchs mesurés ; un abandon immédiat (<= 30 s) est écarté ; aucun
// match mesuré = (0, 0), l'appelant retire l'axe.
func TestObjectiveParticipation_MoyenneDesMatchsMesures(t *testing.T) {
	full := ctfTeam(4) // ja : 50
	carry := ctfTeam(4)
	carry[0].Values = map[string]float64{"flag_captures": 3} // ja : 100
	ghost := ctfTeam(4)
	ghost[0].TimePlayedSeconds = ObjectiveScoreMinPlayedSeconds
	byMatch := map[string][]ObjectiveTeamRow{"m1": full, "m2": carry, "m3": ghost, "m4": nil}
	mean, n := ObjectiveParticipation(byMatch, "ja")
	if n != 2 || !almostEqual(mean, 75) {
		t.Fatalf("(%v, %d), want (75, 2)", mean, n)
	}
	// Plusieurs identités : la première présente dans le match est retenue.
	mean, n = ObjectiveParticipation(map[string][]ObjectiveTeamRow{"m1": full}, "autre", "ja")
	if n != 1 || !almostEqual(mean, 50) {
		t.Fatalf("(%v, %d), want (50, 1) avec deux identités", mean, n)
	}
	if mean, n := ObjectiveParticipation(map[string][]ObjectiveTeamRow{"m3": ghost}, "ja"); mean != 0 || n != 0 {
		t.Fatalf("(%v, %d), want (0, 0) sans match mesuré", mean, n)
	}
	if _, n := ObjectiveParticipation(nil, "ja"); n != 0 {
		t.Fatalf("n = %d, want 0", n)
	}
}

// Un match non terminé du joueur mesuré n'entre pas dans la moyenne (comme la note de performance),
// mais ses lignes et celles de ses coéquipiers non terminés restent dans l'équipe : la part des
// autres joueurs et la taille de l'équipe ne bougent pas.
func TestObjectiveParticipation_MatchNonTermineExclu(t *testing.T) {
	finished := ctfTeam(4) // ja : 50
	abandoned := ctfTeam(4)
	abandoned[0].Outcome = domain.OutcomeDNF
	byMatch := map[string][]ObjectiveTeamRow{"m1": finished, "m2": abandoned}

	mean, n := ObjectiveParticipation(byMatch, "ja")
	if n != 1 || !almostEqual(mean, 50) {
		t.Fatalf("(%v, %d), want (50, 1) : l'abandon n'entre pas dans la moyenne", mean, n)
	}
	// Aucun match terminé : aucune mesure, l'appelant retire l'axe.
	if mean, n := ObjectiveParticipation(map[string][]ObjectiveTeamRow{"m2": abandoned}, "ja"); mean != 0 || n != 0 {
		t.Fatalf("(%v, %d), want (0, 0) : seul match non terminé", mean, n)
	}
	// Un coéquipier non terminé reste dans l'équipe : le joueur mesuré garde sa part et sa taille d'équipe.
	teammateQuit := ctfTeam(4)
	teammateQuit[1].Outcome = domain.OutcomeDNF
	got, ok := ObjectiveMatchScore(teammateQuit, "ja")
	if !ok || !almostEqual(got, 50) {
		t.Fatalf("note avec un coéquipier non terminé = %v ok=%v, want 50", got, ok)
	}
	mean, n = ObjectiveParticipation(map[string][]ObjectiveTeamRow{"m1": teammateQuit}, "ja")
	if n != 1 || !almostEqual(mean, 50) {
		t.Fatalf("(%v, %d), want (50, 1) : le non-terminé d'un coéquipier ne retire pas le match", mean, n)
	}
	// Résultat inconnu (0, titre ou ligne sans résultat) : le match compte.
	unknown := ctfTeam(4)
	unknown[0].Outcome = domain.OutcomeUnknown
	if _, n := ObjectiveParticipation(map[string][]ObjectiveTeamRow{"m": unknown}, "ja"); n != 1 {
		t.Fatalf("n = %d, want 1 : résultat inconnu = match mesuré", n)
	}
}
