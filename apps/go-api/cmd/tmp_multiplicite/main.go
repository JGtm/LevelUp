// tmp_multiplicite — LE DISCRIMINANT GRATUIT : un vrai record tombe a un bit quelconque, un faux
// positif se repete AU MEME BIT d'un paquet a l'autre.
//
// D'OU VIENT L'IDEE. Le chantier armes la designe comme « le discriminant le plus rentable, et
// il est gratuit » (RE_LOG 7ter.61, fc_self.go). Leur mesure : les faux positifs se concentrent
// a des bits FIXES ET BAS (746, 674, 720, 77) repetes sur 9 a 13 paquets d'un meme film, quand
// les vrais se dispersent.
//
// POURQUOI C'EST STRUCTUREL, ET PAS UNE HEURISTIQUE. La position d'un vrai composant depend de
// tout ce qui le precede dans le paquet : nombre de records, masques, largeurs — autant de
// grandeurs qui varient d'un paquet a l'autre. Elle ne PEUT donc pas etre constante. Un faux
// positif, lui, vient d'un motif d'octets qui revient a offset fixe : en-tete de paquet,
// bourrage, champ de longueur. La constance de position EST la signature du faux.
//
// CE QUE CET OUTIL MESURE, sur les candidats d'un scan et contre la verite de position :
//   - la distribution du nombre de paquets distincts partageant un meme offset de candidat ;
//   - ou tombent les VRAIES lectures dans cette distribution.
//
// Si les vraies sont massivement a multiplicite 1 et les fausses a multiplicite elevee, le test
// est utilisable et son seuil sort de la mesure — il n'est pas decrete.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

// i22Bits : largeur mesuree du composant (capture CE, 100 % des lectures).
const i22Bits = 3 + 4*8

const (
	grenadeMaxTypes = 2
	grenadeMaxEach  = 2
)

type cand struct {
	chunk, pktIdx, offInPkt, absBit int
}

func main() {
	dir := flag.String("film", "", "dossier des chunks du film")
	truth := flag.String("truth", "", "positions de reference (tmp_comptruth -out)")
	flag.Parse()
	if *dir == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_multiplicite -film <dir> [-truth positions.tsv]")
		os.Exit(2)
	}

	nc := filmdec.CountFilmChunks(*dir)
	var cands []cand
	type pkRef struct{ chunk, start, size int }
	var allPkts []pkRef
	for c := 1; c <= nc; c++ {
		chunk, err := filmdec.ReadFilmChunk(*dir, c)
		if err != nil {
			continue
		}
		pi := 0
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta {
				continue
			}
			allPkts = append(allPkts, pkRef{c, p.Start, p.Size})
			pay := p.Payload(chunk)
			lim := len(pay)*8 - i22Bits
			for b := 0; b <= lim; b++ {
				if !validI22(pay, b) {
					continue
				}
				cands = append(cands, cand{c, pi, b, p.Start*8 + b})
			}
			pi++
		}
	}
	fmt.Printf("MULTIPLICITE DE POSITION — %s\n\n", *dir)
	fmt.Printf("  %d paquets delta · %d candidats i22 (bornes du jeu seules)\n", len(allPkts), len(cands))

	// MULTIPLICITE = nombre de PAQUETS DISTINCTS portant un candidat au MEME offset interne.
	byOff := map[int]map[int]bool{}
	for _, c := range cands {
		if byOff[c.offInPkt] == nil {
			byOff[c.offInPkt] = map[int]bool{}
		}
		byOff[c.offInPkt][c.chunk*100000+c.pktIdx] = true
	}
	mult := func(off int) int { return len(byOff[off]) }

	hist := map[int]int{}
	for _, c := range cands {
		hist[mult(c.offInPkt)]++
	}
	fmt.Println("\n  distribution des candidats par multiplicite —")
	printHist(hist, len(cands))

	if *truth == "" {
		fmt.Println("\n  (pas de reference : impossible de dire ou tombent les VRAIES lectures)")
		return
	}
	tp, err := loadTruth(*truth, *dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	real := map[int]bool{}
	for _, t := range tp {
		real[t] = true
	}
	tHist, fHist := map[int]int{}, map[int]int{}
	nT, nF := 0, 0
	for _, c := range cands {
		m := mult(c.offInPkt)
		if real[c.absBit] {
			tHist[m]++
			nT++
		} else {
			fHist[m]++
			nF++
		}
	}
	fmt.Printf("\n  %d positions de reference · %d candidats VRAIS · %d FAUX\n", len(real), nT, nF)
	fmt.Println("\n  VRAIES lectures, par multiplicite —")
	printHist(tHist, nT)
	fmt.Println("\n  FAUSSES, par multiplicite —")
	printHist(fHist, nF)

	// LE SEUIL SORT DE LA MESURE. On publie, pour chaque seuil possible, ce que « garder les
	// candidats de multiplicite <= s » conserve et ce qu'il ecarte. Le bon seuil est celui qui
	// ecarte beaucoup de faux en gardant tous les vrais — il se lit, il ne se decrete pas.
	fmt.Println("\n  CHOIX DU SEUIL : garder les candidats de multiplicite <= s —")
	fmt.Printf("    %4s  %12s  %12s  %10s\n", "s", "vrais gardes", "faux gardes", "candidats/vrai")
	ks := keysOf(tHist, fHist)
	for _, s := range ks {
		if s > 12 {
			break
		}
		kt, kf := 0, 0
		for m, n := range tHist {
			if m <= s {
				kt += n
			}
		}
		for m, n := range fHist {
			if m <= s {
				kf += n
			}
		}
		ratio := 0.0
		if kt > 0 {
			ratio = float64(kt+kf) / float64(kt)
		}
		fmt.Printf("    %4d  %6d %5.1f %%  %6d %5.1f %%  %10.1f\n",
			s, kt, 100*float64(kt)/float64(max(1, nT)),
			kf, 100*float64(kf)/float64(max(1, nF)), ratio)
	}
}

func validI22(pay []byte, b int) bool {
	if filmdec.PeekBits(pay, b, 3) != 4 {
		return false
	}
	nz := 0
	for i := 0; i < 4; i++ {
		x := filmdec.PeekBits(pay, b+3+i*8, 8)
		if x > grenadeMaxEach {
			return false
		}
		if x > 0 {
			nz++
		}
	}
	return nz <= grenadeMaxTypes
}

func printHist(h map[int]int, tot int) {
	ks := make([]int, 0, len(h))
	for k := range h {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	for _, k := range ks {
		if k > 14 {
			rest := 0
			for m, n := range h {
				if m > 14 {
					rest += n
				}
			}
			fmt.Printf("    %3d+ paquets : %6d  %5.1f %%\n", 15, rest, 100*float64(rest)/float64(max(1, tot)))
			break
		}
		bar := ""
		for i := 0; i < h[k]*30/max(1, tot); i++ {
			bar += "#"
		}
		fmt.Printf("    %3d  paquets : %6d  %5.1f %%  %s\n", k, h[k], 100*float64(h[k])/float64(max(1, tot)), bar)
	}
}

func keysOf(a, b map[int]int) []int {
	s := map[int]bool{}
	for k := range a {
		s[k] = true
	}
	for k := range b {
		s[k] = true
	}
	out := make([]int, 0, len(s))
	for k := range s {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func loadTruth(path, dir string) ([]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	nc := filmdec.CountFilmChunks(dir)
	type pk struct{ start, size int }
	packets := make([][]pk, nc+1)
	for c := 1; c <= nc; c++ {
		ch, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(ch) {
			if p.Type == filmdec.PacketTypeDelta {
				packets[c] = append(packets[c], pk{p.Start, p.Size})
			}
		}
	}
	var out []int
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fs := strings.Split(line, "\t")
		if len(fs) < 5 {
			continue
		}
		c, _ := strconv.Atoi(fs[0])
		off, _ := strconv.Atoi(fs[1])
		cur, _ := strconv.Atoi(fs[4])
		if c >= len(packets) {
			continue
		}
		for _, p := range packets[c] {
			if off >= p.start && off < p.start+p.size {
				out = append(out, p.start*8+cur)
				break
			}
		}
	}
	return out, sc.Err()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
