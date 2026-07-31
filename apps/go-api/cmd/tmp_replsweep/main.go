package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"levelup/go-api/internal/analysis/filmdec"
	"os"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}

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
func listFrames(d []byte) [][]byte {
	var out [][]byte
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, d[off+16:off+16+sz])
		}
		off += 16 + sz
	}
	return out
}
func freshWorld(reg *filmdec.Registry) *filmdec.World {
	raw, _ := os.ReadFile(cache + "/world_dump.txt")
	w := filmdec.NewWorld(reg)
	for _, tok := range bytes.Fields(raw) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			w.BindFull(slot, ti)
		}
	}
	return w
}
func main() {
	filmdec.PositionCalibratedSkip = true
	for n, w := range map[string]int{"object-forward-and-up-component": 9, "object-angular-velocity-component": 1, "object-shield-vitality-component": 29, "object-region-state-component": 358, "object-multiplayer-properties-component": 334} {
		filmdec.SetCalibratedWidth(n, w)
	}
	reg, _ := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	var frames [][]byte
	for idx := 2; idx <= 8; idx++ {
		frames = append(frames, listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx)))...)
	}
	for _, rsp := range []uint32{0, 1, 2, 3, 4} {
		filmdec.SetRecordStateParam(rsp)
		bip, clean, total := 0, 0, 0
		for _, fr := range frames {
			w := freshWorld(reg)
			recs, _ := filmdec.DecodeFrameRecords(filmdec.NewBitReader(fr), w, calCfg)
			for _, r := range recs {
				total++
				if r.TypeIndex == 35 && r.DesyncAt != -1 {
					bip++
				}
				if r.DesyncAt == -1 {
					clean++
				}
			}
		}
		fmt.Printf("recordStateParam=%d : biped désync=%-5d ; records propres=%d/%d\n", rsp, bip, clean, total)
	}
}
