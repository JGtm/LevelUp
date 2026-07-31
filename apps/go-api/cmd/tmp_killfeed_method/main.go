//go:build ignore

// tmp_killfeed_method : cherche un champ "méthode/arme" dans l'event.
//
// Stratégie :
//  1. Apparier KILL@t avec MEDAL(type-100)@t du MÊME joueur -> le medal_type
//     local (block[59]) donne la "méthode" du highlight (ex. sniper/melee).
//  2. Vérifier que TOUT octet/bit du bloc 60o KILL est constant (=> l'arme
//     n'est PAS dans l'event type-3, elle est dans l'ECS 'obje').
//  3. Dump bit-niveau de la zone time/type pour exclure un champ sous-octet.
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

type ev struct {
	xuid     uint64
	gt       string
	typeHint int
	timeMS   int
	block    [60]byte
	bitStart int
}

func main() {
	path := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950/chunk_27.bin`
	data, _ := os.ReadFile(path)
	payload := data
	if r, e := zlib.NewReader(bytes.NewReader(data)); e == nil {
		if inf, e2 := io.ReadAll(r); e2 == nil {
			payload = inf
		}
	}
	evs := scan(payload)

	var kills, medals []ev
	for _, e := range evs {
		if e.typeHint == 50 {
			kills = append(kills, e)
		}
		if e.typeHint == 100 {
			medals = append(medals, e)
		}
	}

	// 1) KILL <-> MEDAL même joueur même temps : le kill a-t-il une médaille
	//    (=> méthode "highlight") ? medal_type local = block[59] du medal event.
	fmt.Println("=== KILL appariés à un MEDAL (type-100) du même joueur @t ===")
	withMedal := 0
	for _, k := range kills {
		var m *ev
		for i := range medals {
			if medals[i].xuid == k.xuid && abs(medals[i].timeMS-k.timeMS) <= 3 {
				m = &medals[i]
				break
			}
		}
		if m != nil {
			withMedal++
			fmt.Printf("  KILL t=%-7d gt=%-16s -> MEDAL medal_type(b59)=%d\n", k.timeMS, k.gt, m.block[59])
		}
	}
	fmt.Printf("  %d/%d KILL ont une médaille highlight simultanée\n", withMedal, len(kills))

	// 2) Tout octet du bloc 60o KILL est-il constant ? (preuve d'absence d'arme)
	fmt.Println("\n=== octets NON constants du bloc 60o KILL (hors gamertag 0:31, time 48:51) ===")
	for off := 0; off < 60; off++ {
		vals := map[byte]bool{}
		for _, k := range kills {
			vals[k.block[off]] = true
		}
		if len(vals) > 1 {
			// montrer seulement hors gamertag (variable par nature) et time
			region := "?"
			switch {
			case off < 32:
				region = "gamertag"
			case off >= 48 && off <= 51:
				region = "time_ms"
			case off == 36:
				region = "DUO"
			case off == 37 || off == 38:
				region = "TEAM"
			case off == 47:
				region = "type_hint"
			}
			fmt.Printf("  off=%2d (%s) : %d valeurs distinctes\n", off, region, len(vals))
		}
	}
	fmt.Println("  (si seuls gamertag/time/duo/team/type_hint varient => AUCUN champ arme dans l'event)")

	// 3) Dump bit-exact : 16 bits AVANT le gamertag (xuid prefix 0x2d/0x25 + 0xc0)
	//    et la zone 0:64 du bloc. Cherche un champ sous-octet entre xuid et bloc.
	fmt.Println("\n=== contexte bit autour de l'event (prefix avant gamertag) : 6 KILL ===")
	n := 0
	for _, k := range kills {
		// 80 bits avant le gamertag = xuid(64) + prefix(8) + 0xc0(8) ; on dump
		// les 24 bits juste avant le bloc event (= juste après 0xc0).
		pre := readBytes(payload, k.bitStart-24, 3)
		fmt.Printf("  KILL t=%-7d gt=%-16s [pre24=%02x] block[32:40]=%02x\n", k.timeMS, k.gt, pre, k.block[32:40])
		n++
		if n >= 6 {
			break
		}
	}

	// 4) medal_type local (block[59]) : table valeur -> #occurrences (méthode highlight)
	fmt.Println("\n=== medal_type local (type-100, block[59]) global ===")
	mt := map[byte]int{}
	for _, m := range medals {
		mt[m.block[59]]++
	}
	mks := []int{}
	for k := range mt {
		mks = append(mks, int(k))
	}
	sort.Ints(mks)
	for _, k := range mks {
		fmt.Printf("  medal_type=%3d (0x%02x) : %d\n", k, k, mt[byte(k)])
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func scan(data []byte) []ev {
	totalBits := len(data) * 8
	var out []ev
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
		e, ok := parseAt(data, xs, xuid)
		if !ok {
			continue
		}
		out = append(out, e)
		seen[xs] = true
	}
	return out
}

func parseAt(data []byte, xs int, xuid uint64) (ev, bool) {
	total := len(data) * 8
	wend := xs + eventWindowBits
	if wend > total {
		wend = total
	}
	from := xs
	for {
		end := findMarker(data, from, wend, endMarker)
		if end < 0 {
			return ev{}, false
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
		valid := th == 50 || th == 20 || th == 10 || th == 100 || th == 101 ||
			th == 51 || th == 52 || th == 150 || th == 200 || th == 205 ||
			th == 210 || th == 220 || th == 225 || th == 230 || th == 235 ||
			th == 240 || th == 245 || th == 250
		if !valid {
			from = end + 1
			continue
		}
		var e ev
		copy(e.block[:], eb)
		e.xuid = xuid
		e.gt = utf16le(eb[0:32])
		e.typeHint = th
		e.timeMS = int(binary.BigEndian.Uint32(eb[48:52]))
		e.bitStart = st
		return e, true
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
