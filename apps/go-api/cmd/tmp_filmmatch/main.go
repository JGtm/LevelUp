// tmp_filmmatch — IDENTIFIER le film rejoue, et ANCRER la capture CE dedans, par la seule
// signature d'octets bruts. Aucune base de donnees, aucun manifeste.
//
// LE PRINCIPE. Le crochet CE capture, a chaque composant deserialise, 16 octets lus depuis
// [bitreader+0x40] — le pointeur d'octet courant DANS LE CHUNK DECOMPRESSE du film. Ces 16
// octets existent donc a l'identique dans le film hors ligne, une fois le chunk inflate.
// Chercher la signature dans les 948 films du cache repond a deux questions d'un coup :
//
//	QUEL film a ete rejoue          -> celui qui contient les signatures
//	OU se trouve chaque composant   -> l'offset exact, en octets, dans le chunk
//
// Le second point est le vrai gain : il transforme une comparaison de DISTRIBUTIONS (tout ce
// qu'on pouvait faire avec une capture d'un autre film) en comparaison RECORD CONTRE RECORD.
//
// POURQUOI 16 OCTETS SUFFISENT A IDENTIFIER. Une fenetre de 16 octets porte 128 bits. La
// probabilite qu'une signature donnee apparaisse par hasard dans un film de 70 Mo est de
// 7e7 / 2^128, soit ~2e-31. Le nombre de faux positifs attendus sur 948 films est donc nul a
// toute echelle utile. On exige quand meme PLUSIEURS signatures concordantes par film : une
// seule correspondance sur une signature degeneree (16 octets a zero, ou une repetition)
// ne vaut rien, et ce sont precisement les signatures que le filtre ci-dessous ecarte.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const recSize = 40

// minDistinctBytes ecarte les signatures degenerees. Une fenetre de 16 octets dont trop peu
// de valeurs sont distinctes (zeros, 0xFF, motif court repete) apparait en beaucoup d'endroits :
// elle n'identifie rien et pollue le comptage.
//
// MESURE DU 2026-07-27 : a 8, l'appariement echoue (14/40 sur le meilleur film, 13/40 sur le
// second, etale sur des dizaines) parce que des fenetres de bourrage du type
// `...fffffffe07fffffffc17` et `0000000000000700...` passent le filtre et se retrouvent dans
// tous les films. Le seuil est devenu un drapeau pour pouvoir mesurer sa sensibilite : un
// appariement qui tient doit RESISTER au durcissement, pas en dependre.
var minDistinctBytes = 8

func main() {
	in := flag.String("in", "", "dump binaire de la capture CE")
	cache := flag.String("cache", `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\cache\film_chunks`, "racine du cache de films")
	nSig := flag.Int("sigs", 60, "nombre de signatures a tester")
	only := flag.String("only", "", "ne tester que ce film (prefixe d'identifiant)")
	stride := flag.Int("stride", 997, "pas d'echantillonnage des signatures (premier, pour eviter la periodicite)")
	minDist := flag.Int("mindist", 8, "octets distincts minimum dans une signature (durcit le filtre d'entropie)")
	flag.Parse()
	minDistinctBytes = *minDist
	if *in == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_filmmatch -in <dump.bin> [-cache DIR] [-sigs N] [-only ID]")
		os.Exit(2)
	}

	sigs, err := loadSignatures(*in, *nSig, *stride)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("APPARIEMENT FILM — %d signatures retenues (16 o, non degenerees)\n\n", len(sigs))
	if len(sigs) == 0 {
		fmt.Fprintln(os.Stderr, "aucune signature exploitable — la capture a-t-elle bien lu [rdi+40] ?")
		os.Exit(1)
	}
	for i := 0; i < 3 && i < len(sigs); i++ {
		fmt.Printf("  exemple : %s\n", hex.EncodeToString(sigs[i][:]))
	}
	fmt.Println()

	films, err := listFilms(*cache, *only)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("  %d films a tester dans %s\n\n", len(films), *cache)

	type result struct {
		id       string
		hits     int
		chunkOf  map[int]int
		firstOff int
	}
	var best []result
	for _, f := range films {
		r := result{id: filepath.Base(f), chunkOf: map[int]int{}, firstOff: -1}
		n := filmdec.CountFilmChunks(f)
		for c := 1; c <= n; c++ {
			chunk, err := filmdec.ReadFilmChunk(f, c)
			if err != nil {
				continue
			}
			for _, s := range sigs {
				if idx := bytes.Index(chunk, s[:]); idx >= 0 {
					r.hits++
					r.chunkOf[c]++
					if r.firstOff < 0 {
						r.firstOff = idx
					}
				}
			}
		}
		if r.hits > 0 {
			best = append(best, r)
		}
	}
	if len(best) == 0 {
		fmt.Println("  AUCUN film du cache ne contient ces signatures.")
		fmt.Println("  -> le film rejoue n'est pas (encore) telecharge. La capture reste exploitable")
		fmt.Println("     en distribution, mais pas en comparaison record contre record.")
		return
	}
	sort.Slice(best, func(i, j int) bool { return best[i].hits > best[j].hits })
	fmt.Printf("  %-12s  %8s  %8s  %s\n", "film", "sigs", "chunks", "1er offset")
	for i, r := range best {
		if i >= 10 {
			break
		}
		fmt.Printf("  %-12s  %5d/%-3d  %8d  %d\n",
			r.id, r.hits, len(sigs), len(r.chunkOf), r.firstOff)
	}
	fmt.Println()
	top := best[0]
	fmt.Printf("  FILM IDENTIFIE : %s (%d signatures sur %d, reparties sur %d chunks)\n",
		top.id, top.hits, len(sigs), len(top.chunkOf))
	if len(best) > 1 && best[1].hits*2 > top.hits {
		fmt.Printf("  ATTENTION : le second (%s, %d) est proche — identification NON tranchee.\n",
			best[1].id, best[1].hits)
	}
}

// loadSignatures echantillonne des signatures non degenerees dans la capture. Le pas est un
// nombre premier pour ne pas retomber periodiquement sur le meme composant.
func loadSignatures(path string, want, stride int) ([][16]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture %s : %w", path, err)
	}
	total := len(raw) / recSize
	var out [][16]byte
	for i := 0; i < total && len(out) < want; i += stride {
		b := raw[i*recSize : (i+1)*recSize]
		if binary.LittleEndian.Uint32(b[0:]) == 0 && binary.LittleEndian.Uint32(b[16:]) == 0 {
			continue // enregistrement non ecrit
		}
		var s [16]byte
		copy(s[:], b[24:40])
		if !usable(s) {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

// usable ecarte les fenetres trop pauvres pour identifier quoi que ce soit.
func usable(s [16]byte) bool {
	seen := map[byte]bool{}
	for _, b := range s {
		seen[b] = true
	}
	return len(seen) >= minDistinctBytes
}

// listFilms enumere les dossiers de films du cache.
func listFilms(root, only string) ([]string, error) {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("lecture du cache %s : %w", root, err)
	}
	var out []string
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		if only != "" && !bytes.HasPrefix([]byte(e.Name()), []byte(only)) {
			continue
		}
		out = append(out, filepath.Join(root, e.Name()))
	}
	return out, nil
}
