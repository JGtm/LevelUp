package wire

// registry_pages_matchview_wiring_test.go — LE CABLAGE DES BLOCS DU FILM DE LA VUE MATCH (plan
// `.ai/V7.5/PLAN_MATCHVIEW_EMPRISE_2026-10-06.md`, D18). Meme mode de panne que l'Escouade et les Series
// temporelles : le service degrade EN SILENCE sur une dependance nil. Les options sont lues dans
// l'arbre syntaxique — leur argument et la porte `if` qui les entoure.

import (
	"strings"
	"testing"
)

func appelsDansFilmMatchView(t *testing.T, methode string) []appelOptionCable {
	t.Helper()
	return appelsDansFactory(t, "registry_pages_matchview.go", "cablerFilmMatchView", methode)
}

// La factory appelle le câblage SANS condition.
func TestMatchView_AppelleLeCablageDuFilmSansCondition(t *testing.T) {
	appels := appelsDansFactory(t, "registry_pages.go", "MatchView", "cablerFilmMatchView")
	if len(appels) != 1 || len(appels[0].portes) != 0 {
		t.Errorf("cablerFilmMatchView = %+v : attendu un appel inconditionnel dans MatchView", appels)
	}
}

// Feuille de match et portée du radar : sans condition, sur le PlayerDB du joueur.
func TestMatchView_CableFeuilleEtPorteeSansCondition(t *testing.T) {
	for methode, arg := range map[string]string{
		"WithEmpriseSheet": "duckdb.NewSquadEmpriseRepo(pdb, r.killSourceClassifierFor(pdb))",
		"WithRadarRange":   "r.radarRangeFor(pdb)",
	} {
		appels := appelsDansFilmMatchView(t, methode)
		if len(appels) != 1 || strings.Join(appels[0].args, ", ") != arg || len(appels[0].portes) != 0 {
			t.Errorf("%s = %+v : attendu un appel inconditionnel %s(%s)", methode, appels, methode, arg)
		}
	}
}

// Résumé d'usage, vies, véhicules : chacun sous SA seule porte de capability.
func TestMatchView_CableChaqueSourceSousSaPorte(t *testing.T) {
	for methode, porte := range map[string]string{
		"WithEmpriseUsageSummary": "games.CapFilmUsageSummary",
		"WithCampLives":           "games.CapFilmKillPositions",
		"WithEmpriseVehicles":     "games.CapFilmVehicleUsage",
	} {
		appels := appelsDansFilmMatchView(t, methode)
		if len(appels) != 1 || len(appels[0].portes) != 1 || !strings.Contains(appels[0].portes[0], porte) {
			t.Errorf("%s = %+v : attendu un appel sous la seule porte %s", methode, appels, porte)
		}
	}
	vies := appelsDansFilmMatchView(t, "WithCampLives")
	if len(vies) == 1 && strings.Join(vies[0].args, "") != "duckdb.NewSoloLivesRepo(pdb)" {
		t.Errorf("WithCampLives(%s) : attendu duckdb.NewSoloLivesRepo(pdb)", strings.Join(vies[0].args, ""))
	}
}
