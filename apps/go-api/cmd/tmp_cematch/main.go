// tmp_cematch — THROWAWAY : apparie chaque record CE dead-state RÉSOLU (oracle
// ground-truth) à sa frame offline via le fingerprint (16 octets au buffer base f8)
// + la position bit b2c, puis compare mon EnumA/EnumB DÉCODÉ aux victime/tueur RÉELS
// résolus par le jeu. Tranche : décodage aligné (EnumA correle la victime) vs garbage.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_cematch [prefix]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const wt = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce`

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

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

type packet struct {
	chunk   int
	payload []byte
}

func listFrames(d []byte, chunk int) []packet {
	var out []packet
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, packet{chunk, d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
}

func freshWorld(reg *filmdec.Registry) *filmdec.World {
	dumpPath := cache + "/world_dump.txt"
	if p := os.Getenv("WORLD_DUMP"); p != "" {
		dumpPath = p // binding complet capturé en CE (tmp_mkworld)
	}
	raw, _ := os.ReadFile(dumpPath)
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

// idx via base 0xE1500000 / 0xEC500000 stride 0x10002.
func tryIdx(h uint32) int {
	for _, base := range []uint32{0xE1500000, 0xEC500000} {
		if h >= base {
			d := h - base
			if d%0x10002 == 0 && d/0x10002 <= 31 {
				return int(d / 0x10002)
			}
		}
	}
	return -1
}

type ceRec struct {
	vic, kil, gid uint32
	b2c           int
	fp            []byte
}

func main() {
	prefix := "000d5950"
	if len(os.Args) >= 2 {
		prefix = os.Args[1]
	}
	raw, err := os.ReadFile(wt + "/" + prefix + "_deadstate.bin")
	if err != nil {
		fmt.Printf("pas de capture: %v\n", err)
		return
	}
	const stride = 0x60
	u := func(off int, b []byte) uint32 { return binary.LittleEndian.Uint32(b[off:]) }
	var ces []ceRec
	for i := 0; i+stride <= len(raw); i += stride {
		b := raw[i : i+stride]
		if u(0x0c, b) == 0xffffffff {
			continue // non résolu
		}
		ces = append(ces, ceRec{
			vic: u(0x0c, b), kil: u(0x10, b), gid: u(0x14, b),
			b2c: int(u(0x1c, b)), fp: append([]byte(nil), b[0x40:0x50]...),
		})
	}
	fmt.Printf("=== %d records CE RÉSOLUS ===\n\n", len(ces))

	reg, _ := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))

	// World persistant : replay chunks 1..26, accumule les packets ET, à chaque packet,
	// teste si son payload contient un fingerprint CE -> décode, localise le dead-state à b2c.
	w := freshWorld(reg)
	type matchRow struct {
		ce        ceRec
		found     bool
		clean     bool
		eA, eB    int32
		slot      uint32
		startBit  int
		nDeadHere int
		expected  int
		endBit    int
		desync    string
	}
	matched := map[int]matchRow{} // index dans ces -> meilleur match
	verbosePrinted := false
	for idx := 1; idx <= 26; idx++ {
		for _, pk := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx)), idx) {
			// décode (avec World persistant) AVANT de tester les fp, pour faire avancer le World
			br := filmdec.NewBitReader(pk.payload)
			recs, derr := filmdec.DecodeFrameRecords(br, w, calCfg)
			for ci, ce := range ces {
				p := bytes.Index(pk.payload, ce.fp)
				if p < 0 {
					continue
				}
				// anti faux-match : la frame doit être assez longue pour contenir la
				// mort à b2c (sinon c'est une frame courte qui partage les 16 octets de fp).
				if p*8+ce.b2c > len(pk.payload)*8 {
					continue
				}
				if _, done := matched[ci]; done {
					continue
				}
				expected := p*8 + ce.b2c
				row := matchRow{ce: ce, found: true, clean: derr == nil, expected: expected}
				if os.Getenv("VERBOSE") != "" && !verbosePrinted && len(recs) > 0 && recs[len(recs)-1].Trace.EndBit > 500 {
					verbosePrinted = true
					fmt.Printf("--- VERBOSE frame de mort (chunk %d, payload %d octets, fp@%d, expBit=%d, err=%v) ---\n", pk.chunk, len(pk.payload), p, expected, derr)
					for ri := 0; ri < len(recs) && ri < 6; ri++ {
						r := recs[ri]
						fmt.Printf("  rec%d type=%d slot=%d ti=%d mask=%016x desyncAt=%d endBit=%d\n",
							ri, r.Type, r.Slot, r.TypeIndex, r.Trace.Mask, r.DesyncAt, r.Trace.EndBit)
						for _, c := range r.Trace.Comps {
							fmt.Printf("      i%-2d %-40s startBit=%d ported=%v\n", c.Index, c.Name, c.StartBit, c.Ported)
						}
					}
				}
				// où/à quel bit s'arrête le décodage, et sur quel composant
				if len(recs) > 0 {
					last := recs[len(recs)-1]
					row.endBit = last.Trace.EndBit
					if last.DesyncAt >= 0 {
						if arch, ok := reg.Archetype(int(last.TypeIndex)); ok && last.DesyncAt < len(arch.Components) {
							row.desync = fmt.Sprintf("ti=%d i%d %s", last.TypeIndex, last.DesyncAt, arch.Components[last.DesyncAt])
						}
					}
				}
				best := 1 << 30
				for _, r := range recs {
					if !bipedSlots[r.Slot] {
						continue
					}
					for _, c := range r.Trace.Comps {
						if c.Name != "object-dead-state-component" {
							continue
						}
						row.nDeadHere++
						d := c.StartBit - expected
						if d < 0 {
							d = -d
						}
						if d < best {
							best = d
							row.startBit = c.StartBit
							row.slot = r.Slot
							if r.Trace.Dead != nil {
								row.eA, row.eB = r.Trace.Dead.EnumA, r.Trace.Dead.EnumB
							}
						}
					}
				}
				matched[ci] = row
			}
		}
	}

	// rapport
	alignedOK, found := 0, 0
	for ci, ce := range ces {
		m, ok := matched[ci]
		vIdx, kIdx := tryIdx(ce.vic), tryIdx(ce.kil)
		if !ok || !m.found {
			fmt.Printf("CE vic=idx%d kil=idx%d b2c=%d : PAS DE FRAME OFFLINE (fp introuvable)\n", vIdx, kIdx, ce.b2c)
			continue
		}
		found++
		fmt.Printf("CE vic=idx%d kil=idx%d expBit=%d | offline: clean=%v endBit=%d desync=[%s] deadStatesHere=%d slot=%d startBit=%d eA=%d eB=%d\n",
			vIdx, kIdx, m.expected, m.clean, m.endBit, m.desync, m.nDeadHere, m.slot, m.startBit, m.eA, m.eB)
		if m.nDeadHere > 0 {
			alignedOK++
		}
	}
	fmt.Printf("\n%d/%d CE résolus appariés à une frame offline ; %d ont un dead-state décodé près de b2c.\n",
		found, len(ces), alignedOK)
}
