// tmp_dscap — THROWAWAY : décode la capture CE dead-state (tools/ce/<prefix>_deadstate.bin,
// records 0x40, cf filmdec_deadstate_capture.lua). Prouve que victime/tueur résolus sont
// sains (cluster sur 8 joueurs, victime==entité du record), et expose la table de
// résolution handle->idx. Bonus : rdtsc = même horloge que le dual-cap 0xd2.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_dscap [prefix]
package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

const wt = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce`

// handle -> idx : essais de bases connues. Les bipeds live = 0xEC500000 + idx*0x10002.
func tryIdx(h uint32) int {
	for _, base := range []uint32{0xEC500000, 0xE1500000} {
		if h >= base {
			d := h - base
			if d%0x10002 == 0 {
				idx := d / 0x10002
				if idx <= 31 {
					return int(idx)
				}
			}
		}
	}
	return -1
}

type rec struct {
	tsc                 uint64
	state               uint32
	victim, killer, gid uint32
	b28, b2c, b38       uint32
	ptr, f0, f8, end    uint32
	accLo, accHi, ptrHi uint32
}

func main() {
	prefix := "000d5950"
	if len(os.Args) >= 2 {
		prefix = os.Args[1]
	}
	path := wt + "/" + prefix + "_deadstate.bin"
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("lecture impossible: %v\n(attends la capture CE)\n", err)
		return
	}
	const stride = 0x60 // record stride (matches lua DREC=0x60 + tmp_oraclediff REC=0x60)
	n := len(raw) / stride
	fmt.Printf("=== %s : %d records dead-state ===\n\n", path, n)
	u := func(off int, b []byte) uint32 { return binary.LittleEndian.Uint32(b[off:]) }
	var recs []rec
	for i := 0; i < n; i++ {
		b := raw[i*stride : i*stride+0x40]
		r := rec{
			tsc:   uint64(u(0, b)) | uint64(u(4, b))<<32,
			state: u(8, b), victim: u(0x0c, b), killer: u(0x10, b), gid: u(0x14, b),
			b28: u(0x18, b), b2c: u(0x1c, b), b38: u(0x20, b),
			ptr: u(0x24, b), f0: u(0x28, b), f8: u(0x2c, b), end: u(0x30, b),
			accLo: u(0x34, b), accHi: u(0x38, b), ptrHi: u(0x3c, b),
		}
		recs = append(recs, r)
	}

	// 1) distribution victime/tueur résolus -> idx
	histVic, histKil := map[int]int{}, map[int]int{}
	rawVic := map[uint32]int{}
	for _, r := range recs {
		histVic[tryIdx(r.victim)]++
		histKil[tryIdx(r.killer)]++
		rawVic[r.victim]++
	}
	dump := func(name string, h map[int]int) {
		type kv struct{ v, n int }
		var kvs []kv
		for v, c := range h {
			kvs = append(kvs, kv{v, c})
		}
		sort.Slice(kvs, func(i, j int) bool { return kvs[i].n > kvs[j].n })
		fmt.Printf("%s -> idx (via base EC500000/E1500000) : ", name)
		for _, k := range kvs {
			fmt.Printf("%d:%d  ", k.v, k.n)
		}
		fmt.Println()
	}
	dump("victime", histVic)
	dump("tueur  ", histKil)

	fmt.Println("\ntop handles victime BRUTS (hex:n) — pour trouver la base si idx tous -1 :")
	type kvh struct {
		h uint32
		n int
	}
	var khs []kvh
	for h, c := range rawVic {
		khs = append(khs, kvh{h, c})
	}
	sort.Slice(khs, func(i, j int) bool { return khs[i].n > khs[j].n })
	for i, k := range khs {
		if i >= 12 {
			break
		}
		fmt.Printf("  0x%08x : %d\n", k.h, k.n)
	}

	// 2) cohérence : par entité (state ptr), victime doit être CONSTANTE (== l'entité),
	//    tueur stable par span de mort. On groupe par state.
	byState := map[uint32][]rec{}
	for _, r := range recs {
		byState[r.state] = append(byState[r.state], r)
	}
	fmt.Printf("\n%d entités distinctes (state ptr). Échantillon (state -> victime résolue, set tueurs) :\n", len(byState))
	var states []uint32
	for s := range byState {
		states = append(states, s)
	}
	sort.Slice(states, func(i, j int) bool { return len(byState[states[i]]) > len(byState[states[j]]) })
	for i, s := range states {
		if i >= 12 {
			break
		}
		rs := byState[s]
		vset, kset := map[uint32]bool{}, map[uint32]bool{}
		for _, r := range rs {
			vset[r.victim] = true
			kset[r.killer] = true
		}
		fmt.Printf("  state=0x%08x n=%-4d victimes=%d tueurs-distincts=%d  (vic idx=%d)\n",
			s, len(rs), len(vset), len(kset), tryIdx(rs[0].victim))
	}

	// 3) GÉOMÉTRIE buffer des records RÉSOLUS (vic != ffffffff) : base candidate f0/f8,
	//    end, ptr, taille (end-base), offset (ptr-base) — pour apparier la frame offline.
	fmt.Println("\nrecords RÉSOLUS (géométrie buffer pour appariement) :")
	shown := 0
	for i := 0; i < n && shown < 20; i++ {
		r := recs[i]
		if r.victim == 0xffffffff {
			continue
		}
		shown++
		szF0 := int64(r.end) - int64(r.f0)
		szF8 := int64(r.end) - int64(r.f8)
		offF0 := int64(r.ptr) - int64(r.f0)
		offF8 := int64(r.ptr) - int64(r.f8)
		fmt.Printf("  vic=idx%d kil=idx%d gid=%08x | f0=%08x f8=%08x ptr=%08x end=%08x | szF0=%d szF8=%d offF0=%d offF8=%d b2c=%d acc=%08x%08x\n",
			tryIdx(r.victim), tryIdx(r.killer), r.gid, r.f0, r.f8, r.ptr, r.end, szF0, szF8, offF0, offF8, r.b2c, r.accHi, r.accLo)
	}
}
