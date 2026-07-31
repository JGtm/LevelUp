// tmp_dsmatch — THROWAWAY : VALIDATION FINALE de la théorie dead-state. Les dead-states
// VALIDES (Mort, EnumA/EnumB in 0-7, EnumA!=EnumB) de frames PROPRES matchent-ils les
// vraies paires (tueur,victime) du kill-feed au bon instant ? Si oui → le décodeur
// dead-state MARCHE (victime=EnumA, tueur=EnumB) et le "2.9%" était la fraction d'onsets,
// pas du bruit. Alors la voie exacte same-clock est ouverte (corréler par packet-ts).
//
// piXuid (slot->xuid) bit-vérifié (HANDOFF). EnumA/EnumB = absolute-participant-index = pi.
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_dsmatch [maxChunk] [winMs]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

var piXuid = map[int]uint64{
	0: 2535467794760703, 1: 2535437947245250, 2: 2533274823110022, 3: 2533274980284321,
	4: 2533274815845110, 5: 2535444178793711, 6: 2533274882097883, 7: 2533274826120416,
}
var xuidName = map[uint64]string{
	2535467794760703: "whiteknight2519", 2535437947245250: "JAVIERLOLITO540",
	2533274823110022: "JGtm", 2533274980284321: "LORD PEINX13",
	2533274815845110: "IKE ILYA", 2535444178793711: "Akatsuki fire17",
	2533274882097883: "aldusbroncus", 2533274826120416: "VitaminA1688",
}

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

type kfRow struct {
	killerPi, victimPi int
	t                  int
}

func main() {
	maxChunk, win := 26, 600
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &maxChunk)
	}
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &win)
	}
	pi := func(x uint64) int {
		for p, xu := range piXuid {
			if xu == x {
				return p
			}
		}
		return -1
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	// kill feed (chunk_27) -> paires (killerPi, victimPi, t)
	mustRead := func(p string) []byte { b, _ := os.ReadFile(p); return b }
	events, _ := analysis.ParseHighlightEvents(mustRead(cache+"/chunk_27.bin"), 0)
	type ev struct {
		x uint64
		t int
	}
	var kills, deaths []ev
	for _, e := range events {
		switch e.EventType {
		case analysis.EventTypeKill:
			kills = append(kills, ev{e.XUID, e.TimeMS})
		case analysis.EventTypeDeath:
			deaths = append(deaths, ev{e.XUID, e.TimeMS})
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
	usedD := make([]bool, len(deaths))
	var feed []kfRow
	for _, k := range kills {
		best, bd := -1, 400
		for i, d := range deaths {
			if usedD[i] || d.x == k.x {
				continue
			}
			dt := k.t - d.t
			if dt < 0 {
				dt = -dt
			}
			if dt < bd {
				bd, best = dt, i
			}
		}
		if best >= 0 {
			usedD[best] = true
			feed = append(feed, kfRow{pi(k.x), pi(deaths[best].x), k.t})
		}
	}
	fmt.Printf("=== kill feed : %d paires (killerPi,victimPi,t) ===\n", len(feed))

	// dead-states VALIDES de frames PROPRES
	type obs struct {
		t      int
		slot   uint32
		eA, eB int32
		gid    uint32
	}
	var valids []obs
	for idx := 2; idx <= maxChunk; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			w := freshWorld(reg)
			br := filmdec.NewBitReader(fr.payload)
			recs, derr := filmdec.DecodeFrameRecords(br, w, calCfg)
			if derr != nil {
				continue // frames propres uniquement
			}
			tms := int((fr.ts - t0Us) / 1000)
			for _, r := range recs {
				d := r.Trace.Dead
				if !bipedSlots[r.Slot] || d == nil || !d.Mort {
					continue
				}
				if d.EnumA >= 0 && d.EnumA <= 7 && d.EnumB >= 0 && d.EnumB <= 7 && d.EnumA != d.EnumB {
					valids = append(valids, obs{tms, r.Slot, d.EnumA, d.EnumB, d.GlobalID})
				}
			}
		}
	}
	fmt.Printf("=== %d dead-states VALIDES (frames propres) ===\n\n", len(valids))

	// MATCH : pour chaque obs, existe-t-il une paire kill-feed (killerPi==EnumB, victimPi==EnumA) à ±win ?
	matchPair, matchKiller, matchVictim, matchSlotVic := 0, 0, 0, 0
	for _, o := range valids {
		var bestPair, bestKiller, bestVictim, bestSlot bool
		for _, f := range feed {
			dt := o.t - f.t
			if dt < 0 {
				dt = -dt
			}
			if dt > win {
				continue
			}
			if f.killerPi == int(o.eB) && f.victimPi == int(o.eA) {
				bestPair = true
			}
			if f.killerPi == int(o.eB) {
				bestKiller = true
			}
			if f.victimPi == int(o.eA) {
				bestVictim = true
			}
			if f.victimPi == int(o.slot-512) {
				bestSlot = true
			}
		}
		if bestPair {
			matchPair++
		}
		if bestKiller {
			matchKiller++
		}
		if bestVictim {
			matchVictim++
		}
		if bestSlot {
			matchSlotVic++
		}
	}
	n := float64(len(valids))
	if n == 0 {
		n = 1
	}
	fmt.Printf("MATCH kill-feed à ±%dms (random ~ pair %.0f%%) :\n", win, 100.0/56)
	fmt.Printf("  paire (EnumB==tueur ET EnumA==victime) : %d (%.1f%%)\n", matchPair, 100*float64(matchPair)/n)
	fmt.Printf("  EnumB==tueur (seul)                    : %d (%.1f%%)\n", matchKiller, 100*float64(matchKiller)/n)
	fmt.Printf("  EnumA==victime (seul)                  : %d (%.1f%%)\n", matchVictim, 100*float64(matchVictim)/n)
	fmt.Printf("  slot-512==victime (contrôle)           : %d (%.1f%%)\n", matchSlotVic, 100*float64(matchSlotVic)/n)

	_ = xuidName
}
