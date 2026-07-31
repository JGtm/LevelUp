// tmp_dmgprobe — lane VICTIME + coup-fatal du record 0xd2 (déser FUN_14080c1f8).
//
// Cmd DÉDIÉE (ne touche ni tmp_kwval, ni tmp_acurtis, ni tmp_dmgamount qu'un agent parallèle édite).
// Objectif : PROUVER empiriquement, sur 000d5950, où sont (a) le MONTANT et (b) la VICTIME dans un
// record de dégât 0xd2, à partir du modèle Ghidra suivant (film, param_5==0, base=24) :
//
//	préambule -> attaquant (readOpt5, +0xc) -> famille (+0x10) -> variant (+0x14, const 0x42C9679F)
//	-> (chemin long) FUN_14080cc68 -> (f8, s34)
//	-> TABLEAU A : s34 sections, par entrée [R2 tag][R1 bool][R32 magnitude] (film : 32b bruts ;
//	   live : FUN_1406d3140 = déquantificateur table-de-plage ~13b -> le 32b est un SCALAIRE quantifié)
//	-> TABLEAU B : f8 impacts (FUN_140c1e924 = vecteur 3 comp), réfère A par index.
//
// Tests :
//
//	V1  coup fatal = tueur : dans un paquet 0xd2 FATAL, record.attacker == tueur du kill-event embarqué
//	    (FUN_14104bd08 : [victime R5][tueur R5]). Preuve que la source de dégât fatale = le tueur.
//	V4  la VICTIME est-elle un champ du corps ? Balayage de tous les champs candidats (readOpt5 à chaque
//	    bit, en-tête 8b, tag s34) vs la victime VRAIE (kill-event). Aucune position ~100%% stable =>
//	    la victime n'est PAS dans le corps du record (elle est l'entité ECS porteuse / le kill-event).
//	AMT structure du slot magnitude (dump brut) : confirme la LOCALISATION (offset), sans surinterpréter
//	    l'encodage exact (float IEEE échoue -> point-fixe quantifié, cf. tmp_dmgamount).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_dmgprobe [filmID=000d5950]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
)

const root = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
const dmgBase = 24
const sfx = uint32(0x42c9679f)

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

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

type br struct {
	pl  []byte
	bp  int
	bad bool
}

func (r *br) R(n int) uint64 {
	if r.bp+n > len(r.pl)*8 {
		r.bad = true
	}
	v := bitsAt(r.pl, r.bp, n)
	r.bp += n
	return v
}
func (r *br) g1() int { return int(r.R(1)) }

// dmgRec : préambule + tableau A + positions bit des slots (pour la traçabilité victime/montant).
type dmgRec struct {
	att        int
	family     uint32
	variant    uint32
	out0, out1 int
	s34, f8    int
	arrStart   int      // bit de départ du tableau A
	rawAmts    []uint32 // slot 32b (magnitude) par section
	ampPos     []int    // bit de départ du slot 32b de chaque section
	ok         bool
}

func readCC68(r *br) (int, int) {
	if r.g1() == 1 {
		return 0, 0
	}
	f8 := 1
	if r.g1() == 0 {
		f8 = int(r.R(4))
	}
	if r.g1() == 1 {
		return f8, 0
	}
	if r.g1() == 0 {
		return f8, int(r.R(4))
	}
	return f8, 1
}

func parseDmg(pl []byte) dmgRec {
	var rec dmgRec
	rec.att = -1
	rec.family = 0xffffffff
	r := &br{pl: pl, bp: dmgBase}
	rec.out0 = r.g1()
	rec.out1 = r.g1()
	r.R(8)
	if r.g1() == 0 {
		rec.att = int(r.R(5))
	}
	if r.g1() == 0 {
		r.R(2)
	}
	if r.g1() == 1 {
		rec.family = uint32(r.R(32))
	}
	rec.variant = uint32(r.R(32))
	r.R(1)
	r.R(1)
	f2dd := 0
	if rec.out1 == 1 {
		r.R(1)
		f2dd = r.g1()
	}
	if f2dd != 0 {
		if r.g1() == 1 {
			r.R(10)
		}
	}
	if rec.out0 == 1 {
		rec.ok = !r.bad
		return rec
	}
	rec.f8, rec.s34 = readCC68(r)
	rec.arrStart = r.bp
	if rec.s34 < 0 || rec.s34 >= 64 {
		return rec
	}
	for i := 0; i < rec.s34 && !r.bad; i++ {
		r.R(2)
		r.R(1)
		rec.ampPos = append(rec.ampPos, r.bp)
		rec.rawAmts = append(rec.rawAmts, uint32(r.R(32)))
	}
	rec.ok = !r.bad
	return rec
}

// ---- kill-event embarqué ----

func keReadOpt(pl []byte, bp int) (int, int) {
	if bp < 0 || bp>>3 >= len(pl) {
		return -2, bp
	}
	if bitsAt(pl, bp, 1) == 0 {
		return int(bitsAt(pl, bp+1, 5)), bp + 6
	}
	return -1, bp + 1
}

func validKE(pl []byte, b int) bool {
	v, b2 := keReadOpt(pl, b)
	k, b3 := keReadOpt(pl, b2)
	if v < 0 || k < 0 || v >= 16 || k >= 16 || v == k {
		return false
	}
	if bitsAt(pl, b3, 32) > 0xffff {
		return false
	}
	a, _ := keReadOpt(pl, b3+33)
	return a == -1 || (a >= 0 && a < 16)
}

func decodeKE(pl []byte, c int) (vic, kil int) {
	v, b2 := keReadOpt(pl, c)
	k, _ := keReadOpt(pl, b2)
	return v, k
}

func locateKE(pl []byte) int {
	hi := len(pl) * 8
	for x := 140; x+17 <= hi; x++ {
		if bitsAt(pl, x, 10) == 0x2A8 && validKE(pl, x+10) {
			return x + 10
		}
	}
	return -1
}

// ---- collecte ----

type frame struct {
	marker byte
	pl     []byte
	ts     uint64
}

func (f frame) sz() int { return len(f.pl) }

func collect(m string) []frame {
	cache := root + "/" + m
	var out []frame
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := make([]byte, sz)
			copy(pl, d[off+16:off+16+sz])
			off += 16 + sz
			if typ != 0 || len(pl) == 0 {
				continue
			}
			out = append(out, frame{pl[0], pl, ts})
		}
	}
	return out
}

func pctf(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) * 100 / float64(b)
}

func main() {
	m := "000d5950"
	if len(os.Args) > 1 {
		m = os.Args[1]
	}
	frames := collect(m)
	var d2 []frame
	for _, f := range frames {
		if f.marker == 0xd2 {
			d2 = append(d2, f)
		}
	}
	fmt.Printf("=== tmp_dmgprobe %s : %d frames, %d records 0xd2 ===\n", m, len(frames), len(d2))

	// (0) structure : s34/f8, alignement variant.
	nSfx, s34h := 0, map[int]int{}
	for _, f := range d2 {
		rec := parseDmg(f.pl)
		if rec.variant == sfx {
			nSfx++
		}
		s34h[rec.s34]++
	}
	fmt.Printf("firearm(variant==0x42C9679F)=%d/%d | s34: %v\n", nSfx, len(d2), s34h)

	// (1) V1 — coup fatal = tueur (kill-event embarqué). Aussi : la magnitude fatale est-elle non nulle ?
	nFatal, atkEqKil, atkEqVic := 0, 0, 0
	nSecFatal, nSecNonZero := 0, 0
	shown := 0
	for _, f := range d2 {
		if f.sz() < 700 {
			continue
		}
		cur := locateKE(f.pl)
		if cur < 0 {
			continue
		}
		vic, kil := decodeKE(f.pl, cur)
		if vic < 0 || kil < 0 {
			continue
		}
		rec := parseDmg(f.pl)
		nFatal++
		if rec.att == kil {
			atkEqKil++
		}
		if rec.att == vic {
			atkEqVic++
		}
		for _, raw := range rec.rawAmts {
			nSecFatal++
			if raw != 0 {
				nSecNonZero++
			}
		}
		if shown < 16 {
			fmt.Printf("  FATAL cur=%d tueur=%d victime=%d | att=%d(=tueur?%v =victime?%v) fam=%08X s34=%d raw=%v\n",
				cur, kil, vic, rec.att, rec.att == kil, rec.att == vic, rec.family, rec.s34, rec.rawAmts)
			shown++
		}
	}
	fmt.Printf("\n--- V1 coup fatal ---\n")
	fmt.Printf(">>> record.attacker == TUEUR (kill-event) : %d/%d = %.0f%%\n", atkEqKil, nFatal, pctf(atkEqKil, nFatal))
	fmt.Printf("    record.attacker == VICTIME (contrôle, doit être ~0%%) : %d/%d = %.0f%%\n", atkEqVic, nFatal, pctf(atkEqVic, nFatal))
	fmt.Printf("    sections de dégât des records fataux : %d, dont slot 32b != 0 : %d\n", nSecFatal, nSecNonZero)

	// (2) V4 — la VICTIME est-elle un champ du corps ? Balayage.
	nKE := 0
	hHdr, hHdr3, hTag := 0, 0, 0
	roHit := map[int]int{} // readOpt5 à position absolue -> #coïncidences victime
	roRel := map[int]int{} // readOpt5 relatif à arrStart -> #coïncidences victime
	for _, f := range d2 {
		if f.sz() < 700 {
			continue
		}
		cur := locateKE(f.pl)
		if cur < 0 {
			continue
		}
		vic, kil := decodeKE(f.pl, cur)
		if vic < 0 || kil < 0 {
			continue
		}
		rec := parseDmg(f.pl)
		nKE++
		if int(bitsAt(f.pl, 16, 8)) == vic {
			hHdr++
		}
		if int(bitsAt(f.pl, 16, 8))&7 == vic {
			hHdr3++
		}
		// tag R2 d'une section (2 bits, 0..3 : ne peut pas coder 0..7 mais on teste)
		for _, ap := range rec.ampPos {
			if int(bitsAt(f.pl, ap-3, 2)) == vic {
				hTag++
				break
			}
		}
		// balayage readOpt5 absolu (bit 16..cur-6) et relatif à arrStart
		lim := cur - 6
		if lim > 500 {
			lim = 500
		}
		for o := 16; o < lim; o++ {
			if v, _ := keReadOpt(f.pl, o); v == vic {
				roHit[o]++
			}
		}
		if rec.arrStart > 0 {
			for d := -40; d <= 200; d++ {
				o := rec.arrStart + d
				if o < 0 || o+6 > len(f.pl)*8 {
					continue
				}
				if v, _ := keReadOpt(f.pl, o); v == vic {
					roRel[d]++
				}
			}
		}
	}
	fmt.Printf("\n--- V4 recherche VICTIME (n=%d paquets fataux) ---\n", nKE)
	fmt.Printf("  en-tête 8b [16,24) == victime : %d (%.0f%%) | &7 == victime : %d (%.0f%%) | tag R2 section == victime : %d (%.0f%%)\n",
		hHdr, pctf(hHdr, nKE), hHdr3, pctf(hHdr3, nKE), hTag, pctf(hTag, nKE))
	fmt.Printf("  readOpt5 ABSOLU — meilleures positions (pos:coïncidences/%d) : %s\n", nKE, top(roHit, 10))
	fmt.Printf("  readOpt5 REL. arrStart — meilleurs décalages (Δ:coïncidences/%d) : %s\n", nKE, top(roRel, 10))
	fmt.Printf("  => une VRAIE victime-field donnerait ~%d (=100%%) sur UNE position STABLE ; sinon = hasard (~nKE/8≈%d).\n",
		nKE, nKE/8)

	// (3) AMT — localisation du slot magnitude (dump brut, sans surinterpréter l'encodage).
	fmt.Printf("\n--- AMT slot magnitude (32b @ ampPos, records s34>=1) ---\n")
	rawHist := map[uint32]int{}
	var fixed []float64 // interprétation point-fixe (u>>16)/256, indicative
	shown = 0
	for _, f := range d2 {
		rec := parseDmg(f.pl)
		if rec.s34 == 0 || !rec.ok {
			continue
		}
		for _, raw := range rec.rawAmts {
			rawHist[raw]++
			fixed = append(fixed, float64(raw>>16)/256.0)
		}
		if shown < 10 {
			fmt.Printf("  att=%2d fam=%08X arrStart=%d s34=%d raw=%08X floatBE=%.3g floatLE=%.3g (u>>16)/256=%.2f\n",
				rec.att, rec.family, rec.arrStart, rec.s34, rec.rawAmts[0],
				math.Float32frombits(rec.rawAmts[0]),
				math.Float32frombits(byterev(rec.rawAmts[0])),
				float64(rec.rawAmts[0]>>16)/256.0)
			shown++
		}
	}
	if len(fixed) > 0 {
		sort.Float64s(fixed)
		fmt.Printf("  (u>>16)/256 sur %d slots : min=%.2f méd=%.2f max=%.2f (indicatif — encodage exact non tranché)\n",
			len(fixed), fixed[0], fixed[len(fixed)/2], fixed[len(fixed)-1])
	}
	fmt.Printf("  valeurs brutes distinctes : %d (top: %s)\n", len(rawHist), topU32(rawHist, 6))
}

func byterev(v uint32) uint32 {
	return v>>24 | (v&0xff0000)>>8 | (v&0xff00)<<8 | v<<24
}

func top(h map[int]int, n int) string {
	type kv struct{ k, v int }
	var a []kv
	for k, v := range h {
		a = append(a, kv{k, v})
	}
	sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
	s := ""
	for i := 0; i < n && i < len(a); i++ {
		s += fmt.Sprintf("%d:%d ", a[i].k, a[i].v)
	}
	return s
}

func topU32(h map[uint32]int, n int) string {
	type kv struct {
		k uint32
		v int
	}
	var a []kv
	for k, v := range h {
		a = append(a, kv{k, v})
	}
	sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
	s := ""
	for i := 0; i < n && i < len(a); i++ {
		s += fmt.Sprintf("%08X:%d ", a[i].k, a[i].v)
	}
	return s
}
