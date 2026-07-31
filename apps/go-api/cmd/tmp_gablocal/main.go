// tmp_gablocal — GABARIT LOCAL : s'ancrer sur le composant vise et prolonger VERS L'AVANT, sans
// jamais remonter au debut du record.
//
// L'ERREUR QUE CECI CORRIGE, signalee par l'utilisateur : « le truc qui marchait bien sur
// l'autre worktree, c'est qu'on n'avait pas besoin de tout parcourir pour atteindre le composant
// desire ». Mon gabarit precedent (cmd/tmp_gabarit) partait du debut du record et traversait
// tous les composants precedents pour atteindre i22. Consequences mesurees : une seule largeur
// variable sur le chemin fait echouer le gabarit, et le rappel plafonne a 24 % malgre 96 % de
// precision.
//
// EN RELISANT LEUR GABARIT DE 58 BITS, l'evidence saute : `mort`, `porte du tag`, `tag`, `porte
// victime`, `victime`, `porte tueur`, `tueur`, `categorie` sont TOUS des champs INTERNES au
// composant i11. Ils ne touchent ni l'en-tete du record ni le masque. Ils s'ancrent sur la
// structure interne du composant vise, qui porte a elle seule assez de bits imposes.
//
// CE QUE FAIT CE GABARIT :
//
//	[i22 : 35 bits]  compteur == 4 (3 bits imposes) + 4 octets tous dans {0,1,2}
//	[maillon]        un composant d'index > 22, largeur connue, traverse
//	[i47 : 9 bits]   valeur du catalogue ET invariant selection-dans-le-masque
//
// i25 est le maillon naturel : largeur UNIQUE de 10 bits (aucune variante mesuree) et present
// dans 99,5 % des records. L'ecart i22 -> i47 mesure vaut +45 bits dans 27 % des cas, soit
// exactement 35 (i22) + 10 (i25) — le modele se referme.
//
// ON N'EXIGE JAMAIS i47 : il n'est present que dans 96 des 133 records portant i22. Un candidat
// sans prolongement valide n'est pas rejete, il est CLASSE separement — c'est la difference
// entre un test et un filtre, et elle decide du rappel.
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

const (
	i22Bits         = 3 + 4*8
	i47Bits         = 9
	grenadeMaxTypes = 2
	grenadeMaxEach  = 2
)

// chainWidths : largeurs des composants pouvant separer i22 d'i47, tous d'index strictement
// compris entre 22 et 47. Seules les largeurs MESUREES figurent ici.
var chainWidths = map[uint32][]int{
	23: {19, 31}, 24: {12}, 25: {10}, 26: {22}, 28: {10, 34},
	30: {14, 10, 2}, 31: {11}, 32: {9}, 33: {14, 10, 2}, 34: {11}, 35: {9}, 42: {7, 5},
	43: {15}, 44: {15}, 45: {17}, 46: {17},
}

func main() {
	dir := flag.String("film", "", "dossier des chunks du film")
	truth := flag.String("truth", "", "positions de reference i22")
	maxChain := flag.Int("chain", 3, "nombre maximum de composants intercales entre i22 et i47")
	flag.Parse()
	if *dir == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_gablocal -film <dir> [-truth pos.tsv] [-chain 3]")
		os.Exit(2)
	}

	nc := filmdec.CountFilmChunks(*dir)
	var withI47, bare []int
	for c := 1; c <= nc; c++ {
		chunk, err := filmdec.ReadFilmChunk(*dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta {
				continue
			}
			pay := p.Payload(chunk)
			lim := len(pay)*8 - i22Bits
			for b := 0; b <= lim; b++ {
				if !validI22(pay, b) {
					continue
				}
				abs := p.Start*8 + b
				if extendsToI47(pay, b+i22Bits, *maxChain, 0) {
					withI47 = append(withI47, abs)
				} else {
					bare = append(bare, abs)
				}
			}
		}
	}
	fmt.Printf("GABARIT LOCAL — ancre sur i22, prolonge vers l'avant\n\n")
	fmt.Printf("  %d candidats avec prolongement i47 valide\n", len(withI47))
	fmt.Printf("  %d candidats sans prolongement (non rejetes : i47 n'est pas toujours present)\n\n",
		len(bare))

	if *truth == "" {
		return
	}
	tp, err := loadTruth(*truth, *dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	set := map[int]bool{}
	for _, t := range tp {
		set[t] = true
	}
	hitW, hitB := count(withI47, set), count(bare, set)
	fmt.Printf("  CONFRONTATION a %d positions de reference —\n\n", len(set))
	fmt.Printf("    %-28s %8s %8s %10s %12s\n", "population", "cands", "vrais", "precision", "rappel")
	row := func(name string, n, h int) {
		fmt.Printf("    %-28s %8d %8d %9.1f %% %11.1f %%\n", name, n, h,
			100*float64(h)/float64(max(1, n)), 100*float64(h)/float64(max(1, len(set))))
	}
	row("avec prolongement i47", len(withI47), hitW)
	row("sans prolongement", len(bare), hitB)
	row("TOTAL", len(withI47)+len(bare), hitW+hitB)
	fmt.Println("\n  LECTURE — le prolongement doit CONCENTRER les vrais : si sa precision ecrase")
	fmt.Println("  celle des candidats nus, il est un test utile ; si les deux se valent, il")
	fmt.Println("  n'apporte rien et il faut chercher un autre maillon.")
}

// extendsToI47 essaie de traverser jusqu'a `depth` composants intercales, chacun a l'une de ses
// largeurs mesurees, et cherche un i47 valide au bout. Zero composant intercale est un cas
// legitime : i22 et i47 peuvent se suivre directement.
func extendsToI47(pay []byte, p, depth, used int) bool {
	if p+i47Bits <= len(pay)*8 && validI47(pay, p) {
		return true
	}
	if used >= depth {
		return false
	}
	for _, ws := range chainWidths {
		for _, w := range ws {
			if p+w+i47Bits > len(pay)*8 {
				continue
			}
			if extendsToI47(pay, p+w, depth, used+1) {
				return true
			}
		}
	}
	return false
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

// validI47 applique la grammaire craquee le 2026-07-27 : [6 bits masque][3 bits selection], le
// type selectionne appartenant TOUJOURS au masque (verifie 12 fois sur 12), et les deux bits de
// poids fort du masque toujours nuls (4 types de grenades dans un champ de 6 bits).
func validI47(pay []byte, b int) bool {
	v := filmdec.PeekBits(pay, b, i47Bits)
	mask := (v >> 3) & 0x3F
	sel := v & 7
	if mask&0x30 != 0 {
		return false // types 5 et 6 inexistants
	}
	if sel == 0 {
		return mask == 0
	}
	if sel > 4 {
		return false
	}
	return mask&(1<<(sel-1)) != 0
}

func count(xs []int, set map[int]bool) int {
	n := 0
	for _, x := range xs {
		if set[x] {
			n++
		}
	}
	return n
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
	sort.Ints(out)
	return out, sc.Err()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
