package mapvar

// frag_types_test.go — SONDE : de quoi est fait le fichier de carte d'une carte DEV a vehicules.
//
// Le scenario `levl` ne porte AUCUN placement de vehicule (mesure du 2026-08-31 : ses blocs
// referencent `scen`, `bloc`, `mach`, `lens`, `licn`, `lsnd`, `effe`, `ligr`, `bitm` — jamais
// `vehi` ni `weap`). Les vehicules sont donc dans le FICHIER DE CARTE apres tout ; si le
// balayage par les 21 `type_id` de la palette Forge n'en a trouve qu'UN sur Fragmentation, c'est
// que ces `type_id` ne couvrent pas ce que les cartes DEV posent.
//
// Cette sonde imprime l'histogramme COMPLET des `type_id` de la carte, avec ce que la palette en
// dit — et surtout ce qu'elle n'en dit pas.
//
//	$env:FRAG_MVAR="C:/.../mvar/<map_id>/btb_fragmentation.mvar"
//	go test ./internal/analysis/replay/mapvar/ -run FragTypes -v

import (
	"os"
	"sort"
	"testing"
)

func TestFragTypes(t *testing.T) {
	chemin := os.Getenv("FRAG_MVAR")
	if chemin == "" {
		t.Skip("mesure non demandee : FRAG_MVAR requis")
	}
	buf, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}
	v, err := Parse(buf)
	if err != nil {
		t.Fatalf("parse : %v", err)
	}
	compte := map[int32]int{}
	for _, o := range v.Objects {
		compte[o.TypeID]++
	}
	type l struct {
		t int32
		n int
	}
	ls := make([]l, 0, len(compte))
	for k, n := range compte {
		ls = append(ls, l{k, n})
	}
	sort.Slice(ls, func(i, j int) bool { return ls[i].n > ls[j].n })
	t.Logf("%d objets, %d type_id distincts ; %d noms lisibles", len(v.Objects), len(compte), len(v.Names))
	for i, x := range ls {
		if i >= 40 {
			break
		}
		nom, connu := vehiTypes[x.t]
		if !connu {
			nom = "-"
		}
		fam, estSocle := PadFamilyOf(x.t)
		if !estSocle {
			fam = "-"
		}
		t.Logf("  type_id %-12d x%-4d  vehi=%-20s socle=%s", x.t, x.n, nom, fam)
	}
}
