// tmp_kftable — parseur OFFLINE de la table keyframe type-2 d'un film Halo.
//
// Format confirmé (records 0..4 matchent l'oracle) : payload = bitstream MSB-first,
// 1 bit de préfixe puis suite d'enregistrements par entité en ordre de slot CROISSANT.
// Enregistrement = [id:u32 BE][typeIndex:u32 BE][état largeur variable]. id = handle
// datum = (gen:2)<<30|(slot:30) ; ici gen==1 => id == 0x40000000|slot.
//
// La Voie B (scan-forward naïf) déraille sur des faux positifs dans l'état d'un record
// (ti=6 fait ~2060 bits). Ce tool fait donc 2 choses :
//  1. DÉCOUVERTE guidée par l'oracle : localise l'ancre EXACTE (id||ti) de chaque entité
//     connue -> largeur d'état par record -> table stateBits[ti]. Confirme le format
//     bout-en-bout et si la largeur est constante par archétype.
//  2. WALKER offline (Voie A) : sans oracle, avance pos += 64 + stateBits[ti] en se
//     servant de la table découverte. Validé contre l'oracle.
//
// Oracle : .ai/V7.5/dumps/crack/world_dump_000d5950.txt (250 entités, bipeds ti=35 512-519).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_kftable [filmDir] [worldDump]
package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	defFilm  = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	defDump  = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/crack/world_dump_000d5950.txt`
	sent     = 0xFFFFFFFF
	tableCap = 8192
)

func inflate(p string) []byte {
	raw, err := os.ReadFile(p)
	if err != nil {
		panic(err)
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

func framePayload(d []byte, want uint16) (uint64, []byte) {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == want {
			return ts, d[off+16 : off+16+sz]
		}
		off += 16 + sz
	}
	return 0, nil
}

func readBits(buf []byte, pos, n int) uint64 {
	var r uint64
	for i := 0; i < n; i++ {
		p := pos + i
		var bit uint64
		if idx := p >> 3; idx < len(buf) {
			bit = uint64(buf[idx]>>(7-uint(p&7))) & 1
		}
		r = r<<1 | bit
	}
	return r
}

func bitAt(buf []byte, p int) uint64 {
	if idx := p >> 3; idx < len(buf) {
		return uint64(buf[idx]>>(7-uint(p&7))) & 1
	}
	return 0
}

// findAnchorExact : 1re position bit p >= from où readBits(p,32)==id ET readBits(p+32,32)==ti,
// via fenêtre glissante 64-bit (O(1)/pos). -1 si absente.
func findAnchorExact(buf []byte, from, id, ti, total int) int {
	if from+64 > total {
		return -1
	}
	target := uint64(uint32(id))<<32 | uint64(uint32(ti))
	acc := readBits(buf, from, 64)
	for p := from; p+64 <= total; p++ {
		if acc == target {
			return p
		}
		acc = (acc<<1 | bitAt(buf, p+64)) // le débordement du haut est écarté par la comparaison 64-bit
	}
	return -1
}

func loadOracle(p string) map[int]int {
	f, err := os.Open(p)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	m := map[int]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, tok := range strings.Fields(line) {
			kv := strings.SplitN(tok, ":", 2)
			if len(kv) != 2 {
				continue
			}
			s, e1 := strconv.Atoi(kv[0])
			t, e2 := strconv.Atoi(kv[1])
			if e1 == nil && e2 == nil {
				m[s] = t
			}
		}
	}
	return m
}

func main() {
	dir, dump := defFilm, defDump
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if len(os.Args) > 2 {
		dump = os.Args[2]
	}

	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	numArch := len(reg.Archetypes)

	raw := inflate(dir + "/chunk_02.bin")
	ts, pay := framePayload(raw, 2)
	if pay == nil {
		fmt.Println("type-2 introuvable")
		return
	}
	total := len(pay) * 8
	fmt.Printf("registre : %d archétypes\n", numArch)
	fmt.Printf("type-2 : ts=%d, %d octets (%d bits)\n", ts, len(pay), total)

	oracle := loadOracle(dump)
	// ordre de slot croissant
	slots := make([]int, 0, len(oracle))
	for s := range oracle {
		slots = append(slots, s)
	}
	sort.Ints(slots)

	// --- DÉCOUVERTE : ancre exacte de chaque entité connue, en ordre croissant ---
	fmt.Printf("\n=== DÉCOUVERTE guidée oracle (%d entités) ===\n", len(slots))
	type found struct {
		slot, ti, bit, width int
	}
	var recs []found
	var missing []int
	pos := 0 // on cherche à partir du début
	prevBit := -1
	for _, slot := range slots {
		ti := oracle[slot]
		id := 0x40000000 | slot
		p := findAnchorExact(pay, pos, id, ti, total)
		if p < 0 {
			missing = append(missing, slot)
			continue
		}
		w := -1
		if prevBit >= 0 {
			w = p - prevBit - 64 // état du record précédent
		}
		if len(recs) > 0 {
			recs[len(recs)-1].width = w
		}
		recs = append(recs, found{slot, ti, p, -1})
		prevBit = p
		pos = p + 64
	}
	fmt.Printf("entités localisées : %d / %d (manquantes : %d %v)\n", len(recs), len(slots), len(missing), missing)

	// premiers records + largeurs
	fmt.Println("premiers records (slot ti @bit width) :")
	for i := 0; i < len(recs) && i < 15; i++ {
		fmt.Printf("  slot=%-5d ti=%-3d @bit %-7d width=%d\n", recs[i].slot, recs[i].ti, recs[i].bit, recs[i].width)
	}

	// largeurs par ti : constantes ?
	type wstat struct {
		ti       int
		n        int
		min, max int
		widths   map[int]int
	}
	byTi := map[int]*wstat{}
	for _, r := range recs {
		if r.width < 0 {
			continue
		}
		w := byTi[r.ti]
		if w == nil {
			w = &wstat{ti: r.ti, min: r.width, max: r.width, widths: map[int]int{}}
			byTi[r.ti] = w
		}
		w.n++
		if r.width < w.min {
			w.min = r.width
		}
		if r.width > w.max {
			w.max = r.width
		}
		w.widths[r.width]++
	}
	var tis []int
	for ti := range byTi {
		tis = append(tis, ti)
	}
	sort.Ints(tis)
	fmt.Println("\nlargeur d'état par archétype (ti : n occurrences, min..max, constant?) :")
	constCount, varCount := 0, 0
	for _, ti := range tis {
		w := byTi[ti]
		constant := w.min == w.max
		if constant {
			constCount++
		} else {
			varCount++
		}
		flag := "CONST"
		if !constant {
			flag = "VARIABLE"
		}
		fmt.Printf("  ti=%-3d n=%-3d width %d..%d  %s\n", ti, w.n, w.min, w.max, flag)
	}
	fmt.Printf("archétypes à largeur constante : %d ; variable : %d\n", constCount, varCount)

	// bipeds découverts
	var bipedSlots []int
	for _, r := range recs {
		if r.ti == 35 {
			bipedSlots = append(bipedSlots, r.slot)
		}
	}
	sort.Ints(bipedSlots)
	fmt.Printf("bipeds (ti=35) localisés : %d slots=%v\n", len(bipedSlots), bipedSlots)

	// ================= WALKER OFFLINE (aucune entrée oracle) =================
	fmt.Printf("\n=== WALKER OFFLINE (sans oracle) ===\n")
	walked := walkOffline(pay, numArch, total)
	compare(walked, oracle)
}

// validAnchor : q est-il une ancre valide (gen==1, prev < slot < cap, ti < numArch) ?
func validAnchor(buf []byte, q, prevSlot, numArch, total int) (slot, ti int, ok bool) {
	if q < 0 || q+64 > total {
		return 0, 0, false
	}
	id := readBits(buf, q, 32)
	if id == sent || int(id>>30) != 1 {
		return 0, 0, false
	}
	slot = int(id & 0x3FFFFFFF)
	if slot <= prevSlot || slot >= tableCap {
		return 0, 0, false
	}
	ti = int(readBits(buf, q+32, 32))
	if ti >= numArch {
		return 0, 0, false
	}
	return slot, ti, true
}

// scanNext : prochaine ancre après `from`. Fast-path : 1re ancre dont slot==prev+1
// (run dense, contourne les faux positifs à gros slot dans l'état). Sinon fallback
// min-slot (puis min-bitpos) sur la fenêtre — pour les sauts de run.
func scanNext(buf []byte, from, prevSlot, numArch, total, maxWin int) (slot, ti, at int) {
	bestSlot, bestTi, bestAt := 1<<30, 0, -1
	end := from + maxWin
	if end > total {
		end = total
	}
	sentStreak := 0
	for q := from; q+64 <= end; q++ {
		// garde padding : un long run de 0xff (SENT) marque la fin de la zone d'entités
		if readBits(buf, q, 32) == sent {
			if sentStreak++; sentStreak >= 2048 {
				return bestSlot, bestTi, bestAt
			}
			continue
		}
		sentStreak = 0
		s, t, ok := validAnchor(buf, q, prevSlot, numArch, total)
		if !ok {
			continue
		}
		if s == prevSlot+1 { // consécutif : quasi certainement le vrai bord
			return s, t, q
		}
		if s < bestSlot {
			bestSlot, bestTi, bestAt = s, t, q
		}
	}
	return bestSlot, bestTi, bestAt
}

type wrec struct {
	slot, ti, bit int
	jumped        bool
}

// walkOffline reconstruit (slot,ti) sans oracle : saute la largeur d'état connue par
// archétype (auto-calibrée) sinon scanNext ; apprend/valide les largeurs constantes.
func walkOffline(buf []byte, numArch, total int) []wrec {
	const maxWin = 120000  // > plus grand état biped observé (84811)
	width := map[int]int{} // ti -> largeur d'état constante confirmée
	seen := map[int]int{}  // ti -> 1re largeur observée (candidate constante)
	var out []wrec

	// 1re ancre : préfixe 1 bit
	pos := 1
	prev := -1
	if _, _, ok := validAnchor(buf, pos, prev, numArch, total); !ok {
		_, _, pos = scanNext(buf, pos, prev, numArch, total, maxWin)
	}
	for pos >= 0 {
		slot, ti, ok := validAnchor(buf, pos, prev, numArch, total)
		if !ok {
			break
		}
		startState := pos + 64
		// chercher le PROCHAIN record : prevSlot = slot du record courant
		var nat int
		jumped := false
		if w, has := width[ti]; has {
			if _, _, vok := validAnchor(buf, startState+w, slot, numArch, total); vok {
				nat, jumped = startState+w, true
			} else {
				_, _, nat = scanNext(buf, startState, slot, numArch, total, maxWin)
			}
		} else {
			_, _, nat = scanNext(buf, startState, slot, numArch, total, maxWin)
		}
		out = append(out, wrec{slot, ti, pos, jumped})
		prev = slot
		if nat < 0 {
			break
		}
		// apprentissage de largeur (état du record courant = nat-startState)
		w := nat - startState
		if prevW, s := seen[ti]; s {
			if prevW == w {
				width[ti] = w // 2 observations identiques => constante confirmée
			} else {
				delete(width, ti) // largeur variable : ne plus sauter
			}
		} else {
			seen[ti] = w
		}
		pos = nat
	}
	return out
}

func compare(walked []wrec, oracle map[int]int) {
	parsed := map[int]int{}
	jumps := 0
	for _, r := range walked {
		parsed[r.slot] = r.ti
		if r.jumped {
			jumps++
		}
	}
	match, mismatch, extra := 0, 0, 0
	var sampleMis []string
	for slot, ti := range parsed {
		ot, ok := oracle[slot]
		switch {
		case !ok:
			extra++
		case ot == ti:
			match++
		default:
			mismatch++
			if len(sampleMis) < 15 {
				sampleMis = append(sampleMis, fmt.Sprintf("slot %d: walker %d, oracle %d", slot, ti, ot))
			}
		}
	}
	var missing []int
	for slot := range oracle {
		if _, ok := parsed[slot]; !ok {
			missing = append(missing, slot)
		}
	}
	sort.Ints(missing)
	var bipeds []int
	for slot, ti := range parsed {
		if ti == 35 {
			bipeds = append(bipeds, slot)
		}
	}
	sort.Ints(bipeds)
	want := map[int]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}
	bh := 0
	for _, s := range bipeds {
		if want[s] {
			bh++
		}
	}
	fmt.Printf("records walkés : %d (dont %d par saut de largeur)\n", len(walked), jumps)
	fmt.Printf("slots uniques : %d\n", len(parsed))
	fmt.Printf("MATCH (slot+ti) : %d / %d oracle\n", match, len(oracle))
	fmt.Printf("MISMATCH (ti faux) : %d\n", mismatch)
	fmt.Printf("slots hors oracle : %d\n", extra)
	fmt.Printf("slots oracle manquants : %d %v\n", len(missing), missing)
	fmt.Printf("bipeds (ti=35) : %d slots=%v -> 512-519 retrouvés %d/8\n", len(bipeds), bipeds, bh)
	for _, m := range sampleMis {
		fmt.Println("  " + m)
	}
	// extras (hors oracle) : bit position + ti, pour comprendre la zone de padding
	if extra > 0 {
		fmt.Println("extras (slot ti @bit) :")
		for _, r := range walked {
			if _, ok := oracle[r.slot]; !ok {
				fmt.Printf("  slot=%-6d ti=%-3d @bit %d\n", r.slot, r.ti, r.bit)
			}
		}
	}
}
