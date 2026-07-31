// tmp_killmethod_fields — M1 (suite) : cherche un CHAMP DE METHODE dans le bloc 60o
// des events KILL et DEATH du chunk_27 (au-dela des medailles type-100).
//
// Strategie : dumper tous les octets b32..b59 (zone pad entre gamertag et type_hint/time)
// pour KILL et DEATH, et comparer les 4 morts marteau connues de JGtm (IKE->JGtm a
// 115.5/292.5/355.7/375.1s = MELEE certain, 0 record d'arme) vs les autres morts.
// Si un octet discrimine -> c'est le champ methode du kill feed.
//
// Scan local (copie du parser highlight : bit-non-aligne, marqueur 0xc0 + prefix 0x2d/0x25).
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
	chunk27         = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950/chunk_27.bin`
)

var endMarker = []byte{0x00, 0x00, 0x2e, 0xe0}

type ev struct {
	xuid     uint64
	gt       string
	typeHint int
	timeMS   int
	block    [60]byte
}

func gtFor(x uint64) string {
	m := map[uint64]string{
		2535467794760703: "whiteknight2519", 2535437947245250: "JAVIERLOLITO540",
		2533274823110022: "JGtm", 2533274980284321: "LORD PEINX13",
		2533274815845110: "IKE ILYA", 2535444178793711: "Akatsuki fire17",
		2533274882097883: "aldusbroncus", 2533274826120416: "VitaminA1688",
	}
	if g, ok := m[x]; ok {
		return g
	}
	return fmt.Sprintf("xuid:%d", x)
}

func main() {
	data, _ := os.ReadFile(chunk27)
	payload := inflateZlib(data)
	evs := scan(payload)

	var kills, deaths []ev
	for _, e := range evs {
		if e.typeHint == 50 {
			kills = append(kills, e)
		}
		if e.typeHint == 20 {
			deaths = append(deaths, e)
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].timeMS < kills[j].timeMS })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].timeMS < deaths[j].timeMS })
	fmt.Printf("kills=%d deaths=%d\n\n", len(kills), len(deaths))

	// 1) Octets NON constants du bloc DEATH (b32..b59) + leurs valeurs distinctes.
	fmt.Println("=== DEATH : octets b32..b59 non constants (valeurs distinctes) ===")
	for off := 32; off < 60; off++ {
		vals := map[byte]int{}
		for _, d := range deaths {
			vals[d.block[off]]++
		}
		if len(vals) > 1 {
			fmt.Printf("  b%-2d : %d distinctes  %v\n", off, len(vals), topVals(vals))
		}
	}

	// 2) Idem KILL.
	fmt.Println("\n=== KILL : octets b32..b59 non constants (valeurs distinctes) ===")
	for off := 32; off < 60; off++ {
		vals := map[byte]int{}
		for _, k := range kills {
			vals[k.block[off]]++
		}
		if len(vals) > 1 {
			fmt.Printf("  b%-2d : %d distinctes  %v\n", off, len(vals), topVals(vals))
		}
	}

	// 3) VERITE-TERRAIN MELEE : les 4 morts marteau de JGtm (IKE->JGtm).
	const jgtm = uint64(2533274823110022)
	hammerTimes := []int{115500, 292500, 355700, 375100}
	fmt.Println("\n=== DEATH de JGtm aux temps marteau (MELEE certain) : bloc b32..b59 ===")
	for _, d := range deaths {
		if d.xuid != jgtm {
			continue
		}
		isHammer := false
		for _, ht := range hammerTimes {
			if abs(d.timeMS-ht) <= 2000 {
				isHammer = true
			}
		}
		tag := ""
		if isHammer {
			tag = " <== MELEE marteau"
		}
		fmt.Printf("  t=%6.1fs JGtm  b32..59=% 02x%s\n", float64(d.timeMS)/1000, d.block[32:60], tag)
	}

	// 4) Echantillon DEATH d'autres joueurs (probable arme) pour contraste.
	fmt.Println("\n=== DEATH echantillon autres victimes (contraste) : bloc b32..b59 ===")
	n := 0
	for _, d := range deaths {
		if d.xuid == jgtm {
			continue
		}
		fmt.Printf("  t=%6.1fs %-16s  b32..59=% 02x\n", float64(d.timeMS)/1000, gtFor(d.xuid), d.block[32:60])
		n++
		if n >= 14 {
			break
		}
	}

	// 5) KILL d'IKE aux temps marteau : bloc complet (cote tueur).
	const ike = uint64(2533274815845110)
	fmt.Println("\n=== KILL d'IKE ILYA (dont 4 marteau->JGtm) : bloc b32..b59 ===")
	for _, k := range kills {
		if k.xuid != ike {
			continue
		}
		isHammer := false
		for _, ht := range hammerTimes {
			if abs(k.timeMS-ht) <= 2000 {
				isHammer = true
			}
		}
		tag := ""
		if isHammer {
			tag = " <== marteau->JGtm"
		}
		fmt.Printf("  t=%6.1fs IKE  b32..59=% 02x%s\n", float64(k.timeMS)/1000, k.block[32:60], tag)
	}

	// 6) Hypothese champ b34/b35/b38 = methode : table valeur -> exemples de victimes.
	fmt.Println("\n=== DEATH : valeur (b34,b35,b38) groupee -> #occurrences + exemples temps ===")
	type key struct{ a, b, c byte }
	grp := map[key][]int{}
	for _, d := range deaths {
		k := key{d.block[34], d.block[35], d.block[38]}
		grp[k] = append(grp[k], d.timeMS)
	}
	for k, times := range grp {
		sort.Ints(times)
		show := times
		if len(show) > 5 {
			show = show[:5]
		}
		fmt.Printf("  (b34=%d b35=%d b38=%d) : n=%d  ex(ms)=%v\n", k.a, k.b, k.c, len(times), show)
	}
}

func topVals(m map[byte]int) string {
	type kv struct {
		v byte
		c int
	}
	var s []kv
	for v, c := range m {
		s = append(s, kv{v, c})
	}
	sort.Slice(s, func(i, j int) bool { return s[i].c > s[j].c })
	out := ""
	for i, e := range s {
		if i >= 8 {
			out += "..."
			break
		}
		out += fmt.Sprintf("0x%02x:%d ", e.v, e.c)
	}
	return out
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ---- decodeur (copie locale du parser highlight, bit-non-aligne) ----

func inflateZlib(d []byte) []byte {
	if r, e := zlib.NewReader(bytes.NewReader(d)); e == nil {
		if inf, e2 := io.ReadAll(r); e2 == nil {
			return inf
		}
	}
	return d
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
		valid := th == 50 || th == 20 || th == 10 || th == 100
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
