// cmd/tmp_pilu — LE PLAYER INDEX SE LIT, il n'a jamais eu besoin d'etre calcule.
//
// CE QUE CE PROGRAMME VERIFIE. Le rejeu 2D resout aujourd'hui le lien « index de joueur du
// film -> identite » par une AFFECTATION DE COUT MINIMAL sur les 8! permutations. C'est un
// choix, pas une lecture, et sa marge est etroite (32 contradictions contre 39).
//
// Or le chantier voisin (`filmdec-killweapon`) a etabli que ce lien SE LIT : le xuid d'un
// joueur est ecrit dans le film en 8 octets petit-boutiste, et les CINQ BITS qui le precedent
// immediatement portent son index. La fonction existe deja dans ce worktree
// (`weaponv3.ResolveXuidToPI`) — elle n'avait simplement jamais ete branchee ici.
//
// LE PIEGE DOCUMENTE PAR LE VOISIN, et la raison pour laquelle ce programme balaye chunk par
// chunk : applique au chunk 0 (type 1, le registre), le resolveur rend 0 pour les huit xuids.
// Sur les chunks de replication, il rend la bijection. Un balayage indifferencie ecrase donc
// le bon resultat avec le mauvais.
//
// LE CONTROLE : la table lue doit reproduire celle que le rejeu calcule aujourd'hui. Si les
// deux concordent, le calcul peut disparaitre au profit de la lecture.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/weaponv3"
)

func main() {
	repo := flag.String("repo", `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`,
		"racine portant data/cache/film_chunks")
	match := flag.String("match", "000d5950", "identifiant du match")
	flag.Parse()
	filmDir := filepath.Join(*repo, "data", "cache", "film_chunks", *match)

	// Le roster vient du FIL DES MORTS : il porte les xuids, et c'est deja une lecture.
	deaths, err := replay.ScanFilmDeaths(filmDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fil des morts:", err)
		os.Exit(1)
	}
	seen := map[uint64]bool{}
	var roster []uint64
	for _, d := range deaths {
		if !seen[d.XUID] {
			seen[d.XUID] = true
			roster = append(roster, d.XUID)
		}
	}
	sort.Slice(roster, func(i, j int) bool { return roster[i] < roster[j] })
	fmt.Printf("=== LECTURE DU PLAYER INDEX — film %s, %d xuids au roster\n\n", *match, len(roster))

	n := filmdec.CountFilmChunks(filmDir)
	// Par chunk : combien de xuids resolus, et la table obtenue. On veut VOIR le piege du
	// chunk de registre plutot que le contourner en silence.
	fmt.Printf("%-8s %-10s %s\n", "chunk", "resolus", "table (xuid -> index)")
	agree := map[uint64]map[int]int{} // xuid -> index -> nb de chunks qui le disent
	for c := 0; c <= n; c++ {
		raw, err := filmdec.ReadFilmChunk(filmDir, c)
		if err != nil {
			continue
		}
		got := weaponv3.ResolveXuidToPI(roster, raw)
		if len(got) == 0 {
			continue
		}
		var parts []string
		for _, x := range roster {
			if pi, ok := got[x]; ok {
				parts = append(parts, fmt.Sprintf("%d:%d", x%1000000, pi))
				if agree[x] == nil {
					agree[x] = map[int]int{}
				}
				agree[x][pi]++
			}
		}
		fmt.Printf("%-8d %-10d %v\n", c, len(got), parts)
	}

	fmt.Printf("\n--- TABLE RETENUE (index le plus souvent lu par xuid) ---\n")
	final := map[uint64]int{}
	for _, x := range roster {
		best, bestN, total, distinct := -1, 0, 0, 0
		for pi, k := range agree[x] {
			distinct++
			total += k
			if k > bestN {
				bestN, best = k, pi
			}
		}
		final[x] = best
		fmt.Printf("  xuid %d -> index %-3d  (%d lectures sur %d, %d valeurs distinctes)\n",
			x, best, bestN, total, distinct)
	}

	// INJECTIVITE : deux joueurs ne peuvent pas partager un index.
	byIdx := map[int][]uint64{}
	for x, pi := range final {
		byIdx[pi] = append(byIdx[pi], x)
	}
	coll := 0
	for _, xs := range byIdx {
		if len(xs) > 1 {
			coll++
		}
	}
	fmt.Printf("\nindex distincts : %d sur %d joueurs ; collisions : %d\n", len(byIdx), len(roster), coll)
}
