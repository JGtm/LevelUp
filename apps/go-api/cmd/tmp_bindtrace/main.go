// tmp_bindtrace — THROWAWAY : trace l'historique de binding du World pour comprendre
// pourquoi les slots des death-frames (ex 1038) ne sont pas liés. Réponses :
//  1. couverture : combien de slots liés (world_dump initial + recNew atteints) vs
//     combien de slots distincts apparaissent en record.
//  2. pour un slot cible : world_dump ? ses recNew/recDel (chunk, frame, atteint ou
//     derrière un désync) ? lié à la fin ?
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_bindtrace [slot]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}

func inflate(p string) []byte {
	raw, _ := os.ReadFile(p)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

type packet struct{ payload []byte }

func listFrames(d []byte) []packet {
	var out []packet
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, packet{d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
}

func main() {
	target := uint32(1038)
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &target)
	}
	reg, _ := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))

	// world_dump initial
	wd, _ := os.ReadFile(cache + "/world_dump.txt")
	inDump := false
	dumpCount := 0
	for _, tok := range bytes.Fields(wd) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			dumpCount++
			if slot == target {
				inDump = true
				fmt.Printf("world_dump : slot %d présent, lié à ti=%d\n", target, ti)
			}
		}
	}
	if !inDump {
		fmt.Printf("world_dump : slot %d ABSENT (%d slots dans le dump)\n", target, dumpCount)
	}

	w := filmdec.NewWorld(reg)
	for _, tok := range bytes.Fields(wd) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			w.BindFull(slot, ti)
		}
	}

	// replay : pour chaque frame, décode (World persistant). Note les records du slot cible
	// et s'ils sont ATTEINTS (avant le désync de la frame) ou derrière le désync.
	allSlots := map[uint32]bool{}   // slots vus en record
	reachedNew := map[uint32]bool{} // slots avec un recNew ATTEINT (donc liés)
	fmt.Printf("\n--- records pour slot %d (atteints vs derrière désync) ---\n", target)
	nFrames, nClean := 0, 0
	for idx := 1; idx <= 26; idx++ {
		for _, pk := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			nFrames++
			br := filmdec.NewBitReader(pk.payload)
			recs, derr := filmdec.DecodeFrameRecords(br, w, calCfg)
			if derr == nil {
				nClean++
			}
			for _, r := range recs {
				allSlots[r.Slot] = true
				if r.Type == 1 && r.DesyncAt == -1 { // recNew CLEAN -> bind
					reachedNew[r.Slot] = true
				}
				if r.Slot == target {
					tn := map[int]string{0: "end", 1: "NEW", 2: "DEL", 3: "delta"}[r.Type]
					fmt.Printf("  chunk%02d : type=%s ti=%d desyncAt=%d (frameErr=%v)\n", idx, tn, r.TypeIndex, r.DesyncAt, derr != nil)
				}
			}
		}
	}

	// couverture finale
	bound := 0
	var sample []uint32
	for s := range allSlots {
		if _, ok := w.ArchetypeForSlot(s); ok {
			bound++
		} else {
			sample = append(sample, s)
		}
	}
	sort.Slice(sample, func(i, j int) bool { return sample[i] < sample[j] })
	_, tgtBound := w.ArchetypeForSlot(target)
	fmt.Printf("\n--- couverture (après replay %d frames, %d propres) ---\n", nFrames, nClean)
	fmt.Printf("slots distincts vus en record : %d\n", len(allSlots))
	fmt.Printf("slots LIÉS à la fin : %d ; NON liés : %d\n", bound, len(sample))
	fmt.Printf("slot cible %d : lié=%v ; recNew atteint=%v\n", target, tgtBound, reachedNew[target])
	fmt.Printf("échantillon slots NON liés (référencés mais jamais bindés) : ")
	for i, s := range sample {
		if i >= 25 {
			fmt.Printf("... (+%d)", len(sample)-25)
			break
		}
		fmt.Printf("%d ", s)
	}
	fmt.Println()
}
