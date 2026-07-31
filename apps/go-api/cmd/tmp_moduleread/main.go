// tmp_moduleread — PoC : lire le CONTAINER d'un .module Halo Infinite (mohd v53)
// SANS Oodle, pour énumérer la table des tags et prouver qu'on lit le format
// nous-mêmes. Cible : trouver les entrées sbsp (scenario_structure_bsp) = la
// géométrie de la map.
//
// Layout de référence (Kaitai matty45 + InfiniteModuleReader Krevil) :
//
//	header mohd : magic@0 ver@4 moduleId@8 fileCount@0x10 ... fileNameSize@0x24
//	              numResources@0x28 numBlocks@0x2C buildVersion@0x30 ...
//	file entry  : 96 octets, group_tag (u32, fourCC LE) @ +0x14
//	block entry : 20 octets (compOff, compSize, decompOff, decompSize, bCompressed)
//
// On NE FAIT PAS CONFIANCE aux offsets sur parole : on VERROUILLE (headerSize,
// stride, groupOff) en maximisant la fraction de fourCC ASCII imprimables sur les
// fileCount entrées, puis on recense les groupes.
//
// Usage : go run ./cmd/tmp_moduleread "<chemin.module>"
package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// filepathWalk collecte récursivement les fichiers *.module sous root.
func filepathWalk(root string, out *[]string) error {
	return filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(p), ".module") {
			*out = append(*out, p)
		}
		return nil
	})
}

// dirLabel = nom du dossier parent (le nom de map), ou le basename sinon.
func dirLabel(p string) string {
	d := filepath.Base(filepath.Dir(p))
	if d == "" || d == "." {
		return filepath.Base(p)
	}
	return d
}

const defMod = `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/any/levels/multi/catalyst/catalyst-rtx-new.module`

func u32(b []byte, o int) uint32 { return binary.LittleEndian.Uint32(b[o:]) }
func u64(b []byte, o int) uint64 { return binary.LittleEndian.Uint64(b[o:]) }

// fourCC formate un u32 group_tag en chaîne lisible (octets fichier inversés :
// le tag 'mat ' est stocké LE -> octets ' ','t','a','m').
func fourCC(v uint32) string {
	b := []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
	return string(b)
}

func printableCC(v uint32) bool {
	for _, c := range []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)} {
		if c < 0x20 || c > 0x7e {
			return false
		}
	}
	return true
}

// scanGroups lit un module et renvoie (fileCount, map group->count) ou ok=false.
func scanGroups(path string) (int, map[string]int, bool) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 0x40 || string(data[:4]) != "mohd" {
		return 0, nil, false
	}
	fileCount := int(u32(data, 0x10))
	bestFrac, bH, bStride, bGoff := -1.0, 0, 0, 0
	for _, hs := range []int{0x40, 0x48, 0x50} {
		for _, stride := range []int{0x58, 0x60, 0x68, 0x70, 0x78, 0x80} {
			if hs+fileCount*stride > len(data) || fileCount <= 0 {
				continue
			}
			for _, goff := range []int{0x14, 0x18, 0x1C, 0x20} {
				ok := 0
				for i := 0; i < fileCount; i++ {
					if printableCC(u32(data, hs+i*stride+goff)) {
						ok++
					}
				}
				if frac := float64(ok) / float64(fileCount); frac > bestFrac {
					bestFrac, bH, bStride, bGoff = frac, hs, stride, goff
				}
			}
		}
	}
	if bestFrac < 0.5 {
		return fileCount, nil, false
	}
	counts := map[string]int{}
	for i := 0; i < fileCount; i++ {
		if g := u32(data, bH+i*bStride+bGoff); printableCC(g) {
			counts[fourCC(g)]++
		}
	}
	return fileCount, counts, true
}

func main() {
	path := defMod
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	// Mode SCAN répertoire : une ligne récap par module.
	if fi, err := os.Stat(path); err == nil && fi.IsDir() {
		var mods []string
		_ = filepathWalk(path, &mods)
		sort.Strings(mods)
		fmt.Printf("%-26s %-6s %-5s %-5s %-5s %-5s %-5s\n", "module", "files", "sbsp", "mode", "coll", "rtgo", "ok")
		for _, m := range mods {
			fc, c, ok := scanGroups(m)
			label := dirLabel(m)
			if !ok {
				fmt.Printf("%-26s %-6d %-5s %-5s %-5s %-5s %-5s\n", label, fc, "-", "-", "-", "-", "NO")
				continue
			}
			fmt.Printf("%-26s %-6d %-5d %-5d %-5d %-5d %-5s\n",
				label, fc, c["sbsp"], c["mode"], c["coll"], c["rtgo"], "yes")
		}
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("lecture impossible: %v\n", err)
		return
	}
	if string(data[:4]) != "mohd" {
		fmt.Printf("magic inattendu: %q\n", data[:4])
		return
	}
	ver := u32(data, 4)
	fileCount := int(u32(data, 0x10))
	fileNameSize := u32(data, 0x24)
	numResources := u32(data, 0x28)
	numBlocks := u32(data, 0x2C)
	buildVer := u64(data, 0x30)
	fmt.Printf("=== %s ===\n", path)
	fmt.Printf("version=%d moduleId=%#x build=%#x\n", ver, u64(data, 8), buildVer)
	fmt.Printf("fileCount=%d fileNameSize=%d numResources=%d numBlocks=%d taille=%d\n",
		fileCount, fileNameSize, numResources, numBlocks, len(data))

	// VERROUILLAGE empirique du layout : (headerSize, stride, groupOff) maximisant
	// la fraction de fourCC imprimables sur les fileCount entrées.
	bestFrac := -1.0
	var bH, bStride, bGoff int
	for _, hs := range []int{0x40, 0x48, 0x50, 0x58, 0x60} {
		for _, stride := range []int{0x58, 0x5C, 0x60, 0x64, 0x68, 0x70, 0x78, 0x80, 0x88} {
			if hs+fileCount*stride > len(data) {
				continue
			}
			for _, goff := range []int{0x10, 0x14, 0x18, 0x1C, 0x20} {
				ok := 0
				for i := 0; i < fileCount; i++ {
					if printableCC(u32(data, hs+i*stride+goff)) {
						ok++
					}
				}
				frac := float64(ok) / float64(fileCount)
				if frac > bestFrac {
					bestFrac, bH, bStride, bGoff = frac, hs, stride, goff
				}
			}
		}
	}
	fmt.Printf("\nlayout verrouillé : headerSize=%#x stride=%#x groupOff=%#x → fourCC imprimables=%.1f%%\n",
		bH, bStride, bGoff, bestFrac*100)
	if bestFrac < 0.5 {
		fmt.Println("ÉCHEC : aucun layout ne donne des fourCC cohérents (format inattendu).")
		return
	}

	// Recensement des groupes de tags.
	counts := map[string]int{}
	for i := 0; i < fileCount; i++ {
		g := u32(data, bH+i*bStride+bGoff)
		if printableCC(g) {
			counts[fourCC(g)]++
		} else {
			counts["<non-tag/resource>"]++
		}
	}
	type kv struct {
		g string
		n int
	}
	var sorted []kv
	for g, n := range counts {
		sorted = append(sorted, kv{g, n})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].n > sorted[j].n })
	fmt.Printf("\n=== groupes de tags (%d distincts) ===\n", len(sorted))
	for _, e := range sorted {
		mark := ""
		switch e.g {
		case "sbsp":
			mark = "  <<< scenario_structure_bsp (GÉOMÉTRIE)"
		case "scnr":
			mark = "  <<< scenario"
		case "mode":
			mark = "  <<< render_model (mesh)"
		case "sddt", "rtgo":
			mark = "  <<< structure design / instances"
		}
		fmt.Printf("  %-22s x%-5d%s\n", e.g, e.n, mark)
	}

	// Détail des entrées BSP : group_tag + champs bruts pour valider qu'on peut
	// localiser leurs blocs (étape suivante = décompression Oodle).
	fmt.Println("\n=== entrées sbsp (détail brut) ===")
	shown := 0
	for i := 0; i < fileCount && shown < 8; i++ {
		base := bH + i*bStride
		if fourCC(u32(data, base+bGoff)) != "sbsp" {
			continue
		}
		shown++
		// Champs selon le layout de réf (relatifs au début d'entrée).
		blockCount := binary.LittleEndian.Uint16(data[base+0x0A:])
		firstBlock := u32(data, base+0x0C)
		dataOff := u32(data, base+0x18)
		fmt.Printf("  entrée#%d : blockCount=%d firstBlock=%d localDataOffset=%#x\n",
			i, blockCount, firstBlock, dataOff)
	}
	if shown == 0 {
		fmt.Println("  (aucune entrée sbsp dans CE module — la géométrie est peut-être")
		fmt.Println("   dans un module compagnon, ex. *-rtx-new vs un module de structure)")
	}
}
