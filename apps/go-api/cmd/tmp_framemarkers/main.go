// tmp_framemarkers — histogramme des marqueurs payload[0] des frames type-0 d'un film,
// pour localiser le frame kill-event (le warp trouve le dégât via payload[0]==0xd2 ; le
// kill-event a-t-il son propre marqueur ?). Diagnostic offline du build kill-weapon same-clock.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_framemarkers [filmID]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
)

const root = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`

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

func main() {
	m := "000d5950"
	if len(os.Args) > 1 {
		m = os.Args[1]
	}
	cache := root + "/" + m
	cnt := map[byte]int{}
	szSum := map[byte]int{}
	szMin := map[byte]int{}
	szMax := map[byte]int{}
	total := 0
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ != 0 || len(pl) == 0 {
				continue
			}
			b := pl[0]
			cnt[b]++
			szSum[b] += sz
			if szMin[b] == 0 || sz < szMin[b] {
				szMin[b] = sz
			}
			if sz > szMax[b] {
				szMax[b] = sz
			}
			total++
		}
	}
	type mk struct {
		b byte
		n int
	}
	var mks []mk
	for b, n := range cnt {
		mks = append(mks, mk{b, n})
	}
	sort.Slice(mks, func(i, j int) bool { return mks[i].n > mks[j].n })
	fmt.Printf("=== film %s : %d frames type-0, %d marqueurs distincts ===\n", m, total, len(mks))
	fmt.Println("payload[0]  count   sz[min..moy..max]")
	for i, x := range mks {
		if i >= 30 {
			fmt.Printf("... (%d marqueurs de plus)\n", len(mks)-30)
			break
		}
		tag := ""
		if x.b == 0xd2 {
			tag = "  <- DEGAT 0xd2"
		}
		fmt.Printf("  0x%02X       %-6d  [%d..%d..%d]%s\n", x.b, x.n, szMin[x.b], szSum[x.b]/x.n, szMax[x.b], tag)
	}
}
