// tmp_killframe — teste si un marqueur de frame type-0 est le kill-event : décode chaque
// frame du marqueur avec la grammaire §174 (tueur/victime/assist = R1+optR5 ×3, présence =
// R(1)==0 → R(5)) en balayant l'offset, et cherche l'offset où killer≠victim tient sur la
// majorité (= le marqueur/offset du kill-event dans le flux frame, pour la corrélation same-clock).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_killframe [filmID] [markerHex]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
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

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

// decodeKV décode killer/victim/assist à l'offset off (R1+optR5, présence = R1==0).
func decodeKV(pl []byte, off int) (k, v, a int) {
	bp := off
	readOpt := func() int {
		present := bitsAt(pl, bp, 1) == 0
		bp++
		if present {
			val := int(bitsAt(pl, bp, 5))
			bp += 5
			return val
		}
		return -1
	}
	return readOpt(), readOpt(), readOpt()
}

func collect(cache string, marker byte) [][]byte {
	var frames [][]byte
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		off := 0
		for off+16 <= len(d) {
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			typ := binary.LittleEndian.Uint16(d[off:])
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ == 0 && len(pl) > 0 && pl[0] == marker {
				frames = append(frames, pl)
			}
		}
	}
	return frames
}

func testMarker(cache string, marker byte) {
	frames := collect(cache, marker)
	if len(frames) == 0 {
		return
	}
	fmt.Printf("=== marqueur 0x%02X : %d frames ===\n", marker, len(frames))
	best, bestValid := -1, 0
	for o := 0; o <= 88; o++ {
		valid := 0
		distinct := map[[2]int]bool{}
		for _, pl := range frames {
			k, v, _ := decodeKV(pl, o)
			if k >= 0 && v >= 0 && k != v && k < 24 && v < 24 {
				valid++
				distinct[[2]int{k, v}] = true
			}
		}
		if valid > bestValid {
			bestValid, best = valid, o
		}
		if valid*100 >= len(frames)*70 { // >=70% valides
			fmt.Printf("  off=%-2d valid=%d/%d (%.0f%%) distinct=%d\n", o, valid, len(frames), float64(valid)*100/float64(len(frames)), len(distinct))
		}
	}
	fmt.Printf("  => meilleur off=%d valid=%d/%d\n", best, bestValid, len(frames))
}

func main() {
	m := "000d5950"
	if len(os.Args) > 1 {
		m = os.Args[1]
	}
	cache := root + "/" + m
	if len(os.Args) > 2 {
		x, _ := strconv.ParseUint(os.Args[2], 16, 8)
		if len(os.Args) > 3 && os.Args[3] == "dump" {
			frames := collect(cache, byte(x))
			fmt.Printf("=== marqueur 0x%02X : %d frames — 8 premières (48 octets) ===\n", byte(x), len(frames))
			for i := 0; i < len(frames) && i < 8; i++ {
				n := 48
				if n > len(frames[i]) {
					n = len(frames[i])
				}
				fmt.Printf("  [%3do] %x\n", len(frames[i]), frames[i][:n])
			}
			return
		}
		testMarker(cache, byte(x))
		return
	}
	for _, mk := range []byte{0xE6, 0xE5, 0xC7, 0xF3, 0xD3, 0xCA, 0xCB} {
		testMarker(cache, mk)
	}
}
