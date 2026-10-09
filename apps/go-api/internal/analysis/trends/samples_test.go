package trends

import (
	"reflect"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/equipmentusage"
)

const (
	pXUID   = "xuid-moi"
	allyXID = "xuid-allie"
	foeXID  = "xuid-adverse"
)

// teams fabrique un contexte de camp : moi et l'allié au camp 0, l'adversaire au camp 1.
func teams(matchIDs ...string) sessionusage.TeamContext {
	tc := sessionusage.TeamContext{
		PlayerTeam: map[string]int{}, TeamOf: map[string]map[string]int{},
		TeamSize: map[string]int{},
	}
	for _, id := range matchIDs {
		tc.PlayerTeam[id] = 0
		tc.TeamOf[id] = map[string]int{pXUID: 0, allyXID: 0, foeXID: 1}
		tc.TeamSize[id] = 4
	}
	return tc
}

func TestObjectiveSamples_CampInconnuPasDEchantillon(t *testing.T) {
	rows := []sessionusage.ObjectiveRow{{MatchID: "m1", XUID: pXUID, Take: 3}}
	got := ObjectiveSamples(ObjectiveInput{PlayerXUID: pXUID, Rows: rows, Teams: teams()})
	if len(got) != 0 {
		t.Fatalf("camp inconnu : %v", got)
	}
}

func TestObjectiveSamples_JoueurSansLigneDeRoleOuLigneDeLAdversaire(t *testing.T) {
	rows := []sessionusage.ObjectiveRow{
		{MatchID: "m1", XUID: allyXID, Take: 3},
		{MatchID: "m2", XUID: pXUID, Take: 1},
	}
	got := ObjectiveSamples(ObjectiveInput{PlayerXUID: pXUID, Rows: rows, Teams: teams("m1", "m2")})
	if _, ok := got["m1"]; ok {
		t.Fatalf("m1 sans ligne du joueur ne doit pas avoir d'échantillon : %v", got)
	}
	if _, ok := got["m2"]; !ok {
		t.Fatalf("m2 attendu : %v", got)
	}
}

func TestObjectiveSamples_EquipeEstLeCampDuJoueurSeulement(t *testing.T) {
	rows := []sessionusage.ObjectiveRow{
		{MatchID: "m1", XUID: pXUID, Take: 2, Defend: 1, HoldSeconds: 30},
		{MatchID: "m1", XUID: allyXID, Take: 3, Defend: 4, HoldSeconds: 10},
		{MatchID: "m1", XUID: foeXID, Take: 100, Defend: 100, HoldSeconds: 100},
	}
	got := ObjectiveSamples(ObjectiveInput{PlayerXUID: pXUID, Rows: rows, Teams: teams("m1")})["m1"]
	want := ObjectiveSample{Take: 2, Defend: 1, HoldSeconds: 30, TeamTake: 5, TeamDefend: 5, TeamHoldSeconds: 40, TeamSize: 4}
	if got != want {
		t.Fatalf("échantillon = %+v, attendu %+v", got, want)
	}
}

func TestObjectiveSamples_PlusieursLignesSommeesEtPrisesNettes(t *testing.T) {
	rows := []sessionusage.ObjectiveRow{
		{MatchID: "m1", XUID: pXUID, Take: 1, HoldSeconds: 5},
		{MatchID: "m1", XUID: pXUID, Take: 2, HoldSeconds: 7},
		{MatchID: "m1", XUID: allyXID, Take: 1},
	}
	grabs := []sessionusage.FlagGrabsNetRow{
		{MatchID: "m1", XUID: pXUID, Net: 4},
		{MatchID: "m1", XUID: allyXID, Net: 2},
		{MatchID: "m1", XUID: foeXID, Net: 9},
		// Prises nettes seules : pas d'échantillon pour ce match.
		{MatchID: "m2", XUID: pXUID, Net: 5},
	}
	got := ObjectiveSamples(ObjectiveInput{PlayerXUID: pXUID, Rows: rows, FlagGrabs: grabs, Teams: teams("m1", "m2")})
	want := ObjectiveSample{Take: 7, HoldSeconds: 12, TeamTake: 10, TeamHoldSeconds: 12, TeamSize: 4}
	if got["m1"] != want {
		t.Fatalf("m1 = %+v, attendu %+v", got["m1"], want)
	}
	if _, ok := got["m2"]; ok {
		t.Fatalf("m2 sans ligne de rôle ne doit pas avoir d'échantillon")
	}
}

func equipmentRow(match, xuid string, used, kept, dropped int) sessionusage.PlayerRow {
	fam := equipmentusage.EquipmentFamilyPowerupCamo
	return sessionusage.PlayerRow{
		MatchID: match, XUID: xuid, CamoEpisodes: used,
		TakenByFamily:   map[string]int{fam: used + kept + dropped},
		KeptByFamily:    map[string]int{fam: kept},
		DroppedByFamily: map[string]int{fam: dropped},
	}
}

func TestEquipmentSamples(t *testing.T) {
	players := []sessionusage.PlayerRow{
		equipmentRow("m1", pXUID, 2, 1, 1),
		equipmentRow("m1", foeXID, 9, 9, 9),
		equipmentRow("m2", foeXID, 1, 0, 0),
	}
	got := EquipmentSamples(pXUID, players)
	if len(got) != 1 {
		t.Fatalf("un seul match mesuré attendu : %v", got)
	}
	if want := (EquipmentSample{Used: 2, Total: 4}); got["m1"] != want {
		t.Fatalf("m1 = %+v, attendu %+v", got["m1"], want)
	}
}

func TestAttachAndMatchIDsSince(t *testing.T) {
	loc := time.UTC
	ms := []Match{
		{ID: "a", Start: ago(10, 0).In(loc)},
		{ID: "b", Start: ago(1, 0).In(loc)},
		{ID: "", Start: ago(1, 0).In(loc)},
	}
	Attach(ms, map[string]ObjectiveSample{"b": {Take: 1}}, map[string]EquipmentSample{"a": {Used: 1, Total: 2}})
	if ms[0].Objective != nil || ms[0].Equipment == nil || ms[0].Equipment.Total != 2 {
		t.Fatalf("a : %+v", ms[0])
	}
	if ms[1].Objective == nil || ms[1].Objective.Take != 1 || ms[1].Equipment != nil {
		t.Fatalf("b : %+v", ms[1])
	}
	if ms[2].Objective != nil || ms[2].Equipment != nil {
		t.Fatalf("sans identifiant : %+v", ms[2])
	}
	if got := MatchIDsSince(ms, ago(5, 0)); !reflect.DeepEqual(got, []string{"b"}) {
		t.Fatalf("MatchIDsSince = %v", got)
	}
	if got := MatchIDsSince(ms, ago(11, 0)); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("MatchIDsSince = %v", got)
	}
}

// withObjective pose un échantillon d'objectif (joueur 1 / équipe 4 en Prendre, effectif 4).
func withObjective(take, team float64) func(*Match) {
	return func(m *Match) {
		m.Objective = &ObjectiveSample{Take: take, TeamTake: team, Defend: take, TeamDefend: team, HoldSeconds: take, TeamHoldSeconds: team, TeamSize: 4}
	}
}

func TestObjectiveShareAndParityMinimums(t *testing.T) {
	loc := time.UTC
	// 4 matchs mesurés dans la semaine : valeur par jour (min 1), pas de cellule (min 5).
	ms := many(loc, 4, ago(1, 0), withObjective(1, 4))
	resp := BuildSolo(ms, opts(loc))
	if hasIndicator(resp, domain.TrendsKeyObjectiveTakeShare) && indicatorHasCell(indicatorOf(t, resp, domain.TrendsKeyObjectiveTakeShare, "")) {
		t.Fatal("4 matchs : aucune cellule attendue")
	}
	// 5 matchs : cellule d'horizon présente.
	resp = BuildSolo(many(loc, 5, ago(1, 0), withObjective(1, 4)), opts(loc))
	ind := indicatorOf(t, resp, domain.TrendsKeyObjectiveTakeShare, "")
	if c := horizon(t, ind, 7); c.Value == nil || !near(*c.Value, 0.25) {
		t.Fatalf("part de Prendre = %v, attendu 0,25", c.Value)
	}
	par := indicatorOf(t, resp, domain.TrendsKeyObjectiveParity, "")
	if c := horizon(t, par, 30); c.Value == nil || !near(*c.Value, 0.25) || par.InMatrix {
		t.Fatalf("parité = %v (matrice %v)", c.Value, par.InMatrix)
	}
}

func indicatorHasCell(ind domain.TrendsIndicator) bool {
	for _, c := range ind.Horizons {
		if c.Value != nil {
			return true
		}
	}
	for _, c := range ind.Months {
		if c.Value != nil {
			return true
		}
	}
	return false
}

func TestObjectiveSeriesMinimumIsOne(t *testing.T) {
	loc := time.UTC
	// Août : 4 matchs mesurés. Septembre : 3 matchs dont un seul mesuré. L'horizon
	// de 365 jours compte 5 matchs mesurés, la ligne existe ; le point du jour et
	// du match n'exige qu'un match mesuré, celui du mois en exige 5.
	ms := many(loc, 4, ago(40, 0), withObjective(1, 4))
	late := many(loc, 3, ago(2, 0))
	late[1].Objective = &ObjectiveSample{Take: 1, TeamTake: 2, TeamSize: 2}
	ind := indicatorOf(t, BuildSolo(append(ms, late...), opts(loc)), domain.TrendsKeyObjectiveTakeShare, "")
	if len(ind.Series.Match) != 1 || !near(ind.Series.Match[0].Value, 0.5) {
		t.Fatalf("série par match = %+v", ind.Series.Match)
	}
	if len(ind.Series.Day) != 2 || !near(ind.Series.Day[1].Value, 0.5) {
		t.Fatalf("série par jour = %+v", ind.Series.Day)
	}
	if len(ind.Series.Month) != 0 {
		t.Fatalf("série par mois = %+v, attendue vide (sous 5 matchs mesurés)", ind.Series.Month)
	}
	if c := horizon(t, ind, 365); c.Value == nil || !near(*c.Value, 5.0/18.0) {
		t.Fatalf("horizon 365 = %+v", c)
	}
}

func TestObjectiveShareIgnoresMatchesWithoutTeamTotal(t *testing.T) {
	loc := time.UTC
	ms := many(loc, 5, ago(1, 0), withObjective(1, 4))
	for i := 0; i < 5; i++ {
		ms = append(ms, mk(loc, ago(1, time.Hour+time.Duration(i)*time.Minute), withObjective(0, 0)))
	}
	ind := indicatorOf(t, BuildSolo(ms, opts(loc)), domain.TrendsKeyObjectiveTakeShare, "")
	if c := horizon(t, ind, 7); c.Value == nil || !near(*c.Value, 0.25) || c.Matches != 10 {
		t.Fatalf("cellule = %+v", c)
	}
}

func withEquipment(used, total int) func(*Match) {
	return func(m *Match) { m.Equipment = &EquipmentSample{Used: used, Total: total} }
}

func TestEquipmentUsedShareMinimums(t *testing.T) {
	loc := time.UTC
	// 2 matchs mesurés il y a 120 jours, 1 en septembre (avec un match non mesuré).
	aug := many(loc, 2, ago(120, 0), withEquipment(1, 4))
	sep := many(loc, 2, ago(2, 0))
	sep[0].Equipment = &EquipmentSample{Used: 2, Total: 4}
	ind := indicatorOf(t, BuildSolo(append(aug, sep...), opts(loc)), domain.TrendsKeyEquipmentUsedShare, "")
	if c := horizon(t, ind, 365); c.Value == nil || !near(*c.Value, 4.0/12.0) {
		t.Fatalf("horizon 365 = %+v", c)
	}
	if c := horizon(t, ind, 90); c.Value != nil {
		t.Fatalf("horizon 90 : 1 seul match mesuré, aucune valeur attendue : %+v", c)
	}
	if len(ind.Series.Day) != 2 || !near(ind.Series.Day[0].Value, 0.25) || !near(ind.Series.Day[1].Value, 0.5) {
		t.Fatalf("série par jour = %+v", ind.Series.Day)
	}
	if len(ind.Series.Month) != 0 {
		t.Fatalf("série par mois = %+v, attendue vide", ind.Series.Month)
	}
	// Trois matchs mesurés : la cellule existe.
	ind = indicatorOf(t, BuildSolo(many(loc, 3, ago(1, 0), withEquipment(1, 4)), opts(loc)), domain.TrendsKeyEquipmentUsedShare, "")
	if c := horizon(t, ind, 7); c.Value == nil || !near(*c.Value, 0.25) {
		t.Fatalf("part utilisée = %v", c.Value)
	}
	// Deux matchs mesurés : aucune cellule, la ligne est omise.
	if hasIndicator(BuildSolo(many(loc, 2, ago(1, 0), withEquipment(1, 4)), opts(loc)), domain.TrendsKeyEquipmentUsedShare) {
		t.Fatal("2 matchs mesurés : ligne attendue absente")
	}
	// Somme des totaux nulle : pas de valeur.
	if hasIndicator(BuildSolo(many(loc, 4, ago(1, 0), withEquipment(0, 0)), opts(loc)), domain.TrendsKeyEquipmentUsedShare) {
		t.Fatal("total nul : indicateur attendu absent")
	}
	// Match non mesuré (Equipment nil) : pas d'indicateur.
	if hasIndicator(BuildSolo(many(loc, 4, ago(1, 0)), opts(loc)), domain.TrendsKeyEquipmentUsedShare) {
		t.Fatal("aucun match mesuré : indicateur attendu absent")
	}
}

func TestRegistryOrderStyleObjectivesActivity(t *testing.T) {
	var keys []string
	for _, d := range registry(nil, 225) {
		keys = append(keys, d.key)
	}
	pos := map[string]int{}
	for i, k := range keys {
		pos[k] = i
	}
	if !(pos[domain.TrendsKeyPowerWeaponShare] < pos[domain.TrendsKeyEquipmentUsedShare] &&
		pos[domain.TrendsKeyEquipmentUsedShare] < pos[domain.TrendsKeyObjectiveTakeShare] &&
		pos[domain.TrendsKeyObjectiveHoldShare] < pos[domain.TrendsKeyObjectiveParity] &&
		pos[domain.TrendsKeyObjectiveParity] < pos[domain.TrendsKeyMatchCount]) {
		t.Fatalf("ordre inattendu : %v", keys)
	}
}
