// tmp_heldweapon — THROWAWAY : re-scan PROPRE (paquets type-0 validés, ts fiables) des events
// fire/melee/grenade + 2 modèles d'attribution par kill :
//
//	(A) EVENT-À-T   : event auteur dans [T-2500ms, T+400ms] (modèle eventcross, échoué).
//	(B) HELD-WEAPON : dernière arme du tueur (pidx==killer) à tms<=T, sans borne basse (carry-forward = Voie A).
//
// But : trancher si la désync était un artefact de scan (contamination hors type-0) ET si le modèle
// held-weapon (arme tenue ≤ T) attribue mieux que event-à-T. Validation narration + couverture.
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
const maxMatchMs = 600000 // garde-fou anti-contamination (match ~481s)

const (
	minXUID = uint64(2e15)
	maxXUID = uint64(3e15)
)

var endMarker = []byte{0x00, 0x00, 0x2e, 0xe0}

var piName = map[int]string{0: "whiteknight2519", 1: "JAVIERLOLITO540", 2: "JGtm", 3: "LORD PEINX13", 4: "IKE ILYA", 5: "Akatsuki fire17", 6: "aldusbroncus", 7: "VitaminA1688"}
var xuidToPi = map[uint64]int{
	2535467794760703: 0, 2535437947245250: 1, 2533274823110022: 2, 2533274980284321: 3,
	2533274815845110: 4, 2535444178793711: 5, 2533274882097883: 6, 2533274826120416: 7,
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

// bitsAt lit n bits MSB-first à partir du bit absolu bp DANS le buffer p (payload local).
func bitsAt(p []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		q := bp + i
		if q>>3 >= len(p) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((p[q>>3]>>uint(7-(q&7)))&1)
	}
	return v
}

type actEvt struct {
	tms  int
	kind string
	wpn  string
	pidx int
	hit  bool
	ptyp uint16 // type de paquet porteur (diagnostic)
}

// decodeActions : itère PAQUET PAR PAQUET (header 16o), ne scanne que les payloads de paquets
// VALIDES dont le ts donne un tms dans [0, maxMatchMs]. Tag chaque event avec le type de paquet.
func decodeActions() []actEvt {
	var evs []actEvt
	skipped := 0
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		off := 0
		for off+16 <= len(d) {
			ptyp := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if sz < 0 || off+16+sz > len(d) {
				break // chunk pas en flux de paquets propre (registre/keyframe) -> stop
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if ts < t0Us {
				continue
			}
			tms := int((ts - t0Us) / 1000)
			if tms < 0 || tms > maxMatchMs {
				skipped++
				continue
			}
			scanPayload(pl, ptyp, tms, &evs)
		}
	}
	if skipped > 0 {
		fmt.Printf("(info: %d paquets ts-hors-borne ignorés)\n", skipped)
	}
	sort.Slice(evs, func(a, b int) bool { return evs[a].tms < evs[b].tms })
	return evs
}

func scanPayload(p []byte, ptyp uint16, tms int, evs *[]actEvt) {
	total := len(p) * 8
	// GRENADE
	for bp := 0; bp+110 < total; bp++ {
		if bitsAt(p, bp, 24) != 0x4c0c00 {
			continue
		}
		gid := uint32(bitsAt(p, bp+24, 32))
		if gname, ok := grenades[gid]; ok {
			*evs = append(*evs, actEvt{tms, "grenade", gname, int(bitsAt(p, bp+24+32+47, 5)), true, ptyp})
		}
	}
	// MELEE
	for bp := 0; bp+120 < total; bp++ {
		m := bitsAt(p, bp, 11)
		if m != 0x534 && m != 0x535 {
			continue
		}
		anchor := bp + 3
		typ := uint8(bitsAt(p, anchor+76, 8))
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
		hi := uint32(bitsAt(p, woff, 32))
		name, ok := h32[hi]
		if !ok && typ == 0x60 {
			hi = uint32(bitsAt(p, anchor+103, 32))
			name, ok = h32[hi]
		}
		if !ok {
			continue
		}
		*evs = append(*evs, actEvt{tms, "melee", name, int(bitsAt(p, anchor+23, 5)), m == 0x534, ptyp})
	}
	// FIRE
	for bp := 4; bp+96 < total; bp++ {
		hi := uint32(bitsAt(p, bp, 32))
		name, ok := h32[hi]
		if !ok {
			continue
		}
		if uint32(bitsAt(p, bp+32, 32)) != 0x42c9679f {
			continue
		}
		post := uint32(bitsAt(p, bp+64, 32))
		*evs = append(*evs, actEvt{tms, "fire", name, int(bitsAt(p, bp-4, 5)), (post>>28)&0xF == 0x3, ptyp})
	}
}

// ---- KILL FEED chunk_27 (inchangé vs eventcross) ----
type kfEvt struct {
	xuid             uint64
	typeHint, timeMS int
	team, duo        int
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
		if pfx := readByteAtBit(data, xe); pfx != 0x2d && pfx != 0x25 {
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
		return kfEvt{xuid, th, int(binary.BigEndian.Uint32(eb[48:52])), int(eb[37]), int(eb[36])}, true
	}
}

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
}

func buildFeed() []killRow {
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
			if dt := abs(k.timeMS - d.timeMS); dt < bd {
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
		feed = append(feed, killRow{kp, vp, k.xuid, k.timeMS})
	}
	return feed
}

func main() {
	buildCat()
	feed := buildFeed()
	actions := decodeActions()

	// Distribution par type de paquet porteur (diagnostic clé : où vivent les vrais events ?)
	type ptk struct {
		kind string
		ptyp uint16
	}
	byPT := map[ptk]int{}
	for _, a := range actions {
		byPT[ptk{a.kind, a.ptyp}]++
	}
	fmt.Println("=== Events par (type d'event, type de paquet porteur) ===")
	var keys []ptk
	for k := range byPT {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].kind != keys[j].kind {
			return keys[i].kind < keys[j].kind
		}
		return byPT[keys[i]] > byPT[keys[j]]
	})
	for _, k := range keys {
		fmt.Printf("  %-8s paquet-type=%-3d  x%d\n", k.kind, k.ptyp, byPT[k])
	}

	fmt.Println("\n=== Distribution player_index par type (events ts-valides seulement) ===")
	gp, fp, mp := map[int]int{}, map[int]int{}, map[int]int{}
	for _, a := range actions {
		switch a.kind {
		case "grenade":
			gp[a.pidx]++
		case "fire":
			fp[a.pidx]++
		case "melee":
			mp[a.pidx]++
		}
	}
	for i := 0; i < 8; i++ {
		fmt.Printf("  pi%d(%-16s) grenade=%d fire=%d melee=%d\n", i, piName[i], gp[i], fp[i], mp[i])
	}

	// ---- MODÈLE A : event-à-T [T-2500, T+400] ----
	fmt.Printf("\n=== MODÈLE A (event-à-T) sur %d kills ===\n", len(feed))
	attrA := attribute(feed, actions, -400, 2500, false)

	// ---- MODÈLE B : held-weapon carry-forward (dernier event du tueur <= T, fire+melee seulement) ----
	fmt.Printf("\n=== MODÈLE B (held-weapon carry-forward, fire+melee, <=T sans borne) sur %d kills ===\n", len(feed))
	attrB := attributeHeld(feed, actions)

	fmt.Printf("\n>>> RÉSUMÉ : modèle A = %d/%d attribués ; modèle B (held) = %d/%d attribués\n",
		attrA, len(feed), attrB, len(feed))

	// ---- VALIDATION NARRATION (modèle B held + diag temporel) ----
	fmt.Println("\n=== NARRATION (held-weapon) ===")
	narHeld(feed, actions, 4, 2, "Marteau IKE->JGtm", []int{115500, 292500, 355700, 375100})
	narHeld(feed, actions, 2, 5, "BR75 JGtm->Akatsuki", []int{112900, 329800})

	// couverture temporelle par type (les vrais events couvrent-ils tout le match ?)
	fmt.Println("\n=== Couverture temporelle (min/max tms par type, ts-valides) ===")
	for _, kind := range []string{"grenade", "melee", "fire"} {
		mn, mx, c := 1<<30, -1, 0
		for _, a := range actions {
			if a.kind != kind {
				continue
			}
			c++
			if a.tms < mn {
				mn = a.tms
			}
			if a.tms > mx {
				mx = a.tms
			}
		}
		if c > 0 {
			fmt.Printf("  %-8s n=%d [%.1fs .. %.1fs]\n", kind, c, float64(mn)/1000, float64(mx)/1000)
		}
	}
}

func attribute(feed []killRow, actions []actEvt, lo, hi int, _ bool) int {
	n := 0
	for _, k := range feed {
		if k.killerPi < 0 {
			continue
		}
		best, bestScore := -1, 1<<30
		for i, a := range actions {
			if a.pidx != k.killerPi {
				continue
			}
			d := k.t - a.tms
			if d < lo || d > hi {
				continue
			}
			score := abs(d)
			if !a.hit {
				score += 600
			}
			if score < bestScore {
				bestScore, best = score, i
			}
		}
		if best >= 0 {
			n++
		}
	}
	return n
}

func attributeHeld(feed []killRow, actions []actEvt) int {
	n := 0
	for _, k := range feed {
		if k.killerPi < 0 {
			continue
		}
		// dernier event fire/melee de ce pi à tms <= T (carry-forward)
		best, bestT := -1, -1
		for i, a := range actions {
			if a.pidx != k.killerPi || (a.kind != "fire" && a.kind != "melee") {
				continue
			}
			if a.tms <= k.t && a.tms > bestT {
				bestT, best = a.tms, i
			}
		}
		if best >= 0 {
			n++
		}
	}
	return n
}

func narHeld(feed []killRow, actions []actEvt, killerPi, victimPi int, label string, times []int) {
	fmt.Printf("-- %s (pi%d->pi%d) --\n", label, killerPi, victimPi)
	for _, T := range times {
		// kill couple le plus proche
		bk, bkdt := -1, 1<<30
		for i, k := range feed {
			if k.killerPi == killerPi && k.victimPi == victimPi && abs(k.t-T) < bkdt {
				bkdt, bk = abs(k.t-T), i
			}
		}
		// held weapon (dernier fire/melee du tueur <= T)
		bh, bhT := -1, -1
		for i, a := range actions {
			if a.pidx != killerPi || (a.kind != "fire" && a.kind != "melee") {
				continue
			}
			if a.tms <= T && a.tms > bhT {
				bhT, bh = a.tms, i
			}
		}
		ks := "(pas de kill couple)"
		if bk >= 0 {
			ks = fmt.Sprintf("kill@%.1fs(dt=%+dms)", float64(feed[bk].t)/1000, feed[bk].t-T)
		}
		hs := "(pas d'arme tenue)"
		if bh >= 0 {
			hs = fmt.Sprintf("tenue=%s@%.1fs(dt=%+dms,%s)", actions[bh].wpn, float64(actions[bh].tms)/1000, actions[bh].tms-T, actions[bh].kind)
		}
		fmt.Printf("   T=%.1fs : %s | %s\n", float64(T)/1000, ks, hs)
	}
	_ = utf16le
}
