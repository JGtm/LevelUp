// tmp_killfeed_weapons — L5 : kill feed NOMME avec ARME (famille) sur 000d5950 (Fiesta).
//
// Croise :
//   - kill feed (chunk_27) : KILL@t (tueur) apparie a DEATH@t (victime) adjacent.
//   - timeline arme-tenue par slot biped : loadout initial (1er WST vu) + changements
//     (WST high-32 catalogue) au fil des deltas, avec leur ts.
//   - slot->xuid : vote par timing de mort (transitions Mort du dead-state d'un slot
//     appariees aux morts connues du chunk_27).
//
// Pour chaque kill : tueur_xuid -> slot biped -> arme tenue au ts du kill -> famille nommee.
//
// Usage : tmp_killfeed_weapons [maxChunk]
package main

import (
	"bytes"
	"compress/zlib"
	"database/sql"
	"encoding/binary"
	"fmt"
	"io"
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

// map pi->gamertag (film 000d5950, bit-vérifiée, cf HANDOFF).
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

// sets armes : high-32 (famille) + id64 complet -> nom.
var h32set = map[uint32]bool{}
var id64name = map[uint64]string{}

func buildWeaponSets() {
	for id, n := range analysis.WeaponIDToName {
		h32set[uint32(id>>32)] = true
		id64name[id] = n
	}
}

// scanFullID64 cherche un littéral d'arme COMPLET (high32|low32 catalogués, contigus)
// dans [lo,hi] bits. Renvoie le nom de famille du 1er trouvé.
func scanFullID64(d []byte, lo, hi int) (string, bool) {
	if lo < 0 {
		lo = 0
	}
	max := len(d)*8 - 64
	if hi > max {
		hi = max
	}
	for bp := lo; bp <= hi; bp++ {
		h := uint32(bitsAt(d, bp, 32))
		if !h32set[h] {
			continue
		}
		low := uint32(bitsAt(d, bp+32, 32))
		if n, ok := id64name[(uint64(h)<<32)|uint64(low)]; ok {
			return n, true
		}
	}
	return "", false
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

func famName(v uint32) (string, bool) {
	for id, n := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return n, true
		}
	}
	return "", false
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

type wpnEvt struct {
	timeMs int
	fam    string
}
type deathTick struct{ timeMs int }

func main() {
	maxChunk := 26
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &maxChunk)
	}
	filmdec.SetRecordStateParam(2)
	buildWeaponSets()
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	// 1) kill feed chunk_27 : KILL@t (tueur) apparie DEATH@t (victime) adjacent
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
	fmt.Printf("kill feed chunk_27 : %d kills appaires\n", len(feed))

	// 2) replay deltas : timeline arme par slot + ticks de mort par slot
	wpnBySlot := map[uint32][]wpnEvt{}
	deathBySlot := map[uint32][]deathTick{}
	for idx := 2; idx <= maxChunk; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			w := freshWorld(reg)
			br := filmdec.NewBitReader(fr.payload)
			recs, _ := filmdec.DecodeFrameRecords(br, w, calCfg)
			tms := int((fr.ts - t0Us) / 1000)
			for _, r := range recs {
				if !bipedSlots[r.Slot] {
					continue
				}
				// arme : littéral d'arme COMPLET (id64) dans la fenêtre du record biped
				// (un swap/pickup ré-émet l'id64 ; les deltas de mouvement n'en portent pas).
				comps := r.Trace.Comps
				if len(comps) > 0 {
					lo := comps[0].StartBit
					hi := comps[len(comps)-1].StartBit + 512
					if n, ok := scanFullID64(fr.payload, lo, hi); ok {
						wpnBySlot[r.Slot] = append(wpnBySlot[r.Slot], wpnEvt{tms, n})
					}
				}
				// mort : dead-state Mort==true sur record clean
				if r.DesyncAt == -1 && r.Trace.Dead != nil && r.Trace.Dead.Mort {
					deathBySlot[r.Slot] = append(deathBySlot[r.Slot], deathTick{tms})
				}
			}
		}
	}
	for s := range wpnBySlot {
		sort.Slice(wpnBySlot[s], func(i, j int) bool { return wpnBySlot[s][i].timeMs < wpnBySlot[s][j].timeMs })
	}
	fmt.Printf("timeline arme (records biped) : ")
	for s := uint32(512); s <= 519; s++ {
		fmt.Printf("slot%d=%d evts ", s, len(wpnBySlot[s]))
	}
	fmt.Println()

	// 2b) SCAN BRUT : tous les littéraux d'arme complets (id64) du flux type-0, avec ts.
	// = sortie décodeur pure (les ~247 swaps/pickups). Dédup consécutif même arme.
	type litEvt struct {
		timeMs int
		fam    string
	}
	var lits []litEvt
	famCount := map[string]int{}
	for idx := 2; idx <= maxChunk; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			tms := int((fr.ts - t0Us) / 1000)
			max := len(fr.payload)*8 - 64
			for bp := 0; bp <= max; bp++ {
				h := uint32(bitsAt(fr.payload, bp, 32))
				if !h32set[h] {
					continue
				}
				low := uint32(bitsAt(fr.payload, bp+32, 32))
				if n, ok := id64name[(uint64(h)<<32)|uint64(low)]; ok {
					lits = append(lits, litEvt{tms, n})
					famCount[n]++
					bp += 63 // évite de re-scanner le même littéral
				}
			}
		}
	}
	fmt.Printf("\n=== SCAN BRUT type-0 : %d littéraux d'arme complets (id64) décodés du film ===\n", len(lits))
	sort.Slice(lits, func(i, j int) bool { return lits[i].timeMs < lits[j].timeMs })
	fmt.Println("  -- timeline fenetre 285-380s (autour des kills narres marteau/sniper) --")
	for _, l := range lits {
		if l.timeMs < 285000 || l.timeMs > 380000 {
			continue
		}
		fmt.Printf("    t=%6.1fs  %s\n", float64(l.timeMs)/1000, l.fam)
	}
	fmt.Println("  -- familles distinctes trouvées --")
	type fc struct {
		n string
		c int
	}
	var fcs []fc
	for n, c := range famCount {
		fcs = append(fcs, fc{n, c})
	}
	sort.Slice(fcs, func(i, j int) bool { return fcs[i].c > fcs[j].c })
	for _, f := range fcs {
		fmt.Printf("    %-26s x%d\n", f.n, f.c)
	}

	// 3) slot->xuid : vote par timing de mort (transitions >2s) vs morts connues
	slotXUID := map[uint32]uint64{}
	for s := uint32(512); s <= 519; s++ {
		ticks := deathBySlot[s]
		sort.Slice(ticks, func(i, j int) bool { return ticks[i].timeMs < ticks[j].timeMs })
		votes := map[uint64]int{}
		prev := -100000
		for _, t := range ticks {
			if t.timeMs-prev <= 2000 {
				prev = t.timeMs
				continue
			}
			prev = t.timeMs
			best, bd := uint64(0), 400
			for _, d := range deaths {
				dt := t.timeMs - d.t
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
		}
	}
	xuidSlot := map[uint64]uint32{}
	fmt.Println("\n=== slot -> xuid (vote timing de mort) ===")
	for s := uint32(512); s <= 519; s++ {
		if x, ok := slotXUID[s]; ok {
			xuidSlot[x] = s
			fmt.Printf("  slot %d -> %s (xuid %d)\n", s, nameOf(gt, x), x)
		} else {
			fmt.Printf("  slot %d -> (non identifie)\n", s)
		}
	}

	// 4) kill feed + arme tenue du tueur au ts du kill
	fmt.Println("\n=== KILL FEED + ARME (tueur -> arme famille -> victime) ===")
	named, total := 0, 0
	for _, r := range feed {
		total++
		killerSlot, ok := xuidSlot[r.killer]
		wpn := "(tueur slot non identifie)"
		if ok {
			wpn = heldAt(wpnBySlot[killerSlot], r.t)
			if wpn != "" {
				named++
			} else {
				wpn = "(pas d'arme captee avant ce kill)"
			}
		}
		fmt.Printf("  t=%6.1fs  %-16s -> %-16s | %s\n",
			float64(r.t)/1000, nameOf(gt, r.killer), nameOf(gt, r.victim), wpn)
	}
	fmt.Printf("\n>>> %d/%d kills avec arme-famille du tueur attribuee\n", named, total)
}

// heldAt : derniere arme captee a <= t pour ce slot (loadout/swap propage).
func heldAt(evts []wpnEvt, t int) string {
	res := ""
	for _, e := range evts {
		if e.timeMs <= t+200 {
			res = e.fam
		} else {
			break
		}
	}
	return res
}

func nameOf(gt map[uint64]string, x uint64) string {
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
		var x sql.NullString
		var g sql.NullString
		rows.Scan(&x, &g)
		var xu uint64
		fmt.Sscanf(x.String, "%d", &xu)
		gt[xu] = g.String
	}
	return gt
}
