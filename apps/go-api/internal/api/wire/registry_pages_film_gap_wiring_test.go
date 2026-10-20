package wire

// registry_pages_film_gap_wiring_test.go — LE CÂBLAGE DE « L'ANALYSE DU FILM EN COURS » des pages qui
// lisent le film (Sessions, Séries temporelles, Synthèse).
//
// Un `WithAnalysisPending` oublié ne rougit aucun test de service : le service dit alors « film non
// analysé » pour des matchs dont l'analyse tourne. L'option est lue dans l'arbre syntaxique de chaque
// factory — son argument, et l'absence de porte `if` autour (la porte de titre est dans
// `analysisPendingFor`, jamais ici).

import (
	"strings"
	"testing"
)

func TestFilmGap_PagesDuFilmCablentLeResolveurSansCondition(t *testing.T) {
	cas := []struct{ fichier, factory string }{
		{"registry_pages.go", "SessionPage"},
		{"registry_pages.go", "Timeseries"},
		{"registry_pages_home.go", "SynthesisCtx"},
	}
	for _, c := range cas {
		appels := appelsDansFactory(t, c.fichier, c.factory, "WithAnalysisPending")
		if len(appels) != 1 {
			t.Errorf("%s : %d appel(s) à WithAnalysisPending, attendu 1", c.factory, len(appels))
			continue
		}
		if got := strings.Join(appels[0].args, ", "); got != "r.analysisPendingFor(pdb)" {
			t.Errorf("%s : WithAnalysisPending(%s), attendu r.analysisPendingFor(pdb)", c.factory, got)
		}
		if len(appels[0].portes) != 0 {
			t.Errorf("%s : WithAnalysisPending sous condition (%v)", c.factory, appels[0].portes)
		}
	}
}
