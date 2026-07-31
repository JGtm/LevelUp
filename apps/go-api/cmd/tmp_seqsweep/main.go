// tmp_seqsweep — diag d'ALIGNEMENT du décodeur SÉQUENTIEL de frames type-0. Pour chaque
// IDLowBits candidat, décode chaque frame (chunks 3..26) depuis le bit 0 avec le World
// seedé (full dump), et mesure : records propres avant desync, records biped (ti35>=528),
// frames terminées proprement (recEnd). Le bon IDLowBits maximise records propres + bipeds.
package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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

type frame struct{ pay []byte }

func listFrames(d []byte) []frame {
	var out []frame
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, frame{d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
}

type binding struct {
	id uint32
	ti uint32
}

func loadBindings(dir string) ([]binding, map[uint32]bool) {
	raw, _ := os.ReadFile(dir + "/world_dump_full.txt")
	var bs []binding
	ti35 := map[uint32]bool{}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, tok := range strings.Fields(line) {
			kv := strings.SplitN(tok, ":", 2)
			if len(kv) != 2 {
				continue
			}
			id, e1 := strconv.ParseUint(kv[0], 10, 64)
			ti, e2 := strconv.Atoi(kv[1])
			if e1 != nil || e2 != nil {
				continue
			}
			bs = append(bs, binding{uint32(id), uint32(ti)})
			if ti == 35 && uint32(id)&0x3fffffff >= 528 {
				ti35[uint32(id)&0x3fffffff] = true
			}
		}
	}
	return bs, ti35
}

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	filmdec.SetRecordStateParam(2)
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	bs, ti35 := loadBindings(dir)

	var frames []frame
	for i := 3; i <= 26; i++ {
		frames = append(frames, listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, i)))...)
	}
	fmt.Printf("frames=%d bindings=%d ti35>=528=%d\n\n", len(frames), len(bs), len(ti35))

	// diag ciblé IDLow=11 : où désynchronise-t-on ? (slot biped ? quel composant i?)
	{
		cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
		desyncBiped, desyncOther := 0, 0
		compHist := map[int]int{}
		bipedCompHist := map[int]int{}
		for _, fr := range frames {
			w := filmdec.NewWorld(reg)
			for _, b := range bs {
				w.BindFull(b.id, b.ti)
			}
			br := filmdec.NewBitReader(fr.pay)
			recs, err := filmdec.DecodeFrameRecords(br, w, cfg)
			if err == nil || len(recs) == 0 {
				continue
			}
			r := recs[len(recs)-1]
			if r.DesyncAt < 0 {
				continue
			}
			compHist[r.Trace.DesyncAt]++
			if ti35[r.Slot] {
				desyncBiped++
				bipedCompHist[r.Trace.DesyncAt]++
			} else {
				desyncOther++
			}
		}
		fmt.Printf("DIAG IDLow=11 desync: biped=%d other=%d\n", desyncBiped, desyncOther)
		fmt.Printf("  desync compIdx (all): %v\n", compHist)
		fmt.Printf("  desync compIdx (biped): %v\n\n", bipedCompHist)
	}

	fmt.Printf("%-6s %10s %10s %10s %10s %10s\n", "IDLow", "cleanEnd", "recsTot", "bipedRec", "avgRecs", "desyncFr")
	for _, idlow := range []int{11} {
		cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idlow}
		cleanEnd, recsTot, bipedRec, desyncFr := 0, 0, 0, 0
		for _, fr := range frames {
			w := filmdec.NewWorld(reg)
			for _, b := range bs {
				w.BindFull(b.id, b.ti)
			}
			br := filmdec.NewBitReader(fr.pay)
			recs, err := filmdec.DecodeFrameRecords(br, w, cfg)
			if err == nil {
				cleanEnd++
			} else {
				desyncFr++
			}
			for _, r := range recs {
				if r.DesyncAt == -1 {
					recsTot++
					if ti35[r.Slot] {
						bipedRec++
					}
				}
			}
		}
		avg := 0
		if len(frames) > 0 {
			avg = recsTot / len(frames)
		}
		fmt.Printf("%-6d %10d %10d %10d %10d %10d\n", idlow, cleanEnd, recsTot, bipedRec, avg, desyncFr)
	}
}
