// tmp_deadreach — CHANTIER walk ECS : mesure la portée + validité du dead-state selon le
// chemin de décodage. Objectif : le basique DecodeFrameRecords n'atteint le dead-state au
// bon alignement que 2.2% (tmp_dsvalid) ; les chemins forts (DecodeFrameInfer chaîne+resync,
// ScanFrameTargets bit-scan) atteignent-ils PLUS de dead-states VALIDES (EnumA/EnumB in 0..7
// distincts = tueur/victime plausibles) ? Si oui, la cause mêlée (tag +0x10) devient
// extractible ; si non (bloqué sur les largeurs runtime map-specific), c'est le point CE.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_deadreach [maxChunk] [calib=1|0]
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
	ts     uint64
	marker byte
	pl     []byte
}

func listFrames(d []byte) []packet {
	var out []packet
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 && sz > 0 {
			pl := d[off+16 : off+16+sz]
			out = append(out, packet{ts, pl[0], pl})
		}
		off += 16 + sz
	}
	return out
}

var dumpFile = "world_dump.txt"

func freshWorld(reg *filmdec.Registry) *filmdec.World {
	raw, _ := os.ReadFile(cache + "/" + dumpFile)
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

// deadStat agrège la portée + validité des dead-states pour un chemin de décodage.
type deadStat struct {
	nDead    int             // dead-states Mort==true rencontrés
	nValid   int             // dont EnumA/EnumB in 0..7 distincts (tueur/victime plausibles)
	gidSet   int             // dont +0x10 (GlobalID) présent (!=ffffffff)
	byMarker map[byte][2]int // marqueur -> [nDead, nValid]
}

func newDeadStat() *deadStat { return &deadStat{byMarker: map[byte][2]int{}} }

func (s *deadStat) add(mk byte, d *filmdec.DeadState) {
	if d == nil || !d.Mort {
		return
	}
	s.nDead++
	valid := d.EnumA >= 0 && d.EnumA <= 7 && d.EnumB >= 0 && d.EnumB <= 7 && d.EnumA != d.EnumB
	m := s.byMarker[mk]
	m[0]++
	if valid {
		s.nValid++
		m[1]++
	}
	s.byMarker[mk] = m
	if d.GlobalID != 0xFFFFFFFF {
		s.gidSet++
	}
}

func (s *deadStat) report(label string) {
	pv := 0.0
	if s.nDead > 0 {
		pv = 100 * float64(s.nValid) / float64(s.nDead)
	}
	fmt.Printf("  %-42s : %4d dead-states, %4d valides (%.1f%%), %d avec GID(+0x10)\n", label, s.nDead, s.nValid, pv, s.gidSet)
}

func collectDead(recs []filmdec.FrameRecord, mk byte, s *deadStat) {
	for _, r := range recs {
		if !bipedSlots[r.Slot] {
			continue
		}
		s.add(mk, r.Trace.Dead)
	}
}

func main() {
	maxChunk := 41
	calib := true
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &maxChunk)
	}
	if len(os.Args) >= 3 && os.Args[2] == "0" {
		calib = false
	}
	scanOn := false
	if len(os.Args) >= 4 {
		dumpFile = os.Args[3] // ex: world_dump_full.txt
	}
	if len(os.Args) >= 5 && os.Args[4] == "scan" {
		scanOn = true
	}
	filmdec.SetRecordStateParam(2)
	if calib {
		// calibration tmp_dsvalid (skip fixe des composants à largeur runtime map-specific).
		for name, w := range map[string]int{
			"object-position-dynamic-precision-component": 47,
			"object-forward-and-up-component":             9,
			"object-angular-velocity-component":           1,
			"object-shield-vitality-component":            29,
			"object-region-state-component":               358,
			"object-multiplayer-properties-component":     334,
		} {
			filmdec.SetCalibratedWidth(name, w)
		}
	}
	filmdec.SetInferResyncTargets(bipedSlots)

	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	fatalMk := map[byte]bool{0xD2: true, 0xC0: true, 0xC2: true, 0xC3: true, 0xCA: true, 0xD3: true, 0xE9: true}

	stBasic := newDeadStat()    // DecodeFrameRecords (baseline)
	stInfer := newDeadStat()    // DecodeFrameInfer (chaîne + resync)
	stScan := newDeadStat()     // ScanFrameTargets (bit-scan indépendant du chaînage)
	stBasicFat := newDeadStat() // baseline, paquets fataux uniquement
	stScanFat := newDeadStat()  // scan, paquets fataux uniquement

	filmdec.SetInferChain(true)

	for ch := 0; ch <= maxChunk; ch++ {
		for _, p := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))) {
			// Path A : basique
			br := filmdec.NewBitReader(p.pl)
			recsA, _ := filmdec.DecodeFrameRecords(br, freshWorld(reg), calCfg)
			collectDead(recsA, p.marker, stBasic)
			if fatalMk[p.marker] {
				collectDead(recsA, p.marker, stBasicFat)
			}
			// Path B : infer (chaîne + resync)
			recsB, _ := filmdec.DecodeFrameInfer(p.pl, freshWorld(reg), calCfg)
			collectDead(recsB, p.marker, stInfer)
			// Path C : scan bit-indépendant (cher : opt-in via arg "scan", paquets fataux)
			if scanOn && fatalMk[p.marker] {
				recsC := filmdec.ScanFrameTargets(p.pl, freshWorld(reg), calCfg, bipedSlots, filmdec.HarvestNextBound)
				collectDead(recsC, p.marker, stScan)
				collectDead(recsC, p.marker, stScanFat)
			}
		}
	}

	fmt.Printf("=== DEADREACH 000d5950 (calib=%v, chunks 0..%d) — dead-states biped par chemin ===\n", calib, maxChunk)
	stBasic.report("A. DecodeFrameRecords (baseline, tous mk)")
	stInfer.report("B. DecodeFrameInfer chaîne+resync (tous mk)")
	fmt.Println("  --- paquets FATAUX seulement ---")
	stBasicFat.report("A'. DecodeFrameRecords (fataux)")
	stScanFat.report("C. ScanFrameTargets bit-scan (fataux)")
	fmt.Println("\n  répartition par marqueur (ScanFrameTargets, fataux) [nDead/nValid] :")
	for _, mk := range []byte{0xD2, 0xC0, 0xC2, 0xC3, 0xCA, 0xD3, 0xE9} {
		if m, ok := stScanFat.byMarker[mk]; ok {
			fmt.Printf("    0x%02X : %d/%d\n", mk, m[0], m[1])
		}
	}
}
