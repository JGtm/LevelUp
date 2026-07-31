package main

import (
	"encoding/binary"
	"fmt"
	"sort"
)

// deser_v2.go — port bit-exact de FUN_14080c1f8 (déser de dégât), CHEMIN param_5=1 (désérialisation, celui
// du dispatcher EventDispatch_Generic_ReadType7 qui passe param_5=1). Le port legacy (damageDeserLenPorted)
// empruntait par erreur les branches param_5==0 (sérialisation) -> résidus. Largeurs issues du workflow
// port-damage-deser (RE Ghidra + vérif table config en mémoire CE) :
//   hit  FUN_1406d3140(param3=1) = R1 gate ; gate=1 -> R9 ; gate=0 -> R13 ; + R2   (12 ou 16 bits)
//   vec  FUN_140c1e924           = 3 × (cnt1==1 ? 12 : 4)
//   FUN_140c9e4d8 widths 1/2/4/6/15/7 (asm-confirmées) ; FUN_1431eb378 (queue kill-event) = 68 fixes
//   FUN_14080cb50 = R1 p + R1 g + (g?R4) ; FUN_14080cb98 = R2 ; FUN_142a40f18/14320c36c = R1 gate + (2×R6)
// Curseur du kill-event = base + L + 10 (préambule [R7][3×R1] de l'event kill). Oracle CE = curseur = 20 + L.

// dwV2 : les quelques largeurs restées incertaines (calibrées contre l'oracle CE).
type dwV2 struct {
	base   int // 10 (départ du déser dans le payload)
	e494   int // FUN_14076e494 total (chemin compressé ~67 / brut 96 ; rare)
	tail84 int // FUN_1406d84b4 dans le tail (out[0x310], out1c==1)
	guard  int // 141102ed0>1 (0/1) : garde du d84b4 out1c==1
}

func defaultV2() dwV2 { return dwV2{base: 24, e494: 67, tail84: 4, guard: 0} }

// vars de debug : dernières valeurs décodées par deserDamageV2 (diagnostic calibration vs oracle CE).
var dbgOut0, dbgOut1c, dbgCnt1, dbgCnt2 int

// positions de bit (packet-local, = r.bp) où le port ATTEINT chaque frontière — à comparer aux positions
// CE réelles (bracket : counter=FUN_14080cc68 entrée, section=FUN_140c9e4d8 entrée).
var dbgPosCounter, dbgPosSection int

// qhit = FUN_1406d3140(param3=1) : R1 gate ; gate=1 -> R9 ; gate=0 -> R13 ; + R2.
func qhit(r *br) {
	if r.g1() == 1 {
		r.R(9)
	} else {
		r.R(13)
	}
	r.R(2)
}

// d9e990 = FUN_140c9e990 : R2 tag ; tag1 -> qhit + R1 + (g?R6) ; tag2 -> qhit ; sinon 0.
func d9e990(r *br) {
	switch r.R(2) {
	case 1:
		qhit(r)
		if r.g1() == 1 {
			r.R(6)
		}
	case 2:
		qhit(r)
	}
}

// d9e738 = FUN_140c9e738 (VEC) : R1 s ; si s==0 -> R15 + R7.
func d9e738(r *br) {
	if r.g1() == 0 {
		r.R(15)
		r.R(7)
	}
}

// d9e4d8 = FUN_140c9e4d8 (param5=1) : renvoie g0 (0 => R30 en amont).
func d9e4d8(r *br) int {
	g0 := r.g1()
	if g0 == 0 {
		return 0
	}
	d9e990(r)
	if r.g1() == 0 { // g2
		r.R(4)
		r.R(4) // 2× FUN_1406d84b4 (EBX=4)
		if r.g1() == 0 {
			return g0
		}
		d9e738(r)
	} else {
		d9e738(r)
	}
	return g0
}

// d8eff64v2 = FUN_1408eff64 (param5=1) : R1 gate ; si gate -> d9e990 (R2 tag ...).
func d8eff64v2(r *br) {
	if r.g1() == 1 {
		d9e990(r)
	}
}

// deserDamageV2 : L = bits consommés par le déser depuis w.base. Curseur kill-event = w.base + L + 10.
func deserDamageV2(pl []byte, w dwV2) int {
	r := &br{pl: pl, bp: w.base}
	out0 := r.g1()  // out[0]
	out1c := r.g1() // out[0x1c]
	dbgOut0, dbgOut1c = out0, out1c
	r.R(8)           // FUN_141fcf670
	if r.g1() == 0 { // FUN_1407f2034 (ecsRef)
		r.R(5)
	}
	if r.g1() == 0 { // FUN_1406d00ec (varuint)
		r.R(2)
	}
	if r.g1() == 1 { // FUN_14080d69c : R1 gate ; si 1 -> R32 (FUN_14080d6f0)
		r.R(32)
	}
	r.R(32) // FUN_14080dec4 variant_name (LA CAUSE)
	postVariantV2(r, out0, out1c, w)
	return r.bp - w.base
}

// postVariantV2 : la partie du déser APRÈS le variant_name (compteurs, hits, boucles, section, tail).
// Modélisée proprement ; le préambule pré-variant (3-loop dispatcher + globalID) est la partie murky.
func postVariantV2(r *br, out0, out1c int, w dwV2) {
	r.g1() // out[0x1d]
	r.g1() // out[2]
	out2dd := 0
	if out1c == 1 {
		r.g1()
		out2dd = r.g1()
	}
	if out2dd != 0 { // FUN_1431a0abc
		if r.g1() == 1 {
			r.R(10)
		}
	}
	if out0 == 1 {
		return // chemin court
	}
	// FUN_14080cc68 : deux compteurs
	dbgPosCounter = r.bp
	cnt1, cnt2 := 0, 0
	if r.g1() == 0 {
		if r.g1() == 1 {
			cnt1 = 1
		} else {
			cnt1 = int(r.R(4))
		}
		if r.g1() == 1 {
			cnt2 = 0
		} else if r.g1() == 1 {
			cnt2 = 1
		} else {
			cnt2 = int(r.R(4))
		}
	}
	dbgCnt1, dbgCnt2 = cnt1, cnt2
	// BOUCLE HITS (param5=1)
	for i := 0; i < cnt2; i++ {
		r.R(2)
		r.g1()
		qhit(r)
	}
	// BOUCLE 2
	wvec := 4
	if cnt1 == 1 {
		wvec = 12
	}
	bVar14, ranLoop2 := true, false
	if cnt1 > 0 {
		ranLoop2 = true
		for i := 0; i < cnt1; i++ {
			r.R(4)
			if r.g1() == 1 {
				bVar14 = r.R(3) != 0 // FUN_1406d310c(6)=3
				if cnt2 < 3 {
					r.R(1)
				} else {
					r.R(4)
				}
				r.R(16)
				r.R(3 * wvec) // FUN_140c1e924
			}
		}
	}
	// SECTION (sautée si boucle2 exécutée ET bVar14==0)
	dbgPosSection = r.bp
	if !(ranLoop2 && !bVar14) {
		cVar2 := d9e4d8(r)
		d8eff64v2(r)
		if cVar2 == 0 {
			r.R(30) // FUN_1406d8288
		}
	}
	// LAB_88d
	if out2dd == 0 {
		if r.g1() == 1 {
			r.R(6)
		}
		r.R(6)
	}
	if out1c == 1 {
		if r.R(2) == 0 { // FUN_142af27f8 type
			r.R(w.e494)
		}
		if w.guard == 1 {
			r.R(w.tail84)
		}
	}
	if out1c == 0 {
		r.R(2) // FUN_14080cb98
		p := r.g1()
		if r.g1() == 1 { // FUN_14080cb50 gateB
			r.R(4)
		}
		if p != 0 {
			if r.g1() == 1 { // FUN_14320c36c
				r.R(6)
				r.R(6)
			}
			if r.g1() == 1 { // FUN_142a40f18
				r.R(6)
				r.R(6)
			}
		}
	}
	r.R(6) // out[0x30c]
	if r.g1() == 1 {
		r.R(w.tail84) // out[0x310]
	}
	// param5==1 : SKIP le R(1) out[4]
	if r.g1() == 1 {
		r.R(w.e494) // FUN_14076e494
	}
}

// deserAnchoredCursor : ANCRE sur le variant (suffixe 0x42c9679f trouvé, = un vrai champ). out0/out1c sont
// à position fixe (bit base/base+1). On saute le pré-variant murky et on modélise le post-variant propre.
// Renvoie le curseur du kill-event = fin du déser + 10 (préambule [R7][3×R1] de l'event kill).
func deserAnchoredCursor(pl []byte, sufBit, out0, out1c int, w dwV2) int {
	r := &br{pl: pl, bp: sufBit}
	r.R(32) // variant_name (déjà à l'ancre)
	postVariantV2(r, out0, out1c, w)
	return r.bp + 10
}

// variantAnchorLast : position du DERNIER couple family(32)+variant(32) d'une CLASSE de dégât connue
// (firearm/mêlée/grenade) AVANT le curseur du kill-event. C'est le record FATAL co-localisé -> sa cause
// déterministe (pas de held-weapon : on lit le variant du dégât qui a tué, pas une arme tenue au hasard).
// Renvoie (famPos, variant) ou (-1, 0).
func variantAnchorLast(pl []byte, cur int) (int, uint32) {
	last, lastV := -1, uint32(0)
	hi := cur
	if hi > len(pl)*8 {
		hi = len(pl) * 8
	}
	for b := 0; b+64 <= hi; b++ {
		v := uint32(bitsAt(pl, b+32, 32))
		if damageClass(v) != dmgOther {
			last, lastV = b, v
		}
	}
	return last, lastV
}

// runVarCls : self-contained. Scanne les paquets fatals (dmgMk+sz>=700), localise le kill-event, lit la
// CLASSE du variant fatal CO-LOCALISÉ (variantAnchorLast, avant le curseur) et sort la distribution
// catégorie déterministe (firearm/mêlée/grenade). À comparer aux compteurs du pipeline propre (killfeed).
func runVarCls(m string, h32 map[uint32]string) {
	cache := root + "/" + m
	dmgMk := map[byte]bool{0xD2: true, 0xC0: true, 0xC2: true, 0xC3: true, 0xCA: true, 0xD3: true, 0xE9: true}
	nFire, nMel, nGren, nOther := 0, 0, 0, 0
	classHist := map[string]int{}
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ != 0 || len(pl) == 0 || !dmgMk[pl[0]] || sz < 700 {
				continue
			}
			cur := locateKillEventCursor(pl)
			if cur < 0 {
				continue
			}
			fam, variant := variantAnchorLast(pl, cur)
			if fam < 0 {
				nOther++
				classHist["cause-?"]++
				continue
			}
			switch damageClass(variant) {
			case dmgFirearm:
				nFire++
				classHist[weaponName(uint32(bitsAt(pl, fam, 32)), h32)]++
			case dmgMelee:
				nMel++
				classHist["Mêlée"]++
			case dmgGrenade:
				nGren++
				classHist["Grenade"]++
			default:
				nOther++
			}
		}
	}
	fmt.Printf("=== VARCLS %s (variant co-localisé, déterministe) : FIREARM=%d MÊLÉE=%d GRENADE=%d cause-?=%d ===\n",
		m, nFire, nMel, nGren, nOther)
	type kv2 struct {
		k string
		v int
	}
	var a []kv2
	for k, v := range classHist {
		a = append(a, kv2{k, v})
	}
	sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
	for i := 0; i < 12 && i < len(a); i++ {
		fmt.Printf("  %-24s %d\n", a[i].k, a[i].v)
	}
}

// traceV2 : rejoue le préambule du port jusqu'au variant (R32) et renvoie (bit du variant, valeur lue, L final).
func traceV2(pl []byte, w dwV2) (vpos int, vval uint32, L int) {
	r := &br{pl: pl, bp: w.base}
	r.g1() // out0
	r.g1() // out1c
	r.R(8) // FUN_141fcf670
	if r.g1() == 0 {
		r.R(5)
	}
	if r.g1() == 0 {
		r.R(2)
	}
	r.g1()      // FUN_14080d69c
	vpos = r.bp // position du variant
	vval = uint32(r.R(32))
	L = deserDamageV2(pl, w)
	return
}

// runDeserLen2 : valide le port V2 (param5=1) contre l'oracle CE (curseur vrai = L + 20). Calibre les 3
// largeurs restées incertaines (e494/tail84/guard) pour maximiser l'exactitude, par marqueur.
func runDeserLen2(m string) {
	pkts := loadFatalPackets(m)
	byMk := map[byte]int{}
	for _, p := range pkts {
		byMk[p.marker]++
	}
	fmt.Printf("=== DESERLEN2 %s : %d paquets fatals (oracle) ; marqueurs %v ===\n", m, len(pkts), byMk)

	eval := func(w dwV2, mk byte, onlyMk bool) (exact, tot int, resid map[int]int) {
		resid = map[int]int{}
		for _, p := range pkts {
			if onlyMk && p.marker != mk {
				continue
			}
			pred := deserDamageV2(p.pl, w) + 20
			d := pred - p.cursor
			resid[d]++
			tot++
			if d == 0 {
				exact++
			}
		}
		return
	}

	// Calibration sur 0xD2 (le marqueur majoritaire, préambule 10b garanti).
	best, bestN := defaultV2(), -1
	for _, e494 := range []int{0, 11, 22, 33, 44, 55, 66, 67, 89, 96} {
		for tail84 := 0; tail84 <= 32; tail84++ {
			for guard := 0; guard <= 1; guard++ {
				w := dwV2{base: 10, e494: e494, tail84: tail84, guard: guard}
				ex, _, _ := eval(w, 0xd2, true)
				if ex > bestN {
					bestN, best = ex, w
				}
			}
		}
	}
	ex, tot, resid := eval(best, 0xd2, true)
	fmt.Printf(">>> 0xD2 : EXACT %d/%d = %.1f%% | calib e494=%d tail84=%d guard=%d\n",
		ex, tot, 100*float64(ex)/float64(max1(tot)), best.e494, best.tail84, best.guard)
	// top résidus
	type kv struct{ k, v int }
	var a []kv
	for k, v := range resid {
		a = append(a, kv{k, v})
	}
	sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
	fmt.Printf("    résidus (pred-curseur) : ")
	for i := 0; i < 12 && i < len(a); i++ {
		fmt.Printf("%d:%d ", a[i].k, a[i].v)
	}
	fmt.Println()

	// ANCRÉ : trouve le suffixe (variant) et modélise le post-variant. sufBit par paquet.
	sufOf := func(pl []byte) int {
		for b := 0; b+32 <= len(pl)*8; b++ {
			if uint32(bitsAt(pl, b, 32)) == sfx {
				return b
			}
		}
		return -1
	}
	evalAnc := func(w dwV2, out0, out1c int, mk byte) (exact, tot int, resid map[int]int) {
		resid = map[int]int{}
		for _, p := range pkts {
			if p.marker != mk {
				continue
			}
			sb := sufOf(p.pl)
			if sb < 0 {
				continue
			}
			d := deserAnchoredCursor(p.pl, sb, out0, out1c, w) - p.cursor
			resid[d]++
			tot++
			if d == 0 {
				exact++
			}
		}
		return
	}
	bestA, bestAN, bestO1c := defaultV2(), -1, 0
	for _, out1c := range []int{0, 1} {
		for _, e494 := range []int{0, 11, 22, 33, 44, 55, 66, 67, 89, 96} {
			for tail84 := 0; tail84 <= 40; tail84++ {
				w := dwV2{base: 10, e494: e494, tail84: tail84, guard: 0}
				exA, _, _ := evalAnc(w, 0, out1c, 0xd2)
				if exA > bestAN {
					bestAN, bestA, bestO1c = exA, w, out1c
				}
			}
		}
	}
	exA, totA, residA := evalAnc(bestA, 0, bestO1c, 0xd2)
	fmt.Printf(">>> 0xD2 ANCRÉ (out0=0, post-variant) : EXACT %d/%d = %.1f%% | out1c=%d e494=%d tail84=%d\n",
		exA, totA, 100*float64(exA)/float64(max1(totA)), bestO1c, bestA.e494, bestA.tail84)
	var aa []kv
	for k, v := range residA {
		aa = append(aa, kv{k, v})
	}
	sort.Slice(aa, func(i, j int) bool { return aa[i].v > aa[j].v })
	fmt.Printf("    résidus ANCRÉ : ")
	for i := 0; i < 12 && i < len(aa); i++ {
		fmt.Printf("%d:%d ", aa[i].k, aa[i].v)
	}
	fmt.Println()

	// TRACE : pour les 3 premiers 0xD2, où mon port lit le variant vs la vraie position du suffixe.
	fmt.Println("--- TRACE variant (mon port) vs suffixe réel 0x42c9679f ---")
	nt := 0
	for _, p := range pkts {
		if p.marker != 0xd2 || nt >= 4 {
			continue
		}
		nt++
		// vraie position du suffixe dans le paquet (variant_name du record de dégât)
		sufBit := -1
		for b := 0; b+32 <= len(p.pl)*8; b++ {
			if uint32(bitsAt(p.pl, b, 32)) == sfx {
				sufBit = b
				break
			}
		}
		vpos, vval, L := traceV2(p.pl, best)
		fmt.Printf("  cur=%4d | suffixe réel @bit %4d | mon variant @bit %4d val=0x%08X | L=%d pred=%d\n",
			p.cursor, sufBit, vpos, vval, L, L+20)
	}

	// Application aux frères avec la même calib (le préambule peut différer -> diagnostic).
	for _, mk := range []byte{0xC0, 0xC2, 0xC3, 0xCA, 0xD3, 0xE9} {
		if byMk[mk] == 0 {
			continue
		}
		ex, tot, _ := eval(best, mk, true)
		fmt.Printf("    0x%02X : EXACT %d/%d\n", mk, ex, tot)
	}
}
