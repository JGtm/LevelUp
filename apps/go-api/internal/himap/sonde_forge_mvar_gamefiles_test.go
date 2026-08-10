package himap

// SONDE (2026-08-10) — les champs du .mvar que la grammaire ne parse pas, et l'ECHELLE.
//
// Le piege annonce de F2 : l'echelle d'instance sbsp etait « reputee vestigiale » et ne
// l'etait pas (deux jours payes). Ici la mesure dit l'INVERSE : le .mvar de Vagabond ne
// porte AUCUNE echelle — champ objet [6] absent, champ [9] = struct VIDE sur 4 709/4 709
// objets. La pose d'objet Forge est donc a echelle unitaire, sur pieces.

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

func mustRoot(ti tagInfo) int {
	r, err := ti.rootBlockIndex()
	if err != nil {
		return 0
	}
	return r
}

// TestSondeForgeChampsMvar — le PIEGE de F2 : l'echelle. La grammaire mvar ne parse pas
// tous les champs des objets ([6] et [9] inconnus). Inventaire des champs presents et
// dump du champ 6 — si c'est un vec3 proche de (1,1,1), c'est l'echelle d'instance.
func TestSondeForgeChampsMvar(t *testing.T) {
	chemin, err := cheminDepuisDepot(".ai/re_dump/mapvar/vagabond_map.mvar")
	if err != nil {
		t.Skip(err)
	}
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
	if err != nil {
		t.Fatal(err)
	}
	root, err := mapvar.DecodeRoot(brut)
	if err != nil {
		t.Fatal(err)
	}
	objs, ok := root.Field(3)
	if !ok {
		t.Fatal("champ racine 3 absent")
	}
	champs := map[uint16]int{}
	types6 := map[byte]int{}
	var val6 []float64
	for _, o := range objs.Items {
		for id := range o.Fields {
			champs[id]++
		}
		if f6, ok := o.Field(6); ok {
			types6[f6.Type]++
			if x, ok := f6.Field(0); ok && len(val6) < 30 {
				y, _ := f6.Field(1)
				z, _ := f6.Field(2)
				val6 = append(val6, x.Float, y.Float, z.Float)
			}
		}
	}
	var ids []int
	for id := range champs {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	for _, id := range ids {
		t.Logf("  champ [%d] present sur %d/%d objets", id, champs[uint16(id)], len(objs.Items))
	}
	t.Logf("champ [6] : types Bond %v · premieres valeurs (x,y,z) %v", types6, val6)

	// Champ [9] : type, sous-champs, distribution des valeurs. Si c'est un vec3 autour de
	// (1,1,1) ou un float autour de 1, c'est l'ECHELLE — le piege paye deux jours cote sbsp.
	types9 := map[byte]int{}
	sousChamps9 := map[uint16]int{}
	valeurs9 := map[string]int{}
	for _, o := range objs.Items {
		f9, ok := o.Field(9)
		if !ok {
			continue
		}
		types9[f9.Type]++
		for id, sv := range f9.Fields {
			sousChamps9[id]++
			if len(valeurs9) < 40 {
				valeurs9[fmt.Sprintf("[%d]=t%d i%d u%d f%.4g s%q", id, sv.Type, sv.Int, sv.Uint, sv.Float, sv.Str)]++
			}
		}
		if f9.Type == btListe9 && len(valeurs9) < 40 {
			valeurs9[fmt.Sprintf("liste de %d items", len(f9.Items))]++
		}
	}
	t.Logf("champ [9] : types Bond %v · sous-champs %v", types9, sousChamps9)
	for v, n := range valeurs9 {
		t.Logf("  [9] %s : x%d", v, n)
	}
}

// btListe9 : valeur du type BT_LIST du format Bond (cf. cb2.go).
const btListe9 = 11
