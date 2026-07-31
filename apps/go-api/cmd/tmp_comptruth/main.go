// tmp_comptruth — LOCALISER dans le film, a l'octet pres, chaque lecture d'un composant par le
// moteur. C'est le juge dont tout scanner a besoin.
//
// LE PROBLEME QUE CA RESOUT. Notre decodeur marche les records sequentiellement depuis le debut
// de chaque paquet, et il decroche : mesure du 2026-07-27, 2,0 records decodes par frame contre
// 13,7 traites par le moteur — on perd les onze douziemes de chaque paquet. Le chantier armes a
// bute sur le meme mur et l'a CONTOURNE (idee de l'utilisateur : « le fichier ne se lit pas en
// continu ») : balayer toutes les positions de bit et ne retenir que ce qui passe plusieurs
// tests simultanes. Plafond casse, 88,7 % -> 97,6 %.
//
// Mais un scanner sans juge est un generateur d'illusions : il rend des candidats plausibles et
// rien ne dit lesquels sont vrais. C'est exactement le piege qui a coute cher sur la geometrie.
//
// LE JUGE. La capture CE du dispatch journalise, pour chaque composant deserialise, 16 octets
// bruts lus depuis [bitreader+0x40] — le pointeur d'octet courant DANS LE CHUNK DECOMPRESSE.
// Ces octets existent a l'identique dans le film hors ligne. Retrouver la signature donne donc
// l'OFFSET EXACT auquel le moteur a lu ce composant. On obtient la liste complete et exacte des
// positions a trouver : precision ET rappel deviennent mesurables, plus estimes.
//
// UNICITE. Une fenetre de 16 octets porte 128 bits ; la probabilite qu'elle apparaisse par
// hasard dans un film de 20 Mo est de 2e7/2^128, negligeable. On rejette quand meme les
// signatures degenerees (bourrage) et on SIGNALE les signatures trouvees plusieurs fois plutot
// que d'en choisir une : une position ambigue ne doit pas entrer dans un jeu de reference.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const recSize = 40

// minDistinct ecarte les fenetres de bourrage, qui n'identifient rien.
const minDistinct = 12

type hit struct {
	EID, TypeIndex, CompIndex, Param4, BitCursor uint32
	Sig                                          [16]byte
}

// found est une lecture de composant localisee dans le film.
type found struct {
	Chunk, Off int
	H          hit
}

func main() {
	in := flag.String("in", "", "dump binaire de la capture CE")
	dir := flag.String("film", "", "dossier des chunks du film")
	comp := flag.Int("comp", 22, "index du composant a localiser")
	ti := flag.Int("ti", 35, "archetype (-1 = tous)")
	maxN := flag.Int("max", 400, "nombre maximum de lectures a localiser")
	out := flag.String("out", "", "ecrire les positions trouvees dans ce fichier")
	bits := flag.Int("bits", 0, "lire N bits a chaque position exacte et publier leur distribution")
	flag.Parse()
	if *in == "" || *dir == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_comptruth -in <capture.bin> -film <dir> [-comp 22] [-ti 35]")
		os.Exit(2)
	}

	hits, err := readHits(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var want []hit
	for _, h := range hits {
		if int(h.CompIndex) != *comp {
			continue
		}
		if *ti >= 0 && int(h.TypeIndex) != *ti {
			continue
		}
		if !usable(h.Sig) {
			continue
		}
		want = append(want, h)
		if len(want) >= *maxN {
			break
		}
	}
	fmt.Printf("VERITE DE POSITION — composant i%d, archetype %d\n\n", *comp, *ti)
	fmt.Printf("  %d lectures dans la capture, %d retenues (signature exploitable)\n",
		countComp(hits, *comp, *ti), len(want))
	if len(want) == 0 {
		fmt.Fprintln(os.Stderr, "aucune lecture exploitable — signatures toutes degenerees ?")
		os.Exit(1)
	}

	// On charge les chunks une fois et on cherche toutes les signatures dedans : le film fait
	// quelques dizaines de Mo, le relire par signature couterait cent fois plus cher.
	nc := filmdec.CountFilmChunks(*dir)
	var chunks [][]byte
	for c := 1; c <= nc; c++ {
		b, err := filmdec.ReadFilmChunk(*dir, c)
		if err != nil {
			chunks = append(chunks, nil)
			continue
		}
		chunks = append(chunks, b)
	}

	var res []found
	ambig, miss := 0, 0
	for _, h := range want {
		hitsFound := 0
		var first found
		for ci, ch := range chunks {
			if ch == nil {
				continue
			}
			base := 0
			for {
				k := bytes.Index(ch[base:], h.Sig[:])
				if k < 0 {
					break
				}
				if hitsFound == 0 {
					first = found{Chunk: ci + 1, Off: base + k, H: h}
				}
				hitsFound++
				base += k + 1
				if hitsFound > 1 {
					break
				}
			}
			if hitsFound > 1 {
				break
			}
		}
		switch {
		case hitsFound == 0:
			miss++
		case hitsFound > 1:
			ambig++ // position non unique : on ne l'inscrit PAS dans la reference
		default:
			res = append(res, first)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].Chunk != res[j].Chunk {
			return res[i].Chunk < res[j].Chunk
		}
		return res[i].Off < res[j].Off
	})

	fmt.Printf("  %d localisees sans ambiguite · %d introuvables · %d ambigues (ecartees)\n\n",
		len(res), miss, ambig)
	if miss > 0 {
		fmt.Printf("  NOTE : %d signatures absentes du film. Cause attendue si la capture vient\n", miss)
		fmt.Printf("  d'un AUTRE film que celui fourni — verifier avec cmd/tmp_filmmatch.\n\n")
	}

	byChunk := map[int]int{}
	for _, r := range res {
		byChunk[r.Chunk]++
	}
	ks := make([]int, 0, len(byChunk))
	for k := range byChunk {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	fmt.Println("  repartition par chunk —")
	for _, k := range ks {
		fmt.Printf("    chunk %02d : %d lectures\n", k, byChunk[k])
	}
	fmt.Println()
	fmt.Println("  echantillon (chunk, offset octet, offset bit, entite, curseur moteur) —")
	for i, r := range res {
		if i >= 12 {
			fmt.Printf("    … %d autres\n", len(res)-12)
			break
		}
		fmt.Printf("    %02d  %8d  %10d  0x%08X  %d  %s\n",
			r.Chunk, r.Off, r.Off*8, r.H.EID, r.H.BitCursor, hex.EncodeToString(r.H.Sig[:6]))
	}

	if *bits > 0 {
		readValues(*dir, res, *bits, *comp)
	}

	if *out != "" {
		var b []byte
		b = append(b, []byte(fmt.Sprintf("# positions de lecture du composant i%d (ti=%d), localisees par signature CE\n", *comp, *ti))...)
		b = append(b, []byte(fmt.Sprintf("# %d positions uniques · %d introuvables · %d ambigues\n", len(res), miss, ambig))...)
		b = append(b, []byte("# chunk\toffset_octet\toffset_bit\tentite\tcurseur_moteur\n")...)
		for _, r := range res {
			b = append(b, []byte(fmt.Sprintf("%d\t%d\t%d\t%d\t%d\n",
				r.Chunk, r.Off, r.Off*8, r.H.EID, r.H.BitCursor))...)
		}
		if err := os.WriteFile(*out, b, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("\n  reference -> %s\n", *out)
	}
}

// readValues lit la VALEUR de chaque lecture localisee, a la position exacte.
//
// L'IDENTITE QUI REND CECI POSSIBLE, etablie le 2026-07-27 sur i22 : le curseur de bits capture
// EST la position absolue dans le payload du paquet, decalage NUL. Balayage de l'amorce sur
// 0..8 : seul +0 produit un parse valide, et il en produit 249 sur 249. Ce n'est donc pas un
// ajustement, c'est une identite — et elle vaut pour TOUT composant, pas seulement i22.
//
//	position_exacte = paquet.Start*8 + curseur_moteur
//
// La signature ne sert donc qu'a APPARIER une lecture capturee a son paquet ; c'est le curseur
// qui donne la position au bit pres.
//
// CE QUE CETTE FONCTION NE FAIT PAS : elle n'interprete pas. Elle publie la distribution des N
// premiers bits, charge a l'appelant de la confronter a la grammaire portee. Un decodeur qui
// interpreterait avant d'avoir regarde la distribution inventerait des champs.
func readValues(dir string, res []found, bits, comp int) {
	nc := filmdec.CountFilmChunks(dir)
	type pk struct{ start, size int }
	packets := make([][]pk, nc+1)
	chunks := make([][]byte, nc+1)
	for c := 1; c <= nc; c++ {
		b, err := filmdec.ReadFilmChunk(dir, c)
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
	hist := map[uint64]int{}
	perBit := make([]int, bits)
	read, noPkt := 0, 0
	for _, r := range res {
		if r.Chunk >= len(chunks) || chunks[r.Chunk] == nil {
			continue
		}
		var f *pk
		for i := range packets[r.Chunk] {
			p := packets[r.Chunk][i]
			if r.Off >= p.start && r.Off < p.start+p.size {
				f = &p
				break
			}
		}
		if f == nil {
			noPkt++
			continue
		}
		pay := chunks[r.Chunk][f.start : f.start+f.size]
		cur := int(r.H.BitCursor)
		if cur < 0 || cur+bits > len(pay)*8 {
			continue
		}
		read++
		hist[filmdec.PeekBits(pay, cur, bits)]++
		for i := 0; i < bits; i++ {
			if filmdec.PeekBits(pay, cur+i, 1) == 1 {
				perBit[i]++
			}
		}
	}
	fmt.Printf("\n  VALEURS de i%d lues aux positions exactes (%d premiers bits)\n\n", comp, bits)
	fmt.Printf("    %d lectures relues · %d hors de tout paquet delta\n", read, noPkt)
	if read == 0 {
		return
	}
	fmt.Printf("    %d valeurs distinctes\n\n", len(hist))

	type kv struct {
		v uint64
		n int
	}
	var rows []kv
	for v, n := range hist {
		rows = append(rows, kv{v, n})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
	fmt.Println("    valeurs les plus frequentes —")
	for i, r := range rows {
		if i >= 14 {
			fmt.Printf("      … %d autres valeurs\n", len(rows)-14)
			break
		}
		fmt.Printf("      0x%0*X  (%d)  %4d fois  %5.1f %%\n",
			(bits+3)/4, r.v, r.v, r.n, 100*float64(r.n)/float64(read))
	}
	// Le taux de bits a 1 par POSITION revele la structure sans rien supposer : un bit toujours
	// nul est un champ inutilise ou un drapeau eteint, un bit a ~50 % porte de l'information.
	fmt.Println("\n    taux de bits a 1, par position (bit 0 = premier lu) —")
	line := "      "
	for i := 0; i < bits; i++ {
		line += fmt.Sprintf("%3.0f ", 100*float64(perBit[i])/float64(read))
		if (i+1)%16 == 0 {
			fmt.Println(line)
			line = "      "
		}
	}
	if strings.TrimSpace(line) != "" {
		fmt.Println(line)
	}
}

func countComp(hits []hit, comp, ti int) int {
	n := 0
	for _, h := range hits {
		if int(h.CompIndex) == comp && (ti < 0 || int(h.TypeIndex) == ti) {
			n++
		}
	}
	return n
}

func usable(s [16]byte) bool {
	seen := map[byte]bool{}
	for _, b := range s {
		seen[b] = true
	}
	return len(seen) >= minDistinct
}

func readHits(path string) ([]hit, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture %s : %w", path, err)
	}
	out := make([]hit, 0, len(raw)/recSize)
	for i := 0; i+recSize <= len(raw); i += recSize {
		b := raw[i : i+recSize]
		var h hit
		h.EID = binary.LittleEndian.Uint32(b[0:])
		h.TypeIndex = binary.LittleEndian.Uint32(b[4:])
		h.CompIndex = binary.LittleEndian.Uint32(b[8:])
		h.Param4 = binary.LittleEndian.Uint32(b[12:])
		h.BitCursor = binary.LittleEndian.Uint32(b[16:])
		copy(h.Sig[:], b[24:40])
		if h.EID == 0 && h.TypeIndex == 0 && h.BitCursor == 0 && h.CompIndex == 0 {
			break
		}
		out = append(out, h)
	}
	return out, nil
}
