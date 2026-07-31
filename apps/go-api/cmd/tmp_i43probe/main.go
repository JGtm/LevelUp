// tmp_i43probe — L'ANCRE LARGE : l'identifiant d'arme dans le composant i43.
//
// POURQUOI CE COMPOSANT. Le gabarit rigide atteint 96,7 % de precision sur i22 mais 24 % de
// rappel ; ancre localement il fait l'inverse (97,9 % / 5,5 %). Il manque un CHAMP LARGE, parce
// que la selectivite vient de la LARGEUR du champ et non de la qualite du catalogue — mesure du
// 2026-07-27 : un catalogue de 8 valeurs sur 512 rend 3,2 millions de candidats.
//
// i43 (weapon-state-type-info) est le seul champ large du voisinage : forme longue de ~195 a
// 204 bits, et il co-occurre avec i22 dans 93 % des records de spawn — le taux le plus eleve de
// toute la table. Ses trente premiers bits ont un taux d'occupation proche de 50 %, signature
// d'un identifiant et non d'un champ structure.
//
// LE TEST ET SON RESULTAT (2026-07-27). Le chantier armes a etabli que les identifiants d'arme
// du film portent tous le meme suffixe sur leurs 32 bits bas : 0x42C9679F. Mesure :
//
//	suffixe present sur les positions reelles : 93 / 192 = 48,4 %
//	CONTROLE, positions decalees              :  0 / 192 =  0,0 %
//	offset dominant                           : +33 bits, 85 fois sur 93 (91,4 %)
//
// Zero faux positif au controle. i43 porte donc un identifiant d'arme 64 bits a place FIXE, et
// les 48,4 % recoupent la repartition des largeurs — la forme courte (15 bits, 44 % des
// lectures) ne porte pas d'identifiant, la forme longue si.
//
// C'EST L'ANCRE CHERCHEE : un motif de 32 bits a valeur imposee, du meme ordre de selectivite
// que celui qui fait marcher le scan du chantier armes.
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

// weaponSuffix est le suffixe commun des identifiants d'arme, etabli par le chantier armes.
const weaponSuffix = 0x42C9679F

type pk struct{ start, size int }

type tpos struct{ chunk, byteOff, cursor int }

func main() {
	dir := flag.String("film", "", "dossier des chunks du film")
	truth := flag.String("truth", "", "positions de reference i43 (tmp_comptruth -out)")
	span := flag.Int("span", 256, "fenetre de recherche en bits depuis la position du composant")
	flag.Parse()
	if *dir == "" || *truth == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_i43probe -film <dir> -truth i43_positions.tsv")
		os.Exit(2)
	}

	nc := filmdec.CountFilmChunks(*dir)
	packets := make([][]pk, nc+1)
	chunks := make([][]byte, nc+1)
	for c := 1; c <= nc; c++ {
		b, err := filmdec.ReadFilmChunk(*dir, c)
		if err != nil {
			continue
		}
		chunks[c] = b
		for _, p := range filmdec.WalkPackets(b) {
			if p.Type == filmdec.PacketTypeDelta {
				packets[c] = append(packets[c], pk{p.Start, p.Size})
			}
		}
	}
	tp, err := loadTruth(*truth)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("SONDE i43 — suffixe d'arme 0x%08X\n\n", weaponSuffix)
	fmt.Printf("  %d positions de reference · fenetre de %d bits\n\n", len(tp), *span)

	hits := map[int]int{}
	found, scanned, ctrl := 0, 0, 0
	for _, t := range tp {
		pay, ok := payloadOf(chunks, packets, t)
		if !ok {
			continue
		}
		scanned++
		if d, ok := findSuffix(pay, t.cursor, *span); ok {
			hits[d]++
			found++
		}
		// CONTROLE NEGATIF : meme paquet, position decalee hors du composant. Meme regime
		// statistique, mais on ne doit plus rien trouver.
		if _, ok := findSuffix(pay, t.cursor+*span*2, *span); ok {
			ctrl++
		}
	}
	fmt.Printf("  POSITIONS REELLES : %d / %d portent le suffixe (%.1f %%)\n",
		found, scanned, pct(found, scanned))
	fmt.Printf("  CONTROLE decale   : %d / %d (%.1f %%)\n\n", ctrl, scanned, pct(ctrl, scanned))
	if found == 0 {
		fmt.Println("  RESULTAT : le suffixe n'apparait pas dans i43. Chercher ailleurs.")
		return
	}

	rows := sortedCounts(hits)
	fmt.Println("  OFFSETS du suffixe depuis le debut d'i43 —")
	for i, r := range rows {
		if i >= 8 {
			fmt.Printf("    … %d autres\n", len(rows)-8)
			break
		}
		fmt.Printf("    +%4d bits : %4d fois  %5.1f %%\n", r.k, r.n, pct(r.n, found))
	}

	// LES IDENTIFIANTS. Le suffixe etant a l'offset dominant, les 32 bits hauts le precedent
	// immediatement : la valeur 64 bits commence donc 32 bits avant. On publie leur
	// distribution — un petit nombre de valeurs recurrentes signe un catalogue d'armes ; du
	// bruit produirait autant de valeurs que de lectures.
	best := rows[0].k
	ids := map[uint64]int{}
	for _, t := range tp {
		pay, ok := payloadOf(chunks, packets, t)
		if !ok {
			continue
		}
		b := t.cursor + best - 32
		if b < 0 || b+64 > len(pay)*8 {
			continue
		}
		if filmdec.PeekBits(pay, b+32, 32) != weaponSuffix {
			continue
		}
		ids[filmdec.PeekBits(pay, b, 64)]++
	}
	fmt.Printf("\n  IDENTIFIANTS 64 bits lus a +%d — %d valeurs distinctes sur %d lectures\n",
		best-32, len(ids), sumOf(ids))
	type iv struct {
		v uint64
		n int
	}
	var irows []iv
	for v, n := range ids {
		irows = append(irows, iv{v, n})
	}
	sort.Slice(irows, func(i, j int) bool { return irows[i].n > irows[j].n })
	for i, r := range irows {
		if i >= 16 {
			fmt.Printf("    … %d autres\n", len(irows)-16)
			break
		}
		fmt.Printf("    0x%016X  %3d fois\n", r.v, r.n)
	}
	fmt.Println("\n  LECTURE — peu de valeurs, chacune vue plusieurs fois : c'est un catalogue.")
	fmt.Println("  Beaucoup de valeurs vues une seule fois : ce serait du bruit.")
}

func payloadOf(chunks [][]byte, packets [][]pk, t tpos) ([]byte, bool) {
	if t.chunk >= len(chunks) || chunks[t.chunk] == nil {
		return nil, false
	}
	for _, p := range packets[t.chunk] {
		if t.byteOff >= p.start && t.byteOff < p.start+p.size {
			return chunks[t.chunk][p.start : p.start+p.size], true
		}
	}
	return nil, false
}

func findSuffix(pay []byte, from, span int) (int, bool) {
	total := len(pay) * 8
	for d := 0; d < span; d++ {
		p := from + d
		if p < 0 || p+32 > total {
			continue
		}
		if filmdec.PeekBits(pay, p, 32) == weaponSuffix {
			return d, true
		}
	}
	return 0, false
}

type kn struct{ k, n int }

func sortedCounts(m map[int]int) []kn {
	out := make([]kn, 0, len(m))
	for k, n := range m {
		out = append(out, kn{k, n})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].n > out[j].n })
	return out
}

func sumOf(m map[uint64]int) int {
	t := 0
	for _, n := range m {
		t += n
	}
	return t
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

func loadTruth(path string) ([]tpos, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []tpos
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
		o, _ := strconv.Atoi(fs[1])
		cur, _ := strconv.Atoi(fs[4])
		out = append(out, tpos{c, o, cur})
	}
	return out, sc.Err()
}
