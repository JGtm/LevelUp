// tmp_reflink — cherche où l'identifiant de tag porté par `meshRef` (@+8 dans le champ
// de référence) apparaît dans la table des entrées du .module. Objectif : résoudre
// instance -> tag runtime_geo -> bornes locales du mesh.
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_reflink [module]
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"

	"levelup/go-api/internal/himap"
)

const defMod = `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/pc/levels/multi/ridgeline/ridgeline-rtx-new.module`

func main() {
	p := defMod
	if len(os.Args) > 1 {
		p = os.Args[1]
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		fmt.Println(err)
		return
	}
	fileCount := int(binary.LittleEndian.Uint32(raw[0x10:]))
	fmt.Printf("fileCount=%d\n", fileCount)

	bsps, err := himap.ReadModuleInstances(p)
	if err != nil {
		fmt.Println(err)
		return
	}
	var mainB *himap.BSPInstances
	best := math.Inf(1)
	for i := range bsps {
		b := &bsps[i]
		if len(b.Instances) == 0 || !b.Bounds.Valid() {
			continue
		}
		if v := b.Bounds.Extent(0) * b.Bounds.Extent(1) * b.Bounds.Extent(2); v < best {
			best, mainB = v, b
		}
	}
	// identifiants candidats portés par le champ de référence
	uniq := map[uint32]int{}
	for _, in := range mainB.Instances {
		uniq[binary.LittleEndian.Uint32(in.MeshRef[8:])]++
	}
	fmt.Printf("%d identifiants distincts @meshRef+8\n", len(uniq))

	// dans chaque entrée fichier (0x58 o), à quel offset retrouve-t-on ces identifiants ?
	hits := map[int]int{}
	for i := 0; i < fileCount; i++ {
		e := 0x48 + i*0x58
		for o := 0; o+4 <= 0x58; o += 4 {
			v := binary.LittleEndian.Uint32(raw[e+o:])
			if uniq[v] > 0 {
				hits[o]++
			}
		}
	}
	fmt.Println("offsets d'entrée fichier où un identifiant meshRef apparaît :")
	for o := 0; o+4 <= 0x58; o += 4 {
		if hits[o] > 0 {
			fmt.Printf("  +%#02x : %d entrées\n", o, hits[o])
		}
	}
	// dump d'une entrée sbsp et d'une entrée quelconque pour référence
	for i := 0; i < fileCount && i < 3; i++ {
		e := 0x48 + i*0x58
		fmt.Printf("  entrée %d : % x\n", i, raw[e:e+0x58])
	}
}
