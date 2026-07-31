// tmp_killframe_anatomy — THROWAWAY (FRONT 3) : dissèque la "kill-FRAME"
// volumineuse. Hypothèse : c'est un burst qui re-transmet, au kill, le record du
// tueur (avec son arme du kill figée). On veut :
//  1. dumper l'en-tête + les ~64 premiers octets (l'arme @bit44 est dans la tête).
//  2. localiser le suffixe 0x42c9679f et l'arme high-32 catalog à TOUTES positions.
//  3. chercher le xuid/gamertag du tueur & de la victime dans la FRAME (ancrage).
//  4. pour @149471 (0 arme 64-align) : chercher le suffixe + scanner le high-32
//     catalog seul (l'arme peut être encodée high-32 isolé sans low-32 contigu).
package main

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

const cacheDir = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

type kill struct {
	ms        int
	killerGT  string
	killerX   uint64
	victimX   uint64
	chunkIdx  int
	chunkStMS int
}

var kills = []kill{
	{104995, "JGtm", 2533274823110022, 2535437947245250, 6, 100010},
	{112869, "JGtm", 2533274823110022, 2535444178793711, 6, 100010},
	{149471, "JGtm", 2533274823110022, 2535437947245250, 8, 140017},
}

func load(idx int) []byte {
	raw, _ := os.ReadFile(fmt.Sprintf("%s/chunk_%02d.bin", cacheDir, idx))
	if len(raw) >= 2 && raw[0] == 0x78 {
		zr, e := zlib.NewReader(bytes.NewReader(raw))
		if e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

type fp struct {
	us   uint64
	size int
	off  int
}

func walkFrames(data []byte) []fp {
	var out []fp
	off := 0
	for off+16 <= len(data) {
		typ := binary.LittleEndian.Uint16(data[off:])
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		us := binary.LittleEndian.Uint64(data[off+8:])
		if size < 0 || size > len(data) {
			break
		}
		if typ == 0 && off+16+size <= len(data) {
			out = append(out, fp{us, size, off + 16})
		}
		off += 16 + size
		if typ == 7 {
			break
		}
	}
	return out
}

func readU64BEAtBit(pl []byte, bit int) uint64 {
	var v uint64
	for i := 0; i < 64; i++ {
		bi := (bit + i) / 8
		if bi >= len(pl) {
			break
		}
		off := 7 - ((bit + i) % 8)
		v = v<<1 | uint64((pl[bi]>>uint(off))&1)
	}
	return v
}
func readU32BEAtBit(pl []byte, bit int) uint32 {
	var v uint32
	for i := 0; i < 32; i++ {
		bi := (bit + i) / 8
		if bi >= len(pl) {
			break
		}
		off := 7 - ((bit + i) % 8)
		v = v<<1 | uint32((pl[bi]>>uint(off))&1)
	}
	return v
}

func main() {
	full := map[uint64]string{}
	high := map[uint32]string{}
	for id, name := range analysis.WeaponIDToName {
		full[id] = name
		high[uint32(id>>32)] = name
	}

	for _, k := range kills {
		data := load(k.chunkIdx)
		frames := walkFrames(data)
		first := frames[0].us
		// kill-FRAME = la plus proche
		bestI, bestD := -1, 1<<30
		for i, f := range frames {
			ms := k.chunkStMS + int(int64(f.us-first)/1000)
			d := ms - k.ms
			if d < 0 {
				d = -d
			}
			if d < bestD {
				bestD, bestI = d, i
			}
		}
		f := frames[bestI]
		pl := data[f.off : f.off+f.size]
		fmt.Printf("\n========= KILL @%dms : kill-FRAME #%d size=%d Δ=%dms =========\n", k.ms, bestI, f.size, bestD)
		dump := 80
		if dump > len(pl) {
			dump = len(pl)
		}
		fmt.Printf("  head[0..%d]: % x\n", dump, pl[:dump])

		// 1) full-id 64-bit (toutes positions bit)
		fmt.Println("  --- weapon 64-bit complet (catalog) ---")
		for bit := 0; bit+64 <= len(pl)*8; bit++ {
			v := readU64BEAtBit(pl, bit)
			if n, ok := full[v]; ok {
				fmt.Printf("     %s  @bit%d (octet %d.%d)  id=0x%016x\n", n, bit, bit/8, bit%8, v)
			}
		}
		// 2) high-32 catalog isolé + suffixe (toutes positions bit)
		fmt.Println("  --- high-32 catalog isolé / suffixe 0x42c9679f ---")
		hi := 0
		suf := 0
		for bit := 0; bit+32 <= len(pl)*8; bit++ {
			v := readU32BEAtBit(pl, bit)
			if n, ok := high[v]; ok {
				if hi < 30 {
					fmt.Printf("     high32 %s @bit%d (%d.%d) =0x%08x\n", n, bit, bit/8, bit%8, v)
				}
				hi++
			}
			if v == 0x42c9679f {
				if suf < 30 {
					fmt.Printf("     SUFFIX @bit%d (%d.%d)\n", bit, bit/8, bit%8)
				}
				suf++
			}
		}
		fmt.Printf("     [total high32=%d suffixe=%d]\n", hi, suf)

		// 3) xuid killer/victim dans la FRAME (LE) — ancrage joueur
		fmt.Println("  --- xuid killer/victim dans la FRAME (LE, octet-aligné) ---")
		for off := 0; off+8 <= len(pl); off++ {
			le := binary.LittleEndian.Uint64(pl[off:])
			if le == k.killerX {
				fmt.Printf("     KILLER xuid @octet %d\n", off)
			}
			if le == k.victimX {
				fmt.Printf("     VICTIM xuid @octet %d\n", off)
			}
		}
		// gamertag UTF-16LE du killer
		gt := utf16le(k.killerGT)
		if idx := bytes.Index(pl, gt); idx >= 0 {
			fmt.Printf("     KILLER gamertag UTF16 @octet %d\n", idx)
		}
	}
	_ = sort.Ints
}

func utf16le(s string) []byte {
	out := make([]byte, 0, len(s)*2)
	for _, r := range s {
		out = append(out, byte(r), byte(r>>8))
	}
	return out
}
