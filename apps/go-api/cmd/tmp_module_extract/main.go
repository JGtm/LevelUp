// tmp_module_extract — PoC PHASE 2 : décompresser un tag sbsp d'un .module ds via
// Oodle, prouver qu'on sort des données décompressées exploitables.
//
// Layout entrée fichier (88o, VÉRIFIÉ par hand-decode sur catalyst/ds, cross-checké
// contre la table des blocs) :
//
//	+0x00 name_offset u32 | +0x04 parent u32 | +0x0A blockCount u16
//	+0x0C firstBlockIndex u32 | +0x14 group_tag u32 (fourCC LE)
//	+0x18 data_offset u32 (relatif à dataBase) | +0x20 compressedSize u32
//	+0x24 uncompressedSize u32 | +0x48..0x57 GUID (asset id + global id)
//
// Bloc (20o) : +0 compOff +4 compSize +8 decompOff +12 decompSize +16 bCompressed.
//
//	compOff est RELATIF au data_offset du fichier.
//
// dataBase = tailleFichier - max(data_offset + compressedSize) sur tous les fichiers.
//
// Oodle = oo2core_*_win64.dll via syscall (pas de CGO). Usage :
//
//	go run ./cmd/tmp_module_extract ["<module>"] ["<oodle.dll>"]
package main

import (
	"encoding/binary"
	"fmt"
	"os"

	"levelup/go-api/internal/ooz"
)

const defMod = `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi/catalyst/catalyst-rtx-new.module`

const (
	headerSize     = 0x48 // entrées commencent à 0x48 (marqueurs name=0/parent=-1 @0xa0,0xf8,...)
	entryStride    = 0x58
	blockEntrySize = 20

	offBlockCount = 0x0A // u16
	offFirstBlock = 0x0C // u32
	offGroup      = 0x14 // u32 fourCC LE
	offDataOffset = 0x18 // u32
	offCompSize   = 0x20 // u32 (compressé total du fichier)
	offUncompSize = 0x24 // u32
)

func u16(b []byte, o int) uint16 { return binary.LittleEndian.Uint16(b[o:]) }
func u32(b []byte, o int) uint32 { return binary.LittleEndian.Uint32(b[o:]) }
func fourCC(v uint32) string {
	return string([]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
}

// lockBlockTable : 1re position où numBlocks entrées de 20o ont leur 5e u32
// (bCompressed) ∈ {0,1} et compSize plausible.
func lockBlockTable(data []byte, numBlocks, from int) int {
	for off := from; off <= len(data)-numBlocks*blockEntrySize; off += 4 {
		ok := true
		for i := 0; i < numBlocks; i++ {
			b := off + i*blockEntrySize
			if u32(data, b+16) > 1 {
				ok = false
				break
			}
			if cs := u32(data, b+4); cs == 0 || cs > 0x4000000 {
				ok = false
				break
			}
		}
		if ok {
			return off
		}
	}
	return -1
}

func main() {
	modPath := defMod
	if len(os.Args) > 1 {
		modPath = os.Args[1]
	}
	data, err := os.ReadFile(modPath)
	if err != nil || string(data[:4]) != "mohd" {
		fmt.Printf("module illisible: %v\n", err)
		return
	}
	fileCount := int(u32(data, 0x10))
	numBlocks := int(u32(data, 0x2C))
	fmt.Printf("=== %s ===\nv%d fileCount=%d numBlocks=%d taille=%d\n",
		modPath, u32(data, 4), fileCount, numBlocks, len(data))

	blockOff := lockBlockTable(data, numBlocks, headerSize+fileCount*entryStride)
	if blockOff < 0 {
		fmt.Println("ÉCHEC : table des blocs introuvable.")
		return
	}
	fmt.Printf("table blocs @ %#x\n", blockOff)

	entry := func(i int) int { return headerSize + i*entryStride }
	block := func(idx int) (co, cs, do, ds, c int) {
		b := blockOff + idx*blockEntrySize
		return int(u32(data, b)), int(u32(data, b+4)), int(u32(data, b+8)), int(u32(data, b+12)), int(u32(data, b+16))
	}

	// dataBase = tailleFichier - max(data_offset + compressedSize).
	maxEnd := 0
	for i := 0; i < fileCount; i++ {
		e := entry(i)
		if end := int(u32(data, e+offDataOffset)) + int(u32(data, e+offCompSize)); end > maxEnd {
			maxEnd = end
		}
	}
	dataBase := len(data) - maxEnd
	fmt.Printf("dataBase = %#x (sectionDonnées=%#x)\n", dataBase, maxEnd)

	// 1re entrée sbsp.
	sbsp := -1
	for i := 0; i < fileCount; i++ {
		if fourCC(u32(data, entry(i)+offGroup)) == "sbsp" {
			sbsp = i
			break
		}
	}
	if sbsp < 0 {
		fmt.Println("aucun sbsp dans ce module.")
		return
	}
	e := entry(sbsp)
	fb := int(u32(data, e+offFirstBlock))
	bc := int(u16(data, e+offBlockCount))
	fileDataOff := int(u32(data, e+offDataOffset))
	compTot := int(u32(data, e+offCompSize))
	uncompTot := int(u32(data, e+offUncompSize))
	fmt.Printf("\nsbsp entrée#%d : firstBlock=%d blockCount=%d data_offset=%#x compTot=%#x uncompTot=%#x\n",
		sbsp, fb, bc, fileDataOff, compTot, uncompTot)

	fmt.Println("  blocs : idx  compOff   compSize  decompOff decompSize comp")
	totalDecomp := 0
	for i := fb; i < fb+bc; i++ {
		co, cs, dorel, ds, c := block(i)
		fmt.Printf("        %4d %9x %9x %9x %9x  %d\n", i, co, cs, dorel, ds, c)
		if dorel+ds > totalDecomp {
			totalDecomp = dorel + ds
		}
	}

	out := make([]byte, totalDecomp)
	for i := fb; i < fb+bc; i++ {
		co, cs, dorel, ds, c := block(i)
		base := dataBase + fileDataOff + co
		src := data[base : base+cs]
		if c == 0 {
			copy(out[dorel:dorel+ds], src) // verbatim
			continue
		}
		dec, err := ooz.Decompress(src, ds)
		if err != nil {
			fmt.Printf("  [ooz] bloc %d ÉCHEC : %v (@abs %#x)\n", i, err, base)
			return
		}
		copy(out[dorel:dorel+ds], dec)
	}
	fmt.Printf("\n*** DÉCOMPRESSION RÉUSSIE : %d octets ***\n", len(out))
	n := 64
	if len(out) < n {
		n = len(out)
	}
	fmt.Printf("premiers octets:\n% x\n", out[:n])
	zeros := 0
	for _, b := range out {
		if b == 0 {
			zeros++
		}
	}
	fmt.Printf("ratio zéros = %.1f%% (structuré = mélange)\n", float64(zeros)/float64(len(out))*100)

	// dump pour analyse offline.
	outPath := os.TempDir() + "/sbsp_catalyst_block0.bin"
	if err := os.WriteFile(outPath, out, 0o644); err == nil {
		fmt.Printf("dump → %s\n", outPath)
	}
}
