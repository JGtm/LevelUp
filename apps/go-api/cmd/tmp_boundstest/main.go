// tmp_boundstest — TEST DÉCISIF de la loi de quantification.
//
// Source A (films, mesure) : DetectI0Layout lit les 3 frontières de champ d'i0 par
// profil de bascule, sans aucun a priori de largeur.
// Source B (modules de carte, totalement disjointe) : `world bounds x/y/z` du tag sbsp
// principal, puis W = min(26, ceilLog2(ceil(60*extent))).
//
// Le test compare B à A carte par carte. Le TÉMOIN croise les cartes : appliquer les
// bornes d'une carte aux largeurs d'une autre doit échouer.
//
// Usage : CGO_ENABLED=1 go run ./cmd/tmp_boundstest
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/himap"
)

const (
	levelsDir = `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy/ds/levels/multi`
	filmCache = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
	mapNames  = `C:/Users/GUILLA~1/AppData/Local/Temp/claude/C--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration--claude-worktrees-filmdec-continuation/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad/mapnames.csv`
	maxFilms  = 4
)

// aprioriModule : correspondance nom affiché -> dossier de module, établie AVANT toute
// mesure et sur le seul nom (aucune carte n'est appariée par son résultat).
var aprioriModule = map[string]string{
	"Catalyst":      "catalyst",
	"Chasm":         "chasm",
	"Aquarius":      "ctf_aquarius",
	"Bazaar":        "ctf_bazaar",
	"Illusion":      "ctf_illusion",
	"Streets":       "sgh_streets",
	"Behemoth":      "va_behemoth",
	"Forest":        "forest",
	"Forbidden":     "ctf_forbidden",
	"Breaker":       "ctf_breaker",
	"Launch Site":   "va_launchsite",
	"Highpower":     "btb_highpower",
	"Fragmentation": "btb_fragmentation",
}

// cartes mesurées mais dont le module n'est PAS déductible du nom : elles servent de
// prédiction, pas de confirmation.
var unresolved = []string{"Cliffhanger", "Live Fire", "Recharge", "Prism"}

func main() {
	mods := readModules()
	films := readFilms()

	fmt.Println("=== A. modules : BSP principal, bornes monde et largeur prédite ===")
	var names []string
	for n := range mods {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		b := mods[n]
		w := b.AxisWidths()
		fmt.Printf("%-22s x[%10.4f,%10.4f] y[%10.4f,%10.4f] z[%10.4f,%10.4f]  E=%9.3f/%9.3f/%9.3f  W=%2d/%2d/%2d\n",
			n, b.Min[0], b.Max[0], b.Min[1], b.Max[1], b.Min[2], b.Max[2],
			b.Extent(0), b.Extent(1), b.Extent(2), w[0], w[1], w[2])
	}

	fmt.Println("\n=== B. mesure dans les films (profil de bascule) et confrontation ===")
	type res struct {
		mapName, module string
		measured        [3]uint
		bounds          []int
		predicted       [3]int
		gate            int
		nFilms          int
	}
	var results []res
	var display []string
	for d := range aprioriModule {
		display = append(display, d)
	}
	display = append(display, unresolved...)
	sort.Strings(display)

	for _, disp := range display {
		ids := films[disp]
		if len(ids) == 0 {
			fmt.Printf("%-16s aucun film en cache\n", disp)
			continue
		}
		var lays []filmdec.I0Layout
		var bnds [][]int
		for _, id := range ids {
			if len(lays) >= maxFilms {
				break
			}
			lay, rep, err := filmdec.DetectI0Layout(filepath.Join(filmCache, id))
			if err != nil {
				continue
			}
			lays = append(lays, lay)
			bnds = append(bnds, rep.Boundaries[:3])
		}
		if len(lays) == 0 {
			fmt.Printf("%-16s aucun film exploitable (%d essais)\n", disp, len(ids))
			continue
		}
		ok := true
		for _, l := range lays[1:] {
			if l != lays[0] {
				ok = false
			}
		}
		if !ok {
			fmt.Printf("%-16s VARIANCE INTRA-CARTE : %v\n", disp, lays)
			continue
		}
		r := res{mapName: disp, module: aprioriModule[disp], measured: lays[0].AxisW, bounds: bnds[0], nFilms: len(lays)}
		if r.module == "" {
			fmt.Printf("%-16s [module non déductible du nom]  mesuré %d/%d/%d  frontieres %v  (%d films)\n",
				disp, r.measured[0], r.measured[1], r.measured[2], r.bounds, r.nFilms)
			results = append(results, r)
			continue
		}
		b, has := mods[r.module]
		if !has {
			fmt.Printf("%-16s module %s illisible\n", disp, r.module)
			continue
		}
		r.predicted = b.AxisWidths()
		r.gate = r.bounds[0] - r.predicted[0]
		verdict := "CONCORDE"
		if [3]int{int(r.measured[0]), int(r.measured[1]), int(r.measured[2])} != r.predicted {
			verdict = "ECHEC"
		}
		fmt.Printf("%-16s module=%-18s mesuré %2d/%2d/%2d  prédit %2d/%2d/%2d  frontieres %v  gate=b0-W0=%d  %d films  %s\n",
			disp, r.module, r.measured[0], r.measured[1], r.measured[2],
			r.predicted[0], r.predicted[1], r.predicted[2], r.bounds, r.gate, r.nFilms, verdict)
		results = append(results, r)
	}

	// --- score ---
	tested, okCount := 0, 0
	for _, r := range results {
		if r.module == "" {
			continue
		}
		tested++
		if [3]int{int(r.measured[0]), int(r.measured[1]), int(r.measured[2])} == r.predicted {
			okCount++
		}
	}
	fmt.Printf("\nSCORE : %d/%d cartes a priori concordent sur les 3 axes.\n", okCount, tested)

	// --- témoin : bornes d'une AUTRE carte ---
	fmt.Println("\n=== C. TÉMOIN — bornes d'un AUTRE module appliquées à chaque carte ===")
	totalPairs, hits := 0, 0
	for _, r := range results {
		if r.module == "" {
			continue
		}
		want := [3]int{int(r.measured[0]), int(r.measured[1]), int(r.measured[2])}
		var matches []string
		for n, b := range mods {
			if n == r.module {
				continue
			}
			totalPairs++
			if b.AxisWidths() == want {
				hits++
				matches = append(matches, n)
			}
		}
		sort.Strings(matches)
		fmt.Printf("%-16s mesuré %2d/%2d/%2d : %2d/%2d modules ÉTRANGERS reproduisent ce triplet %v\n",
			r.mapName, want[0], want[1], want[2], len(matches), len(mods)-1, matches)
	}
	fmt.Printf("\nTÉMOIN GLOBAL : %d appariements croisés réussis sur %d (%.2f %%).\n",
		hits, totalPairs, 100*float64(hits)/float64(max(totalPairs, 1)))

	// --- prédiction pour les cartes non déductibles du nom ---
	fmt.Println("\n=== D. cartes sans module déductible : modules compatibles ===")
	for _, r := range results {
		if r.module != "" {
			continue
		}
		want := [3]int{int(r.measured[0]), int(r.measured[1]), int(r.measured[2])}
		var cand []string
		for n, b := range mods {
			if b.AxisWidths() == want {
				cand = append(cand, n)
			}
		}
		sort.Strings(cand)
		fmt.Printf("%-16s mesuré %2d/%2d/%2d -> candidats %v\n", r.mapName, want[0], want[1], want[2], cand)
	}
}

// readModules lit le BSP PRINCIPAL (le plus gros tag sbsp) de chaque module.
func readModules() map[string]himap.Bounds {
	out := map[string]himap.Bounds{}
	dirs, err := os.ReadDir(levelsDir)
	if err != nil {
		panic(err)
	}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		mods, _ := filepath.Glob(filepath.Join(levelsDir, d.Name(), "*.module"))
		for _, mp := range mods {
			bsps, err := himap.ReadModuleBSPBounds(mp)
			if err != nil || len(bsps) == 0 || !bsps[0].Bounds.Valid() {
				continue
			}
			out[d.Name()] = bsps[0].Bounds
		}
	}
	return out
}

// readFilms indexe les films en cache par nom de carte affiché.
func readFilms() map[string][]string {
	f, err := os.Open(mapNames)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	out := map[string][]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.SplitN(strings.TrimSpace(sc.Text()), ",", 2)
		if len(parts) != 2 || parts[1] == "" {
			continue
		}
		name := strings.TrimSuffix(parts[1], " - Ranked")
		if _, err := os.Stat(filepath.Join(filmCache, parts[0])); err != nil {
			continue
		}
		out[name] = append(out[name], parts[0])
	}
	for k := range out {
		sort.Strings(out[k])
	}
	return out
}
