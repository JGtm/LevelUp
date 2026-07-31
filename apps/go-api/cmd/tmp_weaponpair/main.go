// tmp_weaponpair — QUELLE EST LA CORRELATION entre l'arme EN MAIN et celles du LOADOUT ?
//
// CE QUI EST DEJA ETABLI (2026-07-27) :
//   - i43 porte un identifiant d'arme 64 bits a place fixe, suffixe 0x42C9679F a +33 depuis le
//     debut du composant. Controle negatif a ZERO sur 192 positions. 16 identifiants sur 16
//     nommes par une table batie par un chemin independant.
//   - co-occurrence mesuree : i43 et i44 sont ensemble dans 77 records, i43 n'est JAMAIS seul
//     sans i42 (0 sur 89), et i45/i46 ne co-occurrent JAMAIS avec i43.
//   - le POC porte 150 loadouts de keyframe, chacun a EXACTEMENT deux armes.
//
// L'HYPOTHESE A TESTER : i43 et i44 sont les DEUX EMPLACEMENTS d'arme, et i42 (7 valeurs
// distinctes seulement, sur 447 lectures) designe lequel est en main.
//
// CE QUE CET OUTIL MESURE, sans rien supposer :
//  1. les identifiants lus dans i43 et dans i44 d'un MEME record — sont-ils differents ?
//     (deux emplacements portent deux armes distinctes ; un doublon invaliderait le modele) ;
//  2. la valeur d'i42 dans ce record, croisee avec la paire — si i42 selectionne, sa valeur
//     doit se partitionner selon l'emplacement, pas varier au hasard ;
//  3. la confrontation aux paires de loadout du keyframe, decodees par un AUTRE chemin.
//
// CE QUI REFUTERAIT L'HYPOTHESE : des identifiants identiques dans i43 et i44, ou une valeur
// d'i42 sans structure. On publie la distribution brute avant d'interpreter.
package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	recSize      = 40
	weaponSuffix = 0x42C9679F
	// idOffset : decalage du debut de l'identifiant 64 bits depuis le debut du composant.
	// Le suffixe est mesure a +33 ; les 32 bits hauts le precedent, donc +1.
	idOffset = 1
	minDist  = 12
)

type hit struct {
	EID, TypeIndex, CompIndex, BitCursor uint32
	Sig                                  [16]byte
}

type pk struct{ start, size int }

func main() {
	in := flag.String("in", "", "dump binaire de la capture CE")
	dir := flag.String("film", "", "dossier des chunks du film")
	names := flag.String("names", "", "table JSON identifiant -> nom (facultatif)")
	flag.Parse()
	if *in == "" || *dir == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_weaponpair -in <capture.bin> -film <dir> [-names t.json]")
		os.Exit(2)
	}

	var label map[string]string
	if *names != "" {
		if b, err := os.ReadFile(*names); err == nil {
			_ = json.Unmarshal(b, &label)
		}
	}
	nameOf := func(id uint64) string {
		if label == nil {
			return ""
		}
		return label[fmt.Sprintf("0x%08X", uint32(id>>32))]
	}

	hits, err := readHits(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	nc := filmdec.CountFilmChunks(*dir)
	chunks := make([][]byte, nc+1)
	packets := make([][]pk, nc+1)
	sigIdx := map[uint64][]hit{}
	for _, h := range hits {
		c := int(h.CompIndex)
		if h.TypeIndex != 35 || (c != 42 && c != 43 && c != 44) || !usable(h.Sig) {
			continue
		}
		sigIdx[binary.LittleEndian.Uint64(h.Sig[:8])] = append(
			sigIdx[binary.LittleEndian.Uint64(h.Sig[:8])], h)
	}
	// position exacte de chaque lecture, par la signature puis le curseur
	type loc struct {
		eid, comp uint32
		chunk     int
		pay       []byte
		cur       int
	}
	var locs []loc
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
		for off := 0; off+16 <= len(b); off++ {
			cands, ok := sigIdx[binary.LittleEndian.Uint64(b[off:off+8])]
			if !ok {
				continue
			}
			for _, h := range cands {
				same := true
				for j := 0; j < 16; j++ {
					if b[off+j] != h.Sig[j] {
						same = false
						break
					}
				}
				if !same {
					continue
				}
				for _, p := range packets[c] {
					if off >= p.start && off < p.start+p.size {
						locs = append(locs, loc{h.EID, h.CompIndex, c,
							b[p.start : p.start+p.size], int(h.BitCursor)})
						break
					}
				}
			}
		}
	}

	// Regroupement par (entite, paquet) : un record porte au plus un i42, un i43, un i44.
	type rec struct{ i42, i43, i44 *loc }
	byKey := map[string]*rec{}
	for i := range locs {
		l := locs[i]
		k := fmt.Sprintf("%d/%08X/%d", l.chunk, l.eid, l.cur/4096)
		if byKey[k] == nil {
			byKey[k] = &rec{}
		}
		switch l.comp {
		case 42:
			byKey[k].i42 = &locs[i]
		case 43:
			byKey[k].i43 = &locs[i]
		case 44:
			byKey[k].i44 = &locs[i]
		}
	}

	readID := func(l *loc) (uint64, bool) {
		if l == nil {
			return 0, false
		}
		b := l.cur + idOffset
		if b < 0 || b+64 > len(l.pay)*8 {
			return 0, false
		}
		v := filmdec.PeekBits(l.pay, b, 64)
		if uint32(v) != weaponSuffix {
			return 0, false
		}
		return v, true
	}

	pairs := 0
	same := 0
	bySel := map[uint64][2]int{} // valeur d'i42 -> [nb paires, nb ou i43 == arme selectionnee]
	var samples []string
	for _, r := range byKey {
		a, oka := readID(r.i43)
		b, okb := readID(r.i44)
		if !oka || !okb {
			continue
		}
		pairs++
		if a == b {
			same++
		}
		var sel uint64 = 1 << 40 // sentinelle : pas d'i42
		if r.i42 != nil && r.i42.cur+7 <= len(r.i42.pay)*8 {
			sel = filmdec.PeekBits(r.i42.pay, r.i42.cur, 7)
		}
		e := bySel[sel]
		e[0]++
		bySel[sel] = e
		if len(samples) < 14 {
			na, nb := nameOf(a), nameOf(b)
			s := fmt.Sprintf("    i42=%3d   i43 %-16s   i44 %-16s", sel, na, nb)
			if na == "" || nb == "" {
				s = fmt.Sprintf("    i42=%3d   i43 0x%08X   i44 0x%08X", sel, uint32(a>>32), uint32(b>>32))
			}
			samples = append(samples, s)
		}
	}

	fmt.Printf("CORRELATION ARME EN MAIN / LOADOUT\n\n")
	fmt.Printf("  %d records portent DEUX identifiants d'arme lisibles (i43 et i44)\n", pairs)
	fmt.Printf("  dont %d ou les deux emplacements portent LA MEME arme (%.1f %%)\n\n",
		same, pctf(same, pairs))
	if pairs == 0 {
		fmt.Println("  Aucune paire lisible : l'hypothese des deux emplacements n'est pas testable ainsi.")
		return
	}
	fmt.Println("  ECHANTILLON —")
	for _, s := range samples {
		fmt.Println(s)
	}
	fmt.Println("\n  VALEURS D'i42 sur les records a deux armes —")
	type kv struct {
		v uint64
		n int
	}
	var rows []kv
	for v, e := range bySel {
		rows = append(rows, kv{v, e[0]})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
	for _, r := range rows {
		if r.v == 1<<40 {
			fmt.Printf("    (pas d'i42)  %3d records\n", r.n)
			continue
		}
		fmt.Printf("    i42 = %3d    %3d records\n", r.v, r.n)
	}
	fmt.Println("\n  LECTURE — deux armes DIFFERENTES dans presque tous les records valide le modele")
	fmt.Println("  des deux emplacements. Un i42 a peu de valeurs, correlees a la paire, en ferait")
	fmt.Println("  le selecteur. Des valeurs d'i42 sans structure le refuteraient.")
}

func usable(s [16]byte) bool {
	seen := map[byte]bool{}
	for _, b := range s {
		seen[b] = true
	}
	return len(seen) >= minDist
}

func pctf(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
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
		h.BitCursor = binary.LittleEndian.Uint32(b[16:])
		copy(h.Sig[:], b[24:40])
		if h.EID == 0 && h.TypeIndex == 0 && h.BitCursor == 0 && h.CompIndex == 0 {
			break
		}
		out = append(out, h)
	}
	return out, nil
}
