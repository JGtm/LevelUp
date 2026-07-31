// tmp_fireevt — harnais THROWAWAY : histogramme des types d'events du film (payload[0]>>1)
// et décodage de la tête du record type 105 (event de tir) selon la spec Ghidra.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_fireevt <filmDir> [csvOut]
package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: tmp_fireevt <filmDir> [csvOut]")
		os.Exit(2)
	}
	dir := os.Args[1]
	n := filmdec.CountFilmChunks(dir)
	fmt.Printf("chunks=%d dir=%s\n", n, dir)

	histType := map[int]int{}
	histByte := map[byte]int{}
	type ev struct {
		chunk, pkt       int
		ts               uint64
		variant          int
		attackerRaw      uint32
		wHigh, wLow      uint32
		f108, f109       uint32
		f110, f111, f112 uint32
		aimOK            bool
		aimCode          uint32
		ax, ay, az       float32
		payloadLen       int
	}
	var evs []ev
	nType0 := 0

	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta || p.Size < 1 {
				continue
			}
			nType0++
			pay := p.Payload(chunk)
			t := int(pay[0] >> 1)
			histType[t]++
			histByte[pay[0]]++
			if t != 105 {
				continue
			}
			e := ev{chunk: c, pkt: p.Index, ts: p.TimestampUS, variant: int(pay[0] & 1), payloadLen: p.Size}
			e.attackerRaw = filmdec.ReadBitsAtForDiag(pay, 36, 5)
			e.wHigh = filmdec.ReadBitsAtForDiag(pay, 44, 32)
			e.wLow = filmdec.ReadBitsAtForDiag(pay, 76, 32)
			e.f108 = filmdec.ReadBitsAtForDiag(pay, 108, 1)
			e.f109 = filmdec.ReadBitsAtForDiag(pay, 109, 1)
			e.f110 = filmdec.ReadBitsAtForDiag(pay, 110, 1)
			e.f111 = filmdec.ReadBitsAtForDiag(pay, 111, 1)
			e.f112 = filmdec.ReadBitsAtForDiag(pay, 112, 1)
			if e.variant == 0 && e.f110 == 1 && e.f111 == 0 && e.f112 == 0 && p.Size*8 >= 143 {
				e.aimCode = filmdec.ReadBitsAtForDiag(pay, 113, 30)
				v, ok := filmdec.DecodeAimVectorChecked(e.aimCode, 30)
				e.aimOK = ok
				e.ax, e.ay, e.az = v[0], v[1], v[2]
			}
			evs = append(evs, e)
		}
	}

	fmt.Printf("paquets type0=%d  events type105=%d\n", nType0, len(evs))
	type kv struct {
		k, v int
	}
	var list []kv
	for k, v := range histType {
		list = append(list, kv{k, v})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].v > list[j].v })
	fmt.Println("-- histogramme des types (top 20) --")
	for i, e := range list {
		if i >= 20 {
			break
		}
		fmt.Printf("  type %3d : %6d\n", e.k, e.v)
	}
	fmt.Println("-- octets 0 des type-105 --")
	for b, c := range histByte {
		if int(b>>1) == 105 {
			fmt.Printf("  0x%02X : %d\n", b, c)
		}
	}

	// statistiques rapides
	nLong, nShort, nAim := 0, 0, 0
	slotHist := map[uint32]int{}
	oddSlot := 0
	for _, e := range evs {
		if e.variant == 0 {
			nLong++
			slotHist[e.attackerRaw]++
			if e.attackerRaw&1 == 1 {
				oddSlot++
			}
			if e.aimOK {
				nAim++
			}
		} else {
			nShort++
		}
	}
	fmt.Printf("long(0xD2)=%d court(0xD3)=%d visée décodée=%d (%.1f%% des longs)\n",
		nLong, nShort, nAim, 100*float64(nAim)/float64(maxi(nLong, 1)))
	fmt.Printf("attackerRaw impairs = %d / %d\n", oddSlot, nLong)
	var ks []int
	for k := range slotHist {
		ks = append(ks, int(k))
	}
	sort.Ints(ks)
	for _, k := range ks {
		fmt.Printf("  attackerRaw=%2d (slot %2d) : %d\n", k, k/2, slotHist[uint32(k)])
	}

	if len(os.Args) < 3 {
		return
	}
	f, err := os.Create(os.Args[2])
	if err != nil {
		fmt.Println("csv:", err)
		return
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"chunk", "pkt", "ts_us", "variant", "attacker_raw", "player", "weapon64",
		"f108", "f109", "f110", "f111", "f112", "aim_ok", "aim_code", "ax", "ay", "az", "paylen"})
	for _, e := range evs {
		w64 := uint64(e.wHigh)<<32 | uint64(e.wLow)
		_ = w.Write([]string{
			strconv.Itoa(e.chunk), strconv.Itoa(e.pkt), strconv.FormatUint(e.ts, 10),
			strconv.Itoa(e.variant), strconv.Itoa(int(e.attackerRaw)), strconv.Itoa(int(e.attackerRaw >> 1)),
			fmt.Sprintf("0x%016X", w64),
			strconv.Itoa(int(e.f108)), strconv.Itoa(int(e.f109)), strconv.Itoa(int(e.f110)),
			strconv.Itoa(int(e.f111)), strconv.Itoa(int(e.f112)),
			strconv.FormatBool(e.aimOK), strconv.Itoa(int(e.aimCode)),
			fmt.Sprintf("%.6f", e.ax), fmt.Sprintf("%.6f", e.ay), fmt.Sprintf("%.6f", e.az),
			strconv.Itoa(e.payloadLen),
		})
	}
	fmt.Println("csv écrit :", os.Args[2])
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}
