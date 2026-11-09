package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

// LA QUESTION QUE CE FICHIER TRANCHE. La note NOTE_I0_TI41 §2.2 declare les largeurs « non
// derivables statiquement » parce que le pas vient d'une globale de runtime. Si la table
// dumpee n'est QUE la forme fermee precalculee sur la plage par defaut, alors elle n'apporte
// aucune information neuve : la loi suffit, et le dump ne fait que la confirmer.
//
// Il y a une correction a porter au passage : `FUN_1406d310c` n'est pas `bitLen` mais
// `ceilLog2`. Les deux coincident partout SAUF sur les puissances exactes de deux — et c'est
// precisement ce cas qui decide des niveaux 17 a 22, ou la saturation a 2^22 mord.

// ceilLog2 rend le plus petit W tel que n <= 2^W.
func ceilLog2(n uint64) int {
	if n <= 1 {
		return 0
	}
	w := 0
	for (uint64(1) << uint(w)) < n {
		w++
	}
	return w
}

// stepFor est FUN_140be9c78 : step(L) = 2^(16-L) / 120.
func stepFor(level int) float64 {
	return math.Exp2(float64(16-level)) / 120
}

// widthClosedForm est FUN_140be9b88, corrige : saturation du compte a 2^22, puis ceilLog2,
// puis plafond 26. Sous le seuil de pas, largeur forcee a 26.
func widthClosedForm(extent float64, level int) int {
	const stepEpsilon = 1e-4 // DAT_143cd837c
	const countSaturation = 1 << 22
	s := stepFor(level)
	if s < stepEpsilon {
		return 0x1a
	}
	n := math.Ceil(extent / (2 * s))
	if n > countSaturation {
		n = countSaturation
	}
	return min(ceilLog2(uint64(n)), 0x1a)
}

// verifyWidths confronte les 32 premieres entrees de la table dumpee a la forme fermee
// appliquee a l'etendue 40000 (DAT_143b8c6b8, la plage par defaut +-20000).
func verifyWidths(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	const levels = 32
	const defaultExtent = 40000.0

	fmt.Println("=== LA TABLE DUMPEE EST-ELLE LA FORME FERMEE PRECALCULEE ? ===")
	fmt.Printf("hypothese : table[L] = min(26, ceilLog2(min(ceil(%.0f / (2*step(L))), 2^22))), step(L)=2^(16-L)/120\n\n",
		defaultExtent)
	fmt.Println("  L   table   forme fermee   pas          compte      verdict")

	accord, total := 0, 0
	for l := 0; l < levels; l++ {
		off := l * widthEntry
		if off+12 > len(raw) {
			break
		}
		got := binary.LittleEndian.Uint32(raw[off:])
		b := binary.LittleEndian.Uint32(raw[off+4:])
		c := binary.LittleEndian.Uint32(raw[off+8:])
		want := widthClosedForm(defaultExtent, l)
		s := stepFor(l)
		n := math.Ceil(defaultExtent / (2 * s))
		verdict := "ACCORD"
		if int(got) != want || got != b || got != c {
			verdict = "DESACCORD"
		} else {
			accord++
		}
		total++
		fmt.Printf("  %2d  %2d/%2d/%2d      %2d        %.6g   %10.0f   %s\n", l, got, b, c, want, s, n, verdict)
	}
	fmt.Printf("\n%d / %d niveaux en accord.\n", accord, total)
	if accord == total {
		fmt.Println("VERDICT : la table dumpee N'EST QUE la loi, precalculee. Elle ne porte aucune")
		fmt.Println("          information que le desassemblage ne donnait pas — elle la CONFIRME.")
	}
	fmt.Println()

	fmt.Println("=== APPLICATION AUX PLAGES QUI COMPTENT (niveau L = 16, le site d'appel d'i0) ===")
	type rng struct {
		name string
		ext  [3]float64
	}
	for _, r := range []rng{
		{"carte Cliffhanger (AABB du .module)", [3]float64{113.212307, 113.818703, 137.551193}},
		{"DAT_143b8c6d0 (+-100), la branche par defaut", [3]float64{200, 200, 200}},
		{"DAT_143b8c6b8 (+-20000)", [3]float64{40000, 40000, 40000}},
	} {
		var w [3]int
		sum := 0
		for a := 0; a < 3; a++ {
			w[a] = widthClosedForm(r.ext[a], 16)
			sum += w[a]
		}
		fmt.Printf("  %-44s -> %2d/%2d/%2d   total axes %2d   +porte+R(2) = %2d\n",
			r.name, w[0], w[1], w[2], sum, sum+3)
	}
	fmt.Println()
	return nil
}
