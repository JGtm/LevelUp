//go:build ignore

// tmp_killfeed_decode : décode le chunk_27 (highlight events type-3) du match
// 000d5950 et dump le bloc 60o COMPLET de chaque event, en isolant les octets
// "pad" non décodés (32:47, 52:59) pour KILL (type_hint=50) et DEATH (20).
//
// Objectif : trouver le champ tueur<->victime + le champ méthode/arme dans le pad.
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
)

const (
	minXUID         = uint64(2e15)
	maxXUID         = uint64(3e15)
	eventWindowBits = 20000
	eventDataBytes  = 60
)

var endMarker = []byte{0x00, 0x00, 0x2e, 0xe0}

// roster pi -> xuid (mission)
var rosterPI = map[int]uint64{
	0: 2535467794760703,
	1: 2535437947245250,
	2: 2533274823110022, // JGtm
	3: 2533274980284321,
	4: 2533274815845110,
	5: 2535444178793711,
	6: 2533274882097883,
	7: 2533274826120416,
}

func xuidToPI(x uint64) int {
	for pi, xu := range rosterPI {
		if xu == x {
			return pi
		}
	}
	return -1
}

type rawEvent struct {
	xuid     uint64
	gamertag string
	typeHint int
	timeMS   int
	isMedal  bool
	medalT   int
	block    [60]byte
	bitPos   int
}

func main() {
	path := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950/chunk_27.bin`
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("read:", err)
		os.Exit(1)
	}
	payload := data
	if r, err := zlib.NewReader(bytes.NewReader(data)); err == nil {
		defer r.Close()
		if inf, e := io.ReadAll(r); e == nil {
			payload = inf
		}
	}
	fmt.Printf("payload=%d bytes\n", len(payload))

	events := scan(payload)
	fmt.Printf("events scanned=%d\n\n", len(events))

	// Histogramme type_hint
	hist := map[int]int{}
	for _, e := range events {
		hist[e.typeHint]++
	}
	keys := []int{}
	for k := range hist {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	fmt.Println("=== type_hint histogram ===")
	for _, k := range keys {
		fmt.Printf("  type_hint=%-4d count=%d\n", k, hist[k])
	}
	fmt.Println()

	// Dump complet hex des 60 octets pour 6 KILL + 6 DEATH (vue brute)
	fmt.Println("=== FULL 60-byte hex : 6 KILL ===")
	dumpFull(events, 50, 6)
	fmt.Println("\n=== FULL 60-byte hex : 6 DEATH ===")
	dumpFull(events, 20, 6)
	fmt.Println()

	// Per-byte variance : quels octets varient ? (sur tous les KILL)
	fmt.Println("=== per-byte distinct values across all KILL (type_hint=50) ===")
	byteVariance(events, 50)
	fmt.Println("\n=== per-byte distinct values across all DEATH (type_hint=20) ===")
	byteVariance(events, 20)
	fmt.Println()

	// Table pi -> {block[36], block[37], block[38]} pour comprendre ce qu'ils encodent.
	fmt.Println("=== per-PI mapping of block[36],[37],[38] (subject attributes) ===")
	type attrKey struct{ pi, b36, b37, b38 int }
	seen := map[attrKey]int{}
	for _, e := range events {
		if e.isMedal {
			continue
		}
		pi := xuidToPI(e.xuid)
		if pi < 0 {
			continue
		}
		k := attrKey{pi, int(e.block[36]), int(e.block[37]), int(e.block[38])}
		seen[k]++
	}
	type kv struct {
		k attrKey
		c int
	}
	var kvs []kv
	for k, c := range seen {
		kvs = append(kvs, kv{k, c})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].k.pi < kvs[j].k.pi })
	// b37 (=team) par xuid pour valider contre killer_victim_pairs (no team-kill)
	teamOf := map[uint64]int{}
	for _, e := range kvs {
		fmt.Printf("  pi=%d xuid=%d  b36=%d b37=%d b38=%d  (n=%d)\n", e.k.pi, rosterPI[e.k.pi], e.k.b36, e.k.b37, e.k.b38, e.c)
		teamOf[rosterPI[e.k.pi]] = e.k.b37
	}
	fmt.Println("\n  Team A (b37=0):")
	for x, t := range teamOf {
		if t == 0 {
			fmt.Printf("    %d (pi=%d)\n", x, xuidToPI(x))
		}
	}
	fmt.Println("  Team B (b37=1):")
	for x, t := range teamOf {
		if t == 1 {
			fmt.Printf("    %d (pi=%d)\n", x, xuidToPI(x))
		}
	}
}

func dumpFull(events []rawEvent, typeHint, limit int) {
	n := 0
	for _, e := range events {
		if e.typeHint != typeHint || e.isMedal {
			continue
		}
		fmt.Printf("[t=%-7d gt=%-16s pi=%d]\n  %02x\n", e.timeMS, e.gamertag, xuidToPI(e.xuid), e.block[:])
		n++
		if n >= limit {
			break
		}
	}
}

// byteVariance : pour chaque offset 32..59, liste les valeurs distinctes observées.
func byteVariance(events []rawEvent, typeHint int) {
	for off := 32; off < 60; off++ {
		vals := map[byte]int{}
		for _, e := range events {
			if e.typeHint != typeHint || e.isMedal {
				continue
			}
			vals[e.block[off]]++
		}
		if len(vals) <= 1 {
			// constant — montrer la valeur
			for v := range vals {
				fmt.Printf("  off=%2d CONST=%02x\n", off, v)
			}
			continue
		}
		// variable — lister
		ks := []int{}
		for v := range vals {
			ks = append(ks, int(v))
		}
		sort.Ints(ks)
		s := ""
		for _, k := range ks {
			s += fmt.Sprintf("%02x:%d ", k, vals[byte(k)])
		}
		fmt.Printf("  off=%2d VAR  %s\n", off, s)
	}
}

func dumpEvents(events []rawEvent, typeHint, limit int) {
	n := 0
	for _, e := range events {
		if e.typeHint != typeHint || e.isMedal {
			continue
		}
		pi := xuidToPI(e.xuid)
		// octets pad
		pad32_47 := e.block[32:47]
		fmt.Printf("[#%02d t=%-7d gt=%-16s pi=%d] pad32:47=% 02x | mtype=%02x | pad52:55=% 02x ismedal=%02x pad56:59=% 02x mtyp=%02x\n",
			n, e.timeMS, e.gamertag, pi,
			pad32_47,
			e.block[47],
			e.block[52:55], e.block[55], e.block[56:59], e.block[59])
		n++
		if n >= limit {
			fmt.Printf("  ... (%d more)\n", countType(events, typeHint)-n)
			break
		}
	}
}

func countType(events []rawEvent, th int) int {
	c := 0
	for _, e := range events {
		if e.typeHint == th && !e.isMedal {
			c++
		}
	}
	return c
}

func scan(data []byte) []rawEvent {
	totalBits := len(data) * 8
	var out []rawEvent
	seen := map[int]bool{}
	for markerStart := 8; markerStart <= totalBits-8; markerStart++ {
		if readByteAtBit(data, markerStart) != 0xc0 {
			continue
		}
		xuidEnd := markerStart - 8
		if xuidEnd < 64 {
			continue
		}
		prefix := readByteAtBit(data, xuidEnd)
		if prefix != 0x2d && prefix != 0x25 {
			continue
		}
		xuidStart := xuidEnd - 64
		if seen[xuidStart] {
			continue
		}
		xuid := readUint64LEAtBit(data, xuidStart)
		if xuid <= minXUID || xuid >= maxXUID {
			continue
		}
		ev, ok := parseAt(data, xuidStart, xuid)
		if !ok {
			continue
		}
		out = append(out, ev)
		seen[xuidStart] = true
	}
	return out
}

func parseAt(data []byte, xuidStartBit int, xuid uint64) (rawEvent, bool) {
	totalBits := len(data) * 8
	windowEnd := xuidStartBit + eventWindowBits
	if windowEnd > totalBits {
		windowEnd = totalBits
	}
	searchFrom := xuidStartBit
	for {
		endPos := findBitMarker(data, searchFrom, windowEnd, endMarker)
		if endPos < 0 {
			return rawEvent{}, false
		}
		start := endPos - eventDataBytes*8
		if start < xuidStartBit {
			searchFrom = endPos + 1
			continue
		}
		eb := readBytesAtBit(data, start, eventDataBytes)
		if eb == nil {
			searchFrom = endPos + 1
			continue
		}
		th := int(eb[47])
		// reconnaître event valide : kill/death/mode/medal
		valid := th == 50 || th == 20 || th == 10 ||
			th == 51 || th == 52 || th == 100 || th == 101 || th == 150 ||
			th == 200 || th == 205 || th == 210 || th == 220 || th == 225 ||
			th == 230 || th == 235 || th == 240 || th == 245 || th == 250
		if !valid {
			searchFrom = endPos + 1
			continue
		}
		var ev rawEvent
		copy(ev.block[:], eb)
		ev.xuid = xuid
		ev.gamertag = decodeUTF16LE(eb[0:32])
		ev.typeHint = th
		ev.timeMS = int(binary.BigEndian.Uint32(eb[48:52]))
		ev.isMedal = eb[55] == 1
		ev.medalT = int(eb[59])
		ev.bitPos = start
		return ev, true
	}
}

func decodeUTF16LE(b []byte) string {
	u16 := make([]uint16, len(b)/2)
	for i := range u16 {
		u16[i] = binary.LittleEndian.Uint16(b[i*2 : i*2+2])
	}
	for i, c := range u16 {
		if c == 0 {
			u16 = u16[:i]
			break
		}
	}
	return string(utf16.Decode(u16))
}

func readByteAtBit(data []byte, bit int) byte {
	if bit < 0 || bit+8 > len(data)*8 {
		return 0
	}
	bi := bit / 8
	off := uint(bit % 8)
	if off == 0 {
		return data[bi]
	}
	hi := data[bi] << off
	lo := data[bi+1] >> (8 - off)
	return hi | lo
}

func readBytesAtBit(data []byte, bit, n int) []byte {
	if bit < 0 || bit+n*8 > len(data)*8 {
		return nil
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = readByteAtBit(data, bit+i*8)
	}
	return out
}

func readUint64LEAtBit(data []byte, bit int) uint64 {
	b := readBytesAtBit(data, bit, 8)
	if b == nil {
		return 0
	}
	var x uint64
	for i := 0; i < 8; i++ {
		x |= uint64(b[i]) << (uint(i) * 8)
	}
	return x
}

func findBitMarker(data []byte, startBit, endBit int, pattern []byte) int {
	if startBit < 0 {
		startBit = 0
	}
	totalBits := len(data) * 8
	if endBit > totalBits {
		endBit = totalBits
	}
	patBits := len(pattern) * 8
	for bit := startBit; bit <= endBit-patBits; bit++ {
		match := true
		for i := 0; i < len(pattern); i++ {
			if readByteAtBit(data, bit+i*8) != pattern[i] {
				match = false
				break
			}
		}
		if match {
			return bit
		}
	}
	return -1
}
