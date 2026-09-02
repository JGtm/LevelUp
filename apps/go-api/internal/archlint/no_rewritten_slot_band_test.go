// no_rewritten_slot_band_test.go — DEUX RÈGLES DE BANDE, DEUX IMPLÉMENTATIONS, PAS UNE DE PLUS.
//
// CE QUE CE GARDE-RAIL EMPÊCHE DE RÉÉCRIRE. Ancrer un record delta d'objet du monde exige une
// BANDE DE SLOTS relevée aux images-clés, et il existe exactement DEUX règles pour la construire.
// Elles ne se choisissent pas au goût : elles répondent à la durée de vie de l'objet.
//
//	COMBLÉE (`filmdec.worldObjectSlotBand`)   toute la plage [min, max] des slots vus, trous
//	                                          compris, moins ceux vus porter un autre archétype.
//	                                          Pour les objets NOMBREUX ET ÉPHÉMÈRES, dont un slot
//	                                          peut vivre entre deux images-clés sans y paraître.
//	OBSERVÉE (`filmdec.observedSlotBand`)     les seuls slots vus, même exclusion. Pour les objets
//	                                          RARES ET DURABLES, présents à chaque image-clé.
//
// POURQUOI LE TEST EXISTE, ET IL A UNE DATE. Le 2026-09-01, la règle OBSERVÉE avait TROIS
// implémentations : `objectiveSlotSet` (ti=11, production), `ti11SlotSetPour` (le même code dans
// un fichier de test) et rien du tout pour ti=13, qui prenait la règle COMBLÉE faute d'avoir la
// sienne — ce qui divisait par dix le chaînage de ses records. Les trois copies ont été réduites
// à une ; sans garde-rail, la dette re-croît (règle n°6 du dépôt : à la troisième copie, on
// centralise ET on pose le garde-rail).
//
// CE QU'IL DÉTECTE. Un fichier qui marche les images-clés (`WalkKeyframeWorld`) ET tient sa propre
// liste d'exclusion (`others[...] = ...`) reconstruit une règle de bande. Hors de l'allowlist,
// c'est une quatrième copie.
//
// CE QU'IL NE PRÉTEND PAS. Une réécriture qui nommerait sa carte d'exclusion autrement passerait :
// aucun test grep ne remplace une revue. Il bloque la copie la plus probable — celle qui part du
// code existant — là où elle s'écrit.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// slotBandScope : le paquet où les bandes de slots se construisent.
var slotBandScope = filepath.Join("analysis", "filmdec")

// slotBandAllowed : les SEULS fichiers autorisés à tenir une liste d'exclusion de slots, avec la
// raison de chacun. Ajouter une entrée ici est une décision, pas une formalité.
var slotBandAllowed = map[string]string{
	// La règle COMBLÉE, avec la mesure du 2026-07-26 qui l'a établie. Elle a quitté
	// `projectiles.go` au lot 1 de PLAN_CUISSON_PERF (2026-09-02, deplacement pur : le fichier
	// passait au-dessus de 500 lignes) pour rejoindre sa jumelle observée.
	"slot_band_filled.go": "règle COMBLÉE (worldObjectSlotBand / slotBandExcluding)",
	// La règle OBSERVÉE, avec la mesure du 2026-09-01 qui l'a séparée de la précédente.
	"slot_band_observed.go": "règle OBSERVÉE (observedSlotBand)",
	// Relève seen/others dans la MÊME marche que le recensement, puis délègue la règle à
	// `slotBandExcluding` : elle ne la réécrit pas.
	"world_object_census.go": "relève seen/others puis délègue à slotBandExcluding",
	// Bande FANTÔME : un témoin d'ancrage bâti sur les slots des AUTRES archétypes. Ce n'est pas
	// une bande d'archétype, c'est son négatif — et le témoin doit rester libre de le construire.
	"equipment_creation_test.go": "bande FANTÔME (témoin d'ancrage, pas une bande d'archétype)",
}

var (
	slotBandWalkRE    = regexp.MustCompile(`WalkKeyframeWorld\(`)
	slotBandExcludeRE = regexp.MustCompile(`\bothers\[[^\]]*\]\s*=`)
)

// TestNoRewrittenSlotBand refuse une quatrième implémentation de la règle de bande.
func TestNoRewrittenSlotBand(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	root := filepath.Join(filepath.Dir(filepath.Dir(thisFile)), slotBandScope)
	if _, err := os.Stat(root); err != nil {
		t.Skipf("paquet %s absent : ce test n'a pas d'objet ici", slotBandScope)
	}
	var violations []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !slotBandBuildsBand(string(src)) {
			return nil
		}
		if _, allowed := slotBandAllowed[d.Name()]; allowed {
			return nil
		}
		violations = append(violations, d.Name())
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de %s : %v", root, err)
	}
	if len(violations) == 0 {
		return
	}
	t.Fatalf("règle de bande de slots RÉÉCRITE dans %d fichier(s) : %s\n"+
		"Deux règles existent et suffisent : `worldObjectSlotBand` (comblée, objets nombreux et"+
		" éphémères) et `observedSlotBand` (observée, objets rares et durables). Appeler la"+
		" bonne, ou justifier une entrée de plus dans `slotBandAllowed`.",
		len(violations), strings.Join(violations, ", "))
}

// slotBandBuildsBand dit qu'un fichier marche les images-clés ET tient sa propre liste
// d'exclusion de slots — la signature d'une règle de bande réécrite. Les commentaires ont le
// droit de citer l'un et l'autre : seul le code compte.
func slotBandBuildsBand(src string) bool {
	walks, excludes := false, false
	for _, line := range strings.Split(src, "\n") {
		code := strings.TrimSpace(line)
		if strings.HasPrefix(code, "//") {
			continue
		}
		if slotBandWalkRE.MatchString(code) {
			walks = true
		}
		if slotBandExcludeRE.MatchString(code) {
			excludes = true
		}
	}
	return walks && excludes
}
