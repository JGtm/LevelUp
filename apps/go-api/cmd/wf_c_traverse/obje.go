package main

// obje : cherche l'identifiant spartan (obje VariantName) par record. Hypothèse :
// comme l'arme est un littéral brut à offset fixe dans le record, l'obje variant
// l'est peut-être aussi. On cherche, dans la fenêtre AVANT chaque ancre d'arme, un
// u32 stable/haute-entropie qui se répète à offset constant entre records.

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmdec"
)

// scanU32 collecte tous les offsets où un u32 "plausible" (haute entropie) apparaît.
func plausibleU32(v uint32) bool {
	if v == 0 || v == 0xFFFFFFFF {
		return false
	}
	pc := 0
	for x := v; x != 0; x &= x - 1 {
		pc++
	}
	return pc >= 8 && pc <= 24
}

// findObjeBeforeAnchors : pour chaque record, liste les u32 plausibles dans
// [anchor-spacing, anchor) et tente de trouver un offset relatif constant.
func findObjeBeforeAnchors(payload []byte) {
	fmt.Println("\n=== RECHERCHE obje (id spartan) par record ===")
	// le record commence avant l'ancre d'arme ; spacing inter-record ≈ 2793..2817.
	// On regarde la zone [anchor-300, anchor-33) (entre les composants amont et l'arme).
	const back = 700
	for ai, a := range chunk02Anchors {
		fmt.Printf("\n  %s (ancre arme=%d)\n", a.Pair, a.Bit)
		lo := a.Bit - back
		if lo < 0 {
			lo = 0
		}
		shown := 0
		for bit := lo; bit < a.Bit-33 && shown < 6; bit++ {
			v := readU32At(payload, bit)
			if plausibleU32(v) {
				// relatif à l'ancre
				fmt.Printf("    @%d (ancre-%d) = 0x%08x\n", bit, a.Bit-bit, v)
				shown++
			}
		}
		_ = ai
	}
}

// tryDecodeObjeAtFixedOffset : si l'obje est décodé par DecodeEntityRecordQ à un
// offset fixe avant l'ancre, on balaie l'offset et on garde celui où le variant est
// stable et plausible à travers les 8 records.
func tryDecodeObjeAtFixedOffset(payload []byte) {
	fmt.Println("\n=== DÉCODAGE obje via DecodeEntityRecordQ à offset fixe avant l'ancre ===")
	best := -1
	bestScore := -1
	for off := 40; off <= 2800; off += 1 {
		score := 0
		valid := 0
		for _, a := range chunk02Anchors {
			start := a.Bit - off
			if start < 0 {
				break
			}
			br := filmdec.NewBitReader(payload)
			br.Skip(start)
			rec := filmdec.DecodeEntityRecordQ(br, 13)
			if rec.Valid && plausibleU32(rec.VariantName) {
				score++
			}
			valid++
		}
		if valid == len(chunk02Anchors) && score > bestScore {
			bestScore = score
			best = off
		}
	}
	fmt.Printf("  meilleur offset=%d : %d/%d records donnent un obje variant plausible\n",
		best, bestScore, len(chunk02Anchors))
	if best >= 0 {
		for _, a := range chunk02Anchors {
			start := a.Bit - best
			if start < 0 {
				continue
			}
			br := filmdec.NewBitReader(payload)
			br.Skip(start)
			rec := filmdec.DecodeEntityRecordQ(br, 13)
			fmt.Printf("    %-32s start=%d objeVar=0x%08x valid=%v\n", a.Pair, start, rec.VariantName, rec.Valid)
		}
	}
}
