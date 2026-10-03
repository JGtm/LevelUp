package grammar

// ecs_dispatch_table_guard_test.go — LE CONTROLE G6 DE LA TABLE ECS : un nom de composant que le
// jeu enregistre sous plusieurs tables de grammaires differentes se lit par la TABLE de
// l archetype, jamais par le nom seul.
//
// D OU VIENT LA TABLE. Chaque ligne de `testdata/ecs_table.tsv` porte en `deser_addr` le lecteur
// de la table que la fonction d enregistrement de l archetype pose a cet index
// (`FUN_14064dd28(archetype + 8, index, &objet)`, `+0x4754 = ti`). Deux lignes d un meme nom a
// deux lecteurs differents sont donc un HOMONYME de grammaire : le dispatch, qui route par nom,
// doit y choisir le lecteur par l archetype du record.
//
// CE QUE LE CONTROLE EXIGE, pour chaque homonyme :
//   - qu il soit dans la liste recensee ci-dessous (un homonyme neuf se route avant d entrer) ;
//   - que chacune de ses lignes porte une largeur ENTIERE et que le dispatch la consomme sur les
//     trois motifs de G4 (une lecture par le nom seul donnerait la meme largeur aux deux lignes) ;
//   - qu un archetype absent de ses lignes ne le lise pas (aucun repli sur la grammaire d un autre
//     archetype).

import (
	"slices"
	"strings"
	"testing"
)

// homonymesDeGrammaire : les noms enregistres sous deux tables de grammaires differentes, recenses
// dans le binaire sur les 326 noms des registres du corpus (`simulation-state-component` et
// `simulation-state-playback-component` ont deux tables mais un seul lecteur derriere leurs
// thunks : ce ne sont pas des homonymes de grammaire).
var homonymesDeGrammaire = []string{compHighFrequency}

// homonymesDeLaTable rend, par nom, les lignes de la table dont le nom porte au moins deux
// lecteurs distincts (adresse comparee sans casse).
func homonymesDeLaTable(rows []ecsRow) map[string][]ecsRow {
	lecteurs := map[string]map[string]bool{}
	parNom := map[string][]ecsRow{}
	for _, r := range rows {
		d := strings.ToLower(r.DeserAddr)
		if r.TI < 0 || d == "" {
			continue
		}
		if lecteurs[r.Component] == nil {
			lecteurs[r.Component] = map[string]bool{}
		}
		lecteurs[r.Component][d] = true
		parNom[r.Component] = append(parNom[r.Component], r)
	}
	out := map[string][]ecsRow{}
	for nom, l := range lecteurs {
		if len(l) > 1 {
			out[nom] = parNom[nom]
		}
	}
	return out
}

// TestG6LesHomonymesSeRoutentParTable — LE CONTROLE.
func TestG6LesHomonymesSeRoutentParTable(t *testing.T) {
	rows := loadECSTable(t)
	homonymes := homonymesDeLaTable(rows)
	noms := make([]string, 0, len(homonymes))
	for nom := range homonymes {
		noms = append(noms, nom)
	}
	slices.Sort(noms)
	if !slices.Equal(noms, homonymesDeGrammaire) {
		t.Fatalf("G6 : homonymes de grammaire de la table %v, recenses %v — un nom a deux lecteurs "+
			"se route par la table de l archetype (cf. components_frequences.go) avant d entrer ici",
			noms, homonymesDeGrammaire)
	}
	for _, nom := range noms {
		porteurs := map[int]bool{}
		for _, r := range homonymes[nom] {
			porteurs[r.TI] = true
			if r.BitsTyp < 0 {
				t.Errorf("G6 : ligne %d (ti=%d i=%d %s) : un homonyme porte une largeur entiere", r.LineNo, r.TI, r.I, nom)
				continue
			}
			want := [3]int{r.BitsTyp, r.BitsTyp, r.BitsTyp}
			if got := ecsLargeursParMotif(r); got != want {
				t.Errorf("G6 : ligne %d (ti=%d i=%d %s) : le dispatch consomme %v, la table de cet archetype "+
					"annonce %d bits", r.LineNo, r.TI, r.I, nom, got, r.BitsTyp)
			}
		}
		for ti := range 64 {
			if porteurs[ti] {
				continue
			}
			r := ecsRow{TI: ti, Component: nom}
			if got := ecsLargeursParMotif(r); got != [3]int{-1, -1, -1} {
				t.Errorf("G6 : ti=%d ne declare pas %s, et le dispatch le lit pourtant (%v bits) : "+
					"repli sur la grammaire d un autre archetype", ti, nom, got)
			}
		}
	}
}
