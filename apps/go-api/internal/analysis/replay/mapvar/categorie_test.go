package mapvar

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// LES COQUES SONT-ELLES DES BLOCS, OU AUTRE CHOSE ? — hypothese de l'utilisateur, 2026-08-27 :
// « ce ne serait pas des effets de lumiere ces trucs qui nous posent probleme ? Ce doit etre
// des elements qui ne sont pas des blocs, peut-etre un genre de mur transparent ».
//
// Forge classe chaque objet par CATEGORIE, et le parseur la lit deja (champ 1 du sac gameplay,
// Object.Category) sans que personne s'en serve. Si les 32 pieces qui peignent 82,7 % de
// l'image d'Isolation portent une categorie differente des pieces de structure, on tient une
// regle universelle — ne pas dessiner cette categorie — plutot qu'une liste de types par carte.
//
// Le test ne conclut pas : il croise categorie et drapeaux avec le type fautif.
func TestCategorieDesCoquesForge(t *testing.T) {
	chemin := filepath.Join("C:\\", "Users", "Guillaume", "Projects", "LevelUp",
		".ai", "re_dump", "mapvar", "isolation_map.mvar")
	brut, err := os.ReadFile(chemin)
	if err != nil {
		t.Skipf("variante absente : %v", err)
	}
	v, err := Parse(brut)
	if err != nil {
		t.Fatalf("variante illisible : %v", err)
	}

	const typeCoque = -1342618612 // peint 82,7 % de l'image, 32 exemplaires, poses vers Z 139
	const typeSecond = 1574763282 // peint 46,7 % une fois le dome retire, 3 exemplaires

	parCategorie := map[int]int{}
	parDrapeau := map[uint8]int{}
	for _, o := range v.Objects {
		parCategorie[o.Category]++
		parDrapeau[o.Flags]++
	}
	t.Logf("categories sur %d objets : %s", len(v.Objects), tri(parCategorie))
	t.Logf("drapeaux (champ 7) : %s", triU8(parDrapeau))

	for _, cible := range []int32{typeCoque, typeSecond} {
		cats, flags := map[int]int{}, map[uint8]int{}
		n := 0
		for _, o := range v.Objects {
			if o.TypeID != cible {
				continue
			}
			n++
			cats[o.Category]++
			flags[o.Flags]++
		}
		t.Logf("type %d (%d exemplaires) : categories %s | drapeaux %s", cible, n, tri(cats), triU8(flags))
	}
}

func tri(m map[int]int) string {
	type kv struct {
		k, v int
	}
	var l []kv
	for k, v := range m {
		l = append(l, kv{k, v})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].v > l[j].v })
	s := ""
	for i, x := range l {
		if i >= 8 {
			break
		}
		s += formate(x.k, x.v)
	}
	return s
}

func triU8(m map[uint8]int) string {
	c := map[int]int{}
	for k, v := range m {
		c[int(k)] = v
	}
	return tri(c)
}

func formate(k, v int) string {
	return fmt.Sprintf(" %d x%d", k, v)
}
