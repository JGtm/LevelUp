// tmp_geodump — exporte la MATIERE BRUTE de la geometrie d'une carte pour analyse hors Go.
//
// POURQUOI : la chaine instance -> maillage est cassee sur ridgeline (l'offset `Per Mesh Data`
// du plugin rawg.xml vaut pour catalyst ; ici il rend 1 908 maillages invalides sur 2 369 et un
// X a 1e33 — derive de version, deja rencontree sur sbsp.xml). Le craquage du layout se fait
// mieux par force brute en Python que par lecture de plugin.
//
// CE QUE CE BINAIRE PRODUIT :
//   - instances.csv     : les 10 223 instances, avec leur transformation ET leur AABB monde
//   - manifest.csv      : toutes les entrees du module (groupe, taille, magic)
//   - res/NNNN.bin      : les ressources brutes decompressees (celles qui portent la geometrie)
//
// LE CRITERE DE VALIDATION que ce dump rend possible : l'AABB monde de chaque instance est
// CONNU ET JUSTE (il alimente deja un fond de carte a 100 % de couverture, valide a 8 mm de
// mediane sous les joueurs). Donc pour toute hypothese de bornes LOCALES d'un maillage, on
// exige que `position + lx*forward + ly*left + lz*up` reproduise cet AABB. C'est un probleme a
// texte clair connu, pas une recherche a l'aveugle.
//
// Usage :
//
//	CGO_ENABLED=1 go run ./cmd/tmp_geodump [--groups g1,g2] [--max-res-mb N] [--manifest-only] <module> <outDir>
//
// PIEGE : le paquet flag arrete l'analyse au PREMIER argument positionnel. Les options doivent
// donc preceder <module> -- sinon --groups est ignore en silence et on ecrit 23 000 fichiers.
// (Meme piege deja documente en tete de cmd/replay-build.)
//
// --groups sert aux modules GLOBAUX : any/globals/common porte 23 225 entrees dont 11 `eqip`
// (les equipements Spartan), 238 `uslg` (tables de chaines), 95 `proj`, 81 `weap`.
package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/himap"
	"levelup/go-api/internal/himodule"
)

func main() {
	maxMB := flag.Int("max-res-mb", 600, "plafond cumule des ressources ecrites, en Mo")
	manifestOnly := flag.Bool("manifest-only", false, "n'ecrire que le manifeste")
	groups := flag.String("groups", "", "groupes de tags a ecrire, separes par des virgules (vide = tous)")
	flag.Parse()
	// Un module global comme common/ porte 23 000 entrees : sans filtre le dump est ingerable.
	keepGroup := map[string]bool{}
	for _, g := range strings.Split(*groups, ",") {
		if g = strings.TrimSpace(g); g != "" {
			keepGroup[g] = true
		}
	}
	args := flag.Args()
	if len(args) < 2 {
		fmt.Println("usage: tmp_geodump <module> <outDir> [--max-res-mb N] [--manifest-only]")
		os.Exit(2)
	}
	modPath, outDir := args[0], args[1]
	if err := os.MkdirAll(filepath.Join(outDir, "res"), 0o755); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// --- 1. instances : transformation + AABB monde (la verite terrain du craquage) --------
	// Un module `any/` (gameplay : scenario, collision, placements) n'a PAS de tag sbsp :
	// l'absence d'instances y est normale et ne doit pas empecher le dump des ressources.
	bsps, err := himap.ReadModuleInstances(modPath)
	if err != nil {
		fmt.Println("pas d'instances de geometrie dans ce module (attendu pour un module any/) :", err)
		bsps = nil
	}
	fi, err := os.Create(filepath.Join(outDir, "instances.csv"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	wi := bufio.NewWriter(fi)
	fmt.Fprintln(wi, "bsp,idx,meshIndex,uniqueIO,flags,flags2,meshRef,"+
		"px,py,pz,fx,fy,fz,lx,ly,lz,ux,uy,uz,"+
		"aabbMinX,aabbMinY,aabbMinZ,aabbMaxX,aabbMaxY,aabbMaxZ,"+
		"sphX,sphY,sphZ,radius,scaleX,scaleY,scaleZ")
	total := 0
	for bi, b := range bsps {
		fmt.Printf("bsp %d : %d instances, bornes %v .. %v\n", bi, len(b.Instances), b.Bounds.Min, b.Bounds.Max)
		for _, in := range b.Instances {
			total++
			fmt.Fprintf(wi, "%d,%d,%d,%d,%d,%d,%s,"+
				"%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,"+
				"%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f\n",
				bi, in.Index, in.MeshIndex, in.UniqueIO, in.Flags, in.Flags2,
				hex.EncodeToString(in.MeshRef[:]),
				in.Position[0], in.Position[1], in.Position[2],
				in.Forward[0], in.Forward[1], in.Forward[2],
				in.Left[0], in.Left[1], in.Left[2],
				in.Up[0], in.Up[1], in.Up[2],
				in.AABBMin[0], in.AABBMin[1], in.AABBMin[2],
				in.AABBMax[0], in.AABBMax[1], in.AABBMax[2],
				in.Sphere[0], in.Sphere[1], in.Sphere[2], in.Radius,
				in.Scale[0], in.Scale[1], in.Scale[2])
		}
	}
	wi.Flush()
	fi.Close()
	fmt.Printf("instances.csv : %d instances\n", total)

	// --- 2. manifeste + ressources brutes ---------------------------------------------------
	m, err := himodule.Open(modPath)
	if err != nil {
		fmt.Println("module:", err)
		os.Exit(1)
	}
	all := m.Files("")
	fm, err := os.Create(filepath.Join(outDir, "manifest.csv"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	wm := bufio.NewWriter(fm)
	fmt.Fprintln(wm, "index,group,blockCount,compSize,uncompSize,magic,ecrit")
	budget := *maxMB << 20
	written, nUcsh, nHd1, okHd1 := 0, 0, 0, 0
	for _, fl := range all {
		if fl.UseHd1() {
			nHd1++
		}
		data, err := m.Extract(fl)
		if fl.UseHd1() && err == nil && len(data) > 0 {
			okHd1++
		}
		if err != nil || len(data) < 4 {
			fmt.Fprintf(wm, "%d,%s,%d,%d,%d,,non\n", fl.Index, fl.Group, fl.BlockCount, fl.CompSize, fl.UncompSize)
			continue
		}
		magic := string(data[:4])
		for i, c := range []byte(magic) { // magic non imprimable -> hexa, pour ne pas casser le CSV
			if c < 0x20 || c > 0x7e {
				magic = hex.EncodeToString(data[:4])
				break
			}
			_ = i
		}
		// On ne retient que les entrees qui portent de la geometrie de rendu : magic `ucsh`
		// (tag/resource valide). Le reste (audio, materiaux) alourdirait le dump sans servir.
		keep := magic == "ucsh" && !*manifestOnly && len(data) <= budget
		if len(keepGroup) > 0 && !keepGroup[safeGroup(fl.Group)] {
			keep = false
		}
		if magic == "ucsh" {
			nUcsh++
		}
		if keep {
			p := filepath.Join(outDir, "res", fmt.Sprintf("%05d_%s.bin", fl.Index, safeGroup(fl.Group)))
			if err := os.WriteFile(p, data, 0o644); err == nil {
				budget -= len(data)
				written++
			}
		}
		fmt.Fprintf(wm, "%d,%s,%d,%d,%d,%s,%s\n", fl.Index, safeGroup(fl.Group), fl.BlockCount,
			fl.CompSize, fl.UncompSize, magic, yesNo(keep))
	}
	wm.Flush()
	fm.Close()
	fmt.Printf("manifest.csv : %d entrees, %d en ucsh, %d ecrites (%d Mo restants au budget)\n",
		len(all), nUcsh, written, budget>>20)
	// COMPAGNON `.module_hd1` : les entrees dont le drapeau UseHd1 est arme y pointent.
	// Sur ridgeline le compagnon fait 206 Mo et n'avait jamais ete ouvert -- le lecteur
	// lisait dataOffset comme un u32 alors que c'est un entier 48 bits + 16 bits de drapeaux.
	if nHd1 > 0 {
		fmt.Printf("COMPAGNON _hd1 : %d entrees y pointent, %d extraites (%.1f %%)\n",
			nHd1, okHd1, 100*float64(okHd1)/float64(nHd1))
	} else {
		fmt.Println("COMPAGNON _hd1 : aucune entree ne l'utilise")
	}
}

func safeGroup(g string) string {
	if g == "" {
		return "res"
	}
	out := make([]byte, 0, len(g))
	for i := 0; i < len(g); i++ {
		if g[i] >= 0x20 && g[i] <= 0x7e && g[i] != ',' {
			out = append(out, g[i])
		}
	}
	if len(out) == 0 {
		return "res"
	}
	return string(out)
}

func yesNo(b bool) string {
	if b {
		return "oui"
	}
	return "non"
}

var _ = binary.LittleEndian
