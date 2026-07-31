// tmp_decor — extraction de la STRUCTURE d'une carte depuis le bloc
// `instanced geometry instances` du sbsp, puis TÉMOIN DE SURFACE : les joueurs du film
// marchent-ils SUR les faces supérieures des emprises ?
//
// La métrique retenue est |dz| au-dessus de la surface la plus haute SOUS le joueur.
// Elle est comparée à des témoins (géométrie tournée / translatée / axes permutés /
// altitudes permutées) et à un tirage uniforme de positions. La métrique
// « les points tombent dans une emprise » est volontairement ÉCARTÉE : elle est
// quasi tautologique dès que les emprises couvrent la zone de jeu.
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_decor <module> <trajectoires.csv> [sortie.csv]
package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"strconv"

	"levelup/go-api/internal/himap"
)

const (
	defModule = `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/pc/levels/multi/ridgeline/ridgeline-rtx-new.module`
	defTraj   = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/offline_trajectories_clean.csv`
)

func main() {
	modPath, trajPath, outPath := defModule, defTraj, ""
	if len(os.Args) > 1 {
		modPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		trajPath = os.Args[2]
	}
	if len(os.Args) > 3 {
		outPath = os.Args[3]
	}
	bsp, kept := loadStructure(modPath)
	if bsp == nil {
		return
	}
	pts, err := loadTraj(trajPath)
	if err != nil {
		fmt.Println("trajectoires:", err)
		return
	}
	grounded := groundedOnly(trajPath)
	fmt.Printf("\n%d positions joueur lues (%s)\n", len(pts), trajPath)
	fmt.Printf("dont %d à vitesse verticale quasi nulle (|vz| <= %.2f m/s) — filtre défini\n"+
		"UNIQUEMENT sur les données joueur, donc indépendant de la géométrie testée\n",
		len(grounded), maxVZ)
	runWitnesses(bsp, kept, pts, grounded)
	coverage(bsp, kept, pts)
	if outPath != "" {
		if err := writeFootprints(outPath, bsp, kept); err != nil {
			fmt.Println("écriture:", err)
			return
		}
		fmt.Printf("\nemprises écrites : %s (%d lignes)\n", outPath, len(kept))
	}
}

// loadStructure lit les instances du BSP principal et applique les exclusions.
func loadStructure(modPath string) (*himap.BSPInstances, []himap.Instance) {
	bsps, err := himap.ReadModuleInstances(modPath)
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}
	var main *himap.BSPInstances
	best := math.Inf(1)
	for i := range bsps {
		b := &bsps[i]
		if len(b.Instances) == 0 || !b.Bounds.Valid() {
			continue
		}
		if v := b.Bounds.Extent(0) * b.Bounds.Extent(1) * b.Bounds.Extent(2); v < best {
			best, main = v, b
		}
	}
	if main == nil {
		fmt.Println("aucun BSP porteur d'instances")
		return nil, nil
	}
	fmt.Printf("BSP principal sbsp#%d : %d instances brutes ; bornes x[%.1f,%.1f] y[%.1f,%.1f] z[%.1f,%.1f]\n",
		main.FileIndex, len(main.Instances),
		main.Bounds.Min[0], main.Bounds.Max[0], main.Bounds.Min[1], main.Bounds.Max[1],
		main.Bounds.Min[2], main.Bounds.Max[2])

	drop := map[string]int{}
	var kept []himap.Instance
	for _, in := range main.Instances {
		switch {
		case in.QuickDeleted():
			drop["quick deleted (flags2 bit 2)"]++
		case !finiteAABB(in):
			drop["AABB non finie ou dégénérée"]++
		case !insideBounds(in, main.Bounds):
			drop["AABB hors des bornes monde du BSP"]++
		default:
			kept = append(kept, in)
		}
	}
	fmt.Printf("exclusions :\n")
	for k, v := range drop {
		fmt.Printf("  %-36s %d\n", k, v)
	}
	fmt.Printf("  => %d instances conservées\n", len(kept))
	return main, kept
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

// insideBounds : l'AABB doit INTERSECTER les bornes monde du BSP (une instance
// entièrement hors des bornes n'est jamais rendue dans la zone de jeu).
func insideBounds(in himap.Instance, b himap.Bounds) bool {
	for a := 0; a < 3; a++ {
		if in.AABBMax[a] < b.Min[a] || in.AABBMin[a] > b.Max[a] {
			return false
		}
	}
	return true
}

func topSurfaces(ins []himap.Instance) []surface {
	out := make([]surface, 0, len(ins))
	for _, in := range ins {
		out = append(out, surface{
			minX: in.AABBMin[0], minY: in.AABBMin[1],
			maxX: in.AABBMax[0], maxY: in.AABBMax[1], z: in.AABBMax[2],
		})
	}
	return out
}

func loadTraj(path string) ([][3]float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(bufio.NewReaderSize(f, 1<<20))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	out := make([][3]float64, 0, len(rows))
	for i, rec := range rows {
		if i == 0 || len(rec) < 7 {
			continue
		}
		var p [3]float64
		bad := false
		for a := 0; a < 3; a++ {
			v, err := strconv.ParseFloat(rec[4+a], 64)
			if err != nil {
				bad = true
				break
			}
			p[a] = v
		}
		if !bad {
			out = append(out, p)
		}
	}
	return out, nil
}

func writeFootprints(path string, bsp *himap.BSPInstances, ins []himap.Instance) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	head := []string{"index", "min_x", "min_y", "max_x", "max_y", "top_z", "bottom_z",
		"pos_x", "pos_y", "pos_z", "mesh_index", "flags", "flags2", "area_m2"}
	if err := w.Write(head); err != nil {
		return err
	}
	for _, in := range ins {
		fp := in.Footprint()
		rec := []string{
			strconv.Itoa(in.Index),
			ff(fp.MinX), ff(fp.MinY), ff(fp.MaxX), ff(fp.MaxY), ff(fp.TopZ), ff(fp.BottomZ),
			ff(in.Position[0]), ff(in.Position[1]), ff(in.Position[2]),
			strconv.Itoa(in.MeshIndex),
			strconv.FormatUint(uint64(in.Flags), 10), strconv.FormatUint(uint64(in.Flags2), 10),
			ff(fp.Area()),
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

func ff(v float64) string { return strconv.FormatFloat(v, 'f', 4, 64) }
