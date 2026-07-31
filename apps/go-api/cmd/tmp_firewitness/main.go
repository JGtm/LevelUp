// tmp_firewitness — THROWAWAY : témoins des events de tir (record type 105) décodés hors
// ligne, croisés avec les positions bipeds et killer_victim_pairs.
//
// Usage :
//
//	CGO_ENABLED=0 go run ./cmd/tmp_firewitness <filmDir> <carte> <kv.csv> <participants.csv> <outDir>
package main

import (
	"fmt"
	"math/rand"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/domain/title"
)

const mainRepo = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`

func main() {
	if len(os.Args) < 6 {
		fmt.Println("usage: tmp_firewitness <filmDir> <carte> <kv.csv> <participants.csv> <outDir>")
		os.Exit(2)
	}
	filmDir, mapName, kvPath, partPath, outDir := os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5]

	rng, err := worldRange(mapName)
	if err != nil {
		fmt.Println("bornes:", err)
		os.Exit(1)
	}
	opt := filmdec.DefaultScanFilmOptions()
	opt.WorldRange = rng
	opt.CaptureDirs = true
	pos, err := filmdec.ScanFilmBipedPositions(filmDir, opt)
	if err != nil {
		fmt.Println("positions:", err)
		os.Exit(1)
	}
	lives := buildLives(pos)
	fmt.Printf("positions=%d vies=%d\n", len(pos), len(lives))

	events := scanFireEvents(filmDir)
	nLong := 0
	for _, e := range events {
		if e.Variant == 0 {
			nLong++
		}
	}
	fmt.Printf("events type105=%d (long=%d)\n", len(events), nLong)

	names := map[string]string{}
	deaths, err := loadDeaths(kvPath, names)
	if err != nil {
		fmt.Println("kv:", err)
		os.Exit(1)
	}
	parts, err := loadParticipants(partPath, names)
	if err != nil {
		fmt.Println("participants:", err)
		os.Exit(1)
	}

	t0 := int64(pos[0].TimestampUS)
	for _, p := range pos {
		if int64(p.TimestampUS) < t0 {
			t0 = int64(p.TimestampUS)
		}
	}
	ends := make([]int64, len(lives))
	for i, l := range lives {
		ends[i] = (l.EndUS - t0) / 1000
	}

	off, n := bestOffset(ends, deaths)
	fmt.Printf("offset film<->match = %d ms (appariements a +-150 ms : %d/%d)\n", off, n, len(deaths))
	matched := matchDeaths(ends, deaths, off, lives)
	fmt.Printf("appariements retenus : %d/%d\n", matched, len(deaths))

	// TÉMOIN de la corrélation : morts tirées au hasard sur la même plage.
	ctrl := controlMatch(ends, deaths, off, len(lives))
	fmt.Printf("TEMOIN corrélation (morts aléatoires) : %d/%d\n", ctrl, len(deaths))

	added, agree, checked, ambiguous := chainRespawns(lives, deaths, t0, off)
	named := 0
	for _, l := range lives {
		if l.Player != "" {
			named++
		}
	}
	fmt.Printf("chaînage contraint des respawns : +%d vies nommées (total %d/%d), %d ambigües ;"+
		" CONTROLE : la règle retrouve %d/%d des noms venus des morts\n",
		added, named, len(lives), ambiguous, agree, checked)

	run(outDir, lives, deaths, events, parts, t0, off)
}

// worldRange charge les bornes de la carte dans le catalogue versionné.
func worldRange(mapName string) (*filmdec.Vec3Range, error) {
	p := title.NewPathResolver(mainRepo).MapQuantBoundsPath(title.DefaultSlug)
	cat, err := filmdec.LoadMapQuantCatalog(p)
	if err != nil {
		return nil, err
	}
	e, err := cat.Lookup(mapName)
	if err != nil {
		return nil, err
	}
	r := e.Range()
	return &r, nil
}

// bestOffset balaie le décalage horloge film <-> horloge match qui apparie le plus de morts
// à une fin de vie. Le maximum est un PLATEAU (toute la largeur de la fenêtre d'acceptation
// donne le même compte) : on retient son CENTRE, sans quoi l'offset retenu est au bord et
// tous les écarts d'appariement valent ~la demi-fenêtre.
func bestOffset(ends []int64, deaths []Death) (int64, int) {
	bestN := -1
	var plateau []int64
	for off := int64(-20000); off <= 20000; off += 10 {
		n := countMatches(ends, deaths, off)
		if n > bestN {
			bestN, plateau = n, []int64{off}
		} else if n == bestN {
			plateau = append(plateau, off)
		}
	}
	return plateau[len(plateau)/2], bestN
}

const matchWindowMS = 150

func countMatches(ends []int64, deaths []Death, off int64) int {
	used := make([]bool, len(ends))
	n := 0
	for _, d := range deaths {
		target := d.TimeMS + off
		bi, bd := -1, int64(matchWindowMS+1)
		for i, e := range ends {
			if used[i] {
				continue
			}
			delta := e - target
			if delta < 0 {
				delta = -delta
			}
			if delta < bd {
				bd, bi = delta, i
			}
		}
		if bi >= 0 {
			used[bi] = true
			n++
		}
	}
	return n
}

// matchDeaths apparie définitivement (glouton par écart croissant) et pose le nom de la
// victime sur la vie.
func matchDeaths(ends []int64, deaths []Death, off int64, lives []Life) int {
	type pair struct {
		di, li int
		d      int64
	}
	var ps []pair
	for di, d := range deaths {
		target := d.TimeMS + off
		for li, e := range ends {
			delta := e - target
			if delta < 0 {
				delta = -delta
			}
			if delta <= matchWindowMS {
				ps = append(ps, pair{di, li, delta})
			}
		}
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].d < ps[j].d })
	usedD := make([]bool, len(deaths))
	usedL := make([]bool, len(lives))
	n := 0
	var deltas []float64
	for _, p := range ps {
		if usedD[p.di] || usedL[p.li] {
			continue
		}
		usedD[p.di], usedL[p.li] = true, true
		deaths[p.di].LifeIndex = p.li
		lives[p.li].Player = deaths[p.di].Victim
		lives[p.li].DeathNamed = true
		deltas = append(deltas, float64(p.d))
		n++
	}
	sort.Float64s(deltas)
	fmt.Printf("  écart médian d'appariement = %.0f ms (p90 %.0f)\n", median(deltas), quantile(deltas, 0.9))
	return n
}

// controlMatch : témoin — les mêmes morts, replacées uniformément au hasard sur la plage.
func controlMatch(ends []int64, deaths []Death, off int64, nLives int) int {
	if len(ends) == 0 {
		return 0
	}
	lo, hi := ends[0], ends[0]
	for _, e := range ends {
		lo, hi = minI(lo, e), maxI(hi, e)
	}
	r := rand.New(rand.NewSource(1))
	fake := make([]Death, len(deaths))
	for i := range fake {
		fake[i].TimeMS = lo + r.Int63n(hi-lo+1) - off
	}
	sort.Slice(fake, func(i, j int) bool { return fake[i].TimeMS < fake[j].TimeMS })
	return countMatches(ends, fake, off)
}

func minI(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxI(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
