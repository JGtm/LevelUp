// tmp_killweapon_probe — THROWAWAY (FRONT 3 empirique).
//
// But : autour de kills CONNUS (oracle 000d5950), chercher dans les données film
// une trace de l'arme DU KILL ancrée au kill, par 3 voies :
//
//	(A) FOOTER type-3 (chunk_27) : pour chaque event th=10 dont t == kill_ms,
//	    dumper le bloc de 60 octets qui précède l'end-marker [00 00 2e e0] et
//	    chercher, au-delà de slot(b36)/team(b55)/medal(b59), un champ ressemblant
//	    à un weapon-id (suffixe 0x42c9679f / high-32 catalogué) ou un index.
//	(B) FRAME type-0 du chunk gameplay couvrant le kill : scan des weapon-id 64-bit
//	    catalogués ET du suffixe 0x42c9679f, fenêtré sur les FRAME proches du kill
//	    (us -> ms via 1er FRAME du chunk + start_ms manifest).
//	(C) Bilan : présence d'arme = pool présent (held) vs arme ancrée au kill.
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

// kills d'intérêt (ms) couverts par chunks 00-09. xuid killer/victim de l'oracle.
type kill struct {
	ms       int
	killer   uint64
	victim   uint64
	killerGT string
	v2weapon string
}

var kills = []kill{
	{104995, 2533274823110022, 2535437947245250, "JGtm", "CQS48 Bulldog"},
	{112869, 2533274823110022, 2535444178793711, "JGtm", "CQS48 Bulldog"},
	{149471, 2533274823110022, 2535437947245250, "JGtm", "Disruptor(v2)"},
}

// chunk gameplay couvrant chaque kill (start_ms du manifest) : chunk_NN, startMS.
var chunkStart = []struct {
	idx     int
	startMS int
}{
	{1, 0}, {2, 19995}, {3, 40000}, {4, 60003}, {5, 80006},
	{6, 100010}, {7, 120013}, {8, 140017}, {9, 160021},
}

func load(idx int) []byte {
	raw, err := os.ReadFile(fmt.Sprintf("%s/chunk_%02d.bin", cacheDir, idx))
	if err != nil {
		panic(err)
	}
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

func readByteAtBit(data []byte, bit int) byte {
	if bit < 0 || bit+8 > len(data)*8 {
		return 0
	}
	bi := bit / 8
	off := uint(bit % 8)
	if off == 0 {
		return data[bi]
	}
	return data[bi]<<off | data[bi+1]>>(8-off)
}

func readU64LEAtBit(data []byte, bit int) uint64 {
	var x uint64
	for i := 0; i < 8; i++ {
		x |= uint64(readByteAtBit(data, bit+i*8)) << (uint(i) * 8)
	}
	return x
}

func main() {
	fmt.Println("############ (A) FOOTER type-3 : blocs d'event 60o autour des kills ############")
	probeFooter()
	fmt.Println("\n############ (B) FRAME type-0 : armes catalog/suffixe fenêtrées au kill ############")
	probeFrames()
}

// ---------------------------------------------------------------------------
// (A) Footer : reconstruit la logique scanTh10Events mais en dumpant le bloc
//     complet de 60 octets pour chaque event, pour TOUS les th (pas que 10),
//     et en cherchant un champ arme.
// ---------------------------------------------------------------------------

type evBlock struct {
	xuid    uint64
	th      int
	t       int
	slot    int
	team    int
	medal   int
	blockBS int // bit-start du bloc de 60 octets
}

func probeFooter() {
	data := load(27)
	fmt.Printf("footer chunk_27 décompressé = %d octets\n", len(data))
	// Le footer est un paquet unique type-3 ; on scanne le contenu brut.
	// Marche des paquets pour trouver le payload type-3.
	payload := data
	off := 0
	for off+16 <= len(data) {
		typ := binary.LittleEndian.Uint16(data[off:])
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		if size <= 0 || off+16+size > len(data) {
			break
		}
		if typ == 3 {
			payload = data[off+16 : off+16+size]
			fmt.Printf("payload type-3 = %d octets @off %d\n", size, off+16)
			break
		}
		off += 16 + size
	}

	blocks := scanAllEventBlocks(payload)
	fmt.Printf("%d blocs d'event décodés\n", len(blocks))

	// Catalogue : ensemble des high-32 et l'ID complet.
	for _, k := range kills {
		fmt.Printf("\n--- KILL @%dms  killer=%s(%d) victim=%d  v2=%s ---\n", k.ms, k.killerGT, k.killer, k.victim, k.v2weapon)
		hits := 0
		for _, b := range blocks {
			if abs(b.t-k.ms) <= 60 { // events ~exactement au temps du kill (±60ms)
				role := "?"
				if b.xuid == k.killer {
					role = "KILLER"
				} else if b.xuid == k.victim {
					role = "victim"
				}
				fmt.Printf("  ev t=%d th=%d xuid=%d [%s] slot=%d team=%d medal=%d\n",
					b.t, b.th, b.xuid, role, b.slot, b.team, b.medal)
				dumpBlockWeaponSearch(payload, b)
				hits++
			}
		}
		if hits == 0 {
			fmt.Println("  (aucun bloc d'event à ce temps)")
		}
	}
}

// scanAllEventBlocks : repère chaque XUID (préfixe 0x2d/0x25, suffixe 0xc0),
// trouve l'end-marker [00 00 2e e0], recule de 60 octets et lit th/t/slot/team/medal.
func scanAllEventBlocks(data []byte) []evBlock {
	total := len(data) * 8
	var out []evBlock
	seen := map[int]bool{}
	for ms := 8; ms <= total-8; ms++ {
		if readByteAtBit(data, ms) != 0xc0 {
			continue
		}
		xe := ms - 8
		if xe < 64 {
			continue
		}
		if p := readByteAtBit(data, xe); p != 0x2d && p != 0x25 {
			continue
		}
		xstart := xe - 64
		if seen[xstart] {
			continue
		}
		x := readU64LEAtBit(data, xstart)
		if x <= uint64(2e15) || x >= uint64(3e15) {
			continue
		}
		seen[xstart] = true
		// cherche end-marker
		win := xstart + 20000
		if win > total {
			win = total
		}
		for b := xstart; b <= win-32; b++ {
			if readByteAtBit(data, b) == 0 && readByteAtBit(data, b+8) == 0 &&
				readByteAtBit(data, b+16) == 0x2e && readByteAtBit(data, b+24) == 0xe0 {
				ebs := b - 60*8
				if ebs < xstart {
					break
				}
				out = append(out, evBlock{
					xuid:    x,
					th:      int(readByteAtBit(data, ebs+47*8)),
					t:       int(readByteAtBit(data, ebs+48*8))<<24 | int(readByteAtBit(data, ebs+49*8))<<16 | int(readByteAtBit(data, ebs+50*8))<<8 | int(readByteAtBit(data, ebs+51*8)),
					slot:    int(readByteAtBit(data, ebs+36*8)),
					team:    int(readByteAtBit(data, ebs+55*8)),
					medal:   int(readByteAtBit(data, ebs+59*8)),
					blockBS: ebs,
				})
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].t < out[j].t })
	return out
}

// dumpBlockWeaponSearch : dumpe les 60 octets du bloc + cherche une trace d'arme :
//   - chaque octet du bloc en hex (par offset)
//   - tous les u32 BE/LE à offsets octet alignés == high-32 catalogué ?
//   - suffixe 0x42c9679f présent quelque part dans le bloc (et 32o autour) ?
func dumpBlockWeaponSearch(data []byte, b evBlock) {
	// 60 octets du bloc + 24 octets après l'end-marker (le bloc peut continuer).
	raw := make([]byte, 84)
	for i := range raw {
		raw[i] = readByteAtBit(data, b.blockBS+i*8)
	}
	fmt.Printf("     bloc[0..59]: % x\n", raw[:60])
	fmt.Printf("     bloc[60..83]: % x\n", raw[60:])

	// high-32 catalogue
	highSet := map[uint32]string{}
	for id, name := range analysis.WeaponIDToName {
		highSet[uint32(id>>32)] = name
	}
	// scan u32 BE et LE alignés octet sur 84 octets
	for off := 0; off+4 <= len(raw); off++ {
		be := binary.BigEndian.Uint32(raw[off:])
		le := binary.LittleEndian.Uint32(raw[off:])
		if n, ok := highSet[be]; ok {
			fmt.Printf("     >> high-32 BE @+%d = 0x%08x (%s)\n", off, be, n)
		}
		if n, ok := highSet[le]; ok {
			fmt.Printf("     >> high-32 LE @+%d = 0x%08x (%s)\n", off, le, n)
		}
		if be == 0x42c9679f {
			fmt.Printf("     >> SUFFIXE 0x42c9679f (BE) @+%d\n", off)
		}
		if le == 0x42c9679f {
			fmt.Printf("     >> SUFFIXE 0x42c9679f (LE) @+%d\n", off)
		}
	}
	// scan bit-non-aligné du suffixe sur le bloc
	for bit := 0; bit+32 <= len(raw)*8; bit++ {
		if readU32BEAtBit(raw, bit) == 0x42c9679f {
			fmt.Printf("     >> SUFFIXE 0x42c9679f (bit BE) @bit+%d (octet %d.%d)\n", bit, bit/8, bit%8)
		}
	}
}

func readU32BEAtBit(data []byte, bit int) uint32 {
	var x uint32
	for i := 0; i < 32; i++ {
		x = x<<1 | uint32((readByteAtBit(data, (bit/8)*8+0)>>0)&0) // placeholder
		_ = i
		break
	}
	// recompute properly
	x = 0
	for i := 0; i < 32; i++ {
		bi := (bit + i) / 8
		if bi >= len(data) {
			break
		}
		off := 7 - ((bit + i) % 8)
		x = x<<1 | uint32((data[bi]>>uint(off))&1)
	}
	return x
}

// ---------------------------------------------------------------------------
// (B) FRAME : scan des weapon-id catalogués (64-bit complet) ET du suffixe
//     0x42c9679f dans les FRAME proches du kill, fenêtré ±2s.
// ---------------------------------------------------------------------------

func probeFrames() {
	fullSet := map[uint64]string{}
	for id, name := range analysis.WeaponIDToName {
		fullSet[id] = name
	}

	for _, k := range kills {
		ci := chunkForMS(k.ms)
		if ci < 0 {
			fmt.Printf("KILL @%d : pas de chunk gameplay\n", k.ms)
			continue
		}
		data := load(chunkStart[ci].idx)
		startMS := chunkStart[ci].startMS
		frames := walkFrames(data)
		if len(frames) == 0 {
			fmt.Printf("KILL @%d chunk_%02d : aucun FRAME\n", k.ms, chunkStart[ci].idx)
			continue
		}
		first := frames[0].us
		fmt.Printf("\n--- KILL @%dms -> chunk_%02d (start %dms, %d FRAME) ---\n", k.ms, chunkStart[ci].idx, startMS, len(frames))
		// pour chaque FRAME, ms = startMS + (us-first)/1000 ; fenêtre ±2000ms
		near := 0
		for i, f := range frames {
			if i == 0 {
				continue
			}
			ms := startMS + int(int64(f.us-first)/1000)
			if abs(ms-k.ms) > 2000 {
				continue
			}
			near++
			pl := data[f.off : f.off+f.size]
			catHits := scanCatalogWeapons(pl, fullSet)
			sufBits := countSuffix(pl)
			if len(catHits) > 0 || sufBits > 0 {
				fmt.Printf("  FRAME #%d ms=%d size=%d : armes=%v suffixe0x42c9679f×%d\n", i, ms, f.size, catHits, sufBits)
			}
		}
		fmt.Printf("  (%d FRAME dans ±2s du kill)\n", near)
	}
}

func chunkForMS(ms int) int {
	for i := len(chunkStart) - 1; i >= 0; i-- {
		if ms >= chunkStart[i].startMS {
			return i
		}
	}
	return -1
}

type framePacket struct {
	us   uint64
	size int
	off  int
}

func walkFrames(data []byte) []framePacket {
	var out []framePacket
	off := 0
	for off+16 <= len(data) {
		typ := binary.LittleEndian.Uint16(data[off:])
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		us := binary.LittleEndian.Uint64(data[off+8:])
		if size < 0 || size > len(data) {
			break
		}
		if typ == 0 && off+16+size <= len(data) {
			out = append(out, framePacket{us: us, size: size, off: off + 16})
		}
		off += 16 + size
		if typ == 7 {
			break
		}
	}
	return out
}

// scanCatalogWeapons : weapon-id 64-bit complet (MSB-first) à tous alignements bit.
func scanCatalogWeapons(pl []byte, set map[uint64]string) []string {
	found := map[string]bool{}
	total := len(pl) * 8
	for start := 0; start+64 <= total; start++ {
		var v uint64
		for i := 0; i < 64; i++ {
			bi := (start + i) / 8
			off := 7 - ((start + i) % 8)
			v = v<<1 | uint64((pl[bi]>>uint(off))&1)
		}
		if n, ok := set[v]; ok {
			found[n] = true
		}
	}
	var out []string
	for n := range found {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// countSuffix : nombre d'occurrences bit-alignées du suffixe 0x42c9679f.
func countSuffix(pl []byte) int {
	c := 0
	total := len(pl) * 8
	for start := 0; start+32 <= total; start++ {
		var v uint32
		for i := 0; i < 32; i++ {
			bi := (start + i) / 8
			off := 7 - ((start + i) % 8)
			v = v<<1 | uint32((pl[bi]>>uint(off))&1)
		}
		if v == 0x42c9679f {
			c++
		}
	}
	return c
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
