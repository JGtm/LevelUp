// tmp_frame — L3 kickoff : comprendre la structure d'un paquet FRAME type-0.
// Header de record supposé : [R1 more][R2 type 1=new/2=del/3=delta][R7 entity-id] (MSB-first),
// puis corps (mask + composants). On parse les premiers records + dump les bits pour
// reconnaître la grammaire (typeIndex ? mask ? id range ?).
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

type pkt struct {
	typ  uint16
	ts   uint64
	data []byte
}

func listPackets(d []byte) []pkt {
	var out []pkt
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		out = append(out, pkt{typ, ts, d[off+16 : off+16+sz]})
		off += 16 + sz
	}
	return out
}

// br : lecteur MSB-first.
type br struct {
	d   []byte
	pos int
}

func (b *br) bit() int {
	if b.pos>>3 >= len(b.d) {
		b.pos++
		return 0
	}
	v := int((b.d[b.pos>>3] >> uint(7-(b.pos&7))) & 1)
	b.pos++
	return v
}
func (b *br) read(n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v = (v << 1) | uint64(b.bit())
	}
	return v
}

func main() {
	d := inflate(cache + "/chunk_05.bin")
	pkts := listPackets(d)
	// compter par type
	cnt := map[uint16]int{}
	for _, p := range pkts {
		cnt[p.typ]++
	}
	fmt.Printf("chunk_05 : %d paquets %v\n", len(pkts), cnt)

	// 1er paquet type-0
	var t0 *pkt
	for i := range pkts {
		if pkts[i].typ == 0 {
			t0 = &pkts[i]
			break
		}
	}
	if t0 == nil {
		fmt.Println("pas de type-0 dans chunk_05")
		return
	}
	// Confirmer le FrameMarker a0 7b 42 sur plusieurs type-0
	fmt.Println("\n=== prefixe des type-0 (FrameMarker a07b42 ?) ===")
	nt0 := 0
	for i := range pkts {
		if pkts[i].typ != 0 {
			continue
		}
		fmt.Printf("  type-0 #%d : %x  (ts=%d, %do)\n", nt0, pkts[i].data[:6], pkts[i].ts, len(pkts[i].data))
		nt0++
		if nt0 >= 5 {
			break
		}
	}

	// Parser la boucle de records APRÈS le marker (bit 24). Header [R1 more][R2 type][R7 id].
	fmt.Printf("\n=== records après marker (bit24), type-0 #0 (%do) ===\n", len(t0.data))
	b := &br{d: t0.data, pos: 24}
	for rec := 0; rec < 6; rec++ {
		startBit := b.pos
		more := b.read(1)
		typ := b.read(2)
		id := b.read(7)
		save := b.pos
		var dump []byte
		for i := 0; i < 12; i++ {
			dump = append(dump, byte(b.read(8)))
		}
		b.pos = save
		fmt.Printf("rec#%d @bit%d : more=%d type=%d id=%d  corps[96b]=%x\n", rec, startBit, more, typ, id, dump)
		if more == 0 {
			fmt.Println("  (more=0 -> fin de frame)")
			break
		}
		break // corps non consommé -> stop (1er record seulement)
	}
}
