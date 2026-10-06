package wire

// registry_pages_sessions_wiring_test.go — LE CABLAGE DES BLOCS DU FILM DE LA PAGE SESSIONS (plan
// `.ai/PLAN_SESSIONS_EMPRISE_2026-10-06.md`, lot S2.9).
//
// Même mode de panne que l'Escouade et les Séries temporelles : le service dégrade EN SILENCE sur
// une dépendance nil. Les options sont lues dans l'arbre syntaxique des factories — leur argument et
// la porte `if` qui les entoure.

import (
	"strings"
	"testing"
)

func appelsDansSessionPage(t *testing.T, methode string) []appelOptionCable {
	t.Helper()
	return appelsDansFactory(t, "registry_pages.go", "SessionPage", methode)
}

func appelsDansBlocsSessions(t *testing.T, methode string) []appelOptionCable {
	t.Helper()
	return appelsDansFactory(t, "registry_pages_sessions.go", "cablerBlocsSessions", methode)
}

// La factory appelle le câblage des blocs SANS condition.
func TestSessionPage_AppelleLeCablageDesBlocsSansCondition(t *testing.T) {
	appels := appelsDansSessionPage(t, "cablerBlocsSessions")
	if len(appels) != 1 || len(appels[0].portes) != 0 {
		t.Errorf("cablerBlocsSessions = %+v : attendu un appel inconditionnel dans SessionPage", appels)
	}
}

// La feuille de match et le joueur de la page (qui sert aussi la coordination) : SANS condition.
func TestSessionPage_CableLaFeuilleEtLeJoueurSansCondition(t *testing.T) {
	appels := appelsDansBlocsSessions(t, "WithSessionEmprise")
	if len(appels) != 1 {
		t.Fatalf("%d appel(s) à WithSessionEmprise, attendu 1", len(appels))
	}
	if got := strings.Join(appels[0].args, ", "); got != "duckdb.NewSquadEmpriseRepo(pdb), pdb.XUID" {
		t.Errorf("WithSessionEmprise(%s) : attendu WithSessionEmprise(duckdb.NewSquadEmpriseRepo(pdb), pdb.XUID)", got)
	}
	if len(appels[0].portes) != 0 {
		t.Errorf("WithSessionEmprise est sous condition (%v) : la feuille est écrite par tous les titres", appels[0].portes)
	}
}

// Emblème, portées du radar et manches : sans condition.
func TestSessionPage_CableEmblemePorteeEtManchesSansCondition(t *testing.T) {
	for methode, arg := range map[string]string{
		"WithSessionEmblemLoader": "NewSquadV2LoaderAdapter",
		"WithSessionRadarRange":   "r.radarRangeFor(pdb)",
		"WithRoundsDecide":        "r.roundsDecideFor(pdb)",
	} {
		appels := appelsDansBlocsSessions(t, methode)
		if len(appels) != 1 || len(appels[0].portes) != 0 || !strings.Contains(strings.Join(appels[0].args, ""), arg) {
			t.Errorf("%s = %+v : attendu un appel inconditionnel avec %s", methode, appels, arg)
		}
	}
}

// Chaque source fine sous SA seule porte.
func TestSessionPage_SourcesSousLeurPorte(t *testing.T) {
	for methode, cas := range map[string]struct{ arg, porte string }{
		"WithSessionLives":        {"duckdb.NewSoloLivesRepo(pdb)", "games.CapFilmKillPositions"},
		"WithSessionVehicleUsage": {"duckdb.NewSquadVehicleRepo", "games.CapFilmVehicleUsage"},
		"WithSessionObjectives":   {"duckdb.NewObjectiveStatsRepo(pdb)", "games.CapMatchObjectiveStats"},
	} {
		appels := appelsDansBlocsSessions(t, methode)
		if len(appels) != 1 || !strings.Contains(strings.Join(appels[0].args, ""), cas.arg) ||
			len(appels[0].portes) != 1 || !strings.Contains(appels[0].portes[0], cas.porte) {
			t.Errorf("%s = %+v : attendu %s sous la seule porte %s", methode, appels, cas.arg, cas.porte)
		}
	}
}

// Le résumé d'usage reste sous la seule porte film.usage_summary, posé par le câblage des blocs ; la
// factory ne câble plus le bloc d'usage retiré (plan, S5).
func TestSessionPage_LeResumeDUsageResteSousFilmUsageSummary(t *testing.T) {
	appels := appelsDansBlocsSessions(t, "WithSessionUsageSummary")
	if len(appels) != 1 || len(appels[0].portes) != 1 || !strings.Contains(appels[0].portes[0], "games.CapFilmUsageSummary") {
		t.Fatalf("WithSessionUsageSummary = %+v : attendu un appel sous la seule porte games.CapFilmUsageSummary", appels)
	}
	if got := strings.Join(appels[0].args, ", "); got != "duckdb.NewSessionUsageRepo(pdb), r.cfg.RepoRoot" {
		t.Errorf("WithSessionUsageSummary(%s) : attendu le résumé d'usage et la racine du dépôt", got)
	}
}
