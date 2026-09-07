package replaybuild

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// GARDE-RAIL : tout champ de `filmStats` doit etre CABLE dans les `replay.Options` de
// `BuildMatch` (constat C1 de la revue VIES-R1, 2026-09-07).
//
// LE DEFAUT QU'IL FERME. `objectivesUnnamed` a ete ecrit par `readFilmStats` et n'a JAMAIS ete
// lu : le litteral `replay.Options{...}` ne le passait pas. Consequence sur l artefact SERVI —
// `coverage.objectives.available` restait un compte de RESCAPES et `coverage.objectives.noSlot`
// restait structurellement a zero, c est-a-dire exactement le defaut que le lot declarait
// corriger. Le champ etait mort (anti-patron n 1).
//
// POURQUOI AUCUN AUTRE GATE NE L ATTRAPE. `go vet` et `golangci-lint` ne signalent pas un champ
// de structure inutilise (contrairement a une variable locale), et un test unitaire qui
// reconstruirait le litteral resterait vert quoi qu il arrive : il faut lire LA SOURCE de
// l assemblage. C est le meme raisonnement que `published_tracks_guard_test.go`.
//
// CE QU IL N EXIGE PAS : que le champ soit passe SOUS SON NOM. Il exige que le nom du champ
// apparaisse dans le litteral d Options — `Objectives: stats.objectives` couvre `objectives`,
// `ObjectivesUnnamed: stats.objectivesUnnamed` couvre `objectivesUnnamed`. Un champ
// deliberement non cable s ajoute a `champsNonCables` AVEC SA RAISON.

// champsNonCables : les champs de `filmStats` qui n ont pas a rejoindre `replay.Options`, avec
// la raison de chacun. Toute entree ajoutee ici est une decision, pas un oubli.
var champsNonCables = map[string]string{}

func TestChaqueChampDeFilmStatsEstCableDansOptions(t *testing.T) {
	src, err := os.ReadFile(filepath.Clean("replaybuild.go"))
	if err != nil {
		t.Fatalf("lecture de replaybuild.go : %v", err)
	}
	litteral := optionsLitteral(t, string(src))

	typ := reflect.TypeOf(filmStats{})
	var orphelins []string
	for i := 0; i < typ.NumField(); i++ {
		nom := typ.Field(i).Name
		if raison, exempte := champsNonCables[nom]; exempte {
			t.Logf("champ %s non cable, raison : %s", nom, raison)
			continue
		}
		if !strings.Contains(litteral, "stats."+nom) {
			orphelins = append(orphelins, nom)
		}
	}
	if len(orphelins) > 0 {
		t.Errorf("champ(s) de filmStats ECRIT(S) mais jamais passe(s) a replay.Options : %v — "+
			"un champ mort ne fait echouer aucun autre gate, et la couverture publiee reste "+
			"fausse en silence (cf. C1). Cabler, ou justifier dans champsNonCables", orphelins)
	}
}

// optionsLitteral extrait le corps du litteral `replay.Options{...}` de `BuildMatch`.
//
// LE GARDE-RAIL DOIT POUVOIR ECHOUER : si le litteral n est plus reconnu, le test s arrete au
// lieu de passer a vide (lecon J4.0 — un garde qui ne peut pas echouer ne garde rien).
func optionsLitteral(t *testing.T, src string) string {
	t.Helper()
	debut := regexp.MustCompile(`replay\.BuildFromFilm\([^)]*replay\.Options\{`).FindStringIndex(src)
	if debut == nil {
		t.Fatal("litteral replay.Options{...} de BuildMatch introuvable : le garde-rail ne " +
			"verifie plus rien (l assemblage a-t-il ete deplace ?)")
	}
	reste := src[debut[1]:]
	profondeur := 1
	for i, r := range reste {
		switch r {
		case '{':
			profondeur++
		case '}':
			if profondeur--; profondeur == 0 {
				return reste[:i]
			}
		}
	}
	t.Fatal("litteral replay.Options{...} non ferme")
	return ""
}
