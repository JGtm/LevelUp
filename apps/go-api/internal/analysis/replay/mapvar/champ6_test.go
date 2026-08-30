package mapvar

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// QUE CONTIENT LE CHAMP 6 D'UN OBJET FORGE ? — hypothese de l'utilisateur, 2026-08-27 :
// « ce gribouillis vient peut-etre d'un parametre qu'on interprete alors qu'on ne devrait pas ».
//
// L'inverse est aussi possible : un parametre qu'on NE lit PAS. `parseObject` lit les champs
// 2 (type), 3 (position), 4 (up), 5 (forward), 7 (drapeaux), 8 (sac gameplay) et 10
// (instance). Le champ 6 est SAUTE, et c'est la place ou un moteur range typiquement
// l'ECHELLE — que la cuisson force aujourd'hui a 1 pour tous les objets Forge
// (`himap.InstanceForge`). Si Forge permet de redimensionner une piece et que l'echelle vit
// la, chaque objet est dessine au mauvais gabarit.
//
// Le test ne conclut pas : il decrit ce que le champ contient, sur une carte reelle.
func TestChamp6DesObjetsForge(t *testing.T) {
	chemin := filepath.Join("C:\\", "Users", "Guillaume", "Projects", "LevelUp", ".ai", "re_dump", "mapvar", "isolation_map.mvar")
	brut, err := os.ReadFile(chemin)
	if err != nil {
		t.Skipf("variante absente : %v", err)
	}
	v, err := Parse(brut)
	if err != nil {
		t.Fatalf("variante illisible : %v", err)
	}
	t.Logf("%d objets", len(v.Objects))

	// Le parseur ne conserve pas l'arbre brut : on le reparcourt ici.
	root, err := DecodeRoot(brut)
	if err != nil {
		t.Skipf("arbre brut indisponible : %v", err)
	}
	objs, ok := root.Field(3)
	if !ok {
		t.Skip("liste d objets introuvable")
	}
	racine := objs.Items
	types := map[byte]int{}
	var flottants []float64
	nAbsent := 0
	for _, o := range racine {
		f, ok := o.Field(6)
		if !ok {
			nAbsent++
			continue
		}
		types[f.Type]++
		if f.Float != 0 {
			flottants = append(flottants, f.Float)
		}
		for k := uint16(0); k < 4; k++ {
			if c, ok := f.Field(k); ok && c.Float != 0 {
				flottants = append(flottants, c.Float)
			}
		}
	}
	t.Logf("champ 6 absent sur %d objets ; types rencontres : %v", nAbsent, types)
	if len(flottants) == 0 {
		t.Log("aucun flottant non nul dans le champ 6")
		return
	}
	sort.Float64s(flottants)
	t.Logf("flottants du champ 6 : n=%d  min=%.4f  p50=%.4f  p95=%.4f  max=%.4f",
		len(flottants), flottants[0], flottants[len(flottants)/2],
		flottants[len(flottants)*95/100], flottants[len(flottants)-1])
}
