// tmp_projtraj — valide le decodeur de trajectoires de projectile contre les 70 lancers.
//
// TEMOIN CENTRAL, et il est refutable : chaque lancer de grenade (marqueur d'etat par defaut
// 0x4C0C00, horodatage + type) doit voir NAITRE une trajectoire de projectile dans les 200 ms.
// CONTROLE NEGATIF : le meme appariement contre les horodatages de lancer decales en bloc.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

func main() {
	dir := flag.String("dir", `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\cache\film_chunks\000d5950`, "chunks du film")
	mapName := flag.String("map", "Cliffhanger", "carte (pour les bornes de dequantification)")
	repo := flag.String("repo", `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\.claude\worktrees\filmdec-continuation`, "racine du depot")
	flag.Parse()

	wr, err := loadRange(*repo, *mapName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bornes:", err)
		os.Exit(1)
	}
	tracks, err := filmdec.ScanFilmProjectiles(*dir, wr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan:", err)
		os.Exit(1)
	}
	throws, err := filmdec.ScanFilmGrenadeThrows(*dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "lancers:", err)
		os.Exit(1)
	}

	fmt.Printf("TRAJECTOIRES DE PROJECTILE — %s\n\n", *dir)
	fmt.Printf("  %d trajectoires (>= 3 points), %d lancers de grenade connus\n", len(tracks), len(throws))

	var npts, atRest int
	for _, t := range tracks {
		npts += len(t.Pts)
		if t.Pts[len(t.Pts)-1].AtRest {
			atRest++
		}
	}
	if len(tracks) > 0 {
		fmt.Printf("  %d positions au total, %.1f par trajectoire en moyenne\n", npts, float64(npts)/float64(len(tracks)))
		fmt.Printf("  %d/%d trajectoires finissent sur `projectile-at-rest-state`\n", atRest, len(tracks))
	}

	starts := make([]uint64, len(tracks))
	for i, t := range tracks {
		starts[i] = t.Pts[0].TimestampUS
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i] < starts[j] })

	const win = 200_000 // 200 ms
	match := func(shift int64) int {
		n := 0
		for _, g := range throws {
			t := int64(g.TimestampUS) + shift
			i := sort.Search(len(starts), func(k int) bool { return int64(starts[k]) >= t-win })
			if i < len(starts) && int64(starts[i]) <= t+win {
				n++
			}
		}
		return n
	}
	got := match(0)
	fmt.Printf("\n  TEMOIN  appariement lancer -> naissance (+-200 ms) : %d/%d = %.1f %%\n",
		got, len(throws), 100*float64(got)/float64(len(throws)))
	fmt.Print("  CONTROLE  memes lancers decales en bloc :")
	for _, sh := range []int64{3_000_000, 7_000_000, 13_000_000, -5_000_000} {
		fmt.Printf("  %+ds:%d", sh/1_000_000, match(sh))
	}
	fmt.Println()

	fmt.Println("\n  echantillon (10 premieres trajectoires) :")
	t0 := uint64(0)
	if len(tracks) > 0 {
		t0 = tracks[0].Pts[0].TimestampUS
	}
	for i, t := range tracks {
		if i >= 10 {
			break
		}
		a, b := t.Pts[0], t.Pts[len(t.Pts)-1]
		dur := float64(b.TimestampUS-a.TimestampUS) / 1e6
		fmt.Printf("    slot %4d gen %d  t=%6.1fs  %3d pts  %4.2fs  (%.1f,%.1f,%.1f) -> (%.1f,%.1f,%.1f)%s\n",
			t.Slot, t.Gen, float64(a.TimestampUS-t0)/1e6, len(t.Pts), dur,
			a.X, a.Y, a.Z, b.X, b.Y, b.Z, restLabel(b.AtRest))
	}
}

func restLabel(b bool) string {
	if b {
		return "  [au repos]"
	}
	return ""
}

func loadRange(repo, name string) (*filmdec.Vec3Range, error) {
	p := filepath.Join(repo, "data", "titles", "halo_infinite", "reference", "map_quant_bounds.json")
	cat, err := filmdec.LoadMapQuantCatalog(p)
	if err != nil {
		return nil, err
	}
	e, err := cat.Lookup(name)
	if err != nil {
		return nil, err
	}
	r := e.Range()
	return &r, nil
}
