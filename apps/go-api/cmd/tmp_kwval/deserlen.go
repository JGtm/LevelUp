// deserlen.go — PORT OFFLINE du déser de dégât (FUN_14080c1f8) pour calculer L (bits consommés),
// afin de localiser EXACTEMENT le kill-event embarqué dans le paquet de dégât FATAL 0xd2.
//
// Modèle (vérifié CE) : dans le payload d'un paquet 0xd2, le déser de dégât démarre à un bit `base`,
// consomme L bits ; le kill-event suit immédiatement (R7 type=0x55 + 3 gates = 10 bits) puis field0.
// Donc le CURSEUR CE (position de field0) = base + L + 10. La mission fixe base=10 → curseur = 20 + L.
//
// Les largeurs QUANTIZED-RUNTIME (chargées au map-load, non résolubles en statique) sont laissées
// en PARAMÈTRES (deserWidths) et CALIBRÉES contre l'oracle CE (134 paquets fatals, curseur connu).
package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

// keFloor : plancher de bit du scan de kill-event (voir locateKillEventCursor). ABAISSÉ 140->80
// (workflow freres-coverage-lift, 2026-07-10) : les vrais curseurs des marqueurs FRÈRES (0xC0/C2/C3/
// E9) tombent dans la bande [90,115), SOUS l'ancien plancher 140 -> ils étaient jetés (le 140 était
// calibré sur des films 0xD2-dominants dont tous les curseurs sont >=155). Le motif 0x2A8 (R7=85 +
// 3 gates 000) + validKE filtrent déjà les faux R7 précoces, donc 80 ne crée PAS de faux positifs :
// PROUVÉ par l'oracle CE 9b191a7f (locator EXACT 129/134 identique de keFloor=80 à 140 ; la "grappe
// 0xC0 @124" ne matérialise aucune perte). GAIN mesuré 000d5950 : couverture 53.8%->78.5% (+24.7pts),
// accuracy pairmatrix 96.2%->97.3%. Non-régression : 0014603f inchangé (81.5%/100%).
const keFloor = 80

// fatalPkt : un paquet de dégât fatal identifié via l'oracle CE align_killdeser.bin.
type fatalPkt struct {
	pl      []byte // payload (commence par le marqueur 0xd2)
	cursor  int    // position de bit du kill-event field0 (vérité-terrain CE)
	marker  byte
	ch, off int // localisation physique (chunk, offset de paquet dans le chunk inflaté)
}

// loadFatalPackets : lit align_killdeser.bin (rec 128o : rdtsc@0(8) cursor@8(u32) bytePtr@12(u32)
// data@16(112)), matche la fenêtre de 16 octets dans les chunks offline, et renvoie le payload +
// curseur du paquet la contenant. Même logique que runFatalDet, factorisée.
func loadFatalPackets(m string) []fatalPkt {
	baseP := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/` + m + "_align_"
	kd, err := os.ReadFile(baseP + "killdeser.bin")
	if err != nil {
		fmt.Printf("killdeser introuvable: %v\n", err)
		return nil
	}
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	var out []fatalPkt
	for o := 0; o+128 <= len(kd); o += 128 {
		cursor := int(binary.LittleEndian.Uint32(kd[o+8:]))
		win := kd[o+16 : o+16+16]
		hit, hitCh := -1, -1
		for ci, d := range chunks {
			if x := indexBytes(d, win); x >= 0 {
				hit, hitCh = x, ci
				break
			}
		}
		if hit < 0 {
			continue
		}
		d := chunks[hitCh]
		off := 0
		for off+16 <= len(d) {
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			if hit >= off && hit < off+16+sz {
				pl := make([]byte, sz)
				copy(pl, d[off+16:off+16+sz])
				out = append(out, fatalPkt{pl, cursor, pl[0], hitCh, off})
				break
			}
			off += 16 + sz
		}
	}
	return out
}

// deserWidths : largeurs quantized-runtime à calibrer (bits).
type deserWidths struct {
	base    int // bit de départ du déser dans le payload (attendu 10)
	d84b4   int // FUN_1406d84b4 (primitive "consommer N bits", N = global map-load)
	c1e924  int // FUN_140c1e924 param_4 (largeur composante vecteur ; ×3)
	e494    int // FUN_14076e494 (largeur nominale ~16)
	d3140   int // FUN_1406d3140 largeur principale
	d76dc04 int // FUN_14076dc04 largeur (chemin court out[0]==1)
	// choix de branche state-dependent (non lisibles du flux) :
	eff64Type int // *param_1 dans FUN_1408eff64 (0..3) ; 0/3 => court
	cVar2Zero int // 1 si cVar2==0 (step19 lit R30)
	flag0x2c3 int // 1 si (param_2[3]&2)!=0 dans FUN_1406cd5b8
	out1c     int // valeur out[0x1c] (constante sur ces paquets) : gate steps 11/20/21/22
}

func defaultWidths() deserWidths {
	return deserWidths{base: 10, d84b4: 8, c1e924: 8, e494: 16, d3140: 8, d76dc04: 8,
		eff64Type: 0, cVar2Zero: 1, flag0x2c3: 0, out1c: 0}
}

// bitLen : nombre de bits pour représenter la valeur v (FUN_1406d310c ~= ceil(log2)).
func bitLen(v int) int {
	b := 0
	for (1 << b) < v {
		b++
	}
	return b
}

// br : lecteur de bits séquentiel sur le payload.
type br struct {
	pl []byte
	bp int
}

func (r *br) R(n int) uint64 { v := bitsAt(r.pl, r.bp, n); r.bp += n; return v }
func (r *br) g1() int        { return int(r.R(1)) } // FUN_1406cf008

// --- sous-desers (avancent r.bp selon les specs Phase 1) ---

func d_1fcf670(r *br) { r.R(8) } // fixe 8
func d_7f2034(r *br) {
	if r.g1() == 0 {
		r.R(5)
	}
} // gate ; 0 -> +5
func d_6d00ec(r *br) int { // gate ; 1 -> -1 ; 0 -> +2
	if r.g1() == 1 {
		return -1
	}
	return int(r.R(2))
}
func d_80d69c(r *br) {
	if r.g1() == 1 {
		r.R(32)
	}
}                    // opt 32 bits
func d_80dec4(r *br) { r.R(32) } // weapon (fixe 32)
func d_31a0abc(r *br) {
	if r.g1() == 1 {
		r.R(10)
	}
} // gate ; 1 -> +10

// FUN_14080cc68 : décode (out[0xf8], out[0x34]).
func d_80cc68(r *br) (int, int) {
	if r.g1() == 1 { // A
		return 0, 0
	}
	v2 := 0
	if r.g1() == 0 { // B
		v2 = int(r.R(4))
	}
	if r.g1() == 1 { // C
		return v2, 0
	}
	v3 := 0
	if r.g1() == 0 { // D
		v3 = int(r.R(4))
	}
	return v2, v3
}

// FUN_140c1e924 : lecteur vecteur 3 composantes de largeur w -> 3*w bits.
func d_c1e924(r *br, w int) { r.R(3 * w) }

// FUN_1406d84b4 : primitive "consommer N bits".
func d_6d84b4(r *br, w int) { r.R(w) }

// FUN_14076e494 : dispatcher vecteur. Le choix de branche dépend d'un flag GLOBAL (chargé au
// map-load, non lu du flux) donc le TOTAL est constant par map. Traité comme largeur totale
// calibrée wd.e494 (candidat naturel : 96 = branche raw FUN_1406d676c).
func d_76e494(r *br, w int) { r.R(w) }

// FUN_1431a0cbc : tag 2 bits ; tag0 -> e494 ; tag1 -> +19 ; tag2/3 -> +0.
func d_31a0cbc(r *br, wd deserWidths) {
	tag := int(r.R(2))
	switch tag {
	case 0:
		d_76e494(r, wd.e494)
	case 1:
		r.R(19)
	}
}

// FUN_1408eff64 : gate présence 1 bit ; si 1 -> +2 + dispatch type (param_3==0 -> R32).
func d_8eff64(r *br, wd deserWidths) {
	if r.g1() == 0 {
		return
	}
	r.R(2)
	switch wd.eff64Type {
	case 1:
		r.R(32)
		if r.g1() == 1 {
			r.R(6)
		}
	case 2:
		r.R(32)
	}
}

// FUN_1406cd5b8 : toujours 2 gates ; conditionnels sous B et A.
func d_6cd5b8(r *br, wd deserWidths) {
	A := r.g1()
	B := r.g1()
	if B == 1 {
		if r.g1() == 1 { // FUN_140c9eabc
			t := int(r.R(2)) // FUN_1407f0278
			if t == 1 {
				r.R(32)
				if r.g1() == 1 {
					r.R(6)
				}
			} else if t == 2 {
				r.R(32)
			}
		}
		if A == 1 {
			d_6d84b4(r, wd.d84b4)
			d_6d84b4(r, wd.d84b4)
		}
		r.R(3) // FUN_1407ef8e4
		if wd.flag0x2c3 == 1 {
			if r.g1() == 0 { // FUN_14076d528 gate
				r.R(20)
				d_6d84b4(r, wd.d84b4) // FUN_14076d6dc largeur runtime
			}
		}
	}
	if A == 1 {
		if r.g1() == 1 { // flag3
			r.R(5)
		}
	}
}

// damageDeserLenPorted : porte le flow FUN_14080c1f8 et renvoie L = bits consommés par le déser
// (à partir de wd.base). param_5==0 (film). Voir en-tête pour le modèle du curseur.
func damageDeserLenPorted(pl []byte, wd deserWidths) int {
	r := &br{pl: pl, bp: wd.base}
	out0 := r.g1()  // step1 out[0]
	out1c := r.g1() // step2 out[0x1c]
	d_1fcf670(r)    // step3
	d_7f2034(r)     // step4
	d_6d00ec(r)     // step5
	d_80d69c(r)     // step6
	d_80dec4(r)     // step7 weapon
	// step8 : 0 bit
	r.R(1) // step9 out[0x1d]
	r.g1() // step10 out[2]
	b := 0
	if out1c == 1 { // step11
		r.g1()
		b = r.g1()
	}
	if b == 1 { // step12
		d_31a0abc(r)
	}
	if out0 == 1 { // step13 chemin court
		d_6d84b4(r, wd.d76dc04)
		return r.bp - wd.base
	}
	f8, s34 := d_80cc68(r)     // step14
	for i := 0; i < s34; i++ { // step15
		r.R(2)
		r.g1()
		r.R(32)
	}
	local98 := 4 // step16
	if f8 == 1 {
		local98 = 0xc
	}
	for j := 0; j < f8; j++ {
		r.R(4)
		if r.g1() == 1 {
			w := bitLen(6)
			r.R(w)
			if s34 < 3 {
				r.R(1)
			} else {
				r.R(4)
			}
			r.R(16)
			d_c1e924(r, local98)
		}
	}
	d_6cd5b8(r, wd)        // step17
	d_8eff64(r, wd)        // step18
	if wd.cVar2Zero == 1 { // step19
		r.R(30)
	}
	if out1c == 1 { // step21
		d_31a0cbc(r, wd)
		// si FUN_141102ed0(0x24)>1 FUN_1406d84b4(R) — état inconnu, ignoré par défaut
	}
	if out1c == 0 { // step22
		r.R(2)         // FUN_14080cb98 -> out[0x1e]
		v20 := r.g1()  // FUN_14080cb50 bit0
		flag := r.g1() // FUN_14080cb50 bit1
		if flag == 1 {
			d_6d84b4(r, wd.d84b4)
		}
		if v20 != 0 {
			// FUN_14320c36c
			if r.g1() == 1 {
				d_6d84b4(r, wd.d84b4)
				d_6d84b4(r, wd.d84b4)
			}
			// FUN_142a40f18
			if r.g1() == 1 {
				d_6d84b4(r, wd.d84b4)
				d_6d84b4(r, wd.d84b4)
			}
		}
	}
	r.R(6)           // step23 out[0x30c]
	if r.g1() == 1 { // step24
		d_6d84b4(r, wd.d84b4)
	}
	r.g1()           // step25 out[4] (param_5==0)
	if r.g1() == 1 { // step26 quat
		d_76e494(r, wd.e494)
	}
	return r.bp - wd.base
}

// weaponAnchor : position de bit du DERNIER couple famille(32)+suffixe(sfx,32) du paquet 0xd2.
// Multi-record (vérifié : 2-3 armes/paquet) : un dégât fatal porte plusieurs enregistrements
// d'arme ; le kill-event suit le DERNIER. Renvoie famPos ou -1 ; déser reprend à famPos+64.
func weaponAnchor(pl []byte) int {
	for b := 0; b+64 <= len(pl)*8; b++ {
		if uint32(bitsAt(pl, b+32, 32)) == sfx {
			return b
		}
	}
	return -1
}

// weaponAnchors : TOUTES les positions (famille+suffixe) du paquet, dans l'ordre.
func weaponAnchors(pl []byte) []int {
	var a []int
	for b := 0; b+64 <= len(pl)*8; b++ {
		if uint32(bitsAt(pl, b+32, 32)) == sfx {
			a = append(a, b)
		}
	}
	return a
}

// damageDeserLenAnchored : reprend le déser à la fin du couple arme (famPos+64) et exécute les
// steps 9→26 (out[0]=1 long path, out[0x1c]=0) pour trouver le bit de fin du déser (= R7 du
// kill-event). Renvoie endBit ; curseur = endBit + 10. Le préambule (steps 1-8, constant ~44 bits)
// est absorbé par l'ancre arme.
func damageDeserLenAnchored(pl []byte, famPos int, wd deserWidths) int {
	r := &br{pl: pl, bp: famPos + 64}
	r.R(1) // step9 out[0x1d]
	r.g1() // step10 out[2]
	b := 0
	if wd.out1c == 1 { // step11
		r.g1()
		b = r.g1()
	}
	if b == 1 { // step12
		d_31a0abc(r)
	}
	// step13 : chemin LONG (out0=1 = long ; le chemin court est écarté empiriquement)
	f8, s34 := d_80cc68(r) // step14
	for i := 0; i < s34; i++ {
		r.R(2)
		r.g1()
		r.R(32)
	}
	local98 := 4
	if f8 == 1 {
		local98 = 0xc
	}
	for j := 0; j < f8; j++ {
		r.R(4)
		if r.g1() == 1 {
			r.R(bitLen(6))
			if s34 < 3 {
				r.R(1)
			} else {
				r.R(4)
			}
			r.R(16)
			d_c1e924(r, local98)
		}
	}
	d_6cd5b8(r, wd) // step17
	d_8eff64(r, wd) // step18
	if wd.cVar2Zero == 1 {
		r.R(30) // step19
	}
	// step20 : if out[0x2dd]==0 { g=R1; si g R6; R6 }
	if b == 0 {
		if r.g1() == 1 {
			r.R(6)
		}
		r.R(6)
	}
	if wd.out1c == 1 { // step21
		d_31a0cbc(r, wd)
	} else { // step22
		r.R(2)        // FUN_14080cb98
		v20 := r.g1() // FUN_14080cb50 bit0
		flag := r.g1()
		if flag == 1 {
			d_6d84b4(r, wd.d84b4)
		}
		if v20 != 0 {
			if r.g1() == 1 {
				d_6d84b4(r, wd.d84b4)
				d_6d84b4(r, wd.d84b4)
			}
			if r.g1() == 1 {
				d_6d84b4(r, wd.d84b4)
				d_6d84b4(r, wd.d84b4)
			}
		}
	}
	r.R(6) // step23
	if r.g1() == 1 {
		d_6d84b4(r, wd.d84b4) // step24
	}
	r.g1() // step25
	if r.g1() == 1 {
		d_76e494(r, wd.e494) // step26 quat
	}
	return r.bp
}

// runDeserProbe3 : ancré arme + flow post-arme complet ; distribution des résidus vs curseur vrai.
func runDeserProbe3(m string, wd deserWidths) {
	pkts := loadFatalPackets(m)
	res := map[int]int{}
	exact := 0
	nD2 := 0
	for _, p := range pkts {
		if p.marker != 0xd2 {
			continue
		}
		fam := weaponAnchor(p.pl)
		if fam < 0 {
			continue
		}
		fam = weaponAnchorLast(p.pl, p.cursor)
		nD2++
		pred := damageDeserLenAnchored(p.pl, fam, wd) + 10
		d := pred - p.cursor
		res[d]++
		if d == 0 {
			exact++
		}
	}
	fmt.Printf("=== DESERPROBE3 %s : EXACT %d/%d ===\n", m, exact, nD2)
	type kv struct{ k, v int }
	var a []kv
	for k, v := range res {
		a = append(a, kv{k, v})
	}
	sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
	fmt.Printf("résidus (pred-curseur) : ")
	for i := 0; i < 20 && i < len(a); i++ {
		fmt.Printf("%d:%d ", a[i].k, a[i].v)
	}
	fmt.Println()
}

// runDeserProbe2 : diagnostic ANCRÉ — bp=famPos+64, décode out[0x34]/out[0xf8] (FUN_14080cc68) et
// trace la position atteinte à chaque bloc vs le curseur vrai. out1c=0 supposé (branche step22).
func runDeserProbe2(m string) {
	pkts := loadFatalPackets(m)
	fmt.Printf("=== DESERPROBE2 (ancré arme) %s ===\n", m)
	n := 0
	f8hist, s34hist := map[int]int{}, map[int]int{}
	postLen := map[int]int{} // (curseur - (famPos+64)) = bits entre fin arme et field0
	for _, p := range pkts {
		if p.marker != 0xd2 || n >= 200 {
			continue
		}
		fam := weaponAnchor(p.pl)
		if fam < 0 {
			continue
		}
		start := fam + 64
		r := &br{pl: p.pl, bp: start + 2} // +2 : step9 R1 + step10 R1 (out1c=0 -> pas step11/12)
		f8, s34 := d_80cc68(r)
		f8hist[f8]++
		s34hist[s34]++
		postLen[p.cursor-start]++
		n++
	}
	top := func(h map[int]int, label string) {
		type kv struct{ k, v int }
		var a []kv
		for k, v := range h {
			a = append(a, kv{k, v})
		}
		sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
		fmt.Printf("%s : ", label)
		for i := 0; i < 15 && i < len(a); i++ {
			fmt.Printf("%d:%d ", a[i].k, a[i].v)
		}
		fmt.Println()
	}
	top(f8hist, "out[0xf8]")
	top(s34hist, "out[0x34] (nb sections)")
	top(postLen, "curseur-(fam+64) [post-arme]")
}

// runDeserProbe : diagnostic — distribution de L vrai (=curseur-20), out[0], out[0x1c], et
// prédiction avec largeurs par défaut. Sert à comprendre les chemins avant calibration.
func runDeserProbe(m string) {
	pkts := loadFatalPackets(m)
	fmt.Printf("=== DESERPROBE %s : %d paquets fatals ===\n", m, len(pkts))
	d2 := 0
	for _, p := range pkts {
		if p.marker == 0xd2 {
			d2++
		}
	}
	fmt.Printf("dont 0xd2 : %d\n", d2)
	wd := defaultWidths()
	var trueLs []int
	out0hist, out1chist := map[int]int{}, map[int]int{}
	exact, near := 0, 0
	for _, p := range pkts {
		if p.marker != 0xd2 {
			continue
		}
		trueL := p.cursor - 20
		trueLs = append(trueLs, trueL)
		out0hist[int(bitsAt(p.pl, wd.base, 1))]++
		out1chist[int(bitsAt(p.pl, wd.base+1, 1))]++
		pred := damageDeserLenPorted(p.pl, wd) + 20
		if pred == p.cursor {
			exact++
		}
		if abs(pred-p.cursor) <= 4 {
			near++
		}
	}
	sort.Ints(trueLs)
	if len(trueLs) > 0 {
		fmt.Printf("L vrai (curseur-20) : min=%d méd=%d max=%d\n", trueLs[0], trueLs[len(trueLs)/2], trueLs[len(trueLs)-1])
	}
	fmt.Printf("out[0]@base : %v | out[0x1c]@base+1 : %v\n", out0hist, out1chist)
	fmt.Printf("défaut widths : EXACT %d/%d, |Δ|<=4 %d/%d\n", exact, d2, near, d2)
	// échantillon de résidus
	fmt.Println("échantillon (curseur vrai, pred défaut, Δ) :")
	n := 0
	for _, p := range pkts {
		if p.marker != 0xd2 || n >= 20 {
			continue
		}
		pred := damageDeserLenPorted(p.pl, wd) + 20
		fmt.Printf("  cur=%4d pred=%4d Δ=%+d  out0=%d out1c=%d\n", p.cursor, pred, pred-p.cursor,
			int(bitsAt(p.pl, wd.base, 1)), int(bitsAt(p.pl, wd.base+1, 1)))
		n++
	}
}

// locateKillEventCursor : localise le kill-event embarqué dans un paquet de dégât fatal 0xd2.
// Renvoie le CURSEUR (position de bit de field0) ou -1. Méthode : premier R7=0x55(85) suivi d'un
// kill-event valide (validKE, grammaire corrigée), en partant de la fin du 1er couple arme
// (famille+suffixe) sinon du bit 100. Exactitude mesurée vs oracle CE : 79/80 = 99%.
func locateKillEventCursor(pl []byte) int {
	// Marker-agnostic : ne PAS ancrer sur le suffixe d'arme (weaponAnchor+64). Chez les frères
	// (0xC2/0xC3/0xE9...) le kill-event PRÉCÈDE l'enregistrement d'arme, donc fam+64 survole le
	// vrai curseur. Le 1er R7=0x55(85)+validKE depuis un plancher bas est le vrai curseur
	// (prouvé : 0xD2 79/80, 0xC2 6/6, 0xC3 21/21).
	// Plancher = keFloor : TOUS les curseurs vrais de l'oracle sont >= 155 (E9 min=155, puis C2=210,
	// D2=260, ...). Un plancher à 140 écarte les faux R7=85+validKE précoces (grappe 0xC0 à bit 124,
	// pur artefact d'un champ fixe des paquets 0xC0 non-fataux) sans perdre aucun vrai kill-event.
	c := keCandidates(pl, keFloor, len(pl)*8)
	if len(c) == 0 {
		return -1
	}
	return c[0]
}

// damageDeserLen : longueur (bits) du déser de dégât L, telle que curseur = 20 + L. Autorité =
// le locator grammatical (curseur - 20). Le port pas-à-pas du flow (damageDeserLenAnchored) sert
// à confirmer/narrower mais la grammaire kill-event corrigée est décisive (voir en-tête).
func damageDeserLen(pl []byte) int {
	c := locateKillEventCursor(pl)
	if c < 0 {
		return -1
	}
	return c - 20
}

// keReadOpt : readOpt du kill-event (R1 gate ; si 0 -> R5 index local = 2*idx).
func keReadOpt(pl []byte, bp int) (int, int) {
	if bp < 0 || bp>>3 >= len(pl) {
		return -2, bp
	}
	if bitsAt(pl, bp, 1) == 0 {
		return int(bitsAt(pl, bp+1, 5)), bp + 6
	}
	return -1, bp + 1
}

// validKE : la grammaire kill-event tient-elle à field0=b ? [readOpt vic][readOpt kil][R32][R1]
// [readOpt assist][R32], vic/kil pairs distincts < 16, assist absent ou pair < 16.
// validKE : grammaire kill-event à field0=b. field0/field1 = index joueur DIRECT (0..15, ici 0..7
// pour 8 joueurs) via readOpt (gate 1 bit + R5) — PAS 2*idx (corrigé via dump oracle). victime !=
// tueur, tous deux présents. field2=R32 (souvent petit), field3=R1, assist=readOpt.
func validKE(pl []byte, b int) bool {
	v, b2 := keReadOpt(pl, b)
	k, b3 := keReadOpt(pl, b2)
	if v < 0 || k < 0 || v >= 16 || k >= 16 || v == k {
		return false
	}
	f2 := bitsAt(pl, b3, 32) // field2 : petit en pratique
	if f2 > 0xffff {
		return false
	}
	a, _ := keReadOpt(pl, b3+33)
	return a == -1 || (a >= 0 && a < 16)
}

// decodeKE : décode (victime=field0, tueur=field1) au curseur c.
func decodeKE(pl []byte, c int) (int, int) {
	v, b2 := keReadOpt(pl, c)
	k, _ := keReadOpt(pl, b2)
	return v, k
}

// locateKillEventAnchored : à partir de l'ancre arme (famPos+64), cherche le PREMIER R7=0x55 (=85)
// dont field0 (à +10) satisfait la grammaire kill-event. Renvoie le curseur (field0) ou -1.
func locateKillEventAnchored(pl []byte, famPos, minOff int) int {
	start := famPos + 64 + minOff
	for x := start; x+17 <= len(pl)*8; x++ {
		if bitsAt(pl, x, 7) != 85 {
			continue
		}
		if validKE(pl, x+10) {
			return x + 10
		}
	}
	return -1
}

// locateCombined : porte le déser (prédiction endBit) puis SNAP vers l'avant sur le premier R7=85
// + kill-event valide à partir de endBit-margin. Le déser saute les faux R7 précoces ; le snap
// corrige le résidu (le port sous-estime). Renvoie le curseur (field0) ou -1.
func locateCombined(pl []byte, fam int, wd deserWidths, margin int) int {
	endBit := damageDeserLenAnchored(pl, fam, wd)
	start := endBit - margin
	if start < fam+64 {
		start = fam + 64
	}
	for x := start; x+17 <= len(pl)*8; x++ {
		if bitsAt(pl, x, 7) != 85 {
			continue
		}
		if validKE(pl, x+10) {
			return x + 10
		}
	}
	return -1
}

// runAnchorScan : teste (a) le locator naïf arme->R7 et (b) le locator combiné port+snap.
func runAnchorScan(m string) {
	pkts := loadFatalPackets(m)
	best, _ := calibrateWidths(pkts)
	fmt.Printf("=== ANCHORSCAN %s ===\n", m)
	fmt.Println("(a) naïf arme -> premier R7=85 valide :")
	for _, minOff := range []int{0, 150, 190} {
		exact, found, n := 0, 0, 0
		for _, p := range pkts {
			if p.marker != 0xd2 {
				continue
			}
			fam := weaponAnchor(p.pl)
			if fam < 0 {
				continue
			}
			n++
			loc := locateKillEventAnchored(p.pl, fam, minOff)
			if loc >= 0 {
				found++
				if loc == p.cursor {
					exact++
				}
			}
		}
		fmt.Printf("  minOff=%3d : EXACT %d/%d (trouvés %d)\n", minOff, exact, n, found)
	}
	fmt.Println("(b) combiné port+snap (marge variable) :")
	for _, margin := range []int{0, 3, 8, 16, 32} {
		exact, found, n := 0, 0, 0
		for _, p := range pkts {
			if p.marker != 0xd2 {
				continue
			}
			fam := weaponAnchor(p.pl)
			if fam < 0 {
				continue
			}
			n++
			loc := locateCombined(p.pl, fam, best, margin)
			if loc >= 0 {
				found++
				if loc == p.cursor {
					exact++
				}
			}
		}
		fmt.Printf("  margin=%2d : EXACT %d/%d (trouvés %d)\n", margin, exact, n, found)
	}
}

// anchored : paquet 0xd2 avec ancre arme précalculée (évite de re-scanner dans la boucle).
type anchored struct {
	pl     []byte
	fam    int
	cursor int
}

func prepAnchored(pkts []fatalPkt) []anchored {
	var a []anchored
	for _, p := range pkts {
		if p.marker != 0xd2 {
			continue
		}
		// ancre = DERNIÈRE arme AVANT le curseur (guidée oracle) : le kill-event suit le dernier
		// enregistrement d'arme sérialisé avant lui. Révèle la qualité réelle du tail porté.
		fam := weaponAnchorLast(p.pl, p.cursor)
		if fam < 0 {
			continue
		}
		a = append(a, anchored{p.pl, fam, p.cursor})
	}
	return a
}

// scoreAnchored : nb de paquets dont curseur prédit (ancré) == curseur vrai.
func scoreAnchored(as []anchored, wd deserWidths) int {
	ok := 0
	for _, a := range as {
		if damageDeserLenAnchored(a.pl, a.fam, wd)+10 == a.cursor {
			ok++
		}
	}
	return ok
}

// calibrateWidths : balayage des largeurs quantized-runtime + flags d'état contre l'oracle.
func calibrateWidths(pkts []fatalPkt) (deserWidths, int) {
	as := prepAnchored(pkts)
	best := defaultWidths()
	bestScore := -1
	for _, o1c := range []int{0, 1} {
		for _, eff := range []int{0, 1, 2} {
			for _, cv := range []int{0, 1} {
				for _, fl := range []int{0, 1} {
					for d84 := 1; d84 <= 40; d84++ {
						for e494 := 1; e494 <= 120; e494++ {
							wd := deserWidths{base: 10, d84b4: d84, c1e924: 8, e494: e494, d3140: 8,
								d76dc04: 8, eff64Type: eff, cVar2Zero: cv, flag0x2c3: fl, out1c: o1c}
							if s := scoreAnchored(as, wd); s > bestScore {
								bestScore, best = s, wd
							}
						}
					}
				}
			}
		}
	}
	// affine c1e924 (utilisé seulement quand out[0xf8]>0, minoritaire) autour du meilleur.
	for c := 1; c <= 32; c++ {
		wd := best
		wd.c1e924 = c
		if s := scoreAnchored(as, wd); s > bestScore {
			bestScore, best = s, wd
		}
	}
	return best, bestScore
}

// runDeserLen : calibre L contre l'oracle CE et rapporte la précision + la distribution des résidus.
func runDeserLen(m string) {
	pkts := loadFatalPackets(m)
	nD2 := 0
	for _, p := range pkts {
		if p.marker == 0xd2 && weaponAnchor(p.pl) >= 0 {
			nD2++
		}
	}
	// (1) L via le LOCATOR grammatical (autorité) : L = damageDeserLen(pl) = curseur - 20.
	locExact := 0
	for _, p := range pkts {
		if p.marker == 0xd2 && weaponAnchor(p.pl) >= 0 && damageDeserLen(p.pl)+20 == p.cursor {
			locExact++
		}
	}
	fmt.Printf("=== DESERLEN %s ===\n", m)
	fmt.Printf(">>> L EXACT via locator grammatical (damageDeserLen+20==curseur) : %d/%d = %.0f%%\n",
		locExact, nD2, float64(locExact)*100/float64(max(nD2, 1)))
	// (2) port pas-à-pas du flow (confirmation ; largeurs quantized calibrées) :
	best, score := calibrateWidths(pkts)
	fmt.Printf("port pas-à-pas (damageDeserLenPorted) : %d/%d = %.0f%% ; largeurs d84b4=%d e494=%d "+
		"eff64Type=%d cVar2Zero=%d flag0x2c3=%d out1c=%d\n",
		score, nD2, float64(score)*100/float64(max(nD2, 1)),
		best.d84b4, best.e494, best.eff64Type, best.cVar2Zero, best.flag0x2c3, best.out1c)
	runDeserProbe3(m, best)
}

// preWeaponEnd : simule steps 1-6 depuis base, renvoie le bit de départ de l'arme (step7).
func preWeaponEnd(pl []byte, base int) int {
	r := &br{pl: pl, bp: base}
	r.g1() // out0
	r.g1() // out1c
	d_1fcf670(r)
	d_7f2034(r)
	d_6d00ec(r)
	d_80d69c(r)
	return r.bp
}

// runDeserAnchor : ancre le début du déser. Pour chaque base candidate, compte les paquets 0xd2 où
// (a) le flow steps1-6 amène l'arme step7 sur une famille connue + suffixe sfx, et (b) mesure aussi
// la position brute où (famille connue||suffixe) apparaît, pour recouper.
func runDeserAnchor(m string) {
	pkts := loadFatalPackets(m)
	h32 := map[uint32]string{}
	// note : analysis importé dans main.go ; reconstruire la map ici via bitsAt scan direct.
	_ = h32
	fmt.Printf("=== DESERANCHOR %s ===\n", m)
	// position BRUTE du suffixe sfx dans chaque paquet 0xd2 (ancre indépendante du flow).
	sfxPosHist := map[int]int{}
	famPosHist := map[int]int{}
	nD2 := 0
	for _, p := range pkts {
		if p.marker != 0xd2 {
			continue
		}
		nD2++
		for b := 0; b+64 <= len(p.pl)*8; b++ {
			if uint32(bitsAt(p.pl, b+32, 32)) == sfx {
				sfxPosHist[b+32]++ // position du suffixe
				famPosHist[b]++    // position de la famille (juste avant)
				break
			}
		}
	}
	fmt.Printf("0xd2 : %d\n", nD2)
	top := func(h map[int]int, label string) {
		type kv struct{ k, v int }
		var a []kv
		for k, v := range h {
			a = append(a, kv{k, v})
		}
		sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
		fmt.Printf("%s (top) : ", label)
		for i := 0; i < 12 && i < len(a); i++ {
			fmt.Printf("%d:%d ", a[i].k, a[i].v)
		}
		fmt.Println()
	}
	top(famPosHist, "position famille brute (1er suffixe)")
	// flow-based : pour base 0..20, combien de paquets ont step7 == position famille brute.
	fmt.Println("--- base sweep : step7(base) == position famille brute ---")
	for base := 0; base <= 24; base++ {
		match := 0
		for _, p := range pkts {
			if p.marker != 0xd2 {
				continue
			}
			w7 := preWeaponEnd(p.pl, base)
			// position famille brute pour ce paquet
			famPos := -1
			for b := 0; b+64 <= len(p.pl)*8; b++ {
				if uint32(bitsAt(p.pl, b+32, 32)) == sfx {
					famPos = b
					break
				}
			}
			if famPos >= 0 && w7 == famPos {
				match++
			}
		}
		fmt.Printf("  base=%2d : step7==fam %d/%d\n", base, match, nD2)
	}
}

// runDeserSplit : par paquet 0xd2 : résidu (port calibré), (f8,s34) décodés, taille payload.
// But : comprendre les gros sous-estimés (loops non déclenchés).
func runDeserSplit(m string) {
	pkts := loadFatalPackets(m)
	best, _ := calibrateWidths(pkts)
	fmt.Printf("=== DESERSPLIT %s (out1c=%d d84b4=%d e494=%d) ===\n", m, best.out1c, best.d84b4, best.e494)
	type row struct{ res, f8, s34, plen, cur int }
	var rows []row
	for _, p := range pkts {
		if p.marker != 0xd2 {
			continue
		}
		fam := weaponAnchor(p.pl)
		if fam < 0 {
			continue
		}
		r := &br{pl: p.pl, bp: fam + 64 + 2}
		if best.out1c == 1 {
			r.bp += 2
		}
		f8, s34 := d_80cc68(r)
		pred := damageDeserLenAnchored(p.pl, fam, best) + 10
		rows = append(rows, row{pred - p.cursor, f8, s34, len(p.pl) * 8, p.cursor})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].res > rows[j].res })
	for _, r := range rows {
		fmt.Printf("  res=%+5d f8=%d s34=%d plen=%d cur=%d\n", r.res, r.f8, r.s34, r.plen, r.cur)
	}
}

// weaponAnchorLast : dernière position (famille+suffixe) AVANT limit bits (ou fin si limit<0).
func weaponAnchorLast(pl []byte, limit int) int {
	last := -1
	hi := len(pl) * 8
	if limit >= 0 && limit < hi {
		hi = limit
	}
	for b := 0; b+64 <= hi; b++ {
		if uint32(bitsAt(pl, b+32, 32)) == sfx {
			last = b
		}
	}
	return last
}

// Classes de dégât par variant_name (champ 32 bits à famPos+32, RE Ghidra FUN_14080c1f8/FUN_14080dec4).
// Le variant_name discrimine la CAUSE du dégât — c'est la clé doctrine-correcte : l'arme d'un kill = la
// SOURCE DE DÉGÂT fatale, quelle que soit sa classe, JAMAIS le held-weapon.
const (
	dmgOther   = -1
	dmgFirearm = 0
	dmgMelee   = 1
	dmgGrenade = 2
)

// Préfixes de CLASSE (24 bits de poids fort du variant_name ; l'octet bas = index de sous-variante).
// RE : le variant_name est un tag GUID discret [groupe 24b = classe][index 8b]. Appliqué à la position
// DÉTERMINISTE du préambule (pas au scan bit-à-bit), le préfixe 24 bits est fiable.
const (
	firearmVar24 = 0x42C967 // arme-à-feu (famille high32 = l'arme, épée/marteau inclus)
	meleeVar24   = 0x592CF3 // mêlée mains-nues / pummel / assassinat
	grenadeVar24 = 0x164B3C // grenade
)

// damageClass classe un variant_name 32 bits par préfixe 24 bits (firearm / mêlée / grenade / autre).
func damageClass(variant uint32) int {
	switch variant >> 8 {
	case firearmVar24:
		return dmgFirearm
	case meleeVar24:
		return dmgMelee
	case grenadeVar24:
		return dmgGrenade
	default:
		return dmgOther
	}
}

// weaponName nomme une famille d'arme (S6/+0x10 = tag source de dégât = LA cause). h32 = catalogue
// (high32 -> nom, + alias décalage 1-bit). Inconnue -> "fam-XXXXXXXX" (candidat mêlée/grenade/exotique).
func weaponName(family uint32, h32 map[uint32]string) string {
	if nm, ok := h32[family]; ok {
		return nm
	}
	return fmt.Sprintf("fam-%08X", family)
}

// classifyDmg nomme la cause d'un record depuis (famille, variant) : arme nommée (épée/marteau inclus,
// via le catalogue) / "Mêlée" / "Grenade" / "cause-<variant>". usable=false pour la classe inconnue
// (candidat splatter/environnement à cataloguer, exclu de l'attribution warp).
func classifyDmg(family, variant uint32, h32 map[uint32]string) (string, bool) {
	switch damageClass(variant) {
	case dmgFirearm:
		if nm, ok := h32[family]; ok {
			return nm, true
		}
		return fmt.Sprintf("fam-%08X", family), true
	case dmgMelee:
		return "Mêlée", true
	case dmgGrenade:
		return "Grenade", true
	default:
		return fmt.Sprintf("cause-%08X", variant), false
	}
}

// dmgRecord = préambule déterministe d'un record de dégât (steps 1-7 du déser FUN_14080c1f8).
type dmgRecord struct {
	attacker   int    // index joueur 0..7 (readOpt5) ou -1 si absent
	family     uint32 // famille high32 de l'arme, ou 0xffffffff si absente
	variant    uint32 // variant_name (discrimine la classe de cause)
	variantPos int    // position bit du variant
	endPos     int    // position bit après le variant
}

// parsePreamble parse le PRÉAMBULE DÉTERMINISTE (aucune largeur quantized-runtime) d'un record de dégât à
// partir de `base`. C'est la lecture film-INDÉPENDANTE de la cause (remplace la position-fixe bp=41).
// Grammaire (MSB-first, RE Ghidra FUN_14080c1f8) : step1 R1 ; step2 R1 ; step3 R8 ; step4 attaquant
// readOpt5 (g1, si 0 -> R5) ; step5 dmgIndex (g1, si 0 -> R2) ; step6 famille optionnelle (g1 présence,
// si 1 -> R32) ; step7 variant R32 toujours.
func parsePreamble(pl []byte, base int) (dmgRecord, bool) {
	var r dmgRecord
	hi := len(pl) * 8
	bp := base
	if base < 0 || base+13 > hi {
		return r, false
	}
	rd := func(n int) uint32 { v := uint32(bitsAt(pl, bp, n)); bp += n; return v }
	rd(1) // step1 out0
	rd(1) // step2 out1c
	rd(8) // step3 fixe (R7+R1)
	if rd(1) == 0 {
		r.attacker = int(rd(5)) // step4 attaquant présent
	} else {
		r.attacker = -1
	}
	if rd(1) == 0 {
		rd(2) // step5 dmgIndex présent
	}
	if rd(1) == 1 {
		r.family = rd(32) // step6 famille présente
	} else {
		r.family = 0xffffffff
	}
	if bp+32 > hi {
		return r, false
	}
	r.variantPos = bp
	r.variant = rd(32) // step7 variant toujours
	r.endPos = bp
	return r, true
}

// nonFirearmAnchorLast : position (bit) du DERNIER enregistrement de dégât MÊLÉE ou GRENADE avant limit,
// + sa classe. Appelé UNIQUEMENT quand aucun record arme-à-feu (0x42C9679F, précis) n'existe avant le
// curseur = kill non-arme-à-feu. Ne pas mélanger avec le scan firearm : les valeurs variant mêlée/grenade
// (32 bits) collisionnent avec des octets structurels au scan bit-à-bit, donc firearm reste prioritaire.
func nonFirearmAnchorLast(pl []byte, limit int) (int, int) {
	last, cls := -1, dmgOther
	hi := len(pl) * 8
	if limit >= 0 && limit < hi {
		hi = limit
	}
	for b := 0; b+64 <= hi; b++ {
		if c := damageClass(uint32(bitsAt(pl, b+32, 32))); c == dmgMelee || c == dmgGrenade {
			last, cls = b, c
		}
	}
	return last, cls
}

// runSfxCount : nb d'occurrences du suffixe par paquet + delta (curseur - dernierFam-64) et
// (curseur - premierFam-64). But : confirmer multi-record + trouver l'ancre stable.
func runSfxCount(m string) {
	pkts := loadFatalPackets(m)
	fmt.Printf("=== SFXCOUNT %s ===\n", m)
	deltaLast, deltaFirst, nSfxHist := map[int]int{}, map[int]int{}, map[int]int{}
	for _, p := range pkts {
		if p.marker != 0xd2 {
			continue
		}
		cnt := 0
		for b := 0; b+64 <= len(p.pl)*8; b++ {
			if uint32(bitsAt(p.pl, b+32, 32)) == sfx {
				cnt++
			}
		}
		nSfxHist[cnt]++
		first := weaponAnchor(p.pl)
		// dernier suffixe AVANT le curseur vrai
		lastBefore := weaponAnchorLast(p.pl, p.cursor)
		if first >= 0 {
			deltaFirst[p.cursor-(first+64)]++
		}
		if lastBefore >= 0 {
			deltaLast[p.cursor-(lastBefore+64)]++
		}
	}
	top := func(h map[int]int, label string) {
		type kv struct{ k, v int }
		var a []kv
		for k, v := range h {
			a = append(a, kv{k, v})
		}
		sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
		fmt.Printf("%s : ", label)
		for i := 0; i < 16 && i < len(a); i++ {
			fmt.Printf("%d:%d ", a[i].k, a[i].v)
		}
		fmt.Println()
	}
	top(nSfxHist, "nb suffixes/paquet")
	top(deltaFirst, "curseur - (PREMIER fam+64)")
	top(deltaLast, "curseur - (DERNIER fam avant curseur +64)")
}

// locateByAnchorTail : essaie chaque ancre arme (dans l'ordre) ; retient le PREMIER dont le tail
// porté (damageDeserLenAnchored) tombe exactement sur R7=0x55 + kill-event valide. Robuste au
// multi-record (fatale = 1re arme pour la majorité, arme plus tardive sinon).
func locateByAnchorTail(pl []byte, wd deserWidths) int {
	for _, a := range weaponAnchors(pl) {
		end := damageDeserLenAnchored(pl, a, wd)
		if end >= 0 && end+7 <= len(pl)*8 && bitsAt(pl, end, 7) == 85 && validKE(pl, end+10) {
			return end + 10
		}
	}
	return -1
}

// keCandidates : toutes les positions field0 candidates dans [lo,hi[. Ancre = motif 10 bits 0x2A8 =
// R7 type=0x55(85, 7 bits) + 3 bits de gate du dispatcher TOUS à 0 (constant 134/134 aux curseurs
// oracle). Exiger les 3 gates=000 (pas seulement le type 7 bits) écarte des faux R7=85 dont les gates
// sont non nuls. field0 = x+10.
func keCandidates(pl []byte, lo, hi int) []int {
	var c []int
	if lo < 0 {
		lo = 0
	}
	for x := lo; x+17 <= hi && x+17 <= len(pl)*8; x++ {
		if bitsAt(pl, x, 10) == 0x2A8 && validKE(pl, x+10) {
			c = append(c, x+10)
		}
	}
	return c
}

// runLocate : compare des stratégies de localisation (validKE corrigé).
func runLocate(m string) {
	pkts := loadFatalPackets(m)
	best, _ := calibrateWidths(pkts)
	nKE, nD2 := 0, 0
	candHist := map[int]int{}
	rankHist := map[int]int{} // rang (index) du vrai curseur parmi les candidats triés
	exFirstAll, exFirstW, exNearest, exTail := 0, 0, 0, 0
	for _, p := range pkts {
		if p.marker != 0xd2 {
			continue
		}
		fam := weaponAnchor(p.pl)
		if fam < 0 {
			continue
		}
		nD2++
		if validKE(p.pl, p.cursor) {
			nKE++
		}
		// candidats sur tout le paquet
		cands := keCandidates(p.pl, fam+64, len(p.pl)*8)
		candHist[len(cands)]++
		for i, c := range cands {
			if c == p.cursor {
				rankHist[i]++
			}
		}
		// (A) premier candidat >= bit 100 (style pipeline actuel)
		if a := keCandidates(p.pl, 100, len(p.pl)*8); len(a) > 0 && a[0] == p.cursor {
			exFirstAll++
		}
		// (B) premier candidat après l'ancre 1re arme +64
		if len(cands) > 0 && cands[0] == p.cursor {
			exFirstW++
		}
		// (C) candidat le plus proche de la prédiction du port (tail depuis 1re arme)
		pred := damageDeserLenAnchored(p.pl, fam, best) + 10
		bestC, bd := -1, 1<<30
		for _, c := range keCandidates(p.pl, fam+64, len(p.pl)*8) {
			if d := abs(c - pred); d < bd {
				bd, bestC = d, c
			}
		}
		if bestC == p.cursor {
			exNearest++
		}
		// (D) locateByAnchorTail
		if locateByAnchorTail(p.pl, best) == p.cursor {
			exTail++
		}
	}
	// généralisation TOUS marqueurs (pas seulement 0xd2)
	allN, allExact := 0, 0
	markerExact := map[byte][2]int{}
	for _, p := range pkts {
		allN++
		cur := locateKillEventCursor(p.pl)
		e := markerExact[p.marker]
		e[1]++
		if cur == p.cursor {
			allExact++
			e[0]++
		}
		markerExact[p.marker] = e
	}
	fmt.Printf("=== LOCATE %s (n=%d) ===\n", m, nD2)
	fmt.Printf("validKE@curseur vrai : %d/%d\n", nKE, nD2)
	fmt.Printf("locateKillEventCursor EXACT tous marqueurs : %d/%d\n", allExact, allN)
	fmt.Printf("  par marqueur : ")
	for mk, e := range markerExact {
		fmt.Printf("0x%02X:%d/%d ", mk, e[0], e[1])
	}
	fmt.Println()
	fmt.Printf("nb candidats R7=85+validKE / paquet : %v\n", sortMapDesc(candHist))
	fmt.Printf("rang du vrai curseur parmi candidats (0=premier) : %v\n", sortMapDesc(rankHist))
	fmt.Printf("(A) 1er candidat depuis bit100      : %d/%d\n", exFirstAll, nD2)
	fmt.Printf("(B) 1er candidat depuis 1re arme+64 : %d/%d\n", exFirstW, nD2)
	fmt.Printf("(C) candidat le + proche du port    : %d/%d\n", exNearest, nD2)
	fmt.Printf("(D) locateByAnchorTail (port exact) : %d/%d\n", exTail, nD2)
}

func sortMapDesc(h map[int]int) string {
	type kv struct{ k, v int }
	var a []kv
	for k, v := range h {
		a = append(a, kv{k, v})
	}
	sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
	s := ""
	for _, x := range a {
		s += fmt.Sprintf("%d:%d ", x.k, x.v)
	}
	return s
}

// runPipeDet : DIAGNOSTIC détecteur. Énumère tous les paquets damage-family sz>=700 (mes détections
// candidates), les classe TRUE-fatal (dans l'oracle CE par (chunk,off)) vs FAUX-POSITIF, et pour
// chacun montre le curseur localisé vs le curseur vrai + la paire décodée. Révèle si les paires
// fausses viennent d'une DÉTECTION erronée (paquet non-fatal) ou d'une MAUVAISE LOCALISATION.
func runPipeDet(m string) {
	pkts := loadFatalPackets(m)
	// index oracle par (chunk,off) -> curseur vrai
	oracleCur := map[[2]int]int{}
	for _, p := range pkts {
		oracleCur[[2]int{p.ch, p.off}] = p.cursor
	}
	cache := root + "/" + m
	dmgMk := map[byte]bool{0xD2: true, 0xC0: true, 0xC2: true, 0xC3: true, 0xCA: true, 0xD3: true, 0xE9: true}
	fmt.Printf("=== PIPEDET %s (oracle : %d paquets fatals physiques distincts) ===\n", m, len(oracleCur))
	// complétude du validateur truePairs (_align_kill.bin) : nb enregistrements, paires distinctes,
	// et paires décodées aux CURSEURS VRAIS de l'oracle qui NE sont PAS dans truePairs (= trous du
	// validateur, pas des erreurs de décodage).
	if kcb, err := os.ReadFile(`c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/` + m + "_align_kill.bin"); err == nil {
		truePairs := map[[2]int]bool{}
		nrec := 0
		for o := 0; o+16 <= len(kcb); o += 16 {
			vic := idxK(binary.LittleEndian.Uint32(kcb[o:]))
			kil := idxK(binary.LittleEndian.Uint32(kcb[o+4:]))
			if kil >= 0 && vic >= 0 {
				truePairs[[2]int{kil, vic}] = true
				nrec++
			}
		}
		missAtTrue := map[[2]int]int{}
		for _, p := range pkts {
			v, b2 := keReadOpt(p.pl, p.cursor)
			k, _ := keReadOpt(p.pl, b2)
			if v >= 0 && k >= 0 && !truePairs[[2]int{k, v}] {
				missAtTrue[[2]int{k, v}]++
			}
		}
		fmt.Printf("truePairs : %d enreg. -> %d paires distinctes | paires décodées @curseur VRAI absentes de truePairs : %v\n",
			nrec, len(truePairs), missAtTrue)
	}
	// gate candidat sfxAny>=0 (le paquet contient un enreg. arme/cause 0x42c9679f) : combien de
	// vrais fatals le RATERAIENT ? + borne f2 : quel f2 max aux curseurs vrais ?
	{
		noSfx, maxF2, maxCur := 0, 0, 0
		seen := map[[2]int]bool{}
		var noSfxInfo []string
		for _, p := range pkts {
			if seen[[2]int{p.ch, p.off}] {
				continue
			}
			seen[[2]int{p.ch, p.off}] = true
			if weaponAnchor(p.pl) < 0 {
				noSfx++
				noSfxInfo = append(noSfxInfo, fmt.Sprintf("0x%02X@cur%d", p.marker, p.cursor))
			}
			if p.cursor > maxCur {
				maxCur = p.cursor
			}
			_, b2 := keReadOpt(p.pl, p.cursor)
			_, b3 := keReadOpt(p.pl, b2)
			if f2 := int(bitsAt(p.pl, b3, 32)); f2 > maxF2 {
				maxF2 = f2
			}
		}
		fmt.Printf("vrais fatals physiques SANS enreg. arme (sfxAny<0) : %d/%d %v | f2 max @curseur vrai : %d | curseur vrai max : %d\n",
			noSfx, len(seen), noSfxInfo, maxF2, maxCur)
	}
	type row struct {
		mk              byte
		ch, off, sz     int
		locCur, trueCur int
		vic, kil        int
		isFatal, locOK  bool
	}
	var rows []row
	nCandFatal := map[byte][2]int{} // marqueur -> [détectés, dont vrai-fatal]
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
			po := off
			off += 16 + sz
			if typ != 0 || len(pl) == 0 || !dmgMk[pl[0]] || sz < 700 {
				continue
			}
			loc := locateKillEventCursor(pl)
			if loc < 0 {
				continue
			}
			tc, isFatal := oracleCur[[2]int{ch, po}]
			e := nCandFatal[pl[0]]
			e[0]++
			if isFatal {
				e[1]++
			}
			nCandFatal[pl[0]] = e
			vic, kil := decodeKE(pl, loc)
			rows = append(rows, row{pl[0], ch, po, sz, loc, tc, vic, kil, isFatal, isFatal && loc == tc})
		}
	}
	// distribution de field2 (R32) AU CURSEUR VRAI (oracle) : borne discriminante ?
	f2true := map[int]int{}
	for _, p := range pkts {
		_, b2 := keReadOpt(p.pl, p.cursor)
		_, b3 := keReadOpt(p.pl, b2)
		f2true[int(bitsAt(p.pl, b3, 32))]++
	}
	{
		var ks []int
		for k := range f2true {
			ks = append(ks, k)
		}
		sort.Ints(ks)
		fmt.Printf("field2 @curseur VRAI (oracle, %d paquets) : ", len(pkts))
		for _, k := range ks {
			fmt.Printf("%d:%d ", k, f2true[k])
		}
		fmt.Println()
	}
	// synthèse par marqueur
	fmt.Println("détections (sz>=700 + locate) par marqueur : [détectés, dont vrai-fatal] :")
	for _, mk := range []byte{0xD2, 0xD3, 0xC0, 0xC2, 0xC3, 0xCA, 0xE9} {
		if nCandFatal[mk][0] > 0 {
			fmt.Printf("  0x%02X : %d détectés, %d vrai-fatal\n", mk, nCandFatal[mk][0], nCandFatal[mk][1])
		}
	}
	// détail des faux-positifs (détecté mais PAS dans l'oracle) et des mislocations (fatal mais loc!=vrai)
	fmt.Println("FAUX-POSITIFS (détecté, PAS fatal oracle) :")
	for _, r := range rows {
		if !r.isFatal {
			d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, r.ch))
			pl := d[r.off+16 : r.off+16+r.sz]
			_, b2 := keReadOpt(pl, r.locCur)
			_, b3 := keReadOpt(pl, b2)
			f2 := int(bitsAt(pl, b3, 32))
			famAny := weaponAnchor(pl)
			famBefore := weaponAnchorLast(pl, r.locCur)
			fmt.Printf("  0x%02X ch%d off%d sz=%d locCur=%d kil=%d vic=%d f2=%d sfxAny=%d sfxBefore=%d\n",
				r.mk, r.ch, r.off, r.sz, r.locCur, r.kil, r.vic, f2, famAny, famBefore)
		}
	}
	fmt.Println("MISLOCATIONS (vrai-fatal mais locCur != trueCur) :")
	for _, r := range rows {
		if r.isFatal && !r.locOK {
			fmt.Printf("  0x%02X ch%d off%d sz=%d locCur=%d trueCur=%d kil=%d vic=%d\n", r.mk, r.ch, r.off, r.sz, r.locCur, r.trueCur, r.kil, r.vic)
		}
	}
}

// runKEDump : dump des bits au curseur vrai pour corriger la grammaire du kill-event.
// Affiche 3 gates (cur..cur-3?), et plusieurs décodages readOpt candidats.
func runKEDump(m string) {
	pkts := loadFatalPackets(m)
	fmt.Printf("=== KEDUMP %s (bits @ curseur vrai) ===\n", m)
	n := 0
	for _, p := range pkts {
		if p.marker != 0xd2 || n >= 24 {
			continue
		}
		n++
		c := p.cursor
		// 32 bits à partir du curseur
		var bits string
		for i := 0; i < 40; i++ {
			bits += fmt.Sprintf("%d", bitsAt(p.pl, c+i, 1))
		}
		// readOpt v/k
		v, b2 := keReadOpt(p.pl, c)
		k, b3 := keReadOpt(p.pl, b2)
		fmt.Printf("cur=%4d R7@-10=%d 3gate@-3..0=%d%d%d | bits=%s | rdOpt v=%d k=%d (nextbp=%d)\n",
			c, bitsAt(p.pl, c-10, 7),
			bitsAt(p.pl, c-3, 1), bitsAt(p.pl, c-2, 1), bitsAt(p.pl, c-1, 1),
			bits, v, k, b3)
	}
}
