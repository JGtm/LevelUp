// tmp_victimpos — PARTIE 1 : POSITION VICTIME par slot/joueur sur 000d5950.
//
// Extrait la position absolue du biped i0 (object-position-dynamic-precision) par
// frame, l'attribue à un slot via le record FRAME, accumule par slot (baseline absolu
// + accumulation des deltas), et mappe slot->joueur (vote timing de mort, chunk_27).
//
// Sortie : pour les victimes narrées (JGtm, Akatsuki) aux temps des kills, leur
// position (x,y,z) au sample le plus proche, + sanity (déplacement continu).
//
// Décode = Go PUR (CGO_ENABLED=0 sauf duckdb gamertags). Usage : tmp_victimpos [maxChunk]
package main

import (
	"bytes"
	"compress/zlib"
	"database/sql"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"

	_ "github.com/duckdb/duckdb-go/v2"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const sharedDB = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
const t0Us = uint64(4537898226)

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

var xuidGamertag = map[uint64]string{
	2535467794760703: "whiteknight2519",
	2535437947245250: "JAVIERLOLITO540",
	2533274823110022: "JGtm",
	2533274980284321: "LORD PEINX13",
	2533274815845110: "IKE ILYA",
	2535444178793711: "Akatsuki fire17",
	2533274882097883: "aldusbroncus",
	2533274826120416: "VitaminA1688",
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

// posSample : une position i0 décodée, horodatée, attribuée à un slot.
type posSample struct {
	timeMs int
	kind   filmdec.PosKind
	vec    [3]float32
}

func main() {
	maxChunk := 26
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &maxChunk)
	}
	filmdec.SetRecordStateParam(2)
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	// hook global : collecte les samples i0 du frame courant (clé = BitPos de départ
	// du composant i0 == comps[0].StartBit du record).
	var frameSamples []filmdec.PositionSample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) {
		frameSamples = append(frameSamples, s)
	})

	// timeline de positions par slot + ticks de mort (pour le vote slot->xuid).
	posBySlot := map[uint32][]posSample{}
	deathBySlot := map[uint32][]int{}

	kindCount := map[filmdec.PosKind]int{}

	for idx := 2; idx <= maxChunk; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			w := freshWorld(reg)
			br := filmdec.NewBitReader(fr.payload)
			frameSamples = frameSamples[:0]
			recs, _ := filmdec.DecodeFrameRecords(br, w, calCfg)
			tms := int((fr.ts - t0Us) / 1000)

			// index des samples par BitPos pour attribution O(1).
			byBit := map[int]filmdec.PositionSample{}
			for _, s := range frameSamples {
				byBit[s.BitPos] = s
			}
			for _, r := range recs {
				if !bipedSlots[r.Slot] || len(r.Trace.Comps) == 0 {
					continue
				}
				// i0 est le 1er composant présent ; son StartBit == comps[0].StartBit
				// SSI i0 est présent dans le mask (sinon comps[0] est un autre composant).
				c0 := r.Trace.Comps[0]
				if c0.Name != "object-position-dynamic-precision-component" {
					continue
				}
				if s, ok := byBit[c0.StartBit]; ok {
					kindCount[s.Kind]++
					posBySlot[r.Slot] = append(posBySlot[r.Slot], posSample{tms, s.Kind, s.Vec})
				}
				if r.DesyncAt == -1 && r.Trace.Dead != nil && r.Trace.Dead.Mort {
					deathBySlot[r.Slot] = append(deathBySlot[r.Slot], tms)
				}
			}
		}
	}
	filmdec.SetPositionCaptureHook(nil)

	fmt.Println("=== samples i0 capturés par type d'encodage ===")
	for k, c := range kindCount {
		fmt.Printf("  %-4s x%d\n", k.String(), c)
	}

	// DIAGNOSTIC : combien d'absolus quantisés sortent de [-100,100] (devrait être 0
	// si le décode quantisé est bit-exact) + échantillon raw pour juger leur plausibilité.
	absOut, absTot, rawShown := 0, 0, 0
	fmt.Println("=== diag : échantillons bruts (raw 3xfloat32) ===")
	for s := uint32(512); s <= 519; s++ {
		for _, p := range posBySlot[s] {
			if p.kind == filmdec.PosKindAbsolute {
				absTot++
				for a := 0; a < 3; a++ {
					if p.vec[a] < -100.001 || p.vec[a] > 100.001 {
						absOut++
						break
					}
				}
			}
			if p.kind == filmdec.PosKindRaw && rawShown < 8 {
				fmt.Printf("  slot%d raw @%.1fs (%.3g, %.3g, %.3g)\n", s, float64(p.timeMs)/1000, p.vec[0], p.vec[1], p.vec[2])
				rawShown++
			}
		}
	}
	fmt.Printf("=== diag : absolus quantisés hors [-100,100] = %d/%d ===\n", absOut, absTot)

	// DIAG continuité : timeline chrono des absolus de slot519 (le plus décodé).
	// Un vrai joueur bouge en continu → pas petits & cohérents. Du bruit (mauvaise
	// largeur/spine) → sauts uniformément distribués dans la boîte.
	fmt.Println("=== diag : timeline absolus slot519 (20 premiers) ===")
	var s519 []posSample
	for _, p := range posBySlot[519] {
		if p.kind == filmdec.PosKindAbsolute { // DIRECT abs uniquement (pas le fallback bruité)
			s519 = append(s519, p)
		}
	}
	sort.Slice(s519, func(i, j int) bool { return s519[i].timeMs < s519[j].timeMs })
	for i := 0; i < len(s519) && i < 20; i++ {
		step := 0.0
		if i > 0 {
			step = dist(s519[i-1].vec, s519[i].vec)
		}
		fmt.Printf("  @%6.1fs (%7.2f,%7.2f,%7.2f)  pas=%.1f\n",
			float64(s519[i].timeMs)/1000, s519[i].vec[0], s519[i].vec[1], s519[i].vec[2], step)
	}
	fmt.Println("=== samples i0 par slot biped ===")
	for s := uint32(512); s <= 519; s++ {
		fmt.Printf("  slot%d : %d samples\n", s, len(posBySlot[s]))
	}

	// accumulation : baseline = 1er sample absolu/raw ; deltas accumulés ensuite.
	accBySlot := map[uint32][]posSample{}
	for s := uint32(512); s <= 519; s++ {
		raw := posBySlot[s]
		sort.Slice(raw, func(i, j int) bool { return raw[i].timeMs < raw[j].timeMs })
		var cur [3]float32
		have := false
		var acc []posSample
		for _, p := range raw {
			switch p.kind {
			case filmdec.PosKindAbsolute:
				// SEUL le direct-abs sert de baseline (raw=mauvais format, fallback=bruité).
				cur = p.vec
				have = true
			case filmdec.PosKindDelta8, filmdec.PosKindDeltaAxis:
				if !have {
					continue // pas de baseline encore : on saute
				}
				cur[0] += p.vec[0]
				cur[1] += p.vec[1]
				cur[2] += p.vec[2]
			default:
				continue // raw / absfb : ignorés pour l'accumulation
			}
			acc = append(acc, posSample{p.timeMs, p.kind, cur})
		}
		accBySlot[s] = acc
	}

	// slot -> xuid : vote timing de mort vs chunk_27.
	gt := loadGamertags()
	events, _ := analysis.ParseHighlightEvents(mustRead(cache+"/chunk_27.bin"), 0)
	type ev struct {
		xuid uint64
		t    int
	}
	var deaths []ev
	for _, e := range events {
		if e.EventType == analysis.EventTypeDeath {
			deaths = append(deaths, ev{e.XUID, e.TimeMS})
		}
	}
	slotXUID := map[uint32]uint64{}
	xuidSlot := map[uint64]uint32{}
	for s := uint32(512); s <= 519; s++ {
		ticks := deathBySlot[s]
		sort.Ints(ticks)
		votes := map[uint64]int{}
		prev := -100000
		for _, t := range ticks {
			if t-prev <= 2000 {
				prev = t
				continue
			}
			prev = t
			best, bd := uint64(0), 400
			for _, d := range deaths {
				dt := t - d.t
				if dt < 0 {
					dt = -dt
				}
				if dt < bd {
					bd, best = dt, d.xuid
				}
			}
			if best != 0 {
				votes[best]++
			}
		}
		var bx uint64
		bv := 0
		for x, c := range votes {
			if c > bv {
				bv, bx = c, x
			}
		}
		if bx != 0 {
			slotXUID[s] = bx
			xuidSlot[bx] = s
		}
	}
	fmt.Println("\n=== slot -> xuid (vote timing mort) ===")
	for s := uint32(512); s <= 519; s++ {
		if x, ok := slotXUID[s]; ok {
			fmt.Printf("  slot %d -> %s\n", s, nameOf(gt, x))
		} else {
			fmt.Printf("  slot %d -> (non identifié)\n", s)
		}
	}

	// sanity globale : amplitude + continuité par slot (absolus uniquement).
	fmt.Println("\n=== sanity positions absolues par slot (min/max/mediane pas) ===")
	for s := uint32(512); s <= 519; s++ {
		reportSanity(s, posBySlot[s], nameOf(gt, slotXUID[s]))
	}

	// kills narrés : position de la victime au sample accumulé le plus proche.
	type narr struct {
		victim string
		xuid   uint64
		tMs    int
		desc   string
	}
	narrs := []narr{
		{"JGtm", 2533274823110022, 329800, "BR75 JGtm->Akatsuki (victime Akatsuki en réalité)"},
		{"Akatsuki fire17", 2535444178793711, 329800, "BR75 JGtm->Akatsuki @329.8s (VICTIME)"},
		{"JGtm", 2533274823110022, 115500, "marteau IKE->JGtm @115.5s (VICTIME)"},
		{"JGtm", 2533274823110022, 292500, "marteau IKE->JGtm @292.5s (VICTIME)"},
	}
	// NB : le mapping slot->xuid (vote timing de mort) est le maillon FAIBLE (2/8 slots,
	// instable). La sonde sœur tmp_killfeed_weapons identifie slot519=JGtm. On reporte
	// donc la position de CHAQUE slot biped aux temps narrés (table position-par-slot),
	// pour que le mapping puisse être appliqué indépendamment.
	fmt.Println("\n=== TABLE POSITION-PAR-SLOT aux temps narrés (accumulé, direct-abs only) ===")
	for _, n := range narrs {
		fmt.Printf("  [%s] T=%.1fs\n", n.desc, float64(n.tMs)/1000)
		for s := uint32(512); s <= 519; s++ {
			p, _, ok := nearest(accBySlot[s], n.tMs)
			if !ok {
				fmt.Printf("    slot%d %-16s : (aucun sample)\n", s, nameOf(gt, slotXUID[s]))
				continue
			}
			fmt.Printf("    slot%d %-16s : (%7.2f,%7.2f,%7.2f) [%s, %+dms]\n",
				s, nameOf(gt, slotXUID[s]), p.vec[0], p.vec[1], p.vec[2], p.kind.String(), p.timeMs-n.tMs)
		}
	}
}

func nearest(ps []posSample, t int) (posSample, int, bool) {
	best := -1
	bd := 1 << 30
	for i, p := range ps {
		dt := p.timeMs - t
		if dt < 0 {
			dt = -dt
		}
		if dt < bd {
			bd, best = dt, i
		}
	}
	if best < 0 {
		return posSample{}, 0, false
	}
	return ps[best], bd, true
}

func reportSanity(s uint32, ps []posSample, name string) {
	var abs []posSample
	for _, p := range ps {
		if p.kind == filmdec.PosKindAbsolute { // QUANTIZED ONLY: borné [-100,100] par construction
			abs = append(abs, p)
		}
	}
	if len(abs) == 0 {
		fmt.Printf("  slot%d %-16s : 0 absolu\n", s, name)
		return
	}
	sort.Slice(abs, func(i, j int) bool { return abs[i].timeMs < abs[j].timeMs })
	var mn, mx [3]float32
	mn = abs[0].vec
	mx = abs[0].vec
	var steps []float64
	for i, p := range abs {
		for a := 0; a < 3; a++ {
			if p.vec[a] < mn[a] {
				mn[a] = p.vec[a]
			}
			if p.vec[a] > mx[a] {
				mx[a] = p.vec[a]
			}
		}
		if i > 0 {
			d := dist(abs[i-1].vec, p.vec)
			steps = append(steps, d)
		}
	}
	sort.Float64s(steps)
	med := 0.0
	if len(steps) > 0 {
		med = steps[len(steps)/2]
	}
	fmt.Printf("  slot%d %-16s : n=%d x[%.1f,%.1f] y[%.1f,%.1f] z[%.1f,%.1f] pasMédian=%.2f\n",
		s, name, len(abs), mn[0], mx[0], mn[1], mx[1], mn[2], mx[2], med)
}

func dist(a, b [3]float32) float64 {
	dx := float64(a[0] - b[0])
	dy := float64(a[1] - b[1])
	dz := float64(a[2] - b[2])
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func nameOf(gt map[uint64]string, x uint64) string {
	if x == 0 {
		return "(?)"
	}
	if g, ok := xuidGamertag[x]; ok {
		return g
	}
	if g, ok := gt[x]; ok && g != "" {
		return g
	}
	return fmt.Sprintf("xuid:%d", x)
}

func mustRead(p string) []byte { b, _ := os.ReadFile(p); return b }

func loadGamertags() map[uint64]string {
	gt := map[uint64]string{}
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		return gt
	}
	defer db.Close()
	var full string
	if db.QueryRow(`SELECT match_id FROM match_registry WHERE match_id LIKE '000d5950%' LIMIT 1`).Scan(&full) != nil {
		return gt
	}
	rows, err := db.Query(`SELECT DISTINCT xuid, gamertag FROM match_participants WHERE match_id=?`, full)
	if err != nil {
		return gt
	}
	defer rows.Close()
	for rows.Next() {
		var x, g sql.NullString
		rows.Scan(&x, &g)
		var xu uint64
		fmt.Sscanf(x.String, "%d", &xu)
		gt[xu] = g.String
	}
	return gt
}
