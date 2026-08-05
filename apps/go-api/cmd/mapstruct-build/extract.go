// cmd/mapstruct-build — extract.go : sélection du BSP, exclusions, statistiques, écriture.
package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/himap"
)

// coverCell est le pas de rastérisation de la couverture, en mètres (même pas que le
// contrôle anti-tautologie du témoin de surface).
const coverCell = 0.25

// areaCap est le plafond d'aire du contrôle anti-tautologie : une seule dalle géante
// suffirait à « couvrir » une carte, on remesure donc la couverture sans les grandes
// boîtes englobantes.
const areaCap = 200.0

// Motifs d'exclusion (clés de MapStructure.Excluded, publiées telles quelles).
const (
	dropQuickDeleted = "quick deleted (flags2 bit 2)"
	dropBadAABB      = "AABB non finie ou dégénérée"
	dropOutOfBounds  = "AABB hors des bornes monde du BSP"
)

// extractStructure lit les instances du module et produit le document figé.
func extractStructure(modPath, module string, names []string) (*replay.MapStructure, error) {
	bsps, err := himap.ReadModuleInstances(modPath)
	if err != nil {
		return nil, err
	}
	main := pickBSP(bsps)
	if main == nil {
		return nil, os.ErrNotExist
	}
	ms := &replay.MapStructure{
		SchemaVersion: replay.MapStructureSchemaVersion,
		Module:        module,
		MapNames:      names,
		Source: "bloc `instanced geometry instances` du tag sbsp, AABB monde @0x7C lue telle " +
			"quelle (aucune transformation appliquée), " + filepath.Base(modPath),
		GeneratedAt: time.Now().UTC(),
		BSPIndex:    main.FileIndex,
		Excluded:    map[string]int{},
	}
	for a := 0; a < 3; a++ {
		ms.WorldMin[a] = float32(main.Bounds.Min[a])
		ms.WorldMax[a] = float32(main.Bounds.Max[a])
	}
	for _, in := range main.Instances {
		switch {
		case in.QuickDeleted():
			ms.Excluded[dropQuickDeleted]++
		case !finiteAABB(in):
			ms.Excluded[dropBadAABB]++
		case !intersectsBounds(in, main.Bounds):
			ms.Excluded[dropOutOfBounds]++
		default:
			fp := in.Footprint()
			ms.Surfaces = append(ms.Surfaces, replay.Surface{
				X0: r2(fp.MinX), Y0: r2(fp.MinY), X1: r2(fp.MaxX), Y1: r2(fp.MaxY),
				Z: r2(fp.TopZ), ZB: r2(fp.BottomZ),
			})
		}
	}
	ms.Stats = statsOf(ms.Surfaces, main.Bounds)
	return ms, nil
}

// pickBSP retient le BSP DE JEU : parmi les tags sbsp porteurs d'instances, celui dont les
// bornes monde ont le plus petit volume. Un critère de taille brute retiendrait la skybox
// (bornes ±3300 m, ses instances sont du décor lointain).
func pickBSP(bsps []himap.BSPInstances) *himap.BSPInstances {
	var best *himap.BSPInstances
	bestVol := math.Inf(1)
	for i := range bsps {
		b := &bsps[i]
		if len(b.Instances) == 0 || !b.Bounds.Valid() {
			continue
		}
		if v := b.Bounds.Extent(0) * b.Bounds.Extent(1) * b.Bounds.Extent(2); v < bestVol {
			bestVol, best = v, b
		}
	}
	return best
}

func finiteAABB(in himap.Instance) bool {
	for a := 0; a < 3; a++ {
		lo, hi := in.AABBMin[a], in.AABBMax[a]
		if math.IsNaN(lo) || math.IsNaN(hi) || math.IsInf(lo, 0) || math.IsInf(hi, 0) || hi < lo {
			return false
		}
	}
	return true
}

// intersectsBounds : une instance entièrement hors des bornes monde du BSP n'est jamais
// rendue dans la zone de jeu.
func intersectsBounds(in himap.Instance, b himap.Bounds) bool {
	for a := 0; a < 3; a++ {
		if in.AABBMax[a] < b.Min[a] || in.AABBMin[a] > b.Max[a] {
			return false
		}
	}
	return true
}

// statsOf résume les emprises : distribution des aires + couverture des bornes monde.
func statsOf(surf []replay.Surface, b himap.Bounds) replay.MapStructureStats {
	st := replay.MapStructureStats{Count: len(surf), CoverCell: coverCell}
	if len(surf) == 0 {
		return st
	}
	areas := make([]float32, 0, len(surf))
	for _, s := range surf {
		areas = append(areas, s.Area())
	}
	sort.Slice(areas, func(i, j int) bool { return areas[i] < areas[j] })
	st.AreaMedian = areas[len(areas)/2]
	st.AreaMax = areas[len(areas)-1]
	st.CoveragePct = coverage(surf, b, math.Inf(1))
	st.AreaCap = areaCap
	st.CoveragePctCapped = coverage(surf, b, areaCap)
	return st
}

// coverage rastérise les emprises d'aire <= cap sur les bornes monde du BSP et renvoie le
// pourcentage de cellules couvertes. C'est LE chiffre qui dit si une carte est exploitable :
// ridgeline est à 100 % (et 98 % en ne gardant que les emprises <= 200 m²), catalyst,
// forest et ctf_aquarius plafonnent à 40-49 % — leur structure vit dans le maillage de
// rendu NON instancié, que ce binaire ne lit pas.
func coverage(surf []replay.Surface, b himap.Bounds, cap float64) float32 {
	lo := [2]float64{b.Min[0], b.Min[1]}
	hi := [2]float64{b.Max[0], b.Max[1]}
	nx := int((hi[0]-lo[0])/coverCell) + 1
	ny := int((hi[1]-lo[1])/coverCell) + 1
	if nx <= 0 || ny <= 0 || nx*ny > 40_000_000 {
		return 0
	}
	occ := make([]bool, nx*ny)
	for _, s := range surf {
		if float64(s.Area()) > cap {
			continue
		}
		if float64(s.X1) < lo[0] || float64(s.X0) > hi[0] ||
			float64(s.Y1) < lo[1] || float64(s.Y0) > hi[1] {
			continue
		}
		x0 := clamp(int((float64(s.X0)-lo[0])/coverCell), 0, nx-1)
		x1 := clamp(int((float64(s.X1)-lo[0])/coverCell), 0, nx-1)
		y0 := clamp(int((float64(s.Y0)-lo[1])/coverCell), 0, ny-1)
		y1 := clamp(int((float64(s.Y1)-lo[1])/coverCell), 0, ny-1)
		for y := y0; y <= y1; y++ {
			row := y * nx
			for x := x0; x <= x1; x++ {
				occ[row+x] = true
			}
		}
	}
	n := 0
	for _, o := range occ {
		if o {
			n++
		}
	}
	return float32(100*float64(n)) / float32(nx*ny)
}

// writeStructure sérialise le document (compact : ~10 000 emprises par carte, une écriture
// indentée quadruplerait le poids versionné) de façon atomique.
func writeStructure(path string, ms *replay.MapStructure) (int, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, err
	}
	blob, err := json.Marshal(ms)
	if err != nil {
		return 0, err
	}
	blob = append(blob, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, blob, 0o644); err != nil {
		return 0, err
	}
	return len(blob), os.Rename(tmp, path)
}

// r2 arrondit au centimètre : le quantum du décodeur de positions est de ~1,4 cm, deux
// décimales ne perdent donc rien et allègent nettement le fichier.
func r2(v float64) float32 {
	return float32(math.Round(v*100) / 100)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func sorted(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func sortedTargets(in []target) []target {
	sort.Slice(in, func(i, j int) bool { return in[i].module < in[j].module })
	return in
}
