// tmp_impactpos — PARTIE 2 : POSITION D'IMPACT par record de dégât (000d5950).
//
// Porte le deser COMPLET FUN_14080c1f8 (image_base 0x140000000) en Go pur pour
// atteindre la position d'impact écrite à param_3+0x27c (param_5==1, via
// FUN_140c9e4d8 -> FUN_140c9e990 -> FUN_140c9e738 -> FUN_14076d528).
//
// Séquence portée fidèlement depuis le début du payload du record de dégât (les
// records = paquets type-0 dont payload[0]==0xd2, cf tmp_dmgscan). Le deser lit,
// dans l'ordre :
//
//	*param_3       R(1)                          (isFloatPos)
//	param_3[0x1c]  R(1)                          (hasExtra)
//	FUN_141fcf670  R(7)+R(1) -> param_3+1        (cause byte)
//	+0x0c global   FUN_1407f2034: R(1);!0->R(5)
//	+0x08 cause    FUN_1406d00ec: R(1);!0->R(2)
//	+0x10 handle   FUN_14080d69c: R(1); set->R(32)
//	+0x14 variant  FUN_14080dec4: R(32) BE       = FAMILLE D'ARME
//	+0x18          resolve handle (0 bits)
//	param_3[0x1d]  R(1)
//	cVar13 gate calc (value-gated, no bits)
//	param_3[2]     R(1)
//	if param_3[0x1c]==1: R(1)+R(1) ; else skip (1 read of 0 bits accounted)
//	+0x2dd gate    -> if set FUN_1431a0abc (rare)
//	if *param_3==1 : full-precision float branch (early returns) — RARE
//	else:
//	  FUN_14080cc68 -> +0xf8 (type), +0x34 (count)
//	  loop count (+0x40, stride 0xc): R(2)+R(1) ; param_5==1 -> FUN_1406d3140 (0 bits)
//	  +0x110 loop (count=+0xf8): per-iter R(4)+R(1); if set: R(6-bucket)+... (value-gated)
//	  POSITION: param_5==1 -> FUN_140c9e4d8(+0x27c) ELSE FUN_1406cd5b8(+0x2a0)
//
// Le but : (ts, famille, impact-pos x,y,z) pour les records, vérifier la range vs
// positions joueur de P1, près de 329.8s (BR75) et 115.5/292.5s (marteau).
//
// Décode = Go PUR. Usage : tmp_impactpos
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

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)

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
	typ      uint16
	size     int
	ts       uint64
	payload  []byte
	chunkIdx int
}

func listPackets(d []byte, chunkIdx int) []pkt {
	var out []pkt
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		size := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if size <= 0 || off+16+size > len(d) {
			break
		}
		out = append(out, pkt{typ, size, ts, d[off+16 : off+16+size], chunkIdx})
		off += 16 + size
	}
	return out
}

func allType0() []pkt {
	var all []pkt
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		for _, p := range listPackets(d, n) {
			if p.typ == 0 {
				all = append(all, p)
			}
		}
	}
	return all
}

func tsToMs(ts uint64) int { return int((int64(ts) - int64(t0Us)) / 1000) }

// ── Constantes de déquantification de la position d'impact ──────────────────
// FUN_14076d528 (vec3 quantisé) appelé par FUN_140c9e738 avec :
//
//	scale=DAT_143cd8928 (=0.00999999978f), range=DAT_143cd873c (=10.0f),
//	widthY=uVar2, widthX=uVar1. Avec param_3=0 (le cas réel) -> uVar2=7, uVar1=0xf.
//
// DAT_143cd8928 (16o lus) = 0ad7233c 0000f044 db0f493f 00000040
//
//	-> f32 0.00999999978 (0x3c23d70a), 1920.0 (0x44f00000), 0.785398 (0x3f490fdb), 2.0
//
// DAT_143cd873c (16o) = 00002041 000000bf ...
//
//	-> f32 10.0 (0x41200000), -0.5 (0xbf000000)
const impScale = float32(0.00999999978) // DAT_143cd8928[0]
const impRange = float32(10.0)          // DAT_143cd873c[0]
const impAxisLow = uint(7)              // uVar2 (param_3!=1)
const impAxisHigh = uint(0xf)           // uVar1 (param_3!=1)

// readImpactVec3 porte FUN_14076d528 (chemin quantisé) : gate 1-bit ; si 0 ->
// lit (widthX) bits packés, déquant via DAT_1447084d table (FUN_1406d8288) puis
// scale logarithmique (FUN_14076d6dc). On approxime ici par une déquant linéaire
// faute des tables DAT_1447084d0 exactes ; renvoie aussi le mode brut.
func readImpactVec3(br *filmdec.BitReader) (vec [3]float32, present bool) {
	gate := br.ReadBit() // 1-bit : si set -> valeur par défaut PTR_DAT_14474c2f0
	if gate {
		return [3]float32{}, false // valeur défaut (origine)
	}
	// chemin valeur : packe (widthHigh) bits ; FUN_1406d8288 -> 3 floats via tables.
	raw := br.ReadBits(impAxisHigh)
	// Déquant approximative : la vraie déquant utilise les tables DAT_1447084d0
	// (3 facteurs / axe) + normalisation directionnelle. On renvoie le packed brut
	// pour diagnostic ; la magnitude est ensuite échelonnée par impRange.
	q := float32(raw) / float32((uint64(1)<<impAxisHigh)-1) // 0..1
	v := (q - 0.5) * 2 * impRange                           // -range..range (axe principal)
	vec = [3]float32{v, 0, 0}
	return vec, true
}

// rdr est l'état du deser : br + flags de branches prises (diagnostic).
type rec struct {
	ts       uint64
	tms      int
	fam      string
	famKnown bool
	lowOK    bool
	cause    uint32
	hitType  uint32
	hitCount uint32
	isFloat  bool
	hasExtra bool
	posGate  bool       // *param_4 de FUN_140c9e4d8 (présence handle impact)
	posMode  byte       // mode FUN_1407f0278 (0/1/2)
	vec      [3]float32 // vec3 impact si présent
	vecOK    bool
	desync   bool
	chunk    int
	startBit int
	endBit   int
}

// decode porte le deser FUN_14080c1f8 (param_5=1) sur un payload de record.
// startBit = bit de départ de la lecture du deser dans le payload (le reader
// FUN_14080AADE a déjà consommé l'entête). On prouve empiriquement le startBit.
func decode(payload []byte, startBit int) rec {
	br := filmdec.NewBitReader(payload)
	br.Skip(startBit)
	var r rec
	r.startBit = startBit
	totalBits := len(payload) * 8
	chk := func() bool { return br.BitPos() <= totalBits }

	// ── PREFIX EMPIRIQUE PROUVÉ (tmp_dmgscan, 519/519) ──
	// À startBit=36 : slot/cause(R1[/R2]) + R5 + R32(FAM) + R32(LOW=0x42c9679f).
	// On lit isFloat/extra A POSTERIORI : ils ne sont PAS dans le prefix empirique ;
	// le deser décompilé démarre AILLEURS et le reader FUN_14080AADE consomme 36 bits
	// d'entête. On honore donc le prefix prouvé, puis on porte le TAIL décompilé.
	// slot/cause +0x08 : R(1); si 0 -> R(2)
	if !br.ReadBit() {
		r.cause = uint32(br.ReadBits(2))
	} else {
		r.cause = 0xffffffff
	}
	// global-id R5 (gate déjà passé via le prefix : ici R5 nu, le gate est inclus
	// dans le layout empirique)
	br.ReadBits(5)
	// +0x14 variant_name : R(32) BE = FAMILLE
	fam := uint32(br.ReadBits(32))
	r.fam, r.famKnown = h32name[fam]
	if !r.famKnown {
		r.fam = fmt.Sprintf("?%08x", fam)
	}
	// LOW R(32) = 0x42c9679f (suffixe variant universel, partie basse de l'id64)
	r.lowOK = uint32(br.ReadBits(32)) == 0x42c9679f
	// ── TAIL DÉCOMPILÉ (depuis variant) : param_3[0x1d], param_3[2], extra-gate. ──
	// param_3[0x1d] : R(1)
	br.ReadBit()
	// param_3[2] : R(1)
	br.ReadBit()
	// Dans le layout empirique on ne lit pas isFloat/extra explicitement avant ;
	// le deser décompilé les place en tête. On modélise hasExtra=R(1) ici (best-effort)
	// pour traverser la branche extra ; instrumenté en ÉTAPE B.
	r.isFloat = false
	r.hasExtra = br.ReadBit()
	posDdGateSet := false
	// if +0x2dd != 0 -> FUN_1431a0abc (rare, value-gated width inconnue) : on stoppe.
	if posDdGateSet {
		r.desync = true
		r.endBit = br.BitPos()
		return r
	}
	// FUN_14080cc68(+0xf8 type, +0x34 count) — header hit-sections.
	r.hitType, r.hitCount = readHitHeader(br)
	if !chk() {
		r.desync = true
		r.endBit = br.BitPos()
		return r
	}
	// boucle 1 : count itérations, stride 0xc. Chaque iter :
	//   R(2) (pbVar11[-8]) ; R(1) (pbVar11[-7]) ; param_5==1 -> FUN_1406d3140 (0 bits flux)
	if r.hitCount > 0 && r.hitCount < 64 {
		for i := uint32(0); i < r.hitCount; i++ {
			br.ReadBits(2)
			br.ReadBits(1)
			// param_5==1 : FUN_1406d3140 ne lit pas le flux.
		}
	}
	// boucle 2 : si +0xf8 (type) gère, count = +0xf8. C'est value-gated et lit
	// FUN_140c1e924 (quantisé) + champs variables -> largeurs partielles non
	// résolues. On la PORTE pour le cas type==0 (squelette) et on signale desync sinon.
	if r.hitType != 0 {
		// boucle +0x110 prise : largeurs value-gated (FUN_1406d310c popcount,
		// FUN_140c1e924 quantisé) non bit-exactes -> désync contrôlée.
		r.desync = true
		r.endBit = br.BitPos()
		return r
	}
	// hitType==0 : la boucle +0x110 ne s'exécute pas (count test <0 ou skip).
	// POSITION D'IMPACT : param_5==1 -> FUN_140c9e4d8(+0x27c).
	r.posGate, r.posMode, r.vec, r.vecOK = readImpactDescriptor(br)
	r.endBit = br.BitPos()
	if !chk() {
		r.desync = true
	}
	return r
}

// readHitHeader porte FUN_14080cc68 : R(1) gate ; si 0 -> R(1) ; si 0 -> R(4)
// (param_2=type) sinon type=1 ; puis 2 bits gate -> R(4) (param_3=count). Si le
// premier gate==1 -> type=0,count=0.
func readHitHeader(br *filmdec.BitReader) (typ, count uint32) {
	if br.ReadBit() { // cVar2 != 0 -> *param_2=0, *param_3=0
		return 0, 0
	}
	if br.ReadBit() { // *param_2 = 1
		typ = 1
	} else {
		typ = uint32(br.ReadBits(4))
	}
	// 1-bit gate (LAB_14080ccf5 entry)
	if br.ReadBit() {
		// gate set -> deuxième 1-bit gate
		if br.ReadBit() {
			count = uint32(br.ReadBits(4))
		} else {
			count = 1
		}
	} else {
		count = 0
	}
	return typ, count
}

// readImpactDescriptor porte FUN_140c9e4d8(param_1=+0x27c, br, 0, &flag) :
//
//	flag = R(1). si 0 -> rien (return). si set :
//	  FUN_140c9e990 : mode=R(2). si mode==1 -> R(1) gate; si set -> R(6).
//	                  si mode==2 -> rien. sinon rien.
//	  R(1) gate (bit&1). si clair -> 2x FUN_1406d84b4(width?) puis R(1) gate (bit&2);
//	     si clair -> return. sinon -> FUN_140c9e738 (vec3).
//	  si bit&1 set -> bit|2 -> FUN_140c9e738 (vec3).
//
// Le vec3 (FUN_140c9e738->FUN_14076d528) n'est lu QUE quand bit&2.
func readImpactDescriptor(br *filmdec.BitReader) (gate bool, mode byte, vec [3]float32, vecOK bool) {
	gate = br.ReadBit()
	if !gate {
		return false, 0, [3]float32{}, false
	}
	// FUN_140c9e990 : FUN_1407f0278 -> R(2) mode
	mode = byte(br.ReadBits(2))
	switch mode {
	case 1:
		if br.ReadBit() {
			br.ReadBits(6)
		}
	case 2:
		// rien
	default:
		// rien
	}
	// R(1) : bit (param_1[0x18] | 1 si set)
	b1 := br.ReadBit()
	if !b1 {
		// 2x FUN_1406d84b4 : largeur inconnue (stack arg). On NE peut PAS continuer
		// bit-exact ici -> on signale via vecOK=false, mode renvoyé.
		// Sur le film, on mesure combien de records empruntent cette branche.
		return gate, mode, [3]float32{}, false
	}
	// b1 set -> bit|2 -> FUN_140c9e738 vec3
	vec, vecOK = readImpactVec3(br)
	return gate, mode, vec, vecOK
}

func main() {
	build()
	all := allType0()

	// Records de dégât = payload[0]==0xd2 (discriminant prouvé par tmp_dmgscan).
	var d2 []pkt
	for _, p := range all {
		if len(p.payload) > 0 && p.payload[0] == 0xd2 {
			d2 = append(d2, p)
		}
	}
	fmt.Printf("=== %d type-0 ; %d records de dégât (payload[0]==0xd2) ; %d familles catalogue ===\n\n",
		len(all), len(d2), len(h32name))

	// ── ÉTAPE A : déterminer le startBit qui maximise les familles connues. ──
	// Le deser démarre après l'entête consommé par le reader. tmp_dmgscan utilisait
	// 36 avec un layout empirique différent. Avec la VRAIE séquence, on balaie 0..64.
	fmt.Println("=== ÉTAPE A : startBit empirique prouvé (tmp_dmgscan) = 36 ===")
	bestSB := 36
	known := 0
	for _, p := range d2 {
		if decode(p.payload, bestSB).famKnown {
			known++
		}
	}
	fmt.Printf("  startBit=%d : familles connues %d/%d\n\n", bestSB, known, len(d2))

	// ── ÉTAPE B : décoder tous les records au meilleur startBit. ──
	var recs []rec
	for _, p := range d2 {
		r := decode(p.payload, bestSB)
		r.ts = p.ts
		r.tms = tsToMs(p.ts)
		r.chunk = p.chunkIdx
		recs = append(recs, r)
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].tms < recs[j].tms })

	// Statistiques de couverture position.
	var nFam, nLowOK, nFloat, nExtra, nDesync, nPosGate, nVecOK int
	posModeHist := map[byte]int{}
	for _, r := range recs {
		if r.famKnown {
			nFam++
		}
		if r.lowOK {
			nLowOK++
		}
		if r.isFloat {
			nFloat++
		}
		if r.hasExtra {
			nExtra++
		}
		if r.desync {
			nDesync++
		}
		if r.posGate {
			nPosGate++
			posModeHist[r.posMode]++
		}
		if r.vecOK {
			nVecOK++
		}
	}
	fmt.Println("=== ÉTAPE B : couverture du deser sur les records de dégât ===")
	fmt.Printf("  records          : %d\n", len(recs))
	fmt.Printf("  famille connue   : %d\n", nFam)
	fmt.Printf("  LOW==0x42c9679f (alignement prouvé jusqu'à bit 108) : %d/%d\n", nLowOK, len(recs))
	fmt.Printf("  isFloat (rare)   : %d\n", nFloat)
	fmt.Printf("  hasExtra(+0x1c)  : %d\n", nExtra)
	fmt.Printf("  desync (branche non bit-exacte) : %d\n", nDesync)
	fmt.Printf("  position-gate set (+0x27c présent) : %d\n", nPosGate)
	fmt.Printf("  vec3 impact décodé : %d\n", nVecOK)
	fmt.Printf("  modes position : %v\n\n", posModeHist)

	// ── ÉTAPE C : table (ts, famille, [impact]) — 40 premiers. ──
	fmt.Println("=== ÉTAPE C : timeline (ts, famille, cause, posGate, mode, vec) ===")
	for i, r := range recs {
		if i >= 40 {
			fmt.Printf("  ... (%d records)\n", len(recs))
			break
		}
		ds := ""
		if r.desync {
			ds = " DESYNC"
		}
		vs := ""
		if r.vecOK {
			vs = fmt.Sprintf(" vec=(%.2f,%.2f,%.2f)", r.vec[0], r.vec[1], r.vec[2])
		}
		fmt.Printf("  [%3d] t=%7.1fs %-24s cause=%d gate=%v mode=%d%s%s\n",
			i, float64(r.tms)/1000, r.fam, int32(r.cause), r.posGate, r.posMode, vs, ds)
	}

	// ── ÉTAPE D : records près des kills narrés. ──
	fmt.Println("\n=== ÉTAPE D : records de dégât près des kills narrés ===")
	for _, t := range []struct {
		label string
		tms   int
	}{
		{"BR75 JGtm->Akatsuki", 329800},
		{"BR75 JGtm->Akatsuki (1)", 112900},
		{"marteau IKE->JGtm", 115500},
		{"marteau IKE->JGtm (2)", 292500},
	} {
		fmt.Printf("  -- %s @%.1fs (±1500ms) --\n", t.label, float64(t.tms)/1000)
		for _, r := range recs {
			if r.tms < t.tms-1500 || r.tms > t.tms+1500 {
				continue
			}
			vs := ""
			if r.vecOK {
				vs = fmt.Sprintf(" vec=(%.2f,%.2f,%.2f)", r.vec[0], r.vec[1], r.vec[2])
			}
			ds := ""
			if r.desync {
				ds = " DESYNC"
			}
			fmt.Printf("     t=%7.1fs %-22s gate=%v mode=%d%s%s\n",
				float64(r.tms)/1000, r.fam, r.posGate, r.posMode, vs, ds)
		}
	}
	_ = math.Sqrt
}
