// tmp_killfeed — THROWAWAY : cherche l'ARME DU KILL FEED dans les kill events.
// Le chunk highlight (type-3) contient des events [xuid][0x2d|0x25][0xc0]...[60o]...[endmarker].
// Le parseur actuel ignore ~21 octets "pad". On dump les octets bruts des kill events de
// JGtm + on scanne les abords pour des weapon-id (high-32 famille) = l'arme du kill feed.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

const jgtmXUID = uint64(2533274823110022)

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

func readByteAtBit(d []byte, bit int) byte {
	if bit < 0 || bit+8 > len(d)*8 {
		return 0
	}
	bi := bit / 8
	off := uint(bit % 8)
	if off == 0 {
		return d[bi]
	}
	return (d[bi] << off) | (d[bi+1] >> (8 - off))
}
func readBitsU64(d []byte, bit, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		bp := bit + i
		v = (v << 1) | uint64((d[bp>>3]>>uint(7-(bp&7)))&1)
	}
	return v
}
func readU64LEAtBit(d []byte, bit int) uint64 {
	var x uint64
	for i := 0; i < 8; i++ {
		x |= uint64(readByteAtBit(d, bit+i*8)) << (uint(i) * 8)
	}
	return x
}

type ev struct {
	xuidStart int
	xuid      uint64
}

// scanEvents : positions des xuid (kill/death/medal events).
func scanEvents(d []byte) []ev {
	var out []ev
	tot := len(d) * 8
	seen := map[int]bool{}
	for ms := 8; ms <= tot-8; ms++ {
		if readByteAtBit(d, ms) != 0xc0 {
			continue
		}
		xe := ms - 8
		if xe < 64 {
			continue
		}
		pre := readByteAtBit(d, xe)
		if pre != 0x2d && pre != 0x25 {
			continue
		}
		xs := xe - 64
		if seen[xs] {
			continue
		}
		x := readU64LEAtBit(d, xs)
		if x <= 2e15 || x >= 3e15 {
			continue
		}
		out = append(out, ev{xs, x})
		seen[xs] = true
	}
	return out
}

func main() {
	// high-32 -> nom
	h2n := map[uint32]string{}
	for id, n := range analysis.WeaponIDToName {
		h2n[uint32(id>>32)] = n
	}
	// localiser le chunk highlight (le plus d'events)
	best := ""
	var bestData []byte
	bestN := 0
	for n := 0; n <= 27; n++ {
		name := fmt.Sprintf("chunk_%02d.bin", n)
		d := inflate(cache + "/" + name)
		evs := scanEvents(d)
		if len(evs) > bestN {
			bestN = len(evs)
			best = name
			bestData = d
		}
	}
	fmt.Printf("chunk highlight = %s (%d events)\n", best, bestN)
	d := bestData
	evs := scanEvents(d)

	// pour chaque event : essayer de décoder (gamertag/type/time) via la fenêtre + end-marker
	endMarker := []byte{0x00, 0x00, 0x2e, 0xe0}
	tot := len(d) * 8
	findEnd := func(from int) int {
		for bit := from; bit <= tot-32; bit++ {
			if readByteAtBit(d, bit) == endMarker[0] && readByteAtBit(d, bit+8) == endMarker[1] &&
				readByteAtBit(d, bit+16) == endMarker[2] && readByteAtBit(d, bit+24) == endMarker[3] {
				return bit
			}
		}
		return -1
	}

	type kill struct {
		xs, endBit int
		typeHint   int
		timeMS     int
		medalType  int
	}
	var jgtmEvents []kill
	allByType := map[int]int{}
	for _, e := range evs {
		endBit := findEnd(e.xuidStart)
		if endBit < 0 || endBit-60*8 < e.xuidStart {
			continue
		}
		blk := make([]byte, 60)
		for i := 0; i < 60; i++ {
			blk[i] = readByteAtBit(d, endBit-60*8+i*8)
		}
		th := int(blk[47])
		tms := int(binary.BigEndian.Uint32(blk[48:52]))
		allByType[th]++
		if e.xuid == jgtmXUID && (th == 50 || th == 100) {
			jgtmEvents = append(jgtmEvents, kill{e.xuidStart, endBit, th, tms, int(blk[59])})
		}
	}
	fmt.Printf("events par type_hint : %v\n", allByType)
	sort.Slice(jgtmEvents, func(i, j int) bool { return jgtmEvents[i].timeMS < jgtmEvents[j].timeMS })
	fmt.Printf("\nJGtm events (50=kill,100=medal) : %d\n", len(jgtmEvents))

	for ki, k := range jgtmEvents {
		blkStart := k.endBit - 60*8
		blk := make([]byte, 60)
		for i := 0; i < 60; i++ {
			blk[i] = readByteAtBit(d, blkStart+i*8)
		}
		fmt.Printf("\n=== JGtm #%d type=%d medalType=%d  t=%dms (%.1fs) ===\n", ki, k.typeHint, k.medalType, k.timeMS, float64(k.timeMS)/1000)
		// bloc 60o annoté
		fmt.Printf("  bloc60 [gamertag 0:32] %x\n", blk[0:32])
		fmt.Printf("         [pad 32:47]      %x\n", blk[32:47])
		fmt.Printf("         [type 47]=%d [time 48:52]=%x [pad 52:55]=%x [isMedal 55]=%d [pad 56:59]=%x [medalType 59]=%d\n",
			blk[47], blk[48:52], blk[52:55], blk[55], blk[56:59], blk[59])
		// 48 octets juste après le marqueur xuid+0x2d+0xc0 (le xuid fait 8o, +2 marqueurs)
		afterMarker := k.xs + 64 + 8 + 8 // après xuid(64b)+0x2d(8b)+0xc0(8b)
		after := make([]byte, 48)
		for i := 0; i < 48; i++ {
			after[i] = readByteAtBit(d, afterMarker+i*8)
		}
		fmt.Printf("  apres xuid+2d+c0 (48o) : %x\n", after)
		// scan TOUT le span pour weapon high-32
		var wp []string
		for bp := k.xs; bp+32 <= k.endBit; bp++ {
			if name, ok := h2n[uint32(readBitsU64(d, bp, 32))]; ok {
				wp = append(wp, fmt.Sprintf("%s@%+d", name, bp-k.xs))
			}
		}
		if len(wp) > 0 {
			fmt.Printf("  WEAPONS high-32 dans le span : %v\n", wp)
		} else {
			fmt.Printf("  WEAPONS high-32 dans le span : (aucune)\n")
		}
		// autres xuids (victim) dans le span
		var others []string
		for bp := k.xs + 1; bp+64 <= k.endBit; bp += 8 {
			x := readU64LEAtBit(d, bp)
			if x > 2e15 && x < 3e15 && x != jgtmXUID {
				others = append(others, fmt.Sprintf("%d@%+d", x, (bp-k.xs)/8))
			}
		}
		if len(others) > 0 {
			fmt.Printf("  autres xuids (victim?) : %v\n", others)
		}
	}
}
