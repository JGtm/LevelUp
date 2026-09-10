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

// TestOutcomes_UnDeployableSansPieceSeLitSurSesConsommations — CORRECTION C1
// (revue de la vague 5, 2026-09-10). Le cas exact du constat : un capteur
// `taken=3, spent=2, dropped=1, deployed=0`. Le capteur n'engendre aucune pièce
// (replay.UsageFamilySpawnsPiece), son « utilisé » se lit donc sur les CHARGES
// CONSOMMÉES, comme le fait le résumé depuis `us6`.
//
// AVANT LA CORRECTION, l'agrégat lisait `deployed` pour toutes les familles : la
// page Sessions affichait « utilisé 0 · gardé 0 · lâché 1 » là où la vue match
// affichait « utilisé 2 », la valeur de la grandeur valait 1 au lieu de 3, et la
// famille disparaissait même de la liste des grandeurs dès que ses poses étaient
// nulles (metricKeys n'ouvre une ligne que sur un total non nul).
func TestOutcomes_UnDeployableSansPieceSeLitSurSesConsommations(t *testing.T) {
	in := Input{
		PlayerXUID: "P",
		Matches: []MatchInput{{
			MatchID: "m1", Measured: true, DurationSeconds: 600,
			PlayerTeam: intp(0), TeamOf: map[string]int{"P": 0},
			TeamSize: 1, LobbySize: 1,
			Players: []PlayerRow{{
				MatchID: "m1", XUID: "P",
				TakenByFamily:   map[string]int{"sensor": 3},
				SpentByFamily:   map[string]int{"sensor": 2},
				DroppedByFamily: map[string]int{"sensor": 1},
				// Aucune pose, et pourtant deux objets servis.
			}},
		}},
	}
	m := findMetric(t, ComputeUsage(in).Metrics, MetricEquipmentPrefix+"sensor")
	if m.Outcomes == nil {
		t.Fatal("equipment_sensor sans Outcomes")
	}
	o := m.Outcomes
	if o.Used != 2 || o.Kept != 0 || o.Dropped != 1 || o.Taken != 3 {
		t.Errorf("issues = (utilisé %v, gardé %v, lâché %v, pris %v), attendu (2, 0, 1, 3)",
			o.Used, o.Kept, o.Dropped, o.Taken)
	}
	if m.PlayerTotal != 3 {
		t.Errorf("PlayerTotal = %v, attendu 3 (la barre remplit ses trois prises)", m.PlayerTotal)
	}
	if !closeTo(o.UsedRatePct, 200.0/3) {
		t.Errorf("UsedRatePct = %v, attendu 66,67 %% (2 utilisés sur 3)", o.UsedRatePct)
	}
}

// TestOutcomes_LeMurResteLuSurSesPoses — l'autre côté de la même frontière : le mur
// est le SEUL équipement du manifeste qui engendre une pièce distincte (ses
// panneaux), son « utilisé » reste donc sur les poses. Une correction qui
// basculerait TOUT sur les consommations perdrait 50 murs utilisés au parc.
func TestOutcomes_LeMurResteLuSurSesPoses(t *testing.T) {
	in := Input{
		PlayerXUID: "P",
		Matches: []MatchInput{{
			MatchID: "m1", Measured: true, DurationSeconds: 600,
			PlayerTeam: intp(0), TeamOf: map[string]int{"P": 0},
			TeamSize: 1, LobbySize: 1,
			Players: []PlayerRow{{
				MatchID: "m1", XUID: "P",
				TakenByFamily:    map[string]int{"wall": 2},
				DeployedByFamily: map[string]int{"wall": 2},
				// Le film annonce moins de consommations que de poses de panneau
				// (118 pour 252 au parc) : les lire ici sous-compterait.
				SpentByFamily: map[string]int{"wall": 1},
			}},
		}},
	}
	o := findMetric(t, ComputeUsage(in).Metrics, MetricEquipmentPrefix+"wall").Outcomes
	if o == nil || o.Used != 2 {
		t.Errorf("mur = %+v, attendu 2 utilisés (ses POSES de panneau, pas ses consommations)", o)
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

// TestOutcomes_BarreVideRendUnTauxNil — 0/0 n'est pas 0 %. Un camp sans aucun
// coéquipier n'a pas un taux de référence de zéro : il n'en a pas.
//
// DEPUIS LE LOT 6.4 POINT 4, une ligne "equipment_<famille>" n'entre QUE si le
// SUJET l'a lui-même touchée ([subjectBilanFamilies], source commune à
// [metricKeys] et [overviewFamilies]) : le « 0/0 » du joueur lui-même ne se
// manifeste donc plus comme une ligne à taux nil, mais comme une ligne ABSENTE
// (TestOutcomes_LeRepulseurNaJamaisDeLigne le couvre). Le joueur touche ici son
// mur en le lâchant SANS l'utiliser, pour que la ligne entre et que le nil
// restant (aucun allié) reste observable sur la référence d'équipe.
func TestOutcomes_BarreVideRendUnTauxNil(t *testing.T) {
	in := Input{
		PlayerXUID: "P",
		Matches: []MatchInput{{
			MatchID: "m1", Measured: true, DurationSeconds: 600,
			PlayerTeam: intp(0), TeamOf: map[string]int{"P": 0, "E1": 1},
			TeamSize: 1, LobbySize: 2,
			Players: []PlayerRow{
				{MatchID: "m1", XUID: "P", DroppedByFamily: map[string]int{"wall": 1}},
				{MatchID: "m1", XUID: "E1", TakenByFamily: map[string]int{"wall": 2},
					DroppedByFamily: map[string]int{"wall": 2}},
			},
		}},
	}
	o := findMetric(t, ComputeUsage(in).Metrics, MetricEquipmentPrefix+"wall").Outcomes
	if o == nil {
		t.Fatal("equipment_wall sans Outcomes — le joueur a lâché un mur")
	}
	if !closeTo(o.UsedRatePct, 0) {
		t.Errorf("UsedRatePct = %v, attendu 0 %% (le joueur a lâché son mur sans l'utiliser)", o.UsedRatePct)
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
	// Le mur de cette session n'est posé QUE par un ALLIÉ (A) : le joueur de la
	// route ne l'a jamais touché. DEPUIS LE LOT 6.4 POINT 4, une ligne
	// "equipment_<famille>" n'entre QUE sur le sujet lui-même
	// ([subjectBilanFamilies]) — avant ce lot, la ligne existait quand même (sur
	// la seule pose de l'allié) et affichait une barre entièrement vide pour le
	// joueur : le « reproche sans objet » que ce lot supprime. `deployed_wall`
	// (grandeur LOBBY, inchangée) reste, lui, observé sur A — voir
	// TestComputeUsage_LignesEscouade dans usage_test.go.
	for _, m := range out.Metrics {
		if m.Key == MetricEquipmentPrefix+"wall" {
			t.Fatalf("equipment_wall présent alors que le joueur n'a jamais touché de mur : %+v", m)
		}
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

// TestBilan_MetricKeysEtOverviewFamiliesPartagentLeCritere — lot 6.4 point 4.
// AVANT ce lot, [metricKeys] (page Sessions) ouvrait une ligne "equipment_<famille>"
// dès qu'UN JOUEUR DU LOBBY la touchait, pendant qu'[overviewFamilies] (Synthèse,
// Escouade) l'ouvrait seulement sur LE SUJET — deux critères pour la MÊME barre
// subjet-only. Ce test verrouille la source commune ([subjectBilanFamilies]) : sur
// un scope où seul un COÉQUIPIER touche "shroud_screen", NI l'une NI l'autre ne
// doit publier de ligne pour cette famille ; sur "wall"/"sensor", que LE SUJET
// touche, LES DEUX doivent en publier une.
func TestBilan_MetricKeysEtOverviewFamiliesPartagentLeCritere(t *testing.T) {
	measured := sessionIssuesDeTest().Matches
	// Un coéquipier (A, déjà du camp de P sur m1) touche une troisième famille que
	// P ne touche jamais.
	for i := range measured {
		if measured[i].MatchID != "m1" {
			continue
		}
		for j := range measured[i].Players {
			if measured[i].Players[j].XUID == "A" {
				measured[i].Players[j].TakenByFamily["shroud_screen"] = 1
				measured[i].Players[j].DeployedByFamily["shroud_screen"] = 1
			}
		}
	}

	sessionKeys := metricKeys("P", measured)
	overview := overviewFamilies("P", measured)
	overviewKeys := map[string]bool{}
	for _, f := range overview {
		overviewKeys[MetricEquipmentPrefix+f.FamilyKey] = true
	}
	sessionBilan := map[string]bool{}
	for _, k := range sessionKeys {
		if strings.HasPrefix(k, MetricEquipmentPrefix) {
			sessionBilan[k] = true
		}
	}

	for _, touched := range []string{MetricEquipmentPrefix + "wall", MetricEquipmentPrefix + "sensor"} {
		if !sessionBilan[touched] {
			t.Errorf("metricKeys omet %q, que le sujet a pourtant touchée", touched)
		}
		if !overviewKeys[touched] {
			t.Errorf("overviewFamilies omet %q, que le sujet a pourtant touchée", touched)
		}
	}
	interdite := MetricEquipmentPrefix + "shroud_screen"
	if sessionBilan[interdite] {
		t.Errorf("metricKeys publie %q sur la seule foi d'un coéquipier — critère du lobby, pas du sujet", interdite)
	}
	if overviewKeys[interdite] {
		t.Errorf("overviewFamilies publie %q sur la seule foi d'un coéquipier", interdite)
	}
	if len(sessionBilan) != len(overviewKeys) {
		t.Errorf("ensembles divergents : metricKeys=%v, overviewFamilies=%v", sessionBilan, overviewKeys)
	}
}
