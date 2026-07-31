// tmp_killeridx — THROWAWAY : VALIDATION que le dead-state du biped (FUN_140c1dd44) porte bien
// le tueur et la victime en clair, comme le nomme le sérialiseur FUN_143203460 :
//
//	+4 = absolute-participant-index (VICTIME), +8 = killer-absolute-participant-index (TUEUR).
//
// Le DeadState Go les capture en EnumA(+4)/EnumB(+8). Personne n'a jamais vérifié qu'ils
// bijectent les joueurs connus. On le fait ici contre la vérité-terrain kill feed chunk_27 (94/94).
//
// Méthode (ZÉRO interprétation codée en dur) :
//  1. chunk_27 -> kills (tueur xuid, t) + deaths (victime xuid, t), appariés en feed.
//  2. Décodage de tous les dead-states CLEAN (DesyncAt==-1, Mort==true) : (t, slot, EnumA, EnumB, GID).
//  3. Chaque dead-state obs -> DEATH event le plus proche (±win) => (obs <-> victime xuid).
//     Puis le kill correspondant => tueur xuid.
//  4. Cross-tabs : slot->victime, EnumA->victime, EnumB->TUEUR, + GID par (tueur->victime).
//     Si EnumB->tueur et EnumA->victime sont des bijections => décodeur bit-exact jusqu'à +0x10
//     => l'arme (GID source @+0x10) est bien lue, reste à la résoudre.
//
// Usage : tmp_killeridx [maxChunk] [matchWinMs]   (CGO_ENABLED=0, pas de DuckDB)
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

// pi (player-index 0..7) -> xuid, bit-vérifié (HANDOFF).
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

func xuidPi(x uint64) int {
	for pi, xu := range piXuid {
		if xu == x {
			return pi
		}
	}
	return -1
}
func nm(x uint64) string {
	if g, ok := xuidName[x]; ok {
		return g
	}
	return fmt.Sprintf("xuid:%d", x)
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

type ev struct {
	xuid uint64
	t    int
}
type deadObs struct {
	t    int
	slot uint32
	ds   filmdec.DeadState
}
type kfRow struct {
	killer, victim uint64
	t              int
}

func main() {
	maxChunk := 26
	win := 250
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &maxChunk)
	}
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &win)
	}
	filmdec.SetRecordStateParam(2)
	// CALIBRATION complète (largeurs CE constantes, filmdec_delta_death.csv) des composants
	// pré-i11. Diff propre i0→i9 avec ces valeurs.
	filmdec.SetCalibratedWidth("object-position-dynamic-precision-component", 47) // i0
	filmdec.SetCalibratedWidth("object-forward-and-up-component", 9)              // i2
	filmdec.SetCalibratedWidth("object-angular-velocity-component", 1)            // i3
	filmdec.SetCalibratedWidth("object-shield-vitality-component", 29)            // i5
	filmdec.SetCalibratedWidth("object-region-state-component", 358)              // i6
	filmdec.SetCalibratedWidth("object-multiplayer-properties-component", 334)    // i9
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	// 1) kill feed chunk_27
	events, _ := analysis.ParseHighlightEvents(mustRead(cache+"/chunk_27.bin"), 0)
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
			if usedD[i] || d.xuid == k.xuid {
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
			feed = append(feed, kfRow{k.xuid, deaths[best].xuid, k.t})
		}
	}
	fmt.Printf("=== kill feed : %d kills appariés (%d kills, %d deaths) ===\n", len(feed), len(kills), len(deaths))

	// 2) dead-states CLEAN Mort==true
	var dos []deadObs
	for idx := 2; idx <= maxChunk; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			w := freshWorld(reg)
			br := filmdec.NewBitReader(fr.payload)
			recs, _ := filmdec.DecodeFrameRecords(br, w, calCfg)
			tms := int((fr.ts - t0Us) / 1000)
			for _, r := range recs {
				// On NE requiert PLUS DesyncAt==-1 : le deser dead-state ne lit que la tête
				// (52 bits) et désync APRÈS i11 (queue ~95 bits non portée) ; mais la tête
				// (EnumA/EnumB/GID) est déjà lue au bon endroit car i0..i9 sont calibrés.
				if !bipedSlots[r.Slot] || r.Trace.Dead == nil || !r.Trace.Dead.Mort {
					continue
				}
				dos = append(dos, deadObs{tms, r.Slot, *r.Trace.Dead})
			}
		}
	}
	sort.Slice(dos, func(i, j int) bool { return dos[i].t < dos[j].t })
	fmt.Printf("=== dead-states CLEAN Mort==true : %d obs ===\n\n", len(dos))

	// 3) ISOLATION PAR SLOT : on ne décode de façon fiable que le slot 519 (limite boucle records).
	// Slot 519 = UN joueur. On groupe ses obs Mort==true en "spans de mort" (gap>800ms = nouvelle
	// mort), onset = 1re obs du span = l'instant de mort. => séquence de morts propres du slot.
	bySlot := map[uint32][]deadObs{}
	for _, o := range dos {
		bySlot[o.slot] = append(bySlot[o.slot], o)
	}
	type onset struct {
		t            int
		enumA, enumB int32
		gid          uint32
	}
	onsetsBySlot := map[uint32][]onset{}
	for slot, obs := range bySlot {
		sort.Slice(obs, func(i, j int) bool { return obs[i].t < obs[j].t })
		last := -100000
		for _, o := range obs {
			if o.t-last > 800 {
				onsetsBySlot[slot] = append(onsetsBySlot[slot], onset{o.t, o.ds.EnumA, o.ds.EnumB, o.ds.GlobalID})
			}
			last = o.t
		}
	}

	// 4) Pour chaque slot avec >=4 onsets, identifier le JOUEUR = celui dont les morts (chunk_27)
	// couvrent le mieux les onsets ; puis tester EnumB == tueur de CE joueur à chaque mort.
	var slots []uint32
	for s := range onsetsBySlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	for _, slot := range slots {
		ons := onsetsBySlot[slot]
		if len(ons) < 4 {
			continue
		}
		// identifier le joueur : pour chaque joueur P, combien de SES morts tombent à ±win d'un onset.
		bestP, bestCov := uint64(0), -1
		for _, P := range piXuid {
			cov := 0
			for _, d := range deaths {
				if d.xuid != P {
					continue
				}
				for _, o := range ons {
					dt := o.t - d.t
					if dt < 0 {
						dt = -dt
					}
					if dt <= win {
						cov++
						break
					}
				}
			}
			if cov > bestCov {
				bestCov, bestP = cov, P
			}
		}
		// nb morts de bestP au total
		nDeathsP := 0
		for _, d := range deaths {
			if d.xuid == bestP {
				nDeathsP++
			}
		}
		fmt.Printf("=== slot%d : %d onsets ; joueur identifié = %s (couvre %d/%d de ses morts à ±%dms) ===\n",
			slot, len(ons), nm(bestP), bestCov, nDeathsP, win)

		// pour chaque onset, la mort de bestP la plus proche -> tueur attendu ; comparer EnumB.
		okB, okA, nB := 0, 0, 0
		for _, o := range ons {
			// mort de bestP la plus proche
			var dt0 = win + 1
			var dtime = -1
			for _, d := range deaths {
				if d.xuid != bestP {
					continue
				}
				dt := o.t - d.t
				if dt < 0 {
					dt = -dt
				}
				if dt < dt0 {
					dt0, dtime = dt, d.t
				}
			}
			if dtime < 0 {
				fmt.Printf("    onset %6.1fs EnumA=%-3d EnumB=%-3d GID=0x%08x  (pas de mort %s appariée)\n",
					float64(o.t)/1000, o.enumA, o.enumB, o.gid, nm(bestP))
				continue
			}
			// tueur de bestP à dtime
			killer := uint64(0)
			kdt := 600
			for _, f := range feed {
				if f.victim != bestP {
					continue
				}
				dt := dtime - f.t
				if dt < 0 {
					dt = -dt
				}
				if dt < kdt {
					kdt, killer = dt, f.killer
				}
			}
			nB++
			kpi := xuidPi(killer)
			vpi := xuidPi(bestP)
			matchB := int(o.enumB) == kpi
			matchA := int(o.enumA) == vpi
			if matchB {
				okB++
			}
			if matchA {
				okA++
			}
			fmt.Printf("    onset %6.1fs EnumA=%-3d(victime pi%d %s) EnumB=%-3d(tueur pi%d %s) GID=0x%08x  %s\n",
				float64(o.t)/1000, o.enumA, vpi, btoa(matchA), o.enumB, kpi, btoa(matchB), o.gid, nm(killer))
		}
		if nB > 0 {
			fmt.Printf("    >>> EnumB==tueur.pi : %d/%d (%.0f%%)  |  EnumA==victime.pi : %d/%d (%.0f%%)\n\n",
				okB, nB, 100*float64(okB)/float64(nB), okA, nB, 100*float64(okA)/float64(nB))
		}
	}

	fmt.Println(">>> Lecture : si EnumB==tueur.pi élevé sur slot519 => +8 EST killer-participant-index (décodeur bit-exact jusqu'à +0x10).")
	fmt.Println(">>> GID présent à un onset => candidat référence source/arme @+0x10 (à résoudre).")
}

func btoa(b bool) string {
	if b {
		return "OK"
	}
	return "X"
}

func mustRead(p string) []byte { b, _ := os.ReadFile(p); return b }
