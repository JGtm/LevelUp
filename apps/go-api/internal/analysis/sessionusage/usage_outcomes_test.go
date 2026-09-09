package sessionusage

// usage_outcomes_test.go — LES TROIS ISSUES au grain session (étape E3) : la
// grandeur "equipment_<famille>" et son remplissage, les deux taux de référence
// qui EXCLUENT le joueur (décision P7), et la règle de scope — un sous-ensemble
// vide rend nil, jamais un zéro inventé.

import (
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// sessionIssuesDeTest — deux matchs mesurés, camps connus.
//
//	m1 : P prend 3 murs, en pose 1, en lâche 1 -> gardé 1 (barre 3, utilisé 1)
//	     A (mon camp) prend 2 murs, en pose 2 -> barre 2, utilisé 2
//	     E1 (eux) prend 4 murs, n'en pose aucun, en lâche 4 -> barre 4, utilisé 0
//	m2 : P prend 1 capteur, ne l'utilise ni ne le lâche -> gardé 1
func sessionIssuesDeTest() Input {
	return Input{
		PlayerXUID: "P",
		Matches: []MatchInput{
			{
				MatchID: "m1", Measured: true, DurationSeconds: 600,
				PlayerTeam: intp(0),
				TeamOf:     map[string]int{"P": 0, "A": 0, "E1": 1},
				TeamSize:   2, LobbySize: 3,
				Players: []PlayerRow{
					{
						MatchID: "m1", XUID: "P",
						DeployedByFamily: map[string]int{"wall": 1},
						TakenByFamily:    map[string]int{"wall": 3},
						DroppedByFamily:  map[string]int{"wall": 1},
						KeptByFamily:     map[string]int{"wall": 1},
					},
					{
						MatchID: "m1", XUID: "A",
						DeployedByFamily: map[string]int{"wall": 2},
						TakenByFamily:    map[string]int{"wall": 2},
					},
					{
						MatchID: "m1", XUID: "E1",
						TakenByFamily:   map[string]int{"wall": 4},
						DroppedByFamily: map[string]int{"wall": 4},
					},
				},
			},
			{
				MatchID: "m2", Measured: true, DurationSeconds: 600,
				PlayerTeam: intp(0),
				TeamOf:     map[string]int{"P": 0, "E1": 1},
				TeamSize:   1, LobbySize: 2,
				Players: []PlayerRow{
					{
						MatchID: "m2", XUID: "P",
						TakenByFamily: map[string]int{"sensor": 1},
						KeptByFamily:  map[string]int{"sensor": 1},
					},
				},
			},
		},
	}
}

// TestOutcomes_LaBarreEstLaSommeDesTroisIssues — la valeur de la grandeur remplit
// exactement sa pile : sans cette égalité, une barre empilée déborderait ou
// laisserait un trou.
func TestOutcomes_LaBarreEstLaSommeDesTroisIssues(t *testing.T) {
	out := ComputeUsage(sessionIssuesDeTest())
	m := findMetric(t, out.Metrics, MetricEquipmentPrefix+"wall")
	if m.Outcomes == nil {
		t.Fatal("equipment_wall sans Outcomes")
	}
	o := m.Outcomes
	if o.Used != 1 || o.Kept != 1 || o.Dropped != 1 {
		t.Errorf("issues du joueur = (%v, %v, %v), attendu (1, 1, 1)", o.Used, o.Kept, o.Dropped)
	}
	if m.PlayerTotal != o.Used+o.Kept+o.Dropped {
		t.Errorf("PlayerTotal = %v != somme des issues %v", m.PlayerTotal, o.Used+o.Kept+o.Dropped)
	}
	// Les PRISES restent servies à part : dénominateur d'honnêteté, jamais l'axe.
	if o.Taken != 3 {
		t.Errorf("Taken = %v, attendu 3", o.Taken)
	}
	if !closeTo(o.UsedRatePct, 100.0/3) {
		t.Errorf("UsedRatePct = %v, attendu 33,33 %% (1 utilisé sur 3)", o.UsedRatePct)
	}
}

// TestOutcomes_LesReferencesExcluentLeJoueur — décision P7. Sur m1 : mon camp
// moins moi, c'est A seul (2 utilisés sur 2) ; eux, c'est E1 seul (0 sur 4).
func TestOutcomes_LesReferencesExcluentLeJoueur(t *testing.T) {
	out := ComputeUsage(sessionIssuesDeTest())
	o := findMetric(t, out.Metrics, MetricEquipmentPrefix+"wall").Outcomes
	if o == nil {
		t.Fatal("equipment_wall sans Outcomes")
	}
	if !closeTo(o.TeammatesUsedRatePct, 100) {
		t.Errorf("TeammatesUsedRatePct = %v, attendu 100 %% (A seul : 2 posés sur 2) — "+
			"le joueur de la route a été inclus dans sa propre référence", o.TeammatesUsedRatePct)
	}
	if !closeTo(o.OpponentsUsedRatePct, 0) {
		t.Errorf("OpponentsUsedRatePct = %v, attendu 0 %% (E1 : 4 lâchés, aucun posé)",
			o.OpponentsUsedRatePct)
	}
}

// TestOutcomes_FamilleSansAucunDeploiement — le capteur du parc compte 4 objets
// utilisés pour 36 pris : une famille jamais posée DOIT quand même avoir sa
// ligne, sans quoi l'écran efface précisément l'histoire qu'il raconte.
func TestOutcomes_FamilleSansAucunDeploiement(t *testing.T) {
	out := ComputeUsage(sessionIssuesDeTest())
	m := findMetric(t, out.Metrics, MetricEquipmentPrefix+"sensor")
	if m.Outcomes == nil || m.Outcomes.Kept != 1 || m.Outcomes.Used != 0 {
		t.Errorf("equipment_sensor = %+v, attendu 1 gardé et 0 utilisé", m.Outcomes)
	}
	// Et son cousin `deployed_sensor` n'existe pas : personne n'en a posé.
	for _, x := range out.Metrics {
		if x.Key == MetricDeployedPrefix+"sensor" {
			t.Error("deployed_sensor ne devrait pas exister (aucune pose) — la ligne d'issue ne le remplace pas")
		}
	}
}

// TestOutcomes_FFA_ReferencesNil — sur une session sans aucun camp connu, les deux
// taux de référence restent nil : « je ne sais pas » ne s'écrit pas 0 %.
func TestOutcomes_FFA_ReferencesNil(t *testing.T) {
	in := sessionIssuesDeTest()
	for i := range in.Matches {
		in.Matches[i].PlayerTeam = nil
	}
	out := ComputeUsage(in)
	o := findMetric(t, out.Metrics, MetricEquipmentPrefix+"wall").Outcomes
	if o == nil {
		t.Fatal("equipment_wall sans Outcomes")
	}
	if o.TeammatesUsedRatePct != nil || o.OpponentsUsedRatePct != nil {
		t.Errorf("FFA : références = (%v, %v), attendu nil/nil",
			o.TeammatesUsedRatePct, o.OpponentsUsedRatePct)
	}
	// Les issues du joueur, elles, restent mesurées : elles ne dépendent d'aucun camp.
	if o.Used != 1 || o.Kept != 1 || o.Dropped != 1 {
		t.Errorf("issues du joueur en FFA = (%v, %v, %v), attendu (1, 1, 1)", o.Used, o.Kept, o.Dropped)
	}
}

// TestOutcomes_BarreVideRendUnTauxNil — 0/0 n'est pas 0 %. Un joueur qui n'a
// touché à rien n'a pas un taux d'utilisation de zéro : il n'en a pas.
func TestOutcomes_BarreVideRendUnTauxNil(t *testing.T) {
	in := Input{
		PlayerXUID: "P",
		Matches: []MatchInput{{
			MatchID: "m1", Measured: true, DurationSeconds: 600,
			PlayerTeam: intp(0), TeamOf: map[string]int{"P": 0, "E1": 1},
			TeamSize: 1, LobbySize: 2,
			Players: []PlayerRow{
				{MatchID: "m1", XUID: "P"},
				{MatchID: "m1", XUID: "E1", TakenByFamily: map[string]int{"wall": 2},
					DroppedByFamily: map[string]int{"wall": 2}},
			},
		}},
	}
	o := findMetric(t, ComputeUsage(in).Metrics, MetricEquipmentPrefix+"wall").Outcomes
	if o == nil {
		t.Fatal("equipment_wall sans Outcomes — la famille est mesurée dans le lobby")
	}
	if o.UsedRatePct != nil {
		t.Errorf("UsedRatePct = %v, attendu nil (le joueur n'a pris aucun mur)", *o.UsedRatePct)
	}
	// Mon camp moins moi est VIDE : nil, pas 0 %.
	if o.TeammatesUsedRatePct != nil {
		t.Errorf("TeammatesUsedRatePct = %v, attendu nil (aucun allié)", *o.TeammatesUsedRatePct)
	}
	if !closeTo(o.OpponentsUsedRatePct, 0) {
		t.Errorf("OpponentsUsedRatePct = %v, attendu 0 %% (E1 a tout lâché)", o.OpponentsUsedRatePct)
	}
}

// TestOutcomes_SeulesLesGrandeursDEquipementPortentLesIssues — le champ est
// `omitempty` : il ne s'attache QU'aux clés "equipment_*". Une session dont
// AUCUNE ligne ne porte les colonnes `us4` (passe de résumé antérieure) ne publie
// donc rien — jamais des zéros.
func TestOutcomes_SeulesLesGrandeursDEquipementPortentLesIssues(t *testing.T) {
	out := ComputeUsage(sessionDeTest())
	for _, m := range out.Metrics {
		estEquipement := strings.HasPrefix(m.Key, MetricEquipmentPrefix)
		if (m.Outcomes != nil) != estEquipement {
			t.Errorf("la grandeur %q porte Outcomes=%v — seules les clés %q en portent",
				m.Key, m.Outcomes != nil, MetricEquipmentPrefix)
		}
	}
	// Le mur de cette session n'a AUCUNE prise mesurée (passe antérieure à `us4`,
	// ou équipement de réapparition — jamais `taken`) et sa seule issue est la
	// pose d'un ALLIÉ. La ligne existe quand même, parce qu'une pose EST une
	// issue ; ma part y est nulle, et c'est bien ce que le bloc doit montrer.
	m := findMetric(t, out.Metrics, MetricEquipmentPrefix+"wall")
	if m.Outcomes == nil || m.Outcomes.Used != 0 || m.Outcomes.Taken != 0 {
		t.Fatalf("equipment_wall = %+v, attendu aucune issue pour le joueur", m.Outcomes)
	}
	if m.PlayerTotal != 0 || m.LobbyTotal != 1 {
		t.Errorf("(PlayerTotal, LobbyTotal) = (%v, %v), attendu (0, 1)", m.PlayerTotal, m.LobbyTotal)
	}
	if m.Outcomes.UsedRatePct != nil {
		t.Errorf("UsedRatePct = %v, attendu nil (le joueur n'a aucune issue)", *m.Outcomes.UsedRatePct)
	}
	if !closeTo(m.Outcomes.TeammatesUsedRatePct, 100) {
		t.Errorf("TeammatesUsedRatePct = %v, attendu 100 %% (l'allié a posé son mur)",
			m.Outcomes.TeammatesUsedRatePct)
	}
}

// TestOutcomes_LeRepulseurNaJamaisDeLigne — décision P4 : son usage n'est mesuré
// par AUCUN canal (négatif mesuré, neuf canaux fouillés). Une ligne dirait
// « 0 utilisation » là où la vérité est « non mesuré ».
func TestOutcomes_LeRepulseurNaJamaisDeLigne(t *testing.T) {
	in := Input{
		PlayerXUID: "P",
		Matches: []MatchInput{{
			MatchID: "m1", Measured: true, DurationSeconds: 600,
			PlayerTeam: intp(0), TeamOf: map[string]int{"P": 0},
			TeamSize: 1, LobbySize: 1,
			Players: []PlayerRow{{
				MatchID: "m1", XUID: "P",
				// La projection ne produit jamais ces clés (aucune racine ne les
				// nomme) ; si un jour elle le faisait, l'agrégat ne doit pas suivre.
				TakenByFamily:   map[string]int{"repulsor": 5, "grapple": 3, "thruster": 2},
				DroppedByFamily: map[string]int{"repulsor": 5},
			}},
		}},
	}
	for _, m := range ComputeUsage(in).Metrics {
		for _, interdite := range []string{"repulsor", "grapple", "thruster"} {
			if m.Key == MetricEquipmentPrefix+interdite {
				t.Errorf("%q a une ligne d'issue alors que son usage n'est mesuré nulle part", m.Key)
			}
		}
	}
}

// TestOutcomes_LesLignesDEscouadeSuivent — une grandeur d'équipement est une
// grandeur comme les autres : elle hérite du tronc commun (parts, cadences,
// escouade) sans une ligne de code de plus.
func TestOutcomes_LesLignesDEscouadeSuivent(t *testing.T) {
	in := sessionIssuesDeTest()
	in.SquadXUIDs = []string{"A"}
	m := findMetric(t, ComputeUsage(in).Metrics, MetricEquipmentPrefix+"wall")
	if len(m.Squad) != 1 || m.Squad[0].XUID != "A" {
		t.Fatalf("Squad = %+v, attendu une ligne pour A", m.Squad)
	}
	if m.Squad[0].Total != 2 {
		t.Errorf("Total(A) = %v, attendu 2 (deux murs posés)", m.Squad[0].Total)
	}
	var _ domain.SessionUsageMetric = m
}
