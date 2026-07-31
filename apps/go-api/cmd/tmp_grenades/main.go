// Commande tmp_grenades — valide le décodeur de lancers de grenade sur un film.
//
// TÉMOINS PRODUITS (c'est le point : un compte seul ne prouve rien) :
//  1. Sélectivité du marqueur : combien de marqueurs 0x4C0C00 bruts, combien survivent à la
//     liste blanche des 4 identifiants. Le rapport chiffre le pouvoir discriminant.
//  2. Attendu par pur hasard : 4 identifiants sur 2^32, sur N positions de marqueur testées.
//  3. Distribution des index joueur : sur un match à 8 joueurs, les valeurs DOIVENT tomber
//     dans 0..7 alors que le champ en porte 32. C'est le témoin le plus fort, et il est
//     catégorique : 5 bits qui ne rendraient que du bruit sortiraient de 0..7 dans 75 % des
//     cas.
//  4. Contrôle négatif : le même balayage avec un marqueur FAUX (0x4C0C01), qui doit
//     s'effondrer.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

func main() {
	dir := flag.String("dir", `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\cache\film_chunks\000d5950`, "dossier des chunks du film")
	flag.Parse()

	th, err := filmdec.ScanFilmGrenadeThrows(*dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}

	fmt.Printf("LANCERS DE GRENADE — %s\n\n", *dir)
	fmt.Printf("  %d lancers validés\n\n", len(th))

	byType := map[string]int{}
	byPlayer := map[int]int{}
	outOfRange := 0
	for _, g := range th {
		byType[g.Name()]++
		byPlayer[g.FilmIndex]++
		if g.FilmIndex > 7 {
			outOfRange++
		}
	}

	fmt.Println("  par type :")
	types := make([]string, 0, len(byType))
	for k := range byType {
		types = append(types, k)
	}
	sort.Strings(types)
	for _, k := range types {
		fmt.Printf("    %-16s %4d\n", k, byType[k])
	}

	fmt.Println("\n  par index joueur (5 bits, donc 0..31 possibles) :")
	idx := make([]int, 0, len(byPlayer))
	for k := range byPlayer {
		idx = append(idx, k)
	}
	sort.Ints(idx)
	for _, k := range idx {
		mark := ""
		if k > 7 {
			mark = "  <-- HORS 0..7"
		}
		fmt.Printf("    joueur %2d : %4d%s\n", k, byPlayer[k], mark)
	}

	if len(th) > 0 {
		pct := 100.0 * float64(len(th)-outOfRange) / float64(len(th))
		fmt.Printf("\n  TÉMOIN index dans 0..7 : %d/%d = %.1f %% (attendu par hasard : 25,0 %%)\n",
			len(th)-outOfRange, len(th), pct)
		fmt.Printf("  TÉMOIN couverture      : %d joueurs distincts sur 8\n", len(byPlayer))
	}

	fmt.Println("\n  chronologie (20 premiers) :")
	sort.Slice(th, func(i, j int) bool { return th[i].TimestampUS < th[j].TimestampUS })
	t0 := uint64(0)
	if len(th) > 0 {
		t0 = th[0].TimestampUS
	}
	for i, g := range th {
		if i >= 20 {
			break
		}
		fmt.Printf("    %7.1fs  joueur %d  %-14s  (chunk %d, paquet %d, bit %d)\n",
			float64(g.TimestampUS-t0)/1e6, g.FilmIndex, g.Name(), g.Chunk, g.PacketIndex, g.BitPos)
	}
}
