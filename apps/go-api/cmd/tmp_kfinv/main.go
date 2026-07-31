// cmd/tmp_kfinv — SONDE : inventaire lu dans les records de biped des images-clés.
//
// Objectif : refaire le bloc `inv` du POC sur le film 000d5950 (celui de la vérité terrain),
// sans capture Cheat Engine (qui ne couvre que 9e8fb31b).
//
// Cette passe est EXPLORATOIRE : elle mesure la sélectivité des ancres avant de produire
// quoi que ce soit. Sorties chiffrées, aucune conclusion sans son dénominateur.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/weaponv3"
)

const bipedTI = 35

// abilityAnchor : ancre 28 bits du record d'archétype 35 portant la capacité (RECETTE §2).
const abilityAnchor uint32 = 0x8CAC57A

// abilityPattern : motif 20 bits cherché dans les 60 bits qui suivent l'ancre.
const abilityPattern uint32 = 0x00012 // 00000000000000010010

type recSpan struct {
	slot, ti, gen int
	from, to      int // bornes bit dans le payload
}

func bitAt(buf []byte, p int) uint32 {
	if idx := p >> 3; idx >= 0 && idx < len(buf) {
		return uint32(buf[idx]>>(7-uint(p&7))) & 1
	}
	return 0
}

func bits(buf []byte, p, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		v = v<<1 | bitAt(buf, p+i)
	}
	return v
}

func spans(pay []byte) []recSpan {
	recs := filmdec.WalkKeyframeWorld(pay)
	if len(recs) == 0 {
		return nil
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
	out := make([]recSpan, 0, len(recs))
	total := len(pay) * 8
	for i, r := range recs {
		to := total
		if i+1 < len(recs) {
			to = recs[i+1].Bit
		}
		out = append(out, recSpan{slot: r.Slot, ti: r.TI, gen: r.Gen, from: r.Bit, to: to})
	}
	return out
}

// famsIn rend les familles d'arme connues trouvées dans [from,to), avec leur position bit.
func famsIn(pay []byte, from, to int, known map[uint32]string) (pos []int, fam []uint32) {
	var w uint32
	start := from
	if start < 31 {
		start = 31
	}
	for b := from; b < to; b++ {
		w = w<<1 | bitAt(pay, b)
		if b-from < 31 {
			continue
		}
		if _, ok := known[w]; ok {
			pos = append(pos, b-31)
			fam = append(fam, w)
		}
	}
	return
}

// abilityIn cherche l'ancre puis le motif ; rend les index lus sur 3 et 6 bits + la porte.
type abilHit struct {
	anchorBit int
	patEnd    int
	idx3      uint32
	idx6      uint32
	gate      uint32
}

func abilityIn(pay []byte, from, to int) []abilHit {
	var out []abilHit
	var w uint32
	const mask28 = (uint32(1) << 28) - 1
	for b := from; b < to; b++ {
		w = ((w << 1) | bitAt(pay, b)) & mask28
		if b-from < 27 {
			continue
		}
		if w != abilityAnchor {
			continue
		}
		anchorEnd := b + 1
		// motif 20 bits dans les 60 bits suivants
		for off := 0; off <= 60; off++ {
			p := anchorEnd + off
			if p+20 > to {
				break
			}
			if bits(pay, p, 20) != abilityPattern {
				continue
			}
			pe := p + 20
			out = append(out, abilHit{
				anchorBit: b - 27,
				patEnd:    pe,
				idx3:      bits(pay, pe, 3),
				idx6:      bits(pay, pe-3, 6),
				gate:      bitAt(pay, pe-4),
			})
			break
		}
	}
	return out
}

// grenIn cherche le motif d'i22 : R(3)=4 puis 4 x R(8) à petites valeurs.
type grenHit struct {
	bit int
	c   [4]uint32
}

func grenIn(pay []byte, from, to int, maxVal uint32) []grenHit {
	var out []grenHit
	for b := from; b+35 <= to; b++ {
		if bits(pay, b, 3) != 4 {
			continue
		}
		var c [4]uint32
		ok := true
		sum := uint32(0)
		for i := 0; i < 4; i++ {
			v := bits(pay, b+3+8*i, 8)
			if v > maxVal {
				ok = false
				break
			}
			c[i] = v
			sum += v
		}
		if !ok || sum == 0 {
			continue
		}
		out = append(out, grenHit{bit: b, c: c})
	}
	return out
}

func main() {
	dir := flag.String("dir", "", "répertoire des chunks du film")
	mode := flag.String("mode", "diag", "diag | dump")
	slotLo := flag.Int("lo", 512, "slot min")
	slotHi := flag.Int("hi", 519, "slot max")
	maxKF := flag.Int("kf", 0, "nombre max d'images-clés (0 = toutes)")
	grenMax := flag.Int("grenmax", 2, "valeur max d'un compteur de grenade")
	kfWanted := flag.Int("kfw", 1, "numéro (1-based) de l'image-clé pour les modes de chasse")
	originFlag := flag.Int64("origin", 0, "origine de l'horloge en us (0 = calculée)")
	film := flag.String("film", "000d5950", "identifiant du film (tracabilité)")
	out := flag.String("out", "", "chemin du JSON de sortie")
	flag.Parse()
	if *dir == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_kfinv -dir <chunks>")
		os.Exit(2)
	}
	known := map[uint32]string{}
	for f, n := range weaponv3.KnownWeaponHigh32 {
		known[f] = n
	}
	if *mode == "ammo" || *mode == "uses" || *mode == "pre" || *mode == "solve" {
		runHunt(*dir, *kfWanted, uint32(*grenMax), *mode)
		return
	}
	if *mode == "extract" {
		origin := *originFlag
		if origin == 0 {
			sopt := filmdec.DefaultScanFilmOptions()
			sopt.QuantaOnly = true
			pos, err := filmdec.ScanFilmBipedPositions(*dir, sopt)
			if err != nil || len(pos) == 0 {
				fmt.Fprintf(os.Stderr, "origine indisponible: %v\n", err)
				os.Exit(1)
			}
			origin = int64(pos[0].TimestampUS)
			for _, p := range pos {
				if int64(p.TimestampUS) < origin {
					origin = int64(p.TimestampUS)
				}
			}
			fmt.Printf("origine (1er paquet de position) = %d us\n", origin)
		}
		runExtract(*dir, *film, uint32(*grenMax), uint64(origin), 100000, *out)
		return
	}
	if *mode == "uses-fit" {
		runUsesFit(*dir, *kfWanted, uint32(*grenMax))
		return
	}

	n := filmdec.CountFilmChunks(*dir)
	kf := 0
	// statistiques globales
	statBiped, statAbil0, statAbil1, statAbilN := 0, 0, 0, 0
	statGren0, statGren1, statGrenN := 0, 0, 0
	abilVals := map[uint32]int{}
	idx6vals := map[uint32]int{}
	gateVals := map[uint32]int{}

	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(*dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			kf++
			if *maxKF > 0 && kf > *maxKF {
				break
			}
			pay := p.Payload(chunk)
			sp := spans(pay)
			for _, s := range sp {
				if s.ti != bipedTI {
					continue
				}
				statBiped++
				inRange := s.slot >= *slotLo && s.slot <= *slotHi
				ah := abilityIn(pay, s.from, s.to)
				switch len(ah) {
				case 0:
					statAbil0++
				case 1:
					statAbil1++
				default:
					statAbilN++
				}
				for _, h := range ah {
					abilVals[h.idx3]++
					idx6vals[h.idx6]++
					gateVals[h.gate]++
				}
				gh := grenIn(pay, s.from, s.to, uint32(*grenMax))
				switch len(gh) {
				case 0:
					statGren0++
				case 1:
					statGren1++
				default:
					statGrenN++
				}
				if *mode == "dump" && inRange {
					pos, fam := famsIn(pay, s.from, s.to, known)
					fmt.Printf("KF%02d chunk%02d t=%d slot=%d gen=%d bits=[%d,%d) len=%d\n",
						kf, c, p.TimestampUS, s.slot, s.gen, s.from, s.to, s.to-s.from)
					for i := range pos {
						fmt.Printf("   arme @+%-6d %08x %s\n", pos[i]-s.from, fam[i], known[fam[i]])
					}
					for _, h := range ah {
						fmt.Printf("   abil @+%-6d idx3=%d idx6=%d gate=%d\n", h.anchorBit-s.from, h.idx3, h.idx6, h.gate)
					}
					for _, h := range gh {
						fmt.Printf("   gren @+%-6d %v\n", h.bit-s.from, h.c)
					}
				}
			}
		}
		if *maxKF > 0 && kf > *maxKF {
			break
		}
	}
	fmt.Printf("\n=== BILAN : %d images-clés, %d records biped\n", kf, statBiped)
	fmt.Printf("capacité : 0 occurrence %d · 1 occurrence %d · >1 %d\n", statAbil0, statAbil1, statAbilN)
	fmt.Printf("grenades : 0 occurrence %d · 1 occurrence %d · >1 %d\n", statGren0, statGren1, statGrenN)
	fmt.Printf("idx3 : %v\n", sortedMap(abilVals))
	fmt.Printf("idx6 : %v\n", sortedMap(idx6vals))
	fmt.Printf("gate : %v\n", sortedMap(gateVals))
}

func sortedMap(m map[uint32]int) string {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	s := ""
	for _, k := range keys {
		s += fmt.Sprintf("%d:%d ", k, m[uint32(k)])
	}
	return s
}
