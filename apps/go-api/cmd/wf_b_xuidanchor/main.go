package main

// wf_b_xuidanchor — peut-on ancrer le record d'un joueur via la position bit de
// SON xuid dans le payload type-2 ? Si la distance (xuid_pos -> record_weapons)
// est stable, on localise le record de JGtm sans dépendre de l'index positionnel.

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

var roster = map[uint64]string{
	2533274980284321: "p_980284321(pi3)",
	2533274815845110: "p_815845110(pi4)",
	2533274882097883: "p_882097883(pi6)",
	2535437947245250: "p_947245250(pi1)",
	2535467794760703: "p_794760703(pi0)",
	2535444178793711: "p_793711(pi5)",
	2533274826120416: "p_826120416(pi7)",
	2533274823110022: "JGtm(pi2)",
}

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
func extractPacket(d []byte, want uint16) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == want {
			return d[off+16 : off+16+sz]
		}
		off += 16 + sz
	}
	return nil
}
func bitAt(d []byte, p int) uint64 {
	if p < 0 || p>>3 >= len(d) {
		return 0
	}
	return uint64((d[p>>3] >> uint(7-(p&7))) & 1)
}
func rb(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v = (v << 1) | bitAt(d, bp+i)
	}
	return v
}
func target(x uint64) uint64 {
	le := make([]byte, 8)
	binary.LittleEndian.PutUint64(le, x)
	return binary.BigEndian.Uint64(le)
}

type wh struct {
	pos  int
	name string
}

func scanW(d []byte) []wh {
	h2n := map[uint32]string{}
	for id, n := range analysis.WeaponIDToName {
		if c, ok := analysis.WeaponFusionMap[n]; ok {
			n = c
		}
		h2n[uint32(id>>32)] = n
	}
	var out []wh
	tot := len(d) * 8
	for bp := 0; bp+32 <= tot; bp++ {
		if n, ok := h2n[uint32(rb(d, bp, 32))]; ok {
			out = append(out, wh{bp, n})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].pos < out[j].pos })
	return out
}

// xuid positions (bit) dans le payload type-2.
func xuidPositions(d []byte, x uint64) []int {
	t := target(x)
	tot := len(d) * 8
	var out []int
	for bp := 0; bp+64 <= tot; bp++ {
		if rb(d, bp, 64) == t {
			out = append(out, bp)
		}
	}
	return out
}

func main() {
	// Pour chunk_02 (ground truth connu) : où sont les xuids dans le PAYLOAD type-2
	// et à quelle distance des records armes (195k-215k) ?
	for _, ci := range []int{2, 3, 19, 20} {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ci))
		pl := extractPacket(d, 2)
		if pl == nil {
			fmt.Printf("chunk_%02d: pas de payload type-2\n", ci)
			continue
		}
		hits := scanW(pl)
		fmt.Printf("\n=== chunk_%02d : payload type-2 %d octets, %d littéraux armes (1er=%d dernier=%d) ===\n",
			ci, len(pl), len(hits), firstPos(hits), lastPos(hits))
		// positions xuid dans le payload
		for x, lbl := range roster {
			ps := xuidPositions(pl, x)
			fmt.Printf("  %-18s xuid@payload bits=%v\n", lbl, ps)
		}
	}

	// Le payload type-2 commence à un offset DANS le chunk. Les records armes sont
	// à ~195k-220k bits DANS LE CHUNK (pas le payload). Vérifions si les littéraux
	// armes tombent DANS le payload type-2 ou AVANT (région full-state).
	fmt.Println("\n=== cohérence : 1er littéral arme vs taille payload type-2 (chunk_02) ===")
	d := inflate(fmt.Sprintf("%s/chunk_02.bin", cache))
	pl := extractPacket(d, 2)
	fmt.Printf("  payload type-2 = %d octets = %d bits\n", len(pl), len(pl)*8)
	hitsChunk := scanW(d)
	hitsPayload := scanW(pl)
	fmt.Printf("  littéraux armes dans CHUNK entier : %d (1er@%d)\n", len(hitsChunk), firstPos(hitsChunk))
	fmt.Printf("  littéraux armes dans PAYLOAD type-2 : %d (1er@%d)\n", len(hitsPayload), firstPos(hitsPayload))
}

func firstPos(h []wh) int {
	if len(h) == 0 {
		return -1
	}
	return h[0].pos
}
func lastPos(h []wh) int {
	if len(h) == 0 {
		return -1
	}
	return h[len(h)-1].pos
}
