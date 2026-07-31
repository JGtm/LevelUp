//go:build ignore

// tmp_medal_decode : décode le champ medal_type local (block[59]) des events
// type-100 en le croisant avec medals_earned + medal_definitions (noms).
// Teste : block[59] == (medal_name_id & 0xFF) ? == index ? == hash tronqué ?
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
	sharedDB        = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
	metaDB          = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/metadata.duckdb`
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

	// medals par joueur (DB) + noms
	medalsByX, names := loadMedals()

	// Test 1 : block[59] == medal_name_id & 0xFF ? pour chaque medal type-100,
	// on a le joueur — on liste ses medal_name_id et on regarde si l'octet bas matche.
	fmt.Println("=== type-100 events : block[59] vs (medal_name_id & 0xFF) des médailles DB du joueur ===")
	hitLow, total := 0, 0
	for _, e := range evs {
		if e.typeHint != 100 {
			continue
		}
		total++
		b59 := e.block[59]
		mlist := medalsByX[e.xuid]
		matched := ""
		for _, mid := range mlist {
			if byte(uint64(mid)&0xFF) == b59 {
				matched += fmt.Sprintf("%s(id=%d) ", names[mid], mid)
			}
		}
		if matched != "" {
			hitLow++
		}
		fmt.Printf("  t=%-7d gt=%-14s b59=%3d  matchLowByte: %s\n", e.timeMS, e.gt, b59, ifEmpty(matched))
	}
	fmt.Printf("  block[59]==(mid&0xFF) trouvé pour %d/%d events\n", hitLow, total)

	// Test 2 : block[58]+block[59] (uint16 BE) ? ou block[56:60] uint32 ?
	fmt.Println("\n=== type-100 : block[56:60] as uint32 BE (autre encodage medal_type) ===")
	for _, e := range evs {
		if e.typeHint != 100 {
			continue
		}
		u32 := binary.BigEndian.Uint32(e.block[56:60])
		u16 := binary.BigEndian.Uint16(e.block[58:60])
		fmt.Printf("  t=%-7d gt=%-14s b56:60=%08x(u32=%d) b58:60=%04x(u16=%d)\n", e.timeMS, e.gt, u32, u32, u16, u16)
	}

	// Liste médailles DB du match avec noms (référence)
	fmt.Println("\n=== medals DB du match (medal_name_id -> nom, low byte) ===")
	allIDs := map[int64]bool{}
	for _, l := range medalsByX {
		for _, m := range l {
			allIDs[m] = true
		}
	}
	idl := []int64{}
	for id := range allIDs {
		idl = append(idl, id)
	}
	sort.Slice(idl, func(i, j int) bool { return idl[i] < idl[j] })
	for _, id := range idl {
		fmt.Printf("  id=%-12d low=0x%02x(%3d)  name=%s\n", id, byte(uint64(id)&0xFF), byte(uint64(id)&0xFF), names[id])
	}
}

func ifEmpty(s string) string {
	if s == "" {
		return "(no match)"
	}
	return s
}

func loadMedals() (map[uint64][]int64, map[int64]string) {
	db, _ := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	defer db.Close()
	rows, _ := db.Query(`SELECT xuid, medal_name_id FROM medals_earned WHERE match_id=?`, matchID)
	byX := map[uint64][]int64{}
	ids := map[int64]bool{}
	for rows.Next() {
		var x string
		var mid int64
		rows.Scan(&x, &mid)
		var xu uint64
		fmt.Sscan(x, &xu)
		byX[xu] = append(byX[xu], mid)
		ids[mid] = true
	}
	rows.Close()

	// noms depuis metadata.medal_definitions (best-effort)
	names := map[int64]string{}
	mdb, err := sql.Open("duckdb", metaDB+"?access_mode=read_only")
	if err == nil {
		defer mdb.Close()
		// table peut être medal_definitions
		r2, e2 := mdb.Query(`SELECT medal_name_id, COALESCE(name,'') FROM medal_definitions`)
		if e2 == nil {
			for r2.Next() {
				var id int64
				var nm string
				r2.Scan(&id, &nm)
				names[id] = nm
			}
			r2.Close()
		} else {
			fmt.Println("  (medal_definitions indispo:", e2, ")")
		}
	}
	return byX, names
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
