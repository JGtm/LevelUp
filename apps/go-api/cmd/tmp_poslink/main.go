// tmp_poslink — PARTIE 3 : MATCH position-victime (P1) <-> position-impact (P2)
// pour attribuer l'ARME (famille) par kill sur 000d5950.
//
// Pour chaque kill du feed (chunk_27 : tueur/victime/temps T) :
//  1. POSITION VICTIME a T (P1, tmp_victimpos) : accumulation direct-abs+deltas par
//     slot biped, victime->slot via vote timing de mort.
//  2. RECORDS de degat (P2, tmp_impactpos) : payload[0]==0xd2, deser FUN_14080c1f8
//     porte -> (ts, famille @+0x14, posGate, vec3 impact).
//  3. MATCH : parmi les records dans [T-win, T+win], celui dont la pos d'impact est
//     la PLUS PROCHE de la pos victime -> coup letal -> son arme (famille).
//
// Mesure honnete : combien de kills ont (a) une pos victime, (b) un record dans la
// fenetre, (c) un record avec vec3 impact decodable -> donc combien sont matchables
// PAR POSITION. Puis valide la narration (BR75 JGtm->Akatsuki ; marteau IKE->JGtm).
//
// Decode = Go PUR (CGO_ENABLED=1 seulement pour duckdb gamertags). Usage : tmp_poslink [win_ms]
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

// ── P2 : deser record de degat (porte de tmp_impactpos) ────────────────────
var h32name = map[uint32]string{}

func buildWeaponNames() {
	for id, n := range analysis.WeaponIDToName {
		h32name[uint32(id>>32)] = n
	}
}

const impAxisHigh = uint(0xf)
const impRange = float32(10.0)

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

type pkt struct {
	typ     uint16
	ts      uint64
	payload []byte
}

func listPackets(d []byte) []pkt {
	var out []pkt
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		size := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if size <= 0 || off+16+size > len(d) {
			break
		}
		out = append(out, pkt{typ, ts, d[off+16 : off+16+size]})
		off += 16 + size
	}
	return out
}

func tsToMs(ts uint64) int { return int((int64(ts) - int64(t0Us)) / 1000) }

// damRec : record de degat decode (famille + pos impact si decodable).
type damRec struct {
	tms     int
	fam     string
	posGate bool
	vec     [3]float32
	vecOK   bool
}

// decodeDamage porte FUN_14080c1f8 (param_5=1) au startBit=36 prouve, jusqu'a la
// position d'impact (param_3+0x27c via FUN_140c9e4d8). Copie fidele de tmp_impactpos.
func decodeDamage(payload []byte) damRec {
	br := filmdec.NewBitReader(payload)
	br.Skip(36)
	var r damRec
	total := len(payload) * 8
	chk := func() bool { return br.BitPos() <= total }

	// slot/cause +0x08 : R(1); si 0 -> R(2)
	if !br.ReadBit() {
		br.ReadBits(2)
	}
	br.ReadBits(5) // global-id R5
	fam := uint32(br.ReadBits(32))
	if n, ok := h32name[fam]; ok {
		r.fam = n
	} else {
		r.fam = fmt.Sprintf("?%08x", fam)
	}
	br.ReadBits(32) // LOW 0x42c9679f
	br.ReadBit()    // param_3[0x1d]
	br.ReadBit()    // param_3[2]
	br.ReadBit()    // hasExtra
	// header hit-sections
	typ, count := readHitHeader(br)
	if !chk() {
		return r
	}
	if count > 0 && count < 64 {
		for i := uint32(0); i < count; i++ {
			br.ReadBits(2)
			br.ReadBits(1)
		}
	}
	if typ != 0 {
		return r // boucle +0x110 value-gated -> desync
	}
	// POSITION D'IMPACT : FUN_140c9e4d8(+0x27c)
	r.posGate, r.vec, r.vecOK = readImpactDescriptor(br)
	return r
}

func readHitHeader(br *filmdec.BitReader) (typ, count uint32) {
	if br.ReadBit() {
		return 0, 0
	}
	if br.ReadBit() {
		typ = 1
	} else {
		typ = uint32(br.ReadBits(4))
	}
	if br.ReadBit() {
		if br.ReadBit() {
			count = uint32(br.ReadBits(4))
		} else {
			count = 1
		}
	}
	return typ, count
}

func readImpactDescriptor(br *filmdec.BitReader) (gate bool, vec [3]float32, vecOK bool) {
	gate = br.ReadBit()
	if !gate {
		return false, [3]float32{}, false
	}
	mode := byte(br.ReadBits(2))
	if mode == 1 {
		if br.ReadBit() {
			br.ReadBits(6)
		}
	}
	b1 := br.ReadBit()
	if !b1 {
		return gate, [3]float32{}, false // largeur FUN_1406d84b4 inconnue
	}
	// vec3 FUN_140c9e738 : gate 1-bit ; si 0 -> packed widthHigh bits (log-quantise ±10)
	if br.ReadBit() {
		return gate, [3]float32{}, false // valeur par defaut (origine)
	}
	raw := br.ReadBits(impAxisHigh)
	q := float32(raw) / float32((uint64(1)<<impAxisHigh)-1)
	v := (q - 0.5) * 2 * impRange
	return gate, [3]float32{v, 0, 0}, true
}

// ── P1 : positions victime (porte de tmp_victimpos) ────────────────────────
type posSample struct {
	timeMs int
	kind   filmdec.PosKind
	vec    [3]float32
}

func listFrames(d []byte) []pkt {
	var out []pkt
	for _, p := range listPackets(d) {
		if p.typ == 0 {
			out = append(out, p)
		}
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
	win := 1500
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &win)
	}
	buildWeaponNames()
	filmdec.SetRecordStateParam(2)
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	// 1) RECORDS de degat (P2) sur tous les chunks.
	var recs []damRec
	var nGate, nVec int
	for n := 0; n <= 27; n++ {
		for _, p := range listPackets(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))) {
			if p.typ != 0 || len(p.payload) == 0 || p.payload[0] != 0xd2 {
				continue
			}
			r := decodeDamage(p.payload)
			r.tms = tsToMs(p.ts)
			if r.posGate {
				nGate++
			}
			if r.vecOK {
				nVec++
			}
			recs = append(recs, r)
		}
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].tms < recs[j].tms })
	fmt.Printf("=== P2 records de degat : %d ; posGate set=%d ; vec3 impact decode=%d ===\n", len(recs), nGate, nVec)

	// 2) POSITIONS VICTIME (P1) : accumulation par slot + mort par slot.
	var frameSamples []filmdec.PositionSample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { frameSamples = append(frameSamples, s) })
	posBySlot := map[uint32][]posSample{}
	deathBySlot := map[uint32][]int{}
	for idx := 2; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			w := freshWorld(reg)
			br := filmdec.NewBitReader(fr.payload)
			frameSamples = frameSamples[:0]
			frecs, _ := filmdec.DecodeFrameRecords(br, w, calCfg)
			tms := tsToMs(fr.ts)
			byBit := map[int]filmdec.PositionSample{}
			for _, s := range frameSamples {
				byBit[s.BitPos] = s
			}
			for _, r := range frecs {
				if !bipedSlots[r.Slot] || len(r.Trace.Comps) == 0 {
					continue
				}
				c0 := r.Trace.Comps[0]
				if c0.Name != "object-position-dynamic-precision-component" {
					continue
				}
				if s, ok := byBit[c0.StartBit]; ok {
					posBySlot[r.Slot] = append(posBySlot[r.Slot], posSample{tms, s.Kind, s.Vec})
				}
				if r.DesyncAt == -1 && r.Trace.Dead != nil && r.Trace.Dead.Mort {
					deathBySlot[r.Slot] = append(deathBySlot[r.Slot], tms)
				}
			}
		}
	}
	filmdec.SetPositionCaptureHook(nil)

	// accumulation par slot (baseline direct-abs + deltas).
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
				cur = p.vec
				have = true
			case filmdec.PosKindDelta8, filmdec.PosKindDeltaAxis:
				if !have {
					continue
				}
				cur[0] += p.vec[0]
				cur[1] += p.vec[1]
				cur[2] += p.vec[2]
			default:
				continue
			}
			acc = append(acc, posSample{p.timeMs, p.kind, cur})
		}
		accBySlot[s] = acc
	}

	// 3) kill feed + slot->xuid (vote timing mort).
	gt := loadGamertags()
	events, _ := analysis.ParseHighlightEvents(mustRead(cache+"/chunk_27.bin"), 0)
	type ev struct {
		xuid uint64
		t    int
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

	type kfRow struct {
		killer, victim uint64
		t              int
	}
	var feed []kfRow
	usedD := make([]bool, len(deaths))
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

	// slot->xuid par vote timing mort
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

	// 4) MATCH par kill.
	fmt.Printf("\n=== MATCH par kill (feed=%d kills ; fenetre=±%dms) ===\n", len(feed), win)
	var nVictimPos, nRecInWin, nRecVecInWin, nMatched int
	for _, k := range feed {
		// pos victime a T via slot de la victime
		vslot, hasSlot := xuidSlot[k.victim]
		var vpos [3]float32
		hasPos := false
		if hasSlot {
			if p, _, ok := nearest(accBySlot[vslot], k.t); ok {
				vpos, hasPos = p.vec, true
			}
		}
		if hasPos {
			nVictimPos++
		}
		// records dans la fenetre
		var winRecs, winVec []damRec
		for _, r := range recs {
			if r.tms < k.t-win || r.tms > k.t+win {
				continue
			}
			winRecs = append(winRecs, r)
			if r.vecOK {
				winVec = append(winVec, r)
			}
		}
		if len(winRecs) > 0 {
			nRecInWin++
		}
		if len(winVec) > 0 {
			nRecVecInWin++
		}
		// match position : seulement possible si pos victime ET >=1 record avec vec3
		if hasPos && len(winVec) > 0 {
			nMatched++
			_ = vpos
		}
	}
	fmt.Printf("  kills avec pos victime (P1)          : %d/%d\n", nVictimPos, len(feed))
	fmt.Printf("  kills avec >=1 record dans fenetre   : %d/%d\n", nRecInWin, len(feed))
	fmt.Printf("  kills avec >=1 record a vec3 impact  : %d/%d\n", nRecVecInWin, len(feed))
	fmt.Printf("  kills MATCHABLES PAR POSITION (P1∩P2): %d/%d\n", nMatched, len(feed))

	// 4-bis) FALLBACK temporel (sans position) : record famille unique dans la fenetre.
	fmt.Printf("\n=== ECHANTILLON table par kill (fallback temporel, sans position) ===\n")
	shown := 0
	var nUniqueFam int
	for _, k := range feed {
		fams := map[string]int{}
		for _, r := range recs {
			if r.tms >= k.t-win && r.tms <= k.t+win {
				fams[r.fam]++
			}
		}
		if len(fams) == 1 {
			nUniqueFam++
		}
		if shown < 20 {
			famStr := "(aucun record)"
			if len(fams) > 0 {
				famStr = fmt.Sprintf("%d familles dans ±%dms", len(fams), win)
				if len(fams) == 1 {
					for f := range fams {
						famStr = f + " (unique)"
					}
				}
			}
			vslot, _ := xuidSlot[k.victim]
			fmt.Printf("  %-16s -> %-16s @%6.1fs vslot=%d : %s\n",
				nameOf(gt, k.killer), nameOf(gt, k.victim), float64(k.t)/1000, vslot, famStr)
			shown++
		}
	}
	fmt.Printf("  ... kills avec famille UNIQUE dans fenetre (proxy temporel) : %d/%d\n", nUniqueFam, len(feed))

	// 5) NARRATION.
	fmt.Println("\n=== VALIDATION NARRATION ===")
	checkNarr("BR75 JGtm->Akatsuki", 329800, "BR75", recs, win)
	checkNarr("BR75 JGtm->Akatsuki(1)", 112900, "BR75", recs, win)
	checkNarr("marteau IKE->JGtm", 115500, "Hammer", recs, win)
	checkNarr("marteau IKE->JGtm(2)", 292500, "Hammer", recs, win)
	checkNarr("marteau IKE->JGtm(3)", 355700, "Hammer", recs, win)
	checkNarr("marteau IKE->JGtm(4)", 375100, "Hammer", recs, win)
}

func checkNarr(label string, t int, expect string, recs []damRec, win int) {
	var got []string
	hit := false
	for _, r := range recs {
		if r.tms >= t-win && r.tms <= t+win {
			got = append(got, fmt.Sprintf("%s@%.1fs", r.fam, float64(r.tms)/1000))
		}
	}
	for _, g := range got {
		if len(expect) > 0 && containsCI(g, expect) {
			hit = true
		}
	}
	status := "KO"
	if hit {
		status = "OK"
	}
	if len(got) == 0 {
		fmt.Printf("  [%s] @%.1fs attendu=%s : AUCUN record en ±%dms -> %s\n", label, float64(t)/1000, expect, win, status)
		return
	}
	fmt.Printf("  [%s] @%.1fs attendu=%s : records=%v -> %s\n", label, float64(t)/1000, expect, got, status)
}

func containsCI(s, sub string) bool {
	return len(s) >= len(sub) && bytes.Contains([]byte(toLow(s)), []byte(toLow(sub)))
}
func toLow(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
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

func _dist(a, b [3]float32) float64 {
	dx := float64(a[0] - b[0])
	dy := float64(a[1] - b[1])
	dz := float64(a[2] - b[2])
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
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
