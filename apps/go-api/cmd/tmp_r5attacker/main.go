// tmp_r5attacker — THROWAWAY. Hypothese (Ghidra FUN_14080c1f8) : le deser lit R5 (+0x0c,
// FUN_1407f2034 -> FUN_1407f2058 = 5 bits) AVANT le slot/cause (+0x08, FUN_1406d00ec).
// tmp_dmgattacker lisait slot AVANT r5 -> son "R5" etait desaligne (chevauche le slot).
//
// FUN_140495860 resout ce R5 en index de table d'entites ; les 8 joueurs occupent les
// slots 0..7 (handles live 0xEC500000+idx*0x10002). DONC R5 = INDEX ATTAQUANT.
//
// Test : extraire le VRAI R5 (5 bits au bit 36, lu en premier), montrer sa distribution.
//   - ~8 valeurs ~equilibrees -> ATTAQUANT (joueur). Le firearm-par-kill devient offline.
//   - peu de valeurs clusterisees par arme -> propriete d'arme (hypothese fausse).
//
// Sanity : la famille doit TOUJOURS decoder avec le nouvel ordre (budget de bits identique).
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
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)
const deserStartBit = 36
const variantSuffix = uint32(0x42c9679f)

var h32name = map[uint32]string{}

func build() {
	for id, n := range analysis.WeaponIDToName {
		h32name[uint32(id>>32)] = n
	}
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

type pkt struct {
	typ     uint16
	ts      uint64
	payload []byte
}

func allType0Dmg() []pkt {
	var all []pkt
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		if len(d) == 0 {
			continue
		}
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			size := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if size <= 0 || off+16+size > len(d) {
				break
			}
			pl := d[off+16 : off+16+size]
			if typ == 0 && len(pl) > 0 && pl[0] == 0xd2 {
				all = append(all, pkt{typ, ts, pl})
			}
			off += 16 + size
		}
	}
	return all
}

func tsToMs(ts uint64) int { return int((int64(ts) - int64(t0Us)) / 1000) }

type rec struct {
	tms       int
	r5proj    uint64 // ordre tmp_dmgattacker (slot AVANT r5) — desaligne
	r5corr    uint64 // ordre deser CORRIGE (r5 AVANT slot) — vrai attaquant
	slot      uint64
	famProj   string
	famCorr   string
	famCorrOK bool
}

// decodeProj = ancien ordre (slot, r5, fam).
func decodeProj(d []byte) (uint64, uint64, string) {
	br := filmdec.NewBitReader(d)
	br.Skip(deserStartBit)
	var slot uint64
	if br.ReadBit() {
		slot = 0
	} else {
		slot = 1 + br.ReadBits(2)
	}
	r5 := br.ReadBits(5)
	fam32 := uint32(br.ReadBits(32))
	return slot, r5, h32name[fam32]
}

// decodeCorr = ordre deser (r5 AVANT slot). Le budget de bits avant fam32 est identique
// (5 + slot_bits), donc fam32 doit decoder pareil SI l'hypothese d'ordre est juste.
func decodeCorr(d []byte) (uint64, uint64, string, uint32) {
	br := filmdec.NewBitReader(d)
	br.Skip(deserStartBit)
	r5 := br.ReadBits(5)
	var slot uint64
	if br.ReadBit() {
		slot = 0
	} else {
		slot = 1 + br.ReadBits(2)
	}
	fam32 := uint32(br.ReadBits(32))
	low := uint32(br.ReadBits(32))
	_ = low
	return r5, slot, h32name[fam32], fam32
}

func main() {
	build()
	pkts := allType0Dmg()
	var recs []rec
	for _, p := range pkts {
		slotP, r5P, famP := decodeProj(p.payload)
		r5C, slotC, famC, _ := decodeCorr(p.payload)
		_ = slotP
		recs = append(recs, rec{
			tms: tsToMs(p.ts), r5proj: r5P, r5corr: r5C, slot: slotC,
			famProj: famP, famCorr: famC, famCorrOK: famC != "",
		})
	}
	// on garde ceux dont la famille decode (les vrais records d'arme).
	var ok []rec
	for _, r := range recs {
		if r.famProj != "" { // discriminant identique a tmp_dmgattacker
			ok = append(ok, r)
		}
	}
	fmt.Printf("=== %d paquets 0xd2 ; %d avec famille catalogue ===\n\n", len(pkts), len(ok))

	// SANITY : la famille decode-t-elle toujours avec l'ordre corrige ?
	famCorrOK := 0
	for _, r := range ok {
		if r.famCorr != "" {
			famCorrOK++
		}
	}
	fmt.Printf("SANITY ordre corrige : %d/%d records ont une famille catalogue (doit ~= %d)\n",
		famCorrOK, len(ok), len(ok))
	if famCorrOK < len(ok)*9/10 {
		fmt.Println("  /!\\ l'ordre corrige casse la famille -> mon hypothese d'ordre est fausse, STOP.")
	} else {
		fmt.Println("  OK : la famille decode dans les deux ordres (budget de bits identique).")
	}

	// DISTRIBUTION : R5 corrige (vrai attaquant) vs R5 projet (desaligne).
	fmt.Println("\n=== R5 CORRIGE (bit 36, lu en premier = attaquant ?) ===")
	dist(ok, func(r rec) uint64 { return r.r5corr })
	fmt.Println("\n=== R5 PROJET (desaligne, slot lu avant) — pour comparaison ===")
	dist(ok, func(r rec) uint64 { return r.r5proj })

	// VERDICT : combien de valeurs distinctes ? ~8 = joueurs.
	cset := map[uint64]bool{}
	for _, r := range ok {
		cset[r.r5corr] = true
	}
	fmt.Printf("\n=== VERDICT : R5 corrige prend %d valeurs distinctes (8 attendu si = attaquant) ===\n", len(cset))

	// Croisement R5corr x famille : si une meme famille porte plusieurs R5 => R5 = joueur (pas arme).
	fmt.Println("\n=== R5 corrige x famille (si meme arme -> plusieurs R5, alors R5 = joueur) ===")
	famR5 := map[string]map[uint64]int{}
	for _, r := range ok {
		if famR5[r.famCorr] == nil {
			famR5[r.famCorr] = map[uint64]int{}
		}
		famR5[r.famCorr][r.r5corr]++
	}
	var fams []string
	for f := range famR5 {
		if f != "" {
			fams = append(fams, f)
		}
	}
	sort.Strings(fams)
	for _, f := range fams {
		fmt.Printf("  %-24s R5corr: %s\n", f, kc(famR5[f]))
	}
}

func dist(recs []rec, get func(rec) uint64) {
	m := map[uint64]int{}
	for _, r := range recs {
		m[get(r)]++
	}
	var ks []uint64
	for k := range m {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i] < ks[j] })
	for _, k := range ks {
		bar := ""
		for i := 0; i < m[k]/3; i++ {
			bar += "#"
		}
		fmt.Printf("  R5=%2d (0b%05b) x%-4d %s\n", k, k, m[k], bar)
	}
}

func kc(m map[uint64]int) string {
	var ks []uint64
	for k := range m {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i] < ks[j] })
	s := ""
	for _, k := range ks {
		s += fmt.Sprintf("%d:%d ", k, m[k])
	}
	return s
}
