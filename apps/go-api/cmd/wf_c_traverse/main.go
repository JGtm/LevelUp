// wf_c_traverse — ANGLE C : traversée ECS ancrée + décodage de l'obje (i9) par record.
//
// Objectif : valider une traversée bit-exacte d'un record biped (archétype #35) en
// utilisant l'ANCRE arme connue de chunk_02 (R0=whiteknight, Hydra @bit195323). On
// câble le composant i0 object-position-dynamic-precision (widths 1, 6/6/6 mesurées
// via Cheat Engine) et on brute-force (start, default-state) jusqu'à ce qu'un slot
// weapon-state-type-info atterrisse EXACTEMENT au bit ancre. Depuis la traversée
// validée on extrait l'obje variant (i9) = id spartan du joueur.
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

// chunk_02 anchors (premier littéral high-32 d'arme de chaque record) + arme attendue.
var chunk02Anchors = []struct {
	Bit  int
	Pair string
}{
	{195323, "R0 whiteknight {Hydra,Hammer}"},
	{198140, "R1 Javier {ShockRifle,Hammer}"},
	{200933, "R2 {M41,Heatwave}"},
	{203736, "R3 {Mangler,Ravager}"},
	{206529, "R4 {MA40,Disruptor}"},
	{209339, "R5 {Mangler,Skewer}"},
	{212127, "R6 {Cindershot,ShockRifle}"},
	{214922, "R7 VitaminA {Bulldog,M41}"},
}

func inflate(path string) []byte {
	raw, _ := os.ReadFile(path)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

func extractType2(data []byte) []byte {
	off := 0
	for off+16 <= len(data) {
		typ := binary.LittleEndian.Uint16(data[off:])
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		if size <= 0 || off+16+size > len(data) {
			break
		}
		if typ == 2 {
			return data[off+16 : off+16+size]
		}
		off += 16 + size
	}
	return nil
}

func knownWeapon(v uint32) (string, bool) {
	for id, n := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return n, true
		}
	}
	return "", false
}

func main() {
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	payload := extractType2(inflate(cache + "/chunk_02.bin"))
	fmt.Printf("registre : %d archétypes ; keyframe type-2 : %d octets\n", len(reg.Archetypes), len(payload))

	// --- BIPED #35 : liste ordonnée + indices weapon-state-type-info ---
	biped, _ := reg.Archetype(35)
	fmt.Printf("\n=== archétype #35 (BIPED) : %d composants ===\n", len(biped.Components))
	var wstIdx []int
	for i, c := range biped.Components {
		mark := ""
		if c == "weapon-state-type-info" {
			wstIdx = append(wstIdx, i)
			mark = "  <== HELD WEAPON"
		}
		if c == "object-multiplayer-properties-component" {
			mark = "  <== OBJE (i9)"
		}
		if c == "object-position-dynamic-precision-component" {
			mark = "  <== POSITION (i0)"
		}
		if i < 12 || mark != "" {
			fmt.Printf("  i%-2d %s%s\n", i, c, mark)
		}
	}
	fmt.Printf("weapon-state-type-info aux index : %v\n", wstIdx)
	dumpBipedTail(reg)
	dumpWeaponsAroundR0(payload)
	validateLocalWST(payload)
	validateAllRecordsLocal(payload)
	wstStartScan(payload)
	findObjeBeforeAnchors(payload)
	tryDecodeObjeAtFixedOffset(payload)
	reachWeaponSlot(reg, payload)

	// =========================================================================
	// ÉTAPE 2 : ancrage bit-exact sur R0 (Hydra @195323).
	// Pour chaque (start, default-state) candidat : traverse, et garde le cas
	// où un slot weapon-state-type-info commence EXACTEMENT au bit ancre.
	// =========================================================================
	const anchorR0 = 195323
	fmt.Printf("\n=== ÉTAPE 2 : ancrage bit-exact R0 (arme @bit%d) ===\n", anchorR0)
	fmt.Printf("widths i0 par défaut : IndexW=%d AxisW=%v\n", filmdec.TraversalPrecision.IndexW, filmdec.TraversalPrecision.AxisW)

	hits := anchorScan(reg, payload, anchorR0, anchorR0-3000, anchorR0-1500)
	if len(hits) == 0 {
		fmt.Println("  AUCUN hit avec widths 6/6/6. -> balayage du total de bits i0.")
		sweepI0(reg, payload, anchorR0, anchorR0-3000, anchorR0-1500)
		return
	}
	reportHits(reg, payload, hits, "6/6/6")

	// =========================================================================
	// ÉTAPE 3 : si on a un start validé, traverse les 8 records, extrait obje i9.
	// =========================================================================
	best := hits[0]
	fmt.Printf("\n=== ÉTAPE 3 : traversée des 8 records depuis le start validé ===\n")
	traverseAllRecords(reg, payload, best.Start, best.Default)
}

type hit struct {
	Start      int
	Default    int
	WSTBit     int // bit de départ du slot weapon-state-type-info qui matche l'ancre
	ObjeVar    uint32
	HeldWeapon uint32
	Mask       uint64
}

// anchorScan brute-force (start ∈ [lo,hi), default ∈ pairs 50..140) ; garde les cas
// où un weapon-state-type-info démarre exactement au bit ancre.
func anchorScan(reg *filmdec.Registry, payload []byte, anchor, lo, hi int) []hit {
	if lo < 0 {
		lo = 0
	}
	var out []hit
	for start := lo; start < hi; start++ {
		// pré-filtre : R6 == 35 au start
		b0 := filmdec.NewBitReader(payload)
		b0.Skip(start)
		if uint32(b0.ReadBits(6)) != 35 {
			continue
		}
		for d := 50; d <= 140; d++ {
			b := filmdec.NewBitReader(payload)
			b.Skip(start)
			t := filmdec.TraverseEntity(b, reg, d)
			if t.TypeIndex != 35 {
				continue
			}
			for _, c := range t.Comps {
				if c.Name == "weapon-state-type-info" && c.StartBit == anchor {
					var obje uint32 = 0xFFFFFFFF
					for _, cc := range t.Comps {
						if cc.Name == "object-multiplayer-properties-component" {
							obje = cc.Variant
						}
					}
					out = append(out, hit{start, d, c.StartBit, obje, t.HeldWeapon, t.Mask})
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Default != out[j].Default {
			return out[i].Default < out[j].Default
		}
		return out[i].Start < out[j].Start
	})
	return out
}

func reportHits(reg *filmdec.Registry, payload []byte, hits []hit, label string) {
	fmt.Printf("  %d hit(s) bit-exact (arme @ancre) avec widths %s :\n", len(hits), label)
	shown := 0
	for _, h := range hits {
		if shown >= 16 {
			fmt.Printf("  ... (%d hits supplémentaires)\n", len(hits)-shown)
			break
		}
		wn := "?"
		if n, ok := knownWeapon(h.HeldWeapon); ok {
			wn = n
		}
		on := "?"
		if n, ok := knownWeapon(h.ObjeVar); ok {
			on = "(arme?)" + n
		}
		fmt.Printf("    start=%d default=%d  objeVar=0x%08x %s  held=0x%08x %s  mask=0x%016x\n",
			h.Start, h.Default, h.ObjeVar, on, h.HeldWeapon, wn, h.Mask)
		shown++
	}
}

// sweepI0 : si 6/6/6 ne donne pas l'alignement, balaye le COMPTE TOTAL de bits i0
// (0..40), puisque l'ancre arme exacte valide chaque hypothèse. On encode le total
// dans une PrecisionDescriptor à un seul axe (IndexW=0, AxisW=[total,0,0]) — le deser
// i0 absolu lit alors `total` bits sur la branche (bUsePred=0,bDelta=0,gate=1).
func sweepI0(reg *filmdec.Registry, payload []byte, anchor, lo, hi int) {
	fmt.Println("\n=== BALAYAGE total-bits-i0 (0..60) ===")
	orig := filmdec.TraversalPrecision
	type res struct {
		total, count int
	}
	var found []res
	for total := 0; total <= 60; total++ {
		// répartit total sur 3 axes (le deser additionne AxisW[0..2] sur la branche absolue)
		a := total / 3
		r := total - 3*a
		ax := [3]uint{uint(a), uint(a), uint(a)}
		ax[0] += uint(r)
		filmdec.TraversalPrecision = filmdec.PrecisionDescriptor{IndexW: 0, AxisW: ax}
		h := anchorScan(reg, payload, anchor, lo, hi)
		if len(h) > 0 {
			found = append(found, res{total, len(h)})
			if len(found) <= 6 {
				reportHits(reg, payload, h, fmt.Sprintf("total=%d (axes %v)", total, ax))
			}
		}
	}
	filmdec.TraversalPrecision = orig
	fmt.Printf("\n  totaux i0 qui alignent l'ancre : ")
	for _, f := range found {
		fmt.Printf("%d(×%d) ", f.total, f.count)
	}
	fmt.Println()
	if len(found) == 0 {
		fmt.Println("  AUCUN total i0 dans 0..60 n'aligne l'ancre via la branche absolue.")
		fmt.Println("  -> i0 emprunte probablement une autre branche (prédite/keep) ou un record")
		fmt.Println("     n'est PAS un biped #35 plein. Voir diagnostic ci-dessous.")
		diagnoseStart(reg, payload, anchor, lo, hi)
	}
}

// diagnoseStart : pour la fenêtre, trouve les starts R6==35 et montre où s'arrête la
// traversée (desyncAt + dernier composant atteint) pour comprendre le blocage.
func diagnoseStart(reg *filmdec.Registry, payload []byte, anchor, lo, hi int) {
	fmt.Printf("\n=== DIAGNOSTIC : starts R6==35 dans [%d,%d) ===\n", lo, hi)
	shown := 0
	for start := lo; start < hi && shown < 20; start++ {
		b0 := filmdec.NewBitReader(payload)
		b0.Skip(start)
		if uint32(b0.ReadBits(6)) != 35 {
			continue
		}
		// teste un default-state médian pour voir la forme
		for _, d := range []int{72, 76, 96, 110} {
			b := filmdec.NewBitReader(payload)
			b.Skip(start)
			t := filmdec.TraverseEntity(b, reg, d)
			if t.TypeIndex != 35 {
				continue
			}
			maxIdx := -1
			lastName := ""
			for _, c := range t.Comps {
				if c.Index > maxIdx {
					maxIdx = c.Index
					lastName = c.Name
				}
			}
			fmt.Printf("  start=%d d=%d mask=0x%016x pop=%d desyncAt=%d lastComp=i%d(%s)\n",
				start, d, t.Mask, popcount(t.Mask), t.DesyncAt, maxIdx, lastName)
			shown++
			if shown >= 20 {
				break
			}
		}
	}
}

// traverseAllRecords : depuis le start R0 validé et le default-state, suit la chaîne
// de records (chacun ~2800 bits) et extrait obje i9 + held weapon de chaque record.
func traverseAllRecords(reg *filmdec.Registry, payload []byte, start, def int) {
	pos := start
	for r := 0; r < 8; r++ {
		b := filmdec.NewBitReader(payload)
		b.Skip(pos)
		t := filmdec.TraverseEntity(b, reg, def)
		obje := uint32(0xFFFFFFFF)
		var weapons []string
		var weaponVars []uint32
		for _, c := range t.Comps {
			if c.Name == "object-multiplayer-properties-component" {
				obje = c.Variant
			}
			if c.Name == "weapon-state-type-info" && c.Variant != 0xFFFFFFFF {
				weaponVars = append(weaponVars, c.Variant)
				if n, ok := knownWeapon(c.Variant); ok {
					weapons = append(weapons, n)
				} else {
					weapons = append(weapons, fmt.Sprintf("0x%08x?", c.Variant))
				}
			}
		}
		fmt.Printf("  R%d start=%d typeIdx=%d objeVar=0x%08x desyncAt=%d endBit=%d armes=%v\n",
			r, pos, t.TypeIndex, obje, t.DesyncAt, t.EndBit, weapons)
		_ = weaponVars
		// avance au record suivant : si la traversée a atteint la fin proprement,
		// utilise endBit ; sinon saute ~2800 bits (saut nominal entre records).
		if t.DesyncAt == -1 && t.EndBit > pos {
			pos = t.EndBit
		} else {
			pos += 2800
		}
	}
}

func popcount(v uint64) int {
	c := 0
	for ; v != 0; v &= v - 1 {
		c++
	}
	return c
}
