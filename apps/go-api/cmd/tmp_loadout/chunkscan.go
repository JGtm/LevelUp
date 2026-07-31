package main

import (
	"encoding/binary"
	"fmt"
	"sort"
)

// scanXUIDsInChunk searches a raw (inflated) chunk for the 8 xuids, both as a
// little-endian uint64 byte sequence and as a big-endian bit sequence. The film
// roster stores xuids as 8-byte LE integers; finding them there gives the
// canonical pi order to compare against the keyframe record order.
func scanXUIDsInChunk(name string, data []byte) {
	fmt.Printf("\n--- %s (%d octets inflat.) ---\n", name, len(data))
	type hit struct {
		pi   int
		off  int
		mode string
	}
	var hits []hit
	for pi, x := range piXUID {
		// LE uint64 byte search.
		var le [8]byte
		binary.LittleEndian.PutUint64(le[:], x)
		for off := 0; off+8 <= len(data); off++ {
			if string(data[off:off+8]) == string(le[:]) {
				hits = append(hits, hit{pi, off, "LE64"})
			}
		}
		// BE uint64 byte search.
		var be [8]byte
		binary.BigEndian.PutUint64(be[:], x)
		for off := 0; off+8 <= len(data); off++ {
			if string(data[off:off+8]) == string(be[:]) {
				hits = append(hits, hit{pi, off, "BE64"})
			}
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].off < hits[j].off })
	if len(hits) == 0 {
		fmt.Printf("  aucun xuid (LE64/BE64 byte-aligned)\n")
		return
	}
	for _, h := range hits {
		fmt.Printf("  pi%d %s @byte0x%x (%d)\n", h.pi, h.mode, h.off, h.off)
	}
}
