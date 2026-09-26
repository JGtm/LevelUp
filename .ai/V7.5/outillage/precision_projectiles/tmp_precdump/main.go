// Commande de recherche (jetable, gitignoree) : lecture des tables de precision
// dumpees de la memoire du jeu.
//
//	ce_prec_widths_1445cc9e0.bin  (DAT_1445cc9e0) : triples de largeurs par niveau
//	ce_prec_ranges_14462cbe0.bin  (DAT_14462cbe0) : AABB (float[6]) par index
//
// Objet : trancher les largeurs d'axe de la branche « plage par defaut » de
// object-position-component (ti=41), que la note NOTE_I0_TI41 declarait non
// derivables statiquement. Lecture, pas inference.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"os"
)

const (
	widthEntry = 12 // 3 x u32
	rangeEntry = 24 // 6 x float32 : minX,maxX,minY,maxY,minZ,maxZ
)

type triple struct{ a, b, c uint32 }

type aabb struct{ minX, maxX, minY, maxY, minZ, maxZ float32 }

func (r aabb) extents() (float64, float64, float64) {
	return float64(r.maxX - r.minX), float64(r.maxY - r.minY), float64(r.maxZ - r.minZ)
}

func (r aabb) String() string {
	return fmt.Sprintf("x[%.4f..%.4f] y[%.4f..%.4f] z[%.4f..%.4f]",
		r.minX, r.maxX, r.minY, r.maxY, r.minZ, r.maxZ)
}

func main() {
	widthsPath := flag.String("widths", "", "chemin de ce_prec_widths_*.bin")
	rangesPath := flag.String("ranges", "", "chemin de ce_prec_ranges_*.bin")
	derive := flag.Bool("derive", false, "derive les largeurs par la forme fermee")
	verify := flag.String("verify", "", "confronte la table de largeurs a la forme fermee")
	flag.Parse()

	if *verify != "" {
		if err := verifyWidths(*verify); err != nil {
			fmt.Fprintln(os.Stderr, "verify:", err)
			os.Exit(1)
		}
	}

	if *widthsPath != "" {
		if err := dumpWidths(*widthsPath); err != nil {
			fmt.Fprintln(os.Stderr, "widths:", err)
			os.Exit(1)
		}
	}
	if *rangesPath != "" {
		if err := dumpRanges(*rangesPath); err != nil {
			fmt.Fprintln(os.Stderr, "ranges:", err)
			os.Exit(1)
		}
	}
	if *derive {
		deriveWidths()
	}
}

func dumpWidths(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	n := len(raw) / widthEntry
	fmt.Printf("=== WIDTHS %s : %d octets, %d niveaux (stride %d) ===\n", path, len(raw), n, widthEntry)

	all := make([]triple, n)
	for i := 0; i < n; i++ {
		off := i * widthEntry
		all[i] = triple{
			binary.LittleEndian.Uint32(raw[off:]),
			binary.LittleEndian.Uint32(raw[off+4:]),
			binary.LittleEndian.Uint32(raw[off+8:]),
		}
	}

	// compression par plages : la table est majoritairement constante.
	start := 0
	for i := 1; i <= n; i++ {
		if i < n && all[i] == all[start] {
			continue
		}
		t := all[start]
		label := ""
		if t.a == t.b && t.b == t.c {
			label = fmt.Sprintf("uniforme %d", t.a)
			if t.a == 0x1a {
				label += "  <- PLAFOND 0x1a"
			}
		} else {
			label = "NON UNIFORME"
		}
		if start == i-1 {
			fmt.Printf("  L%-3d        : %2d/%2d/%2d   %s\n", start, t.a, t.b, t.c, label)
		} else {
			fmt.Printf("  L%-3d .. L%-3d: %2d/%2d/%2d   %s\n", start, i-1, t.a, t.b, t.c, label)
		}
		start = i
	}
	fmt.Println()
	return nil
}

func dumpRanges(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	n := len(raw) / rangeEntry
	fmt.Printf("=== RANGES %s : %d octets, %d entrees (stride %d) ===\n", path, len(raw), n, rangeEntry)

	all := make([]aabb, n)
	for i := 0; i < n; i++ {
		off := i * rangeEntry
		f := func(k int) float32 {
			return math.Float32frombits(binary.LittleEndian.Uint32(raw[off+4*k:]))
		}
		all[i] = aabb{f(0), f(1), f(2), f(3), f(4), f(5)}
	}

	start := 0
	for i := 1; i <= n; i++ {
		if i < n && all[i] == all[start] {
			continue
		}
		r := all[start]
		ex, ey, ez := r.extents()
		rng := fmt.Sprintf("L%d", start)
		if start != i-1 {
			rng = fmt.Sprintf("idx %d..%d", start, i-1)
		} else {
			rng = fmt.Sprintf("idx %d", start)
		}
		fmt.Printf("  %-14s %s\n", rng, r)
		fmt.Printf("  %-14s etendues %.4f / %.4f / %.4f\n", "", ex, ey, ez)
		start = i
	}
	fmt.Println()
	return nil
}

// bitLen reproduit FUN_1406d310c : nombre de bits pour representer n.
func bitLen(n uint32) int {
	w := 0
	for n > 0 {
		w++
		n >>= 1
	}
	return w
}

// widthFor reproduit FUN_140be9b88 : W = min(26, bitLen(ceil(extent / (2*step)))).
func widthFor(extent, step float64) int {
	if step < 1e-9 {
		return 0x1a
	}
	n := math.Ceil(extent / (2 * step))
	if n > 0x400000 {
		n = 0x400000
	}
	return min(bitLen(uint32(n)), 0x1a)
}

func deriveWidths() {
	fmt.Println("=== FORME FERMEE  W = min(26, bitLen(ceil(extent / (2*step)))) ===")
	fmt.Println("Objectif : quel pas donne 19 bits sur la plage par defaut DAT_143b8c6d0 (+-100, etendue 200) ?")
	fmt.Println()

	type cand struct {
		name string
		step float64
	}
	cands := []cand{
		{"1/120        (q(16) du dossier)", 1.0 / 120},
		{"1/256", 1.0 / 256},
		{"1/512", 1.0 / 512},
		{"1/1024", 1.0 / 1024},
		{"1/2048", 1.0 / 2048},
		{"1/4096", 1.0 / 4096},
		{"1/8192", 1.0 / 8192},
		{"1/16384", 1.0 / 16384},
		{"0.001 (1 mm)", 0.001},
		{"0.0005", 0.0005},
		{"0.00025", 0.00025},
		{"0.0001", 0.0001},
	}
	extents := []struct {
		name string
		e    float64
	}{
		{"defaut DAT_143b8c6d0 (+-100)", 200},
		{"voisine DAT_143b8c6b8 (+-20000)", 40000},
	}
	for _, ex := range extents {
		fmt.Printf("-- etendue %s = %.1f\n", ex.name, ex.e)
		for _, c := range cands {
			w := widthFor(ex.e, c.step)
			flag := ""
			if w == 19 {
				flag = "   <== 19 : 3x19+2 = 59, le total mesure du port"
			}
			fmt.Printf("   step %-32s -> W = %2d   (3W+2 = %2d)%s\n", c.name, w, 3*w+2, flag)
		}
		fmt.Println()
	}

	fmt.Println("-- inversion : quel intervalle de pas rend exactement W bits sur l'etendue 200 ?")
	for w := 13; w <= 22; w++ {
		lo := 200.0 / (2 * math.Exp2(float64(w)))    // strictement : n < 2^w
		hi := 200.0 / (2 * math.Exp2(float64(w-1)))  // n >= 2^(w-1)
		fmt.Printf("   W=%2d  (3W+2=%2d)  step dans ]%.8f .. %.8f]\n", w, 3*w+2, lo, hi)
	}
}
