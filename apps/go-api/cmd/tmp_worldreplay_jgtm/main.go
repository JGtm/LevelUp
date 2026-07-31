// tmp_worldreplay_jgtm — THROWAWAY : rejoue les deltas type-0 d'un film NEUF (jgtm)
// SANS world_dump CE. World = seed bipeds (512-519 -> typeIndex 35) + auto-bind des
// records NEW propres (DecodeFrameRecords binde le slot quand un NEW décode clean).
// Mesure : combien de bipeds atteints/clean, WST (arme high-32) lus, dead-states (mort
// + GlobalID + EnumA/B) capturés. C'est le jalon "replay delta tous joueurs" sur jgtm.
//
// Usage : tmp_worldreplay_jgtm <dir_chunks> [npkts]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const bipedTypeIndex = 35

func inflate(p string) []byte {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); (e2 == nil || e2 == io.ErrUnexpectedEOF) && len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

type packet struct {
	typ  uint16
	ts   uint64
	data []byte
}

func listPackets(d []byte) []packet {
	var out []packet
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		out = append(out, packet{typ, ts, d[off+16 : off+16+sz]})
		off += 16 + sz
	}
	return out
}

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

func h2nHigh(v uint32) (string, bool) {
	for id, n := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return n, true
		}
	}
	return "", false
}

var bipedSlots = map[uint32]bool{}

// seedWorld binde les slots biped DÉCOUVERTS (bipedSlots) sur l'archétype #35.
func seedWorld(reg *filmdec.Registry) *filmdec.World {
	w := filmdec.NewWorld(reg)
	for s := range bipedSlots {
		w.BindFull(s, bipedTypeIndex)
	}
	return w
}

func main() {
	dir := "internal/sync/testdata/jgtm_full_match/chunks"
	npkts := 200
	if len(os.Args) >= 2 {
		dir = os.Args[1]
	}
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &npkts)
	}
	filmdec.SetRecordStateParam(2)

	files, _ := filepath.Glob(filepath.Join(dir, "filmChunk*"))
	sort.Slice(files, func(i, j int) bool { return chunkNum(files[i]) < chunkNum(files[j]) })
	reg, err := filmdec.ParseRegistryChunk(inflate(files[0]))
	if err != nil {
		panic(err)
	}
	fmt.Printf("registre %d archétypes ; biped #35 OK=%v\n", len(reg.Archetypes), func() bool { _, ok := reg.Archetype(35); return ok }())

	// collecte tous les paquets type-0 (FRAME) dans l'ordre
	var t0 []packet
	for _, f := range files {
		for _, p := range listPackets(inflate(f)) {
			if p.typ == 0 {
				t0 = append(t0, p)
			}
		}
	}
	fmt.Printf("paquets type-0 (FRAME) total = %d\n", len(t0))
	if len(t0) == 0 {
		return
	}

	// --- CALIBRATION IDLowBits : le BON idLow concentre le 1er record sur ~8 slots
	// (les bipeds joueurs émettent une position-delta chaque frame). On découvre les
	// slots biped au lieu de présumer 512-519. ---
	fmt.Printf("\n=== CALIBRATION IDLowBits (concentration du 1er record sur ~8 slots) — %d paquets ===\n", len(t0))
	firstSlot := func(p packet, idLow int) uint32 {
		br := filmdec.NewBitReader(p.data)
		if !br.ReadBit() {
			br.ReadBits(2)
		}
		var low uint32
		if idLow > 0 {
			low = uint32(br.ReadBits(uint(idLow)))
		}
		tag := uint32(br.ReadBits(2))
		return ((tag << 30) | (low & 0x3fffffff)) & 0x3fffffff
	}
	type cand struct {
		idLow    int
		distinct int
		top      []uint32
		topCnt   []int
		cover    int // % couvert par le top-8
	}
	var cands []cand
	for idLow := 8; idLow <= 16; idLow++ {
		h := map[uint32]int{}
		for _, p := range t0 {
			h[firstSlot(p, idLow)]++
		}
		type kv struct {
			s uint32
			c int
		}
		var arr []kv
		for s, c := range h {
			arr = append(arr, kv{s, c})
		}
		sort.Slice(arr, func(a, b int) bool { return arr[a].c > arr[b].c })
		top8 := 0
		var tops []uint32
		var topc []int
		for k := 0; k < len(arr) && k < 8; k++ {
			top8 += arr[k].c
			tops = append(tops, arr[k].s)
			topc = append(topc, arr[k].c)
		}
		cands = append(cands, cand{idLow, len(h), tops, topc, 100 * top8 / len(t0)})
		fmt.Printf("  idLowBits=%-2d : slots distincts=%-6d ; top-8 couvre %3d%% ; top slots=%v\n",
			idLow, len(h), 100*top8/len(t0), tops)
	}
	// meilleur = celui dont le mini-replay (top-8 slots seedés) produit le PLUS de WST
	// nommés high32 (= bipeds qui décodent vraiment + armes lues). La couverture seule
	// est biaisée vers les petits idLow (collisions). On teste la QUALITÉ.
	fmt.Printf("\n--- mini-replay par idLow (400 paquets, top-8 slots seedés) : qualité ---\n")
	probe := 400
	if probe > len(t0) {
		probe = len(t0)
	}
	bestNamed, bestIdx := -1, 0
	for ci := range cands {
		c := cands[ci]
		seed := map[uint32]bool{}
		for _, s := range c.top {
			seed[s] = true
		}
		w := filmdec.NewWorld(reg)
		for s := range seed {
			w.BindFull(s, bipedTypeIndex)
		}
		cfgP := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: c.idLow}
		clean, named := 0, 0
		for i := 0; i < probe; i++ {
			br := filmdec.NewBitReader(t0[i].data)
			recs, _ := filmdec.DecodeFrameRecords(br, w, cfgP)
			for _, r := range recs {
				if !seed[r.Slot] {
					continue
				}
				if r.DesyncAt == -1 {
					clean++
				}
				for _, cc := range r.Trace.Comps {
					if cc.Name != "weapon-state-type-info" {
						continue
					}
					h := uint32(bitsAt(t0[i].data, cc.StartBit+1, 32))
					v := uint32(bitsAt(t0[i].data, cc.StartBit+filmdec.VariantBitOffsetInWST, 32))
					if _, ok := h2nHigh(h); ok {
						named++
					} else if _, ok := h2nHigh(v); ok {
						named++
					}
				}
			}
		}
		fmt.Printf("  idLow=%-2d : cleanBiped=%-5d namedWST=%-4d slots=%v\n", c.idLow, clean, named, c.top)
		if named > bestNamed {
			bestNamed, bestIdx = named, ci
		}
	}
	best := cands[bestIdx]
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: best.idLow}
	bipedSlots = map[uint32]bool{}
	for _, s := range best.top {
		bipedSlots[s] = true
	}
	fmt.Printf(">>> cfg retenu (qualité) : idLowBits=%d ; slots biped=%v ; namedWST=%d\n",
		best.idLow, best.top, bestNamed)

	// --- REPLAY persistant (World seedé + auto-bind) ---
	fmt.Printf("\n=== REPLAY %d paquets (World persistant seed 512-519 + auto-bind) ===\n", min(npkts, len(t0)))
	w := seedWorld(reg)
	var (
		totalRecs, bipedRecs, bipedClean int
		wstSeen, wstNamed                int
		deadSeen, deadWithGID            int
	)
	wstFam := map[string]int{}
	var deadSamples []string
	for i := 0; i < len(t0) && i < npkts; i++ {
		br := filmdec.NewBitReader(t0[i].data)
		recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
		for _, r := range recs {
			totalRecs++
			if !bipedSlots[r.Slot] {
				continue
			}
			bipedRecs++
			if r.DesyncAt == -1 {
				bipedClean++
			}
			if r.Trace.Dead != nil {
				deadSeen++
				ds := r.Trace.Dead
				if ds.GIDPresent && ds.GlobalID != 0xFFFFFFFF {
					deadWithGID++
				}
				if len(deadSamples) < 25 {
					gidName := ""
					if n, ok := h2nHigh(ds.GlobalID); ok {
						gidName = " GID.high32=" + n
					}
					deadSamples = append(deadSamples, fmt.Sprintf(
						"pkt#%d slot=%d mort=%v EnumA=%d EnumB=%d hasRef=%v GID=0x%08x%s",
						i, r.Slot, ds.Mort, ds.EnumA, ds.EnumB, ds.HasRef, ds.GlobalID, gidName))
				}
			}
			for _, c := range r.Trace.Comps {
				if c.Name != "weapon-state-type-info" {
					continue
				}
				wstSeen++
				handle := uint32(bitsAt(t0[i].data, c.StartBit+1, 32))
				variant := uint32(bitsAt(t0[i].data, c.StartBit+filmdec.VariantBitOffsetInWST, 32))
				if n, ok := h2nHigh(handle); ok {
					wstNamed++
					wstFam[n]++
				} else if n, ok := h2nHigh(variant); ok {
					wstNamed++
					wstFam[n]++
				}
			}
		}
	}
	fmt.Printf("records totaux=%d ; biped records=%d (clean=%d) ; WST vus=%d (nommés high32=%d)\n",
		totalRecs, bipedRecs, bipedClean, wstSeen, wstNamed)
	fmt.Printf("dead-states biped vus=%d (avec GlobalID présent=%d)\n", deadSeen, deadWithGID)

	if len(wstFam) > 0 {
		fmt.Println("\n--- familles d'armes lues dans les WST biped (delta) ---")
		type kv struct {
			k string
			v int
		}
		var arr []kv
		for k, v := range wstFam {
			arr = append(arr, kv{k, v})
		}
		sort.Slice(arr, func(a, b int) bool { return arr[a].v > arr[b].v })
		for _, e := range arr {
			fmt.Printf("  %-26s x%d\n", e.k, e.v)
		}
	}
	if len(deadSamples) > 0 {
		fmt.Println("\n--- échantillons dead-state biped (mort + réf arme/source) ---")
		for _, s := range deadSamples {
			fmt.Println("  " + s)
		}
	}
}

func chunkNum(p string) int {
	base := filepath.Base(p)
	n, started := 0, false
	for _, c := range base {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
			started = true
		} else if started {
			break
		}
	}
	return n
}
