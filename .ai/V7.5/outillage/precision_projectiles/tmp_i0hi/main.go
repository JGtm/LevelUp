// Commande de recherche (jetable, gitignoree) : la branche OPAQUE d'`object-position-component`.
//
// CE QUE CE PROGRAMME MESURE. Le port de production (`filmdec/traverse.go`) lit i0 des objets
// du monde en deux branches : porte = 0 -> position absolue aux largeurs de la CARTE (13/13/14
// sur Cliffhanger), decodee et validee ; porte = 1 -> `br.ReadBits(59)`, un TOTAL mesure dont
// rien ne sort. `filmdec/projectiles.go` REJETTE ces records (`PeekBits(pay, at, 3) != 0`).
//
// DEUX QUESTIONS, DANS CET ORDRE :
//
//	1. QUELLE PART des records de projectile part a la poubelle ? (mode -part)
//	2. Ou sont les trois axes dans ces 59 bits ? (mode -reg)
//
// LA METHODE DU (2), ET SA GARDE-FOU. On ne balaye PAS des frontieres de bits pour y lire une
// distribution plausible — le chantier s'interdit ce geste, il fabrique des reponses. On fait
// une REGRESSION : pour chaque champ candidat (offset, largeur), on confronte l'entier brut lu
// a une position VRAIE, interpolee depuis les records voisins de la MEME vie decodes sur la
// branche BASSE (celle qui est validee). Un decoupage juste donne une relation AFFINE
// (R2 ~ 1) ; un decoupage faux ne donne rien. Trois exigences avant de conclure :
//
//	CONTROLE POSITIF : la meme mecanique, appliquee a la branche BASSE, doit retrouver ses
//	                   propres champs (offsets 3/16/29, largeurs 13/13/14) a R2 = 1.
//	NULLE            : la distribution complete des R2 sur tous les candidats est publiee. Le
//	                   gagnant doit etre une valeur aberrante, pas le maximum d'un bruit.
//	HORS ECHANTILLON : le meme decoupage doit tenir sur une AUTRE carte, avec son propre AABB.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	projectileTI = 41
	// lowPosBits est la longueur du record i0 sur la branche basse : 3 de porte + 40 d'axes
	// + 2 de queue. C'est `projPosBits()` de filmdec/projectiles.go.
	lowPosBits = 45
	// hiPosBits est la longueur mesuree de la branche haute : 1 de porte + 59.
	hiPosBits = 60
	// lifeGapUS reprend `projectileGapUS` : au-dela, deux echantillons du meme (slot, gen)
	// sont deux vies distinctes.
	lifeGapUS = 250_000
)

// sample est un record i0 de projectile localise dans le flux.
type sample struct {
	slot, gen   uint32
	timestampUS uint64
	hi          bool       // porte = 1 : la branche opaque
	pos         [3]float32 // coordonnee MONDE, renseignee seulement si !hi
	// norm est la meme position ramenee dans [0,1] par l'AABB de SA carte. Elle sert a
	// mutualiser plusieurs cartes : si la branche opaque encode aux largeurs de la carte,
	// c'est la coordonnee normalisee qui est comparable d'une carte a l'autre ; si elle
	// encode dans une plage FIXE, c'est la coordonnee monde. On teste les deux.
	norm [3]float32
	// bits porte les 64 premiers bits du composant i0, poids fort en tete du flux.
	bits uint64
}

func main() {
	films := flag.String("films", "", "racine du cache de films (LECTURE SEULE)")
	only := flag.String("only", "", "liste de matchs separes par des virgules (defaut : les -limit premiers)")
	limit := flag.Int("limit", 4, "nombre maximum de films")
	catalog := flag.String("catalogue", "", "chemin de map_quant_bounds.json")
	mapName := flag.String("carte", "cliffhanger", "nom de carte pour les bornes")
	mapsCSV := flag.String("cartes", "", "CSV matchID,carte : mutualise plusieurs cartes")
	part := flag.Bool("part", false, "compter la part de la branche opaque")
	reg := flag.Bool("reg", false, "regression position vraie contre champ brut")
	control := flag.Bool("controle", false, "controle positif : la regression sur la branche BASSE")
	flag.Parse()

	if *films == "" || *catalog == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_i0hi -films <dir> -catalogue <json> [-carte N] [-part|-reg|-controle]")
		os.Exit(2)
	}
	cat, err := filmdec.LoadMapQuantCatalog(*catalog)
	if err != nil {
		fmt.Fprintln(os.Stderr, "catalogue:", err)
		os.Exit(1)
	}
	entry, err := cat.Lookup(*mapName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "carte:", err)
		os.Exit(1)
	}
	wr := entry.Range()

	// Deux regimes. Sans -cartes : une seule carte, ses bornes pour tous les films.
	// Avec -cartes : chaque film prend les bornes de SA carte, ce qui mutualise le corpus
	// entier — c'est ce qui fait passer l'echantillon de 127 a quelques milliers.
	var jobs []filmJob
	if *mapsCSV != "" {
		jobs, err = jobsFromCatalog(*films, *mapsCSV, cat, *limit)
	} else {
		var dirs []string
		dirs, err = filmList(*films, *only, *limit)
		for _, d := range dirs {
			jobs = append(jobs, filmJob{dir: d, wr: wr, mapName: *mapName})
		}
		fmt.Printf("carte %s (%s) : AABB %v .. %v, largeurs %v\n",
			*mapName, entry.Module, entry.Min, entry.Max, entry.AxisWidths)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "films:", err)
		os.Exit(1)
	}

	var all []sample
	var tot, hi int
	byMap := map[string][2]int{}
	for _, j := range jobs {
		w := j.wr
		s, t, h := scanFilm(j.dir, &w)
		// ELAGAGE PAR FILM. Seules les vies qui melangent les deux branches servent a la
		// regression : garder les autres ferait croitre la memoire lineairement avec le
		// corpus, et le corpus film est une bombe RAM documentee.
		if *mapsCSV != "" {
			s = keepMixedLives(s)
		}
		all = append(all, s...)
		tot += t
		hi += h
		c := byMap[j.mapName]
		byMap[j.mapName] = [2]int{c[0] + t, c[1] + h}
		if *mapsCSV == "" {
			fmt.Printf("  %-10s records ti=41 %5d   dont porte=1 %5d (%.1f %%)\n",
				filepath.Base(j.dir), t, h, pct(h, t))
		}
	}
	if *mapsCSV != "" {
		names := make([]string, 0, len(byMap))
		for n := range byMap {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			c := byMap[n]
			fmt.Printf("  %-16s records ti=41 %7d   dont porte=1 %6d (%.1f %%)\n", n, c[0], c[1], pct(c[1], c[0]))
		}
	}
	fmt.Printf("TOTAL %d films : %d records, %d sur la branche opaque (%.1f %%)\n\n", len(jobs), tot, hi, pct(hi, tot))

	if *part {
		reportLives(all)
	}
	if *control {
		runRegression(all, &wr, false)
	}
	if *reg {
		runRegression(all, &wr, true)
	}
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

func filmList(root, only string, limit int) ([]string, error) {
	if only != "" {
		var out []string
		for _, id := range splitComma(only) {
			out = append(out, filepath.Join(root, id))
		}
		return out, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		out = append(out, filepath.Join(root, e.Name()))
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func splitComma(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// reportLives publie la structure des vies : combien melangent les deux branches, combien
// n'ont que l'opaque, et combien de records opaques ont un encadrement utilisable.
func reportLives(all []sample) {
	lives := groupLives(all)
	var mixed, pure, bracketable int
	for _, l := range lives {
		var nHi, nLo int
		for _, s := range l {
			if s.hi {
				nHi++
			} else {
				nLo++
			}
		}
		switch {
		case nHi > 0 && nLo > 0:
			mixed++
			bracketable += countBracketed(l)
		case nHi > 0:
			pure++
		}
	}
	fmt.Printf("vies : %d au total, %d melangent les deux branches, %d n'ont que l'opaque\n",
		len(lives), mixed, pure)
	fmt.Printf("records opaques ENCADRES par deux records basse branche : %d\n\n", bracketable)
}

// groupLives regroupe par (slot, gen) puis coupe sur un trou temporel.
func groupLives(all []sample) [][]sample {
	byKey := map[[2]uint32][]sample{}
	for _, s := range all {
		k := [2]uint32{s.slot, s.gen}
		byKey[k] = append(byKey[k], s)
	}
	var out [][]sample
	keys := make([][2]uint32, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		return keys[i][1] < keys[j][1]
	})
	for _, k := range keys {
		pts := byKey[k]
		sort.Slice(pts, func(i, j int) bool { return pts[i].timestampUS < pts[j].timestampUS })
		start := 0
		for i := 1; i <= len(pts); i++ {
			if i < len(pts) && pts[i].timestampUS-pts[i-1].timestampUS <= lifeGapUS {
				continue
			}
			if i-start >= 2 {
				seg := make([]sample, i-start)
				copy(seg, pts[start:i])
				out = append(out, seg)
			}
			start = i
		}
	}
	return out
}

// keepMixedLives ne garde que les echantillons des vies qui portent au moins un record de
// CHAQUE branche : ce sont les seules ou un record opaque peut etre encadre par deux positions
// vraies. Les autres ne serviraient a rien et coutent de la memoire.
func keepMixedLives(all []sample) []sample {
	var out []sample
	for _, l := range groupLives(all) {
		var nHi, nLo int
		for _, s := range l {
			if s.hi {
				nHi++
			} else {
				nLo++
			}
		}
		if nHi > 0 && nLo > 0 {
			out = append(out, l...)
		}
	}
	return out
}

func countBracketed(l []sample) int {
	n := 0
	for i, s := range l {
		if !s.hi {
			continue
		}
		if _, _, ok := bracket(l, i); ok {
			n++
		}
	}
	return n
}

// bracket rend la position vraie interpolee a l'instant du record i — en coordonnee MONDE et
// en coordonnee normalisee — si deux records de la branche BASSE l'encadrent dans la meme vie.
func bracket(l []sample, i int) ([3]float32, [3]float32, bool) {
	var before, after = -1, -1
	for j := i - 1; j >= 0; j-- {
		if !l[j].hi {
			before = j
			break
		}
	}
	for j := i + 1; j < len(l); j++ {
		if !l[j].hi {
			after = j
			break
		}
	}
	if before < 0 || after < 0 {
		return [3]float32{}, [3]float32{}, false
	}
	t0, t1 := l[before].timestampUS, l[after].timestampUS
	if t1 <= t0 {
		return [3]float32{}, [3]float32{}, false
	}
	f := float32(l[i].timestampUS-t0) / float32(t1-t0)
	var v, nv [3]float32
	for a := 0; a < 3; a++ {
		v[a] = l[before].pos[a] + f*(l[after].pos[a]-l[before].pos[a])
		nv[a] = l[before].norm[a] + f*(l[after].norm[a]-l[before].norm[a])
	}
	return v, nv, true
}
