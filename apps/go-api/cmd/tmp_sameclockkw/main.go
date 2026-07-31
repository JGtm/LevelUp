// tmp_sameclockkw — arme par kill SAME-CLOCK, 100% offline.
//
// Déblocage 2026-07-04 (cf project_killfeed_sameclock_localized) : le kill-event et le
// dégât 0xd2 sont décodés dans le MÊME flux frame (confirmé CE, même buffer) → MÊME
// HORLOGE. Le frame kill-event = marqueur 0xE6 (98 frames ≈ 94 kills), tueur/victime à
// l'offset ~80 (grammaire §174 : R1+optR5, présence = R(1)==0 → R(5)). Le dégât 0xd2 porte
// attaquant (R5 bit 36, slot×2) + arme (famille + suffixe 0x42c9679f).
//
// Corrélation : pour chaque kill (tueur, victime, ts-frame), le dégât 0xd2 avec
// attaquant=tueur le plus proche en ts → l'arme. MÊME horloge = pas de résidu 2-horloges
// (vs warp 58% Team Slayer). But : mesurer le taux d'attribution.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_sameclockkw [filmID] [killOffset]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"levelup/go-api/internal/analysis"
	"os"
	"sort"
	"strconv"
)

const root = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
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

type killEv struct {
	killer, victim int
	ts             uint64
	s1, s2         uint32
}
type dmgEv struct {
	attacker int
	fam      string
	ts       uint64
	firearm  bool
	marker   byte
}

func main() {
	m := "000d5950"
	killOff := 80
	if len(os.Args) > 1 {
		m = os.Args[1]
	}
	if len(os.Args) > 2 {
		killOff, _ = strconv.Atoi(os.Args[2])
	}
	cache := root + "/" + m
	h32 := map[uint32]string{}
	for id := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = analysis.WeaponIDToName[id]
	}

	var kills []killEv
	var dmgs []dmgEv
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
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ != 0 || len(pl) == 0 {
				continue
			}
			switch pl[0] {
			case 0xE6: // kill-event
				bp := killOff
				readOpt := func() int {
					present := bitsAt(pl, bp, 1) == 0
					bp++
					if present {
						v := int(bitsAt(pl, bp, 5))
						bp += 5
						return v
					}
					return -1
				}
				k, v := readOpt(), readOpt()
				readOpt() // assist (avance bp)
				s1 := uint32(bitsAt(pl, bp, 32))
				s2 := uint32(bitsAt(pl, bp+32, 32))
				if k >= 0 && v >= 0 && k != v && k < 24 && v < 24 {
					kills = append(kills, killEv{k, v, ts, s1, s2})
				}
			case 0xd2: // UNIQUEMENT le DamageReport (les frères 0xC0/0xC2/... = apparence, ignorés)
				bp := 41
				if bitsAt(pl, bp, 1) == 1 {
					bp++
				} else {
					bp += 3
				}
				f := uint32(bitsAt(pl, bp, 32))
				suf := uint32(bitsAt(pl, bp+32, 32))
				atk := int(bitsAt(pl, 36, 5)) >> 1
				if nm, ok := h32[f]; ok && suf == sfx {
					dmgs = append(dmgs, dmgEv{atk, nm, ts, true, 0xd2}) // arme à feu (suffixe firearm + famille h32)
				} else {
					// 0xd2 non-firearm : suffixe de type ≠ firearm = mêlée/grenade/splatter/véhicule.
					dmgs = append(dmgs, dmgEv{atk, fmt.Sprintf("cause-%08X", suf), ts, false, 0xd2})
				}
			}
		}
	}
	sort.Slice(dmgs, func(i, j int) bool { return dmgs[i].ts < dmgs[j].ts })
	fmt.Printf("=== film %s : %d kills (0xE6@%d), %d dégâts 0xd2 armés ===\n", m, len(kills), killOff, len(dmgs))

	// ATTRIBUTION same-clock : l'arme = le DERNIER dégât ARME-À-FEU (0xd2) du tueur AVANT le kill
	// (même horloge que le kill-event → pas de résidu 2-horloges). Si aucun 0xd2 → mort
	// non-arme-à-feu (mêlée/grenade), couverte par le dernier marqueur frère du tueur avant le kill.
	// Le tout en ordre de flux (ts = ordre) ; le dt (ms) n'est qu'un contrôle, pas un seuil.
	nFirearm, nNonFirearm, nSuicide := 0, 0, 0
	famCount := map[string]int{}
	var dtsMs []float64
	// Le coup fatal = le 0xd2 du tueur (attaquant=tueur) LE PLUS PROCHE du kill (même horloge).
	// Firearm → arme nommée ; 0xd2 non-firearm → cause (mêlée/grenade/...). Aucun → suicide/chute.
	for i, k := range kills {
		var bestFam string
		var bestFirearm bool
		bestDT := uint64(1 << 63)
		var anyFam string
		var anyFirearm bool
		anyDT2 := uint64(1 << 63)
		for _, dg := range dmgs {
			dt := k.ts - dg.ts
			if dg.ts > k.ts {
				dt = dg.ts - k.ts
			}
			if dt < anyDT2 { // 0xd2 le plus proche du kill, TOUT attaquant (= coup fatal)
				anyDT2, anyFam, anyFirearm = dt, dg.fam, dg.firearm
			}
			if dg.attacker != k.killer {
				continue
			}
			if dt < bestDT {
				bestDT, bestFam, bestFirearm = dt, dg.fam, dg.firearm
			}
		}
		// FALLBACK matching : si aucun 0xd2 du tueur 0xE6 (souvent 0xE6 killer mal décodé, ~14%),
		// mais un 0xd2 est COLLÉ au kill (<50ms), c'est le coup fatal → son attaquant = le vrai tueur.
		if bestFam == "" && anyFam != "" && anyDT2 < 50_000_000 {
			bestFam, bestFirearm, bestDT = anyFam, anyFirearm, anyDT2
		}
		switch {
		case bestFam == "": // vraiment aucun 0xd2 proche = suicide / chute / environnement
			nSuicide++
		case bestFirearm: // arme à feu nommée
			nFirearm++
			famCount[bestFam]++
			dtsMs = append(dtsMs, float64(bestDT)/1e6)
			if i < 12 {
				fmt.Printf("  kill tueur=%d victime=%d -> arme=%s (dt=%.1fms)\n", k.killer, k.victim, bestFam, float64(bestDT)/1e6)
			}
		default: // 0xd2 non-firearm = cause (mêlée/grenade/splatter/...)
			nNonFirearm++
			famCount[bestFam]++
			dtsMs = append(dtsMs, float64(bestDT)/1e6)
		}
	}
	sort.Float64s(dtsMs)
	med := 0.0
	if len(dtsMs) > 0 {
		med = dtsMs[len(dtsMs)/2]
	}
	fmt.Printf("\n=== %d kills : %d arme à feu nommée (0xd2, dt médian=%.1fms) + %d non-arme-à-feu + %d suicide/chute ===\n",
		len(kills), nFirearm, med, nNonFirearm, nSuicide)
	fmt.Printf("couverture (mort → cause) : %d/%d = %.0f%%\n", nFirearm+nNonFirearm, len(kills),
		float64(nFirearm+nNonFirearm)*100/float64(len(kills)))
	type fc struct {
		f string
		n int
	}
	var fcs []fc
	for f, n := range famCount {
		fcs = append(fcs, fc{f, n})
	}
	sort.Slice(fcs, func(i, j int) bool { return fcs[i].n > fcs[j].n })
	fmt.Print("distribution: ")
	for _, x := range fcs {
		fmt.Printf("%s=%d ", x.f, x.n)
	}
	fmt.Println()
}
