// tmp_filmprobe — THROWAWAY : décompresse un chunk film (zlib si présent) et liste
// les paquets (header 16o : [Type u16 LE][b2][b3][Size u32 LE][Timestamp u64 LE]).
// Sert à reverser empiriquement le framing FRAME pour le décodeur ECS (M2).
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tmp_filmprobe <chunk.bin> [maxPkts]")
		os.Exit(1)
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	data := raw
	if len(raw) >= 2 && raw[0] == 0x78 {
		zr, zerr := zlib.NewReader(bytes.NewReader(raw))
		if zerr != nil {
			panic(zerr)
		}
		dec, rerr := io.ReadAll(zr)
		if rerr != nil && len(dec) == 0 {
			panic(rerr)
		}
		data = dec
		fmt.Printf("zlib: %d -> %d bytes\n", len(raw), len(data))
	} else {
		fmt.Printf("raw: %d bytes\n", len(data))
	}

	maxPkts := 60
	counts := map[uint16]int{}
	off := 0
	n := 0
	for off+16 <= len(data) {
		typ := binary.LittleEndian.Uint16(data[off:])
		b2, b3 := data[off+2], data[off+3]
		size := binary.LittleEndian.Uint32(data[off+4:])
		ts := binary.LittleEndian.Uint64(data[off+8:])
		counts[typ]++
		if n < maxPkts {
			fmt.Printf("pkt %3d @%-9d type=%-3d b2=%02x b3=%02x size=%-8d ts=%d\n", n, off, typ, b2, b3, size, ts)
		}
		// Dump du payload des FRAME (type-0) pour reverser le framing du record.
		if typ == 0 && size >= 1 && off+16+int(size) <= len(data) {
			pl := data[off+16 : off+16+int(size)]
			dumpLen := len(pl)
			if dumpLen > 64 {
				dumpLen = 64
			}
			fmt.Printf("     FRAME payload (%d o): % x\n", len(pl), pl[:dumpLen])
		}
		if off+16+int(size) > len(data) {
			fmt.Printf("  (size %d depasse le buffer, stop)\n", size)
			break
		}
		off += 16 + int(size)
		n++
	}
	fmt.Printf("\n=== %d paquets, fin off=%d / %d ===\n", n, off, len(data))
	fmt.Printf("histogramme types: %v\n", counts)
}
