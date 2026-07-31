// tmp_eventcross — THROWAWAY : CROISE les events (player_index+arme+temps) du film
// avec le KILL FEED chunk_27 -> SOURCE PAR KILL.
//
// Sources :
//   - KILL FEED chunk_27 : events type-3 nommés (xuid, typeHint 50=kill/20=death, timeMS,
//     duo=b36, team=b37). KILL@t apparié à DEATH@t adverse adjacent => (tueur, victime, T).
//     Le tueur a un (duo,team) => slot via bijection per-match.
//   - EVENTS d'action (scan bit-packé chunks 00..27) :
//     GRENADE : marqueur 0x4c0c00(24b) ; gid@+24(32b) ; player_index@+24+32+47(5b).
//     MELEE   : marqueur 0x534/0x535(11b) ; anchor=bp+3 ; type@+76(8b) ; weapon high32@+86/+88/+101|103 ;
//     player_index@anchor+23(5b).
//     FIRE    : weapon high32(famille cat)@bp + low32==0x42c9679f@bp+32 ; player_index@bp-4(5b).
//     ts via tsAtBit (header paquet 16o, t0Us=4537898226).
//
// Bijection player_index <-> slot tueur :
//
//	player_index (events) == index roster 0..7 == pi-> gamertag connu.
//	chunk_27 donne xuid tueur -> gamertag -> pi. On verifie que (duo,team) du chunk_27 forme
//	bien une bijection vers pi, et on confirme via les grenades/kills grenade.
//
// Attribution : par kill (tueur pi, T) -> event dont player_index==pi et tms<=T le plus proche
//
//	(fenetre courte) -> arme = source.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"unicode/utf16"

	"levelup/go-api/internal/analysis"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)

const (
	minXUID = uint64(2e15)
	maxXUID = uint64(3e15)
)

var endMarker = []byte{0x00, 0x00, 0x2e, 0xe0}

// pi -> gamertag + xuid (bit-vérifié, cf HANDOFF)
var piName = map[int]string{0: "whiteknight2519", 1: "JAVIERLOLITO540", 2: "JGtm", 3: "LORD PEINX13", 4: "IKE ILYA", 5: "Akatsuki fire17", 6: "aldusbroncus", 7: "VitaminA1688"}
var xuidToPi = map[uint64]int{
	2535467794760703: 0, // whiteknight2519
	2535437947245250: 1, // JAVIERLOLITO540
	2533274823110022: 2, // JGtm
	2533274980284321: 3, // LORD PEINX13
	2533274815845110: 4, // IKE ILYA
	2535444178793711: 5, // Akatsuki fire17
	2533274882097883: 6, // aldusbroncus
	2533274826120416: 7, // VitaminA1688
}

var h32 = map[uint32]string{}
var grenades = map[uint32]string{0xB0171062: "Frag Grenade", 0xC0E34C44: "Plasma Grenade", 0x3B2567D4: "Shock Grenade", 0x9212E428: "Spike Grenade"}

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
func buildCat() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
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
func tsAtBit(d []byte, bp int) (int, bool) {
	pos := bp >> 3
	off := 0
	for off+16 <= len(d) {
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if pos >= off+16 && pos < off+16+sz {
			return int((ts - t0Us) / 1000), true
		}
		off += 16 + sz
	}
	return -1, false
}

// ---- EVENTS d'action ----
type actEvt struct {
	tms  int
	kind string // grenade/melee/fire
	wpn  string
	pidx int
	hit  bool // melee: HIT(0x534) ; fire: nibble post-id ; grenade: n/a
}

func decodeActions() []actEvt {
	var evs []actEvt
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		total := len(d) * 8
		// GRENADE
		for bp := 0; bp+110 < total; bp++ {
			if bitsAt(d, bp, 24) != 0x4c0c00 {
				continue
			}
			gid := uint32(bitsAt(d, bp+24, 32))
			gname, ok := grenades[gid]
			if !ok {
				continue
			}
			pidx := int(bitsAt(d, bp+24+32+47, 5))
			tms, okt := tsAtBit(d, bp)
			if !okt {
				continue
			}
			evs = append(evs, actEvt{tms, "grenade", gname, pidx, true})
		}
		// MELEE
		for bp := 0; bp+120 < total; bp++ {
			m := bitsAt(d, bp, 11)
			if m != 0x534 && m != 0x535 {
				continue
			}
			anchor := bp + 3
			typ := uint8(bitsAt(d, anchor+76, 8))
			var woff int
			switch typ {
			case 0x47:
				woff = anchor + 86
			case 0x42:
				woff = anchor + 88
			case 0x60:
				woff = anchor + 101
			default:
				continue
			}
			hi := uint32(bitsAt(d, woff, 32))
			name, ok := h32[hi]
			if !ok {
				if typ == 0x60 {
					hi = uint32(bitsAt(d, anchor+103, 32))
					name, ok = h32[hi]
				}
				if !ok {
					continue
				}
			}
			pidx := int(bitsAt(d, anchor+23, 5))
			tms, okt := tsAtBit(d, bp)
			if !okt {
				continue
			}
			evs = append(evs, actEvt{tms, "melee", name, pidx, m == 0x534})
		}
		// FIRE
		for bp := 4; bp+96 < total; bp++ {
			hi := uint32(bitsAt(d, bp, 32))
			name, ok := h32[hi]
			if !ok {
				continue
			}
			if uint32(bitsAt(d, bp+32, 32)) != 0x42c9679f {
				continue
			}
			pidx := int(bitsAt(d, bp-4, 5))
			tms, okt := tsAtBit(d, bp)
			if !okt {
				continue
			}
			post := uint32(bitsAt(d, bp+64, 32))
			hit := (post>>28)&0xF == 0x3 // nibble haut 0x3 (vs 0x7) candidat hit
			evs = append(evs, actEvt{tms, "fire", name, pidx, hit})
		}
	}
	sort.Slice(evs, func(a, b int) bool { return evs[a].tms < evs[b].tms })
	return evs
}

// ---- KILL FEED chunk_27 ----
type kfEvt struct {
	xuid     uint64
	gt       string
	typeHint int
	timeMS   int
	team     int
	duo      int
}

func scanKF(data []byte) []kfEvt {
	totalBits := len(data) * 8
	var out []kfEvt
	seen := map[int]bool{}
	for ms := 8; ms <= totalBits-8; ms++ {
		if readByteAtBit(data, ms) != 0xc0 {
			continue
		}
		xe := ms - 8
		if xe < 64 {
			continue
		}
		pfx := readByteAtBit(data, xe)
		if pfx != 0x2d && pfx != 0x25 {
			continue
		}
		xs := xe - 64
		if seen[xs] {
			continue
		}
		xuid := readU64LE(data, xs)
		if xuid <= minXUID || xuid >= maxXUID {
			continue
		}
		e, ok := parseKF(data, xs, xuid)
		if !ok {
			continue
		}
		out = append(out, e)
		seen[xs] = true
	}
	return out
}

const eventWindowBits = 20000
const eventDataBytes = 60

func parseKF(data []byte, xs int, xuid uint64) (kfEvt, bool) {
	total := len(data) * 8
	wend := xs + eventWindowBits
	if wend > total {
		wend = total
	}
	from := xs
	for {
		end := findMarker(data, from, wend, endMarker)
		if end < 0 {
			return kfEvt{}, false
		}
		st := end - eventDataBytes*8
		if st < xs {
			from = end + 1
			continue
		}
		eb := readBytes(data, st, eventDataBytes)
		if eb == nil {
			from = end + 1
			continue
		}
		th := int(eb[47])
		valid := th == 50 || th == 20 || th == 10 || th == 51 || th == 52 ||
			th == 100 || th == 101 || th == 150 || th == 200 || th == 205 ||
			th == 210 || th == 220 || th == 225 || th == 230 || th == 235 ||
			th == 240 || th == 245 || th == 250
		if !valid {
			from = end + 1
			continue
		}
		return kfEvt{
			xuid:     xuid,
			gt:       utf16le(eb[0:32]),
			typeHint: th,
			timeMS:   int(binary.BigEndian.Uint32(eb[48:52])),
			duo:      int(eb[36]),
			team:     int(eb[37]),
		}, true
	}
}

// ---- helpers chunk_27 ----
func utf16le(b []byte) string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(b[i*2 : i*2+2])
	}
	for i, c := range u {
		if c == 0 {
			u = u[:i]
			break
		}
	}
	return string(utf16.Decode(u))
}
func readByteAtBit(d []byte, bit int) byte {
	if bit < 0 || bit+8 > len(d)*8 {
		return 0
	}
	bi := bit / 8
	off := uint(bit % 8)
	if off == 0 {
		return d[bi]
	}
	return d[bi]<<off | d[bi+1]>>(8-off)
}
func readBytes(d []byte, bit, n int) []byte {
	if bit < 0 || bit+n*8 > len(d)*8 {
		return nil
	}
	o := make([]byte, n)
	for i := 0; i < n; i++ {
		o[i] = readByteAtBit(d, bit+i*8)
	}
	return o
}
func readU64LE(d []byte, bit int) uint64 {
	b := readBytes(d, bit, 8)
	if b == nil {
		return 0
	}
	var x uint64
	for i := 0; i < 8; i++ {
		x |= uint64(b[i]) << (uint(i) * 8)
	}
	return x
}
func findMarker(d []byte, s, e int, pat []byte) int {
	if s < 0 {
		s = 0
	}
	tb := len(d) * 8
	if e > tb {
		e = tb
	}
	pb := len(pat) * 8
	for bit := s; bit <= e-pb; bit++ {
		m := true
		for i := 0; i < len(pat); i++ {
			if readByteAtBit(d, bit+i*8) != pat[i] {
				m = false
				break
			}
		}
		if m {
			return bit
		}
	}
	return -1
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

type killRow struct {
	killerPi, victimPi int
	killerXUID         uint64
	t                  int
	duo, team          int
}

func main() {
	buildCat()

	// 1) KILL FEED : apparier KILL@t <-> DEATH@t adverse adjacent
	kf := scanKF(inflate(cache + "/chunk_27.bin"))
	var kills, deaths []kfEvt
	for _, e := range kf {
		switch e.typeHint {
		case 50:
			kills = append(kills, e)
		case 20:
			deaths = append(deaths, e)
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].timeMS < kills[j].timeMS })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].timeMS < deaths[j].timeMS })

	usedD := make([]bool, len(deaths))
	var feed []killRow
	for _, k := range kills {
		best, bd := -1, 400
		for i, d := range deaths {
			if usedD[i] || d.xuid == k.xuid {
				continue
			}
			dt := abs(k.timeMS - d.timeMS)
			if dt < bd {
				bd, best = dt, i
			}
		}
		if best < 0 {
			continue
		}
		usedD[best] = true
		kp, kok := xuidToPi[k.xuid]
		vp, vok := xuidToPi[deaths[best].xuid]
		if !kok {
			kp = -1
		}
		if !vok {
			vp = -1
		}
		feed = append(feed, killRow{kp, vp, k.xuid, k.timeMS, k.duo, k.team})
	}

	// 2) BIJECTION player_index <-> slot tueur (duo,team).
	//    On verifie que xuid->pi est coherent avec (duo,team) per-match : chaque pi a
	//    un (duo,team) constant. On le tabule.
	type dt struct{ duo, team int }
	piDT := map[int]map[dt]int{}
	for _, e := range kf {
		p, ok := xuidToPi[e.xuid]
		if !ok {
			continue
		}
		if piDT[p] == nil {
			piDT[p] = map[dt]int{}
		}
		piDT[p][dt{e.duo, e.team}]++
	}
	fmt.Println("=== BIJECTION player_index <-> slot tueur (chunk_27 duo=b36 + team=b37) ===")
	dtToPi := map[dt]int{}
	bijOK := true
	for p := 0; p < 8; p++ {
		m := piDT[p]
		// (duo,team) dominant pour ce pi
		var bestDT dt
		bestC := -1
		for k, c := range m {
			if c > bestC {
				bestC, bestDT = c, k
			}
		}
		if prev, seen := dtToPi[bestDT]; seen {
			bijOK = false
			fmt.Printf("  pi%d(%s): (duo=%d,team=%d) COLLISION avec pi%d\n", p, piName[p], bestDT.duo, bestDT.team, prev)
		} else {
			dtToPi[bestDT] = p
		}
		fmt.Printf("  pi%d(%-16s) -> (duo=%d, team=%d)  [n=%d]\n", p, piName[p], bestDT.duo, bestDT.team, bestC)
	}
	fmt.Printf("  bijection (duo,team)->pi %s (8 combos distincts = %v)\n", map[bool]string{true: "VALIDE", false: "INCOMPLETE"}[bijOK && len(dtToPi) == 8], len(dtToPi) == 8)

	// 2b) Confirmation par les grenades : pi des events grenade doit recouper les
	//     joueurs ayant des kills (et duo/team coherents). On affiche la distribution.
	actions := decodeActions()
	var grenadePi, fireP, meleeP = map[int]int{}, map[int]int{}, map[int]int{}
	for _, a := range actions {
		switch a.kind {
		case "grenade":
			grenadePi[a.pidx]++
		case "fire":
			fireP[a.pidx]++
		case "melee":
			meleeP[a.pidx]++
		}
	}
	fmt.Println("\n=== Confirmation : distribution player_index par type d'event ===")
	for i := 0; i < 8; i++ {
		fmt.Printf("  pi%d(%-16s) grenade=%d fire=%d melee=%d\n", i, piName[i], grenadePi[i], fireP[i], meleeP[i])
	}

	// 3) ATTRIBUTION par kill : event dont player_index==killerPi et tms<=T le plus proche.
	//    fenetre <= 2500ms avant T (preferer hit). melee priorise type marteau/epee.
	fmt.Printf("\n=== ATTRIBUTION SOURCE PAR KILL (%d kills appairés) ===\n", len(feed))
	attributed := 0
	var melHammerIKE, brJGtm int
	type kres struct {
		t              int
		killer, victim string
		src            string
		kind           string
		dt             int
	}
	var results []kres
	for _, k := range feed {
		if k.killerPi < 0 {
			results = append(results, kres{k.t, fmt.Sprintf("xuid:%d", k.killerXUID), piName[k.victimPi], "(tueur non roster)", "", 0})
			continue
		}
		// chercher event de ce pi le plus proche dans [-2500, +400] de T
		best := -1
		bestDt := 1 << 30
		for i, a := range actions {
			if a.pidx != k.killerPi {
				continue
			}
			d := k.t - a.tms
			if d < -400 || d > 2500 {
				continue
			}
			ad := abs(d)
			// preferer hit
			score := ad
			if !a.hit {
				score += 600
			}
			if score < bestDt {
				bestDt = score
				best = i
			}
		}
		if best < 0 {
			results = append(results, kres{k.t, piName[k.killerPi], piName[k.victimPi], "(aucun event auteur près de T)", "", 0})
			continue
		}
		a := actions[best]
		attributed++
		results = append(results, kres{k.t, piName[k.killerPi], piName[k.victimPi], a.wpn, a.kind, k.t - a.tms})
		// validation narration
		if a.kind == "melee" && k.killerPi == 4 && k.victimPi == 2 {
			melHammerIKE++
		}
		if a.kind == "fire" && k.killerPi == 2 && k.victimPi == 5 {
			brJGtm++
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].t < results[j].t })
	for _, r := range results {
		fmt.Printf("  t=%6.1fs  %-16s -> %-16s | %-26s [%s dt=%dms]\n",
			float64(r.t)/1000, r.killer, r.victim, r.src, r.kind, r.dt)
	}
	fmt.Printf("\n>>> %d/%d kills avec une source (event auteur) attribuée\n", attributed, len(feed))

	// 4) VALIDATION NARRATION ciblée
	fmt.Println("\n=== VALIDATION NARRATION ===")
	checkNarration(feed, actions, 4, 2, "melee", []int{115500, 292500, 355700, 375100}, "Marteau IKE->JGtm")
	checkNarration(feed, actions, 2, 5, "fire", []int{112900, 329800}, "BR75 JGtm->Akatsuki")

	// 5) DIAGNOSTIC : tous les events melee de pi4 (IKE) et fire BR75 de pi2 (JGtm)
	fmt.Println("\n=== DIAG : tous les events MELEE de pi4 (IKE ILYA) ===")
	for _, a := range actions {
		if a.kind == "melee" && a.pidx == 4 {
			fmt.Printf("   t=%7.1fs  %-22s hit=%v\n", float64(a.tms)/1000, a.wpn, a.hit)
		}
	}
	fmt.Println("=== DIAG : tous les events FIRE de pi2 (JGtm) (famille) ===")
	famc := map[string]int{}
	for _, a := range actions {
		if a.kind == "fire" && a.pidx == 2 {
			famc[a.wpn]++
		}
	}
	for f, c := range famc {
		fmt.Printf("   %-24s x%d\n", f, c)
	}
	fmt.Println("=== DIAG : BR75 fire events de pi2 (JGtm) timés ===")
	for _, a := range actions {
		if a.kind == "fire" && a.pidx == 2 && a.wpn == "BR75" {
			fmt.Printf("   t=%7.1fs\n", float64(a.tms)/1000)
		}
	}
	// densité temporelle des events vs kills (les actions couvrent-elles tout le match ?)
	fmt.Println("=== DIAG : couverture temporelle des actions (min/max tms par type) ===")
	for _, kind := range []string{"grenade", "melee", "fire"} {
		mn, mx, n := 1<<30, -1, 0
		for _, a := range actions {
			if a.kind != kind {
				continue
			}
			n++
			if a.tms < mn {
				mn = a.tms
			}
			if a.tms > mx {
				mx = a.tms
			}
		}
		fmt.Printf("   %-8s n=%d  [%.1fs .. %.1fs]\n", kind, n, float64(mn)/1000, float64(mx)/1000)
	}
}

// checkNarration : pour chaque temps narré cible, trouve le kill (killerPi->victimPi) le plus
// proche et l'event du bon type/pi le plus proche.
func checkNarration(feed []killRow, actions []actEvt, killerPi, victimPi int, kind string, times []int, label string) {
	fmt.Printf("-- %s (pi%d->pi%d, type %s) --\n", label, killerPi, victimPi, kind)
	for _, T := range times {
		// kill le plus proche de T pour ce couple
		bk := -1
		bkdt := 1 << 30
		for i, k := range feed {
			if k.killerPi != killerPi || k.victimPi != victimPi {
				continue
			}
			if abs(k.t-T) < bkdt {
				bkdt = abs(k.t - T)
				bk = i
			}
		}
		// event du bon kind/pi le plus proche de T
		be := -1
		bedt := 1 << 30
		for i, a := range actions {
			if a.pidx != killerPi || a.kind != kind {
				continue
			}
			if abs(a.tms-T) < bedt {
				bedt = abs(a.tms - T)
				be = i
			}
		}
		ks := "(pas de kill couple)"
		if bk >= 0 {
			ks = fmt.Sprintf("kill@%.1fs(dt=%dms)", float64(feed[bk].t)/1000, feed[bk].t-T)
		}
		es := "(pas d'event)"
		if be >= 0 {
			es = fmt.Sprintf("%s@%.1fs(dt=%dms)", actions[be].wpn, float64(actions[be].tms)/1000, actions[be].tms-T)
		}
		fmt.Printf("   T=%.1fs : %s | %s\n", float64(T)/1000, ks, es)
	}
}
