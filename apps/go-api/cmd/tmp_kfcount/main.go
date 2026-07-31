// tmp_kfcount — THROWAWAY : compte les keyframes type-2 (et leur ts) dans les chunks gameplay.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)

func inf(p string) []byte {
	r, _ := os.ReadFile(p)
	if len(r) >= 2 && r[0] == 0x78 {
		if z, e := zlib.NewReader(bytes.NewReader(r)); e == nil {
			if d, e2 := io.ReadAll(z); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return r
}

func main() {
	t2 := 0
	for n := 0; n <= 27; n++ {
		d := inf(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		o := 0
		for o+16 <= len(d) {
			tp := binary.LittleEndian.Uint16(d[o:])
			sz := int(binary.LittleEndian.Uint32(d[o+4:]))
			ts := binary.LittleEndian.Uint64(d[o+8:])
			if sz < 0 || o+16+sz > len(d) {
				break
			}
			if tp == 2 {
				t2++
				fmt.Printf("chunk_%02d type-2 keyframe ts=%d (t=%.1fs) size=%d\n", n, ts, float64(int64(ts)-int64(t0Us))/1e6, sz)
			}
			o += 16 + sz
		}
	}
	fmt.Printf("total type-2 keyframes = %d\n", t2)
}
