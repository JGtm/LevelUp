package mapvar

// socles_candidats_test.go — TOUS LES SOCLES DE LA CARTE, Y COMPRIS CEUX QUE LE FILM NE
// VOIT PAS.
//
// C'EST LA DEMANDE DE DEPART. La detection en production exige deux apparitions au meme
// point dans le film : un socle servi une seule fois est invisible. La phase 2 a etabli
// que les socles sont dans le `.mvar` au centimetre ; ce fichier en tire la consequence —
// on part des type_id que l'oracle a valides, on enumere TOUS les objets de ces types, et
// on marque ceux que le film n'a jamais montres.
//
// LECTURE SEULE. Gardes : `MVAR_FILE` + `MVAR_ORACLE` (+ `MVAR_ORACLE_POINTS`).

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestSoclesCandidats enumere les socles du fichier de carte et les confronte a l'oracle.
func TestSoclesCandidats(t *testing.T) {
	v, path := soclesVariante(t)
	pts, _ := soclesChargeOracle(t)

	// 1. Les type_id que l'oracle valide : ceux des objets apparies a moins d'un metre.
	types := map[int32][]int{}
	vus := map[int]string{}
	for _, p := range pts {
		idx, d, _ := soclesPlusProche(v.Objects, p.Pos)
		if d >= soclesSeuilM {
			continue
		}
		o := v.Objects[idx]
		types[o.TypeID] = append(types[o.TypeID], idx)
		vus[idx] = p.Arme
		if p.Arme == "" {
			vus[idx] = "(power-up)"
		}
	}
	if len(types) == 0 {
		t.Fatalf("%s : aucun objet apparie, rien a enumerer", filepath.Base(path))
	}

	cles := make([]int32, 0, len(types))
	for id := range types {
		cles = append(cles, id)
	}
	sort.Slice(cles, func(i, j int) bool { return cles[i] < cles[j] })
	t.Logf("== %s : %d type_id valides par l'oracle ==", filepath.Base(path), len(cles))

	totalObjets, totalVus := 0, 0
	var tous []int
	for _, id := range cles {
		n, nv, idxs := soclesLogFamille(t, v, id, vus)
		totalObjets += n
		totalVus += nv
		tous = append(tous, idxs...)
	}
	t.Logf("BILAN : %d objets de socle dans le fichier, %d vus par le film, %d jamais vus",
		totalObjets, totalVus, totalObjets-totalVus)
	t.Logf("EMPLACEMENTS DISTINCTS (regroupement a moins de %.0f m) : %d",
		soclesSeuilM, soclesEmplacements(v, tous))
}

// soclesEmplacements compte les emplacements distincts : deux objets a moins d'un metre
// l'un de l'autre sont le MEME emplacement declare deux fois, pas deux socles.
//
// SANS CE COMPTE, LE BILAN MENT. Sur Catalyst, deux des treize objets sont a 4,7 cm et
// 9 mm d'un socle deja vu : les annoncer comme « deux socles que le film ne voit pas »
// ferait croire a deux emplacements de plus, alors que la carte n'en a aucun de plus.
func soclesEmplacements(v *Variant, idxs []int) int {
	pris := make([]bool, len(idxs))
	groupes := 0
	for i := range idxs {
		if pris[i] {
			continue
		}
		groupes++
		pris[i] = true
		for j := i + 1; j < len(idxs); j++ {
			if pris[j] {
				continue
			}
			a, b := v.Objects[idxs[i]].Pos, v.Objects[idxs[j]].Pos
			dx, dy, dz := a.X-b.X, a.Y-b.Y, a.Z-b.Z
			if math.Sqrt(dx*dx+dy*dy+dz*dz) < soclesSeuilM {
				pris[j] = true
			}
		}
	}
	return groupes
}

// soclesLogFamille detaille une famille de socle : chaque objet, ses champs, et s'il a
// ete vu par l'oracle.
func soclesLogFamille(t *testing.T, v *Variant, typeID int32, vus map[int]string) (int, int, []int) {
	t.Helper()
	n, nv := 0, 0
	var idxs []int
	t.Logf("-- type_id %d --", typeID)
	for i, o := range v.Objects {
		if o.TypeID != typeID {
			continue
		}
		n++
		idxs = append(idxs, i)
		etat := "JAMAIS VU par le film"
		if arme, ok := vus[i]; ok {
			nv++
			etat = "vu, arme " + arme
		}
		t.Logf("  objet %3d (%8.3f %8.3f %8.3f) cat %2d equipe %2d drapeaux %3d inst %6d labels %v forme %s : %s",
			i, o.Pos.X, o.Pos.Y, o.Pos.Z, o.Category, o.TeamIndex, o.Flags, o.InstanceID,
			soclesLabelsLisibles(o.Labels), soclesFormeCourte(o), etat)
	}
	t.Logf("   -> %d objets de ce type, %d vus, %d jamais vus", n, nv, n-nv)
	return n, nv, idxs
}

// soclesLabelsLisibles rend les labels d'un objet, resolus quand la table les connait.
func soclesLabelsLisibles(hs []int32) []string {
	out := make([]string, 0, len(hs))
	for _, h := range hs {
		if n := LabelName(h); n != "" {
			out = append(out, n)
			continue
		}
		out = append(out, fmt.Sprintf("?%d", h))
	}
	return out
}

// soclesFormeCourte resume la forme d'un objet en une chaine.
func soclesFormeCourte(o Object) string {
	s := o.Shape()
	if s == nil {
		return "-"
	}
	if s.Radius != nil {
		return fmt.Sprintf("cyl r=%.2f h=[%.2f %.2f]", *s.Radius, s.UpZ, s.DownZ)
	}
	if s.HalfX != nil && s.HalfY != nil {
		return fmt.Sprintf("boite %.2fx%.2f h=[%.2f %.2f]", *s.HalfX, *s.HalfY, s.UpZ, s.DownZ)
	}
	return "?"
}

// soclesIdxEnv designe les objets a deroider integralement.
const soclesIdxEnv = "MVAR_OBJ_INDEX"

// TestSoclesObjetBrut deroule l'ARBRE BOND COMPLET des objets designes — tous les champs,
// y compris ceux que `parseObject` ignore.
//
// POURQUOI IL EXISTE. La phase 2 a trouve, sur Catalyst, deux socles quasi superposes a
// deux socles vus (4,7 cm et 9 mm d'ecart). Les champs decodes ne les distinguent pas :
// meme type_id, meme categorie, aucun label, aucune forme. Si un discriminant existe —
// quelle arme, quel mode — il est dans un champ que la grammaire ne lit pas encore.
func TestSoclesObjetBrut(t *testing.T) {
	root, nom := soclesRacine(t)
	idx := strings.TrimSpace(os.Getenv(soclesIdxEnv))
	if idx == "" {
		t.Skipf("%s absent — deroulement d'objet ignore", soclesIdxEnv)
	}
	objs, ok := root.Field(3)
	if !ok {
		t.Fatalf("%s : root[3] absent", nom)
	}
	for _, brut := range strings.Split(idx, ",") {
		i, err := strconv.Atoi(strings.TrimSpace(brut))
		if err != nil {
			t.Fatalf("%s: %q illisible: %v", soclesIdxEnv, brut, err)
		}
		if i < 0 || i >= len(objs.Items) {
			t.Fatalf("%s: index %d hors des %d objets", soclesIdxEnv, i, len(objs.Items))
		}
		t.Logf("===== objet %d =====", i)
		soclesDump(t, fmt.Sprintf("obj[%d]", i), objs.Items[i], 0)
	}
}
