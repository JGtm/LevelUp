//go:build ignore

// tmp_killfeed_enum : énumère TOUS les type_hint du chunk_27 (sans filtre
// d'allowlist), dump un échantillon de chaque, et cherche suicides/joins/leaves
// + un champ "méthode/arme" éventuel. Croise medals_earned.
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
	"unicode/utf16"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	minXUID         = uint64(2e15)
	maxXUID         = uint64(3e15)
	eventWindowBits = 20000
	eventDataBytes  = 60
	matchID         = "000d5950-83d9-423f-ab55-d068a7237b9f"
)

var endMarker = []byte{0x00, 0x00, 0x2e, 0xe0}

type ev struct {
	xuid     uint64
	gt       string
	typeHint int
	timeMS   int
	block    [60]byte
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
	fmt.Printf("events (any type_hint 0..255)=%d\n\n", len(evs))

	// Histogramme COMPLET (aucune allowlist)
	hist := map[int]int{}
	for _, e := range evs {
		hist[e.typeHint]++
	}
	ks := []int{}
	for k := range hist {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	fmt.Println("=== type_hint histogram (ALL) ===")
	for _, k := range ks {
		fmt.Printf("  type_hint=%-4d (0x%02x) count=%d\n", k, k, hist[k])
	}

	// Échantillon par type_hint : montrer block[36],[37],[59], time, gt
	fmt.Println("\n=== sample per type_hint (block[36]=duo block[37]=team block[58] block[59]) ===")
	for _, th := range ks {
		n := 0
		for _, e := range evs {
			if e.typeHint != th {
				continue
			}
			fmt.Printf("  th=%-4d t=%-7d gt=%-16s b36=%d b37=%d b53=%d b54=%d b58=%d b59=%d\n",
				th, e.timeMS, e.gt, e.block[36], e.block[37], e.block[53], e.block[54], e.block[58], e.block[59])
			n++
			if n >= 4 {
				break
			}
		}
	}

	// Suicides : DEATH (th=20) qui n'a PAS de KILL adverse simultané.
	var kills, deaths []ev
	for _, e := range evs {
		if e.typeHint == 50 {
			kills = append(kills, e)
		}
		if e.typeHint == 20 {
			deaths = append(deaths, e)
		}
	}
	fmt.Println("\n=== DEATH sans KILL adverse simultané (=> suicide/chute/bornage) ===")
	for _, d := range deaths {
		hasKiller := false
		for _, k := range kills {
			if abs(k.timeMS-d.timeMS) <= 3 && k.xuid != d.xuid {
				hasKiller = true
				break
			}
		}
		if !hasKiller {
			fmt.Printf("  DEATH t=%-7d gt=%-16s b36=%d b37=%d b59=%d  block=%02x\n",
				d.timeMS, d.gt, d.block[36], d.block[37], d.block[59], d.block[32:60])
		}
	}

	// type_hint=100 : médaille. block[55]=is_medal, block[59]=medal_type local.
	fmt.Println("\n=== type_hint=100 : medal_type (block[59]) distribution ===")
	mt := map[byte]int{}
	for _, e := range evs {
		if e.typeHint == 100 {
			mt[e.block[59]]++
		}
	}
	mks := []int{}
	for k := range mt {
		mks = append(mks, int(k))
	}
	sort.Ints(mks)
	for _, k := range mks {
		fmt.Printf("  block[59]=%3d (0x%02x) : %d\n", k, k, mt[byte(k)])
	}

	// Comparaison : par joueur, #medal events type-100 vs medals_earned SUM(count)
	fmt.Println("\n=== par joueur : #type-100 events (film) vs medals_earned (DB) ===")
	mc := loadMedalCount()
	filmMedals := map[uint64]int{}
	for _, e := range evs {
		if e.typeHint == 100 {
			filmMedals[e.xuid]++
		}
	}
	allx := map[uint64]bool{}
	for x := range mc {
		allx[x] = true
	}
	for x := range filmMedals {
		allx[x] = true
	}
	xl := []uint64{}
	for x := range allx {
		xl = append(xl, x)
	}
	sort.Slice(xl, func(i, j int) bool { return xl[i] < xl[j] })
	for _, x := range xl {
		fmt.Printf("  xuid=%d  film_type100=%d  db_medals_sum=%d\n", x, filmMedals[x], mc[x])
	}

	// Recherche élargie : y a-t-il des type_hint hors {20,50,100} si on relâche
	// le filtre gamertag ? (join/leave/mode auraient un gamertag aussi)
	fmt.Println("\n=== recherche join/leave/mode : type_hint NOT IN (20,50,100) ===")
	other := 0
	for _, e := range evs {
		if e.typeHint != 20 && e.typeHint != 50 && e.typeHint != 100 {
			fmt.Printf("  th=%-4d t=%-7d gt=%-16s block=%02x\n", e.typeHint, e.timeMS, e.gt, e.block[32:60])
			other++
		}
	}
	fmt.Printf("  total autres=%d\n", other)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func loadMedalCount() map[uint64]int {
	dbPath := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
	db, _ := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	defer db.Close()
	rows, _ := db.Query(`SELECT xuid, SUM(count) FROM medals_earned WHERE match_id=? GROUP BY 1`, matchID)
	out := map[uint64]int{}
	for rows.Next() {
		var x string
		var c int
		rows.Scan(&x, &c)
		var xu uint64
		fmt.Sscan(x, &xu)
		out[xu] = c
	}
	return out
}

// ---- décodeur ----

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
		// pas d'allowlist : on accepte tout type_hint, mais on exige un gamertag
		// plausible (au moins 2 octets non nuls dans les 4 premiers, ASCII/UTF16)
		// pour éviter les faux positifs de end-marker sur des zones de zéros.
		if eb[0] == 0 && eb[1] == 0 {
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
