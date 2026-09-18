package grammar

// component_param4_ratchet_test.go — LE RATCHET DE `paramByComponent` (lot 5.1.7, 2026-09-18).
//
// # CE QU IL TIENT
//
// `param_4` EST le niveau que le registre du film porte par composant, et la production le LIT
// desormais par ce chemin-la : `traverseComponentLoopFrom` passe `arch.Level(i)` a
// `consumeByName`, qui le donne aux huit deserialiseurs qui branchent dessus. La table
// `paramByComponent` n est plus une source — il lui reste UN appelant, celui qui n a pas
// d archetype sous la main (`offline_aim.go`, via [paramMesureDuComposant]).
//
// UNE TABLE QUI N EST PLUS LA SOURCE MAIS QUI SERT ENCORE PEUT DERIVER EN SILENCE. Ce test
// l interdit : chaque entree doit valoir le `level` que `testdata/ecs_table.tsv` donne au meme
// composant — la colonne que la porte `-update-ecs-table-level` regenere depuis le registre d un
// vrai film. Une valeur ecrite a la main ne peut donc plus contredire ce que le film ecrit.
//
// IL MORD DANS LES DEUX SENS : une entree qui s ecarte du registre rougit, et un composant qui
// porte DEUX niveaux differents selon l archetype rougit aussi — ce serait la preuve que
// `param_4` n est pas une propriete du composant, donc que le raisonnement de ce lot est faux.

import (
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestParamByComponentEgaleLeNiveauDuRegistre confronte les entrees de la table au `level` du
// registre, ligne a ligne.
func TestParamByComponentEgaleLeNiveauDuRegistre(t *testing.T) {
	niveaux := niveauxParComposant(t)
	vus := 0
	for nom, v := range paramByComponent {
		lv, ok := niveaux[nom]
		if !ok {
			// Un ALIAS de nom (les lignes `ti = -1` de la table) n a pas de niveau : la table du
			// depot en porte plusieurs, et ils partagent la valeur de leur nom canonique.
			continue
		}
		vus++
		if v != lv {
			t.Errorf("%s : la table dit param_4 = %d, le registre du film dit level = %d.\n"+
				"`param_4` EST le niveau du registre (lot 5.1.7) : la table ne peut pas le "+
				"contredire. Si le registre a change, regenerer la colonne par sa porte "+
				"(`-update-ecs-table-level`) et corriger l entree dans le MEME commit.", nom, v, lv)
		}
	}
	if vus < 15 {
		t.Fatalf("seulement %d entrees confrontees au registre : la table ou le TSV a change de "+
			"forme, et le ratchet ne garde plus rien", vus)
	}
}

// niveauxParComposant rend le `level` de chaque composant NOMME de la table ECS, et echoue si un
// composant en porte deux — `param_4` est une propriete du COMPOSANT, pas du couple
// (archetype, composant).
func niveauxParComposant(t *testing.T) map[string]uint32 {
	t.Helper()
	out := map[string]uint32{}
	ou := map[string][]string{}
	for _, r := range loadECSTable(t) {
		if r.TI < 0 || r.Component == "" {
			continue // ligne d alias : ni archetype ni niveau
		}
		ou[r.Component] = append(ou[r.Component], "ti="+strconv.Itoa(r.TI)+" i"+
			strconv.Itoa(r.I)+" level="+strconv.FormatUint(uint64(r.Level), 10))
		if prev, dup := out[r.Component]; dup && prev != r.Level {
			sort.Strings(ou[r.Component])
			t.Fatalf("%s porte DEUX niveaux dans le registre : %s.\n"+
				"`param_4` serait alors une propriete du couple (archetype, composant) et non du "+
				"composant : le raisonnement du lot 5.1.7 tomberait, et la table par nom avec lui.",
				r.Component, strings.Join(ou[r.Component], " · "))
		}
		out[r.Component] = r.Level
	}
	return out
}
