// tmp_kfwalk — WALK du KEYFRAME (type-2) par calibration : chaque record NEW =
// R(6) typeIndex + default-state(largeur calibrée) + mask + comps. Reconstruit un
// World complet (slot->typeIndex) 100% OFFLINE (le keyframe a un NEW par entité,
// y compris les dynamiques absentes des dumps CE). Rapporte combien d'entités liées.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_kfwalk [filmDir]
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

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const idLowBits = 11
const maxD = 600

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

func extractType2(d []byte) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == 2 {
			return d[off+16 : off+16+sz]
		}
		off += 16 + sz
	}
	return nil
}

func rdHeader(pay []byte, pos int) (typ int, slot uint32, body int) {
	if pos+3 > len(pay)*8 {
		return -1, 0, pos
	}
	br := filmdec.NewBitReader(pay)
	br.Skip(pos)
	if br.ReadBit() {
		typ = 3
	} else {
		typ = int(br.ReadBits(2))
	}
	if typ == 0 {
		return 0, 0, br.BitPos()
	}
	low := uint32(br.ReadBits(idLowBits))
	br.ReadBits(2)
	slot = low & 0x3fffffff
	return typ, slot, br.BitPos()
}

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	filmdec.SetRecordStateParam(2)
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	pay := extractType2(inflate(dir + "/chunk_02.bin"))
	fmt.Printf("keyframe type-2 : %d octets (%d bits)\n", len(pay), len(pay)*8)

	startSweep := []int{0, 8, 16, 24, 32, 48, 64, 96, 128}
	bestStart, bestWalked := 0, -1
	for _, st := range startSweep {
		ww := filmdec.NewWorld(reg)
		_, n := walk(pay, st, reg, ww, nil)
		if n > bestWalked {
			bestWalked, bestStart = n, st
		}
	}
	fmt.Printf("meilleur départ=%d → %d records walkés\n", bestStart, bestWalked)

	w := filmdec.NewWorld(reg)
	tiHist := map[uint32]int{}
	pos, walked := walk(pay, bestStart, reg, w, tiHist)
	fmt.Printf("walk depuis bit %d : %d records, fin@bit%d (sur %d)\n", bestStart, walked, pos, len(pay)*8)
	fmt.Printf("World construit : %d entités liées\n", w.Bound())
	type kv struct {
		ti uint32
		n  int
	}
	var arr []kv
	for ti, n := range tiHist {
		arr = append(arr, kv{ti, n})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].n > arr[j].n })
	fmt.Println("typeIndex liés (top) :")
	for i := 0; i < 20 && i < len(arr); i++ {
		fmt.Printf("  typeIdx=%-3d : %d\n", arr[i].ti, arr[i].n)
	}
}

func walk(pay []byte, start int, reg *filmdec.Registry, w *filmdec.World, tiHist map[uint32]int) (int, int) {
	pos, walked := start, 0
	for walked < 4000 {
		typ, slot, body := rdHeader(pay, pos)
		if typ <= 0 {
			break
		}
		switch typ {
		case 1: // NEW
			_, end, ti, ok := calib(pay, body, reg, w)
			if !ok {
				return pos, walked
			}
			w.BindFull(slot, ti)
			if tiHist != nil {
				tiHist[ti]++
			}
			pos = end
		case 3: // DELTA
			t, end := filmdec.DecodeDeltaRecordAt(pay, body, w, slot)
			if t.DesyncAt != -1 {
				return pos, walked
			}
			pos = end
		default: // DEL
			pos = body + 32
		}
		walked++
	}
	return pos, walked
}

func calib(pay []byte, body int, reg *filmdec.Registry, w *filmdec.World) (int, int, uint32, bool) {
	for d := 0; d <= maxD; d++ {
		b := filmdec.NewBitReader(pay)
		b.Skip(body)
		t := filmdec.TraverseEntity(b, reg, d)
		if t.DesyncAt != -1 {
			continue
		}
		end := b.BitPos()
		if end <= body || end > len(pay)*8 {
			continue
		}
		nt, ns, nb := rdHeader(pay, end)
		switch nt {
		case -1, 0:
			return d, end, t.TypeIndex, true
		case 1:
			for d2 := 0; d2 <= maxD; d2++ {
				bb := filmdec.NewBitReader(pay)
				bb.Skip(nb)
				if filmdec.TraverseEntity(bb, reg, d2).DesyncAt == -1 {
					return d, end, t.TypeIndex, true
				}
			}
		case 3:
			if _, ok := w.ArchetypeForSlot(ns); ok {
				tt, _ := filmdec.DecodeDeltaRecordAt(pay, nb, w, ns)
				if tt.DesyncAt == -1 {
					return d, end, t.TypeIndex, true
				}
			}
		}
	}
	return 0, 0, 0, false
}
