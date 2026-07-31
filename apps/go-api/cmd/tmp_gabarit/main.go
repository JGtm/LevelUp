// tmp_gabarit — LE GABARIT RIGIDE : plaquer la structure COMPLETE d'un record delta a chaque
// position de bit, au lieu de tester des champs isoles.
//
// POURQUOI CE CHANGEMENT D'APPROCHE. Deux transpositions du chantier armes ont ete tentees et
// REFUTEES par la mesure (2026-07-27) :
//   - catalogue de valeurs        inoperant sur un champ de 9 bits (rapport alphabet/catalogue :
//     2^32 pour 468 chez eux, 512 pour 8 chez moi) ;
//   - multiplicite de position    distributions des vrais et des faux IDENTIQUES (44,1 % contre
//     45,2 %), zero information.
//
// Ce qui reste de leur recette est ce que j'avais mal lu : leur scan plaque 58 bits de structure
// avec QUATRE BITS DE PORTE A VALEUR IMPOSEE, dont la semantique est LUE dans la grammaire du
// deserialiseur. C'est la contrainte qu'aucun catalogue ni aucune statistique ne fournit.
//
// LE GABARIT D'UN RECORD DELTA, tire de frame_records.go et traverse.go :
//
//	p+0                R(1)  prefixe de type          == 1     -> DELTA        [1 bit impose]
//	p+1 .. p+idLow     R(N)  identifiant bas
//	                   R(2)  tag                                                [libre]
//	                   R(1)  porte du masque          == 0     -> epars        [1 bit impose]
//	                   R(3)  compteur                 <= 7     (borne par la largeur)
//	                   count x R(6) index de composant                          [voir ci-dessous]
//
// LA CONTRAINTE GRATUITE QUE JE N'EXPLOITAIS PAS. Les composants sont dispatches par index
// CROISSANT — c'est observe sur toute la capture. Les index du masque sont donc STRICTEMENT
// CROISSANTS. Quatre valeurs de 6 bits tirees au hasard n'ont qu'une chance sur 4! = 24 d'etre
// croissantes : 4,6 bits de contrainte, gratuits. C'est exactement la nature des bits de porte
// imposes qui fait marcher leur gabarit.
//
// VALIDATION, reprise telle quelle de leur methode (RE_LOG 7ter.61 B3) : decaler le gabarit de
// `delta` bits et mesurer le TAUX D'APPARIEMENT, pas le compte de candidats. Le compte ne
// tranche pas — il produit des echos aux decalages voisins. Le taux, lui, s'effondre hors
// position.
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
	grenadeMaxTypes = 2
	grenadeMaxEach  = 2
	i22Bits         = 3 + 4*8
)

// compWidth est la largeur MESUREE de chaque composant (capture CE du 2026-07-27, largeur
// dominante). Les composants absents de cette table rendent le gabarit indecidable : on
// s'arrete alors plutot que de deviner.
// compWidths donne TOUTES les largeurs mesurees de chaque composant (capture CE, film
// 9e8fb31b), la dominante en tete. Plusieurs composants du chemin vers i22 en ont plusieurs, et
// pas marginalement : i18 est a 52 %/48 % entre 2 et 8 bits.
//
// i11 est volontairement ABSENT : six largeurs mesurees dont aucune ne domine (191 a 58 %, 147
// a 24 %, 127 a 9 %, 223 a 6 %) — l inclure ferait exploser la recherche sans garantie. Un
// record qui le porte fait echouer le gabarit plutot que de deviner.
var compWidths = map[uint32][]int{
	0: {47}, 1: {31, 2}, 2: {9}, 4: {11}, 5: {29}, 6: {358}, 9: {334},
	13: {8}, 14: {4}, 15: {72, 76}, 17: {52, 75, 77, 76}, 18: {2, 8},
	21: {25}, 22: {35},
}

type match struct {
	chunk, absBit, i22Bit int
	mask                  []uint32
}

func main() {
	dir := flag.String("film", "", "dossier des chunks du film")
	truth := flag.String("truth", "", "positions de reference i22 (tmp_comptruth -out)")
	idLow := flag.Int("idlow", 11, "largeur du champ identifiant bas (valeur de RUNTIME)")
	sweep := flag.Bool("sweep", false, "balayer le decalage du gabarit pour le valider")
	flag.Parse()
	if *dir == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_gabarit -film <dir> [-truth pos.tsv] [-idlow 11] [-sweep]")
		os.Exit(2)
	}

	if *sweep && *truth != "" {
		tp, err := loadTruth(*truth, *dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("VALIDATION DU GABARIT — balayage du decalage (idLow=%d)\n\n", *idLow)
		fmt.Printf("  Le COMPTE de candidats ne tranche pas : il produit des echos aux decalages\n")
		fmt.Printf("  voisins. C'est le TAUX D'APPARIEMENT qui tranche.\n\n")
		fmt.Printf("  %6s  %12s  %10s  %s\n", "decal.", "candidats", "apparies", "taux")
		for d := -8; d <= 8; d++ {
			ms := scan(*dir, *idLow, d)
			ap := countMatched(ms, tp)
			rate := 0.0
			if len(ms) > 0 {
				rate = 100 * float64(ap) / float64(len(ms))
			}
			mark := ""
			if d == 0 {
				mark = "   <== position nominale"
			}
			fmt.Printf("  %+6d  %12d  %10d  %5.1f %%%s\n", d, len(ms), ap, rate, mark)
		}
		return
	}

	ms := scan(*dir, *idLow, 0)
	fmt.Printf("GABARIT RIGIDE — %s (idLow=%d)\n\n", *dir, *idLow)
	fmt.Printf("  %d records candidats portant i22 dans leur masque\n", len(ms))
	if *truth == "" {
		showSample(ms)
		return
	}
	tp, err := loadTruth(*truth, *dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ap := countMatched(ms, tp)
	fmt.Printf("\n  CONFRONTATION a %d positions de reference —\n\n", len(tp))
	fmt.Printf("    RAPPEL    : %d / %d = %.1f %%\n", ap, len(tp), 100*float64(ap)/float64(max(1, len(tp))))
	fmt.Printf("    PRECISION : %d / %d = %.1f %%\n", ap, len(ms), 100*float64(ap)/float64(max(1, len(ms))))
	fmt.Printf("    candidats par lecture vraie : %.1f\n", float64(len(ms))/float64(max(1, len(tp))))
	showSample(ms)
}

// scan plaque le gabarit a chaque position. `shift` decale le bloc du masque par rapport a
// l'en-tete : c'est le levier de validation, il doit degrader le taux d'appariement.
func scan(dir string, idLow, shift int) []match {
	nc := filmdec.CountFilmChunks(dir)
	var out []match
	for c := 1; c <= nc; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta {
				continue
			}
			pay := p.Payload(chunk)
			total := len(pay) * 8
			for b := 0; b+256 <= total; b++ {
				m, ok := tryTemplate(pay, b, idLow, shift)
				if !ok {
					continue
				}
				m.chunk = c
				m.absBit = p.Start*8 + b
				m.i22Bit = p.Start*8 + m.i22Bit
				out = append(out, m)
			}
		}
	}
	return out
}

// tryTemplate applique le gabarit complet. Chaque `return false` est une contrainte de la
// grammaire, pas un reglage : elles viennent toutes de frame_records.go / traverse.go ou de la
// capture CE.
func tryTemplate(pay []byte, b, idLow, shift int) (match, bool) {
	var m match
	// [1 bit impose] prefixe de type : 1 => DELTA. Un 0 ouvre un code prefixe de 2 bits, donc un
	// autre type de record — hors gabarit.
	if filmdec.PeekBits(pay, b, 1) != 1 {
		return m, false
	}
	p := b + 1 + idLow + 2 // identifiant bas + tag, non contraints
	p += shift             // levier de validation
	if p < 0 {
		return m, false
	}
	// LA PORTE DU MASQUE OUVRE DEUX BRANCHES, et il faut les DEUX.
	//
	// MESURE QUI L'IMPOSE (2026-07-27) : 75 % des lectures d'i22 vivent dans des records a
	// masque DENSE. Or un masque de plus de 7 composants NE PEUT PAS etre epars — le compteur
	// fait 3 bits. Ne traiter que la branche eparse plafonnait donc le rappel a 23,7 %, ce qui
	// correspond presque exactement aux 25 % de records epars. Ce n'etait pas un defaut de
	// largeur, c'etait une branche manquante.
	dense := filmdec.PeekBits(pay, p, 1) == 1
	p++
	has22 := false
	if dense {
		// Branche DENSE : R(64), un bit par composant.
		//
		// ORDRE DES BITS : le composant i est lu au bit p+i, dans l'ordre de lecture du flux.
		// C'est le MEME ordre que la branche eparse, dont les index sont croissants — les deux
		// branches decrivent le meme ensemble, elles ne peuvent pas le parcourir a l'envers.
		// (Premiere version fautive : reconstruire un entier 64 bits puis tester 1<<22 mettait
		// le composant 0 sur le DERNIER bit lu, donc le masque a l'envers.)
		base := p
		p += 64
		if filmdec.PeekBits(pay, base+22, 1) == 0 {
			return m, false
		}
		// Les composants i60 a i63 ne sont JAMAIS dispatches sur ce film (mesure : 0 lecture).
		// Un masque qui les pose est du bruit — contrainte gratuite, 4 bits.
		for i := 60; i < 64; i++ {
			if filmdec.PeekBits(pay, base+i, 1) == 1 {
				return m, false
			}
		}
		for i := 0; i < 60; i++ {
			if filmdec.PeekBits(pay, base+i, 1) == 1 {
				m.mask = append(m.mask, uint32(i))
			}
		}
		if len(m.mask) <= 7 {
			return m, false // un masque de 7 ou moins serait code en epars : incoherent
		}
		has22 = true
	} else {
		count := int(filmdec.PeekBits(pay, p, 3))
		p += 3
		if count == 0 {
			return m, false
		}
		// Index STRICTEMENT CROISSANTS — les composants sont dispatches par index croissant, donc
		// le masque l'est aussi. Contrainte gratuite : 4 valeurs de 6 bits au hasard n'ont qu'une
		// chance sur 24 d'etre croissantes.
		prev := -1
		for i := 0; i < count; i++ {
			idx := int(filmdec.PeekBits(pay, p, 6))
			p += 6
			if idx <= prev {
				return m, false
			}
			prev = idx
			if idx == 22 {
				has22 = true
			}
			m.mask = append(m.mask, uint32(idx))
		}
	}
	if !has22 {
		return m, false // on ne cherche que les records portant i22
	}
	// Derouler les composants du masque jusqu'a i22 en additionnant leurs largeurs.
	//
	// POURQUOI UNE RECHERCHE ET NON UNE SOMME. Plusieurs composants du chemin ont PLUSIEURS
	// largeurs mesurees, et pas marginalement : i18 est a 52 %/48 % entre 2 et 8 bits, i17 a
	// cinq largeurs (52 a 51 %, puis 75, 77, 76…), i11 six, i15 deux, i1 une variante rare a
	// 2 bits. Retenir la seule largeur DOMINANTE faisait echouer tout record empruntant une
	// variante — ce qui plafonnait le rappel a 23,7 % alors que la precision atteignait 96,7 %.
	//
	// On essaie donc toutes les combinaisons. Le produit est BORNE par construction (au plus
	// quelques centaines pour un masque reel) et on s'arrete a la premiere qui fait tomber i22
	// sur des valeurs valides. C'est la precision deja acquise qui donne la marge de se le
	// permettre : a 1,03 candidat par lecture vraie, elargir la recherche ne noie rien.
	var path []uint32
	for _, idx := range m.mask {
		if idx == 22 {
			break
		}
		if _, ok := compWidths[idx]; !ok {
			return m, false // largeur inconnue : indecidable, on abandonne plutot que de deviner
		}
		path = append(path, idx)
	}
	if off, ok := searchOffset(pay, p, path, 0); ok {
		m.i22Bit = off
		return m, true
	}
	return m, false
}

// searchOffset essaie les largeurs possibles de chaque composant du chemin, dans l'ordre, et
// rend la premiere position ou i22 est valide. Profondeur bornee par la taille du masque.
func searchOffset(pay []byte, p int, path []uint32, i int) (int, bool) {
	if i == len(path) {
		if p+i22Bits > len(pay)*8 {
			return 0, false
		}
		// i22 doit satisfaire ses bornes de jeu, verifiees sur 249 lectures reelles.
		if !validI22(pay, p) {
			return 0, false
		}
		return p, true
	}
	for _, w := range compWidths[path[i]] {
		if p+w > len(pay)*8 {
			continue
		}
		if off, ok := searchOffset(pay, p+w, path, i+1); ok {
			return off, true
		}
	}
	return 0, false
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

func countMatched(ms []match, truth []int) int {
	set := map[int]bool{}
	for _, t := range truth {
		set[t] = true
	}
	seen := map[int]bool{}
	n := 0
	for _, m := range ms {
		if set[m.i22Bit] && !seen[m.i22Bit] {
			seen[m.i22Bit] = true
			n++
		}
	}
	return n
}

func showSample(ms []match) {
	if len(ms) == 0 {
		return
	}
	fmt.Println("\n  echantillon (chunk, bit du record, bit d'i22, masque) —")
	for i, m := range ms {
		if i >= 10 {
			fmt.Printf("    … %d autres\n", len(ms)-10)
			break
		}
		fmt.Printf("    %02d  %10d  %10d  %v\n", m.chunk, m.absBit, m.i22Bit, m.mask)
	}
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
