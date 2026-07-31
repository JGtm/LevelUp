// tmp_catscan — SCAN PAR CATALOGUE : la recette universelle du chantier armes, transposee a
// l'inventaire.
//
// D'OU VIENT LA RECETTE. Le chantier `feat/filmdec-killweapon` a etabli (RE_LOG 7ter.73) que
// « la marche est un SOUS-ENSEMBLE POSITIONNEL TOTAL du scan », 346/346 sur quatre films : tout
// ce que la marche sequentielle trouve, le scan le trouve AU MEME BIT. La marche ne sert donc
// plus a la couverture, seulement a la precision. Et le mecanisme de leur selectivite est nomme :
// « un enregistrement de la marche porte forcement un tag DU CATALOGUE, sinon le scan ne l'aurait
// pas produit au meme bit ».
//
// LEUR SELECTIVITE VIENT D'UN CATALOGUE, PAS DE BORNES NUMERIQUES. 468 identifiants valides,
// taux d'accident annonce a 1 sur 9 000 000. Mon scan d'i22 utilisait « valeur <= 2 » — d'ou
// 20,4 candidats par lecture vraie.
//
// OR J'AI DEJA LES CATALOGUES, mesures aux positions exactes par la capture CE :
//
//	i47   8 valeurs distinctes sur 512 possibles
//	i42   7 valeurs sur 128
//	i48  13 valeurs sur 1024
//
// Je ne m'en servais pas comme filtre.
//
// LE TAUX DE FAUX POSITIFS EST MESURE, PAS CALCULE. J'ai deja commis l'erreur inverse : un
// calcul suppose des bits equiprobables et je me suis trompe d'un facteur 20 000, parce que le
// flux est sature d'octets nuls. Ici on COMPTE les candidats tombant hors des positions connues.
// Un calcul de probabilite sur un flux dont on ignore la distribution ne vaut rien.
//
// CE QUE CET OUTIL NE FAIT PAS : il ne suppose pas que le catalogue est complet. Il est mesure
// sur UN film ; une valeur legitime jamais observee serait un faux negatif silencieux. Le
// rappel mesure contre la verite de position est precisement ce qui rend ce risque visible.
package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const recSize = 40
const minDistinct = 12

type hit struct {
	EID, TypeIndex, CompIndex, Param4, BitCursor uint32
	Sig                                          [16]byte
}

func main() {
	in := flag.String("in", "", "dump binaire de la capture CE")
	dir := flag.String("film", "", "dossier des chunks du film")
	truth := flag.String("truth", "", "positions de reference (tmp_comptruth -out)")
	comp := flag.Int("comp", 47, "composant a scanner")
	width := flag.Int("bits", 9, "largeur du champ")
	ti := flag.Int("ti", 35, "archetype")
	inv := flag.Bool("invariant", true, "appliquer l'invariant structurel d'i47 (selection dans le masque)")
	flag.Parse()
	if *in == "" || *dir == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_catscan -in <capture> -film <dir> [-comp 47] [-bits 9]")
		os.Exit(2)
	}

	// 1. CATALOGUE — construit depuis les positions EXACTES, pas depuis le scan. C'est la
	// difference entre un catalogue et une liste de ce qu'on a bien voulu accepter.
	cat, nRead, err := buildCatalogue(*in, *dir, *comp, *width, *ti)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("SCAN PAR CATALOGUE — composant i%d, %d bits, archetype %d\n\n", *comp, *width, *ti)
	fmt.Printf("  catalogue bati sur %d lectures localisees : %d valeurs distinctes sur %d possibles\n",
		nRead, len(cat), 1<<uint(*width))
	vals := make([]uint64, 0, len(cat))
	for v := range cat {
		vals = append(vals, v)
	}
	sort.Slice(vals, func(i, j int) bool { return cat[vals[i]] > cat[vals[j]] })
	fmt.Print("  valeurs : ")
	for i, v := range vals {
		if i >= 16 {
			fmt.Printf("… (+%d)", len(vals)-16)
			break
		}
		fmt.Printf("0x%X(%d) ", v, cat[v])
	}
	fmt.Print("\n\n")

	// 2. SCAN de tout le film sur ce seul critere d'appartenance.
	nc := filmdec.CountFilmChunks(*dir)
	type c2 struct{ chunk, bit int }
	var cands []c2
	scanned := 0
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
			lim := len(pay)*8 - *width
			scanned += lim
			for b := 0; b <= lim; b++ {
				v := filmdec.PeekBits(pay, b, *width)
				if _, ok := cat[v]; !ok {
					continue
				}
				if *inv && *comp == 47 && !i47Coherent(v) {
					continue
				}
				cands = append(cands, c2{c, p.Start*8 + b})
			}
		}
	}
	fmt.Printf("  %d positions examinees · %d candidats retenus\n", scanned, len(cands))

	// 3. MESURE contre la verite. Le taux de faux positifs est COMPTE, pas calcule.
	if *truth == "" {
		fmt.Println("\n  (pas de reference fournie : rappel et precision non mesurables)")
		return
	}
	tp, err := loadTruth(*truth, *dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	set := map[c2]bool{}
	for _, t := range tp {
		set[c2{t.chunk, t.bit}] = true
	}
	got := map[c2]bool{}
	for _, c := range cands {
		got[c] = true
	}
	found := 0
	for k := range set {
		if got[k] {
			found++
		}
	}
	fmt.Printf("\n  CONFRONTATION a %d positions de reference —\n\n", len(set))
	fmt.Printf("    RAPPEL    : %d / %d = %.1f %%\n", found, len(set), 100*float64(found)/float64(max(1, len(set))))
	fmt.Printf("    PRECISION : %d / %d = %.2f %%\n", found, len(cands), 100*float64(found)/float64(max(1, len(cands))))
	fmt.Printf("    candidats par lecture vraie : %.1f\n", float64(len(cands))/float64(max(1, len(set))))
	fpRate := float64(len(cands)-found) / float64(max(1, scanned))
	fmt.Printf("\n    TAUX DE FAUX POSITIFS MESURE : %.3e par position de bit\n", fpRate)
	fmt.Printf("    (compte, pas calcule : %d candidats hors reference sur %d positions)\n",
		len(cands)-found, scanned)
}

// i47Coherent applique l'invariant structurel etabli le 2026-07-27 : le type SELECTIONNE
// appartient toujours au masque des types possedes (verifie 12 fois sur 12).
//
//	i47 = [6 bits masque][3 bits selection]
//
// C'est un test GRATUIT et tres selectif : il rejette toute valeur dont la selection designe un
// type non possede, ce qui est le cas de la grande majorite des combinaisons aleatoires.
func i47Coherent(v uint64) bool {
	mask := (v >> 3) & 0x3F
	sel := v & 7
	if sel == 0 {
		return mask == 0 // rien de selectionne <=> rien de possede
	}
	if mask == 0 {
		return false // selection sans possession : incoherent
	}
	return mask&(1<<(sel-1)) != 0
}

// buildCatalogue lit les valeurs aux positions EXACTES (curseur moteur) et en fait le catalogue.
func buildCatalogue(capture, dir string, comp, width, ti int) (map[uint64]int, int, error) {
	raw, err := os.ReadFile(capture)
	if err != nil {
		return nil, 0, fmt.Errorf("lecture capture : %w", err)
	}
	nc := filmdec.CountFilmChunks(dir)
	type pk struct{ start, size int }
	packets := make([][]pk, nc+1)
	chunks := make([][]byte, nc+1)
	sigIdx := map[uint64][]hit{}
	for i := 0; i+recSize <= len(raw); i += recSize {
		b := raw[i : i+recSize]
		var h hit
		h.EID = binary.LittleEndian.Uint32(b[0:])
		h.TypeIndex = binary.LittleEndian.Uint32(b[4:])
		h.CompIndex = binary.LittleEndian.Uint32(b[8:])
		h.BitCursor = binary.LittleEndian.Uint32(b[16:])
		copy(h.Sig[:], b[24:40])
		if h.EID == 0 && h.TypeIndex == 0 && h.BitCursor == 0 && h.CompIndex == 0 {
			break
		}
		if int(h.CompIndex) != comp || int(h.TypeIndex) != ti || !usable(h.Sig) {
			continue
		}
		k := binary.LittleEndian.Uint64(h.Sig[:8])
		sigIdx[k] = append(sigIdx[k], h)
	}
	cat := map[uint64]int{}
	n := 0
	for c := 1; c <= nc; c++ {
		ch, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		chunks[c] = ch
		for _, p := range filmdec.WalkPackets(ch) {
			if p.Type == filmdec.PacketTypeDelta {
				packets[c] = append(packets[c], pk{p.Start, p.Size})
			}
		}
		for off := 0; off+16 <= len(ch); off++ {
			hs, ok := sigIdx[binary.LittleEndian.Uint64(ch[off:off+8])]
			if !ok {
				continue
			}
			for _, h := range hs {
				same := true
				for j := 0; j < 16; j++ {
					if ch[off+j] != h.Sig[j] {
						same = false
						break
					}
				}
				if !same {
					continue
				}
				for _, p := range packets[c] {
					if off < p.start || off >= p.start+p.size {
						continue
					}
					pay := ch[p.start : p.start+p.size]
					cur := int(h.BitCursor)
					if cur >= 0 && cur+width <= len(pay)*8 {
						cat[filmdec.PeekBits(pay, cur, width)]++
						n++
					}
					break
				}
			}
		}
	}
	if len(cat) == 0 {
		return nil, 0, fmt.Errorf("catalogue vide : aucune lecture d'i%d localisee", comp)
	}
	return cat, n, nil
}

type tp struct{ chunk, bit int }

// loadTruth lit les positions de reference et les convertit en position de BIT via le curseur —
// c'est l'identite etablie : bit = paquet.Start*8 + curseur.
func loadTruth(path, dir string) ([]tp, error) {
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
	var out []tp
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
				out = append(out, tp{c, p.start*8 + cur})
				break
			}
		}
	}
	return out, sc.Err()
}

func usable(s [16]byte) bool {
	seen := map[byte]bool{}
	for _, b := range s {
		seen[b] = true
	}
	return len(seen) >= minDistinct
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
