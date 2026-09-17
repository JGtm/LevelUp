package objectives

// families_parite_ts_test.go — RATCHET DE PARITÉ Go ↔ TypeScript (G.5a des
// finitions v7.5, 2026-09-13 ; constat R6 de la revue E.3).
//
// LE DÉFAUT FERMÉ ICI. Les familles d'objectif vivent en DEUX listes indépendantes :
// `objectiveFamilies` (ce paquet) et `OBJECTIVE_FAMILIES`
// (`apps/web/src/features/match-replay/model/objectiveFamilies.ts`). Aucun test ne
// les comparait, et le TSDoc du fichier web AFFIRMAIT pourtant que la liste était
// « stable et fermée côté serveur (garde-rail Go) ». Conséquence mesurée par la
// revue : ajouter une 7e famille au décodeur rend le Go rouge (bien), laisse le TS
// VERT, et le calque cesse SILENCIEUSEMENT de dessiner les pulses de cette famille.
//
// CE QUE LE RATCHET COMPARE : la liste ET son ordre, plus le type union
// `ObjectiveFamily` qui la borne côté TS. Les deux fichiers se lisent côte à côte ;
// un écart d'ordre est le premier signe qu'une liste a été éditée sans l'autre.
//
// CE QU'IL NE PEUT PAS FAIRE : vérifier que le TS EMPLOIE la liste. C'est le rôle
// de `objectiveFamilies.test.ts` côté web.

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// cheminFamillesTS — le fichier web, relatif à la racine du dépôt.
var cheminFamillesTS = filepath.Join(
	"apps", "web", "src", "features", "match-replay", "model", "objectiveFamilies.ts")

var (
	reTableauFamillesTS = regexp.MustCompile(
		`(?s)export const OBJECTIVE_FAMILIES:[^=]*=\s*\[(.*?)\]`)
	reUnionFamillesTS = regexp.MustCompile(
		`export type ObjectiveFamily\s*=\s*([^\n]+)`)
	reChaineTS = regexp.MustCompile(`'([a-z_]+)'`)
)

// racineDepot remonte de ce paquet jusqu'à la racine du dépôt (celle qui porte
// `apps/`), sans jamais bâtir de chemin « data/… » à la main.
func racineDepot(t *testing.T) string {
	t.Helper()
	_, fichier, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	dir := filepath.Dir(fichier)
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "apps", "web", "package.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("racine du dépôt introuvable depuis %s", filepath.Dir(fichier))
	return ""
}

func TestFamillesObjectifPariteGoTS(t *testing.T) {
	chemin := filepath.Join(racineDepot(t), cheminFamillesTS)
	source, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("lecture de %s : %v — si le fichier a été déplacé, corriger cheminFamillesTS "+
			"(la parité n'a pas le droit de devenir un test qui ne lit plus rien)", cheminFamillesTS, err)
	}
	texte := string(source)

	bloc := reTableauFamillesTS.FindStringSubmatch(texte)
	if bloc == nil {
		t.Fatalf("%s : tableau OBJECTIVE_FAMILIES introuvable — la forme du fichier a changé, "+
			"le ratchet ne compare plus rien", cheminFamillesTS)
	}
	famillesTS := valeursTS(bloc[1])

	if len(famillesTS) != len(objectiveFamilies) {
		t.Fatalf("familles TS = %v (%d), familles Go = %v (%d) — une 7e famille ajoutée d'un "+
			"seul côté fait cesser l'autre de la traiter, en silence",
			famillesTS, len(famillesTS), objectiveFamilies, len(objectiveFamilies))
	}
	for i := range objectiveFamilies {
		if famillesTS[i] != objectiveFamilies[i] {
			t.Errorf("famille n°%d : TS = %q, Go = %q (les deux listes se lisent côte à côte, "+
				"ordre compris)", i, famillesTS[i], objectiveFamilies[i])
		}
	}

	// Le type union borne ce que le TS accepte : une famille présente dans le
	// tableau mais absente de l'union ne compilerait pas, l'inverse passerait.
	union := reUnionFamillesTS.FindStringSubmatch(texte)
	if union == nil {
		t.Fatalf("%s : type ObjectiveFamily introuvable", cheminFamillesTS)
	}
	unionTS := valeursTS(union[1])
	if strings.Join(unionTS, ",") != strings.Join(objectiveFamilies, ",") {
		t.Errorf("type ObjectiveFamily = %v, familles Go = %v", unionTS, objectiveFamilies)
	}
}

// valeursTS extrait les chaînes simples quotées d'un fragment TypeScript.
func valeursTS(fragment string) []string {
	out := []string{}
	for _, m := range reChaineTS.FindAllStringSubmatch(fragment, -1) {
		out = append(out, m[1])
	}
	return out
}

// TestFamillesObjectifPariteMordSurUnEcart — le témoin : la comparaison doit
// ÉCHOUER sur une liste amputée ou réordonnée, sinon le ratchet est décoratif.
func TestFamillesObjectifPariteMordSurUnEcart(t *testing.T) {
	reference := strings.Join(objectiveFamilies, ",")
	cas := map[string][]string{
		"famille en moins": objectiveFamilies[:len(objectiveFamilies)-1],
		"famille en plus":  append(append([]string{}, objectiveFamilies...), "assaut_2"),
		"ordre différent": append([]string{objectiveFamilies[1], objectiveFamilies[0]},
			objectiveFamilies[2:]...),
	}
	for nom, liste := range cas {
		if strings.Join(liste, ",") == reference {
			t.Errorf("%s : la comparaison ne distingue pas ce cas — elle ne mordrait pas", nom)
		}
	}
	// Et le contrôle positif : la liste extraite du TS réel EST la liste Go.
	chemin := filepath.Join(racineDepot(t), cheminFamillesTS)
	source, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("lecture de %s : %v", cheminFamillesTS, err)
	}
	bloc := reTableauFamillesTS.FindStringSubmatch(string(source))
	if bloc == nil {
		t.Fatalf("%s : tableau OBJECTIVE_FAMILIES introuvable", cheminFamillesTS)
	}
	if got := strings.Join(valeursTS(bloc[1]), ","); got != reference {
		t.Fatalf("extraction TS = %q, Go = %q", got, reference)
	}
}
