// tmp_dmgdecode — D1 : PREUVE end-to-end du décodage du record de dégât (FUN_14080c1f8)
// à partir du MESSAGE CAPTURÉ LIVE (debugger, film 000d5950, 214 octets).
//
// Donnée de référence (capture 2026-06-07, bp FUN_14080c1f8) :
//   - message 214 o ci-dessous, @0x237D0340000 ; entête (8 premiers octets) déjà
//     consommée par le lecteur générique FUN_14080AADE.
//   - état bitreader à l'entrée du deser : byteptr=start+8, bitpos=24, count=24,
//     registre=0x0004384BD2000000 (64-bit MSB-first).
//
// FIELD-MAP du deser (offsets du record SORTIE param_3, lus du bitstream dans cet ordre) :
//
//	+0x08 slot/cause        FUN_1406d00ec  = R(1); si bit==0 R(2)         (consumeId2)
//	+0x0c global-id         FUN_1407f2034  = R(5) + R(32) big-endian
//	+0x10 handle optionnel  FUN_14080d69c  = R(1) gate ; si set R(32)     (consumeOpt32)
//	+0x14 variant_name      FUN_14080dec4  = R(32) BIG-ENDIAN = FAMILLE (clé high-32)
//	+0x18 source 'weap'     calculée (PAS lue du flux)
//
// VÉRITÉ-TERRAIN attendue : le message contient l'id64 Disruptor 0x84bd29ed42c9679f
// (suffixe variant 0x42c9679f commun à toutes les armes ; high-32 0x84bd29ed = famille).
// On prouve qu'en rejouant la séquence du deser on tombe EXACTEMENT sur variant_name=0x84bd29ed.
//
// Usage : tmp_dmgdecode
package main

import (
	"encoding/hex"
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

// message capturé : 214 octets (entête 8 o incluse).
const msgHex = "d260440004384bd29ed42c9679f76036028265840c0023483e495a66340f6840c088012000aac828e93e89d2c2ed901d7a22776e0220146002c81cff12d99680097922dc0440690005564138c97c4d161062c0ebc553e27011022500155984108292acc38022988d784841270006c40a900055640dfa9d1cc3801e0ccbc8114267011032400155903f463313f2808ec139600af8dc0440e900055641374924500011d0310ece43d2701283a24a36111fffffffff4f7c0100"

// catalogue : high-32 (famille) -> nom, id64 complet -> nom.
var h32name = map[uint32]string{}
var id64name = map[uint64]string{}

func buildSets() {
	for id, n := range analysis.WeaponIDToName {
		h32name[uint32(id>>32)] = n
		id64name[id] = n
	}
}

// bitsAt lit n bits MSB-first à partir du bit bp (modèle filmdec).
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

// --- helpers du deser, mirroir bit-exact des fonctions natives ---

// consumeId2 = FUN_1406d00ec : R(1); si bit==0 R(2). Retourne les bits consommés.
func consumeId2(br *filmdec.BitReader) int {
	start := br.BitPos()
	if !br.ReadBit() {
		br.ReadBits(2)
	}
	return br.BitPos() - start
}

// readGlobalID = FUN_1407f2034 : R(5) + R(32) big-endian. Le R(32) est le global-id.
func readGlobalID(br *filmdec.BitReader) (r5 uint64, gid uint32) {
	r5 = br.ReadBits(5)
	gid = uint32(br.ReadBits(32))
	return
}

// readOptHandle = FUN_14080d69c : R(1) gate ; si set R(32) (sentinelle 0xffffffff sinon).
func readOptHandle(br *filmdec.BitReader) (present bool, handle uint32) {
	if br.ReadBit() {
		return true, uint32(br.ReadBits(32))
	}
	return false, 0xffffffff
}

// readVariantName = FUN_14080dec4 : R(32) big-endian = FAMILLE (clé high-32).
func readVariantName(br *filmdec.BitReader) uint32 {
	return uint32(br.ReadBits(32))
}

func main() {
	buildSets()
	msg, err := hex.DecodeString(msgHex)
	if err != nil {
		panic(err)
	}
	fmt.Printf("=== message capturé : %d octets ; catalogue : %d familles high-32 ===\n", len(msg), len(h32name))

	// Vérité-terrain : où l'id64 Disruptor 0x84bd29ed42c9679f apparaît-il (bit-aligné) ?
	const truthID64 = uint64(0x84bd29ed42c9679f)
	const truthHigh = uint32(0x84bd29ed)
	truthBit := -1
	for bp := 0; bp+64 <= len(msg)*8; bp++ {
		if bitsAt(msg, bp, 64) == truthID64 {
			truthBit = bp
			break
		}
	}
	fmt.Printf("\n[vérité-terrain] id64 Disruptor 0x%016x trouvé bit-aligné @bit=%d (high-32=0x%08x = %s)\n",
		truthID64, truthBit, truthHigh, id64name[truthID64])

	// balaie aussi tous les high-32 connus pour cartographier le message.
	fmt.Println("\n[carte] high-32 famille bit-alignés dans le message :")
	for bp := 0; bp+32 <= len(msg)*8; bp++ {
		h := uint32(bitsAt(msg, bp, 32))
		if n, ok := h32name[h]; ok {
			low := uint32(bitsAt(msg, bp+32, 32))
			full := ""
			if fn, ok2 := id64name[(uint64(h)<<32)|uint64(low)]; ok2 {
				full = " [id64 COMPLET=" + fn + "]"
			}
			fmt.Printf("    @bit=%-4d high32=0x%08x %s%s\n", bp, h, n, full)
		}
	}

	// ── Rejoue la séquence du deser depuis chaque offset de départ candidat. ──
	// Le variant_name (R32 BE) doit être lu EXACTEMENT au truthBit (où l'id64 Disruptor
	// est bit-aligné). On cherche le startBit tel que slot+globalID+handle consomment
	// précisément (truthBit - startBit) bits, en respectant les branches optionnelles
	// réelles (consumeId2 : R1 puis opt R2 ; handle : gate puis opt R32).
	fmt.Println("\n[rejeu deser] startBit -> séquence canonique -> position du variant_name :")
	type hit struct {
		startBit int
		variant  uint32
		fam      string
		vAt      int // bit où le variant est lu
	}
	var hits []hit
	maxStart := len(msg)*8 - 64
	for startBit := 0; startBit <= maxStart; startBit++ {
		br := filmdec.NewBitReader(msg)
		br.Skip(startBit)
		consumeId2(br)    // +0x08 slot/cause
		readGlobalID(br)  // +0x0c global-id (R5+R32)
		readOptHandle(br) // +0x10 handle (1 bit + opt R32)
		vAt := br.BitPos()
		variant := readVariantName(br) // +0x14 variant_name = FAMILLE
		if n, ok := h32name[variant]; ok {
			hits = append(hits, hit{startBit, variant, n, vAt})
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].startBit < hits[j].startBit })
	if len(hits) == 0 {
		fmt.Println("    (aucun startBit ne fait lire une famille connue avec la séquence canonique)")
	}
	for _, h := range hits {
		mark := ""
		if h.variant == truthHigh {
			mark = "  <<< MATCH VÉRITÉ-TERRAIN (Disruptor) — variant lu au truthBit"
		}
		fmt.Printf("    startBit=%-4d -> variant_name @bit%-4d = 0x%08x %-12s%s\n",
			h.startBit, h.vAt, h.variant, h.fam, mark)
	}

	// ── Trace détaillée de l'alignement qui matche la vérité-terrain. ──
	traced := false
	for _, h := range hits {
		if h.variant != truthHigh {
			continue
		}
		traced = true
		fmt.Printf("\n=== TRACE de l'alignement gagnant (startBit=%d, variant @bit%d) ===\n", h.startBit, h.vAt)
		br := filmdec.NewBitReader(msg)
		br.Skip(h.startBit)
		p0 := br.BitPos()
		slotBits := consumeId2(br)
		fmt.Printf("  +0x08 slot/cause   @bit%-4d consumeId2 (%d bits consommés)\n", p0, slotBits)
		p1 := br.BitPos()
		r5, gid := readGlobalID(br)
		fmt.Printf("  +0x0c global-id    @bit%-4d R(5)=0x%x R(32)=0x%08x (37 bits)\n", p1, r5, gid)
		p2 := br.BitPos()
		present, handle := readOptHandle(br)
		fmt.Printf("  +0x10 handle       @bit%-4d gate=%v handle=0x%08x\n", p2, present, handle)
		p3 := br.BitPos()
		variant := readVariantName(br)
		fmt.Printf("  +0x14 variant_name @bit%-4d R(32) BE = 0x%08x = %s\n", p3, variant, h32name[variant])
		low := uint32(br.ReadBits(32))
		fmt.Printf("  (low-32 suivant)   @bit%-4d R(32) BE = 0x%08x %s\n", p3+32, low,
			ternary(low == 0x42c9679f, "= suffixe variant attendu 0x42c9679f", "!= 0x42c9679f (info)"))
		full := (uint64(variant) << 32) | uint64(low)
		fmt.Printf("  => id64 reconstruit = 0x%016x = %s\n", full, id64name[full])
		break
	}
	if !traced {
		// La séquence canonique n'atteint pas le truthBit : reconstruis-la à l'envers
		// depuis truthBit pour exhiber le préfixe exact (slot/globalID/handle) qui colle.
		fmt.Println("\n[rétro-séquence] reconstruction à rebours depuis le truthBit du variant :")
		reverseFromTruth(msg, truthBit)
	}

	// ── TRACE FORCÉE à l'état capturé exact : position logique = byteptr*8 - count. ──
	// Capture : byteptr=start+8 (octet 8), count=24 -> bit logique de départ = 64-24 = 40.
	const startLogical = 64 - 24
	fmt.Printf("\n=== TRACE FORCÉE depuis l'état capturé (bit logique = byteptr*8 - count = %d) ===\n", startLogical)
	traceSeq(msg, startLogical)

	fmt.Println("\n=== TRACE de l'alignement GAGNANT (startBit=36 : global-id porte la famille) ===")
	traceSeq(msg, 36)

	// Confirmation : le high-32 famille (global-id) est immédiatement suivi du low-32
	// suffixe variant 0x42c9679f -> id64 d'arme complet contigu dans le flux.
	fmt.Println("\n=== CONFIRMATION id64 contigu (high-32 famille + low-32 suffixe) ===")
	br := filmdec.NewBitReader(msg)
	br.Skip(36)
	consumeId2(br)                  // slot
	br.ReadBits(5)                  // R(5) du global-id
	high := uint32(br.ReadBits(32)) // high-32 = famille
	famBit := 39 + 5
	low := uint32(br.ReadBits(32)) // R(32) immédiatement suivant
	full := (uint64(high)<<32 | uint64(low))
	fmt.Printf("  famille high-32 @bit%-3d = 0x%08x = %s\n", famBit, high, h32name[high])
	fmt.Printf("  suffixe low-32  @bit%-3d = 0x%08x %s\n", famBit+32, low,
		ternary(low == 0x42c9679f, "= 0x42c9679f (suffixe variant attendu) ✓", "≠ 0x42c9679f"))
	fmt.Printf("  => id64 contigu = 0x%016x = %q\n", full, id64name[full])
	fmt.Printf("\n  [cohérence état capturé] startBit logique gagnant=36 ; byteptr*8-count=%d (Δ=%d bits)\n",
		startLogical, startLogical-36)

	// On trace aussi depuis le truthBit du variant à rebours sur la séquence canonique,
	// pour exhiber le bit logique de départ RÉEL impliqué par la grammaire.
	fmt.Println("\n=== Position logique de départ impliquée par 'variant @truthBit' ===")
	for _, prefix := range []int{39, 41, 71, 73} {
		fmt.Printf("    si préfixe slot+global+handle = %d bits -> départ @bit %d\n", prefix, truthBit-prefix)
	}

	// ── BALAYAGE FIN autour de l'état capturé : pour chaque startBit, on regarde si
	// l'UN des 4 champs (slot/global/handle/variant) lit une famille connue (high-32). ──
	fmt.Println("\n=== BALAYAGE FIN (startBit 0..96) : quel CHAMP capture une famille connue ? ===")
	for startBit := 0; startBit <= 96; startBit++ {
		br := filmdec.NewBitReader(msg)
		br.Skip(startBit)
		sAt := br.BitPos()
		consumeId2(br)
		gAt := br.BitPos()
		_, gid := readGlobalID(br)
		hAt := br.BitPos()
		_, handle := readOptHandle(br)
		vAt := br.BitPos()
		variant := readVariantName(br)
		report := func(label string, at int, val uint32) {
			if n, ok := h32name[val]; ok {
				m := ""
				if val == truthHigh {
					m = "  <<< Disruptor (vérité-terrain)"
				}
				fmt.Printf("    startBit=%-3d  champ=%-10s @bit%-3d = 0x%08x = %s%s\n", startBit, label, at, val, n, m)
			}
		}
		report("global-id", gAt, gid)
		report("handle", hAt, handle)
		report("variant", vAt, variant)
		_ = sAt
	}
}

// traceSeq déroule et trace la séquence canonique du deser depuis startBit.
func traceSeq(msg []byte, startBit int) {
	if startBit < 0 || startBit >= len(msg)*8 {
		fmt.Printf("    (startBit %d hors buffer)\n", startBit)
		return
	}
	br := filmdec.NewBitReader(msg)
	br.Skip(startBit)
	p0 := br.BitPos()
	slotBits := consumeId2(br)
	fmt.Printf("  +0x08 slot/cause   @bit%-4d consumeId2 (%d bits)\n", p0, slotBits)
	p1 := br.BitPos()
	r5, gid := readGlobalID(br)
	fmt.Printf("  +0x0c global-id    @bit%-4d R(5)=0x%x R(32)=0x%08x\n", p1, r5, gid)
	p2 := br.BitPos()
	present, handle := readOptHandle(br)
	fmt.Printf("  +0x10 handle       @bit%-4d gate=%v handle=0x%08x\n", p2, present, handle)
	p3 := br.BitPos()
	variant := readVariantName(br)
	fam, known := h32name[variant]
	low := uint32(br.ReadBits(32))
	full := (uint64(variant) << 32) | uint64(low)
	fmt.Printf("  +0x14 variant_name @bit%-4d R(32) BE = 0x%08x = %s (connu=%v)\n", p3, variant, fam, known)
	fmt.Printf("  (low-32 suivant)   @bit%-4d R(32) BE = 0x%08x\n", p3+32, low)
	fmt.Printf("  => id64 reconstruit = 0x%016x = %s\n", full, id64name[full])
}

// reverseFromTruth : le variant_name (R32 BE) est lu @truthBit. On reconstitue le
// préfixe exact (slot consumeId2, global-id R5+R32, handle gate+optR32) qui se termine
// précisément à truthBit, en énumérant les branches optionnelles réelles. On affiche
// le startBit cohérent et le détail des champs.
func reverseFromTruth(msg []byte, truthBit int) {
	// Coûts possibles de chaque champ selon ses branches :
	//   slot consumeId2 : 1 (bit==1) ou 3 (bit==0 -> +R2)
	//   global-id       : 37 (R5+R32, fixe)
	//   handle          : 1 (gate==0) ou 33 (gate==1 -> +R32)
	// total prefix ∈ {39,41,71,73}. On teste chaque combinaison et on lit RÉELLEMENT
	// les branches depuis le startBit candidat pour valider la cohérence des bits de gate.
	slotCosts := []int{1, 3}
	handleCosts := []int{1, 33}
	for _, sc := range slotCosts {
		for _, hc := range handleCosts {
			prefix := sc + 37 + hc
			startBit := truthBit - prefix
			if startBit < 0 {
				continue
			}
			// rejoue depuis startBit et vérifie que les coûts réels == sc/hc ET que
			// le variant atterrit bien à truthBit.
			br := filmdec.NewBitReader(msg)
			br.Skip(startBit)
			realSlot := consumeId2(br)
			readGlobalID(br)
			hStart := br.BitPos()
			readOptHandle(br)
			realHandle := br.BitPos() - hStart
			vAt := br.BitPos()
			if realSlot == sc && realHandle == hc && vAt == truthBit {
				variant := bitsAt(msg, truthBit, 32)
				low := bitsAt(msg, truthBit+32, 32)
				full := (variant << 32) | low
				fmt.Printf("  COHÉRENT : startBit=%d | slot=%dbits global=37bits handle=%dbits | "+
					"variant @bit%d=0x%08x=%s | id64=0x%016x=%s\n",
					startBit, sc, hc, truthBit, uint32(variant), h32name[uint32(variant)], full, id64name[full])
			}
		}
	}
}

func ternary(c bool, a, b string) string {
	if c {
		return a
	}
	return b
}
