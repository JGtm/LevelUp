// tmp_deathchain — THROWAWAY : teste l'hypothèse "un packet typ==0 = PLUSIEURS frames
// (ticks) concaténées". Au lieu de décoder UNE frame par packet (tmp_cematch), boucle
// DecodeFrameRecords sur le même BitReader/World jusqu'à épuiser le payload. Pour chaque
// record CE dead-state résolu, localise le packet contenant son fingerprint, puis rejoue
// les sous-frames du payload depuis bit 0 et reporte le PREMIER désync rencontré sur le
// chemin [0, expBit] : c'est le bloqueur exact de la chaîne de mort.
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
		dumpPath = p
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

// decodeAllFrames boucle les sous-frames d'un payload, renvoie : nb de frames décodées,
// bit final atteint, et (si désync) la description du 1er désync + son bit.
func decodeAllFrames(payload []byte, w *filmdec.World, stopBit int, reg *filmdec.Registry) (nFrames, endBit int, desync string) {
	br := filmdec.NewBitReader(payload)
	total := len(payload) * 8
	for br.BitPos() < total-8 {
		startFrame := br.BitPos()
		recs, derr := filmdec.DecodeFrameRecords(br, w, calCfg)
		nFrames++
		endBit = br.BitPos()
		if derr != nil {
			// localise le composant du désync sur le dernier record
			if len(recs) > 0 {
				last := recs[len(recs)-1]
				// slot non-bindé ? decodeDelta met DesyncAt=0 + TypeIndex=0 quand ArchetypeForSlot échoue.
				if _, bound := w.ArchetypeForSlot(last.Slot); !bound && last.Type == 3 {
					kind := "objet-monde"
					if last.Slot >= 512 && last.Slot <= 519 {
						kind = "BIPED-JOUEUR"
					}
					desync = fmt.Sprintf("frame#%d@bit%d SLOT-NON-BINDÉ slot=%d (%s)", nFrames, startFrame, last.Slot, kind)
				} else if arch, ok := reg.Archetype(int(last.TypeIndex)); ok && last.DesyncAt >= 0 && last.DesyncAt < len(arch.Components) {
					desync = fmt.Sprintf("frame#%d@bit%d ti=%d i%d %s slot=%d", nFrames, startFrame, last.TypeIndex, last.DesyncAt, arch.Components[last.DesyncAt], last.Slot)
				} else {
					desync = fmt.Sprintf("frame#%d@bit%d %v", nFrames, startFrame, derr)
				}
			}
			return
		}
		// une frame vide (recEnd immédiat) => fin réelle du payload (padding)
		if len(recs) == 0 {
			return
		}
		if stopBit > 0 && endBit >= stopBit {
			return
		}
	}
	return
}

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
	vic, kil uint32
	b2c      int
	fp       []byte
}

func main() {
	prefix := "000d5950"
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
			continue
		}
		ces = append(ces, ceRec{vic: u(0x0c, b), kil: u(0x10, b), b2c: int(u(0x1c, b)), fp: append([]byte(nil), b[0x40:0x50]...)})
	}
	reg, _ := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	w := freshWorld(reg)
	matched := map[int]bool{}

	// Rejoue tous les packets en MODE MULTI-FRAME (boucle sous-frames) pour garder le World
	// correct, et quand un packet contient un fp CE, l'analyse en profondeur.
	for idx := 1; idx <= 26; idx++ {
		for _, pk := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx)), idx) {
			var hitCE = -1
			var hitP int
			for ci, ce := range ces {
				if matched[ci] {
					continue
				}
				p := bytes.Index(pk.payload, ce.fp)
				if p < 0 || p*8+ce.b2c > len(pk.payload)*8 {
					continue
				}
				hitCE, hitP = ci, p
				break
			}
			if hitCE >= 0 {
				ce := ces[hitCE]
				matched[hitCE] = true
				expBit := hitP*8 + ce.b2c
				// World snapshot inutile : on rejoue le payload sur une COPIE du World ?
				// Non : DecodeFrameRecords mute le World. On accepte la mutation (le packet
				// suivant repartira d'un World légèrement avancé) — diagnostic only.
				nF, endB, ds := decodeAllFrames(pk.payload, w, expBit, reg)
				status := "ATTEINT"
				if endB < expBit {
					status = "BLOQUÉ"
				}
				fmt.Printf("CE vic=idx%d kil=idx%d expBit=%d | %s : %d sous-frames, endBit=%d, désync=[%s]\n",
					tryIdx(ce.vic), tryIdx(ce.kil), expBit, status, nF, endB, ds)
			} else {
				// packet normal : avance le World en multi-frame
				decodeAllFrames(pk.payload, w, 0, reg)
			}
		}
	}
	fmt.Printf("\n%d/%d records CE appariés.\n", len(matched), len(ces))
}
