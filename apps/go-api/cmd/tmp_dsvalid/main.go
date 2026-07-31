// tmp_dsvalid — THROWAWAY : pour chaque combo de masque (composants pré-i11
// présents), mesure le taux de dead-state VALIDE (Mort && EnumA,EnumB in 0..7 &&
// EnumA!=EnumB). Avec la calibration killeridx (i0/i2/i3/i5/i6/i9 + rsp=2). Isole
// le composant coupable : un combo qui contient le coupable -> taux valide bas.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_dsvalid [maxChunk]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

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

type packet struct {
	ts      uint64
	payload []byte
}

func listFrames(d []byte) []packet {
	var out []packet
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, packet{ts, d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
}

func freshWorld(reg *filmdec.Registry) *filmdec.World {
	raw, _ := os.ReadFile(cache + "/world_dump.txt")
	w := filmdec.NewWorld(reg)
	for _, tok := range bytes.Fields(raw) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			w.BindFull(slot, ti)
		}
	}
	return w
}

func main() {
	maxChunk := 26
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &maxChunk)
	}
	// arg2 = liste CSV des composants à CALIBRER (skip fixe) ; les autres utilisent leur deser porté.
	// défaut "all". clés : pos,fwdup,angvel,shield,region,obje.
	calSet := map[string]bool{"pos": true, "fwdup": true, "angvel": true, "shield": true, "region": true, "obje": true}
	if len(os.Args) >= 3 {
		calSet = map[string]bool{}
		if os.Args[2] != "none" {
			for _, k := range bytes.Fields([]byte(bytes.ReplaceAll([]byte(os.Args[2]), []byte(","), []byte(" ")))) {
				calSet[string(k)] = true
			}
		}
	}
	calW := map[string]struct {
		name string
		w    int
	}{
		"pos":    {"object-position-dynamic-precision-component", 47},
		"fwdup":  {"object-forward-and-up-component", 9},
		"angvel": {"object-angular-velocity-component", 1},
		"shield": {"object-shield-vitality-component", 29},
		"region": {"object-region-state-component", 358},
		"obje":   {"object-multiplayer-properties-component", 334},
	}
	filmdec.SetRecordStateParam(2)
	var active []string
	for k, v := range calW {
		if calSet[k] {
			filmdec.SetCalibratedWidth(v.name, v.w)
			active = append(active, k)
		}
	}
	fmt.Printf("[calibrés en skip fixe : %v ; les autres via deser porté]\n", active)

	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	type stat struct{ total, valid int }
	byCombo := map[uint16]*stat{}
	perComp := [11]struct{ presVal, presTot, absVal, absTot int }{}
	allTot, allVal := 0, 0

	// arg3 = "clean" → ne compter que les records de frames qui atteignent recEnd (validées).
	cleanOnly := len(os.Args) >= 4 && os.Args[3] == "clean"
	for idx := 2; idx <= maxChunk; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			w := freshWorld(reg)
			br := filmdec.NewBitReader(fr.payload)
			recs, derr := filmdec.DecodeFrameRecords(br, w, calCfg)
			if cleanOnly && derr != nil {
				continue
			}
			for _, r := range recs {
				if !bipedSlots[r.Slot] || r.Trace.Mask&(1<<11) == 0 {
					continue
				}
				d := r.Trace.Dead
				valid := d != nil && d.Mort && d.EnumA >= 0 && d.EnumA <= 7 && d.EnumB >= 0 && d.EnumB <= 7 && d.EnumA != d.EnumB
				var combo uint16
				for i := 0; i <= 10; i++ {
					if r.Trace.Mask&(1<<uint(i)) != 0 {
						combo |= 1 << uint(i)
					}
				}
				s := byCombo[combo]
				if s == nil {
					s = &stat{}
					byCombo[combo] = s
				}
				s.total++
				allTot++
				if valid {
					s.valid++
					allVal++
				}
				for i := 0; i <= 10; i++ {
					pres := r.Trace.Mask&(1<<uint(i)) != 0
					if pres {
						perComp[i].presTot++
						if valid {
							perComp[i].presVal++
						}
					} else {
						perComp[i].absTot++
						if valid {
							perComp[i].absVal++
						}
					}
				}
			}
		}
	}

	fmt.Printf("=== %d records dead-state, %d valides (%.1f%%) ===\n\n", allTot, allVal, 100*float64(allVal)/float64(allTot))

	if z := byCombo[0]; z != nil {
		fmt.Printf(">>> ISOLATION : records SANS aucun composant i0-i10 (masque=i11 seul) : n=%d, valide=%.1f%%\n\n", z.total, 100*float64(z.valid)/float64(z.total))
	} else {
		fmt.Printf(">>> ISOLATION : AUCUN record avec masque=i11 seul (le dead-state ne vient jamais sans pré-composant)\n\n")
	}

	names := []string{"i0-pos", "i1-transvel", "i2-fwdup", "i3-angvel", "i4-bodyvit", "i5-shield", "i6-region", "i7-damage", "i8-constraint", "i9-obje", "i10-parent"}
	fmt.Println("taux valide quand composant PRÉSENT vs ABSENT (coupable = présent bas) :")
	for i := 0; i <= 10; i++ {
		pv, pt, av, at := perComp[i].presVal, perComp[i].presTot, perComp[i].absVal, perComp[i].absTot
		pp, ap := 0.0, 0.0
		if pt > 0 {
			pp = 100 * float64(pv) / float64(pt)
		}
		if at > 0 {
			ap = 100 * float64(av) / float64(at)
		}
		fmt.Printf("  %-14s présent %5.1f%% (n=%4d)   absent %5.1f%% (n=%4d)   Δ=%+5.1f\n", names[i], pp, pt, ap, at, pp-ap)
	}

	fmt.Println("\ntop combos (n, valide%) :")
	type kv struct {
		c uint16
		s *stat
	}
	var kvs []kv
	for c, s := range byCombo {
		kvs = append(kvs, kv{c, s})
	}
	for i := 0; i < len(kvs); i++ {
		for j := i + 1; j < len(kvs); j++ {
			if kvs[j].s.total > kvs[i].s.total {
				kvs[i], kvs[j] = kvs[j], kvs[i]
			}
		}
	}
	for i, k := range kvs {
		if i >= 14 {
			break
		}
		var bits string
		for b := 0; b <= 10; b++ {
			if k.c&(1<<uint(b)) != 0 {
				bits += fmt.Sprintf("i%d ", b)
			}
		}
		fmt.Printf("  n=%4d  valide=%5.1f%%  [%s]\n", k.s.total, 100*float64(k.s.valid)/float64(k.s.total), bits)
	}
}
