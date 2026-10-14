package wire

// registry_pages_timeseries_wiring_test.go — LE CABLAGE DE L'ONGLET « USAGES » DES SERIES
// TEMPORELLES (Emprise solo, plan `.ai/V7.5/PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05.md`, lot L2.8).
//
// Meme mode de panne que l'Escouade (registry_pages_home_teammates_wiring_test.go) : le service
// degrade EN SILENCE sur une dependance nil. Les options sont lues dans l'arbre syntaxique de la
// factory `Timeseries` — leur argument et la porte `if` qui les entoure.

import (
	"strings"
	"testing"
)

func appelsDansTimeseries(t *testing.T, methode string) []appelOptionCable {
	t.Helper()
	return appelsDansFactory(t, "registry_pages.go", "Timeseries", methode)
}

// appelsDansUsages lit le cablage propre a l'onglet « Usages » (registry_pages_timeseries.go).
func appelsDansUsages(t *testing.T, methode string) []appelOptionCable {
	t.Helper()
	return appelsDansFactory(t, "registry_pages_timeseries.go", "cablerUsagesTimeseries", methode)
}

// La factory appelle le cablage de l'onglet SANS condition.
func TestTimeseries_AppelleLeCablageDesUsagesSansCondition(t *testing.T) {
	appels := appelsDansTimeseries(t, "cablerUsagesTimeseries")
	if len(appels) != 1 || len(appels[0].portes) != 0 {
		t.Errorf("cablerUsagesTimeseries = %+v : attendu un appel inconditionnel dans Timeseries", appels)
	}
}

// La feuille de match de l'Emprise est écrite par tous les titres : câblée SANS condition, sur le
// PlayerDB du joueur.
func TestTimeseries_CableLaFeuilleDeLEmpriseSansCondition(t *testing.T) {
	appels := appelsDansUsages(t, "WithEmprise")
	if len(appels) != 1 {
		t.Fatalf("%d appel(s) à WithEmprise dans le câblage des Usages, attendu 1", len(appels))
	}
	a := appels[0]
	if strings.Join(a.args, ", ") != "duckdb.NewSquadEmpriseRepo(pdb, r.killSourceClassifierFor(pdb))" {
		t.Errorf("WithEmprise(%s) : attendu WithEmprise(duckdb.NewSquadEmpriseRepo(pdb, r.killSourceClassifierFor(pdb)))", strings.Join(a.args, ", "))
	}
	if len(a.portes) != 0 {
		t.Errorf("WithEmprise est sous condition (%v) : la feuille de match est écrite par tous les titres", a.portes)
	}
}

// Le résumé d'usage (lu par l'Emprise et les formes) reste sous la seule porte
// film.usage_summary : un titre sans film ne le reçoit pas (l'Emprise y dit `film_unsupported`).
func TestTimeseries_LeResumeDUsageResteSousFilmUsageSummary(t *testing.T) {
	for _, methode := range []string{"WithUsageSummary", "WithSquadFormes"} {
		appels := appelsDansTimeseries(t, methode)
		if len(appels) != 1 {
			t.Fatalf("%d appel(s) à %s dans Timeseries, attendu 1", len(appels), methode)
		}
		if p := appels[0].portes; len(p) != 1 || !strings.Contains(p[0], "games.CapFilmUsageSummary") {
			t.Errorf("%s hors de la seule porte games.CapFilmUsageSummary (portes : %v)", methode, p)
		}
	}
}

// La ressource véhicules : la seule porte film.vehicle_usage ; l'emblème : sans condition.
func TestTimeseries_CableVehiculesEtEmbleme(t *testing.T) {
	veh := appelsDansUsages(t, "WithVehicleUsage")
	if len(veh) != 1 || len(veh[0].portes) != 1 || !strings.Contains(veh[0].portes[0], "games.CapFilmVehicleUsage") {
		t.Errorf("WithVehicleUsage = %+v : attendu un appel sous la seule porte games.CapFilmVehicleUsage", veh)
	}
	emb := appelsDansUsages(t, "WithEmblemLoader")
	if len(emb) != 1 || len(emb[0].portes) != 0 || !strings.Contains(strings.Join(emb[0].args, ""), "NewSquadV2LoaderAdapter") {
		t.Errorf("WithEmblemLoader = %+v : attendu un appel inconditionnel au chargeur de l'Escouade", emb)
	}
}

// Les vies du joueur : la seule porte film.kill_positions ; la table des portées : sans condition.
func TestTimeseries_CableLesViesEtLaPorteeDuRadar(t *testing.T) {
	vies := appelsDansUsages(t, "WithLivesNearTeammate")
	if len(vies) != 1 || strings.Join(vies[0].args, "") != "duckdb.NewSoloLivesRepo(pdb)" ||
		len(vies[0].portes) != 1 || !strings.Contains(vies[0].portes[0], "games.CapFilmKillPositions") {
		t.Errorf("WithLivesNearTeammate = %+v : attendu duckdb.NewSoloLivesRepo(pdb) sous la seule porte games.CapFilmKillPositions", vies)
	}
	radar := appelsDansUsages(t, "WithRadarRange")
	if len(radar) != 1 || strings.Join(radar[0].args, "") != "r.radarRangeFor(pdb)" || len(radar[0].portes) != 0 {
		t.Errorf("WithRadarRange = %+v : attendu un appel inconditionnel WithRadarRange(r.radarRangeFor(pdb))", radar)
	}
}

// Le résumé d'usage n'a plus de résolveur d'amis (le bloc « servi ou gâché » qui le lisait a quitté
// la page) : le repo et la racine du dépôt, rien d'autre.
func TestTimeseries_LeResumeDUsageSansResolveurDAmis(t *testing.T) {
	appels := appelsDansTimeseries(t, "WithUsageSummary")
	if len(appels) != 1 || strings.Join(appels[0].args, ", ") != "duckdb.NewSessionUsageRepo(pdb), r.cfg.RepoRoot" {
		t.Errorf("WithUsageSummary = %+v : attendu WithUsageSummary(duckdb.NewSessionUsageRepo(pdb), r.cfg.RepoRoot)", appels)
	}
}
