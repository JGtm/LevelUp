package games

// capabilities_front_parity_test.go — GARDE-RAIL de parité entre les CapabilityKey
// DATA-LEVEL déclarées ici et les littéraux de gating data-level du front
// (apps/web/src/lib/capabilities/dataCapabilities.ts et ses appelants).
//
// POURQUOI. Un gate posé sur une clé que le serveur n'émet JAMAIS est toujours faux : la
// feature disparaît de l'interface sans la moindre erreur, sans log, sans test rouge. Le
// dépôt ferme déjà ce piège pour les capabilities TITLE-LEVEL
// (domain/title/capabilities_parity_test.go, TestCapabilityLiteralsInFrontAreDeclaredInGo) ;
// le canal data-level n'avait aucun client jusqu'au 2026-09-05, donc rien à surveiller.
// Maintenant qu'il en a un, il a son garde.
//
// CE QU'IL SCANNE. Les deux formes de gating data-level, et elles seules :
//   - l'appel `useDataCapability('film.xxx')` (ou `hasDataCapabilityIn(..., 'film.xxx')`) ;
//   - la prop JSX / le champ d'objet `dataCapability="film.xxx"`.
// Les fichiers de test sont exclus : ils portent volontairement des clés fabriquées.
//
// CE QU'IL NE SCANNE PAS. La prop `capability=` et `useCapability(` : ce sont les clés
// TITLE-LEVEL, d'un autre système, déjà couvertes par leur propre garde. Les confondre
// ferait échouer les deux.

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// dataGatingCallRe capture le premier littéral d'un appel de gating data-level.
var dataGatingCallRe = regexp.MustCompile(`\b(?:useDataCapability|hasDataCapabilityIn)\s*\([^)]*?['"]([a-z][a-z0-9_.]*)['"]`)

// dataGatingPropRe capture la prop JSX / le champ d'objet `dataCapability`.
var dataGatingPropRe = regexp.MustCompile(`\bdataCapability\s*[:=]\s*['"]([^'"\n]+)['"]`)

// dataCapabilitiesListRe borne l'extraction au tableau DATA_CAPABILITIES.
var dataCapabilitiesListRe = regexp.MustCompile(`(?s)DATA_CAPABILITIES\s*=\s*\[(.*?)\]\s*as\s+const`)

var quotedRe = regexp.MustCompile(`['"]([^'"\n]*)['"]`)

func frontSrcDir(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) a échoué")
	}
	// apps/go-api/internal/games -> racine (4 niveaux) -> apps/web/src
	root := filepath.Join(filepath.Dir(ici), "..", "..", "..", "..")
	return filepath.Join(root, "apps", "web", "src")
}

// litterauxDataGating scanne apps/web/src et rend {clé -> occurrences "fichier:ligne"}.
func litterauxDataGating(t *testing.T, webSrc string) map[string][]string {
	t.Helper()
	out := make(map[string][]string)
	err := filepath.WalkDir(webSrc, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".tsx") {
			return nil
		}
		if strings.HasSuffix(name, ".d.ts") || strings.Contains(name, ".test.") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(filepath.Dir(webSrc), path)
		collecterDataGating(string(data), filepath.ToSlash(rel), out)
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de %s: %v", webSrc, err)
	}
	return out
}

func collecterDataGating(code, rel string, out map[string][]string) {
	noter := func(valeur string, at int) {
		out[valeur] = append(out[valeur], rel+":"+strconv.Itoa(1+strings.Count(code[:at], "\n")))
	}
	for _, re := range []*regexp.Regexp{dataGatingCallRe, dataGatingPropRe} {
		for _, m := range re.FindAllStringSubmatchIndex(code, -1) {
			noter(code[m[2]:m[3]], m[2])
		}
	}
}

// TestDataCapabilityLiteralsInFrontAreDeclaredInGo — tout littéral de gating data-level du
// front doit être une CapabilityKey connue du serveur.
func TestDataCapabilityLiteralsInFrontAreDeclaredInGo(t *testing.T) {
	litteraux := litterauxDataGating(t, frontSrcDir(t))
	if len(litteraux) == 0 {
		t.Fatal("aucun littéral de gating data-level trouvé côté front — le scan ne mord plus " +
			"(hooks renommés, arborescence apps/web déplacée) : corriger l'extracteur, ne pas " +
			"supprimer ce test")
	}

	cles := make([]string, 0, len(litteraux))
	for k := range litteraux {
		cles = append(cles, k)
	}
	sort.Strings(cles)

	for _, valeur := range cles {
		if IsKnownCapabilityKey(CapabilityKey(valeur)) {
			continue
		}
		t.Errorf("le front gate sur la capability data-level %q, qui n'est AUCUNE CapabilityKey "+
			"de games/adapter.go (gate toujours faux → surface invisible) — occurrences :\n    %s",
			valeur, strings.Join(litteraux[valeur], "\n    "))
	}
}

// TestDataCapabilitiesListIsCurrent — hygiène de la liste TS DATA_CAPABILITIES : chaque
// entrée doit être une CapabilityKey réelle ET rester effectivement consommée. Une clé
// listée « pour plus tard » est du vocabulaire mort (CLAUDE.md n°7) ; une clé retirée du Go
// sans nettoyage du TS ferait un gate mort.
func TestDataCapabilitiesListIsCurrent(t *testing.T) {
	webSrc := frontSrcDir(t)
	chemin := filepath.Join(webSrc, "lib", "capabilities", "dataCapabilities.ts")
	data, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("lecture de %s: %v", chemin, err)
	}
	bloc := dataCapabilitiesListRe.FindStringSubmatch(string(data))
	if bloc == nil {
		t.Fatalf("tableau DATA_CAPABILITIES introuvable dans %s (fichier renommé/restructuré ?)", chemin)
	}

	utilises := litterauxDataGating(t, webSrc)
	for _, m := range quotedRe.FindAllStringSubmatch(bloc[1], -1) {
		valeur := m[1]
		if !IsKnownCapabilityKey(CapabilityKey(valeur)) {
			t.Errorf("DATA_CAPABILITIES contient %q, qui n'est AUCUNE CapabilityKey de "+
				"games/adapter.go — typo, ou clé Go renommée/supprimée sans mise à jour du TS", valeur)
			continue
		}
		if len(utilises[valeur]) == 0 {
			t.Errorf("DATA_CAPABILITIES contient %q, mais AUCUN gate du front ne l'utilise — "+
				"vocabulaire mort : la brancher ou la retirer de la liste", valeur)
		}
	}
}
