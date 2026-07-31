// tmp_dmgamount — EXTRACTION + VALIDATION du MONTANT de dégât (et recherche de la VICTIME)
// dans les records 0xd2 du déser FUN_14080c1f8 (RVA 0x80C1F8).
//
// Modèle Ghidra (film, param_5==0), flux de bits démarrant au bit `base`=24 du payload
// (16b marqueur + 8b en-tête parent, confirmé par cmd/tmp_kwval/parsePreamble validé) :
//
//	R1  out0                          (bit 24)
//	R1  out1c                         (bit 25)
//	R8  fixe (FUN_141fcf670)          (bit 26..33)
//	ATTAQUANT ECS_ReadEntityRefIndex5 : g1 ; si 0 -> R5      (gate au bit 34)
//	dmgIndex FUN_1406d00ec           : g1 ; si 0 -> R2
//	FAMILLE  FUN_14080d69c           : g1 ; si 1 -> R32
//	VARIANT  FUN_14080dec4           : R32 (const 0x42C9679F)
//	R1  out[0x1d] ; R1 out[2]
//	si out1c==1 : R1(0x2dc) ; f2dd=R1     sinon f2dd=0
//	si f2dd!=0  : g1 ; si 1 -> R10 (FUN_1431a0abc)
//	si out0==1  : CHEMIN COURT (pas de tableau) -> stop
//	FUN_14080cc68 -> (f8=out[0xf8], s34=out[0x34])
//	--- TABLEAU A (les HITS), s34 éléments, stride 0xc (struct+0x38) :
//	    R2 tag ; R1 bool ; R32 = MONTANT (film : float32 IEEE brut ; live : FUN_1406d3140 déquant)
//	--- TABLEAU B, f8 éléments : positions d'impact (FUN_140c1e924 = vecteur), réf. A par index.
//
// CONCLUSION VICTIME (Ghidra) : FUN_14080c1f8 est l'entrée #2 d'une table de sérialiseurs ECS
// (fptr @ 0x143d0ad08). Aucun champ index-joueur victime dans le payload : la victime = l'ENTITÉ
// qui porte le composant (liée par le framework ECS), et pour un paquet FATAL elle est dans le
// kill-event embarqué (FUN_14104bd08 : [victime R5][tueur R5]). Le tableau B réfère A par index
// local, pas par joueur.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_dmgamount [filmID]   # défaut 000d5950
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"levelup/go-api/internal/analysis"
	"math"
	"os"
	"sort"
)

const root = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
const dmgBase = 24 // bit de départ du déser dans le payload 0xd2 (validé tmp_kwval)

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

// br : lecteur de bits MSB-first (identique à la convention du jeu + tmp_kwval).
type br struct {
	pl  []byte
	bp  int
	bad bool // dépassement de fin de payload
}

func (r *br) R(n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := r.bp + i
		if p>>3 >= len(r.pl) {
			r.bad = true
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((r.pl[p>>3]>>uint(7-(p&7)))&1)
	}
	r.bp += n
	return v
}
func (r *br) g1() int { return int(r.R(1)) }

// dmgRec : ce qu'on extrait d'un record 0xd2.
type dmgRec struct {
	att        int       // attaquant (index joueur 0..7) ou -1
	family     uint32    // famille arme (source de dégât) ou 0xffffffff
	variant    uint32    // variant_name
	out0, out1 int       // flags de branche
	s34, f8    int       // nb hits (tableau A), nb impacts (tableau B)
	arrAStart  int       // position bit du 1er élément du tableau A
	amounts    []float32 // MONTANT par hit (film : float32 brut)
	rawAmts    []uint32  // les 32 bits bruts (diag endianness)
	ok         bool      // grammaire lue sans dépassement + chemin long
}

// readCC68 : port bit-exact de FUN_14080cc68 -> (f8, s34).
func readCC68(r *br) (int, int) {
	if r.g1() == 1 { // A!=0
		return 0, 0
	}
	f8 := 1
	if r.g1() == 0 { // B==0
		f8 = int(r.R(4))
	}
	// s34 :
	if r.g1() == 1 { // C==1
		return f8, 0
	}
	if r.g1() == 0 { // D==0
		return f8, int(r.R(4))
	}
	return f8, 1 // D==1
}

// parseDmg : porte FUN_14080c1f8 (film) jusqu'au tableau A inclus.
func parseDmg(pl []byte) dmgRec {
	var rec dmgRec
	rec.att = -1
	rec.family = 0xffffffff
	r := &br{pl: pl, bp: dmgBase}
	rec.out0 = r.g1()
	rec.out1 = r.g1()
	r.R(8) // FUN_141fcf670 fixe
	if r.g1() == 0 {
		rec.att = int(r.R(5)) // ECS_ReadEntityRefIndex5
	}
	if r.g1() == 0 {
		r.R(2) // dmgIndex
	}
	if r.g1() == 1 {
		rec.family = uint32(r.R(32))
	}
	rec.variant = uint32(r.R(32))
	r.R(1) // out[0x1d]
	r.R(1) // out[2]
	f2dd := 0
	if rec.out1 == 1 {
		r.R(1)        // out[0x2dc]
		f2dd = r.g1() // out[0x2dd]
	}
	if f2dd != 0 {
		if r.g1() == 1 { // FUN_1431a0abc
			r.R(10)
		}
	}
	if rec.out0 == 1 {
		// CHEMIN COURT : pas de tableau A. On s'arrête.
		rec.ok = !r.bad
		return rec
	}
	rec.f8, rec.s34 = readCC68(r)
	rec.arrAStart = r.bp
	// TABLEAU A : s34 hits.
	for i := 0; i < rec.s34 && !r.bad && rec.s34 < 64; i++ {
		r.R(2) // tag
		r.R(1) // bool
		raw := uint32(r.R(32))
		rec.rawAmts = append(rec.rawAmts, raw)
		rec.amounts = append(rec.amounts, math.Float32frombits(raw))
	}
	rec.ok = !r.bad && rec.s34 >= 0 && rec.s34 < 64
	return rec
}

type frame struct {
	marker byte
	pl     []byte
	ts     uint64
}

// collect : toutes les frames type-0 (marqueur, payload, ts) d'un film.
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

// byterev32 : inverse l'ordre des octets d'un mot 32 bits (test endianness alt).
func byterev32(v uint32) uint32 {
	return v>>24 | (v&0xff0000)>>8 | (v&0xff00)<<8 | v<<24
}

func plausible(f float32) bool {
	if f != f || math.IsInf(float64(f), 0) {
		return false
	}
	a := float64(f)
	if a < 0 {
		a = -a
	}
	return a > 0.01 && a < 100000
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
	fmt.Printf("=== DMGAMOUNT %s : %d frames type-0, %d records 0xd2 ===\n", m, len(frames), len(d2))

	// (1) DIAGNOSTIC grammaire : out0/out1, s34, dépassements.
	out0h, out1h, s34h := map[int]int{}, map[int]int{}, map[int]int{}
	nOK, nBad, nShort := 0, 0, 0
	for _, f := range d2 {
		rec := parseDmg(f.pl)
		out0h[rec.out0]++
		out1h[rec.out1]++
		if rec.out0 == 1 {
			nShort++
			continue
		}
		s34h[rec.s34]++
		if rec.ok {
			nOK++
		} else {
			nBad++
		}
	}
	fmt.Printf("out0=%v out1c=%v | chemin-court(out0=1)=%d | long OK=%d bad=%d\n", out0h, out1h, nShort, nOK, nBad)
	fmt.Printf("s34 (nb hits/record, chemin long) : %s\n", topMap(s34h, 12))

	// (2) CALIBRATION MONTANT : on ne sait pas a priori (a) le delta bit exact du champ montant
	// vs arrAStart, ni (b) l'endianness float. On BALAYE delta -8..+40 et 2 endianness, et on
	// compte les floats plausibles [0.5,300] parmi les records s34>=1. Le meilleur (delta,endian)
	// révèle empiriquement où est le montant.
	type cal struct {
		delta int
		le    bool
		n     int
		vals  []float64
	}
	best := cal{}
	for delta := -8; delta <= 48; delta++ {
		for _, le := range []bool{false, true} {
			c := cal{delta: delta, le: le}
			for _, f := range d2 {
				rec := parseDmg(f.pl)
				if rec.out0 == 1 || !rec.ok || rec.s34 < 1 {
					continue
				}
				raw := uint32(bitsAt(f.pl, rec.arrAStart+delta, 32))
				if le {
					raw = byterev32(raw)
				}
				fv := math.Float32frombits(raw)
				if fv >= 0.5 && fv <= 300 {
					c.n++
					c.vals = append(c.vals, float64(fv))
				}
			}
			if c.n > best.n {
				best = c
			}
		}
	}
	fmt.Printf("\n--- CALIBRATION MONTANT (records s34>=1) ---\n")
	fmt.Printf("meilleur : delta=%+d endian=%s -> %d floats dans [0.5,300] %s\n",
		best.delta, pick(best.le, "LE", "BE"), best.n, pct(best.vals))

	_ = best
	// MODÈLE RETENU : le montant est un point-fixe 8.8 dans les 16 bits de poids fort du champ R32
	// de l'élément A (tag R2 + bool R1 = +3). montant = bitsAt(arrAStart+3, 16) / 256. Les 16 bits
	// de poids faible sont ~constants (0x8000). On calibre le delta fin qui maximise la fraction de
	// montants dans [5,160] (plage de dégât Halo).
	sweepFixed := func(delta int) (int, int) {
		in, tot := 0, 0
		for _, f := range d2 {
			rec := parseDmg(f.pl)
			if rec.out0 == 1 || !rec.ok || rec.s34 < 1 {
				continue
			}
			tot++
			v := float64(bitsAt(f.pl, rec.arrAStart+delta, 16)) / 256
			if v >= 5 && v <= 160 {
				in++
			}
		}
		return in, tot
	}
	fmt.Printf("\n--- MONTANT point-fixe 8.8 (bitsAt(offset,16)/256) — sweep delta ---\n")
	for delta := 0; delta <= 6; delta++ {
		in, tt := sweepFixed(delta)
		fmt.Printf("  delta=%+d : %d/%d dans [5,160]\n", delta, in, tt)
	}
	// Ghidra : élément A = tag R2 + bool R1 + R32 ; le montant = 16 bits de poids fort du R32
	// (à arrAStart+3), format 8.8. On retient delta=3 (aligné Ghidra).
	const amtDelta = 3
	amtAt := func(pl []byte, arrAStart, i int) float64 {
		return float64(bitsAt(pl, arrAStart+i*35+amtDelta, 16)) / 256
	}
	// (2b) RAW du champ à arrAStart+3 (tag R2 + bool R1 = +3), 32 bits : distribution + encodages.
	// Le montant devrait y être (Ghidra : R2, bool, R32). On liste les valeurs DISTINCTES et teste
	// plusieurs encodages (float BE/LE, uint, /256, /1000).
	rawHist := map[uint32]int{}
	for _, f := range d2 {
		rec := parseDmg(f.pl)
		if rec.out0 == 1 || !rec.ok || rec.s34 < 1 {
			continue
		}
		raw := uint32(bitsAt(f.pl, rec.arrAStart+3, 32))
		rawHist[raw]++
	}
	type rv struct {
		raw uint32
		n   int
	}
	var rvs []rv
	for k, v := range rawHist {
		rvs = append(rvs, rv{k, v})
	}
	sort.Slice(rvs, func(i, j int) bool { return rvs[i].n > rvs[j].n })
	fmt.Printf("\n--- RAW à arrAStart+3 (32b) : %d valeurs distinctes ---\n", len(rvs))
	for i := 0; i < 20 && i < len(rvs); i++ {
		r := rvs[i].raw
		fmt.Printf("  0x%08X n=%d | fBE=%.4g fLE=%.4g u=%d u>>16=%d (u>>16)/256=%.3f\n",
			r, rvs[i].n, math.Float32frombits(r), math.Float32frombits(byterev32(r)),
			r, r>>16, float64(r>>16)/256)
	}
	// balaye des largeurs plus courtes au même offset (le champ pourrait faire 8/12/16 bits).
	fmt.Printf("--- largeurs courtes à arrAStart+3 (valeurs distinctes) ---\n")
	for _, w := range []int{8, 10, 12, 16} {
		h := map[uint64]int{}
		for _, f := range d2 {
			rec := parseDmg(f.pl)
			if rec.out0 == 1 || !rec.ok || rec.s34 < 1 {
				continue
			}
			h[bitsAt(f.pl, rec.arrAStart+3, w)]++
		}
		type kv struct {
			k uint64
			v int
		}
		var a []kv
		for k, v := range h {
			a = append(a, kv{k, v})
		}
		sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
		s := ""
		for i := 0; i < 8 && i < len(a); i++ {
			s += fmt.Sprintf("%d:%d ", a[i].k, a[i].v)
		}
		fmt.Printf("  w=%2d (%d distinctes) top: %s\n", w, len(a), s)
	}

	// (3) échantillon de records (attaquant + hits + montants).
	fmt.Printf("\n--- ÉCHANTILLON (10 premiers records long) ---\n")
	shown := 0
	for _, f := range d2 {
		rec := parseDmg(f.pl)
		if rec.out0 == 1 || !rec.ok || shown >= 10 {
			continue
		}
		shown++
		var amts []float64
		sum := 0.0
		for i := 0; i < rec.s34; i++ {
			a := amtAt(f.pl, rec.arrAStart, i)
			amts = append(amts, a)
			sum += a
		}
		fmt.Printf("  att=%2d fam=%08X s34=%d f8=%d montants=%s (Σ=%.1f) ts=%d\n",
			rec.att, rec.family, rec.s34, rec.f8, fmtFloats(amts), sum, f.ts)
	}

	// (4) DÉGÂT TOTAL par attaquant vs KILLS par attaquant (0xE6, tueur=R5@83).
	dmgByAtk := map[int]float64{}
	hitsByAtk := map[int]int{}
	for _, f := range d2 {
		rec := parseDmg(f.pl)
		if rec.out0 == 1 || !rec.ok || rec.att < 0 || rec.att > 15 {
			continue
		}
		for i := 0; i < rec.s34; i++ {
			a := amtAt(f.pl, rec.arrAStart, i)
			if a >= 1 && a <= 200 {
				dmgByAtk[rec.att] += a
				hitsByAtk[rec.att]++
			}
		}
	}
	killsByAtk := map[int]int{}
	type kev struct {
		killer, victim int
		ts             uint64
	}
	var kills []kev
	for _, f := range frames {
		if f.marker != 0xE6 {
			continue
		}
		k := int(bitsAt(f.pl, 83, 5))
		v := int(bitsAt(f.pl, 88, 5))
		kills = append(kills, kev{k, v, f.ts})
		if k < 16 {
			killsByAtk[k]++
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].ts < kills[j].ts })
	fmt.Printf("\n--- DÉGÂT TOTAL vs KILLS par joueur ---\n")
	var atks []int
	seen := map[int]bool{}
	for a := range dmgByAtk {
		if !seen[a] {
			atks = append(atks, a)
			seen[a] = true
		}
	}
	for a := range killsByAtk {
		if !seen[a] {
			atks = append(atks, a)
			seen[a] = true
		}
	}
	sort.Slice(atks, func(i, j int) bool { return dmgByAtk[atks[i]] > dmgByAtk[atks[j]] })
	type rankRow struct {
		atk   int
		dmg   float64
		hits  int
		kills int
	}
	var rows []rankRow
	for _, a := range atks {
		rows = append(rows, rankRow{a, dmgByAtk[a], hitsByAtk[a], killsByAtk[a]})
	}
	for _, rw := range rows {
		fmt.Printf("  joueur %2d | dégât total=%9.0f | hits=%5d | kills=%2d\n", rw.atk, rw.dmg, rw.hits, rw.kills)
	}
	fmt.Printf("Spearman(dégât total, kills) = %.3f  |  Spearman(hits, kills) = %.3f\n",
		spearman(rows, func(r rankRow) float64 { return r.dmg }, func(r rankRow) float64 { return float64(r.kills) }),
		spearman(rows, func(r rankRow) float64 { return float64(r.hits) }, func(r rankRow) float64 { return float64(r.kills) }))

	// (5) TEST "dernier hit fatal vient du tueur" : pour chaque kill 0xE6, l'attaquant du DERNIER
	// record 0xd2 (ts<=kill.ts) doit être le tueur.
	var dmgTs []struct {
		att int
		ts  uint64
		sum float64
	}
	for _, f := range d2 {
		rec := parseDmg(f.pl)
		if rec.out0 == 1 || !rec.ok || rec.att < 0 {
			continue
		}
		s := 0.0
		for i := 0; i < rec.s34; i++ {
			s += amtAt(f.pl, rec.arrAStart, i)
		}
		dmgTs = append(dmgTs, struct {
			att int
			ts  uint64
			sum float64
		}{rec.att, f.ts, s})
	}
	sort.Slice(dmgTs, func(i, j int) bool { return dmgTs[i].ts < dmgTs[j].ts })
	matchKiller, totK := 0, 0
	for _, k := range kills {
		if k.killer >= 16 || k.killer == k.victim {
			continue
		}
		lastAtk, found := -1, false
		for _, d := range dmgTs {
			if d.ts > k.ts {
				break
			}
			lastAtk, found = d.att, true
		}
		if !found {
			continue
		}
		totK++
		if lastAtk == k.killer {
			matchKiller++
		}
	}
	fmt.Printf("\n--- TEST coup fatal ---\n")
	fmt.Printf("dernier 0xd2 avant le kill a attaquant==tueur : %d/%d = %.0f%%\n",
		matchKiller, totK, float64(matchKiller)*100/float64(maxi(totK, 1)))
	fmt.Println("NB : att 0xd2 = index ECS local (ECS_ReadEntityRefIndex5), espace DIFFÉRENT du slot 0xE6@83.")

	// (6) DISTRIBUTION DES MONTANTS (intrinsèque, indépendant de l'index) + near-lethal vs kills.
	var allAmts []float64
	nearLethal := 0
	for _, f := range d2 {
		rec := parseDmg(f.pl)
		if rec.out0 == 1 || !rec.ok {
			continue
		}
		for i := 0; i < rec.s34; i++ {
			a := amtAt(f.pl, rec.arrAStart, i)
			allAmts = append(allAmts, a)
			if a >= 120 {
				nearLethal++
			}
		}
	}
	fmt.Printf("\n--- DISTRIBUTION MONTANTS (%d hits, tous records) ---\n", len(allAmts))
	fmt.Printf("percentiles : %s\n", pct(allAmts))
	hb := map[int]int{}
	for _, a := range allAmts {
		hb[int(a)/20*20]++
	}
	var bs []int
	for b := range hb {
		bs = append(bs, b)
	}
	sort.Ints(bs)
	fmt.Print("histogramme (buckets de 20) : ")
	for _, b := range bs {
		fmt.Printf("[%d-%d]:%d ", b, b+20, hb[b])
	}
	fmt.Println()
	// #hits near-lethal (montant>=120, ~full EHP) vs #frames 0xE6 (événements de kill).
	nE6 := 0
	for _, f := range frames {
		if f.marker == 0xE6 {
			nE6++
		}
	}
	fmt.Printf("hits near-lethal (>=120) : %d  |  frames 0xE6 (kill-events) : %d  |  kills chunk_27 : %d\n",
		nearLethal, nE6, countChunk27Kills(m))
}

// countChunk27Kills : nb de kills « vérité offline » via les highlight-events (chunks tardifs),
// même source que tmp_kwval runPipeline. Retourne -1 si indisponible.
func countChunk27Kills(m string) int {
	cache := root + "/" + m
	best := -1
	for chi := 41; chi >= 18; chi-- {
		b := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, chi))
		if len(b) == 0 {
			continue
		}
		evs, _ := analysis.ParseHighlightEvents(b, 0)
		n := 0
		for _, e := range evs {
			if e.EventType == analysis.EventTypeKill {
				n++
			}
		}
		if n > best {
			best = n
		}
	}
	return best
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

func topMap(h map[int]int, n int) string {
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

func pct(v []float64) string {
	if len(v) == 0 {
		return "(aucune)"
	}
	sort.Float64s(v)
	p := func(q float64) float64 { return v[int(q*float64(len(v)-1))] }
	return fmt.Sprintf("min=%.2f p25=%.2f méd=%.2f p75=%.2f p95=%.2f max=%.2f",
		v[0], p(0.25), p(0.5), p(0.75), p(0.95), v[len(v)-1])
}

func fmtFloats(a []float64) string {
	if len(a) == 0 {
		return "[]"
	}
	s := "["
	for i, x := range a {
		if i > 0 {
			s += " "
		}
		s += fmt.Sprintf("%.1f", x)
	}
	return s + "]"
}

func pick(b bool, y, n string) string {
	if b {
		return y
	}
	return n
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// spearman : corrélation de rang entre deux extracteurs sur les lignes.
func spearman[T any](rows []T, fx, fy func(T) float64) float64 {
	n := len(rows)
	if n < 2 {
		return 0
	}
	rank := func(f func(T) float64) []float64 {
		idx := make([]int, n)
		for i := range idx {
			idx[i] = i
		}
		sort.Slice(idx, func(a, b int) bool { return f(rows[idx[a]]) < f(rows[idx[b]]) })
		rk := make([]float64, n)
		for r, i := range idx {
			rk[i] = float64(r)
		}
		return rk
	}
	rx, ry := rank(fx), rank(fy)
	var mx, my float64
	for i := 0; i < n; i++ {
		mx += rx[i]
		my += ry[i]
	}
	mx /= float64(n)
	my /= float64(n)
	var num, dx, dy float64
	for i := 0; i < n; i++ {
		a, b := rx[i]-mx, ry[i]-my
		num += a * b
		dx += a * a
		dy += b * b
	}
	if dx == 0 || dy == 0 {
		return 0
	}
	return num / math.Sqrt(dx*dy)
}
